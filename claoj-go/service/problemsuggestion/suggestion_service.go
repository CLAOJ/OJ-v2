// Package problemsuggestion provides problem suggestion management services.
package problemsuggestion

import (
	"time"

	"github.com/CLAOJ/claoj-go/contribution"
	"github.com/CLAOJ/claoj-go/db"
	"github.com/CLAOJ/claoj-go/models"
	"github.com/CLAOJ/claoj-go/sanitization"
	"gorm.io/gorm"
)

// ProblemSuggestionService provides problem suggestion management operations.
type ProblemSuggestionService struct{}

// NewProblemSuggestionService creates a new ProblemSuggestionService instance.
func NewProblemSuggestionService() *ProblemSuggestionService {
	return &ProblemSuggestionService{}
}

// ListSuggestions retrieves a paginated list of problem suggestions.
func (s *ProblemSuggestionService) ListSuggestions(req ListSuggestionsRequest) (*ListSuggestionsResponse, error) {
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 20
	}
	if req.PageSize > 100 {
		req.PageSize = 100
	}

	var suggestions []models.Problem
	query := db.DB.Model(&models.Problem{}).
		Preload("Types").
		Preload("Group").
		Preload("Suggester.User")

	if req.Status != "" {
		query = query.Where("suggestion_status = ?", req.Status)
	}

	// Get total count
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	// Get paginated results
	if err := query.
		Order("id DESC").
		Offset((req.Page - 1) * req.PageSize).
		Limit(req.PageSize).
		Find(&suggestions).Error; err != nil {
		return nil, err
	}

	result := make([]ProblemSuggestion, len(suggestions))
	for i, p := range suggestions {
		result[i] = suggestionToModel(p)
	}

	return &ListSuggestionsResponse{
		Suggestions: result,
		Total:       total,
		Page:        req.Page,
		PageSize:    req.PageSize,
	}, nil
}

// GetSuggestion retrieves a problem suggestion by ID with full details.
func (s *ProblemSuggestionService) GetSuggestion(req GetSuggestionRequest) (*ProblemSuggestionDetail, error) {
	if req.SuggestionID == 0 {
		return nil, ErrInvalidSuggestionID
	}

	var problem models.Problem
	if err := db.DB.
		Preload("Types").
		Preload("Group").
		Preload("Authors.User").
		Preload("Suggester.User").
		First(&problem, req.SuggestionID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrSuggestionNotFound
		}
		return nil, err
	}

	detail := &ProblemSuggestionDetail{
		Suggestion: suggestionToModel(problem),
	}

	if problem.Suggester != nil {
		detail.SuggesterUsername = problem.Suggester.User.Username
		detail.SuggesterEmail = problem.Suggester.User.Email
	}

	if problem.GroupID > 0 {
		detail.GroupName = problem.Group.FullName
	}

	typeNames := make([]string, len(problem.Types))
	for i, t := range problem.Types {
		typeNames[i] = t.FullName
	}
	detail.TypeNames = typeNames

	return detail, nil
}

// ApproveSuggestion approves a problem suggestion and converts it to a problem.
func (s *ProblemSuggestionService) ApproveSuggestion(req ApproveSuggestionRequest) (*models.Problem, error) {
	if req.SuggestionID == 0 {
		return nil, ErrInvalidSuggestionID
	}

	if req.Code == "" {
		return nil, ErrInvalidSuggestionID // Using same error for simplicity
	}

	// Get the suggestion
	var problem models.Problem
	if err := db.DB.
		Preload("Types").
		First(&problem, req.SuggestionID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrSuggestionNotFound
		}
		return nil, err
	}

	// Check status
	if problem.SuggestionStatus != "pending" {
		if problem.SuggestionStatus == "approved" {
			return nil, ErrSuggestionAlreadyApproved
		}
		return nil, ErrSuggestionNotPending
	}

	// Check if code already exists
	var existingCount int64
	if err := db.DB.Model(&models.Problem{}).Where("code = ?", req.Code).Count(&existingCount).Error; err != nil {
		return nil, err
	}
	if existingCount > 0 {
		return nil, ErrProblemCodeExists
	}

	now := time.Now()

	// Update the problem
	updates := map[string]interface{}{
		"code":                      req.Code,
		"name":                      sanitization.SanitizeTitle(problem.Name),
		"is_public":                 req.IsPublic,
		"suggestion_status":         "approved",
		"suggestion_notes":          problem.SuggestionNotes + "\n\n[Admin Notes] " + req.AdminNotes,
		"suggestion_reviewed_at":    &now,
		"suggestion_reviewed_by_id": req.AdminID,
	}

	if req.MakeFullMarkup {
		updates["is_full_markup"] = true
	}

	if err := db.DB.Model(&problem).Updates(updates).Error; err != nil {
		return nil, err
	}

	// Award contribution points to suggester
	if problem.SuggesterID != nil {
		// Ignore errors for contribution points update
		_ = contribution.UpdateProfileContributionPoints(*problem.SuggesterID)
	}

	// Reload problem with relations
	db.DB.Preload("Types").Preload("Group").First(&problem, problem.ID)

	return &problem, nil
}

