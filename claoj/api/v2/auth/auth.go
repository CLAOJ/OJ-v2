package auth

import (
	"context"
	"net/http"
	"time"

	"github.com/CLAOJ/claoj/auth"
	"github.com/CLAOJ/claoj/auth/tokenstore"
	"github.com/CLAOJ/claoj/cache"
	"github.com/CLAOJ/claoj/config"
	"github.com/CLAOJ/claoj/cookie"
	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/lockout"
	"github.com/CLAOJ/claoj/models"
	"github.com/gin-gonic/gin"
)

// RefreshStore persists refresh-token sessions (outside the shared MySQL
// schema) so token rotation-reuse detection works across requests. It is
// set once at startup in main.go (Redis-backed, or an in-memory fallback
// when Redis is unavailable) and overridden with tokenstore.NewMemoryStore()
// in tests.
var RefreshStore tokenstore.Store

// RotationGracePeriod is how long after a refresh token is rotated out that
// presenting it again is treated as a concurrent-refresh collision (two tabs
// racing) rather than a replay attack.
//
// Racing tabs fire within milliseconds of each other, so this only needs to
// cover one slow round trip. Keeping it short keeps the replay-detection hole
// small. The hole is narrow to begin with: the collision branch issues *no*
// tokens, it merely defers the family revocation, and the very next reuse
// outside the window still kills the family.
//
// Exported so tests can shrink it instead of sleeping.
var RotationGracePeriod = 10 * time.Second

// OneTimeTokens issues and consumes single-use tokens (password reset,
// email verification) outside the shared MySQL schema. It is set once at
// startup in main.go (Redis-backed, or an in-memory fallback when Redis is
// unavailable) and overridden with tokenstore.NewMemoryOneTime() in tests.
// The sibling v2 package (api/v2/verify.go) also uses it via this exported
// var — that package imports this one (not vice versa), so there's no
// import cycle.
var OneTimeTokens tokenstore.OneTimeStore

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
		hasPendingVerification, _ := OneTimeTokens.HasOutstanding(tokenstore.KindEmailVerify, user.ID)
		if hasPendingVerification {
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
		err := db.DB.Where("user_id = ?", user.ID).First(&profile).Error
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

	// Store refresh token for revocation tracking (rotation-reuse detection)
	userAgent := c.Request.UserAgent()
	clientIP := c.ClientIP()
	refreshTokenTTL := 7 * 24 * time.Hour
	cookieMaxAge := 7 * 24 * 60 * 60 // 7 days in seconds
	if req.RememberMe {
		refreshTokenTTL = 30 * 24 * time.Hour
		cookieMaxAge = 30 * 24 * 60 * 60 // 30 days in seconds
	}
	if err := RefreshStore.Save(refreshToken, tokenstore.Entry{
		UserID:    user.ID,
		FamilyID:  familyID,
		ExpiresAt: time.Now().Add(refreshTokenTTL),
		CreatedAt: time.Now(),
		UserAgent: userAgent,
		ClientIP:  clientIP,
	}); err != nil {
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

	// Look up the session for this token.
	entry, found, err := RefreshStore.Get(refreshToken)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to look up refresh token"})
		return
	}
	if !found {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "refresh token not found"})
		return
	}

	if entry.Revoked {
		// This token was already rotated out (or explicitly revoked) but is
		// being presented again. Two very different things look identical here:
		//
		//  1. A genuine replay — an attacker holding a stolen copy of an old
		//     token. The whole family must die.
		//  2. A benign rotation collision — two tabs of the same browser hit a
		//     401 at the same moment and both refreshed with the token that was
		//     current when they started. One wins; the loser arrives moments
		//     later with a token the winner just rotated out.
		//
		// Treating (2) as (1) logged people out at random, because killing the
		// family also revokes the freshly-issued token the winner is now using.
		// Inside a short window after rotation, assume the collision: fail just
		// this request and leave the family intact. The loser's next attempt
		// picks up the winner's cookie from the shared jar and succeeds.
		// Strict `<` so that a zero RotationGracePeriod disables the grace
		// entirely — the clock can report an elapsed time of exactly zero when
		// both reads land in the same tick.
		if !entry.RevokedAt.IsZero() && time.Since(entry.RevokedAt) < RotationGracePeriod {
			c.JSON(http.StatusConflict, gin.H{"error": "refresh already in progress, retry"})
			return
		}
		if err := RefreshStore.RevokeFamily(entry.FamilyID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to revoke token family"})
			return
		}
		c.JSON(http.StatusUnauthorized, gin.H{"error": "refresh token has been revoked"})
		return
	}

	if entry.ExpiresAt.Before(time.Now()) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "refresh token has expired"})
		return
	}

	// Generate new token pair with same family ID (maintain original remember_me setting via family)
	accessToken, newRefreshToken, familyID, err := auth.GenerateTokens(claims.UserID, claims.Username, claims.IsAdmin, entry.FamilyID, false)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate tokens"})
		return
	}

	// Revoke old token and store new one
	if err := RefreshStore.Revoke(refreshToken); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to revoke token"})
		return
	}

	// Store new refresh token - maintain same TTL as original
	userAgent := c.Request.UserAgent()
	clientIP := c.ClientIP()
	if err := RefreshStore.Save(newRefreshToken, tokenstore.Entry{
		UserID:    claims.UserID,
		FamilyID:  familyID,
		ExpiresAt: entry.ExpiresAt, // Maintain original expiration
		CreatedAt: time.Now(),
		UserAgent: userAgent,
		ClientIP:  clientIP,
	}); err != nil {
		// Don't fail the request, just log the error
	}

	// Set httpOnly cookies for tokens
	cookieHelper := cookie.Helper()

	// Calculate remaining time for refresh token cookie based on original expiration
	remainingSeconds := int(entry.ExpiresAt.Sub(time.Now()).Seconds())
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

	// Revoke the refresh token (best-effort: logout still succeeds and
	// clears cookies even if the token was already gone/unknown).
	RefreshStore.Revoke(refreshToken)

	// Clear cookies
	cookieHelper.ClearAuthTokens(c)

	c.JSON(http.StatusOK, gin.H{"message": "logged out successfully"})
}

// RevokeAllSessions - POST /api/v2/auth/revoke-all-sessions
// Revokes all refresh tokens for the current user (force logout all devices)
func RevokeAllSessions(c *gin.Context) {
	// BUG FIX: the auth middleware sets "user_id" on the context (see
	// auth/middleware.go), not "userID" — the old c.GetUint("userID") read
	// always returned the zero value, so this handler 401'd for everyone.
	rawUserID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	userID, ok := rawUserID.(uint)
	if !ok || userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	if err := RefreshStore.RevokeAllForUser(userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to revoke sessions"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "all sessions revoked",
	})
}

// CheckTOTPRequired checks if the user has TOTP enabled
func CheckTOTPRequired(userID uint) bool {
	var totp models.TotpDevice
	err := db.DB.Where("user_id = ?", userID).First(&totp).Error
	return err == nil && totp.Confirmed
}
