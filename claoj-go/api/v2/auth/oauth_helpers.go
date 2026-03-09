package auth

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/CLAOJ/claoj-go/auth"
	"github.com/CLAOJ/claoj-go/db"
	"github.com/CLAOJ/claoj-go/models"
	"github.com/CLAOJ/claoj-go/oauth"
	"gorm.io/gorm"
)

// OAuthUserLink maps OAuth accounts to local users
type OAuthUserLink struct {
	ID           uint      `gorm:"primaryKey;column:id"`
	UserID       uint      `gorm:"column:user_id;not null;uniqueIndex:idx_oauth_link"`
	Provider     string    `gorm:"column:provider;size:20;not null;uniqueIndex:idx_oauth_link"`
	ProviderID   string    `gorm:"column:provider_id;size:100;not null"`
	Email        string    `gorm:"column:email;size:254;not null"`
	AccessToken  string    `gorm:"column:access_token;type:longtext"`
	RefreshToken string    `gorm:"column:refresh_token;type:longtext"`
	Expiry       time.Time `gorm:"column:expiry"`
	CreatedAt    time.Time `gorm:"column:created_at;not null"`
}

func (OAuthUserLink) TableName() string { return "oauth_user_link" }

// findOrCreateOAuthUser finds existing user or creates new one from OAuth info
// Returns: userID, requiresLinking (false if new account was created), error
// Special errors: "password_required_for_linking", "invalid_password_for_linking"
func findOrCreateOAuthUser(userInfo *oauth.UserInfo, password string) (uint, bool, error) {
	// Check if OAuth account is already linked
	var link OAuthUserLink
	err := db.DB.Where("provider = ? AND provider_id = ?", userInfo.Provider, userInfo.ID).First(&link).Error

	if err == nil {
		// OAuth account linked, return user ID
		return link.UserID, false, nil
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, false, err
	}

	// Check if user with this email exists
	var user models.AuthUser
	err = db.DB.Where("email = ?", userInfo.Email).First(&user).Error

	if err == nil {
		// User exists - require password confirmation to link accounts
		// Skip password check if user has no password (OAuth-only account)
		if user.Password != "" {
			if password == "" {
				return 0, true, errors.New("password_required_for_linking")
			}
			// Verify password
			match, pwdErr := auth.CheckPassword(password, user.Password)
			if pwdErr != nil || !match {
				return 0, true, errors.New("invalid_password_for_linking")
			}
		}
		// Password verified, link OAuth account
		link = OAuthUserLink{
			UserID:     user.ID,
			Provider:   userInfo.Provider,
			ProviderID: userInfo.ID,
			Email:      userInfo.Email,
			CreatedAt:  time.Now(),
		}
		if err := db.DB.Create(&link).Error; err != nil {
			return 0, false, err
		}
		return user.ID, false, nil
	}

	// Create new user
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, false, err
	}

	// Generate unique username from email
	username := generateUsernameFromEmail(userInfo.Email)

	// Create auth user
	now := time.Now()
	user = models.AuthUser{
		Username:    username,
		Password:    "", // Empty password for OAuth users
		Email:       userInfo.Email,
		FirstName:   userInfo.Name,
		LastName:    "",
		IsStaff:     false,
		IsActive:    true,
		IsSuperuser: false,
		DateJoined:  now,
	}

	err = db.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&user).Error; err != nil {
			return err
		}

		// Create profile
		profile := models.Profile{
			UserID:                  user.ID,
			Timezone:                "UTC",
			LanguageID:              1, // Default to first language
			LastAccess:              now,
			DisplayRank:             "user",
			AceTheme:                "auto",
			SiteTheme:               "auto",
			MathEngine:              "TeX",
			UserScript:              "",
			UsernameDisplayOverride: userInfo.Name,
		}
		if err := tx.Create(&profile).Error; err != nil {
			return err
		}

		// Create OAuth link
		link = OAuthUserLink{
			UserID:     user.ID,
			Provider:   userInfo.Provider,
			ProviderID: userInfo.ID,
			Email:      userInfo.Email,
			CreatedAt:  now,
		}
		return tx.Create(&link).Error
	})

	if err != nil {
		return 0, false, err
	}

	// New user created, no prior account linking was needed
	return user.ID, true, nil
}

func generateUsernameFromEmail(email string) string {
	// Extract username from email
	parts := strings.Split(email, "@")
	username := parts[0]

	// Sanitize username
	username = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '.' {
			return r
		}
		return -1
	}, username)

	// Check if username exists, append random suffix if needed
	baseUsername := username
	counter := 1
	for {
		var count int64
		db.DB.Model(&models.AuthUser{}).Where("username = ?", username).Count(&count)
		if count == 0 {
			break
		}
		username = fmt.Sprintf("%s%d", baseUsername, counter)
		counter++
	}

	return username
}
