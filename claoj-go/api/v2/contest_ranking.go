package v2

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/CLAOJ/claoj-go/contest_format"
	"github.com/CLAOJ/claoj-go/db"
	"github.com/CLAOJ/claoj-go/models"
	"github.com/CLAOJ/claoj-go/pdf"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ContestRanking fetches the scoreboard for a given contest.
// GET /api/v2/contest/:key/ranking
func ContestRanking(c *gin.Context) {
	key := c.Param("key")

	var ct models.Contest
	if err := db.DB.Where("`key` = ? AND is_visible = ?", key, true).First(&ct).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "contest not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}

	// Fetch all participations for this contest
	var participations []models.ContestParticipation
	if err := db.DB.Preload("User.User").
		Where("contest_id = ? AND virtual = 0 AND is_disqualified = 0", ct.ID).
		Order("score DESC, cumtime ASC, tiebreaker DESC").
		Find(&participations).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load rankings"})
		return
	}

	// Fetch contest problems for breakdown
	var contestProblems []models.ContestProblem
	db.DB.Where("contest_id = ?", ct.ID).Order("`order` ASC").Find(&contestProblems)

	// Build user ID map for rating lookup
	userIDs := make([]uint, len(participations))
	for i, p := range participations {
		userIDs[i] = p.UserID
	}

	// Fetch ratings for all participants
	ratingsMap := make(map[uint]int)    // user_id -> current rating
	ratingChangeMap := make(map[uint]int) // user_id -> rating change in this contest
	performanceMap := make(map[uint]float64) // user_id -> performance in this contest

	if ct.IsRated && ct.EndTime.Before(time.Now()) {
		// Contest is rated and ended - fetch rating changes
		var contestRatings []models.Rating
		if err := db.DB.Where("contest_id = ?", ct.ID).Find(&contestRatings).Error; err == nil {
			for _, cr := range contestRatings {
				ratingChangeMap[cr.UserID] = cr.RatingVal
				performanceMap[cr.UserID] = cr.Performance
				ratingsMap[cr.UserID] = cr.RatingVal // Current rating after this contest
			}
		}
	} else {
		// Contest not rated or not ended - fetch current ratings
		var profiles []models.Profile
		if err := db.DB.Where("user_id IN ?", userIDs).Find(&profiles).Error; err == nil {
			for _, p := range profiles {
				if p.Rating != nil {
					ratingsMap[p.UserID] = *p.Rating
				} else {
					ratingsMap[p.UserID] = 1200 // Default for new users
				}
			}
		}
	}

	cf := contest_format.GetFormat(ct.FormatName, &ct, ct.FormatConfig)

	type RankingRow struct {
		Username     string        `json:"username"`
		Score        float64       `json:"score"`
		Cumtime      uint          `json:"cumtime"`
		Rank         int           `json:"rank"`
		Rating       *int          `json:"rating,omitempty"`
		RatingChange *int          `json:"rating_change,omitempty"`
		Performance  *float64      `json:"performance,omitempty"`
		Breakdown    []interface{} `json:"breakdown"`
	}

	var rankings []RankingRow
	rankNum := 1
	for i, p := range participations {
		if i > 0 && p.Score == participations[i-1].Score && p.Cumtime == participations[i-1].Cumtime && p.Tiebreaker == participations[i-1].Tiebreaker {
			// keep rank same
		} else {
			rankNum = i + 1
		}

		row := RankingRow{
			Username:  p.User.User.Username,
			Score:     p.Score,
			Cumtime:   p.Cumtime,
			Rank:      rankNum,
			Breakdown: cf.GetProblemBreakdown(&p, contestProblems),
		}

		// Add rating info
		if rating, ok := ratingsMap[p.UserID]; ok {
			row.Rating = &rating
		}

		// Add rating change if contest is rated and ended
		if ct.IsRated && ct.EndTime.Before(time.Now()) {
			if change, ok := ratingChangeMap[p.UserID]; ok {
				row.RatingChange = &change
			}
			if perf, ok := performanceMap[p.UserID]; ok {
				row.Performance = &perf
			}
		}

		rankings = append(rankings, row)
	}

	// Prepare problem headers
	type ProblemHeader struct {
		Label  string `json:"label"`
		Points int    `json:"points"`
	}
	headers := make([]ProblemHeader, len(contestProblems))
	for i := range contestProblems {
		headers[i] = ProblemHeader{
			Label:  cf.GetLabelForProblem(i),
			Points: contestProblems[i].Points,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"contest":  ct.Key,
		"problems": headers,
		"rankings": rankings,
	})
}

