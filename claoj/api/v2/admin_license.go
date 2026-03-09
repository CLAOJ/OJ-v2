package v2

import (
	"errors"
	"net/http"

	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/CLAOJ/claoj/service/license"
	"github.com/gin-gonic/gin"
)

// ============================================================
// ADMIN LICENSE MANAGEMENT API
// ============================================================

// AdminLicenseList - GET /api/v2/admin/licenses
// List all licenses
func AdminLicenseList(c *gin.Context) {
	page, pageSize := parsePagination(c)

	resp, err := getLicenseService().ListLicenses(license.ListLicensesRequest{
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	type LicenseItem struct {
		ID      uint   `json:"id"`
		Key     string `json:"key"`
		Name    string `json:"name"`
		Link    string `json:"link"`
		Display string `json:"display"`
		Icon    string `json:"icon"`
	}

	items := make([]LicenseItem, len(resp.Licenses))
	for i, lic := range resp.Licenses {
		items[i] = LicenseItem{
			ID:      lic.ID,
			Key:     lic.Key,
			Name:    lic.Name,
			Link:    lic.Link,
			Display: lic.Display,
			Icon:    lic.Icon,
		}
	}

	c.JSON(http.StatusOK, apiListWithTotal(items, resp.Total))
}

// AdminLicenseDetail - GET /api/v2/admin/license/:id
// Get license detail
func AdminLicenseDetail(c *gin.Context) {
	idStr := c.Param("id")
	var id uint
	if err := parseUint(idStr, &id); err != nil {
		c.JSON(http.StatusBadRequest, apiError("invalid license id"))
		return
	}

	lic, err := getLicenseService().GetLicense(license.GetLicenseRequest{
		LicenseID: id,
	})
	if err != nil {
		if errors.Is(err, license.ErrLicenseNotFound) {
			c.JSON(http.StatusNotFound, apiError("license not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":      lic.ID,
		"key":     lic.Key,
		"name":    lic.Name,
		"link":    lic.Link,
		"display": lic.Display,
		"icon":    lic.Icon,
		"text":    lic.Text,
	})
}

// AdminLicenseCreateRequest - POST /api/v2/admin/licenses
type AdminLicenseCreateRequest struct {
	Key     string `json:"key" binding:"required"`
	Link    string `json:"link" binding:"required"`
	Name    string `json:"name" binding:"required"`
	Display string `json:"display"`
	Icon    string `json:"icon"`
	Text    string `json:"text"`
}

// AdminLicenseCreate - POST /api/v2/admin/licenses
// Create a new license
func AdminLicenseCreate(c *gin.Context) {
	user, _, ok := resolveUserProfile(c)
	if !ok {
		return
	}

	if !user.IsSuperuser {
		c.JSON(http.StatusForbidden, apiError("admin access required"))
		return
	}

	var req AdminLicenseCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, apiError(err.Error()))
		return
	}

	lic, err := getLicenseService().CreateLicense(license.CreateLicenseRequest{
		Key:     req.Key,
		Link:    req.Link,
		Name:    req.Name,
		Display: req.Display,
		Icon:    req.Icon,
		Text:    req.Text,
	})
	if err != nil {
		if errors.Is(err, license.ErrKeyExists) {
			c.JSON(http.StatusBadRequest, apiError("license key already exists"))
			return
		}
		if errors.Is(err, license.ErrEmptyKey) {
			c.JSON(http.StatusBadRequest, apiError("license key cannot be empty"))
			return
		}
		if errors.Is(err, license.ErrEmptyName) {
			c.JSON(http.StatusBadRequest, apiError("license name cannot be empty"))
			return
		}
		if errors.Is(err, license.ErrEmptyLink) {
			c.JSON(http.StatusBadRequest, apiError("license link cannot be empty"))
			return
		}
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "license created",
		"license": gin.H{"id": lic.ID, "key": lic.Key},
	})
}

// AdminLicenseUpdateRequest - PATCH /api/v2/admin/license/:id
type AdminLicenseUpdateRequest struct {
	Link    *string `json:"link"`
	Name    *string `json:"name"`
	Display *string `json:"display"`
	Icon    *string `json:"icon"`
	Text    *string `json:"text"`
}

// AdminLicenseUpdate - PATCH /api/v2/admin/license/:id
// Update a license
func AdminLicenseUpdate(c *gin.Context) {
	user, _, ok := resolveUserProfile(c)
	if !ok {
		return
	}

	if !user.IsSuperuser {
		c.JSON(http.StatusForbidden, apiError("admin access required"))
		return
	}

	idStr := c.Param("id")
	var id uint
	if err := parseUint(idStr, &id); err != nil {
		c.JSON(http.StatusBadRequest, apiError("invalid license id"))
		return
	}

	var req AdminLicenseUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, apiError(err.Error()))
		return
	}

	lic, err := getLicenseService().UpdateLicense(license.UpdateLicenseRequest{
		LicenseID: id,
		Link:      req.Link,
		Name:      req.Name,
		Display:   req.Display,
		Icon:      req.Icon,
		Text:      req.Text,
	})
	if err != nil {
		if errors.Is(err, license.ErrLicenseNotFound) {
			c.JSON(http.StatusNotFound, apiError("license not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "license updated",
		"license": gin.H{"id": lic.ID, "key": lic.Key},
	})
}

// AdminLicenseDelete - DELETE /api/v2/admin/license/:id
// Delete a license
func AdminLicenseDelete(c *gin.Context) {
	user, _, ok := resolveUserProfile(c)
	if !ok {
		return
	}

	if !user.IsSuperuser {
		c.JSON(http.StatusForbidden, apiError("admin access required"))
		return
	}

	idStr := c.Param("id")
	var id uint
	if err := parseUint(idStr, &id); err != nil {
		c.JSON(http.StatusBadRequest, apiError("invalid license id"))
		return
	}

	// Check if license is used by problems
	var problemCount int64
	db.DB.Model(&models.Problem{}).Where("license_id = ?", id).Count(&problemCount)
	if problemCount > 0 {
		c.JSON(http.StatusBadRequest, apiError("cannot delete license used by problems"))
		return
	}

	if err := getLicenseService().DeleteLicense(license.DeleteLicenseRequest{
		LicenseID: id,
	}); err != nil {
		if errors.Is(err, license.ErrLicenseNotFound) {
			c.JSON(http.StatusNotFound, apiError("license not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "license deleted",
	})
}
