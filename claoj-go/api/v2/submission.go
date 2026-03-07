package v2

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/CLAOJ/claoj-go/db"
	"github.com/CLAOJ/claoj-go/models"
	"github.com/gin-gonic/gin"
	"github.com/pmezard/go-difflib/difflib"
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

// SubmissionDiff - GET /api/v2/submissions/:id1/diff/:id2
// Compares two submissions and returns the unified diff
func SubmissionDiff(c *gin.Context) {
	id1, err := strconv.ParseUint(c.Param("id1"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, apiError("invalid submission id"))
		return
	}
	id2, err := strconv.ParseUint(c.Param("id2"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, apiError("invalid submission id"))
		return
	}

	// Get both submissions
	var sub1, sub2 models.Submission
	if err := db.DB.Preload("Source").Where("id = ?", id1).First(&sub1).Error; err != nil {
		c.JSON(http.StatusNotFound, apiError("submission 1 not found"))
		return
	}
	if err := db.DB.Preload("Source").Where("id = ?", id2).First(&sub2).Error; err != nil {
		c.JSON(http.StatusNotFound, apiError("submission 2 not found"))
		return
	}

	// Check permissions: both submissions must be accessible
	// Public submissions or user's own submissions
	userID := uint(0)
	if uid, exists := c.Get("user_id"); exists {
		userID = uid.(uint)
	}

	isAdmin := false
	if userID > 0 {
		var profile models.Profile
		if err := db.DB.Where("user_id = ?", userID).First(&profile).Error; err == nil {
			isAdmin = profile.User.IsStaff || profile.User.IsSuperuser
		}
	}

	// Check access to submission 1
	if !sub1.Problem.IsPublic && sub1.UserID != userID && !isAdmin {
		c.JSON(http.StatusForbidden, apiError("access denied to submission 1"))
		return
	}
	// Check access to submission 2
	if !sub2.Problem.IsPublic && sub2.UserID != userID && !isAdmin {
		c.JSON(http.StatusForbidden, apiError("access denied to submission 2"))
		return
	}

	source1 := ""
	source2 := ""
	if sub1.Source != nil {
		source1 = sub1.Source.Source
	}
	if sub2.Source != nil {
		source2 = sub2.Source.Source
	}

	// Generate unified diff
	lines1 := strings.Split(source1, "\n")
	lines2 := strings.Split(source2, "\n")

	diffResult, err := difflib.GetUnifiedDiffString(difflib.UnifiedDiff{
		A:        lines1,
		B:        lines2,
		FromFile: "Submission #" + strconv.FormatUint(id1, 10),
		ToFile:   "Submission #" + strconv.FormatUint(id2, 10),
		Context:  3,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, apiError("failed to generate diff"))
		return
	}

	// Generate line-by-line diff for frontend rendering
	diffLines := []gin.H{}
	diffObj := difflib.NewMatcher(lines1, lines2)
	opCodes := diffObj.GetGroupedOpCodes(3)

	for _, op := range opCodes {
		tag := op[0]
		for i := int(tag.I1); i < int(tag.I2); i++ {
			diffLines = append(diffLines, gin.H{
				"type":  "delete",
				"line":  i + 1,
				"content": lines1[i],
			})
		}
		for i := int(tag.J1); i < int(tag.J2); i++ {
			diffLines = append(diffLines, gin.H{
				"type":  "add",
				"line":  i + 1,
				"content": lines2[i],
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"submission1": gin.H{
			"id":       sub1.ID,
			"problem":  sub1.Problem.Code,
			"user":     getUserUsername(sub1.UserID),
			"date":     sub1.Date,
			"language": getLanguageKey(sub1.LanguageID),
			"result":   sub1.Result,
			"points":   sub1.Points,
		},
		"submission2": gin.H{
			"id":       sub2.ID,
			"problem":  sub2.Problem.Code,
			"user":     getUserUsername(sub2.UserID),
			"date":     sub2.Date,
			"language": getLanguageKey(sub2.LanguageID),
			"result":   sub2.Result,
			"points":   sub2.Points,
		},
		"unified_diff": diffResult,
		"diff_lines":   diffLines,
		"stats": gin.H{
			"additions": countAdditions(diffLines),
			"deletions": countDeletions(diffLines),
		},
	})
}

// Helper functions for SubmissionDiff
func getUserUsername(userID uint) string {
	var profile models.Profile
	if err := db.DB.Where("user_id = ?", userID).First(&profile).Error; err != nil {
		return "unknown"
	}
	return profile.User.Username
}

func getLanguageKey(languageID uint) string {
	var lang models.Language
	if err := db.DB.Where("id = ?", languageID).First(&lang).Error; err != nil {
		return "unknown"
	}
	return lang.Key
}

func countAdditions(diffLines []gin.H) int {
	count := 0
	for _, line := range diffLines {
		if line["type"] == "add" {
			count++
		}
	}
	return count
}

func countDeletions(diffLines []gin.H) int {
	count := 0
	for _, line := range diffLines {
		if line["type"] == "delete" {
			count++
		}
	}
	return count
}
