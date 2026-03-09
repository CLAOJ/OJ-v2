package contest

import (
	"fmt"
	"net/http"
	"time"

	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/gin-gonic/gin"
)

// apiError creates a standard error response
func apiError(msg string) gin.H {
	return gin.H{"error": msg}
}

// apiList creates a standard list response
func apiList(items interface{}) gin.H {
	return gin.H{"results": items}
}

// parsePagination extracts page and pageSize from query params
func parsePagination(c *gin.Context) (page, pageSize int) {
	page = 1
	pageSize = 50

	if p := c.Query("page"); p != "" {
		if val, err := parseInt(p); err == nil && val > 0 {
			page = val
		}
	}
	if ps := c.Query("page_size"); ps != "" {
		if val, err := parseInt(ps); err == nil && val > 0 && val <= 100 {
			pageSize = val
		}
	}
	return
}

func parseInt(s string) (int, error) {
	var n int
	_, err := fmt.Sscanf(s, "%d", &n)
	return n, err
}

// ContestList – GET /api/v2/contests
func ContestList(c *gin.Context) {
	page, pageSize := parsePagination(c)
	search := c.Query("search")

	var contests []models.Contest
	// Use raw SQL to properly handle 'key' reserved word in MariaDB
	sql := "SELECT * FROM judge_contest WHERE is_visible = ?"
	args := []interface{}{true}

	if search != "" {
		sql += " AND (name LIKE ? OR `key` LIKE ?)"
		args = append(args, "%"+search+"%", "%"+search+"%")
	}

	sql += " ORDER BY start_time DESC LIMIT ? OFFSET ?"
	args = append(args, pageSize, (page-1)*pageSize)

	if err := db.DB.Raw(sql, args...).Scan(&contests).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	// Load tags for all contests
	contestIDs := make([]uint, len(contests))
	for i, ct := range contests {
		contestIDs[i] = ct.ID
	}
	contestTagsMap := make(map[uint][]models.ContestTag)
	if len(contestIDs) > 0 {
		var contestTags []struct {
			ContestID uint              `gorm:"column:contest_id"`
			Tag       models.ContestTag `gorm:"embedded"`
		}
		db.DB.Table("judge_contest_tags").
			Joins("JOIN judge_contesttag ON judge_contesttag.id = judge_contest_tags.contesttag_id").
			Where("judge_contest_tags.contest_id IN ?", contestIDs).
			Select("judge_contest_tags.contest_id, judge_contesttag.*").
			Scan(&contestTags)
		for _, ct := range contestTags {
			contestTagsMap[ct.ContestID] = append(contestTagsMap[ct.ContestID], ct.Tag)
		}
	}

	// Get joined contests for the current user
	joinedKeys := make(map[string]bool)
	if uid, exists := c.Get("user_id"); exists {
		userID := uid.(uint)
		var joined []string
		db.DB.Table("judge_contestparticipation").
			Joins("JOIN judge_contest ON judge_contest.id = judge_contestparticipation.contest_id").
			Where("judge_contestparticipation.user_id = ?", userID).
			Pluck("judge_contest.`key`", &joined)
		for _, key := range joined {
			joinedKeys[key] = true
		}
	}

	type Tag struct {
		ID    uint   `json:"id"`
		Name  string `json:"name"`
		Color string `json:"color"`
	}

	type Item struct {
		Key       string    `json:"key"`
		Name      string    `json:"name"`
		StartTime time.Time `json:"start_time"`
		EndTime   time.Time `json:"end_time"`
		IsRated   bool      `json:"is_rated"`
		Format    string    `json:"format"`
		UserCount int       `json:"user_count"`
		IsJoined  bool      `json:"is_joined"`
		Tags      []Tag     `json:"tags"`
	}
	items := make([]Item, len(contests))
	for i, ct := range contests {
		tags := make([]Tag, len(contestTagsMap[ct.ID]))
		for j, tag := range contestTagsMap[ct.ID] {
			tags[j] = Tag{ID: tag.ID, Name: tag.Name, Color: tag.Color}
		}
		items[i] = Item{
			ct.Key,
			ct.Name,
			ct.StartTime,
			ct.EndTime,
			ct.IsRated,
			ct.FormatName,
			ct.UserCount,
			joinedKeys[ct.Key],
			tags,
		}
	}
	c.JSON(http.StatusOK, apiList(items))
}

// ContestDetail – GET /api/v2/contest/:key
func ContestDetail(c *gin.Context) {
	key := c.Param("key")
	var ct models.Contest
	if err := db.DB.
		Preload("ContestProblems.Problem").
		Where("`key` = ? AND is_visible = ?", key, true).
		First(&ct).Error; err != nil {
		c.JSON(http.StatusNotFound, apiError("contest not found"))
		return
	}

	now := time.Now()
	isActive := now.After(ct.StartTime) && now.Before(ct.EndTime)
	hasEnded := now.After(ct.EndTime)

	// Get user participation and solve status
	isJoined := false
	isVirtual := false
	canVirtual := hasEnded // Can only do virtual participation after contest ends
	solvedCodes := make(map[string]bool)
	var partID uint
	if uid, exists := c.Get("user_id"); exists {
		userID := uid.(uint)
		db.DB.Table("judge_contestparticipation").
			Where("contest_id = ? AND user_id = ?", ct.ID, userID).
			Pluck("id", &partID)
		if partID > 0 {
			isJoined = true
			// Check if this is a virtual participation
			var part models.ContestParticipation
			if err := db.DB.Where("id = ?", partID).First(&part).Error; err == nil {
				isVirtual = part.Virtual > 0
			}
			var codes []string
			db.DB.Table("judge_contestsubmission").
				Joins("JOIN judge_submission ON judge_submission.id = judge_contestsubmission.submission_id").
				Joins("JOIN judge_problem ON judge_problem.id = judge_contestsubmission.problem_id").
				Where("judge_contestsubmission.participation_id = ? AND judge_submission.result = 'AC'", partID).
				Pluck("judge_problem.code", &codes)
			for _, code := range codes {
				solvedCodes[code] = true
			}
		}
	}

	type ProblemItem struct {
		Code     string  `json:"code"`
		Name     string  `json:"name"`
		Points   int     `json:"points"`
		Order    uint    `json:"order"`
		AcRate   float64 `json:"ac_rate"`
		IsSolved bool    `json:"is_solved"`
	}
	problems := make([]ProblemItem, len(ct.ContestProblems))
	for i, cp := range ct.ContestProblems {
		problems[i] = ProblemItem{
			Code:     cp.Problem.Code,
			Name:     cp.Problem.Name,
			Points:   cp.Points,
			Order:    cp.Order,
			AcRate:   cp.Problem.AcRate,
			IsSolved: solvedCodes[cp.Problem.Code],
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"key":         ct.Key,
		"name":        ct.Name,
		"summary":     ct.Summary,
		"start_time":  ct.StartTime,
		"end_time":    ct.EndTime,
		"time_limit":  ct.TimeLimit,
		"is_rated":    ct.IsRated,
		"format":      ct.FormatName,
		"problems":    problems,
		"is_joined":   isJoined,
		"is_virtual":  isVirtual,
		"is_active":   isActive,
		"has_ended":   hasEnded,
		"can_virtual": canVirtual,
	})
}
