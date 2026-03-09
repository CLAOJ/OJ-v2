package contest

import (
	"fmt"
	"net/http"
	"time"

	"github.com/CLAOJ/claoj/contest_format"
	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/gin-gonic/gin"
)

// ContestStats - GET /api/v2/contest/:key/stats
// Returns statistics for the current user's contest performance (personal stats)
func ContestStats(c *gin.Context) {
	key := c.Param("key")

	var ct models.Contest
	if err := db.DB.Where("`key` = ? AND is_visible = ?", key, true).First(&ct).Error; err != nil {
		c.JSON(http.StatusNotFound, apiError("contest not found"))
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, apiError("unauthorized"))
		return
	}

	// Get user's participation
	var participation models.ContestParticipation
	if err := db.DB.Where("contest_id = ? AND user_id = ?", ct.ID, userID).First(&participation).Error; err != nil {
		c.JSON(http.StatusNotFound, apiError("you have not participated in this contest"))
		return
	}

	// Get contest problems
	var contestProblems []models.ContestProblem
	if err := db.DB.Where("contest_id = ?", ct.ID).Order("`order` ASC").Find(&contestProblems).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError("failed to load contest problems"))
		return
	}

	// Get user's submissions in this contest
	type SubmissionStats struct {
		ProblemCode     string     `json:"problem_code"`
		ProblemLabel    string     `json:"problem_label"`
		Points          int        `json:"points"`
		MaxScore        float64    `json:"max_score"`
		IsSolved        bool       `json:"is_solved"`
		AttemptCount    int        `json:"attempt_count"`
		FirstSubmitTime *time.Time `json:"first_submit_time"`
		SolveTime       *uint      `json:"solve_time,omitempty"` // seconds from contest start to AC
		TimeInSeconds   uint       `json:"time_in_seconds"`
	}

	var stats []SubmissionStats
	for _, cp := range contestProblems {
		stat := SubmissionStats{
			ProblemCode:  cp.Problem.Code,
			Points:       cp.Points,
			MaxScore:     0,
			IsSolved:     false,
			AttemptCount: 0,
		}

		// Get all submissions for this problem in the contest
		var submissions []struct {
			SubmissionID uint      `gorm:"column:submission_id"`
			Points       float64   `gorm:"column:points"`
			Date         time.Time `gorm:"column:date"`
		}
		db.DB.Table("judge_contestsubmission").
			Joins("JOIN judge_submission ON judge_submission.id = judge_contestsubmission.submission_id").
			Where("judge_contestsubmission.participation_id = ? AND judge_contestsubmission.problem_id = ?", participation.ID, cp.ID).
			Order("judge_submission.date ASC").
			Select("judge_contestsubmission.submission_id, judge_contestsubmission.points, judge_submission.date").
			Scan(&submissions)

		stat.AttemptCount = len(submissions)
		if stat.AttemptCount > 0 {
			stat.FirstSubmitTime = &submissions[0].Date
			for _, sub := range submissions {
				if sub.Points > stat.MaxScore {
					stat.MaxScore = sub.Points
				}
				// Check if solved (full points or AC)
				if sub.Points >= float64(cp.Points) {
					stat.IsSolved = true
					solveTime := uint(sub.Date.Sub(ct.StartTime).Seconds())
					stat.SolveTime = &solveTime
					stat.TimeInSeconds = solveTime
				}
			}
		}

		// Get problem label
		cf := contest_format.GetFormat(ct.FormatName, &ct, ct.FormatConfig)
		for i, prob := range contestProblems {
			if prob.ID == cp.ID {
				stat.ProblemLabel = cf.GetLabelForProblem(i)
				break
			}
		}

		stats = append(stats, stat)
	}

	// Calculate summary statistics
	totalProblems := len(contestProblems)
	solvedProblems := 0
	totalAttempts := 0
	var totalSolveTime uint = 0
	for _, s := range stats {
		if s.IsSolved {
			solvedProblems++
			totalSolveTime += s.TimeInSeconds
		}
		totalAttempts += s.AttemptCount
	}

	// Get ranking info
	var totalParticipants int64
	db.DB.Model(&models.ContestParticipation{}).
		Where("contest_id = ? AND virtual = 0 AND is_disqualified = 0", ct.ID).
		Count(&totalParticipants)

	var userRank int64
	db.DB.Model(&models.ContestParticipation{}).
		Where("contest_id = ? AND virtual = 0 AND is_disqualified = 0 AND (score > ? OR (score = ? AND cumtime < ?))",
			ct.ID, participation.Score, participation.Score, participation.Cumtime).
		Count(&userRank)
	userRank++ // 1-based rank

	// Calculate percentile
	var percentile float64 = 0
	if totalParticipants > 1 {
		percentile = float64(totalParticipants-userRank) / float64(totalParticipants-1) * 100
	}

	// Get average stats for comparison
	var avgScore float64
	db.DB.Model(&models.ContestParticipation{}).
		Where("contest_id = ? AND virtual = 0 AND is_disqualified = 0", ct.ID).
		Select("AVG(score)").Scan(&avgScore)

	// Get average solve time per problem
	avgSolveTimes := make(map[string]float64)
	for _, cp := range contestProblems {
		var avgTime float64
		db.DB.Table("judge_contestsubmission").
			Joins("JOIN judge_submission ON judge_submission.id = judge_contestsubmission.submission_id").
			Joins("JOIN judge_contestparticipation ON judge_contestparticipation.id = judge_contestsubmission.participation_id").
			Where("judge_contestsubmission.problem_id = ? AND judge_contestparticipation.contest_id = ? AND judge_contestparticipation.virtual = 0 AND judge_contestparticipation.is_disqualified = 0 AND judge_contestsubmission.points >= ?", cp.ID, ct.ID, cp.Points).
			Select("AVG(TIMESTAMPDIFF(SECOND, judge_contest.start_time, judge_submission.date))").
			Scan(&avgTime)
		if avgTime > 0 {
			avgSolveTimes[cp.Problem.Code] = avgTime
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"contest_key":                    ct.Key,
		"contest_name":                   ct.Name,
		"participation_id":               participation.ID,
		"score":                          participation.Score,
		"cumtime":                        participation.Cumtime,
		"rank":                           userRank,
		"total_participants":             totalParticipants,
		"percentile":                     percentile,
		"average_score":                  avgScore,
		"solved_count":                   solvedProblems,
		"total_problems":                 totalProblems,
		"total_attempts":                 totalAttempts,
		"average_solve_time":             totalSolveTime / uint(max(solvedProblems, 1)),
		"problems":                       stats,
		"average_solve_times_by_problem": avgSolveTimes,
	})
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// ContestPublicStats - GET /api/v2/contest/:key/stats/public
// Returns public statistics for a contest (overall stats, not personal)
func ContestPublicStats(c *gin.Context) {
	key := c.Param("key")

	var ct models.Contest
	if err := db.DB.Where("`key` = ? AND (is_visible = ? OR is_public = ?)", key, true, true).First(&ct).Error; err != nil {
		c.JSON(http.StatusNotFound, apiError("contest not found"))
		return
	}

	// Submissions per problem
	type ProblemSubmissionStats struct {
		ProblemCode  string `json:"problem_code"`
		ProblemName  string `json:"problem_name"`
		ProblemLabel string `json:"problem_label"`
		Total        int64  `json:"total"`
		Accepted     int64  `json:"accepted"`
		ACRate       string `json:"ac_rate"`
	}

	var problemStats []ProblemSubmissionStats
	db.DB.Raw(`
		SELECT p.code, p.name, cp.label,
		       COUNT(cs.submission_id) as total,
		       SUM(CASE WHEN cs.points >= cp.points THEN 1 ELSE 0 END) as accepted
		FROM judge_contestproblem cp
		JOIN judge_problem p ON p.id = cp.problem_id
		LEFT JOIN judge_contestsubmission cs ON cs.problem_id = cp.id
		WHERE cp.contest_id = ?
		GROUP BY cp.id, p.code, p.name, cp.label, cp.points
		ORDER BY cp.order
	`, ct.ID).Scan(&problemStats)

	for i := range problemStats {
		if problemStats[i].Total > 0 {
			rate := float64(problemStats[i].Accepted) / float64(problemStats[i].Total) * 100
			problemStats[i].ACRate = fmt.Sprintf("%.1f", rate)
		} else {
			problemStats[i].ACRate = "0.0"
		}
	}

	// Language distribution
	type LanguageStats struct {
		Language string `json:"language"`
		Count    int64  `json:"count"`
	}

	var languageStats []LanguageStats
	db.DB.Raw(`
		SELECT l.name as language, COUNT(cs.submission_id) as count
		FROM judge_contestsubmission cs
		JOIN judge_submission s ON s.id = cs.submission_id
		JOIN judge_language l ON l.id = s.language_id
		JOIN judge_contestparticipation cp ON cp.id = cs.participation_id
		WHERE cp.contest_id = ?
		GROUP BY l.id, l.name
		ORDER BY count DESC
	`, ct.ID).Scan(&languageStats)

	// Participation stats
	var totalParticipants int64
	db.DB.Model(&models.ContestParticipation{}).
		Where("contest_id = ? AND virtual = 0 AND is_disqualified = 0", ct.ID).
		Count(&totalParticipants)

	var totalSubmissions int64
	db.DB.Table("judge_contestsubmission").
		Joins("JOIN judge_contestparticipation cp ON cp.id = judge_contestsubmission.participation_id").
		Where("cp.contest_id = ?", ct.ID).
		Count(&totalSubmissions)

	// Score distribution
	type ScoreDistribution struct {
		ScoreRange string `json:"score_range"`
		Count      int64  `json:"count"`
	}

	var scoreDist []ScoreDistribution
	db.DB.Raw(`
		SELECT
			CASE
				WHEN score = 0 THEN '0'
				WHEN score > 0 AND score < 25 THEN '1-24%'
				WHEN score >= 25 AND score < 50 THEN '25-49%'
				WHEN score >= 50 AND score < 75 THEN '50-74%'
				WHEN score >= 75 AND score < 100 THEN '75-99%'
				WHEN score >= 100 THEN '100%'
			END as score_range,
			COUNT(*) as count
		FROM judge_contestparticipation
		WHERE contest_id = ? AND virtual = 0 AND is_disqualified = 0
		GROUP BY score_range
		ORDER BY MIN(score)
	`, ct.ID).Scan(&scoreDist)

	c.JSON(http.StatusOK, gin.H{
		"contest_key":        ct.Key,
		"contest_name":       ct.Name,
		"total_participants": totalParticipants,
		"total_submissions":  totalSubmissions,
		"problems":           problemStats,
		"languages":          languageStats,
		"score_distribution": scoreDist,
	})
}
