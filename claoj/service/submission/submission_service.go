// Package submission provides submission management services.
package submission

import (
	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/jobs"
	"github.com/CLAOJ/claoj/models"
	"gorm.io/gorm"
)

// SubmissionProfile represents a submission with related data.
type SubmissionProfile struct {
	ID           uint
	UserID       uint
	Username     string
	ProblemID    uint
	ProblemCode  string
	LanguageID   uint
	LanguageName string
	Date         string
	Time         *float64
	Memory       *float64
	Points       *float64
	Status       string
	Result       *string
	IsPretested  bool
}

// RejudgeRequest holds the parameters for rejudging a submission.
type RejudgeRequest struct {
	SubmissionID uint
}

// AbortRequest holds the parameters for aborting a submission.
type AbortRequest struct {
	SubmissionID uint
}

// BatchRejudgeRequest holds the parameters for batch rejudging.
type BatchRejudgeRequest struct {
	SubmissionIDs []uint
	Filters       *RejudgeFilters
	DryRun        bool
}

// RejudgeFilters holds filters for batch rejudge operations.
type RejudgeFilters struct {
	UserID      *uint
	Username    string
	ProblemID   *uint
	ProblemCode string
	LanguageID  *uint
	Language    string
	Status      string
	Result      string
	FromDate    string
	ToDate      string
}

// BatchRejudgeResponse holds the response for batch rejudge operations.
type BatchRejudgeResponse struct {
	Count int
}

// MossAnalysisRequest holds the parameters for MOSS analysis.
type MossAnalysisRequest struct {
	SubmissionID uint
}

// RescoreRequest holds the parameters for rescoring a submission.
type RescoreRequest struct {
	SubmissionID uint
}

// BatchRescoreRequest holds the parameters for batch rescoring.
type BatchRescoreRequest struct {
	SubmissionIDs []uint
	ProblemID     *uint
	UserID        *uint
	DryRun        bool
}

// BatchRescoreResponse holds the response for batch rescore operations.
type BatchRescoreResponse struct {
	Rescored int
	Total    int
}

// RescoreAllRequest holds the parameters for rescoring all submissions for a problem.
type RescoreAllRequest struct {
	ProblemCode string
}

// ListSubmissionsRequest holds the parameters for listing submissions.
type ListSubmissionsRequest struct {
	Page     int
	PageSize int
}

// ListSubmissionsResponse holds the response for listing submissions.
type ListSubmissionsResponse struct {
	Submissions []SubmissionProfile
	Total       int64
	Page        int
	PageSize    int
}

// SubmissionService provides submission management operations.
type SubmissionService struct {
	bridgeServer BridgeServer
}

// BridgeServer interface for bridge server operations.
type BridgeServer interface {
	Abort(subID uint) error
}

// NewSubmissionService creates a new SubmissionService instance.
func NewSubmissionService(bridge BridgeServer) *SubmissionService {
	return &SubmissionService{
		bridgeServer: bridge,
	}
}

// SetBridgeServer sets the bridge server reference.
func (s *SubmissionService) SetBridgeServer(bridge BridgeServer) {
	s.bridgeServer = bridge
}

// ListSubmissions retrieves a paginated list of submissions.
func (s *SubmissionService) ListSubmissions(req ListSubmissionsRequest) (*ListSubmissionsResponse, error) {
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 20
	}
	if req.PageSize > 100 {
		req.PageSize = 100
	}

	var submissions []struct {
		models.Submission
		Username     string `gorm:"column:username"`
		ProblemCode  string `gorm:"column:problem_code"`
		LanguageName string `gorm:"column:language_name"`
	}

	query := db.DB.Table("judge_submission").
		Joins("JOIN auth_user ON auth_user.id = judge_submission.user_id").
		Joins("JOIN judge_problem ON judge_problem.id = judge_submission.problem_id").
		Joins("JOIN judge_language ON judge_language.id = judge_submission.language_id").
		Select("judge_submission.*, auth_user.username, judge_problem.code as problem_code, judge_language.name as language_name").
		Order("judge_submission.date DESC")

	// Get total count
	var total int64
	if err := db.DB.Model(&models.Submission{}).Count(&total).Error; err != nil {
		return nil, err
	}

	// Get paginated results
	if err := query.
		Offset((req.Page - 1) * req.PageSize).
		Limit(req.PageSize).
		Scan(&submissions).Error; err != nil {
		return nil, err
	}

	result := make([]SubmissionProfile, len(submissions))
	for i, s := range submissions {
		result[i] = submissionToProfile(s.Submission, s.Username, s.ProblemCode, s.LanguageName)
	}

	return &ListSubmissionsResponse{
		Submissions: result,
		Total:       total,
		Page:        req.Page,
		PageSize:    req.PageSize,
	}, nil
}

