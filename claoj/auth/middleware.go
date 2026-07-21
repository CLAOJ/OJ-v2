package auth

import (
	"net/http"
	"strings"

	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/gin-gonic/gin"
)

// isJWTFormat checks if a token string is in JWT format (3 base64 parts separated by dots)
func isJWTFormat(token string) bool {
	parts := strings.Split(token, ".")
	return len(parts) == 3 && len(parts[0]) > 0 && len(parts[1]) > 0 && len(parts[2]) > 0
}

// RequiredMiddleware ensures a valid access token is present in httpOnly cookie or Authorization header
func RequiredMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		var tokenString string
		var err error

		// Try to read access token from httpOnly cookie first
		tokenString, err = c.Cookie("access_token")
		if err != nil {
			// Cookie not found, try Authorization header (for API token auth)
			authHeader := c.GetHeader("Authorization")
			if authHeader != "" {
				// Support "Bearer <token>" format
				if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
					tokenString = authHeader[7:]
				} else {
					// Assume it's an API token (64 hex chars)
					tokenString = authHeader
				}
			}
		}

		if tokenString == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "access token required"})
			return
		}

		// Check if this is a JWT access token (JWTs have 3 parts separated by dots)
		if isJWTFormat(tokenString) {
			claims, err := VerifyToken(tokenString, "access")
			if err != nil {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
				return
			}

			// Store user properties in context for downstream handlers
			c.Set("user_id", claims.UserID)
			c.Set("username", claims.Username)
			c.Set("is_admin", claims.IsAdmin)
			c.Next()
			return
		}

		// Otherwise, treat as API token (64 hex character token)
		if len(tokenString) == 64 && isHexString(tokenString) {
			// Look up user by API token
			var profile models.Profile
			if err := db.DB.Where("api_token = ?", tokenString).First(&profile).Error; err != nil {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid API token"})
				return
			}

			// Get auth user info
			var authUser models.AuthUser
			if err := db.DB.First(&authUser, profile.UserID).Error; err != nil {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
				return
			}

			// Store user properties in context
			c.Set("user_id", authUser.ID)
			c.Set("username", authUser.Username)
			c.Set("is_admin", authUser.IsSuperuser)
			c.Next()
			return
		}

		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token format"})
	}
}

// isHexString checks if a string contains only hexadecimal characters
func isHexString(s string) bool {
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

// OptionalMiddleware parses the token from httpOnly cookie if present, but doesn't abort if missing or invalid.
// Useful for endpoints that return different data depending on auth state (like problem list).
func OptionalMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Read access token from httpOnly cookie
		tokenString, err := c.Cookie("access_token")
		if err == nil && tokenString != "" {
			// Try JWT first
			if isJWTFormat(tokenString) {
				claims, err := VerifyToken(tokenString, "access")
				if err == nil {
					c.Set("user_id", claims.UserID)
					c.Set("username", claims.Username)
					c.Set("is_admin", claims.IsAdmin)
				}
			} else if len(tokenString) == 64 && isHexString(tokenString) {
				// Try API token
				var profile models.Profile
				if err := db.DB.Where("api_token = ?", tokenString).First(&profile).Error; err == nil {
					var authUser models.AuthUser
					if err := db.DB.First(&authUser, profile.UserID).Error; err == nil {
						c.Set("user_id", authUser.ID)
						c.Set("username", authUser.Username)
						c.Set("is_admin", authUser.IsSuperuser)
					}
				}
			}
		}
		c.Next()
	}
}

// AdminRequiredMiddleware ensures the user is authenticated AND may use the admin surface (is_staff OR is_superuser).
func AdminRequiredMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// First check if user is authenticated
		uid, exists := c.Get("user_id")
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authorization required"})
			return
		}
		userID := uid.(uint)

		var user models.AuthUser
		if err := db.DB.Select("id", "is_staff", "is_superuser").First(&user, userID).Error; err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
			return
		}

		// v1 parity: is_staff opens the admin surface; a superuser bypasses the
		// staff requirement entirely (Django ModelBackend semantics).
		if !user.IsStaff && !user.IsSuperuser {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "admin access required"})
			return
		}

		c.Set("admin_user_id", user.ID)
		c.Next()
	}
}

