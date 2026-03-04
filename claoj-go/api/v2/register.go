package v2

import (
	"errors"
	"net/http"
	"regexp"
	"time"

	"github.com/CLAOJ/claoj-go/auth"
	"github.com/CLAOJ/claoj-go/config"
	"github.com/CLAOJ/claoj-go/db"
	"github.com/CLAOJ/claoj-go/models"
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

	// 2. Hash Password
	hashedPassword, err := auth.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, apiError("failed to hash password"))
		return
	}

	var userID uint
	var userEmail string
	var userName string

	err = db.DB.Transaction(func(tx *gorm.DB) error {
		// 3. Check uniqueness
		var count int64
		tx.Model(&models.AuthUser{}).Where("username = ?", req.Username).Count(&count)
		if count > 0 {
			return errors.New("username already taken")
		}
		tx.Model(&models.AuthUser{}).Where("email = ?", req.Email).Count(&count)
		if count > 0 {
			return errors.New("email already taken")
		}

		// 4. Create AuthUser
		user := models.AuthUser{
			Username:   req.Username,
			Email:      req.Email,
			Password:   hashedPassword,
			FirstName:  req.FullName,
			IsActive:   true,
			DateJoined: time.Now(),
		}
		if err := tx.Create(&user).Error; err != nil {
			return err
		}

		userID = user.ID
		userEmail = user.Email
		userName = user.Username

		// 5. Get default language/timezone if missing
		langKey := req.Language
		if langKey == "" {
			langKey = config.C.App.DefaultLanguage
		}
		var lang models.Language
		if err := tx.Where("`key` = ?", langKey).First(&lang).Error; err != nil {
			// fallback to any
			tx.First(&lang)
		}

		tz := req.Timezone
		if tz == "" {
			tz = "UTC"
		}

		// 6. Create Profile
		profile := models.Profile{
			UserID:     user.ID,
			LanguageID: lang.ID,
			Timezone:   tz,
			LastAccess: time.Now(),
			MathEngine: "mathjax", // Default
		}
		return tx.Create(&profile).Error
	})

	if err != nil {
		c.JSON(http.StatusBadRequest, apiError(err.Error()))
		return
	}

	// Send verification email asynchronously
	go func() {
		if err := SendVerificationEmailOnRegistration(userID, userEmail, userName); err != nil {
			// Log error but don't fail registration
		}
	}()

	c.JSON(http.StatusCreated, gin.H{
		"message": "account created successfully. Please check your email to verify your account.",
		"requires_verification": true,
	})
}
