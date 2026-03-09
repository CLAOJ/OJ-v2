package auth

import (
	"net/http"
	"strings"

	"github.com/CLAOJ/claoj-go/db"
	"github.com/CLAOJ/claoj-go/models"
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

// AdminRequiredMiddleware ensures the user is authenticated AND is an admin (is_staff = true)
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
		if err := db.DB.Select("is_staff").First(&user, userID).Error; err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
			return
		}

		if !user.IsStaff {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "admin access required"})
			return
		}

		c.Set("admin_user_id", user.ID)
		c.Next()
	}
}

// PermissionRequiredMiddleware creates middleware that checks for specific permissions
func PermissionRequiredMiddleware(permissions ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get user from context
		uid, exists := c.Get("user_id")
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authorization required"})
			return
		}
		userID := uid.(uint)

		// Check if superuser (has all permissions)
		var user models.AuthUser
		if err := db.DB.First(&user, userID).Error; err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
			return
		}

		if user.IsSuperuser {
			c.Next()
			return
		}

		// Load user's permissions from roles
		var profile models.Profile
		if err := db.DB.Preload("Roles.Permissions").First(&profile, userID).Error; err != nil {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "unable to load permissions"})
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
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "permission denied: " + requiredPerm})
				return
			}
		}

		c.Next()
	}
}

// RoleRequiredMiddleware creates middleware that checks for specific roles
func RoleRequiredMiddleware(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get user from context
		uid, exists := c.Get("user_id")
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authorization required"})
			return
		}
		userID := uid.(uint)

		// Check if superuser (has all roles)
		var user models.AuthUser
		if err := db.DB.First(&user, userID).Error; err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
			return
		}

		if user.IsSuperuser {
			c.Next()
			return
		}

		// Load user's roles
		var profile models.Profile
		if err := db.DB.Preload("Roles").First(&profile, userID).Error; err != nil {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "unable to load roles"})
			return
		}

		// Build set of user's role names
		userRoles := make(map[string]bool)
		for _, role := range profile.Roles {
			userRoles[role.Name] = true
		}

		// Check if user has ANY of the required roles
		for _, requiredRole := range roles {
			if userRoles[requiredRole] {
				c.Next()
				return
			}
		}

		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "role required: " + strings.Join(roles, " or ")})
	}
}
