package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/CLAOJ/claoj/auditlog"
	"github.com/CLAOJ/claoj/auth"
	"github.com/CLAOJ/claoj/config"
	"github.com/CLAOJ/claoj/csrf"
	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// adminCSRF adds the double-submit pair the admin group requires on writes.
func adminCSRF(req *http.Request) *http.Request {
	req.AddCookie(&http.Cookie{Name: csrf.CSRFTokenCookie, Value: "tok"})
	req.Header.Set(csrf.CSRFTokenHeader, "tok")
	return req
}

// A signed-in staff user with no Django permissions must be stopped by the
// route-level auth.RequirePerm gates, not only by checks inside handlers.
func TestAdminRoutes_PermissionGates(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldDB, oldCfg := db.DB, config.C
	t.Cleanup(func() { db.DB, config.C = oldDB, oldCfg })

	oldAudit := auditlog.Default
	auditlog.Default = auditlog.NewMemoryStore()
	t.Cleanup(func() { auditlog.Default = oldAudit })

	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	require.NoError(t, err)
	sqlDB, err := database.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, database.AutoMigrate(
		&models.AuthUser{}, &models.Profile{}, &models.Organization{}, &models.Language{},
		&models.ProblemGroup{}, &models.ProblemType{}, &models.Problem{},
		&models.Judge{}, &models.Submission{}, &models.Ticket{},
		&models.AuthGroup{}, &models.DjangoContentType{}, &models.AuthPermission{},
		&models.AuthUserGroup{}, &models.AuthGroupPermission{}, &models.AuthUserPermission{},
	))
	db.DB = database
	config.C.App.JwtSecretKey = "router-test-secret"
	config.C.App.SiteFullURL = "http://localhost:3000"

	staff := models.AuthUser{Username: "staff", IsActive: true, IsStaff: true}
	require.NoError(t, db.DB.Create(&staff).Error)
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&models.Judge{Name: "j", AuthKey: "k", Created: time.Now()}).Error)
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&models.Submission{UserID: 1, ProblemID: 1, LanguageID: 1, Date: time.Now(), Status: "D"}).Error)
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&models.Ticket{Title: "t"}).Error)

	token, _, _, err := auth.GenerateTokens(staff.ID, staff.Username, false, "", false)
	require.NoError(t, err)

	var router *gin.Engine
	require.NotPanics(t, func() { router = NewRouter() })

	routes := []struct{ method, path, body string }{
		{http.MethodPost, "/api/admin/judges", `{"name":"x"}`},
		{http.MethodDelete, "/api/admin/judge/1", ""},
		{http.MethodPost, "/api/admin/users", `{"username":"x","email":"x@example.com","password":"Passw0rdOK"}`},
		{http.MethodPost, "/api/admin/user/1/password", `{"password":"Passw0rdOK"}`},
		{http.MethodDelete, "/api/admin/submission/1", ""},
		{http.MethodDelete, "/api/admin/ticket/1", ""},
		{http.MethodPatch, "/api/admin/ticket/1", `{"title":"x"}`},
		{http.MethodPatch, "/api/admin/ticket/1/message/1", `{"body":"x"}`},
		{http.MethodDelete, "/api/admin/ticket/1/message/1", ""},
		{http.MethodPost, "/api/admin/registration/1/activate", ""},
		{http.MethodPost, "/api/admin/registration/1/resend", ""},
		{http.MethodPost, "/api/admin/flat-pages", `{"url":"/a/","title":"A"}`},
		{http.MethodPatch, "/api/admin/flat-page/1", `{"title":"x"}`},
		{http.MethodDelete, "/api/admin/flat-page/1", ""},
		{http.MethodPost, "/api/admin/redirects", `{"old_path":"/a/","new_path":"/b/"}`},
		{http.MethodPatch, "/api/admin/redirect/1", `{"new_path":"/b/"}`},
		{http.MethodDelete, "/api/admin/redirect/1", ""},
		{http.MethodPost, "/api/admin/navigation-bars", `{"key":"k","label":"L","path":"/p"}`},
		{http.MethodPatch, "/api/admin/navigation-bar/1", `{"label":"x"}`},
		{http.MethodDelete, "/api/admin/navigation-bar/1", ""},
	}
	for _, rt := range routes {
		t.Run(rt.method+" "+rt.path, func(t *testing.T) {
			req := httptest.NewRequest(rt.method, rt.path, strings.NewReader(rt.body))
			req.Header.Set("Content-Type", "application/json")
			req.AddCookie(&http.Cookie{Name: "access_token", Value: token})
			adminCSRF(req)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			require.Equal(t, http.StatusForbidden, w.Code, w.Body.String())
			require.NotContains(t, w.Body.String(), "csrf", "must be stopped by the perm gate, not CSRF")
		})
	}

	for _, m := range []any{&models.Judge{}, &models.Submission{}, &models.Ticket{}} {
		var n int64
		require.NoError(t, db.DB.Model(m).Count(&n).Error)
		require.Equal(t, int64(1), n, "a forbidden delete must leave the row")
	}

	// SameSite=None session cookies make admin writes forgeable cross-site:
	// a signed-in staff session without the double-submit pair is refused
	// before any handler runs. contest-tags is is_staff-only, so CSRF is the
	// only gate that can stop this request.
	noCSRF := httptest.NewRequest(http.MethodPost, "/api/admin/contest-tags", strings.NewReader(`{"name":"x"}`))
	noCSRF.Header.Set("Content-Type", "application/json")
	noCSRF.AddCookie(&http.Cookie{Name: "access_token", Value: token})
	w := httptest.NewRecorder()
	router.ServeHTTP(w, noCSRF)
	require.Equal(t, http.StatusForbidden, w.Code, w.Body.String())
	require.Contains(t, w.Body.String(), "csrf_token_missing")

	// The markdown image upload is admin-only: an anonymous caller is stopped by
	// AdminRequiredMiddleware before the handler reads the body.
	anon := httptest.NewRequest(http.MethodPost, "/api/admin/markdown/image", nil)
	anonW := httptest.NewRecorder()
	router.ServeHTTP(anonW, anon)
	require.Equal(t, http.StatusUnauthorized, anonW.Code, anonW.Body.String())

	// Anonymous callers are stopped by AdminRequiredMiddleware before the
	// handler runs; the same routes also cover the codename-gated CMS writes.
	for _, rt := range routes {
		anon := httptest.NewRequest(rt.method, rt.path, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, anon)
		require.Equal(t, http.StatusUnauthorized, w.Code, rt.method+" "+rt.path+" "+w.Body.String())
	}
}
