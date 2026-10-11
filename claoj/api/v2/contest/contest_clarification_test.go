package contest

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strconv"
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

type clarFixture struct {
	root, plain  models.AuthUser
	contest      models.Contest
	otherContest models.Contest
	clar         models.ContestClarification
}

func setupClarificationDB(t *testing.T) clarFixture {
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
		&models.Contest{}, &models.ContestClarification{},
	))
	db.DB = database

	var f clarFixture
	f.root = models.AuthUser{Username: "root", IsActive: true, IsStaff: true, IsSuperuser: true}
	require.NoError(t, db.DB.Create(&f.root).Error)
	f.plain = models.AuthUser{Username: "bob", IsActive: true}
	require.NoError(t, db.DB.Create(&f.plain).Error)
	for _, u := range []models.AuthUser{f.root, f.plain} {
		require.NoError(t, db.DB.Omit(clause.Associations).Create(&models.Profile{UserID: u.ID, Timezone: "UTC"}).Error)
	}

	now := time.Now()
	f.contest = models.Contest{Key: "c1", Name: "C1", StartTime: now.Add(-time.Hour), EndTime: now.Add(time.Hour)}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&f.contest).Error)
	f.otherContest = models.Contest{Key: "c2", Name: "C2", StartTime: now.Add(-time.Hour), EndTime: now.Add(time.Hour)}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&f.otherContest).Error)

	f.clar = models.ContestClarification{ContestID: f.contest.ID, Question: "Is n positive?", CreateTime: now, AuthorID: 2}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&f.clar).Error)
	return f
}

func clarRequest(userID uint, method, route, target, body string, handler gin.HandlerFunc) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Handle(method, route, func(c *gin.Context) {
		c.Set("user_id", userID)
		handler(c)
	})
	var req *http.Request
	if body != "" {
		req = httptest.NewRequest(method, target, bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, target, nil)
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

// Regression: the handler used to discard the parsed id and always look up
// clarification 0, so every answer returned 404.
func TestContestClarificationAnswer_FindsTheClarification(t *testing.T) {
	f := setupClarificationDB(t)
	id := strconv.Itoa(int(f.clar.ID))

	w := clarRequest(f.root.ID, http.MethodPost, "/contest/:key/clarification/:id/answer",
		"/contest/c1/clarification/"+id+"/answer", `{"answer":"Yes","is_public":true}`, ContestClarificationAnswer)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var got models.ContestClarification
	require.NoError(t, db.DB.First(&got, f.clar.ID).Error)
	require.True(t, got.IsAnswered)
	require.NotNil(t, got.Answer)
	require.Equal(t, "Yes", *got.Answer)
}

func TestContestClarificationDelete(t *testing.T) {
	const route = "/contest/:key/clarification/:id"

	t.Run("editor deletes", func(t *testing.T) {
		f := setupClarificationDB(t)
		id := strconv.Itoa(int(f.clar.ID))
		w := clarRequest(f.root.ID, http.MethodDelete, route, "/contest/c1/clarification/"+id, "", ContestClarificationDelete)
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())

		var n int64
		require.NoError(t, db.DB.Model(&models.ContestClarification{}).Count(&n).Error)
		require.Equal(t, int64(0), n)
	})

	t.Run("non-editor is forbidden", func(t *testing.T) {
		f := setupClarificationDB(t)
		id := strconv.Itoa(int(f.clar.ID))
		w := clarRequest(f.plain.ID, http.MethodDelete, route, "/contest/c1/clarification/"+id, "", ContestClarificationDelete)
		require.Equal(t, http.StatusForbidden, w.Code, w.Body.String())

		var n int64
		require.NoError(t, db.DB.Model(&models.ContestClarification{}).Count(&n).Error)
		require.Equal(t, int64(1), n)
	})

	t.Run("clarification of another contest is not found", func(t *testing.T) {
		f := setupClarificationDB(t)
		id := strconv.Itoa(int(f.clar.ID))
		w := clarRequest(f.root.ID, http.MethodDelete, route, "/contest/c2/clarification/"+id, "", ContestClarificationDelete)
		require.Equal(t, http.StatusNotFound, w.Code, w.Body.String())

		var n int64
		require.NoError(t, db.DB.Model(&models.ContestClarification{}).Count(&n).Error)
		require.Equal(t, int64(1), n)
	})

	t.Run("bad ids", func(t *testing.T) {
		f := setupClarificationDB(t)
		require.Equal(t, http.StatusBadRequest,
			clarRequest(f.root.ID, http.MethodDelete, route, "/contest/c1/clarification/abc", "", ContestClarificationDelete).Code)
		require.Equal(t, http.StatusNotFound,
			clarRequest(f.root.ID, http.MethodDelete, route, "/contest/c1/clarification/999", "", ContestClarificationDelete).Code)
		require.Equal(t, http.StatusNotFound,
			clarRequest(f.root.ID, http.MethodDelete, route, "/contest/nope/clarification/1", "", ContestClarificationDelete).Code)
	})
}
