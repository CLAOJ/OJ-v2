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
		c.JSON(http.StatusBadRequest, apiError(err.Error()))
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
		c.JSON(http.StatusInternalServerError, apiError("failed to hash password"))
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
		c.JSON(http.StatusBadRequest, apiError(err.Error()))
		return
	}
	if err := validatePasswordStrength(input.Password); err != nil {
		c.JSON(http.StatusBadRequest, apiUserError(err.Error()))
		return
	}

	var profile models.Profile
	if err := db.DB.Preload("User").First(&profile, uint(profileID)).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, apiError("user not found"))
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
		c.JSON(http.StatusInternalServerError, apiError("failed to hash password"))
		return
	}
	if err := db.DB.Model(&models.AuthUser{}).Where("id = ?", profile.UserID).
		Update("password", hashedPassword).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	// Sessions opened with the old password must not survive the reset.
	// RefreshStore is nil in tests.
	if authHandlers.RefreshStore != nil {
		if err := authHandlers.RefreshStore.RevokeAllForUser(profile.UserID); err != nil {
			// Best-effort: the password itself is already changed.
			log.Printf("admin: revoke sessions of user %d after password reset: %v", profile.UserID, err)
		}
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}
