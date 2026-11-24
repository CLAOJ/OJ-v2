package v2

import (
	"errors"
	"net/http"
	"strings"

	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ============================================================
// FLAT PAGES + REDIRECTS (Django contrib tables, single site)
// ============================================================
//
// v1 renders flatpages and redirects through Django's fallback middleware.
// v2 serves the same rows: the admin CRUD below writes django_flatpage /
// django_flatpage_sites / django_redirect exactly like Django admin, and the
// two public endpoints feed the Next.js catch-all route and proxy.
// Rows are always scoped to models.CMSSiteID (single-site deployment).

// validCMSPath reports whether p is an absolute site path ("/about/").
func validCMSPath(p string) bool { return strings.HasPrefix(p, "/") }

// validRedirectTarget accepts a site path or an absolute http(s) URL, which is
// what Django's redirects app allows as new_path.
func validRedirectTarget(p string) bool {
	return validCMSPath(p) || strings.HasPrefix(p, "http://") || strings.HasPrefix(p, "https://")
}

// flatPageURLTaken reports whether a flatpage with this url is already attached
// to the v2 site. Django enforces uniqueness per site, not per table.
func flatPageURLTaken(url string, excludeID uint) (bool, error) {
	q := db.DB.Table("django_flatpage").
		Joins("JOIN django_flatpage_sites ON django_flatpage_sites.flatpage_id = django_flatpage.id").
		Where("django_flatpage.url = ? AND django_flatpage_sites.site_id = ?", url, models.CMSSiteID)
	if excludeID != 0 {
		q = q.Where("django_flatpage.id <> ?", excludeID)
	}
	var n int64
	if err := q.Count(&n).Error; err != nil {
		return false, err
	}
	return n > 0, nil
}

func redirectPathTaken(oldPath string, excludeID uint) (bool, error) {
	q := db.DB.Model(&models.Redirect{}).
		Where("site_id = ? AND old_path = ?", models.CMSSiteID, oldPath)
	if excludeID != 0 {
		q = q.Where("id <> ?", excludeID)
	}
	var n int64
	if err := q.Count(&n).Error; err != nil {
		return false, err
	}
	return n > 0, nil
}

// ============================================================
// ADMIN: FLAT PAGES
// ============================================================

// AdminFlatPageList - GET /api/v2/admin/flat-pages
func AdminFlatPageList(c *gin.Context) {
	page, pageSize := parsePagination(c)
	search := strings.TrimSpace(c.Query("search"))

	query := func() *gorm.DB {
		q := db.DB.Model(&models.FlatPage{})
		if search != "" {
			like := "%" + search + "%"
			q = q.Where("url LIKE ? OR title LIKE ?", like, like)
		}
		return q
	}

	var total int64
	if err := query().Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	var pages []models.FlatPage
	if err := query().Order("url ASC").
		Offset((page - 1) * pageSize).Limit(pageSize).Find(&pages).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	type Item struct {
		ID                   uint   `json:"id"`
		URL                  string `json:"url"`
		Title                string `json:"title"`
		RegistrationRequired bool   `json:"registration_required"`
	}
	items := make([]Item, len(pages))
	for i, p := range pages {
		items[i] = Item{p.ID, p.URL, p.Title, p.RegistrationRequired}
	}

	c.JSON(http.StatusOK, apiListWithTotal(items, total))
}

// AdminFlatPageDetail - GET /api/v2/admin/flat-page/:id
func AdminFlatPageDetail(c *gin.Context) {
	idStr, ok := numericParam(c, "id")
	if !ok {
		return
	}
	var page models.FlatPage
	if err := db.DB.First(&page, idStr).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, apiUserError("flat page not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"id":                    page.ID,
		"url":                   page.URL,
		"title":                 page.Title,
		"content":               page.Content,
		"enable_comments":       page.EnableComments,
		"template_name":         page.TemplateName,
		"registration_required": page.RegistrationRequired,
	})
}

