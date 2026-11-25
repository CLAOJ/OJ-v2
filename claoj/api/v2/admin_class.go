package v2

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ============================================================
// ADMIN CLASS CRUD (v1 ClassAdmin: organization-scoped student groups)
// ============================================================

// classAdminRef is a profile reference for the admins/members pickers.
type classAdminRef struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
}

func classProfilesRefs(profiles []models.Profile) []classAdminRef {
	out := make([]classAdminRef, len(profiles))
	for i, p := range profiles {
		out[i] = classAdminRef{ID: p.ID, Username: p.User.Username}
	}
	return out
}

// classDetail is the shape the admin class form edits.
func classDetail(class *models.Class) gin.H {
	return gin.H{
		"id":                class.ID,
		"name":              class.Name,
		"slug":              class.Slug,
		"description":       class.Description,
		"is_active":         class.IsActive,
		"access_code":       class.AccessCode,
		"organization_id":   class.OrganizationID,
		"organization_name": class.Organization.Name,
		"admins":            classProfilesRefs(class.Admins),
		"members":           classProfilesRefs(class.Members),
	}
}

func loadClass(id string) (*models.Class, bool) {
	var class models.Class
	if err := db.DB.Preload("Organization").
		Preload("Admins.User").
		Preload("Members.User").
		First(&class, id).Error; err != nil {
		return nil, false
	}
	return &class, true
}

// normalizeClassSlug derives the slug from the name when it is blank, as v1's
// prepopulated_fields does in the admin form.
func normalizeClassSlug(name, slug string) string {
	if slug != "" {
		return slug
	}
	lower := strings.ToLower(strings.TrimSpace(name))
	var b strings.Builder
	lastDash := true
	for _, r := range lower {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '_':
			b.WriteRune(r)
			lastDash = false
		default:
			if !lastDash {
				b.WriteByte('-')
				lastDash = true
			}
		}
	}
	return strings.Trim(b.String(), "-")
}

// lookupProfiles keeps only real profile ids; a bogus id would otherwise fail
// the many2many insert on the live MySQL schema.
func lookupProfiles(ids []uint) ([]models.Profile, bool) {
	if len(ids) == 0 {
		return nil, true
	}
	var profiles []models.Profile
	if err := db.DB.Where("id IN ?", ids).Find(&profiles).Error; err != nil {
		return nil, false
	}
	if len(profiles) != len(ids) {
		return nil, false
	}
	return profiles, true
}

