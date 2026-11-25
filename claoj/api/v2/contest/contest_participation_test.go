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

func setupParticipationDB(t *testing.T) {
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
		&models.Contest{}, &models.ContestParticipation{},
	))
	db.DB = database
}

func createParticipationContest(t *testing.T, key string, visibility string, start, end time.Time) models.Contest {
	t.Helper()
	ct := models.Contest{Key: key, Name: key, StartTime: start, EndTime: end, IsVisible: true, ScoreboardVisibility: visibility}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&ct).Error)
	return ct
}

// createParticipant registers a user and their participation in the contest.
func createParticipant(t *testing.T, contestID uint, username, displayName string, virtual int, start time.Time) {
	t.Helper()
	u := models.AuthUser{Username: username, IsActive: true}
	require.NoError(t, db.DB.Create(&u).Error)
	p := models.Profile{UserID: u.ID, Timezone: "UTC", UsernameDisplayOverride: displayName}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&p).Error)
	part := models.ContestParticipation{
		ContestID: contestID, UserID: p.ID, RealStart: start,
		Score: float64(100 - virtual*10), Cumtime: uint(60 * (virtual + 1)), Virtual: virtual,
	}
	require.NoError(t, db.DB.Omit("Contest", "User").Create(&part).Error)
}

func participationRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Handle(http.MethodGet, "/contest/:key/participations", ParticipationList)
	// The pre-route URL kept as a compatibility fallback.
	router.Handle(http.MethodGet, "/participations", ParticipationList)
	return router
}

type participationBody struct {
	Results []struct {
		Contest     string  `json:"contest"`
		User        string  `json:"user"`
		DisplayName string  `json:"display_name"`
		Score       float64 `json:"score"`
		Cumtime     uint    `json:"cumtime"`
		IsVirtual   bool    `json:"is_virtual"`
	} `json:"results"`
	Total            int  `json:"total"`
	ScoreboardHidden bool `json:"scoreboard_hidden"`
}

func getParticipations(t *testing.T, router *gin.Engine, target string) participationBody {
	t.Helper()
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, target, nil))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var body participationBody
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	return body
}

// The public participation list must be scoped to the URL's contest: the
// handler used to read only ?contest=, so every request returned the rows of
// every visible contest.
func TestParticipationList_ScopedToURLContest(t *testing.T) {
	setupParticipationDB(t)
	now := time.Now()
	c1 := createParticipationContest(t, "c1", "V", now.Add(-time.Hour), now.Add(time.Hour))
	c2 := createParticipationContest(t, "c2", "V", now.Add(-time.Hour), now.Add(time.Hour))
	createParticipant(t, c1.ID, "alice", "Alice Liddell", 0, now)
	createParticipant(t, c1.ID, "bob", "", 1, now)
	createParticipant(t, c2.ID, "carol", "", 0, now)

	router := participationRouter()

	// Default: live participants of c1 only.
	live := getParticipations(t, router, "/contest/c1/participations")
	require.Equal(t, 1, live.Total)
	require.Len(t, live.Results, 1)
	require.Equal(t, "alice", live.Results[0].User)
	require.Equal(t, "Alice Liddell", live.Results[0].DisplayName)
	require.Equal(t, "c1", live.Results[0].Contest)

	// include_virtual adds c1's virtual row and still nothing from c2.
	all := getParticipations(t, router, "/contest/c1/participations?include_virtual=true")
	require.Equal(t, 2, all.Total)
	require.Len(t, all.Results, 2)
	require.Equal(t, "Alice Liddell", all.Results[0].DisplayName)
	require.Equal(t, "bob", all.Results[1].User)
	require.Equal(t, "bob", all.Results[1].DisplayName)
	require.True(t, all.Results[1].IsVirtual)
	require.Equal(t, uint(120), all.Results[1].Cumtime)

	// The other contest returns only its own rows and total.
	other := getParticipations(t, router, "/contest/c2/participations")
	require.Equal(t, 1, other.Total)
	require.Len(t, other.Results, 1)
	require.Equal(t, "carol", other.Results[0].User)
	require.Equal(t, "c2", other.Results[0].Contest)

	// Legacy ?contest= fallback when the route has no :key param.
	legacy := getParticipations(t, router, "/participations?contest=c2")
	require.Equal(t, 1, legacy.Total)
	require.Len(t, legacy.Results, 1)
	require.Equal(t, "carol", legacy.Results[0].User)
}

// v1 /contest/<c>/participations/<user>: one competitor's rows only.
func TestParticipationList_UserFilter(t *testing.T) {
	setupParticipationDB(t)
	now := time.Now()
	c := createParticipationContest(t, "c1", "V", now.Add(-time.Hour), now.Add(time.Hour))
	createParticipant(t, c.ID, "alice", "Alice Liddell", 0, now)
	createParticipant(t, c.ID, "bob", "", 0, now)

	router := participationRouter()

	alice := getParticipations(t, router, "/contest/c1/participations?user=alice")
	require.Equal(t, 1, alice.Total)
	require.Len(t, alice.Results, 1)
	require.Equal(t, "alice", alice.Results[0].User)

	// An unknown username matches nothing rather than falling back to everyone.
	missing := getParticipations(t, router, "/contest/c1/participations?user=nobody")
	require.Zero(t, missing.Total)
	require.Empty(t, missing.Results)

	// The filter still respects include_virtual.
	createParticipant(t, c.ID, "carol", "", 1, now)
	virtual := getParticipations(t, router, "/contest/c1/participations?user=carol&include_virtual=true")
	require.Equal(t, 1, virtual.Total)
	require.True(t, virtual.Results[0].IsVirtual)
}

// scoreboard_visibility hides the participant list with the standings.
func TestParticipationList_HiddenByScoreboardVisibility(t *testing.T) {
	setupParticipationDB(t)
	now := time.Now()
	running := createParticipationContest(t, "hidden", "C", now.Add(-time.Hour), now.Add(time.Hour))
	createParticipant(t, running.ID, "alice", "", 0, now)

	hidden := getParticipations(t, participationRouter(), "/contest/hidden/participations")
	require.True(t, hidden.ScoreboardHidden)
	require.Zero(t, hidden.Total)
	require.Empty(t, hidden.Results)

	// Ended contests with 'C' become visible.
	ended := createParticipationContest(t, "done", "C", now.Add(-2*time.Hour), now.Add(-time.Hour))
	createParticipant(t, ended.ID, "bob", "", 0, now.Add(-2*time.Hour))
	visible := getParticipations(t, participationRouter(), "/contest/done/participations")
	require.False(t, visible.ScoreboardHidden)
	require.Equal(t, 1, visible.Total)
}