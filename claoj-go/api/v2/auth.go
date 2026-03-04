package v2

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/CLAOJ/claoj-go/auth"
	"github.com/CLAOJ/claoj-go/config"
	"github.com/CLAOJ/claoj-go/db"
	"github.com/CLAOJ/claoj-go/email"
	"github.com/CLAOJ/claoj-go/models"
	"github.com/CLAOJ/claoj-go/oauth"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// Login verifies credentials and returns an access/refresh JWT pair
func Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user models.AuthUser
	if err := db.DB.Where("username = ?", req.Username).First(&user).Error; err != nil {
		// Generic error to prevent username enumeration
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid username or password"})
		return
	}

	if !user.IsActive {
		c.JSON(http.StatusForbidden, gin.H{"error": "account is inactive"})
		return
	}

	// Verify Django pbkdf2_sha256 password hash
	match, err := auth.CheckPassword(req.Password, user.Password)
	if err != nil || !match {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid username or password"})
		return
	}

	// Check if TOTP is required
	if CheckTOTPRequired(user.ID) {
		// Return TOTP challenge
		c.JSON(http.StatusOK, gin.H{
			"requires_totp": true,
			"username":      user.Username,
			"message":       "Please enter your TOTP code",
		})
		return
	}

	// Generate JWTs
	accessToken, refreshToken, err := auth.GenerateTokens(user.ID, user.Username, user.IsSuperuser)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate tokens"})
		return
	}

	// In a real app we might update last_login here
	c.JSON(http.StatusOK, gin.H{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"user": gin.H{
			"id":       user.ID,
			"username": user.Username,
			"is_admin": user.IsSuperuser,
		},
	})
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// Refresh generates a new access token given a valid refresh token.
func Refresh(c *gin.Context) {
	var req RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	claims, err := auth.VerifyToken(req.RefreshToken, "refresh")
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired refresh token"})
		return
	}

	// Just generate a new token pair
	accessToken, refreshToken, err := auth.GenerateTokens(claims.UserID, claims.Username, claims.IsAdmin)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate tokens"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
	})
}

// PasswordResetRequest - POST /api/v2/auth/password/reset
// Sends password reset email
func PasswordResetRequest(c *gin.Context) {
	var req struct {
		Email string `json:"email" binding:"required,email"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Find user by email
	var user models.AuthUser
	if err := db.DB.Where("email = ?", req.Email).First(&user).Error; err != nil {
		// Don't reveal if email exists - always return success
		c.JSON(http.StatusOK, gin.H{"message": "If the email exists, a reset link has been sent"})
		return
	}

	// Generate reset token (32 bytes = 64 hex chars)
	token := make([]byte, 32)
	if _, err := rand.Read(token); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}
	tokenStr := hex.EncodeToString(token)

	// Store token in database (using a simple key-value approach)
	// We'll use a custom table for password reset tokens
	resetToken := models.PasswordResetToken{
		UserID:    user.ID,
		Token:     tokenStr,
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}
	if err := db.DB.Create(&resetToken).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create reset token"})
		return
	}

	// Build reset link
	resetLink := config.C.App.SiteFullURL + "/reset-password?token=" + tokenStr

	// Send email
	if err := email.SendPasswordResetEmail(user.Email, user.Username, resetLink); err != nil {
		// Log error but still return success to avoid enumeration
		log.Printf("auth: failed to send reset email: %v", err)
	}

	c.JSON(http.StatusOK, gin.H{"message": "If the email exists, a reset link has been sent"})
}

// PasswordResetConfirm - POST /api/v2/auth/password/reset/confirm
// Resets password using valid token
func PasswordResetConfirm(c *gin.Context) {
	var req struct {
		Token    string `json:"token" binding:"required"`
		Password string `json:"password" binding:"required,min=6"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Find valid reset token
	var resetToken models.PasswordResetToken
	if err := db.DB.Where("token = ? AND expires_at > ?", req.Token, time.Now()).First(&resetToken).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid or expired token"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}

	// Hash new password
	hashedPassword, err := auth.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
		return
	}

	// Update user password
	err = db.DB.Transaction(func(tx *gorm.DB) error {
		// Update user password
		if err := tx.Model(&models.AuthUser{}).Where("id = ?", resetToken.UserID).Update("password", hashedPassword).Error; err != nil {
			return err
		}
		// Invalidate all reset tokens for this user
		if err := tx.Where("user_id = ?", resetToken.UserID).Delete(&models.PasswordResetToken{}).Error; err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to reset password"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Password reset successfully"})
}

