package v2

import (
	"net/http"
	"strings"
	"time"

	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ============================================================
// ADMIN PROBLEM POINTS VOTES (read-only, v1 ProblemPointsVoteAdmin)
// ============================================================

// AdminPointsVoteList - GET /api/v2/admin/points-votes
// v1's ProblemPointsVoteAdmin is read-only (has_add_permission returns False).
func AdminPointsVoteList(c *gin.Context) {
	page, pageSize := parsePagination(c)

	code := strings.TrimSpace(c.Query("problem"))
	search := strings.TrimSpace(c.Query("search"))

	build := func() *gorm.DB {
		query := db.DB.Model(&models.ProblemPointsVote{})
		if code != "" {
			query = query.Where("problem_id IN (SELECT id FROM judge_problem WHERE code = ?)", code)
		}
		if search != "" {
			pattern := db.ContainsPattern(search)
			query = query.Where(
				`voter_id IN (SELECT p.id FROM judge_profile p JOIN auth_user u ON u.id = p.user_id
					WHERE u.username LIKE ? ESCAPE '!')
				OR problem_id IN (SELECT id FROM judge_problem
					WHERE code LIKE ? ESCAPE '!' OR name LIKE ? ESCAPE '!')`,
				pattern, pattern, pattern)
		}
		return query
	}

	var total int64
	if err := build().Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	var votes []models.ProblemPointsVote
	if err := build().
		Preload("Voter.User").
		Preload("Problem").
		Order("vote_time DESC, id DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&votes).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	type Item struct {
		ID          uint      `json:"id"`
		Points      int       `json:"points"`
		Note        string    `json:"note"`
		VoteTime    time.Time `json:"vote_time"`
		VoterID     uint      `json:"voter_id"`
		Voter       string    `json:"voter"`
		ProblemID   uint      `json:"problem_id"`
		Problem     string    `json:"problem"`
		ProblemName string    `json:"problem_name"`
	}
	items := make([]Item, len(votes))
	for i, v := range votes {
		items[i] = Item{
			ID:          v.ID,
			Points:      v.Points,
			Note:        v.Note,
			VoteTime:    v.VoteTime,
			VoterID:     v.VoterID,
			Voter:       v.Voter.User.Username,
			ProblemID:   v.ProblemID,
			Problem:     v.Problem.Code,
			ProblemName: v.Problem.Name,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"data":      items,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}