// RejectSuggestion rejects a problem suggestion.
func (s *ProblemSuggestionService) RejectSuggestion(req RejectSuggestionRequest) error {
	if req.SuggestionID == 0 {
		return ErrInvalidSuggestionID
	}

	if req.Reason == "" {
		return ErrEmptyReason
	}

	// Get the suggestion
	var problem models.Problem
	if err := db.DB.First(&problem, req.SuggestionID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return ErrSuggestionNotFound
		}
		return err
	}

	// Check status
	if problem.SuggestionStatus != "pending" {
		if problem.SuggestionStatus == "rejected" {
			return ErrSuggestionAlreadyRejected
		}
		return ErrSuggestionNotPending
	}

	now := time.Now()

	// Update the problem
	updates := map[string]interface{}{
		"suggestion_status":         "rejected",
		"suggestion_notes":          problem.SuggestionNotes + "\n\n[Rejected] Reason: " + req.Reason + "\nAdmin Notes: " + req.AdminNotes,
		"suggestion_reviewed_at":    &now,
		"suggestion_reviewed_by_id": req.AdminID,
	}

	return db.DB.Model(&problem).Updates(updates).Error
}

// DeleteSuggestion deletes a problem suggestion.
// Only pending or rejected suggestions can be deleted.
func (s *ProblemSuggestionService) DeleteSuggestion(req DeleteSuggestionRequest) error {
	if req.SuggestionID == 0 {
		return ErrInvalidSuggestionID
	}

	var problem models.Problem
	if err := db.DB.First(&problem, req.SuggestionID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return ErrSuggestionNotFound
		}
		return err
	}

	// Only allow deleting pending or rejected suggestions
	if problem.SuggestionStatus == "approved" {
		return ErrSuggestionAlreadyApproved // Reusing error - cannot delete approved
	}

	return db.DB.Delete(&problem).Error
}

// Helper functions

func suggestionToModel(p models.Problem) ProblemSuggestion {
	typeIDs := make([]uint, len(p.Types))
	for i, t := range p.Types {
		typeIDs[i] = t.ID
	}

	return ProblemSuggestion{
		ID:                   p.ID,
		Code:                 p.Code,
		Name:                 p.Name,
		Description:          p.Description,
		Points:               p.Points,
		Partial:              p.Partial,
		IsPublic:             p.IsPublic,
		TimeLimit:            p.TimeLimit,
		MemoryLimit:          p.MemoryLimit,
		GroupID:              p.GroupID,
		TypeIDs:              typeIDs,
		Source:               p.Source,
		Summary:              p.Summary,
		PdfURL:               p.PdfURL,
		IsFullMarkup:         p.IsFullMarkup,
		ShortCircuit:         p.ShortCircuit,
		SuggesterID:          p.SuggesterID,
		SuggestionStatus:     p.SuggestionStatus,
		SuggestionNotes:      p.SuggestionNotes,
		SuggestionReviewedAt: p.SuggestionReviewedAt,
		SuggestionReviewedBy: p.SuggestionReviewedBy,
		Date:                 p.Date,
	}
}
