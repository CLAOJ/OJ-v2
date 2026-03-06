package v2

import (
	"net/http"
	"time"

	"github.com/CLAOJ/claoj-go/db"
	"github.com/CLAOJ/claoj-go/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ProblemClarificationList – GET /api/v2/problem/:code/clarifications
// Public endpoint to list all clarifications for a problem
func ProblemClarificationList(c *gin.Context) {
	code := c.Param("code")

	// Get problem ID
	var problem models.Problem
	if err := db.DB.Where("code = ?", code).First(&problem).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, apiError("problem not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	var clarifications []models.ProblemClarification
	if err := db.DB.Where("problem_id = ?", problem.ID).
		Order("date DESC").
		Find(&clarifications).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	type Item struct {
		ID          uint      `json:"id"`
		Description string    `json:"description"`
		Date        time.Time `json:"date"`
	}

	items := make([]Item, len(clarifications))
	for i, clar := range clarifications {
		items[i] = Item{
			ID:          clar.ID,
			Description: clar.Description,
			Date:        clar.Date,
		}
	}

	c.JSON(http.StatusOK, apiList(items))
}

// ProblemClarificationCreate – POST /api/v2/admin/problem/:code/clarification
// Admin endpoint to create a new problem clarification
func ProblemClarificationCreate(c *gin.Context) {
	code := c.Param("code")

	// Get problem
	var problem models.Problem
	if err := db.DB.Where("code = ?", code).First(&problem).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, apiError("problem not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	var req struct {
		Description string `json:"description" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, apiError(err.Error()))
		return
	}

	clarification := models.ProblemClarification{
		ProblemID:   problem.ID,
		Description: req.Description,
		Date:        time.Now(),
	}

	if err := db.DB.Create(&clarification).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":           clarification.ID,
		"problem_id":   clarification.ProblemID,
		"description":  clarification.Description,
		"date":         clarification.Date,
		"problem_code": code,
	})
}

// ProblemClarificationDelete – DELETE /api/v2/admin/problem/clarification/:id
// Admin endpoint to delete a problem clarification
func ProblemClarificationDelete(c *gin.Context) {
	id := c.Param("id")

	var clarification models.ProblemClarification
	if err := db.DB.First(&clarification, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, apiError("clarification not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	if err := db.DB.Delete(&clarification).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "clarification deleted"})
}
