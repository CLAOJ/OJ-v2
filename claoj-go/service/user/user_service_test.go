package user

import (
	"testing"
	"time"

	"github.com/CLAOJ/claoj-go/db"
	"github.com/CLAOJ/claoj-go/models"
	"github.com/stretchr/testify/suite"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type UserServiceTestSuite struct {
	suite.Suite
	database *gorm.DB
	service  *UserService
}

func (s *UserServiceTestSuite) SetupTest() {
	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	s.Require().NoError(err)

	s.database = database
	db.DB = database

	// Migrate schema
	err = database.AutoMigrate(
		&models.AuthUser{},
		&models.Profile{},
		&models.Organization{},
		&models.RefreshToken{},
	)
	s.Require().NoError(err)

	s.service = NewUserService()
}

func (s *UserServiceTestSuite) createTestUser(username string, isActive bool) (models.AuthUser, models.Profile) {
	user := models.AuthUser{
		Username:   username,
		Email:      username + "@example.com",
		Password:   "hashedpassword",
		IsActive:   isActive,
		IsStaff:    false,
		DateJoined: time.Now(),
	}
	err := s.database.Create(&user).Error
	s.Require().NoError(err)

	profile := models.Profile{
		UserID:     user.ID,
		Timezone:   "UTC",
		LastAccess: time.Now(),
	}
	err = s.database.Create(&profile).Error
	s.Require().NoError(err)

	return user, profile
}

func (s *UserServiceTestSuite) createTestOrganization(name string) models.Organization {
	org := models.Organization{
		Name:   name,
		Slug:   name,
		IsOpen: true,
	}
	err := s.database.Create(&org).Error
	s.Require().NoError(err)
	return org
}

func TestUserServiceTestSuite(t *testing.T) {
	suite.Run(t, new(UserServiceTestSuite))
}

func (s *UserServiceTestSuite) TestBanUser() {
	user, _ := s.createTestUser("testuser", true)

	req := BanUserRequest{
		UserID: user.ID,
		Reason: "Spam behavior",
	}

	err := s.service.BanUser(req)
	s.NoError(err)

	// Verify user is banned
	var profile models.Profile
	err = s.database.Where("user_id = ?", user.ID).First(&profile).Error
	s.NoError(err)
	s.True(profile.IsUnlisted)
	s.True(profile.Mute)
	s.NotNil(profile.BanReason)
	s.Equal("Spam behavior", *profile.BanReason)
}

func (s *UserServiceTestSuite) TestBanUser_InvalidUserID() {
	req := BanUserRequest{
		UserID: 0,
		Reason: "Test",
	}
	err := s.service.BanUser(req)
	s.ErrorIs(err, ErrInvalidUserID)
}

func (s *UserServiceTestSuite) TestBanUser_InvalidReason() {
	user, _ := s.createTestUser("testuser", true)

	req := BanUserRequest{
		UserID: user.ID,
		Reason: "",
	}
	err := s.service.BanUser(req)
	s.ErrorIs(err, ErrInvalidReason)
}

func (s *UserServiceTestSuite) TestBanUser_UserNotFound() {
	req := BanUserRequest{
		UserID: 99999,
		Reason: "Test reason",
	}
	err := s.service.BanUser(req)
	s.ErrorIs(err, ErrUserNotFound)
}

func (s *UserServiceTestSuite) TestUnbanUser() {
	user, profile := s.createTestUser("banneduser", true)
	// First ban the user
	profile.IsUnlisted = true
	profile.Mute = true
	reason := "Previous violation"
	profile.BanReason = &reason
	s.database.Save(&profile)

	req := UnbanUserRequest{UserID: user.ID}
	err := s.service.UnbanUser(req)
	s.NoError(err)

	// Verify user is unbanned
	var updated models.Profile
	err = s.database.Where("user_id = ?", user.ID).First(&updated).Error
	s.NoError(err)
	s.False(updated.IsUnlisted)
	s.False(updated.Mute)
	s.Nil(updated.BanReason)
}

func (s *UserServiceTestSuite) TestUnbanUser_InvalidUserID() {
	req := UnbanUserRequest{UserID: 0}
	err := s.service.UnbanUser(req)
	s.ErrorIs(err, ErrInvalidUserID)
}

func (s *UserServiceTestSuite) TestUnbanUser_UserNotFound() {
	req := UnbanUserRequest{UserID: 99999}
	err := s.service.UnbanUser(req)
	s.ErrorIs(err, ErrUserNotFound)
}

func (s *UserServiceTestSuite) TestGetUser() {
	user, _ := s.createTestUser("testuser", true)

	req := GetUserRequest{UserID: user.ID}
	profile, err := s.service.GetUser(req)
	s.NoError(err)
	s.NotNil(profile)
	s.Equal("testuser", profile.Username)
	s.Equal(user.Email, profile.Email)
	s.True(profile.IsActive)
}

func (s *UserServiceTestSuite) TestGetUser_InvalidUserID() {
	req := GetUserRequest{UserID: 0}
	profile, err := s.service.GetUser(req)
	s.ErrorIs(err, ErrInvalidUserID)
	s.Nil(profile)
}

func (s *UserServiceTestSuite) TestGetUser_UserNotFound() {
	req := GetUserRequest{UserID: 99999}
	profile, err := s.service.GetUser(req)
	s.ErrorIs(err, ErrUserNotFound)
	s.Nil(profile)
}

func (s *UserServiceTestSuite) TestUpdateUser() {
	user, _ := s.createTestUser("testuser", true)
	org := s.createTestOrganization("TestOrg")

	newDisplayName := "New Display Name"
	isActive := false

	req := UpdateUserRequest{
		UserID:                user.ID,
		DisplayName:           &newDisplayName,
		IsActive:              &isActive,
		AddOrganizationIDs:    []uint{org.ID},
	}

	err := s.service.UpdateUser(req)
	s.NoError(err)

	// Verify updates
	var profile models.Profile
	err = s.database.Preload("User").Preload("Organizations").Where("user_id = ?", user.ID).First(&profile).Error
	s.NoError(err)
	s.Equal("New Display Name", profile.UsernameDisplayOverride)
	s.False(profile.User.IsActive)
	s.Len(profile.Organizations, 1)
}

func (s *UserServiceTestSuite) TestUpdateUser_InvalidUserID() {
	req := UpdateUserRequest{UserID: 0}
	err := s.service.UpdateUser(req)
	s.ErrorIs(err, ErrInvalidUserID)
}

func (s *UserServiceTestSuite) TestUpdateUser_UserNotFound() {
	req := UpdateUserRequest{UserID: 99999}
	err := s.service.UpdateUser(req)
	s.ErrorIs(err, ErrUserNotFound)
}

func (s *UserServiceTestSuite) TestDeleteUser() {
	user, _ := s.createTestUser("todelete", true)

	req := DeleteUserRequest{UserID: user.ID}
	err := s.service.DeleteUser(req)
	s.NoError(err)

	// Verify soft delete (user deactivated and unlisted)
	var profile models.Profile
	err = s.database.Where("user_id = ?", user.ID).First(&profile).Error
	s.NoError(err)
	s.True(profile.IsUnlisted)

	// Verify user is deactivated
	var updatedUser models.AuthUser
	err = s.database.First(&updatedUser, user.ID).Error
	s.NoError(err)
	s.False(updatedUser.IsActive)
}

func (s *UserServiceTestSuite) TestDeleteUser_InvalidUserID() {
	req := DeleteUserRequest{UserID: 0}
	err := s.service.DeleteUser(req)
	s.ErrorIs(err, ErrInvalidUserID)
}

func (s *UserServiceTestSuite) TestDeleteUser_UserNotFound() {
	req := DeleteUserRequest{UserID: 99999}
	err := s.service.DeleteUser(req)
	s.ErrorIs(err, ErrUserNotFound)
}

func (s *UserServiceTestSuite) TestListUsers() {
	// Create test users
	for i := 0; i < 5; i++ {
		s.createTestUser("user"+string(rune('0'+i)), true)
	}

	req := ListUsersRequest{Page: 1, PageSize: 10}
	resp, err := s.service.ListUsers(req)
	s.NoError(err)
	s.NotNil(resp)
	s.Equal(int64(5), resp.Total)
	s.Len(resp.Users, 5)

	// Test pagination
	req = ListUsersRequest{Page: 1, PageSize: 3}
	resp, err = s.service.ListUsers(req)
	s.NoError(err)
	s.Len(resp.Users, 3)
}

func (s *UserServiceTestSuite) TestListUsers_PaginationDefaults() {
	req := ListUsersRequest{}
	resp, err := s.service.ListUsers(req)
	s.NoError(err)
	s.Equal(1, resp.Page)
	s.Equal(20, resp.PageSize)
}

func (s *UserServiceTestSuite) TestListUsers_MaxPageSize() {
	req := ListUsersRequest{PageSize: 200}
	resp, err := s.service.ListUsers(req)
	s.NoError(err)
	s.Equal(100, resp.PageSize)
}
