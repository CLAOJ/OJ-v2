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

// apiListWithTotal is a helper for consistent paginated list responses
func apiListWithTotal(data interface{}, total int64) gin.H {
	return gin.H{"data": data, "total": total}
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

// ============================================================
// STANDARDIZED ERROR HANDLING HELPERS
// ============================================================

// Common error variables for consistent error handling
var (
	ErrNotFound     = "resource not found"
	ErrUnauthorized = "unauthorized access"
	ErrForbidden    = "access denied"
	ErrBadRequest   = "invalid request"
	ErrInternal     = "internal server error"
)

// ErrorResponse returns a standardized error response
func ErrorResponse(msg string) gin.H {
	return gin.H{"error": msg}
}

// ErrorResponseWithStatus returns a standardized error response with HTTP status code
func ErrorResponseWithStatus(c *gin.Context, status int, msg string) {
	c.JSON(status, ErrorResponse(msg))
}

// BadRequestError returns a 400 Bad Request error response
func BadRequestError(c *gin.Context, msg string) {
	if msg == "" {
		msg = ErrBadRequest
	}
	c.JSON(http.StatusBadRequest, ErrorResponse(msg))
}

// UnauthorizedError returns a 401 Unauthorized error response
func UnauthorizedError(c *gin.Context, msg string) {
	if msg == "" {
		msg = ErrUnauthorized
	}
	c.JSON(http.StatusUnauthorized, ErrorResponse(msg))
}

// ForbiddenError returns a 403 Forbidden error response
func ForbiddenError(c *gin.Context, msg string) {
	if msg == "" {
		msg = ErrForbidden
	}
	c.JSON(http.StatusForbidden, ErrorResponse(msg))
}

// NotFoundError returns a 404 Not Found error response
func NotFoundError(c *gin.Context, msg string) {
	if msg == "" {
		msg = ErrNotFound
	}
	c.JSON(http.StatusNotFound, ErrorResponse(msg))
}

// InternalError returns a 500 Internal Server Error response
// Logs the actual error but returns a generic message to avoid information disclosure
func InternalError(c *gin.Context, err error) {
	log.Printf("Internal error: %v", err)
	c.JSON(http.StatusInternalServerError, ErrorResponse(ErrInternal))
}

// InternalErrorWithMessage returns a 500 Internal Server Error response with custom message
func InternalErrorWithMessage(c *gin.Context, msg string) {
	log.Printf("Internal error: %s", msg)
	c.JSON(http.StatusInternalServerError, ErrorResponse(msg))
}

// ConflictError returns a 409 Conflict error response
func ConflictError(c *gin.Context, msg string) {
	c.JSON(http.StatusConflict, ErrorResponse(msg))
}

// ValidationError returns a 422 Unprocessable Entity error response for validation errors
func ValidationError(c *gin.Context, msg string) {
	c.JSON(http.StatusUnprocessableEntity, ErrorResponse(msg))
}

// SuccessResponse returns a standardized success response
func SuccessResponse(data interface{}) gin.H {
	return gin.H{"success": true, "data": data}
}

// SuccessResponseWithMessage returns a standardized success response with message
func SuccessResponseWithMessage(data interface{}, msg string) gin.H {
	return gin.H{"success": true, "data": data, "message": msg}
}

// PaginationResponse returns a standardized paginated response
func PaginationResponse(data interface{}, total int64) gin.H {
	return gin.H{"data": data, "total": total}
}

// PaginationResponseWithPage returns a standardized paginated response with page info
func PaginationResponseWithPage(data interface{}, total int64, page, pageSize int) gin.H {
	return gin.H{
		"data":      data,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
		"has_more":  page*pageSize < int(total),
	}
}
