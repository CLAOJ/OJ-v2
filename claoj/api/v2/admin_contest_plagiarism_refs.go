package v2

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/CLAOJ/claoj/auth"
	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// maxRefSourceBytes caps one reference solution's source.
const maxRefSourceBytes = 65536

// contestProblemIDs selects the problem ids attached to one contest.
func contestProblemIDs(contestID uint) *gorm.DB {
	return db.DB.Model(&models.ContestProblem{}).Select("problem_id").Where("contest_id = ?", contestID)
}

// AdminContestRefList - GET /api/v2/admin/contest/:key/plagiarism/references
func AdminContestRefList(c *gin.Context) {
	contest, ok := contestForMoss(c)
	if !ok {
		return
	}
	type row struct {
		ID          uint      `json:"id"`
		ProblemCode string    `json:"problem_code"`
		Language    string    `json:"language"`
		Origin      string    `json:"origin"`
		Model       *string   `json:"model"`
		Source      string    `json:"source"`
		CreatedAt   time.Time `json:"created_at"`
	}
	var rows []row
	if err := db.DB.Table("plagiarism_ai_reference r").
		Select("r.id, p.code AS problem_code, r.language, r.origin, r.model, r.source, r.created_at").
		Joins("JOIN judge_problem p ON p.id = r.problem_id").
		Joins("JOIN judge_contestproblem cp ON cp.problem_id = r.problem_id AND cp.contest_id = ?", contest.ID).
		Order("cp.`order` ASC, r.language ASC, r.id ASC").
		Scan(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}
	if rows == nil {
		rows = []row{}
	}
	c.JSON(http.StatusOK, gin.H{"references": rows})
}

// AdminContestRefAdd - POST /api/v2/admin/contest/:key/plagiarism/references
func AdminContestRefAdd(c *gin.Context) {
	contest, ok := contestForMoss(c)
	if !ok {
		return
	}
	var req struct {
		ProblemCode string `json:"problem_code"`
		Language    string `json:"language"`
		Source      string `json:"source"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, apiUserError("invalid request body"))
		return
	}
	var problem models.Problem
	if err := db.DB.Select("id").Where("code = ? AND id IN (?)", req.ProblemCode, contestProblemIDs(contest.ID)).First(&problem).Error; err != nil {
		c.JSON(http.StatusBadRequest, apiUserError("problem is not in this contest"))
		return
	}
	var langCount int64
	if err := db.DB.Model(&models.Language{}).Where("common_name = ?", req.Language).Count(&langCount).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}
	if langCount == 0 {
		c.JSON(http.StatusBadRequest, apiUserError("unknown language"))
		return
	}
	if strings.TrimSpace(req.Source) == "" {
		c.JSON(http.StatusBadRequest, apiUserError("source must not be empty"))
		return
	}
	if len(req.Source) > maxRefSourceBytes {
		c.JSON(http.StatusBadRequest, apiUserError("source is too large"))
		return
	}
	var createdBy *uint
	if id, ok := auth.CurrentProfileID(c); ok {
		createdBy = &id
	}
	ref := models.PlagiarismAIReference{
		ProblemID: problem.ID,
		Language:  req.Language,
		Source:    req.Source,
		Origin:    "admin",
		CreatedBy: createdBy,
		CreatedAt: time.Now(),
	}
	if err := db.DB.Create(&ref).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": ref.ID})
}

// AdminContestRefDelete - DELETE /api/v2/admin/contest/:key/plagiarism/references/:id
func AdminContestRefDelete(c *gin.Context) {
	contest, ok := contestForMoss(c)
	if !ok {
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusNotFound, apiUserError("reference not found"))
		return
	}
	res := db.DB.Where("id = ? AND problem_id IN (?)", id, contestProblemIDs(contest.ID)).Delete(&models.PlagiarismAIReference{})
	if res.Error != nil {
		c.JSON(http.StatusInternalServerError, apiError(res.Error.Error()))
		return
	}
	if res.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, apiUserError("reference not found"))
		return
	}
	c.JSON(http.StatusOK, gin.H{"deleted": res.RowsAffected})
}

// AdminContestRefRegenerate - DELETE /api/v2/admin/contest/:key/plagiarism/references?problem_code=X&origin=generated
// Clears one problem's generated references ahead of regenerating them.
func AdminContestRefRegenerate(c *gin.Context) {
	contest, ok := contestForMoss(c)
	if !ok {
		return
	}
	if c.Query("origin") != "generated" {
		c.JSON(http.StatusBadRequest, apiUserError("origin must be generated"))
		return
	}
	var problem models.Problem
	if err := db.DB.Select("id").Where("code = ? AND id IN (?)", c.Query("problem_code"), contestProblemIDs(contest.ID)).First(&problem).Error; err != nil {
		c.JSON(http.StatusBadRequest, apiUserError("problem is not in this contest"))
		return
	}
	res := db.DB.Where("problem_id = ? AND origin = ?", problem.ID, "generated").Delete(&models.PlagiarismAIReference{})
	if res.Error != nil {
		c.JSON(http.StatusInternalServerError, apiError(res.Error.Error()))
		return
	}
	c.JSON(http.StatusOK, gin.H{"deleted": res.RowsAffected})
}
