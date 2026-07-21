package auth

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"time"

	"github.com/CLAOJ/claoj/auth"
	"github.com/CLAOJ/claoj/auth/tokenstore"
	"github.com/CLAOJ/claoj/config"
	"github.com/CLAOJ/claoj/cookie"
	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/email"
	"github.com/CLAOJ/claoj/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

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

	// Store the token in the one-time token store (Redis-backed, outside
	// the shared MySQL schema) instead of a v2-only DB table.
	if err := OneTimeTokens.Issue(tokenstore.KindPasswordReset, tokenStr, user.ID, 1*time.Hour); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create reset token"})
		return
	}

	// Build reset link
	resetLink := config.C.App.SiteFullURL + "/reset-password?token=" + tokenStr

	// Send email
	if err := email.SendPasswordResetEmail(user.Email, user.Username, resetLink); err != nil {
		// Log error but still return success to avoid enumeration
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

	// Atomically consume the reset token (single-use: GETDEL under the
	// hood). ok=false covers both "never issued" and "already used /
	// expired" - the caller doesn't need to distinguish.
	uid, ok, err := OneTimeTokens.Consume(tokenstore.KindPasswordReset, req.Token)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid or expired token"})
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
		return tx.Model(&models.AuthUser{}).Where("id = ?", uid).Update("password", hashedPassword).Error
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to reset password"})
		return
	}

	// Invalidate any other outstanding reset tokens for this user. The
	// consumed token above is already gone; this is defense-in-depth in
	// case multiple reset emails were requested.
	if err := OneTimeTokens.Invalidate(tokenstore.KindPasswordReset, uid); err != nil {
		// Best-effort: password already changed successfully.
	}

	c.JSON(http.StatusOK, gin.H{"message": "Password reset successfully"})
}

// PasswordChange - POST /api/v2/auth/password/change
//
// Changes the password of the currently authenticated user. This is distinct
// from the PasswordReset* pair above, which is the logged-out "I forgot my
// password" flow driven by an emailed one-time token. The settings page has
// always posted here; the route simply didn't exist, so every attempt to
// change a password 404'd.
func PasswordChange(c *gin.Context) {
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

	var req struct {
		CurrentPassword string `json:"current_password" binding:"required"`
		NewPassword     string `json:"new_password" binding:"required,min=6"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user models.AuthUser
	if err := db.DB.Where("id = ?", userID).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	// Knowing the current password is what authorizes the change: without it
	// anyone with a stolen access token could lock the real owner out.
	valid, err := auth.CheckPassword(req.CurrentPassword, user.Password)
	if err != nil || !valid {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "current password is incorrect"})
		return
	}

	if req.NewPassword == req.CurrentPassword {
		c.JSON(http.StatusBadRequest, gin.H{"error": "new password must differ from the current one"})
		return
	}

	hashedPassword, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
		return
	}

	if err := db.DB.Model(&models.AuthUser{}).Where("id = ?", userID).
		Update("password", hashedPassword).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to change password"})
		return
	}

	// A password change must not leave sessions opened with the old password
	// alive — that's the whole point of changing it after a suspected
	// compromise. Revoke every refresh token for this user, then immediately
	// mint a fresh pair so the tab that made the change stays signed in.
	if err := RefreshStore.RevokeAllForUser(userID); err != nil {
		// Best-effort: the password itself is already changed.
	}

	accessToken, refreshToken, familyID, err := auth.GenerateTokens(user.ID, user.Username, user.IsSuperuser, "", false)
	if err != nil {
		// The password change succeeded but we couldn't re-establish this
		// session. Clear the cookies so the user is sent back to the login
		// page rather than left holding tokens that no longer refresh.
		cookie.Helper().ClearAuthTokens(c)
		c.JSON(http.StatusOK, gin.H{"message": "Password changed successfully, please log in again"})
		return
	}

	if err := RefreshStore.Save(refreshToken, tokenstore.Entry{
		UserID:    user.ID,
		FamilyID:  familyID,
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
		CreatedAt: time.Now(),
		UserAgent: c.Request.UserAgent(),
		ClientIP:  c.ClientIP(),
	}); err != nil {
		// Best-effort, mirroring Login.
	}

	cookie.Helper().SetAuthTokens(c, accessToken, refreshToken, cookie.RefreshTokenDuration)

	c.JSON(http.StatusOK, gin.H{"message": "Password changed successfully"})
}
