package auth

import (
	"net/http"
	"strings"

	"github.com/CLAOJ/claoj-go/db"
	"github.com/CLAOJ/claoj-go/models"
	"github.com/gin-gonic/gin"
)

// RequiredMiddleware ensures a valid access token is present
func RequiredMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authorization header required"})
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authorization header must be Bearer token"})
			return
		}

		tokenString := parts[1]
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
	}
}

// OptionalMiddleware parses the token if present, but doesn't abort if missing or invalid.
// Useful for endpoints that return different data depending on auth state (like problem list).
func OptionalMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
				claims, err := VerifyToken(parts[1], "access")
				if err == nil {
					c.Set("user_id", claims.UserID)
					c.Set("username", claims.Username)
					c.Set("is_admin", claims.IsAdmin)
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
