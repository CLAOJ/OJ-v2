package v2

import (
	"net/http"
	"time"

	"github.com/CLAOJ/claoj-go/contribution"
	"github.com/CLAOJ/claoj-go/db"
	"github.com/CLAOJ/claoj-go/models"
	"github.com/CLAOJ/claoj-go/sanitization"
	"github.com/gin-gonic/gin"
)

// ProblemSuggestRequest represents a problem suggestion submission
type ProblemSuggestRequest struct {
	Name              string  `json:"name" binding:"required"`
	Description       string  `json:"description" binding:"required"`
	Points            float64 `json:"points" binding:"required"`
	Partial           bool    `json:"partial"`
	TimeLimit         float64 `json:"time_limit" binding:"required"`
	MemoryLimit       uint    `json:"memory_limit" binding:"required"`
	GroupID           uint    `json:"group_id" binding:"required"`
	TypeIDs           []uint  `json:"type_ids"`
	Source            string  `json:"source"`
	Summary           string  `json:"summary"`
	PdfURL            string  `json:"pdf_url"`
	IsFullMarkup      bool    `json:"is_full_markup"`
	ShortCircuit      bool    `json:"short_circuit"`
	AdditionalNotes   string  `json:"additional_notes"` // Notes from suggester to admins
}

// SuggestProblem - POST /api/v2/problems/suggest
// Submit a new problem suggestion for admin review
func SuggestProblem(c *gin.Context) {
	var input ProblemSuggestRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, apiError(err.Error()))
		return
	}

	// Get current user's profile ID
	profileID, exists := c.Get("profile_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, apiError("user not authenticated"))
		return
	}
	profileIDUint := profileID.(uint)

	// Generate a temporary code (will be finalized on approval)
	code := "SUGGEST_" + time.Now().Format("20060102150405")

	problem := models.Problem{
		Code:              code,
		Name:              sanitization.SanitizeTitle(input.Name),
		Description:       sanitization.SanitizeProblemContent(input.Description),
		Points:            input.Points,
		Partial:           input.Partial,
		IsPublic:          false, // Not public until approved
		TimeLimit:         input.TimeLimit,
		MemoryLimit:       input.MemoryLimit,
		GroupID:           input.GroupID,
		Source:            input.Source,
		Summary:           input.Summary,
		PdfURL:            input.PdfURL,
		IsFullMarkup:      input.IsFullMarkup,
		ShortCircuit:      input.ShortCircuit,
		SuggesterID:       &profileIDUint,
		SuggestionStatus:  "pending",
		SuggestionNotes:   input.AdditionalNotes,
		IsManuallyManaged: true, // Suggestions are manually managed
	}

	if err := db.DB.Create(&problem).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	// Handle many-to-many relations
	if len(input.TypeIDs) > 0 {
		var types []models.ProblemType
		if err := db.DB.Where("id IN ?", input.TypeIDs).Find(&types).Error; err == nil {
			db.DB.Model(&problem).Association("Types").Append(&types)
		}
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Problem suggestion submitted successfully",
		"suggestion": gin.H{
			"id":     problem.ID,
			"code":   problem.Code,
			"name":   problem.Name,
			"status": problem.SuggestionStatus,
		},
	})
}

// GetUserSuggestions - GET /api/v2/my-suggestions
// Get the current user's problem suggestions
func GetUserSuggestions(c *gin.Context) {
	profileID, exists := c.Get("profile_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, apiError("user not authenticated"))
		return
	}
	profileIDUint := profileID.(uint)

	page, pageSize := parsePagination(c)

	var suggestions []models.Problem
	var total int64

	// Count total
	db.DB.Model(&models.Problem{}).
		Where("suggester_id = ?", profileIDUint).
		Count(&total)

	// Get suggestions
	if err := db.DB.
		Preload("Types").
		Preload("Group").
		Where("suggester_id = ?", profileIDUint).
		Order("id DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&suggestions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	type SuggestionItem struct {
		ID               uint       `json:"id"`
		Code             string     `json:"code"`
		Name             string     `json:"name"`
		Points           float64    `json:"points"`
		TimeLimit        float64    `json:"time_limit"`
		MemoryLimit      uint       `json:"memory_limit"`
		Group            string     `json:"group"`
		SuggestionStatus string     `json:"suggestion_status"`
		SuggesterID      *uint      `json:"suggester_id"`
		Date             *time.Time `json:"date"`
	}

	items := make([]SuggestionItem, len(suggestions))
	for i, p := range suggestions {
		item := SuggestionItem{
			ID:               p.ID,
			Code:             p.Code,
			Name:             p.Name,
			Points:           p.Points,
			TimeLimit:        p.TimeLimit,
			MemoryLimit:      p.MemoryLimit,
			Group:            p.Group.FullName,
			SuggestionStatus: p.SuggestionStatus,
			SuggesterID:      p.SuggesterID,
			Date:             p.Date,
		}
		items[i] = item
	}

	c.JSON(http.StatusOK, apiList(items))
}

