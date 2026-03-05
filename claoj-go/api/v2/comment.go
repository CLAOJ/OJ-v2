package v2

import (
	"errors"
	"net/http"
	"time"

	"github.com/CLAOJ/claoj-go/db"
	"github.com/CLAOJ/claoj-go/models"
	"github.com/CLAOJ/claoj-go/sanitization"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// CommentRevision mirrors judge_commentrevision for tracking edits
type CommentRevision struct {
	ID        uint      `gorm:"primaryKey;column:id"`
	CommentID uint      `gorm:"column:comment_id;not null;index"`
	EditorID  uint      `gorm:"column:editor_id;not null;index"`
	Time      time.Time `gorm:"column:time;not null"`
	Body      string    `gorm:"column:body;type:longtext;not null"`
	Reason    string    `gorm:"column:reason;size:200"`
	Comment   models.Comment   `gorm:"foreignKey:CommentID"`
	Editor    models.Profile   `gorm:"foreignKey:EditorID"`
}

func (CommentRevision) TableName() string { return "judge_commentrevision" }

// CommentList - GET /api/v2/comments
// Fetches comments for a specific page (e.g. ?page=p/aplusb)
func CommentList(c *gin.Context) {
	pageFilter := c.Query("page")
	if pageFilter == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "page query parameter is required"})
		return
	}

	page, pageSize := parsePagination(c)
	var comments []models.Comment

	if err := db.DB.
		Preload("Author.User"). // Preload Author profile and their AuthUser for username
		Where("page = ? AND hidden = ?", pageFilter, false).
		Order("time DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&comments).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	type Item struct {
		ID       uint      `json:"id"`
		Body     string    `json:"body"`
		Score    int       `json:"score"`
		Time     time.Time `json:"time"`
		ParentID *uint     `json:"parent_id,omitempty"`
		Author   string    `json:"author"`
	}

	items := make([]Item, len(comments))
	for i, cm := range comments {
		items[i] = Item{
			ID:       cm.ID,
			Body:     cm.Body,
			Score:    cm.Score,
			Time:     cm.Time,
			ParentID: cm.ParentID,
			Author:   cm.Author.User.Username,
		}
	}

	c.JSON(http.StatusOK, apiList(items))
}

type CommentCreateRequest struct {
	Page     string `json:"page" binding:"required"`
	Body     string `json:"body" binding:"required"`
	ParentID *uint  `json:"parent_id"`
}

// CommentCreate - POST /api/v2/comments
// Creates a new comment on a page. Needs Auth middleware.
func CommentCreate(c *gin.Context) {
	_, profile, ok := resolveUserProfile(c)
	if !ok {
		return
	}

	var req CommentCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Basic MPTT compatibility logic.
	// DMOJ uses django-mptt for nested trees. We calculate proper Lft/Rght values
	// to maintain compatibility with Django's MPTT structure.
	var parent models.Comment
	level := 0
	treeID := 0
	lft := 1
	rght := 2

	if req.ParentID != nil {
		if err := db.DB.First(&parent, *req.ParentID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusBadRequest, gin.H{"error": "parent comment not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
			return
		}
		level = parent.Level + 1
		treeID = parent.TreeID
		// Insert as rightmost child of parent
		// Find the maximum rght in this tree that is <= parent's rght
		var maxRght models.Comment
		db.DB.Where("tree_id = ? AND rght <= ?", parent.TreeID, parent.Rght).
			Order("rght DESC").First(&maxRght)
		if maxRght.ID != 0 {
			lft = maxRght.Rght
		} else {
			lft = parent.Lft + 1
		}
		rght = lft + 1
		// Shift existing nodes to make room
		db.DB.Model(&models.Comment{}).
			Where("tree_id = ? AND lft >= ?", parent.TreeID, lft+1).
			UpdateColumn("lft", gorm.Expr("lft + 2"))
		db.DB.Model(&models.Comment{}).
			Where("tree_id = ? AND rght >= ?", parent.TreeID, lft+1).
			UpdateColumn("rght", gorm.Expr("rght + 2"))
	} else {
		// New root comment - get new tree_id and set lft=1, rght=2
		db.DB.Model(&models.Comment{}).Select("IFNULL(MAX(tree_id), 0) + 1").Scan(&treeID)
		// For new tree, check if there are any existing comments on this page
		var existingComment models.Comment
		if err := db.DB.Where("page = ?", req.Page).Order("rght DESC").First(&existingComment).Error; err == nil {
			// Continue the tree from existing max rght
			lft = existingComment.Rght + 1
			rght = lft + 1
		} else {
			lft = 1
			rght = 2
		}
	}

	comment := models.Comment{
		AuthorID: profile.ID,
		Time:     time.Now(),
		Page:     req.Page,
		Score:    0,
		Body:     sanitization.SanitizeComment(req.Body),
		Hidden:   false,
		ParentID: req.ParentID,
		Level:    level,
		TreeID:   treeID,
		Lft:      lft,
		Rght:     rght,
	}

	if err := db.DB.Create(&comment).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create comment"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":    "comment posted",
		"comment_id": comment.ID,
	})
}

