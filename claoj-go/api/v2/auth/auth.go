package auth

import (
	"context"
	"net/http"
	"time"

	"github.com/CLAOJ/claoj-go/auth"
	"github.com/CLAOJ/claoj-go/cache"
	"github.com/CLAOJ/claoj-go/config"
	"github.com/CLAOJ/claoj-go/cookie"
	"github.com/CLAOJ/claoj-go/db"
	"github.com/CLAOJ/claoj-go/lockout"
	"github.com/CLAOJ/claoj-go/models"
	"github.com/gin-gonic/gin"
)

// getLockoutRepo returns the lockout repository (nil if Redis not available)
func getLockoutRepo() *lockout.Repository {
	if cache.Client != nil {
		return lockout.NewRepository(cache.Client)
	}
	return nil
}

type LoginRequest struct {
	Username   string `json:"username" binding:"required"`
	Password   string `json:"password" binding:"required"`
	RememberMe bool   `json:"remember_me"`
}

// Login – POST /api/v2/auth/login
// @Description User login with username and password. Returns JWT tokens on success.
// @Tags Authentication
// @Summary User login
// @Accept json
// @Produce json
// @Param request body LoginRequest true "Login credentials"
// @Success 200 {object} map[string]interface{} "Login successful with tokens or TOTP challenge"
// @Failure 400 {object} map[string]string "Invalid request body"
// @Failure 401 {object} map[string]string "Invalid credentials"
// @Failure 403 {object} map[string]string "Account locked, banned, or email not verified"
// @Failure 429 {object} map[string]interface{} "Account locked due to too many failed attempts"
// @Router /auth/login [post]
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
			// Log error but don't block login
		}
		if locked {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":            "account_locked",
				"message":          "Account temporarily locked due to too many failed login attempts.",
				"retry_after":      int(ttl.Seconds()),
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
				"error":                       "email not verified",
				"message":                     "Please verify your email address before logging in. Check your inbox for the verification link.",
				"requires_email_verification": true,
			})
			return
		}
		// Check if user is banned (profile has ban_reason)
		var profile models.Profile
		err = db.DB.Where("user_id = ?", user.ID).First(&profile).Error
		if err == nil && profile.BanReason != nil && *profile.BanReason != "" {
			c.JSON(http.StatusForbidden, gin.H{
				"error":      "account_banned",
				"message":    "Your account has been banned.",
				"ban_reason": *profile.BanReason,
			})
			return
		}
		// Default for inactive accounts: email not verified
		c.JSON(http.StatusForbidden, gin.H{
			"error":                       "email not verified",
			"message":                     "Please verify your email address before logging in. Check your inbox for the verification link.",
			"requires_email_verification": true,
		})
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
					"error":              "invalid username or password",
					"warning":            lockout.FormatLockoutMessage(remaining, 0),
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
			"error":   "TOTP required for admin accounts",
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
		// Don't fail the login, just log the error
	}

	// Set httpOnly cookies for tokens
	cookieHelper := cookie.Helper()
	cookieHelper.SetAuthTokens(c, accessToken, refreshToken, cookieMaxAge)

	// In a real app we might update last_login here
	c.JSON(http.StatusOK, gin.H{
		"access_token":  accessToken, // Also return in body for backwards compatibility
		"refresh_token": refreshToken,
		"user": gin.H{
			"id":       user.ID,
			"username": user.Username,
			"is_admin": user.IsSuperuser,
		},
	})
}

// Refresh – POST /api/v2/auth/refresh
// @Description Refresh access token using refresh token from httpOnly cookie. Implements token rotation.
// @Tags Authentication
// @Summary Refresh access token
// @Produce json
// @Success 200 {object} map[string]string "New access and refresh tokens"
// @Failure 401 {object} map[string]string "Invalid or expired refresh token"
// @Router /auth/refresh [post]
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
		c.JSON(http.StatusUnauthorized, gin.H{"error": "refresh token not found"})
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
		// Don't fail the request, just log the error
	}
	tx.Commit()

	// Set httpOnly cookies for tokens
	cookieHelper := cookie.Helper()

	// Calculate remaining time for refresh token cookie based on original expiration
	remainingSeconds := int(refreshTokenModel.ExpiresAt.Sub(time.Now()).Seconds())
	if remainingSeconds < 0 {
		remainingSeconds = cookie.RefreshTokenDuration // Fallback to 7 days if somehow expired
	}

	cookieHelper.SetAuthTokens(c, accessToken, newRefreshToken, remainingSeconds)

	c.JSON(http.StatusOK, gin.H{
		"access_token":  accessToken,
		"refresh_token": newRefreshToken,
	})
}

// Logout – POST /api/v2/auth/logout
// @Description Logout user and revoke refresh tokens. Clears authentication cookies.
// @Tags Authentication
// @Summary User logout
// @Produce json
// @Success 200 {object} map[string]string "Logout successful"
// @Router /auth/logout [post]
func Logout(c *gin.Context) {
	cookieHelper := cookie.Helper()

	// Get refresh token from httpOnly cookie
	refreshToken, err := c.Cookie("refresh_token")
	if err != nil {
		// No refresh token in cookie, user might already be logged out
		// Still clear any access token cookie and return success
		cookieHelper.ClearAuthTokens(c)
		c.JSON(http.StatusOK, gin.H{"message": "logged out successfully"})
		return
	}

	// Get user ID from context (set by auth middleware from access token)
	userID := c.GetUint("user_id")
	if userID == 0 {
		// No valid access token, but still clear cookies
		cookieHelper.ClearAuthTokens(c)
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
		db.DB.Model(&models.RefreshToken{}).
			Where("token = ? AND user_id = ?", refreshToken, claims.UserID).
			Update("revoked_at", &revokedNow)
	}

	// Clear cookies
	cookieHelper.ClearAuthTokens(c)

	c.JSON(http.StatusOK, gin.H{"message": "logged out successfully"})
}

// RevokeAllSessions - POST /api/v2/auth/revoke-all-sessions
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

// CheckTOTPRequired checks if the user has TOTP enabled
func CheckTOTPRequired(userID uint) bool {
	var totp models.TotpDevice
	err := db.DB.Where("user_id = ?", userID).First(&totp).Error
	return err == nil && totp.Confirmed
}
