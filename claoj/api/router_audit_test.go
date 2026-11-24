package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/CLAOJ/claoj/auditlog"
	"github.com/CLAOJ/claoj/auth"
	"github.com/CLAOJ/claoj/config"
	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// TestAdminAuditLog_RecordsMutation drives a real admin mutation through the
// full router (auth gate -> Audit() -> handler) and then reads it back from
// the endpoint the admin audit-log page and the dashboard both call. It is the
// end-to-end guard against auditing being silently disabled (a stale path
// guard or an unwired store leaves the trail empty while traffic is live).
func TestAdminAuditLog_RecordsMutation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldDB, oldCfg, oldAudit := db.DB, config.C, auditlog.Default
	t.Cleanup(func() { db.DB, config.C, auditlog.Default = oldDB, oldCfg, oldAudit })

	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	require.NoError(t, err)
	sqlDB, err := database.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, database.AutoMigrate(&models.AuthUser{}, &models.Profile{}, &models.ContestTag{}))
	db.DB = database
	config.C.App.JwtSecretKey = "router-audit-secret"
	config.C.App.SiteFullURL = "http://localhost:3000"

	store := auditlog.NewMemoryStore()
	auditlog.Default = store

	staff := models.AuthUser{Username: "staff", IsActive: true, IsStaff: true}
	require.NoError(t, db.DB.Create(&staff).Error)
	token, _, _, err := auth.GenerateTokens(staff.ID, staff.Username, false, "", false)
	require.NoError(t, err)

	router := NewRouter()

	// Admin mutation: creating a contest tag is gated only by is_staff, so no
	// Django permission fixture is needed.
	req := httptest.NewRequest(http.MethodPost, "/api/admin/contest-tags", strings.NewReader(`{"name":"audit_tag","color":"#123456"}`))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "access_token", Value: token})
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())

	// Audit() writes asynchronously, so poll the list endpoint (the same one
	// the admin page and the dashboard use). Reads are audited too (VIEW), so
	// skip those when looking for the change.
	var change *auditlog.Entry
	var total int64
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) && change == nil {
		listReq := httptest.NewRequest(http.MethodGet, "/api/audit-logs?page=1&page_size=50", nil)
		listReq.AddCookie(&http.Cookie{Name: "access_token", Value: token})
		listW := httptest.NewRecorder()
		router.ServeHTTP(listW, listReq)
		require.Equal(t, http.StatusOK, listW.Code, listW.Body.String())

		var resp struct {
			Results []auditlog.Entry `json:"results"`
			Total   int64            `json:"total"`
			Page    int              `json:"page"`
		}
		require.NoError(t, json.Unmarshal(listW.Body.Bytes(), &resp))
		require.Equal(t, 1, resp.Page)
		total = resp.Total
		for i := range resp.Results {
			if resp.Results[i].Action == "CREATE" {
				change = &resp.Results[i]
				break
			}
		}
		if change == nil {
			time.Sleep(10 * time.Millisecond)
		}
	}

	require.NotNil(t, change, "admin mutation was not recorded (total entries seen: %d)", total)
	require.Equal(t, "CREATE", change.Action)
	require.Equal(t, "Contest-Tags", change.Resource)
	require.Equal(t, "success", change.Status, "a 2xx mutation must be recorded as success")
	require.Equal(t, staff.ID, change.UserID)
	require.Equal(t, "staff", change.Username)
	require.NotEmpty(t, change.ID)
	require.Contains(t, change.Details, `"status_code":201`)
	require.Contains(t, change.Details, `"path":"/api/admin/contest-tags"`)
}