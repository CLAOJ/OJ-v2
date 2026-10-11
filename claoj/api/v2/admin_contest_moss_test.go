package v2

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/jobs"
	"github.com/CLAOJ/claoj/models"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func setupContestMossDB(t *testing.T) (root, staff models.AuthUser, contest models.Contest) {
	t.Helper()
	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, _ := database.DB()
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, database.AutoMigrate(
		&models.AuthUser{}, &models.AuthGroup{}, &models.DjangoContentType{}, &models.AuthPermission{},
		&models.AuthUserGroup{}, &models.AuthGroupPermission{}, &models.AuthUserPermission{},
		&models.Profile{}, &models.Problem{}, &models.Contest{}, &models.ContestMoss{}, &models.MiscConfig{},
	))
	// The list handler LEFT JOINs judge_contestproblem for ordering.
	require.NoError(t, database.Migrator().DropTable("judge_contestproblem"))
	require.NoError(t, database.Migrator().CreateTable(&models.ContestProblem{}))
	db.DB = database
	root = models.AuthUser{Username: "root", IsActive: true, IsStaff: true, IsSuperuser: true}
	staff = models.AuthUser{Username: "staff", IsActive: true, IsStaff: true}
	require.NoError(t, database.Create(&root).Error)
	require.NoError(t, database.Create(&staff).Error)
	contest = models.Contest{Key: "tik25", Name: "TiK25", StartTime: time.Now(), EndTime: time.Now().Add(time.Hour)}
	require.NoError(t, database.Omit(clause.Associations).Create(&contest).Error)
	return
}

func TestContestMoss_RequiresPermission(t *testing.T) {
	_, staff, _ := setupContestMossDB(t)
	router := authedRouter(staff.ID, http.MethodGet, "/admin/contest/:key/moss", AdminContestMossList)
	w := doRequest(router, http.MethodGet, "/admin/contest/tik25/moss", "")
	require.Equal(t, http.StatusForbidden, w.Code)
}

func TestContestMoss_ListReturnsRows(t *testing.T) {
	root, _, c := setupContestMossDB(t)
	p := models.Problem{Code: "knap", Name: "Knapsack"}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&p).Error)
	url := "http://moss/report"
	require.NoError(t, db.DB.Create(&models.ContestMoss{ContestID: c.ID, ProblemID: p.ID, Language: "C++", SubmissionCount: 4, URL: &url}).Error)

	router := authedRouter(root.ID, http.MethodGet, "/admin/contest/:key/moss", AdminContestMossList)
	w := doRequest(router, http.MethodGet, "/admin/contest/tik25/moss", "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var body struct {
		Running bool `json:"running"`
		Results []struct {
			ProblemCode     string  `json:"problem_code"`
			Language        string  `json:"language"`
			SubmissionCount uint    `json:"submission_count"`
			URL             *string `json:"url"`
		} `json:"results"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.False(t, body.Running)
	require.Len(t, body.Results, 1)
	require.Equal(t, "knap", body.Results[0].ProblemCode)
	require.Equal(t, url, *body.Results[0].URL)
}

func TestContestMoss_RunNeedsConfigAndEnqueues(t *testing.T) {
	root, _, c := setupContestMossDB(t)
	var enqueued []uint
	enqueueContestMoss = func(id uint) error { enqueued = append(enqueued, id); return nil }
	t.Cleanup(func() { enqueueContestMoss = jobs.EnqueueContestMoss })

	router := authedRouter(root.ID, http.MethodPost, "/admin/contest/:key/moss", AdminContestMossRun)
	w := doRequest(router, http.MethodPost, "/admin/contest/tik25/moss", "")
	require.Equal(t, http.StatusBadRequest, w.Code)
	require.Contains(t, w.Body.String(), "moss_user_id")

	require.NoError(t, db.DB.Create(&models.MiscConfig{Key: "moss_user_id", Value: "123"}).Error)
	w = doRequest(router, http.MethodPost, "/admin/contest/tik25/moss", "")
	require.Equal(t, http.StatusAccepted, w.Code, w.Body.String())
	require.Equal(t, []uint{c.ID}, enqueued)

	enqueueContestMoss = func(uint) error { return jobs.ErrMossAlreadyRunning }
	w = doRequest(router, http.MethodPost, "/admin/contest/tik25/moss", "")
	require.Equal(t, http.StatusConflict, w.Code)
}

func TestContestMoss_DeleteClearsRows(t *testing.T) {
	root, _, c := setupContestMossDB(t)
	require.NoError(t, db.DB.Create(&models.ContestMoss{ContestID: c.ID, ProblemID: 1, Language: "C"}).Error)
	router := authedRouter(root.ID, http.MethodDelete, "/admin/contest/:key/moss", AdminContestMossDelete)
	w := doRequest(router, http.MethodDelete, "/admin/contest/tik25/moss", "")
	require.Equal(t, http.StatusOK, w.Code)
	var n int64
	db.DB.Model(&models.ContestMoss{}).Count(&n)
	require.Zero(t, n)
}
