package v2

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"
	"time"

	authHandlers "github.com/CLAOJ/claoj/api/v2/auth"
	"github.com/CLAOJ/claoj/auth/tokenstore"
	"github.com/CLAOJ/claoj/config"
	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/email"
	"github.com/CLAOJ/claoj/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// VerifyEmailRequest - POST /api/v2/auth/verify-email
// Verifies email with token
type VerifyEmailRequest struct {
	Token string `json:"token" binding:"required"`
}

func VerifyEmail(c *gin.Context) {
	var req VerifyEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, apiError(err.Error()))
		return
	}

	// Atomically consume the verification token (single-use: GETDEL under
	// the hood). ok=false covers both "never issued" and "already used /
	// expired".
	uid, ok, err := authHandlers.OneTimeTokens.Consume(tokenstore.KindEmailVerify, req.Token)
	if err != nil {
		c.JSON(http.StatusInternalServerError, apiError("database error"))
		return
	}
	if !ok {
		c.JSON(http.StatusBadRequest, apiError("invalid or expired verification token"))
		return
	}

	// Mark user as having verified email
	err = db.DB.Transaction(func(tx *gorm.DB) error {
		// Update user's email verification status
		// Note: Django uses is_active for this purpose in some configurations
		// We'll update a profile field or set is_active to true
		return tx.Model(&models.AuthUser{}).Where("id = ?", uid).Update("is_active", true).Error
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, apiError("failed to verify email"))
		return
	}

	// Invalidate any other outstanding verification tokens for this user.
	if err := authHandlers.OneTimeTokens.Invalidate(tokenstore.KindEmailVerify, uid); err != nil {
		// Best-effort: user already verified successfully.
	}

	c.JSON(http.StatusOK, gin.H{"message": "Email verified successfully"})
}

// ResendVerificationRequest - POST /api/v2/auth/resend-verification
// Resends verification email
type ResendVerificationRequest struct {
	Email string `json:"email,omitempty"`
}

func ResendVerification(c *gin.Context) {
	var req ResendVerificationRequest
	c.ShouldBindJSON(&req) // Email is optional for authenticated users

	var user models.AuthUser

	// Check if user is authenticated (via access token)
	userID, exists := c.Get("user_id")
	if exists && userID != nil && userID.(uint) > 0 {
		// Authenticated user - use their account
		if err := db.DB.First(&user, userID).Error; err != nil {
			c.JSON(http.StatusUnauthorized, apiError("user not found"))
			return
		}
	} else {
		// Unauthenticated - require email
		if req.Email == "" {
			c.JSON(http.StatusBadRequest, apiError("email is required"))
			return
		}
		// Find user by email
		if err := db.DB.Where("email = ?", req.Email).First(&user).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				// Don't reveal if email exists
				c.JSON(http.StatusOK, gin.H{"message": "If the email exists and is not verified, a verification link has been sent"})
				return
			}
			c.JSON(http.StatusInternalServerError, apiError("database error"))
			return
		}
	}

	// Check if user is already verified (is_active = true)
	if user.IsActive {
		c.JSON(http.StatusOK, gin.H{"message": "Email is already verified"})
		return
	}

	// Invalidate any existing tokens for this user
	if err := authHandlers.OneTimeTokens.Invalidate(tokenstore.KindEmailVerify, user.ID); err != nil {
		// Best-effort cleanup; proceed to issue a new token regardless.
	}

	// Generate new verification token
	token := make([]byte, 32)
	if _, err := rand.Read(token); err != nil {
		c.JSON(http.StatusInternalServerError, apiError("failed to generate token"))
		return
	}
	tokenStr := hex.EncodeToString(token)

	// Store the token in the one-time token store (Redis-backed, outside
	// the shared MySQL schema) instead of a v2-only DB table.
	if err := authHandlers.OneTimeTokens.Issue(tokenstore.KindEmailVerify, tokenStr, user.ID, 24*time.Hour); err != nil {
		c.JSON(http.StatusInternalServerError, apiError("failed to create verification token"))
		return
	}

	// Build verification link
	verifyLink := config.C.App.SiteFullURL + "/verify-email?token=" + tokenStr

	// Send verification email
	if err := email.SendVerificationEmail(user.Email, user.Username, verifyLink); err != nil {
		// Log error but still return success to avoid enumeration
		c.JSON(http.StatusOK, gin.H{"message": "If the email exists and is not verified, a verification link has been sent"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Verification email sent. Please check your inbox."})
}

// SendVerificationEmailOnRegistration sends verification email to newly registered user
func SendVerificationEmailOnRegistration(userID uint, emailAddr, username string) error {
	// Invalidate any existing tokens for this user
	if err := authHandlers.OneTimeTokens.Invalidate(tokenstore.KindEmailVerify, userID); err != nil {
		// Best-effort cleanup; proceed to issue a new token regardless.
	}

	// Generate verification token
	token := make([]byte, 32)
	if _, err := rand.Read(token); err != nil {
		return err
	}
	tokenStr := hex.EncodeToString(token)

	// Store the token in the one-time token store (Redis-backed, outside
	// the shared MySQL schema) instead of a v2-only DB table.
	if err := authHandlers.OneTimeTokens.Issue(tokenstore.KindEmailVerify, tokenStr, userID, 24*time.Hour); err != nil {
		return err
	}

	// Build verification link
	verifyLink := config.C.App.SiteFullURL + "/verify-email?token=" + tokenStr

	// Send verification email
	return email.SendVerificationEmail(emailAddr, username, verifyLink)
}
