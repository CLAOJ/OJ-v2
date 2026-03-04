package v2

import (
	"crypto/sha256"
	"encoding/hex"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/CLAOJ/claoj-go/db"
	"github.com/CLAOJ/claoj-go/models"
	"github.com/gin-gonic/gin"
)

// resolveUserProfile fetches the AuthUser and Profile for the currently authenticated user.
// Returns (user, profile, ok). If ok is false, the response has already been sent to the client.
func resolveUserProfile(c *gin.Context) (*models.AuthUser, *models.Profile, bool) {
	uid, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "user ID missing in context"})
		return nil, nil, false
	}
	userID := uid.(uint)

	var user models.AuthUser
	if err := db.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
		return nil, nil, false
	}

	var profile models.Profile
	if err := db.DB.Where("user_id = ?", user.ID).First(&profile).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "profile not found"})
		return nil, nil, false
	}

	return &user, &profile, true
}

// getGravatarURL returns the Gravatar URL for a given email address
// Note: Gravatar officially supports SHA-256 for better security
func getGravatarURL(email string, size int) string {
	// Trim and lowercase email for hash
	email = strings.ToLower(strings.TrimSpace(email))

	// Compute SHA-256 hash (Gravatar supports SHA-256)
	hash := sha256.Sum256([]byte(email))
	hashStr := hex.EncodeToString(hash[:])

	// Use gravatar.com with d=mp (mystery person) as default
	if size > 0 {
		return "https://www.gravatar.com/avatar/" + hashStr + "?s=" + strconv.Itoa(size) + "&d=mp"
	}
	return "https://www.gravatar.com/avatar/" + hashStr + "?d=mp"
}

// getAvatarURL returns the avatar URL for a profile (Gravatar-based)
func getAvatarURL(profile *models.Profile) string {
	if profile == nil {
		return ""
	}
	// Check for logo override from organization
	// For now, just use Gravatar
	return getGravatarURL(profile.User.Email, 80)
}

func parsePagination(c *gin.Context) (int, int) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "100"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 1000 {
		pageSize = 100
	}
	return page, pageSize
}

// apiError is a helper for consistent error responses
// Logs the actual error but returns a generic message to avoid information disclosure
func apiError(msg string) gin.H {
	// Log the actual error for debugging
	log.Printf("API error: %s", msg)
	// Return generic error message to client
	return gin.H{"error": "An error occurred. Please try again."}
}

// apiList is a helper for consistent list responses
func apiList(data interface{}) gin.H {
	return gin.H{"data": data}
}

// parseUint parses a string to uint
func parseUint(s string, result *uint) error {
	val, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return err
	}
	*result = uint(val)
	return nil
}
