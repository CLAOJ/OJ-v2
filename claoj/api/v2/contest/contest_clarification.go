package contest

import (
	"net/http"
	"time"

	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

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
	var clarID int
	if _, err := parseInt(clarificationID); err != nil {
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

// resolveUserProfile gets the user ID and profile from context
func resolveUserProfile(c *gin.Context) (uint, *models.Profile, bool) {
	userID := c.GetUint("user_id")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, apiError("unauthorized"))
		return 0, nil, false
	}

	var profile models.Profile
	if err := db.DB.Where("user_id = ?", userID).First(&profile).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, apiError("profile not found"))
		} else {
			c.JSON(http.StatusInternalServerError, apiError("database error"))
		}
		return 0, nil, false
	}

	return userID, &profile, true
}
