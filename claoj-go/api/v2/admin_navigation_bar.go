package v2

import (
	"errors"
	"net/http"

	"github.com/CLAOJ/claoj-go/db"
	"github.com/CLAOJ/claoj-go/models"
	"github.com/CLAOJ/claoj-go/service/navigation"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ============================================================
// ADMIN NAVIGATION BAR MANAGEMENT API
// ============================================================

// AdminNavigationBarList - GET /api/v2/admin/navigation-bars
func AdminNavigationBarList(c *gin.Context) {
	page, pageSize := parsePagination(c)

	resp, err := getNavigationService().ListNavigation(navigation.ListNavigationRequest{
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	type Item struct {
		ID     uint   `json:"id"`
		Key    string `json:"key"`
		Label  string `json:"label"`
		Path   string `json:"path"`
		Order  int    `json:"order"`
		Level  int    `json:"level"`
		Parent *struct {
			ID    uint   `json:"id"`
			Label string `json:"label"`
		} `json:"parent"`
	}

	items := make([]Item, len(resp.Entries))
	for i, nb := range resp.Entries {
		items[i] = Item{
			ID:    nb.ID,
			Key:   nb.Key,
			Label: nb.Label,
			Path:  nb.Path,
			Order: nb.Order,
			Level: nb.Level,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  items,
		"total": resp.Total,
	})
}

// AdminNavigationBarDetail - GET /api/v2/admin/navigation-bar/:id
func AdminNavigationBarDetail(c *gin.Context) {
	idStr := c.Param("id")
	var id uint
	if err := parseUint(idStr, &id); err != nil {
		c.JSON(http.StatusBadRequest, apiError("invalid navigation bar id"))
		return
	}

	nav, err := getNavigationService().GetNavigation(navigation.GetNavigationRequest{
		NavID: id,
	})
	if err != nil {
		if errors.Is(err, navigation.ErrNavNotFound) {
			c.JSON(http.StatusNotFound, apiError("navigation bar not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":        nav.ID,
		"key":       nav.Key,
		"label":     nav.Label,
		"path":      nav.Path,
		"parent_id": nav.ParentID,
		"order":     nav.Order,
		"level":     nav.Level,
		"lft":       nav.Lft,
		"rght":      nav.Rght,
		"tree_id":   nav.TreeID,
	})
}

// AdminNavigationBarCreate - POST /api/v2/admin/navigation-bars
func AdminNavigationBarCreate(c *gin.Context) {
	var req struct {
		Key      string `json:"key" binding:"required"`
		Label    string `json:"label" binding:"required"`
		Path     string `json:"path" binding:"required"`
		ParentID *uint  `json:"parent_id"`
		Order    int    `json:"order"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, apiError(err.Error()))
		return
	}

	// Validate parent if provided
	if req.ParentID != nil {
		parent, err := getNavigationService().GetNavigation(navigation.GetNavigationRequest{
			NavID: *req.ParentID,
		})
		if err != nil {
			c.JSON(http.StatusBadRequest, apiError("parent navigation bar not found"))
			return
		}

		nav, err := getNavigationService().CreateNavigation(navigation.CreateNavigationRequest{
			Key:      req.Key,
			Label:    req.Label,
			Path:     req.Path,
			ParentID: req.ParentID,
			Order:    req.Order,
		})
		if err != nil {
			if errors.Is(err, navigation.ErrKeyExists) {
				c.JSON(http.StatusBadRequest, apiError("navigation bar key already exists"))
				return
			}
			if errors.Is(err, navigation.ErrEmptyKey) {
				c.JSON(http.StatusBadRequest, apiError("navigation bar key cannot be empty"))
				return
			}
			if errors.Is(err, navigation.ErrEmptyLabel) {
				c.JSON(http.StatusBadRequest, apiError("navigation bar label cannot be empty"))
				return
			}
			if errors.Is(err, navigation.ErrEmptyPath) {
				c.JSON(http.StatusBadRequest, apiError("navigation bar path cannot be empty"))
				return
			}
			c.JSON(http.StatusInternalServerError, apiError(err.Error()))
			return
		}

		// Update tree structure (nested set model)
		var navBar models.NavigationBar
		db.DB.First(&navBar, nav.ID)
		navBar.Level = parent.Level + 1
		navBar.TreeID = parent.TreeID
		navBar.Lft = parent.Rght
		navBar.Rght = parent.Rght + 1
		db.DB.Model(&models.NavigationBar{}).Where("tree_id = ? AND rght > ?", parent.TreeID, parent.Rght).Update("rght", gorm.Expr("rght + 2"))
		db.DB.Save(&navBar)

		c.JSON(http.StatusCreated, gin.H{
			"message":        "navigation bar created",
			"navigation_bar": gin.H{"id": nav.ID, "key": nav.Key},
		})
		return
	}

	// No parent - create root level entry
	nav, err := getNavigationService().CreateNavigation(navigation.CreateNavigationRequest{
		Key:      req.Key,
		Label:    req.Label,
		Path:     req.Path,
		ParentID: req.ParentID,
		Order:    req.Order,
	})
	if err != nil {
		if errors.Is(err, navigation.ErrKeyExists) {
			c.JSON(http.StatusBadRequest, apiError("navigation bar key already exists"))
			return
		}
		if errors.Is(err, navigation.ErrEmptyKey) {
			c.JSON(http.StatusBadRequest, apiError("navigation bar key cannot be empty"))
			return
		}
		if errors.Is(err, navigation.ErrEmptyLabel) {
			c.JSON(http.StatusBadRequest, apiError("navigation bar label cannot be empty"))
			return
		}
		if errors.Is(err, navigation.ErrEmptyPath) {
			c.JSON(http.StatusBadRequest, apiError("navigation bar path cannot be empty"))
			return
		}
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":        "navigation bar created",
		"navigation_bar": gin.H{"id": nav.ID, "key": nav.Key},
	})
}

// AdminNavigationBarUpdate - PATCH /api/v2/admin/navigation-bar/:id
func AdminNavigationBarUpdate(c *gin.Context) {
	idStr := c.Param("id")
	var id uint
	if err := parseUint(idStr, &id); err != nil {
		c.JSON(http.StatusBadRequest, apiError("invalid navigation bar id"))
		return
	}

	var req struct {
		Label string `json:"label"`
		Path  string `json:"path"`
		Order int    `json:"order"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, apiError(err.Error()))
		return
	}

	// Build update request
	updateReq := navigation.UpdateNavigationRequest{
		NavID: id,
	}
	if req.Label != "" {
		updateReq.Label = &req.Label
	}
	if req.Path != "" {
		updateReq.Path = &req.Path
	}
	if req.Order != 0 {
		updateReq.Order = &req.Order
	}

	_, err := getNavigationService().UpdateNavigation(updateReq)
	if err != nil {
		if errors.Is(err, navigation.ErrNavNotFound) {
			c.JSON(http.StatusNotFound, apiError("navigation bar not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "navigation bar updated",
	})
}

// AdminNavigationBarDelete - DELETE /api/v2/admin/navigation-bar/:id
func AdminNavigationBarDelete(c *gin.Context) {
	idStr := c.Param("id")
	var id uint
	if err := parseUint(idStr, &id); err != nil {
		c.JSON(http.StatusBadRequest, apiError("invalid navigation bar id"))
		return
	}

	// Get nav entry first to get tree info
	nav, err := getNavigationService().GetNavigation(navigation.GetNavigationRequest{
		NavID: id,
	})
	if err != nil {
		if errors.Is(err, navigation.ErrNavNotFound) {
			c.JSON(http.StatusNotFound, apiError("navigation bar not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	treeID := nav.TreeID
	rght := nav.Rght
	lft := nav.Lft
	width := rght - lft + 1

	// Delete using service
	if err := getNavigationService().DeleteNavigation(navigation.DeleteNavigationRequest{
		NavID: id,
	}); err != nil {
		if errors.Is(err, navigation.ErrNavNotFound) {
			c.JSON(http.StatusNotFound, apiError("navigation bar not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	// Update tree structure (nested set model)
	db.DB.Model(&models.NavigationBar{}).Where("tree_id = ? AND lft > ?", treeID, lft).Update("lft", gorm.Expr("lft - ?", width))
	db.DB.Model(&models.NavigationBar{}).Where("tree_id = ? AND rght > ?", treeID, rght).Update("rght", gorm.Expr("rght - ?", width))

	c.JSON(http.StatusOK, gin.H{
		"message": "navigation bar deleted",
	})
}