// AdminClassList - GET /api/v2/admin/classes
// Picker source for contest forms; now also carries the CRUD list columns.
func AdminClassList(c *gin.Context) {
	page, pageSize := parsePagination(c)

	search := strings.TrimSpace(c.Query("search"))
	orgID := strings.TrimSpace(c.Query("organization_id"))
	build := func() *gorm.DB {
		q := db.DB.Model(&models.Class{})
		if search != "" {
			pattern := db.ContainsPattern(search)
			q = q.Where("name LIKE ? ESCAPE '!' OR slug LIKE ? ESCAPE '!'", pattern, pattern)
		}
		if orgID != "" {
			q = q.Where("organization_id = ?", orgID)
		}
		return q
	}

	var total int64
	if err := build().Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	var classes []models.Class
	if err := build().Preload("Organization").
		Order("name ASC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&classes).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	ids := make([]uint, len(classes))
	for i, class := range classes {
		ids[i] = class.ID
	}
	adminCounts := map[uint]int64{}
	memberCounts := map[uint]int64{}
	if len(ids) > 0 {
		type countRow struct {
			ClassID uint
			Count   int64
		}
		var rows []countRow
		db.DB.Table("judge_class_admins").Select("class_id, COUNT(*) AS count").
			Where("class_id IN ?", ids).Group("class_id").Scan(&rows)
		for _, r := range rows {
			adminCounts[r.ClassID] = r.Count
		}
		rows = nil
		db.DB.Table("judge_class_members").Select("class_id, COUNT(*) AS count").
			Where("class_id IN ?", ids).Group("class_id").Scan(&rows)
		for _, r := range rows {
			memberCounts[r.ClassID] = r.Count
		}
	}

	type Item struct {
		ID               uint    `json:"id"`
		Name             string  `json:"name"`
		Slug             string  `json:"slug"`
		Description      string  `json:"description"`
		IsActive         bool    `json:"is_active"`
		AccessCode       *string `json:"access_code"`
		OrganizationID   uint    `json:"organization_id"`
		OrganizationName string  `json:"organization_name"`
		AdminCount       int64   `json:"admin_count"`
		MemberCount      int64   `json:"member_count"`
	}
	items := make([]Item, len(classes))
	for i, class := range classes {
		items[i] = Item{
			ID:               class.ID,
			Name:             class.Name,
			Slug:             class.Slug,
			Description:      class.Description,
			IsActive:         class.IsActive,
			AccessCode:       class.AccessCode,
			OrganizationID:   class.OrganizationID,
			OrganizationName: class.Organization.Name,
			AdminCount:       adminCounts[class.ID],
			MemberCount:      memberCounts[class.ID],
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"data":      items,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// AdminClassDetail - GET /api/v2/admin/class/:id
func AdminClassDetail(c *gin.Context) {
	id, ok := numericParam(c, "id")
	if !ok {
		return
	}
	class, found := loadClass(id)
	if !found {
		c.JSON(http.StatusNotFound, apiUserError("class not found"))
		return
	}
	c.JSON(http.StatusOK, classDetail(class))
}

type AdminClassRequest struct {
	Name           string  `json:"name" binding:"required"`
	Slug           string  `json:"slug"`
	OrganizationID uint    `json:"organization_id" binding:"required"`
	Description    string  `json:"description"`
	IsActive       *bool   `json:"is_active"`
	AccessCode     *string `json:"access_code"`
	AdminIDs       []uint  `json:"admin_ids"`
	MemberIDs      []uint  `json:"member_ids"`
}

// AdminClassCreate - POST /api/v2/admin/classes
func AdminClassCreate(c *gin.Context) {
	var req AdminClassRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, apiUserError(err.Error()))
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		c.JSON(http.StatusBadRequest, apiUserError("name is required"))
		return
	}
	if req.AccessCode != nil && len(*req.AccessCode) > 7 {
		c.JSON(http.StatusBadRequest, apiUserError("access code must be at most 7 characters"))
		return
	}

	var org models.Organization
	if err := db.DB.First(&org, req.OrganizationID).Error; err != nil {
		c.JSON(http.StatusBadRequest, apiUserError("organization not found"))
		return
	}
	var existing int64
	db.DB.Model(&models.Class{}).Where("name = ?", req.Name).Count(&existing)
	if existing > 0 {
		c.JSON(http.StatusBadRequest, apiUserError("class name already exists"))
		return
	}
	admins, ok := lookupProfiles(req.AdminIDs)
	if !ok {
		c.JSON(http.StatusBadRequest, apiUserError("invalid admin selection"))
		return
	}
	members, ok := lookupProfiles(req.MemberIDs)
	if !ok {
		c.JSON(http.StatusBadRequest, apiUserError("invalid member selection"))
		return
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}
	class := models.Class{
		OrganizationID: req.OrganizationID,
		Name:           req.Name,
		Slug:           normalizeClassSlug(req.Name, strings.TrimSpace(req.Slug)),
		Description:    req.Description,
		IsActive:       isActive,
		AccessCode:     req.AccessCode,
	}
	if err := db.DB.Omit("Admins", "Members", "Organization").Create(&class).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}
	if len(admins) > 0 {
		db.DB.Model(&class).Association("Admins").Append(admins)
	}
	if len(members) > 0 {
		db.DB.Model(&class).Association("Members").Append(members)
	}

	created, found := loadClass(strconv.FormatUint(uint64(class.ID), 10))
	if !found {
		c.JSON(http.StatusInternalServerError, apiError("class created but could not be reloaded"))
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"message": "class created",
		"class":   classDetail(created),
	})
}

