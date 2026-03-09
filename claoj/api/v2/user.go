package v2

import (
	"fmt"
	"math"
	"net/http"
	"os"
	"time"

	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/jobs"
	"github.com/CLAOJ/claoj/models"
	"github.com/CLAOJ/claoj/scoring"
	"github.com/gin-gonic/gin"
)

// UserList – GET /api/v2/users
// @Description Returns a list of all users with their basic statistics.
// @Tags Users
// @Summary List users
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Items per page" default(100)
// @Success 200 {object} map[string]interface{}
// @Router /users [get]
func UserList(c *gin.Context) {
	page, pageSize := parsePagination(c)
	var profiles []models.Profile

	if err := db.DB.
		Preload("User").
		Where("judge_profile.is_unlisted = ?", false).
		Joins("JOIN auth_user au ON au.id = judge_profile.user_id").
		Order("judge_profile.performance_points DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&profiles).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	type Item struct {
		Username          string  `json:"username"`
		Points            float64 `json:"points"`
		PerformancePoints float64 `json:"performance_points"`
		Rating            *int    `json:"rating"`
		ProblemCount      int     `json:"problem_count"`
		DisplayRank       string  `json:"display_rank"`
		AvatarURL         string  `json:"avatar_url"`
	}
	items := make([]Item, len(profiles))
	for i, p := range profiles {
		items[i] = Item{
			Username:          p.User.Username,
			Points:            p.Points,
			PerformancePoints: p.PerformancePoints,
			Rating:            p.Rating,
			ProblemCount:      p.ProblemCount,
			DisplayRank:       p.DisplayRank,
			AvatarURL:         getAvatarURL(&p),
		}
	}
	c.JSON(http.StatusOK, apiList(items))
}

// UserDetail – GET /api/v2/user/:user
// @Description Get detailed user profile information including ratings and organizations.
// @Tags Users
// @Summary Get user profile
// @Produce json
// @Param user path string true "Username"
// @Success 200 {object} map[string]interface{}
// @Failure 404 {object} map[string]string
// @Router /user/{user} [get]
func UserDetail(c *gin.Context) {
	username := c.Param("user")
	var profile models.Profile

	if err := db.DB.
		Preload("User").
		Preload("Organizations").
		Joins("JOIN auth_user au ON au.id = judge_profile.user_id").
		Where("au.username = ?", username).
		First(&profile).Error; err != nil {
		c.JSON(http.StatusNotFound, apiError("user not found"))
		return
	}

	// Calculate points rank
	var rank int64
	db.DB.Model(&models.Profile{}).Where("points > ?", profile.Points).Count(&rank)
	rank++ // 1-based

	// Calculate rating rank
	var ratingRank int64
	if profile.Rating != nil {
		db.DB.Model(&models.Profile{}).Where("rating > ?", *profile.Rating).Count(&ratingRank)
		ratingRank++
	}

	type OrgItem struct {
		ID   uint   `json:"id"`
		Name string `json:"name"`
	}
	orgs := make([]OrgItem, len(profile.Organizations))
	for i, o := range profile.Organizations {
		orgs[i] = OrgItem{o.ID, o.Name}
	}

	c.JSON(http.StatusOK, gin.H{
		"username":            profile.User.Username,
		"display_name":        profile.UsernameDisplayOverride,
		"about":               profile.About,
		"points":              profile.Points,
		"performance_points":  profile.PerformancePoints,
		"contribution_points": profile.ContributionPoints,
		"rating":              profile.Rating,
		"problem_count":       profile.ProblemCount,
		"display_rank":        profile.DisplayRank,
		"rank":                rank,
		"rating_rank":         ratingRank,
		"avatar_url":          getAvatarURL(&profile),
		"organizations":       orgs,
		"last_access":         profile.LastAccess,
		"date_joined":         profile.User.DateJoined,
	})
}

