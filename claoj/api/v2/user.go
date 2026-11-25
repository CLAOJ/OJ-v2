package v2

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/CLAOJ/claoj/auth"
	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/jobs"
	"github.com/CLAOJ/claoj/models"
	"github.com/CLAOJ/claoj/scoring"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// UserList – GET /api/v2/users
// @Description Returns a list of all users with their basic statistics.
// @Tags Users
// @Summary List users
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Items per page" default(100)
// @Param sort query string false "Ranking field: points, rating, contribution" default(points)
// @Success 200 {object} map[string]interface{}
// @Router /users [get]
func UserList(c *gin.Context) {
	page, pageSize := parsePagination(c)

	// Rankings are always highest-first — only the ranked field varies.
	// MySQL sorts NULL lowest, so unrated users fall to the end on rating DESC.
	rankFields := map[string]string{
		"points":       "judge_profile.performance_points DESC",
		"rating":       "judge_profile.rating DESC",
		"contribution": "judge_profile.contribution_points DESC",
	}
	orderBy, ok := rankFields[c.DefaultQuery("sort", "points")]
	if !ok {
		orderBy = rankFields["points"]
	}

	q := db.DB.Model(&models.Profile{}).
		Where("judge_profile.is_unlisted = ?", false).
		Joins("JOIN auth_user au ON au.id = judge_profile.user_id")

	var total int64
	if err := q.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	var profiles []models.Profile
	if err := q.Session(&gorm.Session{}).
		Preload("User").
		Order(orderBy).
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&profiles).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	type userListOrg struct {
		ID        uint   `json:"id"`
		ShortName string `json:"short_name"`
	}
	// One extra query for every membership on this page — no per-row lookups.
	type membershipRow struct {
		ProfileID uint
		ID        uint
		ShortName string
	}
	orgsByProfile := map[uint][]userListOrg{}
	if len(profiles) > 0 {
		ids := make([]uint, len(profiles))
		for i, p := range profiles {
			ids[i] = p.ID
		}
		var rows []membershipRow
		if err := db.DB.Table("judge_profile_organizations po").
			Select("po.profile_id, o.id, o.short_name").
			Joins("JOIN judge_organization o ON o.id = po.organization_id").
			Where("po.profile_id IN ?", ids).
			Order("o.id ASC").
			Scan(&rows).Error; err != nil {
			c.JSON(http.StatusInternalServerError, apiError(err.Error()))
			return
		}
		for _, r := range rows {
			orgsByProfile[r.ProfileID] = append(orgsByProfile[r.ProfileID], userListOrg{r.ID, r.ShortName})
		}
	}

	type Item struct {
		Username           string        `json:"username"`
		DisplayName        string        `json:"display_name"`
		Points             float64       `json:"points"`
		PerformancePoints  float64       `json:"performance_points"`
		Rating             *int          `json:"rating"`
		ProblemCount       int           `json:"problem_count"`
		ContributionPoints int           `json:"contribution_points"`
		DisplayRank        string        `json:"display_rank"`
		AvatarURL          string        `json:"avatar_url"`
		Organizations      []userListOrg `json:"organizations"`
	}
	items := make([]Item, len(profiles))
	for i, p := range profiles {
		orgs := orgsByProfile[p.ID]
		if orgs == nil {
			orgs = []userListOrg{}
		}
		items[i] = Item{
			Username:           p.User.Username,
			DisplayName:        p.UsernameDisplayOverride,
			Points:             p.Points,
			PerformancePoints:  p.PerformancePoints,
			Rating:             p.Rating,
			ProblemCount:       p.ProblemCount,
			ContributionPoints: p.ContributionPoints,
			DisplayRank:        p.DisplayRank,
			AvatarURL:          getAvatarURL(&p),
			Organizations:      orgs,
		}
	}
	c.JSON(http.StatusOK, apiListWithTotal(items, total))
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
// One row per solved public problem: the best accepted submission's points,
// date, language and test-case score, in a single join (no N+1).
func UserSolvedProblems(c *gin.Context) {
	username := c.Param("user")
	var profile models.Profile

	if err := db.DB.Joins("JOIN auth_user au ON au.id = judge_profile.user_id").
		Where("au.username = ?", username).First(&profile).Error; err != nil {
		c.JSON(http.StatusNotFound, apiError("user not found"))
		return
	}

	type SolvedRow struct {
		Code       string    `json:"code"`
		Name       string    `json:"name"`
		Points     float64   `json:"points"`
		Date       time.Time `json:"date"`
		Language   string    `json:"language"`
		CasePoints float64   `json:"case_points"`
		CaseTotal  float64   `json:"case_total"`
	}
	var solved []SolvedRow
	// Same visibility rule as the PP calculation: v1's solved list hides
	// organization-private problems and archived submissions.
	if err := db.DB.Raw(`
		SELECT pr.code, pr.name, COALESCE(s.points, 0) AS points, s.date,
		       COALESCE(NULLIF(l.short_name, ''), l.key, '') AS language,
		       s.case_points, s.case_total
		FROM judge_submission s
		JOIN judge_problem pr ON pr.id = s.problem_id
		LEFT JOIN judge_language l ON l.id = s.language_id
		WHERE s.user_id = ? AND s.result = 'AC' AND s.is_archived = 0
		  AND pr.is_public = 1 AND pr.is_organization_private = 0
		  AND s.id = (
		      SELECT s2.id FROM judge_submission s2
		      WHERE s2.user_id = s.user_id AND s2.problem_id = s.problem_id
		        AND s2.result = 'AC' AND s2.is_archived = 0
		      ORDER BY COALESCE(s2.points, 0) DESC, s2.id ASC
		      LIMIT 1
		  )
		ORDER BY s.date DESC, pr.code ASC
	`, profile.ID).Scan(&solved).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

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
	// Scope the rating history to contests the viewer may see, so an
	// organization-private contest's name is not revealed on a participant's
	// public profile.
	ratingVisExpr, ratingVisArgs := auth.VisibleContestFilterFor(c, "jc")
	db.DB.Table("judge_rating").
		Select("judge_rating.last_rated as date, judge_rating.rating, jc.name as contest, jc.key as contest_key").
		Joins("JOIN judge_contest jc ON jc.id = judge_rating.contest_id").
		Where("judge_rating.user_id = ?", profile.ID).
		Where(ratingVisExpr, ratingVisArgs...).
		Order("judge_rating.last_rated ASC").
		Scan(&history)

	c.JSON(http.StatusOK, apiList(history))
}