// CommentVoteRequest - POST /api/v2/comment/:id/vote
type CommentVoteRequest struct {
	Score int `json:"score" binding:"required,oneof=-1 1"`
}

// CommentVote - POST /api/v2/comment/:id/vote
// Vote on a comment (1 for upvote, -1 for downvote)
func CommentVote(c *gin.Context) {
	_, profile, ok := resolveUserProfile(c)
	if !ok {
		return
	}

	commentIDStr := c.Param("id")
	var commentID uint
	if err := parseUint(commentIDStr, &commentID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid comment id"})
		return
	}

	var req CommentVoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check comment exists
	var comment models.Comment
	if err := db.DB.First(&comment, commentID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "comment not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}

	// Check if already voted
	var existingVote models.CommentVote
	err := db.DB.Where("comment_id = ? AND voter_id = ?", commentID, profile.ID).First(&existingVote).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		// No existing vote, create new one
		vote := models.CommentVote{
			CommentID: commentID,
			VoterID:   profile.ID,
			Score:     req.Score,
		}
		if err := db.DB.Create(&vote).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to record vote"})
			return
		}
		// Update comment score
		db.DB.Model(&comment).UpdateColumn("score", gorm.Expr("score + ?", req.Score))
	} else if err == nil {
		// Existing vote found - update or remove
		if existingVote.Score == req.Score {
			// Same vote - remove it (toggle off)
			db.DB.Delete(&existingVote)
			db.DB.Model(&comment).UpdateColumn("score", gorm.Expr("score - ?", req.Score))
		} else {
			// Different vote - update it (score changes by 2: -1 to 1 or 1 to -1)
			db.DB.Model(&existingVote).Update("score", req.Score)
			db.DB.Model(&comment).UpdateColumn("score", gorm.Expr("score + ?", req.Score*2))
		}
	} else {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}

	// Get updated comment score
	var updatedComment models.Comment
	db.DB.First(&updatedComment, commentID)

	c.JSON(http.StatusOK, gin.H{
		"message": "vote recorded",
		"score":   updatedComment.Score,
	})
}

// CommentUpdateRequest - PATCH /api/v2/comment/:id
type CommentUpdateRequest struct {
	Body   string `json:"body" binding:"required"`
	Reason string `json:"reason"` // Optional edit reason
}


// CommentUpdateRequest - PATCH /api/v2/comment/:id

