package repository

import (
	"context"

	"github.com/CLAOJ/claoj/models"
)

// MockUserRepo is a mock implementation of UserRepo for testing.
type MockUserRepo struct {
	Users map[uint]*models.AuthUser
}

// NewMockUserRepo creates a new MockUserRepo.
func NewMockUserRepo() *MockUserRepo {
	return &MockUserRepo{
		Users: make(map[uint]*models.AuthUser),
	}
}

// GetByID retrieves a user by their ID.
func (m *MockUserRepo) GetByID(ctx context.Context, id uint) (*models.AuthUser, error) {
	user, ok := m.Users[id]
	if !ok {
		return nil, ErrNotFound
	}
	return user, nil
}

// GetByUsername retrieves a user by their username.
func (m *MockUserRepo) GetByUsername(ctx context.Context, username string) (*models.AuthUser, error) {
	for _, user := range m.Users {
		if user.Username == username {
			return user, nil
		}
	}
	return nil, ErrNotFound
}

// GetByEmail retrieves a user by their email address.
func (m *MockUserRepo) GetByEmail(ctx context.Context, email string) (*models.AuthUser, error) {
	for _, user := range m.Users {
		if user.Email == email {
			return user, nil
		}
	}
	return nil, ErrNotFound
}

// Create creates a new user.
func (m *MockUserRepo) Create(ctx context.Context, user *models.AuthUser) error {
	m.Users[user.ID] = user
	return nil
}

// Update updates an existing user.
func (m *MockUserRepo) Update(ctx context.Context, user *models.AuthUser) error {
	if _, ok := m.Users[user.ID]; !ok {
		return ErrNotFound
	}
	m.Users[user.ID] = user
	return nil
}

// Delete soft-deletes a user by setting IsActive to false.
func (m *MockUserRepo) Delete(ctx context.Context, id uint) error {
	user, ok := m.Users[id]
	if !ok {
		return ErrNotFound
	}
	user.IsActive = false
	return nil
}

// List retrieves a paginated list of users.
func (m *MockUserRepo) List(ctx context.Context, offset, limit int) ([]models.AuthUser, int64, error) {
	users := make([]models.AuthUser, 0, len(m.Users))
	for _, user := range m.Users {
		users = append(users, *user)
	}

	total := int64(len(users))

	start := offset
	if start > len(users) {
		start = len(users)
	}
	end := start + limit
	if end > len(users) {
		end = len(users)
	}

	return users[start:end], total, nil
}

// Search searches for users by username or email.
func (m *MockUserRepo) Search(ctx context.Context, query string, offset, limit int) ([]models.AuthUser, int64, error) {
	var results []models.AuthUser

	for _, user := range m.Users {
		if containsIgnoreCase(user.Username, query) || containsIgnoreCase(user.Email, query) {
			results = append(results, *user)
		}
	}

	total := int64(len(results))

	start := offset
	if start > len(results) {
		start = len(results)
	}
	end := start + limit
	if end > len(results) {
		end = len(results)
	}

	return results[start:end], total, nil
}

// MockProfileRepo is a mock implementation of ProfileRepo for testing.
type MockProfileRepo struct {
	Profiles map[uint]*models.Profile
}

// NewMockProfileRepo creates a new MockProfileRepo.
func NewMockProfileRepo() *MockProfileRepo {
	return &MockProfileRepo{
		Profiles: make(map[uint]*models.Profile),
	}
}

// GetByUserID retrieves a profile by user ID.
func (m *MockProfileRepo) GetByUserID(ctx context.Context, userID uint) (*models.Profile, error) {
	profile, ok := m.Profiles[userID]
	if !ok {
		return nil, ErrNotFound
	}
	return profile, nil
}

// GetByUsername retrieves a profile by username.
func (m *MockProfileRepo) GetByUsername(ctx context.Context, username string) (*models.Profile, error) {
	// Simplified mock implementation
	for _, profile := range m.Profiles {
		if profile.User.Username == username {
			return profile, nil
		}
	}
	return nil, ErrNotFound
}

// Create creates a new profile.
func (m *MockProfileRepo) Create(ctx context.Context, profile *models.Profile) error {
	m.Profiles[profile.ID] = profile
	return nil
}

// Update updates an existing profile.
func (m *MockProfileRepo) Update(ctx context.Context, profile *models.Profile) error {
	if _, ok := m.Profiles[profile.ID]; !ok {
		return ErrNotFound
	}
	m.Profiles[profile.ID] = profile
	return nil
}

// List retrieves a paginated list of profiles.
func (m *MockProfileRepo) List(ctx context.Context, offset, limit int, excludeUnlisted bool) ([]models.Profile, int64, error) {
	profiles := make([]models.Profile, 0, len(m.Profiles))
	for _, profile := range m.Profiles {
		if excludeUnlisted && profile.IsUnlisted {
			continue
		}
		profiles = append(profiles, *profile)
	}

	total := int64(len(profiles))

	start := offset
	if start > len(profiles) {
		start = len(profiles)
	}
	end := start + limit
	if end > len(profiles) {
		end = len(profiles)
	}

	return profiles[start:end], total, nil
}