// UserPPBreakdown – GET /api/v2/user/:user/pp-breakdown
// Uses scoring.CalculatePP, the same calculation that fills the stored
// judge_profile columns, so the breakdown always matches the profile header.
func UserPPBreakdown(c *gin.Context) {
	username := c.Param("user")
	var profile models.Profile

	if err := db.DB.Joins("JOIN auth_user au ON au.id = judge_profile.user_id").
		Where("au.username = ?", username).First(&profile).Error; err != nil {
		c.JSON(http.StatusNotFound, apiError("user not found"))
		return
	}

	pp, err := scoring.CalculatePP(db.DB, profile.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"performance_points": pp.PerformancePoints,
		"problems":           pp.Problems,
		"bonus": gin.H{
			"solved_count": pp.SolvedCount,
			"bonus_points": pp.BonusPoints,
		},
		"weighted_sum": pp.WeightedSum,
		"total":        pp.PerformancePoints,
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

	// Get submission statistics. GORM maps columns to fields by snake_case
	// name, so the field names must match the SQL aliases exactly.
	type SubStats struct {
		TotalSubmissions    int64   `json:"total_submissions"`
		AcceptedSubmissions int64   `json:"accepted_submissions"`
		TotalPoints         float64 `json:"total_points"`
		AveragePoints       float64 `json:"average_points"`
		BestSubmission      float64 `json:"best_submission"`
	}
	var stats SubStats
	if err := db.DB.Table("judge_submission").
		Select("COUNT(*) as total_submissions, SUM(CASE WHEN result = 'AC' THEN 1 ELSE 0 END) as accepted_submissions, SUM(points) as total_points, AVG(points) as average_points, MAX(points) as best_submission").
		Where("user_id = ?", profile.ID).
		Scan(&stats).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

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

	// Get submission activity by date. `activity` keeps the original 30-day
	// sparse shape for existing consumers; `activity_year` is the dense
	// 365-day series (one row per day, zero included) the profile heatmap needs.
	type Activity struct {
		Date  day   `json:"date"`
		Count int64 `json:"count"`
	}
	var activity []Activity
	if err := db.DB.Table("judge_submission").
		Select("DATE(date) as date, COUNT(*) as count").
		Where("user_id = ? AND date >= ?", profile.ID, time.Now().AddDate(0, 0, -30)).
		Group("DATE(date)").
		Order("date ASC").
		Scan(&activity).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	// The past year: today and the 364 days before it. Day keys are computed
	// in UTC so they agree with the DATE() the database groups by. This
	// diverges from v1, which renders the grid in the viewer's timezone (and
	// drifts from its own UTC cache around midnight).
	today := time.Now().UTC()
	yearStart := today.AddDate(0, 0, -364)
	var yearCounts []Activity
	if err := db.DB.Table("judge_submission").
		Select("DATE(date) as date, COUNT(*) as count").
		Where("user_id = ? AND date >= ?", profile.ID, yearStart).
		Group("DATE(date)").
		Order("date ASC").
		Scan(&yearCounts).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	countsByDate := make(map[string]int64, len(yearCounts))
	for _, row := range yearCounts {
		countsByDate[dayKey(string(row.Date))] = row.Count
	}
	activityYear := make([]Activity, 0, 365)
	for d := yearStart; !d.After(today); d = d.AddDate(0, 0, 1) {
		key := d.Format("2006-01-02")
		activityYear = append(activityYear, Activity{Date: day(key), Count: countsByDate[key]})
	}

	// Get problem group distribution
	type GroupDist struct {
		Group  string  `json:"group"`
		Solved int64   `json:"solved"`
		Points float64 `json:"points"`
	}
	var groupDist []GroupDist
	db.DB.Table("judge_submission").
		Select("pg.full_name as `group`, COUNT(DISTINCT judge_submission.problem_id) as solved, SUM(judge_submission.points) as points").
		Joins("JOIN judge_problem jp ON jp.id = judge_submission.problem_id").
		Joins("JOIN judge_problemgroup pg ON pg.id = jp.group_id").
		Where("judge_submission.user_id = ? AND judge_submission.result = 'AC'", profile.ID).
		Group("pg.full_name").
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
	// Scope the contest history to contests the viewer may see (organization-
	// private contests must not surface on a participant's public profile).
	histVisExpr, histVisArgs := auth.VisibleContestFilterFor(c, "jc")
	db.DB.Table("judge_contestparticipation").
		Select("jc.key as contest_key, jc.name as contest_name, judge_contestparticipation.score, judge_contestparticipation.cumtime, jc.start_time as date").
		Joins("JOIN judge_contest jc ON jc.id = judge_contestparticipation.contest_id").
		Where("judge_contestparticipation.user_id = ? AND judge_contestparticipation.virtual = 0", profile.ID).
		Where(histVisExpr, histVisArgs...).
		Order("jc.start_time DESC").
		Scan(&contestHistory)

	// Calculate streak (distinct days with submissions in the past year)
	var streakDays int64
	db.DB.Raw(`
		SELECT COUNT(DISTINCT DATE(date))
		FROM judge_submission
		WHERE user_id = ?
		AND date >= ?
	`, profile.ID, time.Now().AddDate(0, 0, -365)).Scan(&streakDays)

	c.JSON(http.StatusOK, gin.H{
		"username":            profile.User.Username,
		"statistics":          stats,
		"language_stats":      langDist,
		"activity":            activity,
		"activity_year":       activityYear,
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
