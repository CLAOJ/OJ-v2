package v2

import (
	"net/http"
	"strconv"

	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/gin-gonic/gin"
)

// AdminLanguageLimitList - GET /api/v2/admin/language-limits
// Lists all language limits with optional problem_id filter
func AdminLanguageLimitList(c *gin.Context) {
	page, pageSize := parsePagination(c)
	problemID := c.Query("problem_id")

	query := db.DB.Model(&models.LanguageLimit{}).Preload("Language").Preload("Problem")

	if problemID != "" {
		query = query.Where("problem_id = ?", problemID)
	}

	var total int64
	query.Count(&total)

	var limits []models.LanguageLimit
	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Find(&limits).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	c.JSON(http.StatusOK, apiListWithTotal(limits, total))
}

// AdminLanguageLimitDetail - GET /api/v2/admin/language-limit/:id
// Gets a specific language limit by ID
func AdminLanguageLimitDetail(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ID"})
		return
	}

	var limit models.LanguageLimit
	if err := db.DB.Preload("Language").Preload("Problem").First(&limit, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "language limit not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": limit})
}

// AdminLanguageLimitCreate - POST /api/v2/admin/language-limits
// Creates a new language limit for a problem
func AdminLanguageLimitCreate(c *gin.Context) {
	var input struct {
		ProblemID   uint    `json:"problem_id" binding:"required"`
		LanguageID  uint    `json:"language_id" binding:"required"`
		TimeLimit   float64 `json:"time_limit" binding:"required"`
		MemoryLimit int     `json:"memory_limit" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, apiError(err.Error()))
		return
	}

	// Check if problem-language pair already exists
	var existing models.LanguageLimit
	if err := db.DB.Where("problem_id = ? AND language_id = ?", input.ProblemID, input.LanguageID).First(&existing).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "language limit already exists for this problem and language"})
		return
	}

	limit := models.LanguageLimit{
		ProblemID:   input.ProblemID,
		LanguageID:  input.LanguageID,
		TimeLimit:   input.TimeLimit,
		MemoryLimit: input.MemoryLimit,
	}

	if err := db.DB.Create(&limit).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	// Reload with associations
	db.DB.Preload("Language").Preload("Problem").First(&limit, limit.ID)

	c.JSON(http.StatusCreated, gin.H{"data": limit, "success": true, "message": "Language limit created"})
}

// AdminLanguageLimitUpdate - PATCH /api/v2/admin/language-limit/:id
// Updates an existing language limit
func AdminLanguageLimitUpdate(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ID"})
		return
	}

	var input struct {
		TimeLimit   float64 `json:"time_limit"`
		MemoryLimit int     `json:"memory_limit"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, apiError(err.Error()))
		return
	}

	var limit models.LanguageLimit
	if err := db.DB.First(&limit, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "language limit not found"})
		return
	}

	updates := make(map[string]interface{})
	if input.TimeLimit > 0 {
		updates["time_limit"] = input.TimeLimit
	}
	if input.MemoryLimit > 0 {
		updates["memory_limit"] = input.MemoryLimit
	}

	if err := db.DB.Model(&limit).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	// Reload with associations
	db.DB.Preload("Language").Preload("Problem").First(&limit, limit.ID)

	c.JSON(http.StatusOK, gin.H{"data": limit, "success": true, "message": "Language limit updated"})
}

// AdminLanguageLimitDelete - DELETE /api/v2/admin/language-limit/:id
// Deletes a language limit
func AdminLanguageLimitDelete(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ID"})
		return
	}

	var limit models.LanguageLimit
	if err := db.DB.First(&limit, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "language limit not found"})
		return
	}

	if err := db.DB.Delete(&limit).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Language limit deleted"})
}

// ProblemLanguageLimits - GET /api/v2/problem/:code/language-limits
// Gets all language limits for a specific problem (public endpoint)
func ProblemLanguageLimits(c *gin.Context) {
	code := c.Param("code")

	var problem models.Problem
	if err := db.DB.Where("code = ?", code).First(&problem).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "problem not found"})
		return
	}

	var limits []models.LanguageLimit
	if err := db.DB.Where("problem_id = ?", problem.ID).Preload("Language").Find(&limits).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": limits})
}
