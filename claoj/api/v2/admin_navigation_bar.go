package v2

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/CLAOJ/claoj/service/navigation"
	"github.com/gin-gonic/gin"
)

// ============================================================
// ADMIN NAVIGATION BAR MANAGEMENT API
// ============================================================

func navigationBarError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, navigation.ErrNavNotFound):
		c.JSON(http.StatusNotFound, apiUserError("navigation bar not found"))
	case errors.Is(err, navigation.ErrKeyExists):
		c.JSON(http.StatusBadRequest, apiUserError("navigation bar key already exists"))
	case errors.Is(err, navigation.ErrEmptyKey):
		c.JSON(http.StatusBadRequest, apiUserError("navigation bar key cannot be empty"))
	case errors.Is(err, navigation.ErrEmptyLabel):
		c.JSON(http.StatusBadRequest, apiUserError("navigation bar label cannot be empty"))
	case errors.Is(err, navigation.ErrEmptyPath):
		c.JSON(http.StatusBadRequest, apiUserError("navigation bar path cannot be empty"))
	case errors.Is(err, navigation.ErrInvalidRegex):
		c.JSON(http.StatusBadRequest, apiUserError("navigation bar regex is invalid"))
	case errors.Is(err, navigation.ErrParentNotFound):
		c.JSON(http.StatusBadRequest, apiUserError("parent navigation bar not found"))
	case errors.Is(err, navigation.ErrParentCycle):
		c.JSON(http.StatusBadRequest, apiUserError("parent navigation bar cannot be the item itself or one of its descendants"))
	default:
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
	}
}

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
		ID       uint   `json:"id"`
		Key      string `json:"key"`
		Label    string `json:"label"`
		Path     string `json:"path"`
		Regex    string `json:"regex"`
		ParentID *uint  `json:"parent_id"`
		Order    int    `json:"order"`
		Level    int    `json:"level"`
		Parent   *struct {
			ID    uint   `json:"id"`
			Label string `json:"label"`
		} `json:"parent"`
	}

	items := make([]Item, len(resp.Entries))
	for i, nb := range resp.Entries {
		items[i] = Item{
			ID:       nb.ID,
			Key:      nb.Key,
			Label:    nb.Label,
			Path:     nb.Path,
			Regex:    nb.Regex,
			ParentID: nb.ParentID,
			Order:    nb.Order,
			Level:    nb.Level,
		}
		if nb.ParentID != nil {
			items[i].Parent = &struct {
				ID    uint   `json:"id"`
				Label string `json:"label"`
			}{ID: *nb.ParentID, Label: nb.ParentLabel}
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
		navigationBarError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":           nav.ID,
		"key":          nav.Key,
		"label":        nav.Label,
		"path":         nav.Path,
		"regex":        nav.Regex,
		"parent_id":    nav.ParentID,
		"parent_label": nav.ParentLabel,
		"order":        nav.Order,
		"level":        nav.Level,
		"lft":          nav.Lft,
		"rght":         nav.Rght,
		"tree_id":      nav.TreeID,
	})
}

// AdminNavigationBarCreate - POST /api/v2/admin/navigation-bars
func AdminNavigationBarCreate(c *gin.Context) {
	var req struct {
		Key      string `json:"key" binding:"required"`
		Label    string `json:"label" binding:"required"`
		Path     string `json:"path" binding:"required"`
		Regex    string `json:"regex"`
		ParentID *uint  `json:"parent_id"`
		Order    int    `json:"order"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, apiError(err.Error()))
		return
	}

	nav, err := getNavigationService().CreateNavigation(navigation.CreateNavigationRequest{
		Key:      req.Key,
		Label:    req.Label,
		Path:     req.Path,
		Regex:    req.Regex,
		ParentID: req.ParentID,
		Order:    req.Order,
	})
	if err != nil {
		navigationBarError(c, err)
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
		Label    *string `json:"label"`
		Path     *string `json:"path"`
		Regex    *string `json:"regex"`
		ParentID *uint   `json:"parent_id"`
		Order    *int    `json:"order"`
	}
	// Read the body raw: "parent_id": null (move to root) must be
	// distinguishable from an absent key (leave the parent unchanged).
	body, err := c.GetRawData()
	if err != nil {
		c.JSON(http.StatusBadRequest, apiError(err.Error()))
		return
	}
	if err := json.Unmarshal(body, &req); err != nil {
		c.JSON(http.StatusBadRequest, apiError(err.Error()))
		return
	}
	var fields map[string]json.RawMessage
	_ = json.Unmarshal(body, &fields)
	_, setParent := fields["parent_id"]

	updateReq := navigation.UpdateNavigationRequest{
		NavID:     id,
		Label:     req.Label,
		Path:      req.Path,
		Regex:     req.Regex,
		ParentID:  req.ParentID,
		Order:     req.Order,
		SetParent: setParent,
	}

	if _, err := getNavigationService().UpdateNavigation(updateReq); err != nil {
		navigationBarError(c, err)
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

	if err := getNavigationService().DeleteNavigation(navigation.DeleteNavigationRequest{
		NavID: id,
	}); err != nil {
		navigationBarError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "navigation bar deleted",
	})
}
