package repository

import (
	"context"
	"testing"

	"github.com/CLAOJ/claoj/models"
	"github.com/stretchr/testify/assert"
)

func TestMockUserRepo_GetByID(t *testing.T) {
	repo := NewMockUserRepo()
	ctx := context.Background()

	// Create a test user
	user := &models.AuthUser{
		ID:       1,
		Username: "testuser",
		Email:    "test@example.com",
		IsActive: true,
	}
	repo.Users[1] = user

	// Test successful get
	found, err := repo.GetByID(ctx, 1)
	assert.NoError(t, err)
	assert.Equal(t, "testuser", found.Username)

	// Test not found
	_, err = repo.GetByID(ctx, 999)
	assert.Error(t, err)
	assert.Equal(t, ErrNotFound, err)
}

func TestMockUserRepo_GetByUsername(t *testing.T) {
	repo := NewMockUserRepo()
	ctx := context.Background()

	user := &models.AuthUser{
		ID:       1,
		Username: "testuser",
		Email:    "test@example.com",
	}
	repo.Users[1] = user

	found, err := repo.GetByUsername(ctx, "testuser")
	assert.NoError(t, err)
	assert.Equal(t, uint(1), found.ID)

	_, err = repo.GetByUsername(ctx, "nonexistent")
	assert.Error(t, err)
}

func TestMockUserRepo_Create(t *testing.T) {
	repo := NewMockUserRepo()
	ctx := context.Background()

	user := &models.AuthUser{
		ID:       1,
		Username: "newuser",
		Email:    "new@example.com",
	}

	err := repo.Create(ctx, user)
	assert.NoError(t, err)
	assert.Len(t, repo.Users, 1)
	assert.Equal(t, "newuser", repo.Users[1].Username)
}

func TestMockUserRepo_Update(t *testing.T) {
	repo := NewMockUserRepo()
	ctx := context.Background()

	user := &models.AuthUser{
		ID:       1,
		Username: "oldname",
		Email:    "old@example.com",
	}
	repo.Users[1] = user

	user.Username = "newname"
	err := repo.Update(ctx, user)
	assert.NoError(t, err)
	assert.Equal(t, "newname", repo.Users[1].Username)

	// Update non-existent
	user.ID = 999
	err = repo.Update(ctx, user)
	assert.Error(t, err)
}

func TestMockUserRepo_Delete(t *testing.T) {
	repo := NewMockUserRepo()
	ctx := context.Background()

	user := &models.AuthUser{
		ID:       1,
		Username: "testuser",
		IsActive: true,
	}
	repo.Users[1] = user

	err := repo.Delete(ctx, 1)
	assert.NoError(t, err)
	assert.False(t, repo.Users[1].IsActive)

	err = repo.Delete(ctx, 999)
	assert.Error(t, err)
}

func TestMockUserRepo_List(t *testing.T) {
	repo := NewMockUserRepo()
	ctx := context.Background()

	// Add 5 users
	for i := uint(1); i <= 5; i++ {
		repo.Users[i] = &models.AuthUser{
			ID:       i,
			Username: "user" + string(rune('0'+i)),
		}
	}

	users, total, err := repo.List(ctx, 0, 10)
	assert.NoError(t, err)
	assert.Equal(t, int64(5), total)
	assert.Len(t, users, 5)

	// Test pagination
	users, total, err = repo.List(ctx, 2, 2)
	assert.NoError(t, err)
	assert.Equal(t, int64(5), total)
	assert.Len(t, users, 2)
}

func TestMockUserRepo_Search(t *testing.T) {
	repo := NewMockUserRepo()
	ctx := context.Background()

	repo.Users[1] = &models.AuthUser{ID: 1, Username: "alice", Email: "alice@example.com"}
	repo.Users[2] = &models.AuthUser{ID: 2, Username: "bob", Email: "bob@example.com"}
	repo.Users[3] = &models.AuthUser{ID: 3, Username: "charlie", Email: "charlie@example.com"}

	results, total, err := repo.Search(ctx, "ali", 0, 10)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, results, 1)
	assert.Equal(t, "alice", results[0].Username)

	results, total, err = repo.Search(ctx, "example", 0, 10)
	assert.NoError(t, err)
	assert.Equal(t, int64(3), total)
}

func TestMockProfileRepo_GetTopUsers(t *testing.T) {
	repo := NewMockProfileRepo()
	ctx := context.Background()

	rating1, rating2, rating3 := 1500, 2000, 1800
	repo.Profiles[1] = &models.Profile{ID: 1, User: models.AuthUser{Username: "user1"}, PerformancePoints: 100, Rating: &rating1}
	repo.Profiles[2] = &models.Profile{ID: 2, User: models.AuthUser{Username: "user2"}, PerformancePoints: 300, Rating: &rating2}
	repo.Profiles[3] = &models.Profile{ID: 3, User: models.AuthUser{Username: "user3"}, PerformancePoints: 200, Rating: &rating3}

	top, err := repo.GetTopUsers(ctx, 2)
	assert.NoError(t, err)
	assert.Len(t, top, 2)
	// Should be sorted by performance points descending
	assert.Equal(t, "user2", top[0].User.Username)
	assert.Equal(t, "user3", top[1].User.Username)
}

func TestMockProblemRepo_GetByCode(t *testing.T) {
	repo := NewMockProblemRepo()
	ctx := context.Background()

	problem := &models.Problem{
		ID:       1,
		Code:     "TEST",
		Name:     "Test Problem",
		IsPublic: true,
	}
	repo.Problems[1] = problem

	found, err := repo.GetByCode(ctx, "TEST")
	assert.NoError(t, err)
	assert.Equal(t, "Test Problem", found.Name)

	_, err = repo.GetByCode(ctx, "NOTFOUND")
	assert.Error(t, err)
}

func TestMockProblemRepo_List_PublicOnly(t *testing.T) {
	repo := NewMockProblemRepo()
	ctx := context.Background()

	repo.Problems[1] = &models.Problem{ID: 1, Code: "PUBLIC", IsPublic: true}
	repo.Problems[2] = &models.Problem{ID: 2, Code: "PRIVATE", IsPublic: false}

	problems, total, err := repo.List(ctx, 0, 10, true)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, problems, 1)
	assert.Equal(t, "PUBLIC", problems[0].Code)

	// Include private
	problems, total, err = repo.List(ctx, 0, 10, false)
	assert.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Len(t, problems, 2)
}

func TestMockProblemRepo_Search(t *testing.T) {
	repo := NewMockProblemRepo()
	ctx := context.Background()

	repo.Problems[1] = &models.Problem{ID: 1, Code: "ABC", Name: "Alice's Problem"}
	repo.Problems[2] = &models.Problem{ID: 2, Code: "DEF", Name: "Bob's Problem"}

	results, total, err := repo.Search(ctx, "alice", 0, 10, false)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, results, 1)

	results, total, err = repo.Search(ctx, "problem", 0, 10, false)
	assert.NoError(t, err)
	assert.Equal(t, int64(2), total)
}