// CommentUpdate - PATCH /api/v2/comment/:id
// Update a comment (author or admin only). Creates revision history entry.
func CommentUpdate(c *gin.Context) {
	user, profile, ok := resolveUserProfile(c)
	if !ok {
		return
	}

	commentIDStr := c.Param("id")
	var commentID uint
	if err := parseUint(commentIDStr, &commentID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid comment id"})
		return
	}

	var comment models.Comment
	if err := db.DB.First(&comment, commentID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "comment not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}

	// Check permission: author or admin
	isAuthor := comment.AuthorID == profile.ID
	isAdmin := user.IsStaff || user.IsSuperuser

	if !isAuthor && !isAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "permission denied"})
		return
	}

	var req CommentUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Create revision entry before updating
	revision := models.CommentRevision{
		CommentID: comment.ID,
		EditorID:  profile.ID,
		Time:      time.Now(),
		Body:      comment.Body, // Save old body
		Reason:    req.Reason,
	}
	db.DB.Create(&revision)

	// Update comment
	comment.Body = sanitization.SanitizeComment(req.Body)
	if err := db.DB.Save(&comment).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update comment"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":     "comment updated",
		"revision_id": revision.ID,
	})
}

// CommentRevisionList - GET /api/v2/comment/:id/revisions
// Get revision history for a comment
func CommentRevisionList(c *gin.Context) {
	commentIDStr := c.Param("id")
	var commentID uint
	if err := parseUint(commentIDStr, &commentID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid comment id"})
		return
	}

	// Check comment exists
	var comment models.Comment
	if err := db.DB.First(&comment, commentID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "comment not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}

	// Get revisions
	var revisions []models.CommentRevision
	if err := db.DB.Where("comment_id = ?", commentID).
		Preload("Editor.User").
		Order("time DESC").
		Find(&revisions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}

	type RevisionItem struct {
		ID     uint      `json:"id"`
		Editor string    `json:"editor"`
		Time   time.Time `json:"time"`
		Body   string    `json:"body"`
		Reason string    `json:"reason,omitempty"`
	}

	items := make([]RevisionItem, len(revisions))
	for i, rev := range revisions {
		items[i] = RevisionItem{
			ID:     rev.ID,
			Editor: rev.Editor.User.Username,
			Time:   rev.Time,
			Body:   rev.Body,
			Reason: rev.Reason,
		}
	}

	c.JSON(http.StatusOK, apiList(items))
}

// CommentHideRequest - POST /api/v2/admin/comment/:id/hide
type CommentHideRequest struct {
	Hidden bool `json:"hidden"`
}

// CommentHide - POST /api/v2/admin/comment/:id/hide
// Admin-only: Hide or unhide a comment
func CommentHide(c *gin.Context) {
	user, _, ok := resolveUserProfile(c)
	if !ok {
		return
	}

	// Check admin permission
	if !user.IsStaff && !user.IsSuperuser {
		c.JSON(http.StatusForbidden, gin.H{"error": "admin access required"})
		return
	}

	commentIDStr := c.Param("id")
	var commentID uint
	if err := parseUint(commentIDStr, &commentID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid comment id"})
		return
	}

	var req CommentHideRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var comment models.Comment
	if err := db.DB.First(&comment, commentID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "comment not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}

	comment.Hidden = req.Hidden
	if err := db.DB.Save(&comment).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update comment"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "comment updated",
		"hidden":  comment.Hidden,
	})
}

// CommentDelete - DELETE /api/v2/comment/:id
// Soft delete a comment (author or admin only)
func CommentDelete(c *gin.Context) {
	user, profile, ok := resolveUserProfile(c)
	if !ok {
		return
	}

	commentIDStr := c.Param("id")
	var commentID uint
	if err := parseUint(commentIDStr, &commentID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid comment id"})
		return
	}

	var comment models.Comment
	if err := db.DB.First(&comment, commentID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "comment not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}

	// Check permission: author or admin
	isAuthor := comment.AuthorID == profile.ID
	isAdmin := user.IsStaff || user.IsSuperuser

	if !isAuthor && !isAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "permission denied"})
		return
	}

	// Soft delete: clear body and mark hidden
	comment.Body = "[deleted]"
	comment.Hidden = true
	if err := db.DB.Save(&comment).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete comment"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "comment deleted",
	})
}
