package contest

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// The public participation list must be usable by the contest page: display
// names, the virtual flag, and a total for pagination.
func TestParticipationList_IncludeVirtualAndTotal(t *testing.T) {
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
		&models.AuthUserPermission{}, &models.Profile{}, &models.Organization{}, &models.Language{},
		&models.Contest{}, &models.ContestParticipation{},
	))
	db.DB = database

	now := time.Now()
	contest := models.Contest{
		Key: "c1", Name: "C1", StartTime: now.Add(-time.Hour), EndTime: now.Add(time.Hour), IsVisible: true,
	}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&contest).Error)

	for i, username := range []string{"alice", "bob"} {
		u := models.AuthUser{Username: username, IsActive: true}
		require.NoError(t, db.DB.Create(&u).Error)
		display := ""
		if i == 0 {
			display = "Alice Liddell"
		}
		p := models.Profile{UserID: u.ID, Timezone: "UTC", UsernameDisplayOverride: display}
		require.NoError(t, db.DB.Omit(clause.Associations).Create(&p).Error)
		part := models.ContestParticipation{
			ContestID: contest.ID, UserID: p.ID, RealStart: now,
			Score: float64(100 - i*10), Cumtime: uint(60 * (i + 1)), Virtual: i,
		}
		require.NoError(t, db.DB.Omit("Contest", "User").Create(&part).Error)
	}

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Handle(http.MethodGet, "/contest/:key/participations", ParticipationList)

	type body struct {
		Results []struct {
			Contest     string  `json:"contest"`
			User        string  `json:"user"`
			DisplayName string  `json:"display_name"`
			Score       float64 `json:"score"`
			Cumtime     uint    `json:"cumtime"`
			IsVirtual   bool    `json:"is_virtual"`
		} `json:"results"`
		Total int `json:"total"`
	}

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/contest/c1/participations", nil))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var live body
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &live))
	require.Equal(t, 1, live.Total)
	require.Len(t, live.Results, 1)
	require.Equal(t, "alice", live.Results[0].User)
	require.Equal(t, "Alice Liddell", live.Results[0].DisplayName)
	require.Equal(t, "c1", live.Results[0].Contest)

	w = httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/contest/c1/participations?include_virtual=true", nil))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var all body
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &all))
	require.Equal(t, 2, all.Total)
	require.Len(t, all.Results, 2)
	require.Equal(t, "Alice Liddell", all.Results[0].DisplayName)
	require.Equal(t, "bob", all.Results[1].User)
	require.Equal(t, "bob", all.Results[1].DisplayName)
	require.True(t, all.Results[1].IsVirtual)
	require.Equal(t, uint(120), all.Results[1].Cumtime)
}