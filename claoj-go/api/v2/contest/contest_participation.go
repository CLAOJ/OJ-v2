package contest

import (
	"net/http"
	"time"

	"github.com/CLAOJ/claoj-go/db"
	"github.com/CLAOJ/claoj-go/models"
	"github.com/gin-gonic/gin"
)

// ParticipationList – GET /api/v2/participations
// Supports ?contest=key filter
func ParticipationList(c *gin.Context) {
	page, pageSize := parsePagination(c)
	contestKey := c.Query("contest")

	q := db.DB.
		Joins("JOIN judge_contest ON judge_contest.id = judge_contestparticipation.contest_id").
		Joins("JOIN judge_profile ON judge_profile.id = judge_contestparticipation.user_id").
		Joins("JOIN auth_user ON auth_user.id = judge_profile.user_id").
		Where("judge_contest.is_visible = ?", true).
		Where("judge_contestparticipation.virtual = 0"). // live participants only
		Select("judge_contestparticipation.*, judge_contest.`key` as contest_key, auth_user.username").
		Order("judge_contestparticipation.score DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize)

	if contestKey != "" {
		q = q.Where("judge_contest.`key` = ?", contestKey)
	}

	type Row struct {
		ContestKey     string  `json:"contest"`
		Username       string  `json:"user"`
		Score          float64 `json:"score"`
		Cumtime        uint    `json:"cumtime"`
		IsDisqualified bool    `json:"is_disqualified"`
	}

	var rows []struct {
		models.ContestParticipation
		ContestKey string
		Username   string
	}
	if err := q.Find(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	items := make([]Row, len(rows))
	for i, r := range rows {
		items[i] = Row{r.ContestKey, r.Username, r.Score, r.Cumtime, r.IsDisqualified}
	}
	c.JSON(http.StatusOK, apiList(items))
}

// UserParticipationList – GET /api/v2/user/contests
// Gets the current user's contest participations (including virtual)
func UserParticipationList(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, apiError("unauthorized"))
		return
	}

	var participations []models.ContestParticipation
	if err := db.DB.
		Preload("Contest").
		Where("user_id = ?", userID).
		Order("id DESC").
		Find(&participations).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	type Item struct {
		ContestKey     string    `json:"contest_key"`
		ContestName    string    `json:"contest_name"`
		StartTime      time.Time `json:"start_time"`
		EndTime        time.Time `json:"end_time"`
		Score          float64   `json:"score"`
		Cumtime        uint      `json:"cumtime"`
		RealStart      time.Time `json:"real_start"`
		IsVirtual      bool      `json:"is_virtual"`
		IsDisqualified bool      `json:"is_disqualified"`
	}

	items := make([]Item, len(participations))
	for i, p := range participations {
		items[i] = Item{
			ContestKey:     p.Contest.Key,
			ContestName:    p.Contest.Name,
			StartTime:      p.Contest.StartTime,
			EndTime:        p.Contest.EndTime,
			Score:          p.Score,
			Cumtime:        p.Cumtime,
			RealStart:      p.RealStart,
			IsVirtual:      p.Virtual > 0,
			IsDisqualified: p.IsDisqualified,
		}
	}

	c.JSON(http.StatusOK, apiList(items))
}
