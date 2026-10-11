package v2

import (
	"errors"
	"net/http"

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
		URL             *string `json:"url"`
	}
	var rows []row
	if err := db.DB.Table("judge_contestmoss m").
		Select("p.code AS problem_code, p.name AS problem_name, m.language, m.submission_count, m.url").
		Joins("JOIN judge_problem p ON p.id = m.problem_id").
		Joins("LEFT JOIN judge_contestproblem cp ON cp.problem_id = m.problem_id AND cp.contest_id = m.contest_id").
		Where("m.contest_id = ?", contest.ID).
		Order("cp.`order` ASC, m.id ASC").
		Scan(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}
	if rows == nil {
		rows = []row{}
	}
	c.JSON(http.StatusOK, gin.H{"running": jobs.IsContestMossRunning(contest.ID), "results": rows})
}

// AdminContestMossRun - POST /api/v2/admin/contest/:key/moss
func AdminContestMossRun(c *gin.Context) {
	contest, ok := contestForMoss(c)
	if !ok {
		return
	}
	if _, err := jobs.MossUserID(db.DB); err != nil {
		c.JSON(http.StatusBadRequest, apiUserError(err.Error()))
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
	res := db.DB.Where("contest_id = ?", contest.ID).Delete(&models.ContestMoss{})
	if res.Error != nil {
		c.JSON(http.StatusInternalServerError, apiError(res.Error.Error()))
		return
	}
	c.JSON(http.StatusOK, gin.H{"deleted": res.RowsAffected})
}
