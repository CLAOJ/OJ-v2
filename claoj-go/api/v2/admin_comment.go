package v2

import (
	"errors"
	"net/http"
	"time"

	"github.com/CLAOJ/claoj-go/db"
	"github.com/CLAOJ/claoj-go/models"
	"github.com/CLAOJ/claoj-go/service/comment"
	"github.com/gin-gonic/gin"
)

// ============================================================
// ADMIN COMMENT MANAGEMENT API
// ============================================================

// AdminCommentList - GET /api/v2/admin/comments
// List all comments with filtering and pagination
func AdminCommentList(c *gin.Context) {
	page, pageSize := parsePagination(c)
	search := c.Query("search")
	hidden := c.Query("hidden") // "true", "false", or empty for all

	query := db.DB.Model(&models.Comment{}).
		Preload("Author.User").
		Joins("LEFT JOIN judge_profile ON judge_profile.id = judge_comment.author_id").
		Joins("LEFT JOIN auth_user ON auth_user.id = judge_profile.user_id")

	if search != "" {
		query = query.Where("judge_comment.body LIKE ? OR auth_user.username LIKE ? OR judge_comment.page LIKE ?",
			"%"+search+"%", "%"+search+"%", "%"+search+"%")
	}

	if hidden != "" {
		if hidden == "true" {
			query = query.Where("judge_comment.hidden = ?", true)
		} else if hidden == "false" {
			query = query.Where("judge_comment.hidden = ?", false)
		}
	}

	// Get total count
	var total int64
	if err := query.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	// Get comments
	var comments []models.Comment
	if err := query.
		Order("judge_comment.time DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&comments).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	type CommentItem struct {
		ID        uint      `json:"id"`
		AuthorID  uint      `json:"author_id"`
		Username  string    `json:"username"`
		Page      string    `json:"page"`
		Body      string    `json:"body"`
		Score     int       `json:"score"`
		Hidden    bool      `json:"hidden"`
		Time      time.Time `json:"time"`
		ParentID  *uint     `json:"parent_id,omitempty"`
	}

	items := make([]CommentItem, len(comments))
	for i, cm := range comments {
		username := ""
		if cm.Author.User.Username != "" {
			username = cm.Author.User.Username
		}
		items[i] = CommentItem{
			ID:       cm.ID,
			AuthorID: cm.AuthorID,
			Username: username,
			Page:     cm.Page,
			Body:     cm.Body,
			Score:    cm.Score,
			Hidden:   cm.Hidden,
			Time:     cm.Time,
			ParentID: cm.ParentID,
		}
	}

	c.JSON(http.StatusOK, apiListWithTotal(items, total))
}

// AdminCommentUpdate - PATCH /api/v2/admin/comment/:id
// Admin update comment (body, hidden status)
func AdminCommentUpdate(c *gin.Context) {
	user, profile, ok := resolveUserProfile(c)
	if !ok {
		return
	}

	if !user.IsStaff && !user.IsSuperuser {
		c.JSON(http.StatusForbidden, apiError("admin access required"))
		return
	}

	commentIDStr := c.Param("id")
	var commentID uint
	if err := parseUint(commentIDStr, &commentID); err != nil {
		c.JSON(http.StatusBadRequest, apiError("invalid comment id"))
		return
	}

	var req struct {
		Body   *string `json:"body,omitempty"`
		Hidden *bool   `json:"hidden,omitempty"`
		Reason string  `json:"reason,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, apiError(err.Error()))
		return
	}

	_, err := getCommentService().UpdateComment(comment.UpdateCommentRequest{
		CommentID: commentID,
		Body:      req.Body,
		Hidden:    req.Hidden,
		Reason:    req.Reason,
		EditorID:  profile.ID,
	})
	if err != nil {
		if errors.Is(err, comment.ErrCommentNotFound) {
			c.JSON(http.StatusNotFound, apiError("comment not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "comment updated",
	})
}

// AdminCommentDelete - DELETE /api/v2/admin/comment/:id
// Hard delete a comment (admin only)
func AdminCommentDelete(c *gin.Context) {
	user, _, ok := resolveUserProfile(c)
	if !ok {
		return
	}

	if !user.IsStaff && !user.IsSuperuser {
		c.JSON(http.StatusForbidden, apiError("admin access required"))
		return
	}

	commentIDStr := c.Param("id")
	var commentID uint
	if err := parseUint(commentIDStr, &commentID); err != nil {
		c.JSON(http.StatusBadRequest, apiError("invalid comment id"))
		return
	}

	if err := getCommentService().DeleteComment(comment.DeleteCommentRequest{
		CommentID: commentID,
	}); err != nil {
		if errors.Is(err, comment.ErrCommentNotFound) {
			c.JSON(http.StatusNotFound, apiError("comment not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "comment deleted",
	})
}
