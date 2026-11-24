package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/CLAOJ/claoj/auditlog"
	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// An impersonated session's admin requests must be attributed to the
// superuser who started it, with the assumed account kept in details.
func TestAudit_ImpersonatedRequestRecordsActor(t *testing.T) {
	gin.SetMode(gin.TestMode)

	oldDB, oldAudit := db.DB, auditlog.Default
	t.Cleanup(func() { db.DB, auditlog.Default = oldDB, oldAudit })
	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, database.AutoMigrate(&models.AuthUser{}))
	db.DB = database

	root := models.AuthUser{Username: "root", Email: "root@example.com", IsActive: true, IsSuperuser: true}
	require.NoError(t, database.Create(&root).Error)
	store := auditlog.NewMemoryStore()
	auditlog.Default = store

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user_id", uint(999)) // assumed account
		c.Set("username", "bob")
		c.Set("impersonator_id", root.ID)
	})
	router.Use(Audit())
	router.POST("/api/admin/user/1", func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodPost, "/api/admin/user/1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	// Audit writes happen in a goroutine.
	deadline := time.Now().Add(2 * time.Second)
	for {
		entries, _, err := store.List(auditlog.Filters{}, 1, 10)
		require.NoError(t, err)
		if len(entries) > 0 {
			require.Equal(t, root.ID, entries[0].UserID)
			require.Equal(t, "root", entries[0].Username)
			require.Contains(t, entries[0].Details, `"impersonated_user":"bob"`)
			return
		}
		if time.Now().After(deadline) {
			t.Fatal("audit entry was never written")
		}
		time.Sleep(10 * time.Millisecond)
	}
}
