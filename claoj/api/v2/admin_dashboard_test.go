package v2

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// setupDashboardDB builds the in-memory schema the admin dashboard's two
// read endpoints need (user list + submission list) and clears the cached
// service singletons so they bind to the fresh DB.
func setupDashboardDB(t *testing.T) {
	t.Helper()
	old := db.DB
	t.Cleanup(func() { db.DB = old })
	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	require.NoError(t, err)
	sqlDB, err := database.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, database.AutoMigrate(
		&models.AuthUser{}, &models.Profile{}, &models.Organization{}, &models.Language{},
		&models.ProblemGroup{}, &models.ProblemType{}, &models.Problem{}, &models.Judge{},
		&models.Submission{}, &models.AuthGroup{}, &models.DjangoContentType{},
		&models.AuthPermission{}, &models.AuthUserGroup{}, &models.AuthGroupPermission{},
		&models.AuthUserPermission{},
	))
	db.DB = database
	userService = nil
	submissionService = nil
}

// TestAdminUserList_ReturnsTotal pins the shape the dashboard's Users tile
// depends on: the tile reads `.data.total`, and TanStack Query v5 rejects a
// queryFn that resolves to undefined, which is what turns the tile into "—".
func TestAdminUserList_ReturnsTotal(t *testing.T) {
	setupDashboardDB(t)
	for _, name := range []string{"alice", "bob"} {
		u := models.AuthUser{Username: name, Email: name + "@example.com", IsActive: true}
		require.NoError(t, db.DB.Create(&u).Error)
		require.NoError(t, db.DB.Omit(clause.Associations).Create(&models.Profile{UserID: u.ID, Timezone: "UTC"}).Error)
	}

	router := authedRouter(1, http.MethodGet, "/admin/users", AdminUserList)
	w := doRequest(router, http.MethodGet, "/admin/users?page=1&page_size=1", "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var body struct {
		Data  []map[string]any `json:"data"`
		Total int64            `json:"total"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Len(t, body.Data, 1, "page_size=1 still returns a single row")
	require.Equal(t, int64(2), body.Total, "total is the whole user count, not the page length")
}

// TestAdminSubmissionList_DateIsRFC3339 pins the date contract: the
// dashboard renders it with dayjs fromNow(), which only produces a sane
// relative time when the API sends an RFC3339 timestamp.
func TestAdminSubmissionList_DateIsRFC3339(t *testing.T) {
	setupDashboardDB(t)

	u := models.AuthUser{Username: "kn7512", IsActive: true}
	require.NoError(t, db.DB.Create(&u).Error)
	profile := models.Profile{UserID: u.ID, Timezone: "UTC"}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&profile).Error)
	p := models.Problem{Code: "aplusb", Name: "A+B"}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&p).Error)
	lang := models.Language{Key: "PY3", Name: "Python 3"}
	require.NoError(t, db.DB.Create(&lang).Error)

	want := time.Date(2026, 10, 11, 12, 34, 56, 0, time.UTC)
	sub := models.Submission{UserID: profile.ID, ProblemID: p.ID, LanguageID: lang.ID, Date: want, Status: "D"}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&sub).Error)

	router := authedRouter(u.ID, http.MethodGet, "/admin/submissions", AdminSubmissionList)
	w := doRequest(router, http.MethodGet, "/admin/submissions?page=1&page_size=10", "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var body struct {
		Data []struct {
			ID   uint   `json:"id"`
			User string `json:"user"`
			Date string `json:"date"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Len(t, body.Data, 1, w.Body.String())

	parsed, err := time.Parse(time.RFC3339, body.Data[0].Date)
	require.NoError(t, err, "date must be RFC3339, got %q", body.Data[0].Date)
	require.WithinDuration(t, want, parsed.UTC(), time.Second)
}
