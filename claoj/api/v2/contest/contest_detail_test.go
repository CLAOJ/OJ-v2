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
		&models.Contest{}, &models.ContestProblem{},
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