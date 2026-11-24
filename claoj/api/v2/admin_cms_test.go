package v2

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/CLAOJ/claoj/auth"
	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// ============================================================
// Flat pages + redirects: CRUD, gates, validation, public reads.
// ============================================================

func setupCMSDB(t *testing.T) {
	t.Helper()
	old := db.DB
	t.Cleanup(func() { db.DB = old })
	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	require.NoError(t, err)
	sqlDB, err := database.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, database.AutoMigrate(
		&models.AuthUser{}, &models.AuthGroup{}, &models.DjangoContentType{},
		&models.AuthPermission{}, &models.AuthUserGroup{}, &models.AuthGroupPermission{},
		&models.AuthUserPermission{},
		&models.FlatPage{}, &models.FlatPageSite{}, &models.Redirect{},
	))
	db.DB = database
}

// cmsRouter mounts one route; userID 0 means anonymous (no user_id in context).
func cmsRouter(userID uint, method, path string, handlers ...gin.HandlerFunc) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	all := make([]gin.HandlerFunc, 0, len(handlers)+1)
	if userID != 0 {
		all = append(all, func(c *gin.Context) { c.Set("user_id", userID) })
	}
	all = append(all, handlers...)
	router.Handle(method, path, all...)
	return router
}

func seedFlatPage(t *testing.T, url, title, content string, siteID uint, registrationRequired bool) models.FlatPage {
	t.Helper()
	page := models.FlatPage{URL: url, Title: title, Content: content, RegistrationRequired: registrationRequired}
	require.NoError(t, db.DB.Create(&page).Error)
	require.NoError(t, db.DB.Create(&models.FlatPageSite{FlatPageID: page.ID, SiteID: siteID}).Error)
	return page
}

