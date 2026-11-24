package v2

import (
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/CLAOJ/claoj/auth"
	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/CLAOJ/claoj/service/comment"
	"github.com/gin-gonic/gin"
)

// commentPagePattern accepts both the v1 colon code and the v2 slash prefix,
// so an admin edit cannot store a page code the lists and links can't parse.
// Dots and dashes are allowed: real problem/contest keys use them.
var commentPagePattern = regexp.MustCompile(`^\w+[:/][\w.-]+$`)

// ============================================================
// ADMIN COMMENT MANAGEMENT API
// ============================================================

// AdminCommentList - GET /api/v2/admin/comments
// List all comments with filtering and pagination
func AdminCommentList(c *gin.Context) {
	page, pageSize := parsePagination(c)
	search := c.Query("search")
	hidden := c.Query("hidden") // "true", "false", or empty for all
	pageType := c.Query("page_type")

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

	// Two page-code formats coexist: v2 writes slash prefixes (p/, e/, blog/,
	// c/), v1 wrote colon codes (p:, s:, c:, b:). Each type matches both.
	switch pageType {
	case "problem":
		query = query.Where("judge_comment.page LIKE ? OR judge_comment.page LIKE ?", "p/%", "p:%")
	case "editorial":
		query = query.Where("judge_comment.page LIKE ? OR judge_comment.page LIKE ?", "e/%", "s:%")
	case "contest":
		query = query.Where("judge_comment.page LIKE ? OR judge_comment.page LIKE ?", "c/%", "c:%")
	case "blog":
		query = query.Where("judge_comment.page LIKE ? OR judge_comment.page LIKE ?", "blog/%", "b:%")
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
		Revisions int       `json:"revisions"`
	}

	items := make([]CommentItem, len(comments))
	for i, cm := range comments {
		username := ""
		if cm.Author.User.Username != "" {
			username = cm.Author.User.Username
		}
		items[i] = CommentItem{
			ID:        cm.ID,
			AuthorID:  cm.AuthorID,
			Username:  username,
			Page:      cm.Page,
			Body:      cm.Body,
			Score:     cm.Score,
			Hidden:    cm.Hidden,
			Time:      cm.Time,
			ParentID:  cm.ParentID,
			Revisions: cm.Revisions,
		}
	}

	c.JSON(http.StatusOK, apiListWithTotal(items, total))
}

// AdminCommentUpdate - PATCH /api/v2/admin/comment/:id
// Admin update comment (body, hidden status, author, page). time/score and
// parent stay read-only: parent is an MPTT tree edge and v2 has no safe
// reparent helper, so a parent_id in the body is rejected rather than applied.
func AdminCommentUpdate(c *gin.Context) {
	_, profile, ok := resolveUserProfile(c)
	if !ok {
		return
	}

	if !auth.HasPerm(c, "judge.change_comment") {
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
		Body     *string         `json:"body,omitempty"`
		Hidden   *bool           `json:"hidden,omitempty"`
		AuthorID *uint           `json:"author_id,omitempty"`
		Page     *string         `json:"page,omitempty"`
		ParentID json.RawMessage `json:"parent_id,omitempty"`
		Reason   string          `json:"reason,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, apiError(err.Error()))
		return
	}

	// RawMessage is non-nil whenever the key is present, including null, so an
	// explicit parent update (even to null) is an error rather than a no-op.
	if req.ParentID != nil {
		c.JSON(http.StatusBadRequest, apiUserError("reparenting a comment is not supported"))
		return
	}

	if req.AuthorID != nil {
		if *req.AuthorID == 0 {
			c.JSON(http.StatusBadRequest, apiUserError("invalid author id"))
			return
		}
		var authors int64
		if err := db.DB.Model(&models.Profile{}).Where("id = ?", *req.AuthorID).Count(&authors).Error; err != nil {
			c.JSON(http.StatusInternalServerError, apiError(err.Error()))
			return
		}
		if authors == 0 {
			c.JSON(http.StatusBadRequest, apiUserError("author not found"))
			return
		}
	}

	if req.Page != nil {
		page := strings.TrimSpace(*req.Page)
		if page == "" || len(page) > 30 || !commentPagePattern.MatchString(page) {
			c.JSON(http.StatusBadRequest, apiUserError("invalid page code"))
			return
		}
		req.Page = &page
	}

	_, err := getCommentService().UpdateComment(comment.UpdateCommentRequest{
		CommentID: commentID,
		Body:      req.Body,
		Hidden:    req.Hidden,
		AuthorID:  req.AuthorID,
		Page:      req.Page,
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
	_, _, ok := resolveUserProfile(c)
	if !ok {
		return
	}

	if !auth.HasPerm(c, "judge.change_comment") {
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

// AdminCommentBulkHide - POST /api/v2/admin/comments/bulk-hide
// Hide or unhide up to 500 comments at once. The admin audit middleware
// records the request (path + ids) like the single-comment hide/edit.
func AdminCommentBulkHide(c *gin.Context) {
	_, _, ok := resolveUserProfile(c)
	if !ok {
		return
	}

	if !auth.HasPerm(c, "judge.change_comment") {
		c.JSON(http.StatusForbidden, apiError("admin access required"))
		return
	}

	var req struct {
		IDs    []uint `json:"ids"`
		Hidden *bool  `json:"hidden"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, apiError(err.Error()))
		return
	}

	if len(req.IDs) == 0 {
		c.JSON(http.StatusBadRequest, apiUserError("ids cannot be empty"))
		return
	}
	if len(req.IDs) > 500 {
		c.JSON(http.StatusBadRequest, apiUserError("at most 500 comments per request"))
		return
	}
	for _, id := range req.IDs {
		if id == 0 {
			c.JSON(http.StatusBadRequest, apiUserError("invalid comment id"))
			return
		}
	}
	if req.Hidden == nil {
		c.JSON(http.StatusBadRequest, apiUserError("hidden is required"))
		return
	}

	res := db.DB.Model(&models.Comment{}).Where("id IN ?", req.IDs).Update("hidden", *req.Hidden)
	if res.Error != nil {
		c.JSON(http.StatusInternalServerError, apiError(res.Error.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"updated": res.RowsAffected,
		"hidden":  *req.Hidden,
	})
}
