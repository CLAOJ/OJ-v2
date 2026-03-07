package v2

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/CLAOJ/claoj-go/auth"
	"github.com/CLAOJ/claoj-go/cache"
	"github.com/CLAOJ/claoj-go/config"
	"github.com/CLAOJ/claoj-go/db"
	"github.com/CLAOJ/claoj-go/email"
	"github.com/CLAOJ/claoj-go/lockout"
	"github.com/CLAOJ/claoj-go/models"
	"github.com/CLAOJ/claoj-go/oauth"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// getLockoutRepo returns the lockout repository (nil if Redis not available)
func getLockoutRepo() *lockout.Repository {
	if cache.Client != nil {
		return lockout.NewRepository(cache.Client)
	}
	return nil
}

type LoginRequest struct {
	Username    string `json:"username" binding:"required"`
	Password    string `json:"password" binding:"required"`
	RememberMe  bool   `json:"remember_me"`
}

// Login verifies credentials and returns an access/refresh JWT pair
func Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := context.Background()
	lockoutRepo := getLockoutRepo()

	// Check if account is locked due to failed attempts
	if lockoutRepo != nil {
		locked, ttl, err := lockoutRepo.IsLocked(ctx, "user:"+req.Username)
		if err != nil {
			log.Printf("Lockout check error: %v", err)
		}
		if locked {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":         "account_locked",
				"message":       "Account temporarily locked due to too many failed login attempts.",
				"retry_after":   int(ttl.Seconds()),
				"retry_after_text": lockout.FormatLockoutMessage(0, ttl),
			})
			return
		}
	}

	var user models.AuthUser
	if err := db.DB.Where("username = ?", req.Username).First(&user).Error; err != nil {
		// Record failed attempt even for non-existent usernames (prevents enumeration)
		if lockoutRepo != nil {
			lockoutRepo.RecordFailedAttempt(ctx, "ip:"+c.ClientIP())
		}
		// Generic error to prevent username enumeration
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid username or password"})
		return
	}

	if !user.IsActive {
		// Check if user has pending email verification
		var verificationToken models.EmailVerificationToken
		err := db.DB.Where("user_id = ? AND expires_at > ?", user.ID, time.Now()).First(&verificationToken).Error
		if err == nil {
			// Token exists, user needs to verify email
			c.JSON(http.StatusForbidden, gin.H{
				"error": "email not verified",
				"message": "Please verify your email address before logging in. Check your inbox for the verification link.",
				"requires_email_verification": true,
			})
			return
		}
		// Check if user is banned (profile has ban_reason)
		var profile models.Profile
		err = db.DB.Where("user_id = ?", user.ID).First(&profile).Error
		if err == nil && profile.BanReason != nil && *profile.BanReason != "" {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "account_banned",
				"message": "Your account has been banned.",
				"ban_reason": *profile.BanReason,
			})
			return
		}
		// Account is inactive for other reasons (banned, etc.)
		c.JSON(http.StatusForbidden, gin.H{"error": "account is inactive"})
		return
	}

	// Verify Django pbkdf2_sha256 password hash
	match, err := auth.CheckPassword(req.Password, user.Password)
	if err != nil || !match {
		// Record failed attempt
		if lockoutRepo != nil {
			count, _ := lockoutRepo.RecordFailedAttempt(ctx, "user:"+req.Username)
			remaining, _ := lockoutRepo.GetRemainingAttempts(ctx, "user:"+req.Username)

			// Return warning if attempts are running low
			if count >= 5 && remaining > 0 {
				c.JSON(http.StatusUnauthorized, gin.H{
					"error": "invalid username or password",
					"warning": lockout.FormatLockoutMessage(remaining, 0),
					"attempts_remaining": remaining,
				})
				return
			}
		}
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid username or password"})
		return
	}

	// Successful login - reset lockout counter
	if lockoutRepo != nil {
		lockoutRepo.Reset(ctx, "user:"+req.Username)
	}

	// Check if TOTP is required for admins
	if config.C.App.RequireTotpForAdmins && user.IsStaff && !CheckTOTPRequired(user.ID) {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "TOTP required for admin accounts",
			"message": "Administrator accounts must enable two-factor authentication. Please contact the system administrator.",
		})
		return
	}

	// Check if TOTP is required (for all users who have it enabled)
	if CheckTOTPRequired(user.ID) {
		// Return TOTP challenge
		c.JSON(http.StatusOK, gin.H{
			"requires_totp": true,
			"username":      user.Username,
			"message":       "Please enter your TOTP code",
		})
		return
	}

	// Generate JWTs with extended TTL if remember_me is true
	var familyID string
	accessToken, refreshToken, familyID, err := auth.GenerateTokens(user.ID, user.Username, user.IsSuperuser, "", req.RememberMe)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate tokens"})
		return
	}

	// Store refresh token in database for revocation tracking
	userAgent := c.Request.UserAgent()
	clientIP := c.ClientIP()
	refreshTokenTTL := 7 * 24 * time.Hour
	cookieMaxAge := 7 * 24 * 60 * 60 // 7 days in seconds
	if req.RememberMe {
		refreshTokenTTL = 30 * 24 * time.Hour
		cookieMaxAge = 30 * 24 * 60 * 60 // 30 days in seconds
	}
	refreshTokenModel := models.RefreshToken{
		UserID:    user.ID,
		Token:     refreshToken,
		ExpiresAt: time.Now().Add(refreshTokenTTL),
		UserAgent: &userAgent,
		ClientIP:  &clientIP,
		FamilyID:  familyID,
	}
	if err := db.DB.Create(&refreshTokenModel).Error; err != nil {
		log.Printf("Failed to store refresh token: %v", err)
		// Don't fail the login, just log the error
	}

	// Set httpOnly cookies for tokens
	// Access token cookie (15 minutes)
	// Secure=false for HTTP development, true for HTTPS production
	secureCookie := strings.HasPrefix(config.C.App.SiteFullURL, "https://")
	c.SetCookie("access_token", accessToken, 900, "/", "", secureCookie, true)
	// Refresh token cookie (7 or 30 days based on remember_me)
	c.SetCookie("refresh_token", refreshToken, cookieMaxAge, "/", "", secureCookie, true)

	// In a real app we might update last_login here
	c.JSON(http.StatusOK, gin.H{
		"access_token":  accessToken,  // Also return in body for backwards compatibility
		"refresh_token": refreshToken,
		"user": gin.H{
			"id":       user.ID,
			"username": user.Username,
			"is_admin": user.IsSuperuser,
		},
	})
}