// AdminProblemSuggestionList - GET /api/v2/admin/problem-suggestions
// List all problem suggestions (pending, approved, rejected)
func AdminProblemSuggestionList(c *gin.Context) {
	page, pageSize := parsePagination(c)
	status := c.Query("status") // pending, approved, rejected, or empty for all

	var suggestions []models.Problem
	var total int64

	q := db.DB.Model(&models.Problem{}).
		Preload("Types").
		Preload("Group").
		Preload("Suggester.User")

	if status != "" {
		q = q.Where("suggestion_status = ?", status)
	}

	// Count total
	q.Count(&total)

	// Get suggestions
	if err := q.
		Order("id DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&suggestions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	type SuggestionAdminItem struct {
		ID                   uint       `json:"id"`
		Code                 string     `json:"code"`
		Name                 string     `json:"name"`
		Points               float64    `json:"points"`
		SuggestionStatus     string     `json:"suggestion_status"`
		SuggesterID          *uint      `json:"suggester_id"`
		SuggesterUsername    string     `json:"suggester_username"`
		SuggestionNotes      string     `json:"suggestion_notes"`
		SuggestionReviewedAt *time.Time `json:"suggestion_reviewed_at"`
		SuggestionReviewedBy *uint      `json:"suggestion_reviewed_by_id"`
		IsPublic             bool       `json:"is_public"`
		Date                 *time.Time `json:"date"`
	}

	items := make([]SuggestionAdminItem, len(suggestions))
	for i, p := range suggestions {
		username := ""
		if p.Suggester != nil && p.Suggester.User.Username != "" {
			username = p.Suggester.User.Username
		}
		items[i] = SuggestionAdminItem{
			ID:                   p.ID,
			Code:                 p.Code,
			Name:                 p.Name,
			Points:               p.Points,
			SuggestionStatus:     p.SuggestionStatus,
			SuggesterID:          p.SuggesterID,
			SuggesterUsername:    username,
			SuggestionNotes:      p.SuggestionNotes,
			SuggestionReviewedAt: p.SuggestionReviewedAt,
			SuggestionReviewedBy: p.SuggestionReviewedBy,
			IsPublic:             p.IsPublic,
			Date:                 p.Date,
		}
	}

	c.JSON(http.StatusOK, apiList(items))
}

// AdminProblemSuggestionDetail - GET /api/v2/admin/problem-suggestion/:id
// Get details of a specific problem suggestion
func AdminProblemSuggestionDetail(c *gin.Context) {
	id := c.Param("id")

	var problem models.Problem
	if err := db.DB.
		Preload("Types").
		Preload("Group").
		Preload("Authors.User").
		Preload("Suggester.User").
		Where("id = ?", id).
		First(&problem).Error; err != nil {
		c.JSON(http.StatusNotFound, apiError("suggestion not found"))
		return
	}

	suggesterUsername := ""
	suggesterEmail := ""
	if problem.Suggester != nil {
		suggesterUsername = problem.Suggester.User.Username
		suggesterEmail = problem.Suggester.User.Email
	}

	c.JSON(http.StatusOK, gin.H{
		"id":                     problem.ID,
		"code":                   problem.Code,
		"name":                   problem.Name,
		"description":            problem.Description,
		"points":                 problem.Points,
		"partial":                problem.Partial,
		"time_limit":             problem.TimeLimit,
		"memory_limit":           problem.MemoryLimit,
		"group_id":               problem.GroupID,
		"group":                  problem.Group.FullName,
		"types":                  problem.Types,
		"source":                 problem.Source,
		"summary":                problem.Summary,
		"pdf_url":                problem.PdfURL,
		"is_full_markup":         problem.IsFullMarkup,
		"short_circuit":          problem.ShortCircuit,
		"suggestion_status":      problem.SuggestionStatus,
		"suggestion_notes":       problem.SuggestionNotes,
		"suggestion_reviewed_at": problem.SuggestionReviewedAt,
		"suggester_id":           problem.SuggesterID,
		"suggester_username":     suggesterUsername,
		"suggester_email":        suggesterEmail,
		"is_public":              problem.IsPublic,
		"authors":                problem.Authors,
	})
}