// GetTopUsers retrieves top users by performance points.
func (m *MockProfileRepo) GetTopUsers(ctx context.Context, limit int) ([]models.Profile, error) {
	profiles := make([]models.Profile, 0, len(m.Profiles))
	for _, profile := range m.Profiles {
		profiles = append(profiles, *profile)
	}

	// Sort by performance points (simplified)
	for i := 0; i < len(profiles) && i < limit; i++ {
		for j := i + 1; j < len(profiles); j++ {
			if profiles[j].PerformancePoints > profiles[i].PerformancePoints {
				profiles[i], profiles[j] = profiles[j], profiles[i]
			}
		}
	}

	if limit < len(profiles) {
		profiles = profiles[:limit]
	}

	return profiles, nil
}

// UpdateRating updates a user's rating.
func (m *MockProfileRepo) UpdateRating(ctx context.Context, userID uint, rating int) error {
	profile, ok := m.Profiles[userID]
	if !ok {
		return ErrNotFound
	}
	profile.Rating = &rating
	return nil
}

// MockProblemRepo is a mock implementation of ProblemRepo for testing.
type MockProblemRepo struct {
	Problems map[uint]*models.Problem
}

// NewMockProblemRepo creates a new MockProblemRepo.
func NewMockProblemRepo() *MockProblemRepo {
	return &MockProblemRepo{
		Problems: make(map[uint]*models.Problem),
	}
}

// GetByID retrieves a problem by its ID.
func (m *MockProblemRepo) GetByID(ctx context.Context, id uint) (*models.Problem, error) {
	problem, ok := m.Problems[id]
	if !ok {
		return nil, ErrNotFound
	}
	return problem, nil
}

// GetByCode retrieves a problem by its code.
func (m *MockProblemRepo) GetByCode(ctx context.Context, code string) (*models.Problem, error) {
	for _, problem := range m.Problems {
		if problem.Code == code {
			return problem, nil
		}
	}
	return nil, ErrNotFound
}

// Create creates a new problem.
func (m *MockProblemRepo) Create(ctx context.Context, problem *models.Problem) error {
	m.Problems[problem.ID] = problem
	return nil
}

// Update updates an existing problem.
func (m *MockProblemRepo) Update(ctx context.Context, problem *models.Problem) error {
	if _, ok := m.Problems[problem.ID]; !ok {
		return ErrNotFound
	}
	m.Problems[problem.ID] = problem
	return nil
}

// Delete soft-deletes a problem by setting IsPublic to false.
func (m *MockProblemRepo) Delete(ctx context.Context, id uint) error {
	problem, ok := m.Problems[id]
	if !ok {
		return ErrNotFound
	}
	problem.IsPublic = false
	return nil
}

// List retrieves a paginated list of problems.
func (m *MockProblemRepo) List(ctx context.Context, offset, limit int, publicOnly bool) ([]models.Problem, int64, error) {
	problems := make([]models.Problem, 0, len(m.Problems))
	for _, problem := range m.Problems {
		if publicOnly && !problem.IsPublic {
			continue
		}
		problems = append(problems, *problem)
	}

	total := int64(len(problems))

	start := offset
	if start > len(problems) {
		start = len(problems)
	}
	end := start + limit
	if end > len(problems) {
		end = len(problems)
	}

	return problems[start:end], total, nil
}

// Search searches for problems by code or name.
func (m *MockProblemRepo) Search(ctx context.Context, query string, offset, limit int, publicOnly bool) ([]models.Problem, int64, error) {
	var results []models.Problem

	for _, problem := range m.Problems {
		if publicOnly && !problem.IsPublic {
			continue
		}
		if containsIgnoreCase(problem.Code, query) || containsIgnoreCase(problem.Name, query) {
			results = append(results, *problem)
		}
	}

	total := int64(len(results))

	start := offset
	if start > len(results) {
		start = len(results)
	}
	end := start + limit
	if end > len(results) {
		end = len(results)
	}

	return results[start:end], total, nil
}

// ListByGroup retrieves problems filtered by group ID.
func (m *MockProblemRepo) ListByGroup(ctx context.Context, groupID uint, offset, limit int) ([]models.Problem, int64, error) {
	var problems []models.Problem

	for _, problem := range m.Problems {
		if problem.GroupID == groupID {
			problems = append(problems, *problem)
		}
	}

	total := int64(len(problems))

	start := offset
	if start > len(problems) {
		start = len(problems)
	}
	end := start + limit
	if end > len(problems) {
		end = len(problems)
	}

	return problems[start:end], total, nil
}

// GetSolvedProblems retrieves IDs of problems solved by a user.
func (m *MockProblemRepo) GetSolvedProblems(ctx context.Context, userID uint) ([]uint, error) {
	// Simplified mock - returns all problem IDs
	ids := make([]uint, 0, len(m.Problems))
	for id := range m.Problems {
		ids = append(ids, id)
	}
	return ids, nil
}

// Helper functions

func containsIgnoreCase(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || containsLower(s, substr))
}

func containsLower(s, substr string) bool {
	s = toLower(s)
	substr = toLower(substr)
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func toLower(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c = c + 32
		}
		result[i] = byte(c)
	}
	return string(result)
}