func TestAdminFlatPageCRUD(t *testing.T) {
	setupCMSDB(t)
	router := cmsRouter(1, http.MethodPost, "/admin/flat-pages", AdminFlatPageCreate)

	w := doRequest(router, http.MethodPost, "/admin/flat-pages",
		`{"url":"/about/","title":"About","content":"# Hello","registration_required":true}`)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())

	var page models.FlatPage
	require.NoError(t, db.DB.First(&page).Error)
	require.Equal(t, "/about/", page.URL)
	require.True(t, page.RegistrationRequired)

	var site models.FlatPageSite
	require.NoError(t, db.DB.First(&site).Error)
	require.Equal(t, page.ID, site.FlatPageID)
	require.Equal(t, models.CMSSiteID, site.SiteID)

	// List + search + total
	listRouter := cmsRouter(1, http.MethodGet, "/admin/flat-pages", AdminFlatPageList)
	w = doRequest(listRouter, http.MethodGet, "/admin/flat-pages?search=about", "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var list struct {
		Data  []struct{ ID uint `json:"id"` } `json:"data"`
		Total int64                          `json:"total"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &list))
	require.Equal(t, int64(1), list.Total)
	w = doRequest(listRouter, http.MethodGet, "/admin/flat-pages?search=nomatch", "")
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &list))
	require.Equal(t, int64(0), list.Total)

	// Detail
	detailRouter := cmsRouter(1, http.MethodGet, "/admin/flat-page/:id", AdminFlatPageDetail)
	w = doRequest(detailRouter, http.MethodGet, "/admin/flat-page/1", "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var detail map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &detail))
	require.Equal(t, "# Hello", detail["content"])

	// Update
	updateRouter := cmsRouter(1, http.MethodPatch, "/admin/flat-page/:id", AdminFlatPageUpdate)
	w = doRequest(updateRouter, http.MethodPatch, "/admin/flat-page/1", `{"title":"About us","url":"/about-us/"}`)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NoError(t, db.DB.First(&page).Error)
	require.Equal(t, "About us", page.Title)
	require.Equal(t, "/about-us/", page.URL)

	// Delete removes the through row too.
	deleteRouter := cmsRouter(1, http.MethodDelete, "/admin/flat-page/:id", AdminFlatPageDelete)
	w = doRequest(deleteRouter, http.MethodDelete, "/admin/flat-page/1", "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var count int64
	require.NoError(t, db.DB.Model(&models.FlatPageSite{}).Count(&count).Error)
	require.Zero(t, count)
}

func TestAdminFlatPageValidationAndDuplicates(t *testing.T) {
	setupCMSDB(t)
	seedFlatPage(t, "/about/", "About", "", models.CMSSiteID, false)

	createRouter := cmsRouter(1, http.MethodPost, "/admin/flat-pages", AdminFlatPageCreate)

	// Duplicate url on the same site -> 409.
	w := doRequest(createRouter, http.MethodPost, "/admin/flat-pages", `{"url":"/about/","title":"Dup"}`)
	require.Equal(t, http.StatusConflict, w.Code, w.Body.String())

	// A relative url is rejected.
	w = doRequest(createRouter, http.MethodPost, "/admin/flat-pages", `{"url":"about","title":"Bad"}`)
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())

	// The same url for a different site is not a conflict here.
	seedFlatPage(t, "/other/", "Other", "", 2, false)
	w = doRequest(createRouter, http.MethodPost, "/admin/flat-pages", `{"url":"/other/","title":"Clash"}`)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
}

func TestAdminFlatPageUpdateGuards(t *testing.T) {
	setupCMSDB(t)
	seedFlatPage(t, "/about/", "About", "", models.CMSSiteID, false)
	seedFlatPage(t, "/contact/", "Contact", "", models.CMSSiteID, false)

	updateRouter := cmsRouter(1, http.MethodPatch, "/admin/flat-page/:id", AdminFlatPageUpdate)

	w := doRequest(updateRouter, http.MethodPatch, "/admin/flat-page/1", `{"url":"/contact/"}`)
	require.Equal(t, http.StatusConflict, w.Code, w.Body.String())

	// Saving without changing the url must not trip the duplicate check.
	w = doRequest(updateRouter, http.MethodPatch, "/admin/flat-page/1", `{"url":"/about/","title":"About 2"}`)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	w = doRequest(updateRouter, http.MethodPatch, "/admin/flat-page/1", `{"url":"relative"}`)
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
}

func TestAdminCMSWriteGates(t *testing.T) {
	setupCMSDB(t)
	staff := models.AuthUser{Username: "staff", IsActive: true, IsStaff: true}
	require.NoError(t, db.DB.Create(&staff).Error)
	admin := models.AuthUser{Username: "root", IsActive: true, IsStaff: true, IsSuperuser: true}
	require.NoError(t, db.DB.Create(&admin).Error)

	flatWrite := cmsRouter(staff.ID, http.MethodPost, "/admin/flat-pages",
		auth.RequirePerm("flatpages.change_flatpage"), AdminFlatPageCreate)
	w := doRequest(flatWrite, http.MethodPost, "/admin/flat-pages", `{"url":"/a/","title":"A"}`)
	require.Equal(t, http.StatusForbidden, w.Code, w.Body.String())

	redirectWrite := cmsRouter(staff.ID, http.MethodPost, "/admin/redirects",
		auth.RequirePerm("redirects.change_redirect"), AdminRedirectCreate)
	w = doRequest(redirectWrite, http.MethodPost, "/admin/redirects", `{"old_path":"/a/","new_path":"/b/"}`)
	require.Equal(t, http.StatusForbidden, w.Code, w.Body.String())

	// Superuser bypasses the codename (Django ModelBackend semantics).
	flatWrite = cmsRouter(admin.ID, http.MethodPost, "/admin/flat-pages",
		auth.RequirePerm("flatpages.change_flatpage"), AdminFlatPageCreate)
	w = doRequest(flatWrite, http.MethodPost, "/admin/flat-pages", `{"url":"/a/","title":"A"}`)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
}

func TestAdminRedirectCRUDAndValidation(t *testing.T) {
	setupCMSDB(t)
	createRouter := cmsRouter(1, http.MethodPost, "/admin/redirects", AdminRedirectCreate)

	w := doRequest(createRouter, http.MethodPost, "/admin/redirects", `{"old_path":"/old/","new_path":"/new/"}`)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())

	var r models.Redirect
	require.NoError(t, db.DB.First(&r).Error)
	require.Equal(t, models.CMSSiteID, r.SiteID)

	// Duplicate old_path -> 409
	w = doRequest(createRouter, http.MethodPost, "/admin/redirects", `{"old_path":"/old/","new_path":"/other/"}`)
	require.Equal(t, http.StatusConflict, w.Code, w.Body.String())

	// Validation: old_path must be absolute, new_path must be a path or URL,
	// and a self-redirect is rejected.
	w = doRequest(createRouter, http.MethodPost, "/admin/redirects", `{"old_path":"old/","new_path":"/new/"}`)
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
	w = doRequest(createRouter, http.MethodPost, "/admin/redirects", `{"old_path":"/a/","new_path":"ftp://x"}`)
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
	w = doRequest(createRouter, http.MethodPost, "/admin/redirects", `{"old_path":"/loop/","new_path":"/loop/"}`)
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())

	// Absolute URL targets are allowed.
	w = doRequest(createRouter, http.MethodPost, "/admin/redirects", `{"old_path":"/ext/","new_path":"https://example.com/x"}`)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())

	// Update + delete
	updateRouter := cmsRouter(1, http.MethodPatch, "/admin/redirect/:id", AdminRedirectUpdate)
	w = doRequest(updateRouter, http.MethodPatch, "/admin/redirect/1", `{"new_path":"/newer/"}`)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NoError(t, db.DB.First(&r, r.ID).Error)
	require.Equal(t, "/newer/", r.NewPath)

	deleteRouter := cmsRouter(1, http.MethodDelete, "/admin/redirect/:id", AdminRedirectDelete)
	w = doRequest(deleteRouter, http.MethodDelete, "/admin/redirect/1", "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var count int64
	require.NoError(t, db.DB.Model(&models.Redirect{}).Count(&count).Error)
	require.Equal(t, int64(1), count)
}

func TestFlatPageLookup(t *testing.T) {
	setupCMSDB(t)
	seedFlatPage(t, "/about/", "About", "# Hi", models.CMSSiteID, false)
	seedFlatPage(t, "/secret/", "Secret", "hidden", models.CMSSiteID, true)
	// A page on another site must not be served.
	seedFlatPage(t, "/elsewhere/", "Elsewhere", "", 2, false)

	router := cmsRouter(1, http.MethodGet, "/page", FlatPageLookup)

	w := doRequest(router, http.MethodGet, "/page?url=/about/", "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var got map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
	require.Equal(t, "About", got["title"])
	require.Equal(t, "# Hi", got["content"])

	// Trailing-slash normalization, both directions.
	w = doRequest(router, http.MethodGet, "/page?url=/about", "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	w = doRequest(router, http.MethodGet, "/page?url=about/", "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	w = doRequest(router, http.MethodGet, "/page?url=/missing/", "")
	require.Equal(t, http.StatusNotFound, w.Code, w.Body.String())
	w = doRequest(router, http.MethodGet, "/page?url=/elsewhere/", "")
	require.Equal(t, http.StatusNotFound, w.Code, w.Body.String())

	// registration_required is a 404 for anonymous callers ...
	anon := cmsRouter(0, http.MethodGet, "/page", FlatPageLookup)
	require.Equal(t, http.StatusNotFound, doRequest(anon, http.MethodGet, "/page?url=/secret/", "").Code)
	// ... and readable for a signed-in one.
	require.Equal(t, http.StatusOK, doRequest(router, http.MethodGet, "/page?url=/secret/", "").Code)
}

func TestRedirectMapShape(t *testing.T) {
	setupCMSDB(t)
	require.NoError(t, db.DB.Create(&models.Redirect{SiteID: models.CMSSiteID, OldPath: "/old/", NewPath: "/new/"}).Error)
	require.NoError(t, db.DB.Create(&models.Redirect{SiteID: models.CMSSiteID, OldPath: "/a/", NewPath: "https://example.com"}).Error)
	// Other sites are not part of the map.
	require.NoError(t, db.DB.Create(&models.Redirect{SiteID: 2, OldPath: "/other/", NewPath: "/x/"}).Error)

	router := cmsRouter(0, http.MethodGet, "/redirects-map", RedirectMap)
	w := doRequest(router, http.MethodGet, "/redirects-map", "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var resp struct {
		Data []struct {
			OldPath string `json:"old_path"`
			NewPath string `json:"new_path"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Len(t, resp.Data, 2)
	require.Equal(t, "/a/", resp.Data[0].OldPath)
	require.Equal(t, "https://example.com", resp.Data[0].NewPath)
	require.Equal(t, "/old/", resp.Data[1].OldPath)

	// The map is cacheable for the proxy.
	require.Contains(t, w.Header().Get("Cache-Control"), "max-age=60")
}