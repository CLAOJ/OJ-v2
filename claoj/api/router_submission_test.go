package api

import (
	"net/http"
	"testing"

	"github.com/CLAOJ/claoj/config"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// The file-submit flow depends on two route paths: the web app POSTs the
// multipart body to the submit route, and the judge agent downloads the stored
// file from the serve route. A dropped or renamed registration would only show
// up as a 404 at runtime.
func TestFileSubmissionRoutesAreMounted(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldCfg := config.C
	t.Cleanup(func() { config.C = oldCfg })
	config.C.App.JwtSecretKey = "router-test-secret"
	config.C.App.SiteFullURL = "http://localhost:3000"

	var router *gin.Engine
	require.NotPanics(t, func() { router = NewRouter() })

	mounted := map[string]bool{}
	for _, r := range router.Routes() {
		mounted[r.Method+" "+r.Path] = true
	}
	for _, want := range []string{
		http.MethodGet + " /api/submission-file/:name",
		http.MethodPost + " /api/problem/:code/submit",
	} {
		require.True(t, mounted[want], "%s must be registered", want)
	}
}