// ApproveProblemSuggestionInput represents the input for approving a suggestion
type ApproveProblemSuggestionInput struct {
	Code           string `json:"code" binding:"required"` // Final problem code
	AdminNotes     string `json:"admin_notes"`
	IsPublic       bool   `json:"is_public"`
	MakeFullMarkup bool   `json:"make_full_markup"`
}

// AdminProblemSuggestionApprove - POST /api/v2/admin/problem-suggestion/:id/approve
// Approve a problem suggestion and create the problem
func AdminProblemSuggestionApprove(c *gin.Context) {
	id := c.Param("id")

	var input ApproveProblemSuggestionInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, apiError(err.Error()))
		return
	}

	// Get the suggestion
	var problem models.Problem
	if err := db.DB.Where("id = ?", id).First(&problem).Error; err != nil {
		c.JSON(http.StatusNotFound, apiError("suggestion not found"))
		return
	}

	if problem.SuggestionStatus != "pending" {
		c.JSON(http.StatusBadRequest, apiError("suggestion is not pending"))
		return
	}

	// Check if code already exists
	var existingCount int64
	db.DB.Model(&models.Problem{}).Where("code = ?", input.Code).Count(&existingCount)
	if existingCount > 0 {
		c.JSON(http.StatusBadRequest, apiError("problem code already exists"))
		return
	}

	// Get admin profile ID
	adminProfileID, _ := c.Get("profile_id")
	adminProfileIDUint := adminProfileID.(uint)
	now := time.Now()

	// Update the problem
	updates := map[string]interface{}{
		"code":                      input.Code,
		"name":                      sanitization.SanitizeTitle(problem.Name),
		"is_public":                 input.IsPublic,
		"suggestion_status":         "approved",
		"suggestion_notes":          problem.SuggestionNotes + "\n\n[Admin Notes] " + input.AdminNotes,
		"suggestion_reviewed_at":    &now,
		"suggestion_reviewed_by_id": adminProfileIDUint,
	}

	if input.MakeFullMarkup {
		updates["is_full_markup"] = true
	}

	if err := db.DB.Model(&problem).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	// Award contribution points to suggester
	if problem.SuggesterID != nil {
		// Update the suggester's contribution points
		if err := contribution.UpdateProfileContributionPoints(*problem.SuggesterID); err != nil {
			// Log error but don't fail the request
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Problem suggestion approved successfully",
		"problem": gin.H{
			"id":   problem.ID,
			"code": input.Code,
			"name": problem.Name,
		},
	})
}

// RejectProblemSuggestionInput represents the input for rejecting a suggestion
type RejectProblemSuggestionInput struct {
	AdminNotes string `json:"admin_notes"`
	Reason     string `json:"reason"`
}

// AdminProblemSuggestionReject - POST /api/v2/admin/problem-suggestion/:id/reject
// Reject a problem suggestion
func AdminProblemSuggestionReject(c *gin.Context) {
	id := c.Param("id")

	var input RejectProblemSuggestionInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, apiError(err.Error()))
		return
	}

	// Get the suggestion
	var problem models.Problem
	if err := db.DB.Where("id = ?", id).First(&problem).Error; err != nil {
		c.JSON(http.StatusNotFound, apiError("suggestion not found"))
		return
	}

	if problem.SuggestionStatus != "pending" {
		c.JSON(http.StatusBadRequest, apiError("suggestion is not pending"))
		return
	}

	// Get admin profile ID
	adminProfileID, _ := c.Get("profile_id")
	adminProfileIDUint := adminProfileID.(uint)
	now := time.Now()

	// Update the problem
	updates := map[string]interface{}{
		"suggestion_status":       "rejected",
		"suggestion_notes":        problem.SuggestionNotes + "\n\n[Rejected] Reason: " + input.Reason + "\nAdmin Notes: " + input.AdminNotes,
		"suggestion_reviewed_at":  &now,
		"suggestion_reviewed_by_id": adminProfileIDUint,
	}

	if err := db.DB.Model(&problem).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Problem suggestion rejected",
	})
}

// AdminProblemSuggestionDelete - DELETE /api/v2/admin/problem-suggestion/:id
// Delete a problem suggestion
func AdminProblemSuggestionDelete(c *gin.Context) {
	id := c.Param("id")

	var problem models.Problem
	if err := db.DB.Where("id = ?", id).First(&problem).Error; err != nil {
		c.JSON(http.StatusNotFound, apiError("suggestion not found"))
		return
	}

	// Only allow deleting pending or rejected suggestions
	if problem.SuggestionStatus == "approved" {
		c.JSON(http.StatusBadRequest, apiError("cannot delete approved suggestions; use problem delete instead"))
		return
	}

	if err := db.DB.Delete(&problem).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Problem suggestion deleted",
	})
}
