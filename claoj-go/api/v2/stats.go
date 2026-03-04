package v2

import (
	"net/http"
	"time"

	"github.com/CLAOJ/claoj-go/db"
	"github.com/gin-gonic/gin"
)

// LanguageStats – GET /api/v2/stats/languages
func LanguageStats(c *gin.Context) {
	type Result struct {
		Language string  `json:"language"`
		Count    int     `json:"count"`
		ACRate   float64 `json:"ac_rate"`
	}

	var results []Result
	err := db.DB.Raw(`
		SELECT l.name as language, COUNT(*) as count, 
		       SUM(CASE WHEN s.result = 'AC' THEN 1 ELSE 0 END) * 100.0 / COUNT(*) as ac_rate
		FROM judge_submission s
		JOIN judge_language l ON s.language_id = l.id
		GROUP BY l.id
		ORDER BY count DESC
	`).Scan(&results).Error

	if err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	c.JSON(http.StatusOK, apiList(results))
}

// DailySubmissionStats – GET /api/v2/stats/submissions/daily
func DailySubmissionStats(c *gin.Context) {
	type Result struct {
		Date  string `json:"date"`
		Count int    `json:"count"`
	}

	thirtyDaysAgo := time.Now().AddDate(0, 0, -30)

	var results []Result
	err := db.DB.Raw(`
		SELECT DATE(date) as date, COUNT(*) as count
		FROM judge_submission
		WHERE date >= ?
		GROUP BY DATE(date)
		ORDER BY date ASC
	`, thirtyDaysAgo).Scan(&results).Error

	if err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	c.JSON(http.StatusOK, apiList(results))
}

// ProblemStats – GET /api/v2/stats/problem/:code
func ProblemStats(c *gin.Context) {
	code := c.Param("code")

	type StatusCount struct {
		Result string `json:"result"`
		Count  int    `json:"count"`
	}
	var counts []StatusCount

	err := db.DB.Raw(`
		SELECT result, COUNT(*) as count
		FROM judge_submission s
		JOIN judge_problem p ON s.problem_id = p.id
		WHERE p.code = ?
		GROUP BY result
	`, code).Scan(&counts).Error

	if err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"problem": code,
		"stats":   counts,
	})
}