// UserSolvedProblems – GET /api/v2/user/:user/solved
func UserSolvedProblems(c *gin.Context) {
	username := c.Param("user")
	var profile models.Profile

	if err := db.DB.Joins("JOIN auth_user au ON au.id = judge_profile.user_id").
		Where("au.username = ?", username).First(&profile).Error; err != nil {
		c.JSON(http.StatusNotFound, apiError("user not found"))
		return
	}

	var solved []struct {
		Code   string  `json:"code"`
		Points float64 `json:"points"`
	}
	db.DB.Table("judge_submission").
		Select("pr.code, MAX(judge_submission.points) as points").
		Joins("JOIN judge_problem pr ON pr.id = judge_submission.problem_id").
		Where("judge_submission.user_id = ? AND judge_submission.result = 'AC'", profile.ID).
		Group("pr.code").
		Scan(&solved)

	c.JSON(http.StatusOK, apiList(solved))
}

// UserRatingHistory – GET /api/v2/user/:user/rating
func UserRatingHistory(c *gin.Context) {
	username := c.Param("user")
	var profile models.Profile

	if err := db.DB.Joins("JOIN auth_user au ON au.id = judge_profile.user_id").
		Where("au.username = ?", username).First(&profile).Error; err != nil {
		c.JSON(http.StatusNotFound, apiError("user not found"))
		return
	}

	var history []struct {
		Date      time.Time `json:"date"`
		Rating    int       `json:"rating"`
		Contest   string    `json:"contest"`
		ContestID string    `json:"contest_key"`
	}
	db.DB.Table("judge_rating").
		Select("judge_rating.last_rated as date, judge_rating.rating, jc.name as contest, jc.key as contest_key").
		Joins("JOIN judge_contest jc ON jc.id = judge_rating.contest_id").
		Where("judge_rating.user_id = ?", profile.ID).
		Order("judge_rating.last_rated ASC").
		Scan(&history)

	c.JSON(http.StatusOK, apiList(history))
}

// UserPPBreakdown – GET /api/v2/user/:user/pp-breakdown
func UserPPBreakdown(c *gin.Context) {
	username := c.Param("user")
	var profile models.Profile

	if err := db.DB.Joins("JOIN auth_user au ON au.id = judge_profile.user_id").
		Where("au.username = ?", username).First(&profile).Error; err != nil {
		c.JSON(http.StatusNotFound, apiError("user not found"))
		return
	}

	// Get max points per public problem, ordered by points DESC
	type ProblemResult struct {
		Code        string  `json:"code"`
		Name        string  `json:"name"`
		MaxPoints   float64 `json:"points"`
		Weight      float64 `json:"weight"`
		Contribution float64 `json:"contribution"`
	}
	var problems []ProblemResult

	// First get the raw max points data
	type RawResult struct {
		Code        string  `gorm:"column:code"`
		Name        string  `gorm:"column:name"`
		MaxPoints   float64 `gorm:"column:max_points"`
	}
	var rawData []RawResult

	if err := db.DB.Raw(`
		SELECT pr.code, pr.name, MAX(s.points) as max_points
		FROM judge_submission s
		JOIN judge_problem pr ON s.problem_id = pr.id
		WHERE s.user_id = ? AND pr.is_public = 1 AND s.points IS NOT NULL
		GROUP BY pr.id, pr.code, pr.name
		HAVING max_points > 0
		ORDER BY max_points DESC
		LIMIT 100
	`, profile.ID).Scan(&rawData).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	// Calculate weighted contributions
	problems = make([]ProblemResult, len(rawData))
	weightedSum := 0.0
	for i, res := range rawData {
		weight := math.Pow(0.95, float64(i))
		contribution := weight * res.MaxPoints
		weightedSum += contribution
		problems[i] = ProblemResult{
			Code:         res.Code,
			Name:         res.Name,
			MaxPoints:    res.MaxPoints,
			Weight:       weight,
			Contribution: contribution,
		}
	}

	// Calculate bonus from solved count
	var solvedCount int64
	if err := db.DB.Raw(`
		SELECT COUNT(DISTINCT s.problem_id)
		FROM judge_submission s
		JOIN judge_problem p ON s.problem_id = p.id
		WHERE s.user_id = ? AND p.is_public = 1 AND s.result = 'AC' AND s.case_points >= s.case_total
	`, profile.ID).Scan(&solvedCount).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	bonusPoints := scoring.PpBonusFunction(int(solvedCount))
	totalPP := weightedSum + bonusPoints

	c.JSON(http.StatusOK, gin.H{
		"performance_points": totalPP,
		"problems":           problems,
		"bonus": gin.H{
			"solved_count":   solvedCount,
			"bonus_points":   bonusPoints,
		},
		"weighted_sum": weightedSum,
		"total":        totalPP,
	})
}

