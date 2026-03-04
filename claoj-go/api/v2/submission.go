package v2

import (
	"net/http"
	"strconv"
	"time"

	"github.com/CLAOJ/claoj-go/db"
	"github.com/CLAOJ/claoj-go/models"
	"github.com/gin-gonic/gin"
)

// SubmissionList – GET /api/v2/submissions
// Supports ?user=username and ?problem=code filters
func SubmissionList(c *gin.Context) {
	page, pageSize := parsePagination(c)
	userFilter := c.Query("user")
	problemFilter := c.Query("problem")
	resultFilter := c.Query("result")
	langFilter := c.Query("language")

	q := db.DB.Model(&models.Submission{}).
		Joins("JOIN judge_profile jp ON jp.id = judge_submission.user_id").
		Joins("JOIN auth_user au ON au.id = jp.user_id").
		Joins("JOIN judge_problem pr ON pr.id = judge_submission.problem_id").
		Joins("JOIN judge_language l ON l.id = judge_submission.language_id").
		Where("pr.is_public = ?", true).
		Order("judge_submission.date DESC")

	if userFilter != "" {
		q = q.Where("au.username = ?", userFilter)
	}
	if problemFilter != "" {
		q = q.Where("pr.code = ?", problemFilter)
	}
	if resultFilter != "" {
		q = q.Where("judge_submission.result = ?", resultFilter)
	}
	if langFilter != "" {
		q = q.Where("l.key = ?", langFilter)
	}

	q = q.Offset((page - 1) * pageSize).Limit(pageSize)

	type Row struct {
		ID       uint      `json:"id"`
		Problem  string    `json:"problem"`
		User     string    `json:"user"`
		Date     time.Time `json:"date"`
		Language string    `json:"language"`
		Status   string    `json:"status"`
		Result   *string   `json:"result"`
		Points   *float64  `json:"points"`
		Time     *float64  `json:"time"`
		Memory   *float64  `json:"memory"`
	}

	var rows []struct {
		models.Submission
		Username    string
		ProblemCode string
		LangKey     string
	}

	q = q.Select(
		"judge_submission.*, au.username, pr.code as problem_code, l.key as lang_key",
	)

	if err := q.Find(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	items := make([]Row, len(rows))
	for i, r := range rows {
		items[i] = Row{
			ID:       r.Submission.ID,
			Problem:  r.ProblemCode,
			User:     r.Username,
			Date:     r.Submission.Date,
			Language: r.LangKey,
			Status:   r.Submission.Status,
			Result:   r.Submission.Result,
			Points:   r.Submission.Points,
			Time:     r.Submission.Time,
			Memory:   r.Submission.Memory,
		}
	}
	c.JSON(http.StatusOK, apiList(items))
}

// SubmissionDetail – GET /api/v2/submission/:id
func SubmissionDetail(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, apiError("invalid submission id"))
		return
	}

	var sub models.Submission
	if err := db.DB.
		Preload("User.User").
		Preload("Problem").
		Preload("Language").
		Preload("Source").
		Preload("TestCases", "1=1 ORDER BY `case` ASC").
		Where("id = ?", id).First(&sub).Error; err != nil {
		c.JSON(http.StatusNotFound, apiError("submission not found"))
		return
	}

	// Check problem is public
	if !sub.Problem.IsPublic {
		c.JSON(http.StatusNotFound, apiError("submission not found"))
		return
	}

	source := ""
	if sub.Source != nil {
		source = sub.Source.Source
	}

	type TestCaseItem struct {
		Case     int      `json:"case"`
		Status   string   `json:"status"`
		Time     *float64 `json:"time"`
		Memory   *float64 `json:"memory"`
		Points   *float64 `json:"points"`
		Total    *float64 `json:"total"`
		Feedback string   `json:"feedback"`
	}
	testCases := make([]TestCaseItem, len(sub.TestCases))
	for i, tc := range sub.TestCases {
		testCases[i] = TestCaseItem{tc.Case, tc.Status, tc.Time, tc.Memory, tc.Points, tc.Total, tc.Feedback}
	}

	c.JSON(http.StatusOK, gin.H{
		"id":          sub.ID,
		"problem":     sub.Problem.Code,
		"user":        sub.User.User.Username,
		"date":        sub.Date,
		"language":    sub.Language.Key,
		"status":      sub.Status,
		"result":      sub.Result,
		"points":      sub.Points,
		"time":        sub.Time,
		"memory":      sub.Memory,
		"source":      source,
		"case_points": sub.CasePoints,
		"case_total":  sub.CaseTotal,
		"test_cases":  testCases,
	})
}
