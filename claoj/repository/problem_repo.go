package repository

import (
	"context"
	"time"

	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"gorm.io/gorm"
)

// GormProblemRepo is the GORM implementation of ProblemRepo.
type GormProblemRepo struct {
	db *gorm.DB
}

// NewGormProblemRepo creates a new GormProblemRepo.
func NewGormProblemRepo(database *gorm.DB) *GormProblemRepo {
	if database == nil {
		database = db.DB
	}
	return &GormProblemRepo{db: database}
}

// GetByID retrieves a problem by its ID.
func (r *GormProblemRepo) GetByID(ctx context.Context, id uint) (*models.Problem, error) {
	var problem models.Problem
	if err := r.db.WithContext(ctx).First(&problem, id).Error; err != nil {
		return nil, err
	}
	return &problem, nil
}

// GetByCode retrieves a problem by its code.
func (r *GormProblemRepo) GetByCode(ctx context.Context, code string) (*models.Problem, error) {
	var problem models.Problem
	if err := r.db.WithContext(ctx).Where("code = ?", code).First(&problem).Error; err != nil {
		return nil, err
	}
	return &problem, nil
}

// Create creates a new problem.
func (r *GormProblemRepo) Create(ctx context.Context, problem *models.Problem) error {
	return r.db.WithContext(ctx).Create(problem).Error
}

// Update updates an existing problem.
func (r *GormProblemRepo) Update(ctx context.Context, problem *models.Problem) error {
	return r.db.WithContext(ctx).Save(problem).Error
}

// Delete soft-deletes a problem by setting IsPublic to false.
func (r *GormProblemRepo) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Model(&models.Problem{}).Where("id = ?", id).Update("is_public", false).Error
}