// AdminFlatPageCreate - POST /api/v2/admin/flat-pages
func AdminFlatPageCreate(c *gin.Context) {
	var req struct {
		URL                  string `json:"url" binding:"required"`
		Title                string `json:"title" binding:"required"`
		Content              string `json:"content"`
		RegistrationRequired bool   `json:"registration_required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, apiUserError(err.Error()))
		return
	}
	req.URL = strings.TrimSpace(req.URL)
	if !validCMSPath(req.URL) {
		c.JSON(http.StatusBadRequest, apiUserError("url must start with /"))
		return
	}

	taken, err := flatPageURLTaken(req.URL, 0)
	if err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}
	if taken {
		c.JSON(http.StatusConflict, apiUserError("a flat page with this url already exists"))
		return
	}

	page := models.FlatPage{
		URL:                  req.URL,
		Title:                req.Title,
		Content:              req.Content,
		RegistrationRequired: req.RegistrationRequired,
	}
	err = db.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&page).Error; err != nil {
			return err
		}
		return tx.Create(&models.FlatPageSite{FlatPageID: page.ID, SiteID: models.CMSSiteID}).Error
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":   "flat page created",
		"flat_page": gin.H{"id": page.ID, "url": page.URL},
	})
}

// AdminFlatPageUpdate - PATCH /api/v2/admin/flat-page/:id
func AdminFlatPageUpdate(c *gin.Context) {
	idStr, ok := numericParam(c, "id")
	if !ok {
		return
	}
	var page models.FlatPage
	if err := db.DB.First(&page, idStr).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, apiUserError("flat page not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	var req struct {
		URL                  *string `json:"url"`
		Title                *string `json:"title"`
		Content              *string `json:"content"`
		RegistrationRequired *bool   `json:"registration_required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, apiUserError(err.Error()))
		return
	}

	if req.URL != nil {
		url := strings.TrimSpace(*req.URL)
		if !validCMSPath(url) {
			c.JSON(http.StatusBadRequest, apiUserError("url must start with /"))
			return
		}
		if url != page.URL {
			taken, err := flatPageURLTaken(url, page.ID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, apiError(err.Error()))
				return
			}
			if taken {
				c.JSON(http.StatusConflict, apiUserError("a flat page with this url already exists"))
				return
			}
		}
		page.URL = url
	}
	if req.Title != nil {
		page.Title = *req.Title
	}
	if req.Content != nil {
		page.Content = *req.Content
	}
	if req.RegistrationRequired != nil {
		page.RegistrationRequired = *req.RegistrationRequired
	}

	if err := db.DB.Save(&page).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "flat page updated"})
}

// AdminFlatPageDelete - DELETE /api/v2/admin/flat-page/:id
func AdminFlatPageDelete(c *gin.Context) {
	idStr, ok := numericParam(c, "id")
	if !ok {
		return
	}
	var page models.FlatPage
	if err := db.DB.First(&page, idStr).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, apiUserError("flat page not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	// The m2m has no ON DELETE CASCADE for us: Django deletes the through rows.
	err := db.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("flatpage_id = ?", page.ID).Delete(&models.FlatPageSite{}).Error; err != nil {
			return err
		}
		return tx.Delete(&models.FlatPage{}, page.ID).Error
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "flat page deleted"})
}

// ============================================================
// ADMIN: REDIRECTS
// ============================================================

// AdminRedirectList - GET /api/v2/admin/redirects
func AdminRedirectList(c *gin.Context) {
	page, pageSize := parsePagination(c)
	search := strings.TrimSpace(c.Query("search"))

	query := func() *gorm.DB {
		q := db.DB.Model(&models.Redirect{}).Where("site_id = ?", models.CMSSiteID)
		if search != "" {
			like := "%" + search + "%"
			q = q.Where("old_path LIKE ? OR new_path LIKE ?", like, like)
		}
		return q
	}

	var total int64
	if err := query().Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	var redirects []models.Redirect
	if err := query().Order("old_path ASC").
		Offset((page - 1) * pageSize).Limit(pageSize).Find(&redirects).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	type Item struct {
		ID      uint   `json:"id"`
		OldPath string `json:"old_path"`
		NewPath string `json:"new_path"`
	}
	items := make([]Item, len(redirects))
	for i, r := range redirects {
		items[i] = Item{r.ID, r.OldPath, r.NewPath}
	}

	c.JSON(http.StatusOK, apiListWithTotal(items, total))
}

// AdminRedirectDetail - GET /api/v2/admin/redirect/:id
func AdminRedirectDetail(c *gin.Context) {
	idStr, ok := numericParam(c, "id")
	if !ok {
		return
	}
	var r models.Redirect
	if err := db.DB.First(&r, idStr).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, apiUserError("redirect not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": r.ID, "old_path": r.OldPath, "new_path": r.NewPath})
}

// validateRedirect rejects paths Django's redirects admin would never store:
// a target that equals the source is an immediate loop.
func validateRedirect(c *gin.Context, oldPath, newPath string) bool {
	if !validCMSPath(oldPath) {
		c.JSON(http.StatusBadRequest, apiUserError("old_path must start with /"))
		return false
	}
	if !validRedirectTarget(newPath) {
		c.JSON(http.StatusBadRequest, apiUserError("new_path must start with / or be an absolute http(s) URL"))
		return false
	}
	if oldPath == newPath {
		c.JSON(http.StatusBadRequest, apiUserError("old_path and new_path must differ"))
		return false
	}
	return true
}

