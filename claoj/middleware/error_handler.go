package middleware

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// ErrorResponse represents a standardized error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
	Code    string `json:"code,omitempty"`
}

// Common error codes
const (
	ErrCodeNotFound           = "NOT_FOUND"
	ErrCodeUnauthorized       = "UNAUTHORIZED"
	ErrCodeForbidden          = "FORBIDDEN"
	ErrCodeBadRequest         = "BAD_REQUEST"
	ErrCodeValidationError    = "VALIDATION_ERROR"
	ErrCodeInternalError      = "INTERNAL_ERROR"
	ErrCodeRateLimitExceeded  = "RATE_LIMIT_EXCEEDED"
	ErrCodeServiceUnavailable = "SERVICE_UNAVAILABLE"
)

// ErrorHandlerMiddleware catches errors and returns standardized responses
func ErrorHandlerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// Check if there were any errors
		if len(c.Errors) > 0 {
			err := c.Errors.Last()

			// Log the error for debugging
			log.Printf("[ErrorHandler] %s %s: %v", c.Request.Method, c.Request.URL.Path, err.Err)

			// Return standardized error response
			statusCode := c.Writer.Status()
			if statusCode == 200 {
				statusCode = http.StatusInternalServerError
			}

			response := ErrorResponse{
				Error: http.StatusText(statusCode),
			}

			// Add error message based on environment (don't expose internal details in production)
			if gin.Mode() == gin.DebugMode {
				response.Message = err.Err.Error()
			} else {
				response.Message = getPublicErrorMessage(statusCode)
			}

			c.JSON(statusCode, response)
		}
	}
}

// RecoveryMiddleware recovers from panics and returns a 500 error
func RecoveryMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("[PANIC] %s %s: %v", c.Request.Method, c.Request.URL.Path, err)

				response := ErrorResponse{
					Error:   "Internal Server Error",
					Code:    ErrCodeInternalError,
					Message: "An unexpected error occurred. Please try again later.",
				}

				c.AbortWithStatusJSON(http.StatusInternalServerError, response)
			}
		}()

		c.Next()
	}
}

// getPublicErrorMessage returns a user-friendly error message based on status code
func getPublicErrorMessage(statusCode int) string {
	switch statusCode {
	case http.StatusBadRequest:
		return "Invalid request. Please check your input and try again."
	case http.StatusUnauthorized:
		return "Please log in to access this resource."
	case http.StatusForbidden:
		return "You don't have permission to access this resource."
	case http.StatusNotFound:
		return "The requested resource was not found."
	case http.StatusMethodNotAllowed:
		return "This request method is not allowed for this resource."
	case http.StatusConflict:
		return "The request conflicts with the current state of the resource."
	case http.StatusUnprocessableEntity:
		return "Validation failed. Please check your input."
	case http.StatusTooManyRequests:
		return "Too many requests. Please try again later."
	case http.StatusInternalServerError:
		return "An internal error occurred. Please try again later."
	case http.StatusServiceUnavailable:
		return "Service temporarily unavailable. Please try again later."
	default:
		return "An error occurred. Please try again."
	}
}

// AbortWithErrorJSON aborts the request with a standardized error response
func AbortWithErrorJSON(c *gin.Context, status int, err string, message ...string) {
	response := ErrorResponse{
		Error: err,
	}

	if len(message) > 0 {
		response.Message = message[0]
	} else {
		response.Message = getPublicErrorMessage(status)
	}

	c.AbortWithStatusJSON(status, response)
}

// JSONError returns a standardized JSON error response without aborting
func JSONError(c *gin.Context, status int, err string, message ...string) {
	response := ErrorResponse{
		Error: err,
	}

	if len(message) > 0 {
		response.Message = message[0]
	} else {
		response.Message = getPublicErrorMessage(status)
	}

	c.JSON(status, response)
}

// DatabaseError handles database errors and returns appropriate responses
func DatabaseError(c *gin.Context, err error, notFoundMsg ...string) bool {
	if err == nil {
		return false
	}

	log.Printf("[DatabaseError] %s %s: %v", c.Request.Method, c.Request.URL.Path, err)

	msg := "Resource not found"
	if len(notFoundMsg) > 0 {
		msg = notFoundMsg[0]
	}

	c.JSON(http.StatusInternalServerError, ErrorResponse{
		Error:   "Database Error",
		Code:    ErrCodeInternalError,
		Message: msg,
	})
	return true
}
