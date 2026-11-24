package v2

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"
	"sort"
	"time"

	"github.com/CLAOJ/claoj/auth"
	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
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

	// can_organization_admin gates the organization admin/slots/open fields in
	// the FE form, matching the judge.organization_admin check on the API.
	// LoadUserAccess resolves superuser-as-all-perms itself, so no gin context
	// dependency.
	access, err := auth.LoadUserAccess(profile.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}

	// Everyone gets their own effective Django permission codenames (direct
	// plus group grants, never the whole catalogue) so permission-gated public
	// controls - e.g. the suggest link - work for non-staff holders too.
	// Superusers are still told apart by is_superuser.
	perms := append([]string{}, access.PermList...)
	sort.Strings(perms)

	// An impersonated session keeps the *target's* permissions (that is what
	// the UI must reflect); the banner only needs to know who is driving it.
	impersonatorID := c.GetUint("impersonator_id")
	impersonatorUsername := ""
	if impersonatorID != 0 {
		var impersonator models.AuthUser
		if err := db.DB.Select("username").First(&impersonator, impersonatorID).Error; err == nil {
			impersonatorUsername = impersonator.Username
		}
	}

	// We don't expose sensitive info like Password or TOTP keys.
	// is_staff and is_admin drive the frontend admin surface: the admin gate
	// (AdminAccessWrapper / AdminSidebar) hides all admin UI unless is_staff is
	// truthy, and reads is_admin for the super-admin label. is_admin mirrors the
	// login response (is_admin = IsSuperuser).
	c.JSON(http.StatusOK, gin.H{
		"id":           authUser.ID,
		"username":     authUser.Username,
		"email":        authUser.Email,
		"is_active":    authUser.IsActive,
		"is_staff":     authUser.IsStaff,
		"is_admin":     authUser.IsSuperuser,
		"is_superuser": authUser.IsSuperuser,
		"date_joined":  authUser.DateJoined,
		"profile": gin.H{
			"points":   profile.Points,
			"rating":   profile.Rating,
			"about":    profile.About,
			"timezone": profile.Timezone,
		},
		"can_organization_admin": access.HasPerm("judge.organization_admin"),
		"permissions":            perms,
		"is_impersonating":       impersonatorID != 0,
		"impersonator_username":  impersonatorUsername,
	})
}

// GetAPIToken returns the current user's API token info (without exposing the token itself if already generated)
func GetAPIToken(c *gin.Context) {
	userID := c.GetUint("user_id")

	var profile models.Profile
	if err := db.DB.Where("user_id = ?", userID).First(&profile).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}

	if profile.ApiToken == nil {
		c.JSON(http.StatusOK, gin.H{
			"has_token":  false,
			"token":      "",
			"created_at": nil,
		})
		return
	}

	// Token exists - for security, we only return whether it exists, not the actual token
	// User needs to regenerate to get a new token
	c.JSON(http.StatusOK, gin.H{
		"has_token": true,
		"token":     "", // Don't expose existing token
		"message":   "API token exists. Generate a new token to get the value.",
	})
}

// GenerateAPIToken creates or regenerates an API token for the current user
func GenerateAPIToken(c *gin.Context) {
	userID := c.GetUint("user_id")

	// Generate random token (32 bytes = 64 hex characters)
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}
	token := hex.EncodeToString(tokenBytes)

	// Update profile with new token
	if err := db.DB.Model(&models.Profile{}).Where("user_id = ?", userID).Update("api_token", token).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save token"})
		return
	}

	// Update data_last_downloaded to track when token was generated (for GDPR tracking)
	db.DB.Model(&models.Profile{}).Where("user_id = ?", userID).Update("data_last_downloaded", time.Now())

	c.JSON(http.StatusOK, gin.H{
		"token":   token,
		"message": "API token generated successfully. Store it securely - it won't be shown again.",
		"warning": "This token grants full access to your account. Do not share it with anyone.",
	})
}

// RevokeAPIToken deletes the current user's API token
func RevokeAPIToken(c *gin.Context) {
	userID := c.GetUint("user_id")

	if err := db.DB.Model(&models.Profile{}).Where("user_id = ?", userID).Update("api_token", nil).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to revoke token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "API token revoked successfully",
	})
}