// AdminRedirectCreate - POST /api/v2/admin/redirects
func AdminRedirectCreate(c *gin.Context) {
	var req struct {
		OldPath string `json:"old_path" binding:"required"`
		NewPath string `json:"new_path" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, apiUserError(err.Error()))
		return
	}
	req.OldPath = strings.TrimSpace(req.OldPath)
	req.NewPath = strings.TrimSpace(req.NewPath)
	if !validateRedirect(c, req.OldPath, req.NewPath) {
		return
	}

	taken, err := redirectPathTaken(req.OldPath, 0)
	if err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}
	if taken {
		c.JSON(http.StatusConflict, apiUserError("a redirect for this old_path already exists"))
		return
	}

	r := models.Redirect{SiteID: models.CMSSiteID, OldPath: req.OldPath, NewPath: req.NewPath}
	if err := db.DB.Create(&r).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"message":  "redirect created",
		"redirect": gin.H{"id": r.ID, "old_path": r.OldPath},
	})
}

// AdminRedirectUpdate - PATCH /api/v2/admin/redirect/:id
func AdminRedirectUpdate(c *gin.Context) {
	idStr, ok := numericParam(c, "id")
	if !ok {
		return
	}
	var r models.Redirect
	if err := db.DB.First(&r, idStr).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, apiUserError("redirect not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	var req struct {
		OldPath *string `json:"old_path"`
		NewPath *string `json:"new_path"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, apiUserError(err.Error()))
		return
	}

	oldPath, newPath := r.OldPath, r.NewPath
	if req.OldPath != nil {
		oldPath = strings.TrimSpace(*req.OldPath)
	}
	if req.NewPath != nil {
		newPath = strings.TrimSpace(*req.NewPath)
	}
	if !validateRedirect(c, oldPath, newPath) {
		return
	}
	if oldPath != r.OldPath {
		taken, err := redirectPathTaken(oldPath, r.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, apiError(err.Error()))
			return
		}
		if taken {
			c.JSON(http.StatusConflict, apiUserError("a redirect for this old_path already exists"))
			return
		}
	}

	r.OldPath, r.NewPath = oldPath, newPath
	if err := db.DB.Save(&r).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "redirect updated"})
}

// AdminRedirectDelete - DELETE /api/v2/admin/redirect/:id
func AdminRedirectDelete(c *gin.Context) {
	idStr, ok := numericParam(c, "id")
	if !ok {
		return
	}
	result := db.DB.Delete(&models.Redirect{}, idStr)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, apiError(result.Error.Error()))
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, apiUserError("redirect not found"))
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "redirect deleted"})
}

// ============================================================
// PUBLIC: FLAT PAGE LOOKUP + REDIRECT MAP
// ============================================================

// flatPageCandidates normalizes a requested path the way Django's flatpage view
// does: a leading slash is added, and APPEND_SLASH lets "/about" match "/about/".
func flatPageCandidates(raw string) []string {
	u := strings.TrimSpace(raw)
	if u == "" {
		return nil
	}
	if !validCMSPath(u) {
		u = "/" + u
	}
	if strings.HasSuffix(u, "/") {
		return []string{u}
	}
	return []string{u, u + "/"}
}

// FlatPageLookup - GET /api/v2/page?url=/about/
// Public read for the web catch-all route. A registration-required page is a
// 404 for anonymous visitors, matching Django's FlatpageFallbackMiddleware
// (it serves such pages only when request.user.is_authenticated).
func FlatPageLookup(c *gin.Context) {
	candidates := flatPageCandidates(c.Query("url"))
	if len(candidates) == 0 {
		c.JSON(http.StatusBadRequest, apiUserError("url is required"))
		return
	}

	var page models.FlatPage
	found := false
	for _, u := range candidates {
		err := db.DB.Model(&models.FlatPage{}).
			Select("django_flatpage.*").
			Joins("JOIN django_flatpage_sites ON django_flatpage_sites.flatpage_id = django_flatpage.id").
			Where("django_flatpage.url = ? AND django_flatpage_sites.site_id = ?", u, models.CMSSiteID).
			First(&page).Error
		if err == nil {
			found = true
			break
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusInternalServerError, apiError(err.Error()))
			return
		}
	}
	if !found {
		c.JSON(http.StatusNotFound, apiUserError("flat page not found"))
		return
	}

	_, authenticated := c.Get("user_id")
	if page.RegistrationRequired {
		// Never cacheable: the same URL is a 404 for anonymous visitors.
		c.Header("Cache-Control", "private, no-store")
		if !authenticated {
			c.JSON(http.StatusNotFound, apiUserError("flat page not found"))
			return
		}
	} else {
		c.Header("Cache-Control", "public, max-age=60")
	}
	c.JSON(http.StatusOK, gin.H{
		"url":                   page.URL,
		"title":                 page.Title,
		"content":               page.Content,
		"registration_required": page.RegistrationRequired,
	})
}

// RedirectMap - GET /api/v2/redirects-map
// Public old_path -> new_path pairs for the web proxy. The table is small and
// changes rarely, so it is served whole and cached for a minute.
func RedirectMap(c *gin.Context) {
	var redirects []models.Redirect
	if err := db.DB.Select("old_path", "new_path").
		Where("site_id = ?", models.CMSSiteID).
		Order("old_path ASC").Find(&redirects).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	type Item struct {
		OldPath string `json:"old_path"`
		NewPath string `json:"new_path"`
	}
	items := make([]Item, len(redirects))
	for i, r := range redirects {
		items[i] = Item{r.OldPath, r.NewPath}
	}

	c.Header("Cache-Control", "public, max-age=60")
	c.JSON(http.StatusOK, apiList(items))
}