// UserAnalytics – GET /api/v2/user/:user/analytics
func UserAnalytics(c *gin.Context) {
	username := c.Param("user")
	var profile models.Profile

	if err := db.DB.Joins("JOIN auth_user au ON au.id = judge_profile.user_id").
		Where("au.username = ?", username).First(&profile).Error; err != nil {
		c.JSON(http.StatusNotFound, apiError("user not found"))
		return
	}

	// Get submission statistics
	type SubStats struct {
		TotalSubs      int64   `json:"total_submissions"`
		AcceptedSubs   int64   `json:"accepted_submissions"`
		TotalPoints    float64 `json:"total_points"`
		AveragePoints  float64 `json:"average_points"`
		BestSubmission float64 `json:"best_submission"`
	}
	var stats SubStats
	db.DB.Table("judge_submission").
		Select("COUNT(*) as total_submissions, SUM(CASE WHEN result = 'AC' THEN 1 ELSE 0 END) as accepted_submissions, SUM(points) as total_points, AVG(points) as average_points, MAX(points) as best_submission").
		Where("user_id = ?", profile.ID).
		Scan(&stats)

	// Get language distribution
	type LangDist struct {
		Language string `json:"language"`
		Count    int64  `json:"count"`
	}
	var langDist []LangDist
	db.DB.Table("judge_submission").
		Select("l.key as language, COUNT(*) as count").
		Joins("JOIN judge_language l ON l.id = judge_submission.language_id").
		Where("user_id = ?", profile.ID).
		Group("l.key").
		Order("count DESC").
		Scan(&langDist)

	// Get submission activity by date (last 30 days)
	type Activity struct {
		Date  string `json:"date"`
		Count int64  `json:"count"`
	}
	var activity []Activity
	db.DB.Table("judge_submission").
		Select("DATE(date) as date, COUNT(*) as count").
		Where("user_id = ? AND date >= DATE_SUB(NOW(), INTERVAL 30 DAY)", profile.ID).
		Group("DATE(date)").
		Order("date ASC").
		Scan(&activity)

	// Get problem group distribution
	type GroupDist struct {
		Group  string  `json:"group"`
		Solved int64   `json:"solved"`
		Points float64 `json:"points"`
	}
	var groupDist []GroupDist
	db.DB.Table("judge_submission").
		Select("jp.group, COUNT(DISTINCT js.problem_id) as solved, SUM(js.points) as points").
		Joins("JOIN judge_problem jp ON jp.id = js.problem_id").
		Where("js.user_id = ? AND js.result = 'AC'", profile.ID).
		Group("jp.group").
		Scan(&groupDist)

	// Get contest history
	type ContestHistory struct {
		ContestKey  string    `json:"contest_key"`
		ContestName string    `json:"contest_name"`
		Score       float64   `json:"score"`
		Rank        int64     `json:"rank"`
		Date        time.Time `json:"date"`
	}
	var contestHistory []ContestHistory
	db.DB.Table("judge_contestparticipation").
		Select("jc.key as contest_key, jc.name as contest_name, judge_contestparticipation.score, judge_contestparticipation.cumtime, jc.start_time as date").
		Joins("JOIN judge_contest jc ON jc.id = judge_contestparticipation.contest_id").
		Where("judge_contestparticipation.user_id = ? AND judge_contestparticipation.virtual = 0", profile.ID).
		Order("jc.start_time DESC").
		Scan(&contestHistory)

	// Calculate streak (consecutive days with submissions)
	var streakDays int64
	db.DB.Raw(`
		SELECT COUNT(DISTINCT DATE(date))
		FROM judge_submission
		WHERE user_id = ?
		AND date >= DATE_SUB(NOW(), INTERVAL 365 DAY)
	`, profile.ID).Scan(&streakDays)

	c.JSON(http.StatusOK, gin.H{
		"username":            profile.User.Username,
		"statistics":          stats,
		"language_stats":      langDist,
		"activity":            activity,
		"group_stats":         groupDist,
		"contest_history":     contestHistory,
		"streak_days":         streakDays,
		"performance_points":  profile.PerformancePoints,
		"contribution_points": profile.ContributionPoints,
	})
}

