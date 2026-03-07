package v2

import (
	"net/http"
	"time"

	"github.com/CLAOJ/claoj-go/db"
	"github.com/CLAOJ/claoj-go/models"
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

// OverallStats - GET /api/v2/stats/overall
// Returns overall site statistics
func OverallStats(c *gin.Context) {
	type Stats struct {
		TotalUsers       int64 `json:"total_users"`
		TotalProblems    int64 `json:"total_problems"`
		TotalSubmissions int64 `json:"total_submissions"`
		TotalContests    int64 `json:"total_contests"`
		TotalOrganizations int64 `json:"total_organizations"`
		ActiveJudges     int64 `json:"active_judges"`
	}

	var stats Stats

	// Count users
	db.DB.Model(&models.Profile{}).Count(&stats.TotalUsers)

	// Count problems
	db.DB.Model(&models.Problem{}).Where("is_public = ?", true).Count(&stats.TotalProblems)

	// Count submissions
	db.DB.Model(&models.Submission{}).Count(&stats.TotalSubmissions)

	// Count contests
	db.DB.Model(&models.Contest{}).Where("is_visible = ? OR is_public = ?", true, true).Count(&stats.TotalContests)

	// Count organizations
	db.DB.Model(&models.Organization{}).Where("is_public = ?", true).Count(&stats.TotalOrganizations)

	// Count active judges (online in last 5 minutes)
	fiveMinAgo := time.Now().Add(-5 * time.Minute)
	db.DB.Model(&models.Judge{}).Where("last_ping > ?", fiveMinAgo).Count(&stats.ActiveJudges)

	c.JSON(http.StatusOK, stats)
}

// SubmissionStats - GET /api/v2/stats/submissions
// Returns submission statistics by result and language
func SubmissionStats(c *gin.Context) {
	type ResultStats struct {
		Result string `json:"result"`
		Count  int64  `json:"count"`
	}

	type LanguageStatsDetail struct {
		Language string  `json:"language"`
		Count    int64   `json:"count"`
		ACCount  int64   `json:"ac_count"`
		ACRate   float64 `json:"ac_rate"`
	}

	var resultStats []ResultStats
	db.DB.Raw(`
		SELECT result, COUNT(*) as count
		FROM judge_submission
		GROUP BY result
		ORDER BY count DESC
	`).Scan(&resultStats)

	var languageStats []LanguageStatsDetail
	db.DB.Raw(`
		SELECT l.name as language,
		       COUNT(*) as count,
		       SUM(CASE WHEN s.result = 'AC' THEN 1 ELSE 0 END) as ac_count,
		       SUM(CASE WHEN s.result = 'AC' THEN 1 ELSE 0 END) * 100.0 / COUNT(*) as ac_rate
		FROM judge_submission s
		JOIN judge_language l ON s.language_id = l.id
		GROUP BY l.id, l.name
		ORDER BY count DESC
	`).Scan(&languageStats)

	c.JSON(http.StatusOK, gin.H{
		"by_result":  resultStats,
		"by_language": languageStats,
	})
}

// ProblemStatsList - GET /api/v2/stats/problems
// Returns problem statistics
func ProblemStatsList(c *gin.Context) {
	type ProblemStat struct {
		Code        string  `json:"code"`
		Name        string  `json:"name"`
		Points      float64 `json:"points"`
		Submissions int64   `json:"submissions"`
		Solvers     int64   `json:"solvers"`
		ACRate      float64 `json:"ac_rate"`
	}

	var stats []ProblemStat
	db.DB.Raw(`
		SELECT p.code, p.name, p.points,
		       COUNT(s.id) as submissions,
		       COUNT(DISTINCT s.user_id) as solvers,
		       SUM(CASE WHEN s.result = 'AC' THEN 1 ELSE 0 END) * 100.0 / COUNT(*) as ac_rate
		FROM judge_problem p
		LEFT JOIN judge_submission s ON p.id = s.problem_id
		WHERE p.is_public = true
		GROUP BY p.id, p.code, p.name, p.points
		ORDER BY submissions DESC
		LIMIT 100
	`).Scan(&stats)

	c.JSON(http.StatusOK, apiList(stats))
}

// JudgeStats - GET /api/v2/stats/judges
// Returns judge queue statistics
func JudgeStats(c *gin.Context) {
	type JudgeStat struct {
		Name      string    `json:"name"`
		Online    bool      `json:"online"`
		LastPing  time.Time `json:"last_ping"`
		Load      int       `json:"load"`
		Submissions int64   `json:"submissions_processed"`
	}

	fiveMinAgo := time.Now().Add(-5 * time.Minute)

	var judges []JudgeStat
	db.DB.Raw(`
		SELECT name, last_ping, load,
		       CASE WHEN last_ping > ? THEN true ELSE false END as online,
		       0 as submissions_processed
		FROM judge_judge
		ORDER BY online DESC, name
	`, fiveMinAgo).Scan(&judges)

	// Get queue size (pending submissions)
	var queueSize int64
	db.DB.Model(&models.Submission{}).Where("status = ?", "P").Count(&queueSize)

	c.JSON(http.StatusOK, gin.H{
		"judges":     judges,
		"queue_size": queueSize,
	})
}

// UserStats - GET /api/v2/stats/users
// Returns user statistics
func UserStats(c *gin.Context) {
	type UserStat struct {
		Total      int64 `json:"total"`
		Active     int64 `json:"active"`
		Banned     int64 `json:"banned"`
		Unverified int64 `json:"unverified"`
	}

	var stat UserStat

	// Total users
	db.DB.Model(&models.Profile{}).Count(&stat.Total)

	// Active users (logged in within 30 days)
	thirtyDaysAgo := time.Now().AddDate(0, 0, -30)
	db.DB.Table("auth_user").
		Where("last_login > ?", thirtyDaysAgo).
		Count(&stat.Active)

	// Banned users
	db.DB.Table("auth_user").
		Where("is_active = ?", false).
		Count(&stat.Banned)

	// Unverified users
	db.DB.Table("auth_user").
		Where("email_verified = ?", false).
		Count(&stat.Unverified)

	c.JSON(http.StatusOK, stat)
}

// ContestStatsList - GET /api/v2/stats/contests
// Returns contest statistics
func ContestStatsList(c *gin.Context) {
	type ContestStat struct {
		Key           string    `json:"key"`
		Name          string    `json:"name"`
		StartTime     time.Time `json:"start_time"`
		EndTime       time.Time `json:"end_time"`
		Participants  int64     `json:"participants"`
		Submissions   int64     `json:"submissions"`
	}

	var stats []ContestStat
	db.DB.Raw(`
		SELECT c.key, c.name, c.start_time, c.end_time,
		       COUNT(DISTINCT cp.id) as participants,
		       COUNT(s.id) as submissions
		FROM judge_contest c
		LEFT JOIN judge_contestparticipation cp ON c.id = cp.contest_id
		LEFT JOIN judge_submission s ON cp.id = s.contest_participation_id
		WHERE c.is_visible = true OR c.is_public = true
		GROUP BY c.id, c.key, c.name, c.start_time, c.end_time
		ORDER BY c.start_time DESC
		LIMIT 50
	`).Scan(&stats)

	c.JSON(http.StatusOK, apiList(stats))
}
