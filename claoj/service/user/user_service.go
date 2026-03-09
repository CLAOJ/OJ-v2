// Package user provides user management services.
package user

import (
	"time"

	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"gorm.io/gorm"
)

// UserService provides user management operations.
type UserService struct{}

// NewUserService creates a new UserService instance.
func NewUserService() *UserService {
	return &UserService{}
}

// BanUser bans a user with the specified reason.
// This sets the user as unlisted, muted, and sets the ban reason.
func (s *UserService) BanUser(req BanUserRequest) error {
	if req.UserID == 0 {
		return ErrInvalidUserID
	}
	if req.Reason == "" {
		return ErrInvalidReason
	}

	var profile models.Profile
	if err := db.DB.First(&profile, req.UserID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return ErrUserNotFound
		}
		return err
	}

	return db.DB.Model(&profile).Updates(map[string]interface{}{
		"is_unlisted": true,
		"mute":        true,
		"ban_reason":  req.Reason,
	}).Error
}

// UnbanUser unbans a previously banned user.
func (s *UserService) UnbanUser(req UnbanUserRequest) error {
	if req.UserID == 0 {
		return ErrInvalidUserID
	}

	var profile models.Profile
	if err := db.DB.First(&profile, req.UserID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return ErrUserNotFound
		}
		return err
	}

	return db.DB.Model(&profile).Updates(map[string]interface{}{
		"is_unlisted": false,
		"mute":        false,
		"ban_reason":  gorm.Expr("NULL"),
	}).Error
}

// UpdateUser updates a user's profile information.
func (s *UserService) UpdateUser(req UpdateUserRequest) error {
	if req.UserID == 0 {
		return ErrInvalidUserID
	}

	var profile models.Profile
	if err := db.DB.Preload("User").First(&profile, req.UserID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return ErrUserNotFound
		}
		return err
	}

	// Update profile fields
	profileUpdates := make(map[string]interface{})
	if req.DisplayName != nil {
		profileUpdates["username_display_override"] = *req.DisplayName
	}
	if req.About != nil {
		profileUpdates["about"] = *req.About
	}
	if req.IsUnlisted != nil {
		profileUpdates["is_unlisted"] = *req.IsUnlisted
	}
	if req.IsMuted != nil {
		profileUpdates["mute"] = *req.IsMuted
	}
	if req.DisplayRank != nil {
		profileUpdates["display_rank"] = *req.DisplayRank
	}
	if req.BanReason != nil {
		profileUpdates["ban_reason"] = *req.BanReason
	}

	if len(profileUpdates) > 0 {
		if err := db.DB.Model(&profile).Updates(profileUpdates).Error; err != nil {
			return err
		}
	}

	// Update user fields
	userUpdates := make(map[string]interface{})
	if req.IsActive != nil {
		userUpdates["is_active"] = *req.IsActive
	}
	if req.Email != nil {
		userUpdates["email"] = *req.Email
	}

	if len(userUpdates) > 0 {
		if err := db.DB.Model(&profile.User).Updates(userUpdates).Error; err != nil {
			return err
		}
	}

	// Handle organization removals
	if len(req.RemoveOrganizationIDs) > 0 {
		var orgs []models.Organization
		if err := db.DB.Where("id IN ?", req.RemoveOrganizationIDs).Find(&orgs).Error; err != nil {
			return err
		}
		if err := db.DB.Model(&profile).Association("Organizations").Delete(&orgs); err != nil {
			return err
		}
	}

	// Handle organization additions
	if len(req.AddOrganizationIDs) > 0 {
		var orgs []models.Organization
		if err := db.DB.Where("id IN ?", req.AddOrganizationIDs).Find(&orgs).Error; err != nil {
			return err
		}
		if err := db.DB.Model(&profile).Association("Organizations").Append(&orgs); err != nil {
			return err
		}
	}

	// Handle organization admin changes
	// For admin changes, we need to remove and re-add to update admin status
	if len(req.RemoveOrganizationAdmin) > 0 || len(req.AddOrganizationAdmin) > 0 {
		var removeOrgs []models.Organization
		var addOrgs []models.Organization

		if len(req.RemoveOrganizationAdmin) > 0 {
			if err := db.DB.Where("id IN ?", req.RemoveOrganizationAdmin).Find(&removeOrgs).Error; err != nil {
				return err
			}
		}
		if len(req.AddOrganizationAdmin) > 0 {
			if err := db.DB.Where("id IN ?", req.AddOrganizationAdmin).Find(&addOrgs).Error; err != nil {
				return err
			}
		}

		// Remove admin status
		if len(removeOrgs) > 0 {
			if err := db.DB.Model(&profile).Association("Organizations").Delete(&removeOrgs); err != nil {
				return err
			}
		}

		// Add admin status
		if len(addOrgs) > 0 {
			if err := db.DB.Model(&profile).Association("Organizations").Append(&addOrgs); err != nil {
				return err
			}
		}
	}

	return nil
}

