package v2

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/CLAOJ/claoj/auth"
	"github.com/CLAOJ/claoj/contest_format"
	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/CLAOJ/claoj/pdf"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ContestRanking fetches the scoreboard for a given contest.
// GET /api/v2/contest/:key/ranking
func ContestRanking(c *gin.Context) {
	key := c.Param("key")

	var ct models.Contest
	if err := db.DB.Where("`key` = ?", key).First(&ct).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "contest not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}
	// Restrict scoreboards to contests the user may see (organization-private
	// contests must not leak via rankings).
	if !auth.CanViewContest(c, &ct) {
		c.JSON(http.StatusNotFound, gin.H{"error": "contest not found"})
		return
	}

	// Fetch all participations for this contest.
	// Usernames are resolved with an explicit JOIN below rather than
	// Preload("User.User"), which does not populate the nested auth_user and
	// left every "username" blank.
	// include_virtual mirrors v1's show_virtual switch: off shows live
	// participants only, on adds virtual ones (spectators, virtual = -1, are
	// never ranked).
	includeVirtual := c.Query("include_virtual") == "true" || c.Query("include_virtual") == "1"
	virtualCond := "virtual = 0"
	if includeVirtual {
		virtualCond = "virtual >= 0"
	}
	var participations []models.ContestParticipation
	if err := db.DB.
		Where("contest_id = ? AND "+virtualCond+" AND is_disqualified = 0", ct.ID).
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

	// Resolve username, display name and primary organization for each
	// participation's profile id. One JOIN over the profiles, their auth_user
	// row and their (listed) organizations keeps this free of N+1 queries;
	// v1 shows the first listed organization in the scoreboard.
	type RankingOrg struct {
		ID        uint   `json:"id"`
		Name      string `json:"name"`
		ShortName string `json:"short_name"`
	}
	type profileMeta struct {
		ProfileID   uint    `gorm:"column:profile_id"`
		Username    string  `gorm:"column:username"`
		DisplayName string  `gorm:"column:display_name"`
		OrgID       *uint   `gorm:"column:org_id"`
		OrgName     *string `gorm:"column:org_name"`
		OrgShort    *string `gorm:"column:org_short_name"`
	}
	usernameByProfile := make(map[uint]string, len(userIDs))
	displayNameByProfile := make(map[uint]string, len(userIDs))
	orgByProfile := make(map[uint]RankingOrg, len(userIDs))
	if len(userIDs) > 0 {
		var metaRows []profileMeta
		db.DB.Table("judge_profile").
			Select("judge_profile.id AS profile_id, auth_user.username AS username, "+
				"judge_profile.username_display_override AS display_name, "+
				"o.id AS org_id, o.name AS org_name, o.short_name AS org_short_name").
			Joins("JOIN auth_user ON auth_user.id = judge_profile.user_id").
			Joins("LEFT JOIN judge_profile_organizations po ON po.profile_id = judge_profile.id").
			Joins("LEFT JOIN judge_organization o ON o.id = po.organization_id AND o.is_unlisted = 0").
			Where("judge_profile.id IN ?", userIDs).
			Order("judge_profile.id ASC, o.id ASC").
			Scan(&metaRows)
		for _, r := range metaRows {
			usernameByProfile[r.ProfileID] = r.Username
			displayNameByProfile[r.ProfileID] = r.DisplayName
			if _, seen := orgByProfile[r.ProfileID]; !seen && r.OrgID != nil && r.OrgName != nil && r.OrgShort != nil {
				orgByProfile[r.ProfileID] = RankingOrg{ID: *r.OrgID, Name: *r.OrgName, ShortName: *r.OrgShort}
			}
		}
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
		// Contest not rated or not ended - fetch current ratings.
		// userIDs holds judge_profile ids (participation.user_id is a profile
		// FK), so match on judge_profile.id and key the map by profile id.
		var profiles []models.Profile
		if err := db.DB.Where("id IN ?", userIDs).Find(&profiles).Error; err == nil {
			for _, p := range profiles {
				if p.Rating != nil {
					ratingsMap[p.ID] = *p.Rating
				} else {
					ratingsMap[p.ID] = 1200 // Default for new users
				}
			}
		}
	}

	cf := contest_format.GetFormat(ct.FormatName, &ct, ct.FormatConfig)

	type RankingRow struct {
		Username        string        `json:"username"`
		DisplayName     string        `json:"display_name"`
		Organization    *RankingOrg   `json:"organization"`
		Score           float64       `json:"score"`
		Cumtime         uint          `json:"cumtime"`
		Rank            int           `json:"rank"`
		Rating          *int          `json:"rating,omitempty"`
		RatingChange    *int          `json:"rating_change,omitempty"`
		Performance     *float64      `json:"performance,omitempty"`
		Breakdown       []interface{} `json:"breakdown"`
		IsDisqualified  bool          `json:"is_disqualified"`
		IsVirtual       bool          `json:"is_virtual"`
		ParticipationID uint          `json:"participation_id"`
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
			Username:        usernameByProfile[p.UserID],
			DisplayName:     displayNameByProfile[p.UserID],
			Score:           p.Score,
			Cumtime:         p.Cumtime,
			Rank:            rankNum,
			Breakdown:       cf.GetProblemBreakdown(&p, contestProblems),
			IsDisqualified:  p.IsDisqualified,
			IsVirtual:       p.Virtual > 0,
			ParticipationID: p.ID,
		}
		if org, ok := orgByProfile[p.UserID]; ok {
			row.Organization = &org
		}
		// v1: display_name falls back to the username when no override is set.
		if row.DisplayName == "" {
			row.DisplayName = row.Username
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
	if err := db.DB.Where("`key` = ?", key).First(&ct).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "contest not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}
	// Restrict scoreboards to contests the user may see (organization-private
	// contests must not leak via rankings).
	if !auth.CanViewContest(c, &ct) {
		c.JSON(http.StatusNotFound, gin.H{"error": "contest not found"})
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
