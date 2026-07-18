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