// ContestRankingPDF - GET /api/v2/contest/:key/ranking/pdf
// Generates and returns a PDF scoreboard for the contest
func ContestRankingPDF(c *gin.Context) {
	key := c.Param("key")

	var ct models.Contest
	if err := db.DB.Where("`key` = ? AND is_visible = ?", key, true).First(&ct).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "contest not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}

	// Fetch all participations for this contest
	var participations []models.ContestParticipation
	if err := db.DB.Preload("User.User").
		Where("contest_id = ? AND virtual = 0 AND is_disqualified = 0", ct.ID).
		Order("score DESC, cumtime ASC, tiebreaker DESC").
		Find(&participations).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load rankings"})
		return
	}

	// Fetch contest problems for breakdown
	var contestProblems []models.ContestProblem
	db.DB.Where("contest_id = ?", ct.ID).Order("`order` ASC").Find(&contestProblems)

	cf := contest_format.GetFormat(ct.FormatName, &ct, ct.FormatConfig)

	// Build problem headers
	type LocalProblemHeader struct {
		Label  string
		Points int
	}
	headers := make([]LocalProblemHeader, len(contestProblems))
	for i := range contestProblems {
		headers[i] = LocalProblemHeader{
			Label:  cf.GetLabelForProblem(i),
			Points: contestProblems[i].Points,
		}
	}

	// Build ranking rows
	type LocalBreakdownItem struct {
		Points  float64
		Penalty uint
		Solved  bool
	}
	type LocalRankingRow struct {
		Username  string
		Score     float64
		Cumtime   uint
		Rank      int
		Breakdown []LocalBreakdownItem
	}

	var rankings []LocalRankingRow
	rankNum := 1
	for i, p := range participations {
		if i > 0 && p.Score == participations[i-1].Score && p.Cumtime == participations[i-1].Cumtime && p.Tiebreaker == participations[i-1].Tiebreaker {
			// keep rank same
		} else {
			rankNum = i + 1
		}

		// Convert breakdown to our local type
		breakdowns := make([]LocalBreakdownItem, 0)
		rawBreakdown := cf.GetProblemBreakdown(&p, contestProblems)
		for _, bd := range rawBreakdown {
			// Convert interface{} to BreakdownItem
			// The contest_format package returns map[string]interface{}
			if bdMap, ok := bd.(map[string]interface{}); ok {
				points := bdMap["points"]
				solved := bdMap["solved"]
				penalty := bdMap["penalty"]

				bdItem := LocalBreakdownItem{
					Points:  0,
					Penalty: 0,
					Solved:  false,
				}

				if pts, ok := points.(float64); ok {
					bdItem.Points = pts
				}
				if pen, ok := penalty.(float64); ok {
					bdItem.Penalty = uint(pen)
				}
				if s, ok := solved.(bool); ok {
					bdItem.Solved = s
				}

				breakdowns = append(breakdowns, bdItem)
			}
		}

		rankings = append(rankings, LocalRankingRow{
			Username:  p.User.User.Username,
			Score:     p.Score,
			Cumtime:   p.Cumtime,
			Rank:      rankNum,
			Breakdown: breakdowns,
		})
	}

	// Convert to pdf package types
	pdfHeaders := make([]pdf.ProblemHeader, len(headers))
	for i, h := range headers {
		pdfHeaders[i] = pdf.ProblemHeader{Label: h.Label, Points: h.Points}
	}

	pdfRankings := make([]pdf.RankingRow, len(rankings))
	for i, r := range rankings {
		pdfBreakdown := make([]pdf.BreakdownItem, len(r.Breakdown))
		for j, bd := range r.Breakdown {
			pdfBreakdown[j] = pdf.BreakdownItem{
				Points:  bd.Points,
				Penalty: bd.Penalty,
				Solved:  bd.Solved,
			}
		}
		pdfRankings[i] = pdf.RankingRow{
			Username:  r.Username,
			Score:     r.Score,
			Cumtime:   r.Cumtime,
			Rank:      r.Rank,
			Breakdown: pdfBreakdown,
		}
	}

	// Generate PDF
	pdfData := pdf.NewContestScoreboardData(&ct, pdfHeaders, pdfRankings)
	pdfBytes, err := pdf.GenerateContestScoreboard(pdfData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to generate PDF: %v", err)})
		return
	}

	// Return PDF as download
	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=contest_%s_scoreboard.pdf", key))
	c.Data(http.StatusOK, "application/pdf", pdfBytes)
}
