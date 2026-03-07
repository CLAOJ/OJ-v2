package v2

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"
	"time"

	"github.com/CLAOJ/claoj-go/config"
	"github.com/CLAOJ/claoj-go/db"
	"github.com/CLAOJ/claoj-go/email"
	"github.com/CLAOJ/claoj-go/models"
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

	// Find valid verification token
	var verificationToken models.EmailVerificationToken
	if err := db.DB.Where("token = ? AND expires_at > ?", req.Token, time.Now()).First(&verificationToken).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusBadRequest, apiError("invalid or expired verification token"))
			return
		}
		c.JSON(http.StatusInternalServerError, apiError("database error"))
		return
	}

	// Mark user as having verified email
	err := db.DB.Transaction(func(tx *gorm.DB) error {
		// Update user's email verification status
		// Note: Django uses is_active for this purpose in some configurations
		// We'll update a profile field or set is_active to true
		if err := tx.Model(&models.AuthUser{}).Where("id = ?", verificationToken.UserID).Update("is_active", true).Error; err != nil {
			return err
		}

		// Delete all verification tokens for this user
		if err := tx.Where("user_id = ?", verificationToken.UserID).Delete(&models.EmailVerificationToken{}).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, apiError("failed to verify email"))
		return
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

	// Delete any existing tokens for this user
	db.DB.Where("user_id = ?", user.ID).Delete(&models.EmailVerificationToken{})

	// Generate new verification token
	token := make([]byte, 32)
	if _, err := rand.Read(token); err != nil {
		c.JSON(http.StatusInternalServerError, apiError("failed to generate token"))
		return
	}
	tokenStr := hex.EncodeToString(token)

	// Store token in database
	verificationToken := models.EmailVerificationToken{
		UserID:    user.ID,
		Token:     tokenStr,
		Email:     user.Email,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	if err := db.DB.Create(&verificationToken).Error; err != nil {
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
	// Delete any existing tokens for this user
	db.DB.Where("user_id = ?", userID).Delete(&models.EmailVerificationToken{})

	// Generate verification token
	token := make([]byte, 32)
	if _, err := rand.Read(token); err != nil {
		return err
	}
	tokenStr := hex.EncodeToString(token)

	// Store token in database
	verificationToken := models.EmailVerificationToken{
		UserID:    userID,
		Token:     tokenStr,
		Email:     emailAddr,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	if err := db.DB.Create(&verificationToken).Error; err != nil {
		return err
	}

	// Build verification link
	verifyLink := config.C.App.SiteFullURL + "/verify-email?token=" + tokenStr

	// Send verification email
	return email.SendVerificationEmail(emailAddr, username, verifyLink)
}
