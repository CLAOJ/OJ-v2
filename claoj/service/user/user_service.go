// Package user provides user management services.
package user

import (
	"strings"
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
	if req.Notes != nil {
		profileUpdates["notes"] = *req.Notes
	}
	if req.UserScript != nil {
		profileUpdates["user_script"] = *req.UserScript
	}
	if req.Timezone != nil {
		profileUpdates["timezone"] = *req.Timezone
	}
	if req.LanguageID != nil {
		profileUpdates["language_id"] = *req.LanguageID
	}
	if req.AceTheme != nil {
		profileUpdates["ace_theme"] = *req.AceTheme
	}
	if req.SiteTheme != nil {
		profileUpdates["site_theme"] = *req.SiteTheme
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

	// Validate the auth_user updates (username/email uniqueness) before
	// writing anything, so a rejected update cannot leave the profile half
	// changed.
	userUpdates := make(map[string]interface{})
	if req.Username != nil && *req.Username != profile.User.Username {
		var count int64
		if err := db.DB.Model(&models.AuthUser{}).
			Where("username = ? AND id <> ?", *req.Username, profile.UserID).
			Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return ErrUsernameTaken
		}
		userUpdates["username"] = *req.Username
	}
	if req.Email != nil && *req.Email != profile.User.Email {
		var count int64
		if err := db.DB.Model(&models.AuthUser{}).
			Where("email = ? AND id <> ?", *req.Email, profile.UserID).
			Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return ErrEmailTaken
		}
		userUpdates["email"] = *req.Email
	}
	if req.FirstName != nil {
		userUpdates["first_name"] = *req.FirstName
	}
	if req.LastName != nil {
		userUpdates["last_name"] = *req.LastName
	}
	if req.IsActive != nil {
		userUpdates["is_active"] = *req.IsActive
	}
	if req.IsStaff != nil {
		userUpdates["is_staff"] = *req.IsStaff
	}
	if req.IsSuperuser != nil {
		userUpdates["is_superuser"] = *req.IsSuperuser
	}

	// One transaction covers both rows so a failure on either side cannot
	// leave the account half updated.
	if len(profileUpdates) > 0 || len(userUpdates) > 0 {
		if err := db.DB.Transaction(func(tx *gorm.DB) error {
			if len(profileUpdates) > 0 {
				if err := tx.Model(&profile).Updates(profileUpdates).Error; err != nil {
					return err
				}
			}
			if len(userUpdates) > 0 {
				if err := tx.Model(&profile.User).Updates(userUpdates).Error; err != nil {
					return err
				}
			}
			return nil
		}); err != nil {
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
		UserID:             profile.UserID,
		Username:           profile.User.Username,
		FirstName:          profile.User.FirstName,
		LastName:           profile.User.LastName,
		Email:              profile.User.Email,
		DisplayName:        profile.UsernameDisplayOverride,
		About:              getStringPtr(profile.About),
		Notes:              getStringPtr(profile.Notes),
		UserScript:         profile.UserScript,
		Timezone:           profile.Timezone,
		LanguageID:         profile.LanguageID,
		AceTheme:           profile.AceTheme,
		SiteTheme:          profile.SiteTheme,
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
		LastLogin:          profile.User.LastLogin,
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
		Username    string     `gorm:"column:username"`
		FirstName   string     `gorm:"column:first_name"`
		LastName    string     `gorm:"column:last_name"`
		Email       string     `gorm:"column:email"`
		IsActive    bool       `gorm:"column:is_active"`
		IsStaff     bool       `gorm:"column:is_staff"`
		IsSuperuser bool       `gorm:"column:is_superuser"`
		DateJoined  time.Time  `gorm:"column:date_joined"`
		LastLogin   *time.Time `gorm:"column:last_login"`
	}

	// Sort columns are whitelisted here (never interpolated from input).
	// "points" and the rest of the judge_profile columns are already
	// qualified by the judge_profile.* select below.
	sortColumn, ok := map[string]string{
		"username":    "auth_user.username",
		"date_joined": "auth_user.date_joined",
		"last_login":  "auth_user.last_login",
		"points":      "judge_profile.points",
	}[req.Sort]
	if !ok {
		sortColumn = "auth_user.date_joined"
	}
	direction := "DESC"
	if strings.EqualFold(req.Order, "asc") {
		direction = "ASC"
	}

	query := db.DB.Table("judge_profile").
		Joins("JOIN auth_user ON auth_user.id = judge_profile.user_id").
		Select("judge_profile.*, auth_user.username, auth_user.first_name, auth_user.last_name, auth_user.email, auth_user.is_active, auth_user.is_staff, auth_user.is_superuser, auth_user.date_joined, auth_user.last_login").
		Order(sortColumn + " " + direction + ", judge_profile.id " + direction)

	if req.Search != "" {
		searchPattern := "%" + req.Search + "%"
		query = query.Where("auth_user.username LIKE ? OR auth_user.email LIKE ?", searchPattern, searchPattern)
	}
	if req.Staff != nil {
		query = query.Where("auth_user.is_staff = ?", *req.Staff)
	}
	if req.Superuser != nil {
		query = query.Where("auth_user.is_superuser = ?", *req.Superuser)
	}
	if req.Active != nil {
		query = query.Where("auth_user.is_active = ?", *req.Active)
	}
	if req.GroupID != 0 {
		query = query.Where("EXISTS (SELECT 1 FROM auth_user_groups WHERE auth_user_groups.user_id = judge_profile.user_id AND auth_user_groups.group_id = ?)", req.GroupID)
	}

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
	profileIDs := make([]uint, len(profiles))
	for i, p := range profiles {
		profileIDs[i] = p.ID
		users[i] = UserProfile{
			ID:                p.ID,
			UserID:            p.UserID,
			Username:          p.Username,
			FirstName:         p.FirstName,
			LastName:          p.LastName,
			Email:             p.Email,
			DisplayName:       p.UsernameDisplayOverride,
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
			LastLogin:         p.LastLogin,
			LastAccess:        p.LastAccess,
			DisplayRank:       p.DisplayRank,
			BanReason:         p.BanReason,
			Groups:            []GroupRef{},
		}
	}

	// Attach group names for the whole page in one query (no N+1).
	if len(profileIDs) > 0 {
		var rows []struct {
			ProfileID uint
			ID        uint
			Name      string
		}
		err := db.DB.Table("auth_user_groups").
			Select("judge_profile.id AS profile_id, auth_group.id AS id, auth_group.name AS name").
			Joins("JOIN auth_group ON auth_group.id = auth_user_groups.group_id").
			Joins("JOIN judge_profile ON judge_profile.user_id = auth_user_groups.user_id").
			Where("judge_profile.id IN ?", profileIDs).
			Order("auth_group.name ASC").
			Scan(&rows).Error
		if err != nil {
			return nil, err
		}
		byProfile := make(map[uint][]GroupRef, len(users))
		for _, r := range rows {
			byProfile[r.ProfileID] = append(byProfile[r.ProfileID], GroupRef{ID: r.ID, Name: r.Name})
		}
		for i := range users {
			if gs, ok := byProfile[users[i].ID]; ok {
				users[i].Groups = gs
			}
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
