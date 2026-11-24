package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/CLAOJ/claoj/config"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
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