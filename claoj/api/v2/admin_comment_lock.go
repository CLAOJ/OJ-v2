package v2

import (
	"net/http"
	"strings"

	"github.com/CLAOJ/claoj/auth"
	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ============================================================
// ADMIN COMMENT LOCKS (v1 judge_commentlock: one row per locked page)
// ============================================================

// cleanLockPage validates a lock page string with the same pattern the admin
// comment editor enforces, so a lock always names a page comments can use.
func cleanLockPage(page string) (string, bool) {
	page = strings.TrimSpace(page)
	if page == "" || len(page) > 30 || !commentPagePattern.MatchString(page) {
		return "", false
	}
	return page, true
}

// AdminCommentLockList - GET /api/v2/admin/comment-locks
func AdminCommentLockList(c *gin.Context) {
	page, pageSize := parsePagination(c)

	search := strings.TrimSpace(c.Query("search"))
	build := func() *gorm.DB {
		q := db.DB.Model(&models.CommentLock{})
		if search != "" {
			q = q.Where("page LIKE ? ESCAPE '!'", db.ContainsPattern(search))
		}
		return q
	}

	var total int64
	if err := build().Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	var locks []models.CommentLock
	if err := build().Order("page ASC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&locks).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	type Item struct {
		ID   uint   `json:"id"`
		Page string `json:"page"`
	}
	items := make([]Item, len(locks))
	for i, l := range locks {
		items[i] = Item{ID: l.ID, Page: l.Page}
	}

	c.JSON(http.StatusOK, gin.H{
		"data":      items,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

type commentLockRequest struct {
	Page string `json:"page" binding:"required"`
}

// AdminCommentLockCreate - POST /api/v2/admin/comment-locks
func AdminCommentLockCreate(c *gin.Context) {
	var req commentLockRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, apiUserError(err.Error()))
		return
	}
	page, ok := cleanLockPage(req.Page)
	if !ok {
		c.JSON(http.StatusBadRequest, apiUserError("page must look like p/code, e/blog/12, c/key or s/code"))
		return
	}
	var existing int64
	db.DB.Model(&models.CommentLock{}).Where("page = ?", page).Count(&existing)
	if existing > 0 {
		c.JSON(http.StatusBadRequest, apiUserError("page is already locked"))
		return
	}
	lock := models.CommentLock{Page: page}
	if err := db.DB.Create(&lock).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"message": "comment lock created",
		"lock":    gin.H{"id": lock.ID, "page": lock.Page},
	})
}

// AdminCommentLockUpdate - PATCH /api/v2/admin/comment-lock/:id
func AdminCommentLockUpdate(c *gin.Context) {
	id, ok := numericParam(c, "id")
	if !ok {
		return
	}
	var lock models.CommentLock
	if err := db.DB.First(&lock, id).Error; err != nil {
		c.JSON(http.StatusNotFound, apiUserError("comment lock not found"))
		return
	}

	var req commentLockRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, apiUserError(err.Error()))
		return
	}
	page, ok := cleanLockPage(req.Page)
	if !ok {
		c.JSON(http.StatusBadRequest, apiUserError("page must look like p/code, e/blog/12, c/key or s/code"))
		return
	}
	var existing int64
	db.DB.Model(&models.CommentLock{}).Where("page = ? AND id <> ?", page, lock.ID).Count(&existing)
	if existing > 0 {
		c.JSON(http.StatusBadRequest, apiUserError("page is already locked"))
		return
	}
	if err := db.DB.Model(&lock).Update("page", page).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "comment lock updated",
		"lock":    gin.H{"id": lock.ID, "page": page},
	})
}

// AdminCommentLockDelete - DELETE /api/v2/admin/comment-lock/:id
func AdminCommentLockDelete(c *gin.Context) {
	id, ok := numericParam(c, "id")
	if !ok {
		return
	}
	result := db.DB.Delete(&models.CommentLock{}, id)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, apiError(result.Error.Error()))
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, apiUserError("comment lock not found"))
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "comment lock deleted"})
}

// commentPageLocked reports whether comments are locked on page for this
// caller. v1 judge/comments.py:62: a lock row for the page blocks everyone
// without judge.override_comment_lock.
func commentPageLocked(c *gin.Context, page string) bool {
	if auth.HasPerm(c, "judge.override_comment_lock") {
		return false
	}
	var count int64
	db.DB.Model(&models.CommentLock{}).Where("page = ?", page).Count(&count)
	return count > 0
}