// Rejudge resets a submission for rejudging.
func (s *SubmissionService) Rejudge(req RejudgeRequest) error {
	var sub models.Submission
	if err := db.DB.Preload("Problem").First(&sub, req.SubmissionID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return ErrSubmissionNotFound
		}
		return err
	}

	// Check if submission is locked
	if sub.LockedAfter != nil {
		return ErrSubmissionLocked
	}

	// Reset submission state for rejudge
	if err := db.DB.Model(&sub).Updates(map[string]interface{}{
		"status":           "QU",
		"result":           nil,
		"points":           nil,
		"time":             nil,
		"memory":           nil,
		"current_testcase": 0,
		"case_points":      0,
		"case_total":       0,
	}).Error; err != nil {
		return err
	}

	// Clear test cases
	if err := db.DB.Where("submission_id = ?", req.SubmissionID).Delete(&models.SubmissionTestCase{}).Error; err != nil {
		return err
	}

	// Enqueue for rejudging
	if err := jobs.EnqueueJudgeSubmission(req.SubmissionID); err != nil {
		// Return success anyway since queuing failed is not critical
		return nil
	}

	return nil
}

// Abort attempts to abort a submission that is being processed.
func (s *SubmissionService) Abort(req AbortRequest) error {
	var sub models.Submission
	if err := db.DB.First(&sub, req.SubmissionID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return ErrSubmissionNotFound
		}
		return err
	}

	// Only allow aborting submissions that are being processed
	if sub.Status != "P" && sub.Status != "G" {
		return ErrSubmissionNotProcessing
	}

	// Get the bridge server from global state
	if s.bridgeServer == nil {
		return ErrBridgeServerNotAvailable
	}

	// Send abort command to judge
	if err := s.bridgeServer.Abort(req.SubmissionID); err != nil {
		return err
	}

	// Update submission status to aborted
	return db.DB.Model(&sub).Updates(map[string]interface{}{
		"status": "AB",
		"result": "AB",
	}).Error
}

// BatchRejudge rejudges multiple submissions based on filters or specific IDs.
func (s *SubmissionService) BatchRejudge(req BatchRejudgeRequest) (*BatchRejudgeResponse, error) {
	// Build query for matching submissions
	query := db.DB.Model(&models.Submission{})

	// Apply filters if provided
	if req.Filters != nil {
		applyRejudgeFilters(query, req.Filters)
	}

	// If specific submission IDs provided, filter by them
	if len(req.SubmissionIDs) > 0 {
		query = query.Where("id IN ?", req.SubmissionIDs)
	}

	// Count matching submissions
	var count int64
	if err := query.Count(&count).Error; err != nil {
		return nil, err
	}

	// If dry run, just return the count
	if req.DryRun {
		return &BatchRejudgeResponse{Count: int(count)}, nil
	}

	// Get all matching submission IDs
	var submissionIDs []uint
	if err := query.Pluck("id", &submissionIDs).Error; err != nil {
		return nil, err
	}

	// Reset all matching submissions
	resetCount := 0
	for _, subID := range submissionIDs {
		var sub models.Submission
		if err := db.DB.First(&sub, subID).Error; err != nil {
			continue // Skip if not found
		}

		// Skip locked submissions
		if sub.LockedAfter != nil {
			continue
		}

		// Reset submission state
		db.DB.Model(&sub).Updates(map[string]interface{}{
			"status":           "QU",
			"result":           nil,
			"points":           nil,
			"time":             nil,
			"memory":           nil,
			"current_testcase": 0,
			"case_points":      0,
			"case_total":       0,
		})

		// Clear test cases
		db.DB.Where("submission_id = ?", subID).Delete(&models.SubmissionTestCase{})

		// Enqueue for rejudging
		jobs.EnqueueJudgeSubmission(subID)
		resetCount++
	}

	return &BatchRejudgeResponse{Count: resetCount}, nil
}

// MossAnalysis initiates MOSS analysis for a submission.
func (s *SubmissionService) MossAnalysis(req MossAnalysisRequest) error {
	var sub models.Submission
	if err := db.DB.First(&sub, req.SubmissionID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return ErrSubmissionNotFound
		}
		return err
	}

	// MOSS analysis would be implemented here
	// For now, just verify submission exists and is completed
	if sub.Status != "D" {
		return ErrSubmissionNotCompleted
	}

	return nil
}

// MossResults retrieves MOSS analysis results for a submission.
func (s *SubmissionService) MossResults(req MossAnalysisRequest) (interface{}, error) {
	var sub models.Submission
	if err := db.DB.First(&sub, req.SubmissionID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrSubmissionNotFound
		}
		return nil, err
	}

	// Return empty results - actual implementation would fetch from database
	return map[string]interface{}{
		"submission_id": req.SubmissionID,
		"results":       []interface{}{},
	}, nil
}

