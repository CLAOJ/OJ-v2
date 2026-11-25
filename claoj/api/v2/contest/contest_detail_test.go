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

func setupContestDetailDB(t *testing.T) models.Contest {
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
		&models.ProblemGroup{}, &models.ProblemType{}, &models.Problem{}, &models.ContestTag{},
		&models.Contest{}, &models.ContestProblem{}, &models.ContestAnnouncement{},
		&models.ContestParticipation{},
	))
	db.DB = database

	now := time.Now()
	contest := models.Contest{
		Key: "c1", Name: "C1",
		Description: "<p>Full statement</p>",
		Summary:     "Short summary",
		StartTime:   now.Add(-time.Hour), EndTime: now.Add(time.Hour),
		IsVisible: true,
	}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&contest).Error)
	return contest
}

// The public contest page renders contest.description (falling back to a
// placeholder), so the detail payload must carry it.
func TestContestDetail_IncludesDescriptionAndSummary(t *testing.T) {
	setupContestDetailDB(t)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Handle(http.MethodGet, "/contest/:key", ContestDetail)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/contest/c1", nil))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Equal(t, "<p>Full statement</p>", body["description"])
	require.Equal(t, "Short summary", body["summary"])
}

// v1 renders the contest's announcements to participants and after the contest
// ends, newest first; the public detail payload must carry them in that case.
func TestContestDetail_IncludesAnnouncementsNewestFirst(t *testing.T) {
	contest := setupContestDetailDB(t)
	// Ended contests are open to everyone, participant or not.
	require.NoError(t, db.DB.Model(&models.Contest{}).Where("id = ?", contest.ID).
		Updates(map[string]any{
			"start_time": time.Now().Add(-3 * time.Hour),
			"end_time":   time.Now().Add(-time.Hour),
		}).Error)

	base := time.Now().Add(-2 * time.Hour)
	for i, title := range []string{"Older", "Newer"} {
		a := models.ContestAnnouncement{
			ContestID:   contest.ID,
			Title:       title,
			Description: "<p>" + title + "</p>",
			Date:        base.Add(time.Duration(i) * time.Hour),
		}
		require.NoError(t, db.DB.Omit("Contest").Create(&a).Error)
	}

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Handle(http.MethodGet, "/contest/:key", ContestDetail)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/contest/c1", nil))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var body struct {
		Announcements []struct {
			Title       string `json:"title"`
			Description string `json:"description"`
			Date        string `json:"date"`
		} `json:"announcements"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Len(t, body.Announcements, 2)
	require.Equal(t, "Newer", body.Announcements[0].Title)
	require.Equal(t, "Older", body.Announcements[1].Title)
	require.NotEmpty(t, body.Announcements[0].Description)
}

// During a running contest the announcements are for the people taking part
// (and the contest's staff), not for anonymous visitors.
func TestContestDetail_AnnouncementsGatedDuringContest(t *testing.T) {
	contest := setupContestDetailDB(t)
	a := models.ContestAnnouncement{
		ContestID:   contest.ID,
		Title:       "Hidden",
		Description: "<p>Hidden</p>",
		Date:        time.Now(),
	}
	require.NoError(t, db.DB.Omit("Contest").Create(&a).Error)

	user := models.AuthUser{Username: "alice", IsActive: true}
	require.NoError(t, db.DB.Create(&user).Error)
	profile := models.Profile{UserID: user.ID, Timezone: "UTC"}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&profile).Error)
	require.NoError(t, db.DB.Omit("Contest", "User").Create(&models.ContestParticipation{
		ContestID: contest.ID, UserID: profile.ID, RealStart: time.Now(),
	}).Error)

	announcementsFor := func(userID *uint) []map[string]any {
		gin.SetMode(gin.TestMode)
		router := gin.New()
		handler := ContestDetail
		if userID != nil {
			handler = func(c *gin.Context) {
				c.Set("user_id", *userID)
				ContestDetail(c)
			}
		}
		router.Handle(http.MethodGet, "/contest/:key", handler)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/contest/c1", nil))
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())
		var body struct {
			Announcements []map[string]any `json:"announcements"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
		return body.Announcements
	}

	require.Empty(t, announcementsFor(nil), "anonymous non-participant must not see them")
	require.Len(t, announcementsFor(&user.ID), 1, "a participant must see them")

	// Another signed-in user who neither joined nor staffs the contest: none.
	other := models.AuthUser{Username: "bob", IsActive: true}
	require.NoError(t, db.DB.Create(&other).Error)
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&models.Profile{UserID: other.ID, Timezone: "UTC"}).Error)
	require.Empty(t, announcementsFor(&other.ID))
}