// OAuthStart - GET /api/v2/auth/oauth/:provider
// Starts OAuth flow by redirecting to provider
func OAuthStart(c *gin.Context) {
	provider := c.Param("provider")

	if provider != "google" && provider != "github" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported provider"})
		return
	}

	// Generate state token for CSRF protection
	state, err := oauth.GenerateStateToken(config.C.App.SecretKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate state token"})
		return
	}

	// Store state in cookie with SameSite attribute for CSRF protection
	// Using http.Cookie directly to set SameSite
	cookie := &http.Cookie{
		Name:     "oauth_state",
		Value:    state,
		MaxAge:   600,
		Path:     "/api/v2/auth",
		Domain:   "",
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}
	http.SetCookie(c.Writer, cookie)

	// Get auth URL
	authURL, err := oauth.GetAuthURL(oauth.Provider(provider), state)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "OAuth provider not configured"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"auth_url": authURL})
}

// OAuthCallbackRequest - POST /api/v2/auth/oauth/:provider/callback
type OAuthCallbackRequest struct {
	Code  string `json:"code" binding:"required"`
	State string `json:"state" binding:"required"`
}

// OAuthCallback - POST /api/v2/auth/oauth/:provider/callback
// Handles OAuth callback and returns JWT tokens
func OAuthCallback(c *gin.Context) {
	provider := c.Param("provider")

	if provider != "google" && provider != "github" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported provider"})
		return
	}

	var req OAuthCallbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify state token
	storedState, err := c.Cookie("oauth_state")
	if err != nil || storedState != req.State {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid state parameter"})
		return
	}
	// Clear state cookie
	cookie := &http.Cookie{
		Name:     "oauth_state",
		Value:    "",
		MaxAge:   -1,
		Path:     "/api/v2/auth",
		Domain:   "",
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}
	http.SetCookie(c.Writer, cookie)

	ctx := c.Request.Context()

	// Exchange code for token
	token, err := oauth.ExchangeCode(ctx, oauth.Provider(provider), req.Code)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to exchange code"})
		return
	}

	// Get user info
	userInfo, err := oauth.GetUserInfo(ctx, oauth.Provider(provider), token)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get user info"})
		return
	}

	// Find or create user
	userID, err := findOrCreateOAuthUser(userInfo)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Generate JWT tokens
	accessToken, refreshToken, err := auth.GenerateTokens(userID, userInfo.Email, false)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate tokens"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"user": gin.H{
			"id":       userID,
			"username": userInfo.Email,
			"email":    userInfo.Email,
			"name":     userInfo.Name,
			"avatar":   userInfo.AvatarURL,
			"provider": userInfo.Provider,
		},
	})
}

// OAuthUserLink maps OAuth accounts to local users
type OAuthUserLink struct {
	ID           uint   `gorm:"primaryKey;column:id"`
	UserID       uint   `gorm:"column:user_id;not null;uniqueIndex:idx_oauth_link"`
	Provider     string `gorm:"column:provider;size:20;not null;uniqueIndex:idx_oauth_link"`
	ProviderID   string `gorm:"column:provider_id;size:100;not null"`
	Email        string `gorm:"column:email;size:254;not null"`
	AccessToken  string `gorm:"column:access_token;type:longtext"`
	RefreshToken string `gorm:"column:refresh_token;type:longtext"`
	Expiry       time.Time `gorm:"column:expiry"`
	CreatedAt    time.Time `gorm:"column:created_at;not null"`
}

func (OAuthUserLink) TableName() string { return "oauth_user_link" }

// findOrCreateOAuthUser finds existing user or creates new one from OAuth info
func findOrCreateOAuthUser(userInfo *oauth.UserInfo) (uint, error) {
	// Check if OAuth account is already linked
	var link OAuthUserLink
	err := db.DB.Where("provider = ? AND provider_id = ?", userInfo.Provider, userInfo.ID).First(&link).Error

	if err == nil {
		// OAuth account linked, return user ID
		return link.UserID, nil
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, err
	}

	// Check if user with this email exists
	var user models.AuthUser
	err = db.DB.Where("email = ?", userInfo.Email).First(&user).Error

	if err == nil {
		// User exists, link OAuth account
		link = OAuthUserLink{
			UserID:     user.ID,
			Provider:   userInfo.Provider,
			ProviderID: userInfo.ID,
			Email:      userInfo.Email,
			CreatedAt:  time.Now(),
		}
		db.DB.Create(&link)
		return user.ID, nil
	}

	// Create new user
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, err
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
			UserID:             user.ID,
			Timezone:           "UTC",
			LanguageID:         1, // Default to first language
			LastAccess:         now,
			DisplayRank:        "user",
			AceTheme:           "auto",
			SiteTheme:          "auto",
			MathEngine:         "TeX",
			UserScript:         "",
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
		return 0, err
	}

	return user.ID, nil
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