// Refresh generates a new access token given a valid refresh token from cookie
func Refresh(c *gin.Context) {
	// Get refresh token from httpOnly cookie
	refreshToken, err := c.Cookie("refresh_token")
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "refresh token not found in cookie"})
		return
	}

	claims, err := auth.VerifyToken(refreshToken, "refresh")
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired refresh token"})
		return
	}

	// Check if token has been revoked
	var refreshTokenModel models.RefreshToken
	if err := db.DB.Where("token = ? AND user_id = ?", refreshToken, claims.UserID).First(&refreshTokenModel).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "refresh token not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to validate token"})
		return
	}

	if refreshTokenModel.RevokedAt != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "refresh token has been revoked"})
		return
	}

	if refreshTokenModel.ExpiresAt.Before(time.Now()) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "refresh token has expired"})
		return
	}

	// Generate new token pair with same family ID (maintain original remember_me setting via family)
	accessToken, newRefreshToken, familyID, err := auth.GenerateTokens(claims.UserID, claims.Username, claims.IsAdmin, refreshTokenModel.FamilyID, false)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate tokens"})
		return
	}

	// Revoke old token and store new one
	tx := db.DB.Begin()
	if err := tx.Model(&refreshTokenModel).Update("revoked_at", time.Now()).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to revoke token"})
		return
	}

	// Store new refresh token - maintain same TTL as original
	userAgent := c.Request.UserAgent()
	clientIP := c.ClientIP()
	newRefreshTokenModel := models.RefreshToken{
		UserID:    claims.UserID,
		Token:     newRefreshToken,
		ExpiresAt: refreshTokenModel.ExpiresAt, // Maintain original expiration
		UserAgent: &userAgent,
		ClientIP:  &clientIP,
		FamilyID:  familyID,
	}
	if err := tx.Create(&newRefreshTokenModel).Error; err != nil {
		tx.Rollback()
		log.Printf("Failed to store refresh token: %v", err)
		// Don't fail the request, just log the error
	}
	tx.Commit()

	// Set httpOnly cookies for tokens
	secureCookie := strings.HasPrefix(config.C.App.SiteFullURL, "https://")
	c.SetCookie("access_token", accessToken, 900, "/", "", secureCookie, true)
	c.SetCookie("refresh_token", newRefreshToken, 7*24*60*60, "/", "", secureCookie, true)

	c.JSON(http.StatusOK, gin.H{
		"access_token":  accessToken,
		"refresh_token": newRefreshToken,
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
	secureCookie := strings.HasPrefix(config.C.App.SiteFullURL, "https://")
	cookie := &http.Cookie{
		Name:     "oauth_state",
		Value:    state,
		MaxAge:   600,
		Path:     "/api/v2/auth",
		Domain:   "",
		Secure:   secureCookie,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
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
	Code     string `json:"code" binding:"required"`
	State    string `json:"state" binding:"required"`
	Password string `json:"password,omitempty"` // For linking to existing account
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
	secureCookie := strings.HasPrefix(config.C.App.SiteFullURL, "https://")
	cookie := &http.Cookie{
		Name:     "oauth_state",
		Value:    "",
		MaxAge:   -1,
		Path:     "/api/v2/auth",
		Domain:   "",
		Secure:   secureCookie,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
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

	// Find or create user - may require password confirmation
	userID, requiresLinking, err := findOrCreateOAuthUser(userInfo, req.Password)
	if err != nil {
		if err.Error() == "password_required_for_linking" {
			c.JSON(http.StatusConflict, gin.H{
				"error": "account_linking_required",
				"message": "An account with this email already exists. Please provide your password to link accounts.",
				"email": userInfo.Email,
			})
			return
		}
		if err.Error() == "invalid_password_for_linking" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "invalid_password",
				"message": "Invalid password. Please try again or use a different email.",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Generate JWT tokens (OAuth logins don't have remember_me, use default 7 days)
	accessToken, refreshToken, _, err := auth.GenerateTokens(userID, userInfo.Email, false, "", false)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate tokens"})
		return
	}

	// Set httpOnly cookies
	secureCookie = strings.HasPrefix(config.C.App.SiteFullURL, "https://")
	c.SetCookie("access_token", accessToken, 900, "/", "", secureCookie, true)
	c.SetCookie("refresh_token", refreshToken, 7*24*60*60, "/", "", secureCookie, true)

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
			"linked":   !requiresLinking,
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

// Logout - POST /api/v2/auth/logout
// Revokes refresh token and invalidates session
func Logout(c *gin.Context) {
	// Get refresh token from httpOnly cookie
	refreshToken, err := c.Cookie("refresh_token")
	if err != nil {
		// No refresh token in cookie, user might already be logged out
		// Still clear any access token cookie and return success
		secureCookie := strings.HasPrefix(config.C.App.SiteFullURL, "https://")
		c.SetCookie("access_token", "", -1, "/", "", secureCookie, true)
		c.JSON(http.StatusOK, gin.H{"message": "logged out successfully"})
		return
	}

	// Get user ID from context (set by auth middleware from access token)
	userID := c.GetUint("user_id")
	if userID == 0 {
		// No valid access token, but still clear cookies
		secureCookie := strings.HasPrefix(config.C.App.SiteFullURL, "https://")
		c.SetCookie("access_token", "", -1, "/", "", secureCookie, true)
		c.SetCookie("refresh_token", "", -1, "/", "", secureCookie, true)
		c.JSON(http.StatusOK, gin.H{"message": "logged out successfully"})
		return
	}

	// Verify the token first
	claims, err := auth.VerifyToken(refreshToken, "refresh")
	if err != nil {
		// Token is invalid/expired, but still try to revoke it
		revokedNow := time.Now()
		db.DB.Model(&models.RefreshToken{}).
			Where("token = ?", refreshToken).
			Update("revoked_at", &revokedNow)
	} else {
		// Revoke the refresh token in database
		revokedNow := time.Now()
		result := db.DB.Model(&models.RefreshToken{}).
			Where("token = ? AND user_id = ?", refreshToken, claims.UserID).
			Update("revoked_at", &revokedNow)

		if result.RowsAffected == 0 {
			// Token not found in database, but still return success
		}
	}

	// Clear cookies
	secureCookie := strings.HasPrefix(config.C.App.SiteFullURL, "https://")
	c.SetCookie("access_token", "", -1, "/", "", secureCookie, true)
	c.SetCookie("refresh_token", "", -1, "/", "", secureCookie, true)

	c.JSON(http.StatusOK, gin.H{"message": "logged out successfully"})
}

// RevocateAllSessions - POST /api/v2/auth/revoke-all-sessions
// Revokes all refresh tokens for the current user (force logout all devices)
func RevokeAllSessions(c *gin.Context) {
	userID := c.GetUint("userID")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	revokedNow := time.Now()
	result := db.DB.Model(&models.RefreshToken{}).
		Where("user_id = ? AND revoked_at IS NULL", userID).
		Update("revoked_at", &revokedNow)

	c.JSON(http.StatusOK, gin.H{
		"message":          "all sessions revoked",
		"sessions_revoked": result.RowsAffected,
	})
}