// List retrieves a paginated list of problems.
func (r *GormProblemRepo) List(ctx context.Context, offset, limit int, publicOnly bool) ([]models.Problem, int64, error) {
	var problems []models.Problem
	var total int64

	query := r.db.WithContext(ctx).Model(&models.Problem{})
	if publicOnly {
		query = query.Where("is_public = ?", true)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.Order("code ASC").Offset(offset).Limit(limit).Find(&problems).Error; err != nil {
		return nil, 0, err
	}

	return problems, total, nil
}

// Search searches for problems by code or name.
func (r *GormProblemRepo) Search(ctx context.Context, query string, offset, limit int, publicOnly bool) ([]models.Problem, int64, error) {
	var problems []models.Problem
	var total int64

	searchPattern := "%" + query + "%"

	dbQuery := r.db.WithContext(ctx).Model(&models.Problem{})
	if publicOnly {
		dbQuery = dbQuery.Where("is_public = ?", true)
	}

	if err := dbQuery.Where("code LIKE ? OR name LIKE ?", searchPattern, searchPattern).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := dbQuery.Where("code LIKE ? OR name LIKE ?", searchPattern, searchPattern).
		Order("code ASC").Offset(offset).Limit(limit).Find(&problems).Error; err != nil {
		return nil, 0, err
	}

	return problems, total, nil
}

// ListByGroup retrieves problems filtered by group ID.
func (r *GormProblemRepo) ListByGroup(ctx context.Context, groupID uint, offset, limit int) ([]models.Problem, int64, error) {
	var problems []models.Problem
	var total int64

	if err := r.db.WithContext(ctx).Model(&models.Problem{}).
		Where("group_id = ?", groupID).
		Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := r.db.WithContext(ctx).
		Where("group_id = ?", groupID).
		Order("code ASC").Offset(offset).Limit(limit).Find(&problems).Error; err != nil {
		return nil, 0, err
	}

	return problems, total, nil
}

// GetSolvedProblems retrieves IDs of problems solved by a user.
func (r *GormProblemRepo) GetSolvedProblems(ctx context.Context, userID uint) ([]uint, error) {
	var problemIDs []uint

	if err := r.db.WithContext(ctx).
		Table("judge_submission").
		Where("user_id = ? AND result = 'AC'", userID).
		Distinct("problem_id").
		Pluck("problem_id", &problemIDs).Error; err != nil {
		return nil, err
	}

	return problemIDs, nil
}

// GormSubmissionRepo is the GORM implementation of SubmissionRepo.
type GormSubmissionRepo struct {
	db *gorm.DB
}

// NewGormSubmissionRepo creates a new GormSubmissionRepo.
func NewGormSubmissionRepo(database *gorm.DB) *GormSubmissionRepo {
	if database == nil {
		database = db.DB
	}
	return &GormSubmissionRepo{db: database}
}

// GetByID retrieves a submission by its ID.
func (r *GormSubmissionRepo) GetByID(ctx context.Context, id uint) (*models.Submission, error) {
	var submission models.Submission
	if err := r.db.WithContext(ctx).First(&submission, id).Error; err != nil {
		return nil, err
	}
	return &submission, nil
}

// Create creates a new submission.
func (r *GormSubmissionRepo) Create(ctx context.Context, submission *models.Submission) error {
	return r.db.WithContext(ctx).Create(submission).Error
}

// Update updates an existing submission.
func (r *GormSubmissionRepo) Update(ctx context.Context, submission *models.Submission) error {
	return r.db.WithContext(ctx).Save(submission).Error
}

// List retrieves a paginated list of submissions.
func (r *GormSubmissionRepo) List(ctx context.Context, userID, problemID *uint, offset, limit int) ([]models.Submission, int64, error) {
	var submissions []models.Submission
	var total int64

	query := r.db.WithContext(ctx).Model(&models.Submission{})

	if userID != nil {
		query = query.Where("user_id = ?", *userID)
	}
	if problemID != nil {
		query = query.Where("problem_id = ?", *problemID)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.Order("id DESC").Offset(offset).Limit(limit).Find(&submissions).Error; err != nil {
		return nil, 0, err
	}

	return submissions, total, nil
}

// GetUserBestSubmissions retrieves the best submission for each problem by a user.
func (r *GormSubmissionRepo) GetUserBestSubmissions(ctx context.Context, userID uint) ([]models.Submission, error) {
	var submissions []models.Submission

	// Get the best (highest points, then fastest) submission for each problem
	query := `
		SELECT s.* FROM judge_submission s
		INNER JOIN (
			SELECT problem_id, MAX(points) as max_points
			FROM judge_submission
			WHERE user_id = ? AND result = 'AC'
			GROUP BY problem_id
		) best ON s.problem_id = best.problem_id AND s.points = best.max_points
		WHERE s.user_id = ?
	`

	if err := r.db.WithContext(ctx).Raw(query, userID, userID).Find(&submissions).Error; err != nil {
		return nil, err
	}

	return submissions, nil
}

// GetRecentSubmissions retrieves recent submissions with problem and user info.
func (r *GormSubmissionRepo) GetRecentSubmissions(ctx context.Context, limit int) ([]models.Submission, error) {
	var submissions []models.Submission

	if err := r.db.WithContext(ctx).
		Preload("User").
		Preload("Problem").
		Order("id DESC").
		Limit(limit).
		Find(&submissions).Error; err != nil {
		return nil, err
	}

	return submissions, nil
}

// GormContestRepo is the GORM implementation of ContestRepo.
type GormContestRepo struct {
	db *gorm.DB
}

// NewGormContestRepo creates a new GormContestRepo.
func NewGormContestRepo(database *gorm.DB) *GormContestRepo {
	if database == nil {
		database = db.DB
	}
	return &GormContestRepo{db: database}
}

// GetByID retrieves a contest by its ID.
func (r *GormContestRepo) GetByID(ctx context.Context, id uint) (*models.Contest, error) {
	var contest models.Contest
	if err := r.db.WithContext(ctx).First(&contest, id).Error; err != nil {
		return nil, err
	}
	return &contest, nil
}

// GetByKey retrieves a contest by its key.
func (r *GormContestRepo) GetByKey(ctx context.Context, key string) (*models.Contest, error) {
	var contest models.Contest
	if err := r.db.WithContext(ctx).Where("key = ?", key).First(&contest).Error; err != nil {
		return nil, err
	}
	return &contest, nil
}

// Create creates a new contest.
func (r *GormContestRepo) Create(ctx context.Context, contest *models.Contest) error {
	return r.db.WithContext(ctx).Create(contest).Error
}

// Update updates an existing contest.
func (r *GormContestRepo) Update(ctx context.Context, contest *models.Contest) error {
	return r.db.WithContext(ctx).Save(contest).Error
}

// Delete soft-deletes a contest by setting IsVisible to false.
func (r *GormContestRepo) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Model(&models.Contest{}).Where("id = ?", id).Update("is_visible", false).Error
}

// List retrieves a paginated list of contests.
func (r *GormContestRepo) List(ctx context.Context, offset, limit int, publicOnly bool) ([]models.Contest, int64, error) {
	var contests []models.Contest
	var total int64

	query := r.db.WithContext(ctx).Model(&models.Contest{})
	if publicOnly {
		query = query.Where("is_visible = ?", true)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.Order("start_time DESC").Offset(offset).Limit(limit).Find(&contests).Error; err != nil {
		return nil, 0, err
	}

	return contests, total, nil
}

// ListActive retrieves currently active contests.
func (r *GormContestRepo) ListActive(ctx context.Context) ([]models.Contest, error) {
	var contests []models.Contest
	now := time.Now()

	if err := r.db.WithContext(ctx).
		Where("is_visible = ? AND start_time <= ? AND end_time >= ?", true, now, now).
		Find(&contests).Error; err != nil {
		return nil, err
	}

	return contests, nil
}

// ListUpcoming retrieves upcoming contests.
func (r *GormContestRepo) ListUpcoming(ctx context.Context, limit int) ([]models.Contest, error) {
	var contests []models.Contest
	now := time.Now()

	if err := r.db.WithContext(ctx).
		Where("is_visible = ? AND start_time > ?", true, now).
		Order("start_time ASC").
		Limit(limit).
		Find(&contests).Error; err != nil {
		return nil, err
	}

	return contests, nil
}