// DeleteUser performs a soft delete by deactivating the user.
func (s *UserService) DeleteUser(req DeleteUserRequest) error {
	if req.UserID == 0 {
		return ErrInvalidUserID
	}

	var profile models.Profile
	if err := db.DB.Preload("User").First(&profile, req.UserID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return ErrUserNotFound
		}
		return err
	}

	// Soft delete: deactivate user and mark as unlisted
	if err := db.DB.Model(&profile.User).Update("is_active", false).Error; err != nil {
		return err
	}

	return db.DB.Model(&profile).Update("is_unlisted", true).Error
}

// GetUser retrieves a user profile by ID.
func (s *UserService) GetUser(req GetUserRequest) (*UserProfile, error) {
	if req.UserID == 0 {
		return nil, ErrInvalidUserID
	}

	var profile models.Profile
	if err := db.DB.Preload("User").Preload("Organizations").First(&profile, req.UserID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	orgIDs := make([]uint, len(profile.Organizations))
	for i, org := range profile.Organizations {
		orgIDs[i] = org.ID
	}

	return &UserProfile{
		ID:                 profile.ID,
		Username:           profile.User.Username,
		Email:              profile.User.Email,
		DisplayName:        profile.UsernameDisplayOverride,
		About:              getStringPtr(profile.About),
		Points:             profile.Points,
		PerformancePoints:  profile.PerformancePoints,
		ContributionPoints: profile.ContributionPoints,
		Rating:             profile.Rating,
		ProblemCount:       profile.ProblemCount,
		IsStaff:            profile.User.IsStaff,
		IsSuperuser:        profile.User.IsSuperuser,
		IsActive:           profile.User.IsActive,
		IsUnlisted:         profile.IsUnlisted,
		IsMuted:            profile.Mute,
		IsTotpEnabled:      profile.IsTotpEnabled,
		IsWebauthnEnabled:  profile.IsWebauthnEnabled,
		DateJoined:         profile.User.DateJoined,
		LastAccess:         profile.LastAccess,
		DisplayRank:        profile.DisplayRank,
		BanReason:          profile.BanReason,
		OrganizationIDs:    orgIDs,
	}, nil
}

// ListUsers retrieves a paginated list of users.
func (s *UserService) ListUsers(req ListUsersRequest) (*ListUsersResponse, error) {
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 20
	}
	if req.PageSize > 100 {
		req.PageSize = 100
	}

	var profiles []struct {
		models.Profile
		Username    string    `gorm:"column:username"`
		Email       string    `gorm:"column:email"`
		IsActive    bool      `gorm:"column:is_active"`
		IsStaff     bool      `gorm:"column:is_staff"`
		IsSuperuser bool      `gorm:"column:is_superuser"`
		DateJoined  time.Time `gorm:"column:date_joined"`
	}

	query := db.DB.Table("judge_profile").
		Joins("JOIN auth_user ON auth_user.id = judge_profile.user_id").
		Select("judge_profile.*, auth_user.username, auth_user.email, auth_user.is_active, auth_user.is_staff, auth_user.is_superuser, auth_user.date_joined").
		Order("auth_user.date_joined DESC")

	// Get total count
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	// Get paginated results
	if err := query.
		Offset((req.Page - 1) * req.PageSize).
		Limit(req.PageSize).
		Scan(&profiles).Error; err != nil {
		return nil, err
	}

	users := make([]UserProfile, len(profiles))
	for i, p := range profiles {
		users[i] = UserProfile{
			ID:                p.ID,
			Username:          p.Username,
			Email:             p.Email,
			Points:            p.Points,
			PerformancePoints: p.PerformancePoints,
			ProblemCount:      p.ProblemCount,
			Rating:            p.Rating,
			IsStaff:           p.IsStaff,
			IsSuperuser:       p.IsSuperuser,
			IsActive:          p.IsActive,
			IsUnlisted:        p.IsUnlisted,
			IsMuted:           p.Mute,
			DateJoined:        p.DateJoined,
			LastAccess:        p.LastAccess,
			DisplayRank:       p.DisplayRank,
			BanReason:         p.BanReason,
		}
	}

	return &ListUsersResponse{
		Users:    users,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}, nil
}

// Helper function to safely dereference string pointers
func getStringPtr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
