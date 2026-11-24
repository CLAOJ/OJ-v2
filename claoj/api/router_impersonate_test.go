package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/CLAOJ/claoj/auth"
	"github.com/CLAOJ/claoj/config"
	"github.com/CLAOJ/claoj/csrf"
	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Both impersonation routes must be mounted with the expected shapes: the
// start route under the admin gate, the exit route reachable from an
// (usually non-staff) impersonated session. Unauthenticated requests should
// be rejected by auth, never 404.
func TestImpersonateRoutesAreMounted(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldCfg := config.C
	t.Cleanup(func() { config.C = oldCfg })
	config.C.App.JwtSecretKey = "router-test-secret"
	config.C.App.SiteFullURL = "http://localhost:3000"

	var router *gin.Engine
	require.NotPanics(t, func() { router = NewRouter() })

	for _, path := range []string{"/api/admin/impersonate/1", "/api/admin/impersonate/exit"} {
		req := httptest.NewRequest(http.MethodPost, path, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusUnauthorized, w.Code, "%s should require auth, not be missing", path)
	}
}

// Starting an impersonation replaces the session, so it must sit behind the
// CSRF double-submit check like every other mutating protected route.
func TestImpersonateStartRequiresCSRF(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldDB, oldCfg := db.DB, config.C
	t.Cleanup(func() { db.DB, config.C = oldDB, oldCfg })

	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	require.NoError(t, err)
	sqlDB, err := database.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, database.AutoMigrate(
		&models.AuthUser{}, &models.Profile{}, &models.AuthGroup{}, &models.DjangoContentType{},
		&models.AuthPermission{}, &models.AuthUserGroup{}, &models.AuthGroupPermission{},
		&models.AuthUserPermission{},
	))
	db.DB = database
	config.C.App.JwtSecretKey = "router-test-secret"
	config.C.App.SiteFullURL = "http://localhost:3000"

	root := models.AuthUser{Username: "root", IsActive: true, IsStaff: true, IsSuperuser: true}
	require.NoError(t, db.DB.Create(&root).Error)
	token, _, _, err := auth.GenerateTokens(root.ID, root.Username, true, "", false)
	require.NoError(t, err)

	var router *gin.Engine
	require.NotPanics(t, func() { router = NewRouter() })

	// No CSRF cookie/header: the protected group must stop it before the
	// handler runs.
	req := httptest.NewRequest(http.MethodPost, "/api/admin/impersonate/1", nil)
	req.AddCookie(&http.Cookie{Name: "access_token", Value: token})
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusForbidden, w.Code, w.Body.String())
	require.Contains(t, w.Body.String(), "csrf")

	// With the matching pair the request reaches the handler (profile 1 does
	// not exist -> 404), proving CSRF was what blocked the first request.
	req = httptest.NewRequest(http.MethodPost, "/api/admin/impersonate/1", nil)
	req.AddCookie(&http.Cookie{Name: "access_token", Value: token})
	req.AddCookie(&http.Cookie{Name: csrf.CSRFTokenCookie, Value: "tok"})
	req.Header.Set(csrf.CSRFTokenHeader, "tok")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusNotFound, w.Code, w.Body.String())
}