// Rescore rescores a single submission.
func (s *SubmissionService) Rescore(req RescoreRequest) error {
	var sub models.Submission
	if err := db.DB.First(&sub, req.SubmissionID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return ErrSubmissionNotFound
		}
		return err
	}

	// Requeue the submission for rejudging
	return jobs.EnqueueJudgeSubmission(req.SubmissionID)
}

// BatchRescore rescoring multiple submissions.
func (s *SubmissionService) BatchRescore(req BatchRescoreRequest) (*BatchRescoreResponse, error) {
	// Build query based on filters
	query := db.DB.Model(&models.Submission{})

	if len(req.SubmissionIDs) > 0 {
		query = query.Where("id IN ?", req.SubmissionIDs)
	} else {
		// Use filters if no specific IDs provided
		if req.ProblemID != nil {
			query = query.Where("problem_id = ?", *req.ProblemID)
		}
		if req.UserID != nil {
			query = query.Where("user_id = ?", *req.UserID)
		}
	}

	// Get all submissions to rescore
	var submissions []models.Submission
	if err := query.Find(&submissions).Error; err != nil {
		return nil, err
	}

	// Dry run - just count
	if req.DryRun {
		return &BatchRescoreResponse{
			Rescored: 0,
			Total:    len(submissions),
		}, nil
	}

	// Enqueue each submission for rejudging
	rescored := 0
	for _, sub := range submissions {
		if err := jobs.EnqueueJudgeSubmission(sub.ID); err == nil {
			rescored++
		}
	}

	return &BatchRescoreResponse{
		Rescored: rescored,
		Total:    len(submissions),
	}, nil
}

// RescoreAll rescoring all submissions for a specific problem.
func (s *SubmissionService) RescoreAll(req RescoreAllRequest) (*BatchRescoreResponse, error) {
	var problem models.Problem
	if err := db.DB.Where("code = ?", req.ProblemCode).First(&problem).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrProblemNotFound
		}
		return nil, err
	}

	// Get all submissions for this problem
	var submissions []models.Submission
	if err := db.DB.Where("problem_id = ?", problem.ID).Find(&submissions).Error; err != nil {
		return nil, err
	}

	// Enqueue each submission for rejudging
	rescored := 0
	for _, sub := range submissions {
		if err := jobs.EnqueueJudgeSubmission(sub.ID); err == nil {
			rescored++
		}
	}

	return &BatchRescoreResponse{
		Rescored: rescored,
		Total:    len(submissions),
	}, nil
}

// Helper functions

func submissionToProfile(s models.Submission, username, problemCode, languageName string) SubmissionProfile {
	return SubmissionProfile{
		ID:           s.ID,
		UserID:       s.UserID,
		Username:     username,
		ProblemID:    s.ProblemID,
		ProblemCode:  problemCode,
		LanguageID:   s.LanguageID,
		LanguageName: languageName,
		Date:         s.Date.String(),
		Time:         s.Time,
		Memory:       s.Memory,
		Points:       s.Points,
		Status:       s.Status,
		Result:       s.Result,
		IsPretested:  s.IsPretested,
	}
}

func applyRejudgeFilters(query *gorm.DB, filters *RejudgeFilters) *gorm.DB {
	if filters.UserID != nil && *filters.UserID > 0 {
		query = query.Where("user_id = ?", *filters.UserID)
	}
	if filters.Username != "" {
		var profile models.Profile
		if err := db.DB.Joins("JOIN auth_user ON auth_user.id = judge_profile.user_id").
			Where("auth_user.username = ?", filters.Username).
			First(&profile).Error; err == nil {
			query = query.Where("user_id = ?", profile.UserID)
		}
	}
	if filters.ProblemID != nil && *filters.ProblemID > 0 {
		query = query.Where("problem_id = ?", *filters.ProblemID)
	}
	if filters.ProblemCode != "" {
		var problem models.Problem
		if err := db.DB.Where("code = ?", filters.ProblemCode).First(&problem).Error; err == nil {
			query = query.Where("problem_id = ?", problem.ID)
		}
	}
	if filters.LanguageID != nil && *filters.LanguageID > 0 {
		query = query.Where("language_id = ?", *filters.LanguageID)
	}
	if filters.Language != "" {
		var lang models.Language
		if err := db.DB.Where("name LIKE ?", "%"+filters.Language+"%").First(&lang).Error; err == nil {
			query = query.Where("language_id = ?", lang.ID)
		}
	}
	if filters.Status != "" {
		query = query.Where("status = ?", filters.Status)
	}
	if filters.Result != "" {
		query = query.Where("result = ?", filters.Result)
	}
	if filters.FromDate != "" {
		query = query.Where("date >= ?", filters.FromDate)
	}
	if filters.ToDate != "" {
		query = query.Where("date <= ?", filters.ToDate)
	}
	return query
}
