package v2

import (
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/CLAOJ/claoj/auth"
	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/jobs"
	"github.com/CLAOJ/claoj/models"
	"github.com/gin-gonic/gin"
)

// enqueueContestMoss is swapped in tests.
var enqueueContestMoss = jobs.EnqueueContestMoss

// contestForMoss loads the contest and enforces v1's judge.moss_contest
// permission; it writes the error response and returns false on failure.
func contestForMoss(c *gin.Context) (models.Contest, bool) {
	var contest models.Contest
	if !auth.HasPerm(c, "judge.moss_contest") {
		c.JSON(http.StatusForbidden, apiError("you are not allowed to run MOSS"))
		return contest, false
	}
	if err := db.DB.Where("`key` = ?", c.Param("key")).First(&contest).Error; err != nil {
		c.JSON(http.StatusNotFound, apiError("contest not found"))
		return contest, false
	}
	return contest, true
}

// AdminContestMossList - GET /api/v2/admin/contest/:key/moss
func AdminContestMossList(c *gin.Context) {
	contest, ok := contestForMoss(c)
	if !ok {
		return
	}
	type row struct {
		ProblemCode     string  `json:"problem_code"`
		ProblemName     string  `json:"problem_name"`
		Language        string  `json:"language"`
		SubmissionCount uint    `json:"submission_count"`
		ReportPath      *string `json:"report_path"`
	}
	var rows []row
	if err := db.DB.Table("plagiarism_result r").
		Select("p.code AS problem_code, p.name AS problem_name, r.language, r.submission_count, r.report_path").
		Joins("JOIN judge_problem p ON p.id = r.problem_id").
		Joins("LEFT JOIN judge_contestproblem cp ON cp.problem_id = r.problem_id AND cp.contest_id = r.contest_id").
		Where("r.contest_id = ?", contest.ID).
		Order("cp.`order` ASC, r.id ASC").
		Scan(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}
	if rows == nil {
		rows = []row{}
	}
	c.JSON(http.StatusOK, gin.H{
		"running": jobs.IsContestMossRunning(contest.ID),
		"error":   jobs.MossRunError(contest.ID),
		"results": rows,
	})
}

// AdminContestMossRun - POST /api/v2/admin/contest/:key/moss
func AdminContestMossRun(c *gin.Context) {
	contest, ok := contestForMoss(c)
	if !ok {
		return
	}
	if err := enqueueContestMoss(contest.ID); err != nil {
		if errors.Is(err, jobs.ErrMossAlreadyRunning) {
			c.JSON(http.StatusConflict, apiUserError(err.Error()))
			return
		}
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"queued": true})
}

// AdminContestMossDelete - DELETE /api/v2/admin/contest/:key/moss
func AdminContestMossDelete(c *gin.Context) {
	contest, ok := contestForMoss(c)
	if !ok {
		return
	}
	res := db.DB.Where("contest_id = ?", contest.ID).Delete(&models.PlagiarismResult{})
	if res.Error != nil {
		c.JSON(http.StatusInternalServerError, apiError(res.Error.Error()))
		return
	}
	if err := os.RemoveAll(jobs.ContestReportDir(contest.ID)); err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}
	c.JSON(http.StatusOK, gin.H{"deleted": res.RowsAffected})
}

// AdminContestMossReport - GET /api/v2/admin/contest/:key/moss/report/*path
// Serves one file from the contest's compare50 report dir with a sandbox CSP,
// so scripts inside a report cannot act with the admin's session.
func AdminContestMossReport(c *gin.Context) {
	contest, ok := contestForMoss(c)
	if !ok {
		return
	}
	raw := c.Param("path")
	for _, seg := range strings.FieldsFunc(raw, func(r rune) bool { return r == '/' || r == '\\' }) {
		if seg == ".." {
			c.JSON(http.StatusNotFound, apiError("report not found"))
			return
		}
	}
	root := jobs.ContestReportDir(contest.ID)
	full := filepath.Join(root, filepath.Clean("/"+raw))
	rel, err := filepath.Rel(root, full)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		c.JSON(http.StatusNotFound, apiError("report not found"))
		return
	}
	f, err := os.Open(full)
	if err != nil {
		c.JSON(http.StatusNotFound, apiError("report not found"))
		return
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || info.IsDir() {
		c.JSON(http.StatusNotFound, apiError("report not found"))
		return
	}
	c.Header("Content-Security-Policy", "sandbox allow-scripts")
	c.Header("X-Content-Type-Options", "nosniff")
	// ServeContent rather than c.File: http.ServeFile redirects any request
	// ending in /index.html, which is exactly what report pages are.
	http.ServeContent(c.Writer, c.Request, info.Name(), info.ModTime(), f)
}
