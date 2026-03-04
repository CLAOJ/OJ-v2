package v2

import (
	"net/http"
	"time"

	"github.com/CLAOJ/claoj-go/db"
	"github.com/CLAOJ/claoj-go/models"
	"github.com/gin-gonic/gin"
)

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

	type Item struct {
		Key       string    `json:"key"`
		Name      string    `json:"name"`
		StartTime time.Time `json:"start_time"`
		EndTime   time.Time `json:"end_time"`
		IsRated   bool      `json:"is_rated"`
		Format    string    `json:"format"`
		UserCount int       `json:"user_count"`
		IsJoined  bool      `json:"is_joined"`
	}
	items := make([]Item, len(contests))
	for i, ct := range contests {
		items[i] = Item{
			ct.Key,
			ct.Name,
			ct.StartTime,
			ct.EndTime,
			ct.IsRated,
			ct.FormatName,
			ct.UserCount,
			joinedKeys[ct.Key],
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
		"key":           ct.Key,
		"name":          ct.Name,
		"summary":       ct.Summary,
		"start_time":    ct.StartTime,
		"end_time":      ct.EndTime,
		"time_limit":    ct.TimeLimit,
		"is_rated":      ct.IsRated,
		"format":        ct.FormatName,
		"problems":      problems,
		"is_joined":     isJoined,
		"is_virtual":    isVirtual,
		"is_active":     isActive,
		"has_ended":     hasEnded,
		"can_virtual":   canVirtual,
	})
}

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
		ContestKey   string    `json:"contest_key"`
		ContestName  string    `json:"contest_name"`
		StartTime    time.Time `json:"start_time"`
		EndTime      time.Time `json:"end_time"`
		Score        float64   `json:"score"`
		Cumtime      uint      `json:"cumtime"`
		RealStart    time.Time `json:"real_start"`
		IsVirtual    bool      `json:"is_virtual"`
		IsDisqualified bool    `json:"is_disqualified"`
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

// ContestClarificationList – GET /api/v2/contest/:key/clarifications
func ContestClarificationList(c *gin.Context) {
	contestKey := c.Param("key")

	var contest models.Contest
	if err := db.DB.Where("`key` = ?", contestKey).First(&contest).Error; err != nil {
		c.JSON(http.StatusNotFound, apiError("contest not found"))
		return
	}

	var clarifications []models.ContestClarification
	if err := db.DB.
		Preload("Author.User").
		Where("contest_id = ?", contest.ID).
		Order("is_answered ASC, create_time DESC").
		Find(&clarifications).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	type Item struct {
		ID         uint      `json:"id"`
		Question   string    `json:"question"`
		Answer     *string   `json:"answer"`
		CreateTime time.Time `json:"create_time"`
		IsAnswered bool      `json:"is_answered"`
		Author     string    `json:"author"`
	}

	items := make([]Item, len(clarifications))
	for i, cl := range clarifications {
		items[i] = Item{
			ID:         cl.ID,
			Question:   cl.Question,
			Answer:     cl.Answer,
			CreateTime: cl.CreateTime,
			IsAnswered: cl.IsAnswered,
			Author:     cl.Author.User.Username,
		}
	}
	c.JSON(http.StatusOK, apiList(items))
}

// ContestClarificationCreateRequest
type ContestClarificationCreateRequest struct {
	Question string `json:"question" binding:"required"`
}

// ContestClarificationCreate – POST /api/v2/contest/:key/clarifications
func ContestClarificationCreate(c *gin.Context) {
	_, profile, ok := resolveUserProfile(c)
	if !ok {
		return
	}

	contestKey := c.Param("key")

	var contest models.Contest
	if err := db.DB.Where("`key` = ? AND is_visible = ?", contestKey, true).First(&contest).Error; err != nil {
		c.JSON(http.StatusNotFound, apiError("contest not found"))
		return
	}

	// Check if contest is active
	now := time.Now()
	if now.Before(contest.StartTime) || now.After(contest.EndTime) {
		c.JSON(http.StatusBadRequest, apiError("contest is not active"))
		return
	}

	var req ContestClarificationCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	clarification := models.ContestClarification{
		ContestID:  contest.ID,
		Question:   req.Question,
		Answer:     nil,
		CreateTime: now,
		IsAnswered: false,
		IsInlined:  false,
		AuthorID:   profile.ID,
	}

	if err := db.DB.Create(&clarification).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create clarification"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":       "clarification submitted",
		"clarification": clarification.ID,
	})
}

// ContestClarificationAnswerRequest
type ContestClarificationAnswerRequest struct {
	Answer   string `json:"answer" binding:"required"`
	IsPublic bool   `json:"is_public"`
}

// ContestClarificationAnswer – POST /api/v2/contest/:key/clarification/:id/answer
// Admin-only endpoint to answer a clarification
func ContestClarificationAnswer(c *gin.Context) {
	// Check if user is admin/staff
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, apiError("unauthorized"))
		return
	}

	var user models.AuthUser
	if err := db.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusUnauthorized, apiError("unauthorized"))
		return
	}

	if !user.IsStaff && !user.IsSuperuser {
		c.JSON(http.StatusForbidden, apiError("admin access required"))
		return
	}

	contestKey := c.Param("key")
	clarificationID := c.Param("id")
	var clarID uint
	if err := parseUint(clarificationID, &clarID); err != nil {
		c.JSON(http.StatusBadRequest, apiError("invalid clarification id"))
		return
	}

	var contest models.Contest
	if err := db.DB.Where("`key` = ?", contestKey).First(&contest).Error; err != nil {
		c.JSON(http.StatusNotFound, apiError("contest not found"))
		return
	}

	var clarification models.ContestClarification
	if err := db.DB.Where("id = ? AND contest_id = ?", clarID, contest.ID).First(&clarification).Error; err != nil {
		c.JSON(http.StatusNotFound, apiError("clarification not found"))
		return
	}

	var req ContestClarificationAnswerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	answer := req.Answer
	db.DB.Model(&clarification).Updates(map[string]interface{}{
		"answer":      &answer,
		"is_answered": true,
		"is_inlined":  !req.IsPublic,
	})

	c.JSON(http.StatusOK, gin.H{
		"message": "clarification answered",
	})
}
