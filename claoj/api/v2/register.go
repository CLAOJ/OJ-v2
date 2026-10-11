package v2

import (
	"errors"
	"net/http"
	"regexp"
	"time"

	"github.com/CLAOJ/claoj/auth"
	"github.com/CLAOJ/claoj/config"
	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

var usernameRegex = regexp.MustCompile(`^\w+$`)

type RegisterRequest struct {
	Username string `json:"username" binding:"required"`
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
	FullName string `json:"full_name"`
	Language string `json:"language"` // key, e.g. "python3"
	Timezone string `json:"timezone"`
}

// newAccount is what createAccount needs to insert an auth_user row and its
// judge_profile.
type newAccount struct {
	Username       string
	Email          string
	HashedPassword string
	FullName       string
	Language       string // judge_language.key; empty means the site default
	Timezone       string // empty means UTC
	IsActive       bool
}

var (
	errUsernameTaken = errors.New("username already taken")
	errEmailTaken    = errors.New("email already taken")
)

// createAccount inserts the auth_user and judge_profile rows inside tx and
// returns the new profile. Callers validate the username and password first.
func createAccount(tx *gorm.DB, a newAccount) (models.Profile, error) {
	var count int64
	tx.Model(&models.AuthUser{}).Where("username = ?", a.Username).Count(&count)
	if count > 0 {
		return models.Profile{}, errUsernameTaken
	}
	tx.Model(&models.AuthUser{}).Where("email = ?", a.Email).Count(&count)
	if count > 0 {
		return models.Profile{}, errEmailTaken
	}

	user := models.AuthUser{
		Username:   a.Username,
		Email:      a.Email,
		Password:   a.HashedPassword,
		FirstName:  a.FullName,
		IsActive:   a.IsActive,
		DateJoined: time.Now(),
	}
	if err := tx.Create(&user).Error; err != nil {
		return models.Profile{}, err
	}
	// GORM skips boolean zero values on columns with a default, so is_active
	// has to be written explicitly.
	if err := tx.Model(&models.AuthUser{}).Where("id = ?", user.ID).Update("is_active", a.IsActive).Error; err != nil {
		return models.Profile{}, err
	}

	langKey := a.Language
	if langKey == "" {
		langKey = config.C.App.DefaultLanguage
	}
	var lang models.Language
	if err := tx.Where("`key` = ?", langKey).First(&lang).Error; err != nil {
		// fallback to any
		tx.First(&lang)
	}

	tz := a.Timezone
	if tz == "" {
		tz = "UTC"
	}

	profile := models.Profile{
		UserID:     user.ID,
		LanguageID: lang.ID,
		Timezone:   tz,
		LastAccess: time.Now(),
		// judge_profile.math_engine is varchar(4) with Django choices
		// tex/svg/mml/jax/auto. "auto" is v1's MATHOID_DEFAULT_TYPE; the
		// old "mathjax" value overflowed the column (Error 1406) and broke
		// registration entirely.
		MathEngine: "auto",
	}
	if err := tx.Create(&profile).Error; err != nil {
		return models.Profile{}, err
	}
	return profile, nil
}

// Register – POST /api/v2/auth/register
func Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, apiError(err.Error()))
		return
	}

	// 1. Validate Username
	if !usernameRegex.MatchString(req.Username) || len(req.Username) > 30 {
		c.JSON(http.StatusBadRequest, apiError("invalid username: must be 1-30 chars, alphanumeric or underscore"))
		return
	}

	// 2. Validate Password Strength
	if err := validatePasswordStrength(req.Password); err != nil {
		c.JSON(http.StatusBadRequest, apiError(err.Error()))
		return
	}

	// 3. Hash Password
	hashedPassword, err := auth.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, apiError("failed to hash password"))
		return
	}

	var profile models.Profile
	err = db.DB.Transaction(func(tx *gorm.DB) error {
		var txErr error
		profile, txErr = createAccount(tx, newAccount{
			Username:       req.Username,
			Email:          req.Email,
			HashedPassword: hashedPassword,
			FullName:       req.FullName,
			Language:       req.Language,
			Timezone:       req.Timezone,
			IsActive:       false, // require email verification before activation
		})
		return txErr
	})

	if err != nil {
		c.JSON(http.StatusBadRequest, apiError(err.Error()))
		return
	}

	// Send verification email asynchronously
	go func() {
		if err := SendVerificationEmailOnRegistration(profile.UserID, req.Email, req.Username); err != nil {
			// Log error but don't fail registration
		}
	}()

	c.JSON(http.StatusCreated, gin.H{
		"message": "account created successfully. Please check your email to verify your account.",
		"requires_verification": true,
	})
}

// validatePasswordStrength checks password meets minimum security requirements
func validatePasswordStrength(password string) error {
	// Minimum length: 8 characters
	if len(password) < 8 {
		return errors.New("password must be at least 8 characters long")
	}

	// Require at least one uppercase letter
	hasUpper := false
	for _, r := range password {
		if r >= 'A' && r <= 'Z' {
			hasUpper = true
			break
		}
	}
	if !hasUpper {
		return errors.New("password must contain at least one uppercase letter")
	}

	// Require at least one lowercase letter
	hasLower := false
	for _, r := range password {
		if r >= 'a' && r <= 'z' {
			hasLower = true
			break
		}
	}
	if !hasLower {
		return errors.New("password must contain at least one lowercase letter")
	}

	// Require at least one number
	hasNumber := false
	for _, r := range password {
		if r >= '0' && r <= '9' {
			hasNumber = true
			break
		}
	}
	if !hasNumber {
		return errors.New("password must contain at least one number")
	}

	return nil
}
