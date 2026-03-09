package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/gin-gonic/gin"
)

// AuditMiddleware creates middleware that logs admin actions
func Audit() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Only audit admin endpoints
		if !strings.HasPrefix(c.Request.URL.Path, "/api/v2/admin") {
			c.Next()
			return
		}

		// Get user info from context
		userID, userIDExists := c.Get("user_id")
		username, _ := c.Get("username")

		// Read request body for details (without consuming it)
		var requestBody map[string]interface{}
		if c.Request.Body != nil && c.Request.Body != http.NoBody {
			bodyBytes, err := io.ReadAll(c.Request.Body)
			if err == nil {
				json.Unmarshal(bodyBytes, &requestBody)
				// Restore body for downstream handlers
				c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			}
		}

		// Process request
		c.Next()

		// Determine action based on HTTP method
		action := "VIEW"
		switch c.Request.Method {
		case "POST":
			action = "CREATE"
		case "PUT", "PATCH":
			action = "UPDATE"
		case "DELETE":
			action = "DELETE"
		}

		// Determine status
		status := "success"
		if c.Writer.Status() >= 400 {
			status = "failure"
		}

		// Extract resource info from path
		resource, resourceID := extractResourceInfo(c.Request.URL.Path)

		// Create audit log entry
		details, _ := json.Marshal(gin.H{
			"method":       c.Request.Method,
			"path":         c.Request.URL.Path,
			"query":        c.Request.URL.RawQuery,
			"request_body": requestBody,
			"status_code":  c.Writer.Status(),
		})

		auditEntry := models.AuditLog{
			UserID:    0,
			Username:  "anonymous",
			Action:    action,
			Resource:  resource,
			ResourceID: resourceID,
			IPAddress: c.ClientIP(),
			UserAgent: c.Request.UserAgent(),
			Details:   string(details),
			Status:    status,
			CreatedAt: time.Now(),
		}

		// Add user info if authenticated
		if userIDExists {
			if id, ok := userID.(uint); ok {
				auditEntry.UserID = id
			}
			if uname, ok := username.(string); ok {
				auditEntry.Username = uname
			}
		}

		// Save audit log asynchronously (don't block response)
		go func() {
			if err := db.DB.Create(&auditEntry).Error; err != nil {
				// Log error but don't fail the request
				// In production, consider sending to error monitoring
			}
		}()
	}
}

// extractResourceInfo extracts resource name and ID from the URL path
func extractResourceInfo(path string) (string, string) {
	// Remove /api/v2/admin prefix
	path = strings.TrimPrefix(path, "/api/v2/admin")
	path = strings.TrimPrefix(path, "/")

	parts := strings.Split(path, "/")
	if len(parts) == 0 {
		return "unknown", ""
	}

	// First part is usually the resource
	resource := parts[0]

	// Look for ID patterns (:id, :key, :code, or actual values)
	var resourceID string
	for i, part := range parts {
		if i == 0 {
			continue // Skip resource name
		}
		// Skip if it's a parameter name or action
		if strings.HasPrefix(part, ":") {
			continue
		}
		if part == "clone" || part == "lock" || part == "disqualify" {
			continue
		}
		// If previous part was a parameter name, this is the value
		if i > 0 && strings.HasPrefix(parts[i-1], ":") {
			resourceID = part
			break
		}
		// Otherwise, first numeric or alphanumeric ID
		if resourceID == "" && len(part) > 0 {
			resourceID = part
		}
	}

	// Capitalize resource name
	resource = strings.Title(strings.ToLower(resource))

	return resource, resourceID
}