// UserExportRequest – POST /api/v2/user/export/request
func UserExportRequest(c *gin.Context) {
	userID := c.GetUint("user_id")

	var profile models.Profile
	if err := db.DB.Where("user_id = ?", userID).First(&profile).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}

	// Rate limiting: one export per week
	if profile.DataLastDownloaded != nil {
		weekAgo := time.Now().AddDate(0, 0, -7)
		if profile.DataLastDownloaded.After(weekAgo) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":       "you can only request data export once per week",
				"last_export": profile.DataLastDownloaded,
			})
			return
		}
	}

	// Enqueue the export job
	if err := jobs.EnqueueUserExport(userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to queue export job"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":        "data export request queued, you will be notified when ready",
		"estimated_time": "5-10 minutes",
	})
}

// UserExportStatus – GET /api/v2/user/export/status
func UserExportStatus(c *gin.Context) {
	userID := c.GetUint("user_id")

	var profile models.Profile
	if err := db.DB.Where("user_id = ?", userID).First(&profile).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}

	type ExportStatus struct {
		CanRequest       bool       `json:"can_request"`
		LastExport       *time.Time `json:"last_export,omitempty"`
		DaysUntilRequest int        `json:"days_until_request"`
		ExportReady      bool       `json:"export_ready"`
	}

	status := ExportStatus{
		CanRequest:  true,
		LastExport:  profile.DataLastDownloaded,
		ExportReady: false,
	}

	if profile.DataLastDownloaded != nil {
		weekAgo := time.Now().AddDate(0, 0, -7)
		if profile.DataLastDownloaded.After(weekAgo) {
			status.CanRequest = false
			nextRequest := profile.DataLastDownloaded.AddDate(0, 0, 7)
			daysLeft := int(time.Until(nextRequest).Hours() / 24)
			if daysLeft < 0 {
				daysLeft = 0
			}
			status.DaysUntilRequest = daysLeft
		}
	}

	c.JSON(http.StatusOK, status)
}

// UserExportDownload – GET /api/v2/user/export/download/:exportID
func UserExportDownload(c *gin.Context) {
	userID := c.GetUint("user_id")
	exportID := c.Param("export_id")

	if len(exportID) < 5 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid export ID"})
		return
	}

	expectedPrefix := fmt.Sprintf("%d_", userID)
	if len(exportID) < len(expectedPrefix) || exportID[:len(expectedPrefix)] != expectedPrefix {
		c.JSON(http.StatusForbidden, gin.H{"error": "unauthorized access to export file"})
		return
	}

	filePath, err := jobs.GetExportFilePath(userID, exportID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "export file not found or expired"})
		return
	}

	fileInfo, err := os.Stat(filePath)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "export file not found"})
		return
	}

	if time.Since(fileInfo.ModTime()) > 24*time.Hour {
		os.Remove(filePath)
		c.JSON(http.StatusGone, gin.H{"error": "export file has expired"})
		return
	}

	if err := jobs.ServeExportFile(c.Writer, filePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to serve export file"})
		return
	}
}
