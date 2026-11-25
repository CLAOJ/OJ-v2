package contest

import (
	"net/http"
	"strings"
	"time"

	"github.com/CLAOJ/claoj/auth"
	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ParticipationList – GET /api/v2/contest/:key/participations
// Supports ?include_virtual=true filters. Returns one row per participation
// with the contest key, username/display name, score, cumulative time and the
// disqualification/virtual flags, plus the filtered row count so the client can
// page. ?contest= is still accepted as a fallback for the pre-route URL.
func ParticipationList(c *gin.Context) {
	type Row struct {
		ContestKey     string  `json:"contest"`
		Username       string  `json:"user"`
		DisplayName    string  `json:"display_name"`
		Score          float64 `json:"score"`
		Cumtime        uint    `json:"cumtime"`
		IsDisqualified bool    `json:"is_disqualified"`
		IsVirtual      bool    `json:"is_virtual"`
	}

	page, pageSize := parsePagination(c)
	contestKey := c.Param("key")
	if contestKey == "" {
		contestKey = c.Query("contest")
	}
	includeVirtual := c.Query("include_virtual") == "true" || c.Query("include_virtual") == "1"

	var ct models.Contest
	if contestKey != "" {
		if err := db.DB.Where("`key` = ?", contestKey).First(&ct).Error; err != nil {
			c.JSON(http.StatusNotFound, apiError("contest not found"))
			return
		}
		// Organization-private contests must not leak their participants.
		if !auth.CanViewContest(c, &ct) {
			c.JSON(http.StatusNotFound, apiError("contest not found"))
			return
		}
		// v1 hides the whole scoreboard (and its participant list) while
		// scoreboard_visibility says so.
		if !auth.CanSeeContestScoreboard(c, &ct) {
			c.JSON(http.StatusOK, gin.H{"results": []Row{}, "total": 0, "scoreboard_hidden": true})
			return
		}
	}

	// Only expose participations of contests the requesting user may see, so
	// organization-private contests do not leak via their participant lists.
	visExpr, visArgs := auth.VisibleContestFilter(c)
	virtualCond := "judge_contestparticipation.virtual = 0"
	if includeVirtual {
		// virtual >= 0: live participants plus virtual ones; spectate (-1) stays
		// hidden (v1: users.filter(virtual__gte=0)).
		virtualCond = "judge_contestparticipation.virtual >= 0"
	}
	base := db.DB.
		Table("judge_contestparticipation").
		Joins("JOIN judge_contest ON judge_contest.id = judge_contestparticipation.contest_id").
		Joins("JOIN judge_profile ON judge_profile.id = judge_contestparticipation.user_id").
		Joins("JOIN auth_user ON auth_user.id = judge_profile.user_id").
		Where(visExpr, visArgs...).
		Where(virtualCond)
	if contestKey != "" {
		base = base.Where("judge_contest.`key` = ?", contestKey)
	}
	// v1 /contest/<c>/participations/<user>: one competitor's rows. The
	// scoreboard gate above already limited the request to what the viewer may
	// see, so a hidden scoreboard returns no rows here either.
	if user := strings.TrimSpace(c.Query("user")); user != "" {
		base = base.Where("auth_user.username = ?", user)
	}

	var total int64
	if err := base.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	var rows []struct {
		models.ContestParticipation
		ContestKey   string
		Username     string
		DisplayName  string
	}
	if err := base.Session(&gorm.Session{}).
		Select("judge_contestparticipation.*, judge_contest.`key` as contest_key, " +
			"auth_user.username AS username, judge_profile.username_display_override AS display_name").
		Order("judge_contestparticipation.score DESC, judge_contestparticipation.cumtime ASC, judge_contestparticipation.id ASC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	items := make([]Row, len(rows))
	for i, r := range rows {
		displayName := r.DisplayName
		if displayName == "" {
			displayName = r.Username
		}
		items[i] = Row{r.ContestKey, r.Username, displayName, r.Score, r.Cumtime, r.IsDisqualified, r.Virtual > 0}
	}
	c.JSON(http.StatusOK, gin.H{"results": items, "total": total})
}

// UserParticipationList – GET /api/v2/user/contests
// Gets the current user's contest participations (including virtual)
func UserParticipationList(c *gin.Context) {
	if _, exists := c.Get("user_id"); !exists {
		c.JSON(http.StatusUnauthorized, apiError("unauthorized"))
		return
	}
	// judge_contestparticipation.user_id is a judge_profile.id FK
	profileID, ok := auth.CurrentProfileID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, apiError("unauthorized"))
		return
	}

	var participations []models.ContestParticipation
	if err := db.DB.
		Preload("Contest").
		Where("user_id = ?", profileID).
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
