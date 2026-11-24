package v2

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

type rankingFixture struct {
	contest    models.Contest
	liveUser   models.AuthUser
	virtualUsr models.AuthUser
}

func setupRankingDB(t *testing.T) rankingFixture {
	t.Helper()
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
		&models.ProblemGroup{}, &models.ProblemType{}, &models.Problem{},
		&models.Contest{}, &models.ContestProblem{}, &models.ContestParticipation{},
	))
	db.DB = database

	now := time.Now()
	contest := models.Contest{
		Key: "c1", Name: "C1", Description: "d",
		StartTime: now.Add(-time.Hour), EndTime: now.Add(time.Hour),
		IsVisible: true,
	}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&contest).Error)

	org := models.Organization{Name: "Alpha Org", Slug: "alpha", ShortName: "ALPHA", About: ""}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&org).Error)
	// is_unlisted defaults to 1; the scoreboard only shows listed organizations
	// (v1 ranking prefetches Organization.objects.filter(is_unlisted=False)).
	require.NoError(t, db.DB.Model(&org).Update("is_unlisted", false).Error)

	createUser := func(username, displayName string, withOrg bool) models.AuthUser {
		u := models.AuthUser{Username: username, IsActive: true}
		require.NoError(t, db.DB.Create(&u).Error)
		p := models.Profile{UserID: u.ID, Timezone: "UTC", UsernameDisplayOverride: displayName}
		require.NoError(t, db.DB.Omit(clause.Associations).Create(&p).Error)
		if withOrg {
			require.NoError(t, db.DB.Model(&p).Association("Organizations").Append(&org))
		}
		return u
	}
	live := createUser("alice", "Alice Liddell", true)
	virt := createUser("bob", "", true)

	for i, u := range []models.AuthUser{live, virt} {
		var profile models.Profile
		require.NoError(t, db.DB.Where("user_id = ?", u.ID).First(&profile).Error)
		part := models.ContestParticipation{
			ContestID: contest.ID, UserID: profile.ID,
			RealStart: now, Score: float64(100 - i*10), Cumtime: uint(60 * (i + 1)),
			Virtual: i, // 0 = live, 1 = virtual
		}
		require.NoError(t, db.DB.Omit("Contest", "User").Create(&part).Error)
	}

	return rankingFixture{contest: contest, liveUser: live, virtualUsr: virt}
}

func getRankings(t *testing.T, target string) []map[string]any {
	t.Helper()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Handle(http.MethodGet, "/contest/:key/ranking", ContestRanking)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, target, nil))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var body struct {
		Rankings []map[string]any `json:"rankings"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	return body.Rankings
}

func TestContestRanking_HidesVirtualByDefault(t *testing.T) {
	setupRankingDB(t)

	rows := getRankings(t, "/contest/c1/ranking")
	require.Len(t, rows, 1)
	require.Equal(t, "alice", rows[0]["username"])
	// The scoreboard shows the display name and the primary organization.
	require.Equal(t, "Alice Liddell", rows[0]["display_name"])
	org, ok := rows[0]["organization"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "ALPHA", org["short_name"])
	require.Equal(t, false, rows[0]["is_virtual"])
}

func TestContestRanking_IncludeVirtualToggle(t *testing.T) {
	setupRankingDB(t)

	rows := getRankings(t, "/contest/c1/ranking?include_virtual=true")
	require.Len(t, rows, 2)
	require.Equal(t, "alice", rows[0]["username"])
	require.Equal(t, "bob", rows[1]["username"])
	require.Equal(t, true, rows[1]["is_virtual"])
	// No display-name override: v1 falls back to the username.
	require.Equal(t, "bob", rows[1]["display_name"])
}