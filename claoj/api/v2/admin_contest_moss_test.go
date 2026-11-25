package v2

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
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
		&models.Profile{}, &models.Problem{}, &models.Contest{}, &models.PlagiarismResult{}, &models.MiscConfig{},
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

func useTempReportDir(t *testing.T) {
	t.Helper()
	old := jobs.PlagiarismDir
	jobs.PlagiarismDir = t.TempDir()
	t.Cleanup(func() { jobs.PlagiarismDir = old })
}

func writeReportFile(t *testing.T, path, body string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(body), 0o644))
}

func TestContestMoss_RequiresPermission(t *testing.T) {
	_, staff, _ := setupContestMossDB(t)
	router := authedRouter(staff.ID, http.MethodGet, "/admin/contest/:key/moss", AdminContestMossList)
	w := doRequest(router, http.MethodGet, "/admin/contest/tik25/moss", "")
	require.Equal(t, http.StatusForbidden, w.Code)
}

func TestContestMoss_ListReturnsReportPath(t *testing.T) {
	root, _, c := setupContestMossDB(t)
	p := models.Problem{Code: "knap", Name: "Knapsack"}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&p).Error)
	reportPath := "knap/cpp"
	require.NoError(t, db.DB.Create(&models.PlagiarismResult{ContestID: c.ID, ProblemID: p.ID, Language: "C++", SubmissionCount: 4, ReportPath: &reportPath, CreatedAt: time.Now()}).Error)

	router := authedRouter(root.ID, http.MethodGet, "/admin/contest/:key/moss", AdminContestMossList)
	w := doRequest(router, http.MethodGet, "/admin/contest/tik25/moss", "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var body struct {
		Running bool   `json:"running"`
		Error   string `json:"error"`
		Results []struct {
			ProblemCode     string  `json:"problem_code"`
			Language        string  `json:"language"`
			SubmissionCount uint    `json:"submission_count"`
			ReportPath      *string `json:"report_path"`
		} `json:"results"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.False(t, body.Running)
	require.Empty(t, body.Error)
	require.Len(t, body.Results, 1)
	require.Equal(t, "knap", body.Results[0].ProblemCode)
	require.Equal(t, "C++", body.Results[0].Language)
	require.Equal(t, uint(4), body.Results[0].SubmissionCount)
	require.NotNil(t, body.Results[0].ReportPath)
	require.Equal(t, reportPath, *body.Results[0].ReportPath)
}

func TestContestMoss_RunEnqueues(t *testing.T) {
	root, _, c := setupContestMossDB(t)
	var enqueued []uint
	enqueueContestMoss = func(id uint) error { enqueued = append(enqueued, id); return nil }
	t.Cleanup(func() { enqueueContestMoss = jobs.EnqueueContestMoss })

	router := authedRouter(root.ID, http.MethodPost, "/admin/contest/:key/moss", AdminContestMossRun)
	w := doRequest(router, http.MethodPost, "/admin/contest/tik25/moss", "")
	require.Equal(t, http.StatusAccepted, w.Code, w.Body.String())
	require.Equal(t, []uint{c.ID}, enqueued)

	enqueueContestMoss = func(uint) error { return jobs.ErrMossAlreadyRunning }
	w = doRequest(router, http.MethodPost, "/admin/contest/tik25/moss", "")
	require.Equal(t, http.StatusConflict, w.Code)
}

func TestContestMoss_DeleteRemovesReportDir(t *testing.T) {
	useTempReportDir(t)
	root, _, c := setupContestMossDB(t)
	reportPath := "knap/cpp"
	require.NoError(t, db.DB.Create(&models.PlagiarismResult{ContestID: c.ID, ProblemID: 1, Language: "C++", ReportPath: &reportPath, CreatedAt: time.Now()}).Error)
	writeReportFile(t, filepath.Join(jobs.ContestReportDir(c.ID), "knap", "cpp", "index.html"), "report")

	router := authedRouter(root.ID, http.MethodDelete, "/admin/contest/:key/moss", AdminContestMossDelete)
	w := doRequest(router, http.MethodDelete, "/admin/contest/tik25/moss", "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var body struct {
		Deleted int64 `json:"deleted"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Equal(t, int64(1), body.Deleted)

	var n int64
	db.DB.Model(&models.PlagiarismResult{}).Count(&n)
	require.Zero(t, n)
	_, err := os.Stat(jobs.ContestReportDir(c.ID))
	require.True(t, os.IsNotExist(err), "report dir should be removed, stat err: %v", err)
}

func TestContestMoss_ReportServesWithSandbox(t *testing.T) {
	useTempReportDir(t)
	root, _, c := setupContestMossDB(t)
	writeReportFile(t, filepath.Join(jobs.ContestReportDir(c.ID), "knap", "cpp", "index.html"), "<html>knap</html>")

	router := authedRouter(root.ID, http.MethodGet, "/admin/contest/:key/moss/report/*path", AdminContestMossReport)
	w := doRequest(router, http.MethodGet, "/admin/contest/tik25/moss/report/knap/cpp/index.html", "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.Contains(t, w.Body.String(), "<html>knap</html>")
	require.Equal(t, "sandbox allow-scripts allow-popups", w.Header().Get("Content-Security-Policy"))
	require.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))
}

func TestContestMoss_ReportRejectsTraversal(t *testing.T) {
	useTempReportDir(t)
	root, _, c := setupContestMossDB(t)
	other := models.Contest{Key: "other", Name: "Other", StartTime: time.Now(), EndTime: time.Now().Add(time.Hour)}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&other).Error)
	writeReportFile(t, filepath.Join(jobs.ContestReportDir(c.ID), "knap", "cpp", "index.html"), "own")
	writeReportFile(t, filepath.Join(jobs.ContestReportDir(other.ID), "x.html"), "other contest")
	writeReportFile(t, filepath.Join(jobs.PlagiarismDir, "secret"), "secret")

	router := authedRouter(root.ID, http.MethodGet, "/admin/contest/:key/moss/report/*path", AdminContestMossReport)
	for _, target := range []string{
		"/admin/contest/tik25/moss/report/../1/x.html",
		"/admin/contest/tik25/moss/report/%2e%2e/%2e%2e/secret",
		"/admin/contest/tik25/moss/report/etc/passwd",
		"/admin/contest/tik25/moss/report/..%5c..%5csecret",
		"/admin/contest/tik25/moss/report/../" + strconv.FormatUint(uint64(other.ID), 10) + "/x.html",
		"/admin/contest/tik25/moss/report/knap/cpp",
	} {
		w := doRequest(router, http.MethodGet, target, "")
		require.Equal(t, http.StatusNotFound, w.Code, "target %s: %s", target, w.Body.String())
	}
}

func TestContestMoss_ReportRequiresPermission(t *testing.T) {
	useTempReportDir(t)
	_, staff, c := setupContestMossDB(t)
	writeReportFile(t, filepath.Join(jobs.ContestReportDir(c.ID), "knap", "cpp", "index.html"), "report")

	router := authedRouter(staff.ID, http.MethodGet, "/admin/contest/:key/moss/report/*path", AdminContestMossReport)
	w := doRequest(router, http.MethodGet, "/admin/contest/tik25/moss/report/knap/cpp/index.html", "")
	require.Equal(t, http.StatusForbidden, w.Code)
}
