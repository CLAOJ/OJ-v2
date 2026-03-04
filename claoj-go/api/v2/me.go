package v2

import (
	"errors"
	"net/http"

	"github.com/CLAOJ/claoj-go/db"
	"github.com/CLAOJ/claoj-go/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// CurrentUser retrieves the detailed profile of the authenticated user
func CurrentUser(c *gin.Context) {
	// The UserID is guaranteed to be set if the RequiredMiddleware allowed this route
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "user ID missing from context"})
		return
	}

	var authUser models.AuthUser
	if err := db.DB.First(&authUser, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}

	var profile models.Profile
	if err := db.DB.Where("user_id = ?", userID).First(&profile).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "profile not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}

	// We don't expose sensitive info like Password or TOTP keys
	c.JSON(http.StatusOK, gin.H{
		"id":           authUser.ID,
		"username":     authUser.Username,
		"email":        authUser.Email,
		"is_active":    authUser.IsActive,
		"is_superuser": authUser.IsSuperuser,
		"date_joined":  authUser.DateJoined,
		"profile": gin.H{
			"points":   profile.Points,
			"rating":   profile.Rating,
			"about":    profile.About,
			"timezone": profile.Timezone,
		},
	})
}
