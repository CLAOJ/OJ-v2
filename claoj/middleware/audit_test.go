package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/CLAOJ/claoj/auditlog"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// waitForEntries polls store.List until it returns at least one entry or
// the deadline passes, since Audit() writes asynchronously in a goroutine.
func waitForEntries(t *testing.T, store auditlog.Store) []auditlog.Entry {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		entries, _, err := store.List(auditlog.Filters{}, 1, 20)
		require.NoError(t, err)
		if len(entries) > 0 {
			return entries
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("timed out waiting for audit entry to be written")
	return nil
}

// TestAudit_Middleware_WritesEntry proves that a request routed through
// Audit() actually reaches the configured store — regression coverage for
// the removed path guard, which previously matched "/api/v2/admin" and
// therefore never matched the real mounted routes (e.g. "/api/admin/..."),
// silently disabling auditing entirely.
func TestAudit_Middleware_WritesEntry(t *testing.T) {
	gin.SetMode(gin.TestMode)

	store := auditlog.NewMemoryStore()
	prev := auditlog.Default
	auditlog.Default = store
	t.Cleanup(func() { auditlog.Default = prev })

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user_id", uint(7))
		c.Set("username", "adminuser")
		c.Next()
	})
	router.Use(Audit())
	router.POST("/admin/problem/:id", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodPost, "/admin/problem/42", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	entries := waitForEntries(t, store)
	require.Len(t, entries, 1)
	require.Equal(t, "CREATE", entries[0].Action)
	require.Equal(t, "success", entries[0].Status)
	require.Equal(t, uint(7), entries[0].UserID)
	require.Equal(t, "adminuser", entries[0].Username)
}

// TestAudit_Middleware_LogsGetRequests documents that GET (VIEW) requests
// are audited too, matching the pre-existing behavior of logging every
// method — the removed guard did not gate on method, only (incorrectly)
// on path.
func TestAudit_Middleware_LogsGetRequests(t *testing.T) {
	gin.SetMode(gin.TestMode)

	store := auditlog.NewMemoryStore()
	prev := auditlog.Default
	auditlog.Default = store
	t.Cleanup(func() { auditlog.Default = prev })

	router := gin.New()
	router.Use(Audit())
	router.GET("/admin/users", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/admin/users", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	entries := waitForEntries(t, store)
	require.Len(t, entries, 1)
	require.Equal(t, "VIEW", entries[0].Action)
	require.Equal(t, "anonymous", entries[0].Username)
}

// TestExtractResourceInfo proves that extractResourceInfo parses the
// resource/resourceID from the real mounted admin path prefix
// ("/api/admin/...", see api/router.go: apiv2 := r.Group("/api"), admin
// routes registered as "/admin/..."). Previously the function trimmed
// "/api/v2/admin", which never matched any real request path, so every
// admin path fell through to resource="Api", resourceID="admin" —
// defeating resource-based filtering of the audit log.
func TestExtractResourceInfo(t *testing.T) {
	tests := []struct {
		name           string
		path           string
		wantResource   string
		wantResourceID string
	}{
		{
			name:           "users list",
			path:           "/api/admin/users",
			wantResource:   "Users",
			wantResourceID: "",
		},
		{
			name:           "single user by id",
			path:           "/api/admin/user/42",
			wantResource:   "User",
			wantResourceID: "42",
		},
		{
			name:           "single group by id",
			path:           "/api/admin/group/7",
			wantResource:   "Group",
			wantResourceID: "7",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resource, resourceID := extractResourceInfo(tt.path)
			require.Equal(t, tt.wantResource, resource)
			require.Equal(t, tt.wantResourceID, resourceID)
			require.NotEqual(t, "Api", resource)
			require.NotEqual(t, "admin", resourceID)
		})
	}
}

// TestAudit_Middleware_ExtractsResourceFromRealAdminPath drives the full
// Audit() middleware over real admin request paths (as they arrive after
// router mounting, i.e. without the "/api" prefix stripped by the http
// server's mux — see router.go where admin handlers are registered on
// paths like "/admin/user/:id") and confirms the persisted entry's
// Resource/ResourceID reflect the actual target rather than the stale
// "/api/v2/admin"-prefixed parsing.
func TestAudit_Middleware_ExtractsResourceFromRealAdminPath(t *testing.T) {
	gin.SetMode(gin.TestMode)

	store := auditlog.NewMemoryStore()
	prev := auditlog.Default
	auditlog.Default = store
	t.Cleanup(func() { auditlog.Default = prev })

	router := gin.New()
	router.Use(Audit())
	router.GET("/api/admin/user/:id", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/api/admin/user/42", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	entries := waitForEntries(t, store)
	require.Len(t, entries, 1)
	require.Equal(t, "User", entries[0].Resource)
	require.Equal(t, "42", entries[0].ResourceID)
}
