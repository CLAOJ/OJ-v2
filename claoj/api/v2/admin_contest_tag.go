package v2

import (
	"net/http"
	"regexp"

	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ============================================================
// CONTEST TAG ADMIN API
// ============================================================

// AdminContestTagList - GET /api/v2/admin/contest-tags
func AdminContestTagList(c *gin.Context) {
	page, pageSize := parsePagination(c)
	search := c.Query("search")

	var tags []models.ContestTag
	query := db.DB.Model(&models.ContestTag{})

	if search != "" {
		query = query.Where("name LIKE ? OR description LIKE ?", "%"+search+"%", "%"+search+"%")
	}

	if err := query.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&tags).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	type Item struct {
		ID          uint   `json:"id"`
		Name        string `json:"name"`
		Color       string `json:"color"`
		Description string `json:"description"`
	}

	items := make([]Item, len(tags))
	for i, tag := range tags {
		items[i] = Item{
			ID:          tag.ID,
			Name:        tag.Name,
			Color:       tag.Color,
			Description: tag.Description,
		}
	}

	c.JSON(http.StatusOK, apiList(items))
}

// AdminContestTagDetail - GET /api/v2/admin/contest-tag/:id
func AdminContestTagDetail(c *gin.Context) {
	idStr := c.Param("id")
	var id uint
	if err := parseUint(idStr, &id); err != nil {
		c.JSON(http.StatusBadRequest, apiError("invalid tag id"))
		return
	}

	var tag models.ContestTag
	if err := db.DB.First(&tag, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, apiError("tag not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":          tag.ID,
		"name":        tag.Name,
		"color":       tag.Color,
		"description": tag.Description,
	})
}

// ContestTagCreateRequest
type ContestTagCreateRequest struct {
	Name        string `json:"name" binding:"required"`
	Color       string `json:"color" binding:"required"`
	Description string `json:"description"`
}

// AdminContestTagCreate - POST /api/v2/admin/contest-tags
func AdminContestTagCreate(c *gin.Context) {
	var req ContestTagCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, apiError(err.Error()))
		return
	}

	// Validate name format (alphanumeric, underscore, hyphen, max 20 chars)
	validName := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	if !validName.MatchString(req.Name) {
		c.JSON(http.StatusBadRequest, apiError("tag name can only contain letters, numbers, underscore, and hyphen"))
		return
	}

	if len(req.Name) > 20 {
		c.JSON(http.StatusBadRequest, apiError("tag name must be 20 characters or less"))
		return
	}

	// Validate color format (hex color code)
	validColor := regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)
	if !validColor.MatchString(req.Color) {
		c.JSON(http.StatusBadRequest, apiError("color must be a valid hex color code (e.g., #FF5733)"))
		return
	}

	tag := models.ContestTag{
		Name:        req.Name,
		Color:       req.Color,
		Description: req.Description,
	}

	if err := db.DB.Create(&tag).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"tag": gin.H{
			"id":          tag.ID,
			"name":        tag.Name,
			"color":       tag.Color,
			"description": tag.Description,
		},
	})
}

// ContestTagUpdateRequest
type ContestTagUpdateRequest struct {
	Name        string `json:"name"`
	Color       string `json:"color"`
	Description string `json:"description"`
}

// AdminContestTagUpdate - PATCH /api/v2/admin/contest-tag/:id
func AdminContestTagUpdate(c *gin.Context) {
	idStr := c.Param("id")
	var id uint
	if err := parseUint(idStr, &id); err != nil {
		c.JSON(http.StatusBadRequest, apiError("invalid tag id"))
		return
	}

	var tag models.ContestTag
	if err := db.DB.First(&tag, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, apiError("tag not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	var req ContestTagUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, apiError(err.Error()))
		return
	}

	updates := make(map[string]interface{})

	if req.Name != "" {
		validName := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
		if !validName.MatchString(req.Name) {
			c.JSON(http.StatusBadRequest, apiError("tag name can only contain letters, numbers, underscore, and hyphen"))
			return
		}
		if len(req.Name) > 20 {
			c.JSON(http.StatusBadRequest, apiError("tag name must be 20 characters or less"))
			return
		}
		updates["name"] = req.Name
	}

	if req.Color != "" {
		validColor := regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)
		if !validColor.MatchString(req.Color) {
			c.JSON(http.StatusBadRequest, apiError("color must be a valid hex color code (e.g., #FF5733)"))
			return
		}
		updates["color"] = req.Color
	}

	if req.Description != "" {
		updates["description"] = req.Description
	}

	if len(updates) > 0 {
		if err := db.DB.Model(&tag).Updates(updates).Error; err != nil {
			c.JSON(http.StatusInternalServerError, apiError(err.Error()))
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"tag": gin.H{
			"id":          tag.ID,
			"name":        tag.Name,
			"color":       tag.Color,
			"description": tag.Description,
		},
	})
}

// AdminContestTagDelete - DELETE /api/v2/admin/contest-tag/:id
func AdminContestTagDelete(c *gin.Context) {
	idStr := c.Param("id")
	var id uint
	if err := parseUint(idStr, &id); err != nil {
		c.JSON(http.StatusBadRequest, apiError("invalid tag id"))
		return
	}

	if err := db.DB.Delete(&models.ContestTag{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "tag deleted",
	})
}

// AdminContestAddTag - POST /api/v2/admin/contest/:key/tags/:tagId
func AdminContestAddTag(c *gin.Context) {
	contestKey := c.Param("key")
	tagIdStr := c.Param("tagId")

	var contest models.Contest
	if err := db.DB.Where("key = ?", contestKey).First(&contest).Error; err != nil {
		c.JSON(http.StatusNotFound, apiError("contest not found"))
		return
	}

	var tagId uint
	if err := parseUint(tagIdStr, &tagId); err != nil {
		c.JSON(http.StatusBadRequest, apiError("invalid tag id"))
		return
	}

	var tag models.ContestTag
	if err := db.DB.First(&tag, tagId).Error; err != nil {
		c.JSON(http.StatusNotFound, apiError("tag not found"))
		return
	}

	if err := db.DB.Model(&contest).Association("Tags").Append(&tag); err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "tag added to contest",
	})
}

// AdminContestRemoveTag - DELETE /api/v2/admin/contest/:key/tags/:tagId
func AdminContestRemoveTag(c *gin.Context) {
	contestKey := c.Param("key")
	tagIdStr := c.Param("tagId")

	var contest models.Contest
	if err := db.DB.Where("key = ?", contestKey).First(&contest).Error; err != nil {
		c.JSON(http.StatusNotFound, apiError("contest not found"))
		return
	}

	var tagId uint
	if err := parseUint(tagIdStr, &tagId); err != nil {
		c.JSON(http.StatusBadRequest, apiError("invalid tag id"))
		return
	}

	var tag models.ContestTag
	if err := db.DB.First(&tag, tagId).Error; err != nil {
		c.JSON(http.StatusNotFound, apiError("tag not found"))
		return
	}

	if err := db.DB.Model(&contest).Association("Tags").Delete(&tag); err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "tag removed from contest",
	})
}
