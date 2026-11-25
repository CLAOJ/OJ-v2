package v2

import (
	"bytes"
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

type rankingPayload struct {
	Rankings         []map[string]any `json:"rankings"`
	ScoreboardHidden bool             `json:"scoreboard_hidden"`
}

// getRankingPayload hits the ranking endpoint as an anonymous visitor or as
// userID (which the auth helpers resolve from the DB).
func getRankingPayload(t *testing.T, target string, userID *uint) rankingPayload {
	t.Helper()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := ContestRanking
	if userID != nil {
		handler = func(c *gin.Context) {
			c.Set("user_id", *userID)
			ContestRanking(c)
		}
	}
	router.Handle(http.MethodGet, "/contest/:key/ranking", handler)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, target, nil))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var body rankingPayload
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	return body
}

func setScoreboardVisibility(t *testing.T, contestID uint, visibility string) {
	t.Helper()
	require.NoError(t, db.DB.Model(&models.Contest{}).
		Where("id = ?", contestID).Update("scoreboard_visibility", visibility).Error)
}

// 'C' hides the standings until the contest ends and 'P' hides them until the
// viewer's own participation ends; anonymous visitors never see them.
func TestContestRanking_HiddenForDuration(t *testing.T) {
	f := setupRankingDB(t)
	setScoreboardVisibility(t, f.contest.ID, "C")

	hidden := getRankingPayload(t, "/contest/c1/ranking", nil)
	require.True(t, hidden.ScoreboardHidden)
	require.Empty(t, hidden.Rankings)

	setScoreboardVisibility(t, f.contest.ID, "P")
	hidden = getRankingPayload(t, "/contest/c1/ranking", nil)
	require.True(t, hidden.ScoreboardHidden)
	require.Empty(t, hidden.Rankings)

	setScoreboardVisibility(t, f.contest.ID, "V")
	visible := getRankingPayload(t, "/contest/c1/ranking", nil)
	require.False(t, visible.ScoreboardHidden)
	require.Len(t, visible.Rankings, 1)
}

// The scoreboard stays hidden before the start time even when 'V'.
func TestContestRanking_HiddenBeforeStart(t *testing.T) {
	f := setupRankingDB(t)
	now := time.Now()
	require.NoError(t, db.DB.Model(&models.Contest{}).Where("id = ?", f.contest.ID).
		Updates(map[string]any{"start_time": now.Add(time.Hour), "end_time": now.Add(2 * time.Hour)}).Error)

	hidden := getRankingPayload(t, "/contest/c1/ranking", nil)
	require.True(t, hidden.ScoreboardHidden)
	require.Empty(t, hidden.Rankings)
}

func TestContestRanking_CVisibleAfterContestEnds(t *testing.T) {
	f := setupRankingDB(t)
	setScoreboardVisibility(t, f.contest.ID, "C")
	now := time.Now()
	require.NoError(t, db.DB.Model(&models.Contest{}).Where("id = ?", f.contest.ID).
		Updates(map[string]any{"start_time": now.Add(-2 * time.Hour), "end_time": now.Add(-time.Hour)}).Error)

	visible := getRankingPayload(t, "/contest/c1/ranking", nil)
	require.False(t, visible.ScoreboardHidden)
	require.Len(t, visible.Rankings, 1)
}

// 'P' opens for a participant once their per-user window has elapsed, not
// before (v1 Contest.has_completed_contest).
func TestContestRanking_PVisibleAfterOwnWindowEnds(t *testing.T) {
	f := setupRankingDB(t)
	setScoreboardVisibility(t, f.contest.ID, "P")
	limit := int64(time.Hour.Microseconds())
	require.NoError(t, db.DB.Model(&models.Contest{}).Where("id = ?", f.contest.ID).
		Update("time_limit", limit).Error)

	// Alice's participation started just now with a one-hour window: still open.
	stillOpen := getRankingPayload(t, "/contest/c1/ranking", &f.liveUser.ID)
	require.True(t, stillOpen.ScoreboardHidden)

	// Shift the participation start back past its window; the contest itself is
	// still running, so only the completed participation can reveal the board.
	require.NoError(t, db.DB.Model(&models.ContestParticipation{}).
		Where("contest_id = ? AND virtual = 0", f.contest.ID).
		Update("start", time.Now().Add(-2*time.Hour)).Error)

	visible := getRankingPayload(t, "/contest/c1/ranking", &f.liveUser.ID)
	require.False(t, visible.ScoreboardHidden)
	require.Len(t, visible.Rankings, 1)
}

// Judges with see_private_contest bypass the duration gate (v1 permission
// short-circuit).
func TestContestRanking_PrivilegedSeesHiddenStandings(t *testing.T) {
	f := setupRankingDB(t)
	setScoreboardVisibility(t, f.contest.ID, "C")
	judge := createContestTestUser(t, "judge")
	grantContestPerm(t, judge.UserID, "see_private_contest")

	visible := getRankingPayload(t, "/contest/c1/ranking", &judge.UserID)
	require.False(t, visible.ScoreboardHidden)
	require.Len(t, visible.Rankings, 1)
}

func doPDFRequest(t *testing.T) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Handle(http.MethodGet, "/contest/:key/ranking/pdf", ContestRankingPDF)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/contest/c1/ranking/pdf", nil))
	return w
}

// The PDF must honour scoreboard_visibility: a hidden board returns the JSON
// endpoint's hidden payload instead of a PDF that leaks the standings.
func TestContestRankingPDF_HiddenForDuration(t *testing.T) {
	f := setupRankingDB(t)
	setScoreboardVisibility(t, f.contest.ID, "C")

	w := doPDFRequest(t)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NotContains(t, w.Header().Get("Content-Type"), "application/pdf")
	var body rankingPayload
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.True(t, body.ScoreboardHidden)
	require.Empty(t, body.Rankings)

	setScoreboardVisibility(t, f.contest.ID, "V")
	w = doPDFRequest(t)
	require.Equal(t, "application/pdf", w.Header().Get("Content-Type"))
	require.True(t, bytes.HasPrefix(w.Body.Bytes(), []byte("%PDF")))
}

// The PDF used to read p.User.User.Username off a has-one guess that never
// loaded, printing a blank column. Names now come from the same JOIN the
// ranking endpoint uses.
func TestContestRankingPDF_UsernamesResolved(t *testing.T) {
	f := setupRankingDB(t)

	var participations []models.ContestParticipation
	require.NoError(t, db.DB.Where("contest_id = ?", f.contest.ID).Order("id ASC").Find(&participations).Error)
	require.NotEmpty(t, participations)
	ids := make([]uint, len(participations))
	for i, p := range participations {
		ids[i] = p.UserID
	}
	names := contestRankingUsernames(ids)
	require.Equal(t, "alice", names[participations[0].UserID])
	require.Equal(t, "bob", names[participations[1].UserID])

	// With no participation ids the helper must not query or panic.
	require.Empty(t, contestRankingUsernames(nil))
}