package v2

import (
	"errors"
	"log"
	"net/http"
	"net/mail"
	"strconv"

	authHandlers "github.com/CLAOJ/claoj/api/v2/auth"
	"github.com/CLAOJ/claoj/auth"
	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// AdminUserCreate - POST /api/v2/admin/users
// Creates an account on behalf of a user. Unlike self-registration the
// account is active immediately unless is_active is false.
func AdminUserCreate(c *gin.Context) {
	var input struct {
		Username string `json:"username" binding:"required"`
		Email    string `json:"email" binding:"required"`
		Password string `json:"password" binding:"required"`
		FullName string `json:"full_name"`
		IsActive *bool  `json:"is_active"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, apiBindError(err))
		return
	}

	if !usernameRegex.MatchString(input.Username) || len(input.Username) > 30 {
		c.JSON(http.StatusBadRequest, apiUserError("invalid username: must be 1-30 chars, alphanumeric or underscore"))
		return
	}
	if _, err := mail.ParseAddress(input.Email); err != nil {
		c.JSON(http.StatusBadRequest, apiUserError("invalid email address"))
		return
	}
	if err := validatePasswordStrength(input.Password); err != nil {
		c.JSON(http.StatusBadRequest, apiUserError(err.Error()))
		return
	}

	hashedPassword, err := auth.HashPassword(input.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, apiUserError("failed to hash password"))
		return
	}

	active := true
	if input.IsActive != nil {
		active = *input.IsActive
	}

	var profile models.Profile
	err = db.DB.Transaction(func(tx *gorm.DB) error {
		var txErr error
		profile, txErr = createAccount(tx, newAccount{
			Username:       input.Username,
			Email:          input.Email,
			HashedPassword: hashedPassword,
			FullName:       input.FullName,
			IsActive:       active,
		})
		return txErr
	})
	if err != nil {
		if errors.Is(err, errUsernameTaken) || errors.Is(err, errEmailTaken) {
			c.JSON(http.StatusConflict, apiUserError(err.Error()))
			return
		}
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"user": gin.H{
			"id":       profile.ID,
			"username": input.Username,
		},
	})
}

// AdminUserSetPassword - POST /api/v2/admin/user/:id/password
// Sets a new password for the user with judge_profile.id = :id and signs
// them out everywhere.
func AdminUserSetPassword(c *gin.Context) {
	profileID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, apiUserError("invalid user ID"))
		return
	}

	var input struct {
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, apiBindError(err))
		return
	}
	if err := validatePasswordStrength(input.Password); err != nil {
		c.JSON(http.StatusBadRequest, apiUserError(err.Error()))
		return
	}

	var profile models.Profile
	if err := db.DB.Preload("User").First(&profile, uint(profileID)).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, apiUserError("user not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	// Setting a password hands the caller the account, so only a superuser
	// may do it to a staff or superuser account.
	if (profile.User.IsStaff || profile.User.IsSuperuser) && !auth.GetAccess(c).IsSuperuser {
		c.JSON(http.StatusForbidden, apiUserError("superuser access required to set the password of a staff account"))
		return
	}

	hashedPassword, err := auth.HashPassword(input.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, apiUserError("failed to hash password"))
		return
	}
	if err := db.DB.Model(&models.AuthUser{}).Where("id = ?", profile.UserID).
		Update("password", hashedPassword).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	// Sessions opened with the old password must not survive the reset.
	revokeUserSessions(profile.UserID)

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// revokeUserSessions signs the auth_user out everywhere after a credential,
// account-status or 2FA change, so a stolen session cannot outlive it.
// RefreshStore is nil in tests.
func revokeUserSessions(authUserID uint) {
	if authHandlers.RefreshStore == nil {
		return
	}
	if err := authHandlers.RefreshStore.RevokeAllForUser(authUserID); err != nil {
		// Best-effort: the state change itself already landed.
		log.Printf("admin: revoke sessions of user %d: %v", authUserID, err)
	}
}
