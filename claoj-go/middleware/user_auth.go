package middleware

import (
	"net/http"

	"github.com/CLAOJ/claoj-go/db"
	"github.com/CLAOJ/claoj-go/models"
	"github.com/gin-gonic/gin"
)

const (
	// UserIDContextKey is the context key for user ID
	UserIDContextKey = "user_id"
	// UsernameContextKey is the context key for username
	UsernameContextKey = "username"
	// IsAdminContextKey is the context key for admin status
	IsAdminContextKey = "is_admin"
	// UserProfileContextKey is the context key for user profile
	UserProfileContextKey = "user_profile"
	// AuthUserContextKey is the context key for auth user
	AuthUserContextKey = "auth_user"
)

// GetUser retrieves the user ID from context
func GetUser(c *gin.Context) (uint, bool) {
	userID, exists := c.Get(UserIDContextKey)
	if !exists {
		return 0, false
	}
	uid, ok := userID.(uint)
	return uid, ok
}

// MustGetUser retrieves the user ID from context, panics if not present
func MustGetUser(c *gin.Context) uint {
	userID, exists := GetUser(c)
	if !exists {
		panic("user ID not found in context - auth middleware not applied?")
	}
	return userID
}

// GetUsername retrieves the username from context
func GetUsername(c *gin.Context) string {
	username, _ := c.Get(UsernameContextKey)
	return username.(string)
}

// IsAdmin checks if the current user is an admin
func IsAdmin(c *gin.Context) bool {
	isAdmin, _ := c.Get(IsAdminContextKey)
	return isAdmin.(bool)
}

// GetUserProfile retrieves the user profile from context
// Returns nil if profile not loaded
func GetUserProfile(c *gin.Context) *models.Profile {
	profile, _ := c.Get(UserProfileContextKey)
	if profile == nil {
		return nil
	}
	p, ok := profile.(*models.Profile)
	if !ok {
		return nil
	}
	return p
}

// GetAuthUser retrieves the auth user from context
// Returns nil if user not loaded
func GetAuthUser(c *gin.Context) *models.AuthUser {
	user, _ := c.Get(AuthUserContextKey)
	if user == nil {
		return nil
	}
	u, ok := user.(*models.AuthUser)
	if !ok {
		return nil
	}
	return u
}

// WithUserProfile loads the user profile into context
// Returns the profile, or an error response if not found
func WithUserProfile() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := GetUser(c)
		if !exists {
			c.Next()
			return
		}

		var profile models.Profile
		if err := db.DB.Where("user_id = ?", userID).First(&profile).Error; err == nil {
			c.Set(UserProfileContextKey, &profile)
		}

		c.Next()
	}
}

// WithAuthUser loads the full auth user into context
func WithAuthUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := GetUser(c)
		if !exists {
			c.Next()
			return
		}

		var user models.AuthUser
		if err := db.DB.First(&user, userID).Error; err == nil {
			c.Set(AuthUserContextKey, &user)
		}

		c.Next()
	}
}

// OptionalAuth wraps a handler to provide optional authentication
// If auth fails, the handler is still called but user context will be empty
func OptionalAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// This middleware is intentionally a no-op
		// Use auth.OptionalMiddleware() from auth package instead
		c.Next()
	}
}

// RequireUser is a convenience wrapper that combines auth.RequiredMiddleware
// with helper functions for common patterns
// Use auth.RequiredMiddleware() from auth package for actual auth
func RequireUser() gin.HandlerFunc {
	// This is a placeholder - use auth.RequiredMiddleware() instead
	return func(c *gin.Context) {
		c.Next()
	}
}

// RequireAdmin is a convenience wrapper for admin checks
// Use auth.AdminRequiredMiddleware() from auth package for actual admin check
func RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := GetUser(c)
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "authorization required"})
			c.Abort()
			return
		}

		var user models.AuthUser
		if err := db.DB.First(&user, userID).Error; err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
			c.Abort()
			return
		}

		if !user.IsStaff {
			c.JSON(http.StatusForbidden, gin.H{"error": "admin access required"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequirePermission creates a middleware that checks for specific permissions
// Use auth.PermissionRequiredMiddleware() from auth package for actual permission check
func RequirePermission(permissions ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := GetUser(c)
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "authorization required"})
			c.Abort()
			return
		}

		// Check if superuser (has all permissions)
		var user models.AuthUser
		if err := db.DB.First(&user, userID).Error; err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
			c.Abort()
			return
		}

		if user.IsSuperuser {
			c.Next()
			return
		}

		// Load user's permissions from roles
		var profile models.Profile
		if err := db.DB.Preload("Roles.Permissions").First(&profile, userID).Error; err != nil {
			c.JSON(http.StatusForbidden, gin.H{"error": "unable to load permissions"})
			c.Abort()
			return
		}

		// Build set of user's permissions
		userPerms := make(map[string]bool)
		for _, role := range profile.Roles {
			for _, perm := range role.Permissions {
				userPerms[perm.Code] = true
			}
		}

		// Check if user has ALL required permissions
		for _, requiredPerm := range permissions {
			if !userPerms[requiredPerm] {
				c.JSON(http.StatusForbidden, gin.H{"error": "permission denied: " + requiredPerm})
				c.Abort()
				return
			}
		}

		c.Next()
	}
}

// CurrentUser is a helper for handlers that need user, profile, and ok
type CurrentUser struct {
	UserID  uint
	Profile *models.Profile
	User    *models.AuthUser
}

// GetCurrentUser retrieves all current user info from context
// Returns nil if user not authenticated
func GetCurrentUser(c *gin.Context) *CurrentUser {
	userID, exists := GetUser(c)
	if !exists {
		return nil
	}

	result := &CurrentUser{UserID: userID}

	// Try to get profile from context, or load it
	profile := GetUserProfile(c)
	if profile == nil {
		var p models.Profile
		if err := db.DB.Where("user_id = ?", userID).First(&p).Error; err == nil {
			result.Profile = &p
			c.Set(UserProfileContextKey, &p)
		}
	} else {
		result.Profile = profile
	}

	// Try to get auth user from context, or load it
	user := GetAuthUser(c)
	if user == nil {
		var u models.AuthUser
		if err := db.DB.First(&u, userID).Error; err == nil {
			result.User = &u
			c.Set(AuthUserContextKey, &u)
		}
	} else {
		result.User = user
	}

	return result
}
