package v2

import (
	"net/http"
	"strings"

	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/gin-gonic/gin"
)

// ============================================================
// ADMIN CLASS LIST (read-only; picker source for contest forms)
// ============================================================

// AdminClassList - GET /api/v2/admin/classes
func AdminClassList(c *gin.Context) {
	page, pageSize := parsePagination(c)

	q := db.DB.Model(&models.Class{})
	if search := strings.TrimSpace(c.Query("search")); search != "" {
		q = q.Where("name LIKE ? ESCAPE '!'", db.ContainsPattern(search))
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	var classes []models.Class
	if err := q.Preload("Organization").
		Order("name ASC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&classes).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	type Item struct {
		ID               uint   `json:"id"`
		Name             string `json:"name"`
		Slug             string `json:"slug"`
		IsActive         bool   `json:"is_active"`
		OrganizationID   uint   `json:"organization_id"`
		OrganizationName string `json:"organization_name"`
	}
	items := make([]Item, len(classes))
	for i, class := range classes {
		items[i] = Item{
			ID:               class.ID,
			Name:             class.Name,
			Slug:             class.Slug,
			IsActive:         class.IsActive,
			OrganizationID:   class.OrganizationID,
			OrganizationName: class.Organization.Name,
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"data":      items,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}
