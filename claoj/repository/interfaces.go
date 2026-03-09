// Package repository provides repository interfaces and implementations for data access.
// This abstraction layer enables better testability through mocking and interface-based design.
package repository

import (
	"context"

	"github.com/CLAOJ/claoj/models"
)

// UserRepo defines the interface for user data access.
type UserRepo interface {
	// GetByID retrieves a user by their ID.
	GetByID(ctx context.Context, id uint) (*models.AuthUser, error)
	// GetByUsername retrieves a user by their username.
	GetByUsername(ctx context.Context, username string) (*models.AuthUser, error)
	// GetByEmail retrieves a user by their email address.
	GetByEmail(ctx context.Context, email string) (*models.AuthUser, error)
	// Create creates a new user.
	Create(ctx context.Context, user *models.AuthUser) error
	// Update updates an existing user.
	Update(ctx context.Context, user *models.AuthUser) error
	// Delete soft-deletes a user by setting IsActive to false.
	Delete(ctx context.Context, id uint) error
	// List retrieves a paginated list of users.
	List(ctx context.Context, offset, limit int) ([]models.AuthUser, int64, error)
	// Search searches for users by username or email.
	Search(ctx context.Context, query string, offset, limit int) ([]models.AuthUser, int64, error)
}

// ProfileRepo defines the interface for profile data access.
type ProfileRepo interface {
	// GetByUserID retrieves a profile by user ID.
	GetByUserID(ctx context.Context, userID uint) (*models.Profile, error)
	// GetByUsername retrieves a profile by username (joined with user).
	GetByUsername(ctx context.Context, username string) (*models.Profile, error)
	// Create creates a new profile.
	Create(ctx context.Context, profile *models.Profile) error
	// Update updates an existing profile.
	Update(ctx context.Context, profile *models.Profile) error
	// List retrieves a paginated list of profiles.
	List(ctx context.Context, offset, limit int, excludeUnlisted bool) ([]models.Profile, int64, error)
	// GetTopUsers retrieves top users by performance points.
	GetTopUsers(ctx context.Context, limit int) ([]models.Profile, error)
	// UpdateRating updates a user's rating.
	UpdateRating(ctx context.Context, userID uint, rating int) error
}

// ProblemRepo defines the interface for problem data access.
type ProblemRepo interface {
	// GetByID retrieves a problem by its ID.
	GetByID(ctx context.Context, id uint) (*models.Problem, error)
	// GetByCode retrieves a problem by its code.
	GetByCode(ctx context.Context, code string) (*models.Problem, error)
	// Create creates a new problem.
	Create(ctx context.Context, problem *models.Problem) error
	// Update updates an existing problem.
	Update(ctx context.Context, problem *models.Problem) error
	// Delete soft-deletes a problem by setting IsPublic to false.
	Delete(ctx context.Context, id uint) error
	// List retrieves a paginated list of problems.
	List(ctx context.Context, offset, limit int, publicOnly bool) ([]models.Problem, int64, error)
	// Search searches for problems by code or name.
	Search(ctx context.Context, query string, offset, limit int, publicOnly bool) ([]models.Problem, int64, error)
	// ListByGroup retrieves problems filtered by group ID.
	ListByGroup(ctx context.Context, groupID uint, offset, limit int) ([]models.Problem, int64, error)
	// GetSolvedProblems retrieves IDs of problems solved by a user.
	GetSolvedProblems(ctx context.Context, userID uint) ([]uint, error)
}

// SubmissionRepo defines the interface for submission data access.
type SubmissionRepo interface {
	// GetByID retrieves a submission by its ID.
	GetByID(ctx context.Context, id uint) (*models.Submission, error)
	// Create creates a new submission.
	Create(ctx context.Context, submission *models.Submission) error
	// Update updates an existing submission.
	Update(ctx context.Context, submission *models.Submission) error
	// List retrieves a paginated list of submissions.
	List(ctx context.Context, userID, problemID *uint, offset, limit int) ([]models.Submission, int64, error)
	// GetUserBestSubmissions retrieves the best submission for each problem by a user.
	GetUserBestSubmissions(ctx context.Context, userID uint) ([]models.Submission, error)
	// GetRecentSubmissions retrieves recent submissions with problem and user info.
	GetRecentSubmissions(ctx context.Context, limit int) ([]models.Submission, error)
}

// ContestRepo defines the interface for contest data access.
type ContestRepo interface {
	// GetByID retrieves a contest by its ID.
	GetByID(ctx context.Context, id uint) (*models.Contest, error)
	// GetByKey retrieves a contest by its key.
	GetByKey(ctx context.Context, key string) (*models.Contest, error)
	// Create creates a new contest.
	Create(ctx context.Context, contest *models.Contest) error
	// Update updates an existing contest.
	Update(ctx context.Context, contest *models.Contest) error
	// Delete soft-deletes a contest by setting IsVisible to false.
	Delete(ctx context.Context, id uint) error
	// List retrieves a paginated list of contests.
	List(ctx context.Context, offset, limit int, publicOnly bool) ([]models.Contest, int64, error)
	// ListActive retrieves currently active contests.
	ListActive(ctx context.Context) ([]models.Contest, error)
	// ListUpcoming retrieves upcoming contests.
	ListUpcoming(ctx context.Context, limit int) ([]models.Contest, error)
}

// OrganizationRepo defines the interface for organization data access.
type OrganizationRepo interface {
	// GetByID retrieves an organization by its ID.
	GetByID(ctx context.Context, id uint) (*models.Organization, error)
	// GetBySlug retrieves an organization by its slug.
	GetBySlug(ctx context.Context, slug string) (*models.Organization, error)
	// Create creates a new organization.
	Create(ctx context.Context, org *models.Organization) error
	// Update updates an existing organization.
	Update(ctx context.Context, org *models.Organization) error
	// List retrieves a paginated list of organizations.
	List(ctx context.Context, offset, limit int) ([]models.Organization, int64, error)
}

// CommentRepo defines the interface for comment data access.
type CommentRepo interface {
	// GetByID retrieves a comment by its ID.
	GetByID(ctx context.Context, id uint) (*models.Comment, error)
	// Create creates a new comment.
	Create(ctx context.Context, comment *models.Comment) error
	// Update updates an existing comment.
	Update(ctx context.Context, comment *models.Comment) error
	// Delete soft-deletes a comment.
	Delete(ctx context.Context, id uint) error
	// ListByPath retrieves comments filtered by path (e.g., problem path).
	ListByPath(ctx context.Context, path string, offset, limit int) ([]models.Comment, int64, error)
	// ListByAuthor retrieves comments by a specific author.
	ListByAuthor(ctx context.Context, authorID uint, offset, limit int) ([]models.Comment, int64, error)
	// GetChildren retrieves child comments of a parent comment.
	GetChildren(ctx context.Context, parentID uint) ([]models.Comment, error)
}

// Ensure compile-time interface compliance
var (
	_ UserRepo       = (*GormUserRepo)(nil)
	_ ProfileRepo    = (*GormProfileRepo)(nil)
	_ ProblemRepo    = (*GormProblemRepo)(nil)
	_ SubmissionRepo = (*GormSubmissionRepo)(nil)
	_ ContestRepo    = (*GormContestRepo)(nil)
)
