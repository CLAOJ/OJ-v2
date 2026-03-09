package v2

import (
	"net/http"
	"strconv"
	"time"

	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/CLAOJ/claoj/sanitization"
	"github.com/CLAOJ/claoj/service/problemsuggestion"
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

	resp, err := getSuggestionService().ListSuggestions(problemsuggestion.ListSuggestionsRequest{
		Page:     page,
		PageSize: pageSize,
		Status:   status,
	})
	if err != nil {
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

	items := make([]SuggestionAdminItem, len(resp.Suggestions))
	for i, p := range resp.Suggestions {
		// Note: For list view, we don't have suggester username here
		// The service would need to be extended to include this, or we use a separate endpoint
		items[i] = SuggestionAdminItem{
			ID:                   p.ID,
			Code:                 p.Code,
			Name:                 p.Name,
			Points:               p.Points,
			SuggestionStatus:     p.SuggestionStatus,
			SuggesterID:          p.SuggesterID,
			SuggestionNotes:      p.SuggestionNotes,
			SuggestionReviewedAt: p.SuggestionReviewedAt,
			SuggestionReviewedBy: p.SuggestionReviewedBy,
			IsPublic:             p.IsPublic,
			Date:                 p.Date,
		}
	}

	c.JSON(http.StatusOK, apiListWithTotal(items, resp.Total))
}

// AdminProblemSuggestionDetail - GET /api/v2/admin/problem-suggestion/:id
// Get details of a specific problem suggestion
func AdminProblemSuggestionDetail(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, apiError("invalid suggestion ID"))
		return
	}

	detail, err := getSuggestionService().GetSuggestion(problemsuggestion.GetSuggestionRequest{
		SuggestionID: uint(id),
	})
	if err != nil {
		if err == problemsuggestion.ErrSuggestionNotFound {
			c.JSON(http.StatusNotFound, apiError("suggestion not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":                     detail.Suggestion.ID,
		"code":                   detail.Suggestion.Code,
		"name":                   detail.Suggestion.Name,
		"description":            detail.Suggestion.Description,
		"points":                 detail.Suggestion.Points,
		"partial":                detail.Suggestion.Partial,
		"time_limit":             detail.Suggestion.TimeLimit,
		"memory_limit":           detail.Suggestion.MemoryLimit,
		"group_id":               detail.Suggestion.GroupID,
		"group":                  detail.GroupName,
		"types":                  detail.TypeNames,
		"source":                 detail.Suggestion.Source,
		"summary":                detail.Suggestion.Summary,
		"pdf_url":                detail.Suggestion.PdfURL,
		"is_full_markup":         detail.Suggestion.IsFullMarkup,
		"short_circuit":          detail.Suggestion.ShortCircuit,
		"suggestion_status":      detail.Suggestion.SuggestionStatus,
		"suggestion_notes":       detail.Suggestion.SuggestionNotes,
		"suggestion_reviewed_at": detail.Suggestion.SuggestionReviewedAt,
		"suggester_id":           detail.Suggestion.SuggesterID,
		"suggester_username":     detail.SuggesterUsername,
		"suggester_email":        detail.SuggesterEmail,
		"is_public":              detail.Suggestion.IsPublic,
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
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, apiError("invalid suggestion ID"))
		return
	}

	var input ApproveProblemSuggestionInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, apiError(err.Error()))
		return
	}

	// Get admin profile ID
	adminProfileID, _ := c.Get("profile_id")
	adminProfileIDUint := adminProfileID.(uint)

	problem, err := getSuggestionService().ApproveSuggestion(problemsuggestion.ApproveSuggestionRequest{
		SuggestionID:   uint(id),
		Code:           input.Code,
		AdminNotes:     input.AdminNotes,
		IsPublic:       input.IsPublic,
		MakeFullMarkup: input.MakeFullMarkup,
		AdminID:        adminProfileIDUint,
	})
	if err != nil {
		switch {
		case err == problemsuggestion.ErrSuggestionNotFound:
			c.JSON(http.StatusNotFound, apiError("suggestion not found"))
		case err == problemsuggestion.ErrSuggestionNotPending:
			c.JSON(http.StatusBadRequest, apiError("suggestion is not pending"))
		case err == problemsuggestion.ErrSuggestionAlreadyApproved:
			c.JSON(http.StatusBadRequest, apiError("suggestion already approved"))
		case err == problemsuggestion.ErrProblemCodeExists:
			c.JSON(http.StatusBadRequest, apiError("problem code already exists"))
		default:
			c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Problem suggestion approved successfully",
		"problem": gin.H{
			"id":   problem.ID,
			"code": problem.Code,
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
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, apiError("invalid suggestion ID"))
		return
	}

	var input RejectProblemSuggestionInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, apiError(err.Error()))
		return
	}

	// Get admin profile ID
	adminProfileID, _ := c.Get("profile_id")
	adminProfileIDUint := adminProfileID.(uint)

	if err := getSuggestionService().RejectSuggestion(problemsuggestion.RejectSuggestionRequest{
		SuggestionID: uint(id),
		Reason:       input.Reason,
		AdminNotes:   input.AdminNotes,
		AdminID:      adminProfileIDUint,
	}); err != nil {
		switch {
		case err == problemsuggestion.ErrSuggestionNotFound:
			c.JSON(http.StatusNotFound, apiError("suggestion not found"))
		case err == problemsuggestion.ErrSuggestionNotPending:
			c.JSON(http.StatusBadRequest, apiError("suggestion is not pending"))
		case err == problemsuggestion.ErrSuggestionAlreadyRejected:
			c.JSON(http.StatusBadRequest, apiError("suggestion already rejected"))
		case err == problemsuggestion.ErrEmptyReason:
			c.JSON(http.StatusBadRequest, apiError("rejection reason cannot be empty"))
		default:
			c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		}
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
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, apiError("invalid suggestion ID"))
		return
	}

	if err := getSuggestionService().DeleteSuggestion(problemsuggestion.DeleteSuggestionRequest{
		SuggestionID: uint(id),
	}); err != nil {
		switch {
		case err == problemsuggestion.ErrSuggestionNotFound:
			c.JSON(http.StatusNotFound, apiError("suggestion not found"))
		case err == problemsuggestion.ErrSuggestionAlreadyApproved:
			c.JSON(http.StatusBadRequest, apiError("cannot delete approved suggestions; use problem delete instead"))
		default:
			c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Problem suggestion deleted",
	})
}