type AdminClassUpdateRequest struct {
	Name        *string `json:"name"`
	Slug        *string `json:"slug"`
	Description *string `json:"description"`
	IsActive    *bool   `json:"is_active"`
	AccessCode  *string `json:"access_code"`
	AdminIDs    *[]uint `json:"admin_ids"`
	MemberIDs   *[]uint `json:"member_ids"`
}

// AdminClassUpdate - PATCH /api/v2/admin/class/:id
func AdminClassUpdate(c *gin.Context) {
	id, ok := numericParam(c, "id")
	if !ok {
		return
	}
	class, found := loadClass(id)
	if !found {
		c.JSON(http.StatusNotFound, apiUserError("class not found"))
		return
	}

	var req AdminClassUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, apiUserError(err.Error()))
		return
	}

	updates := map[string]any{}
	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			c.JSON(http.StatusBadRequest, apiUserError("name cannot be empty"))
			return
		}
		var existing int64
		db.DB.Model(&models.Class{}).Where("name = ? AND id <> ?", name, class.ID).Count(&existing)
		if existing > 0 {
			c.JSON(http.StatusBadRequest, apiUserError("class name already exists"))
			return
		}
		updates["name"] = name
	}
	if req.Slug != nil {
		updates["slug"] = normalizeClassSlug(class.Name, strings.TrimSpace(*req.Slug))
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
	}
	if req.AccessCode != nil {
		if len(*req.AccessCode) > 7 {
			c.JSON(http.StatusBadRequest, apiUserError("access code must be at most 7 characters"))
			return
		}
		updates["access_code"] = *req.AccessCode
	}
	if len(updates) > 0 {
		if err := db.DB.Model(class).Updates(updates).Error; err != nil {
			c.JSON(http.StatusInternalServerError, apiError(err.Error()))
			return
		}
	}

	if req.AdminIDs != nil {
		admins, ok := lookupProfiles(*req.AdminIDs)
		if !ok {
			c.JSON(http.StatusBadRequest, apiUserError("invalid admin selection"))
			return
		}
		if err := db.DB.Model(class).Association("Admins").Replace(admins); err != nil {
			c.JSON(http.StatusInternalServerError, apiError(err.Error()))
			return
		}
	}
	if req.MemberIDs != nil {
		members, ok := lookupProfiles(*req.MemberIDs)
		if !ok {
			c.JSON(http.StatusBadRequest, apiUserError("invalid member selection"))
			return
		}
		if err := db.DB.Model(class).Association("Members").Replace(members); err != nil {
			c.JSON(http.StatusInternalServerError, apiError(err.Error()))
			return
		}
	}

	updated, found := loadClass(id)
	if !found {
		c.JSON(http.StatusInternalServerError, apiError("class updated but could not be reloaded"))
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "class updated",
		"class":   classDetail(updated),
	})
}

// AdminClassDelete - DELETE /api/v2/admin/class/:id
func AdminClassDelete(c *gin.Context) {
	id, ok := numericParam(c, "id")
	if !ok {
		return
	}
	class, found := loadClass(id)
	if !found {
		c.JSON(http.StatusNotFound, apiUserError("class not found"))
		return
	}
	if err := db.DB.Model(class).Association("Admins").Clear(); err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}
	if err := db.DB.Model(class).Association("Members").Clear(); err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}
	// Contests reference classes through judge_contest_classes; clear those
	// rows too so the delete cannot violate the join table's FK.
	db.DB.Exec("DELETE FROM judge_contest_classes WHERE class_id = ?", class.ID)
	if err := db.DB.Delete(&models.Class{}, class.ID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "class deleted"})
}