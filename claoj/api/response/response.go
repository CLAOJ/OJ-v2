// Package response provides shared response helpers for API handlers.
package response

import "github.com/gin-gonic/gin"

// Error creates an error response.
func Error(msg string) gin.H {
	return gin.H{"error": msg}
}

// Success creates a success response with optional data.
func Success(message string, data ...interface{}) gin.H {
	result := gin.H{
		"success": true,
		"message": message,
	}
	if len(data) > 0 {
		result["data"] = data[0]
	}
	return result
}

// List creates a paginated list response.
func List(data interface{}) gin.H {
	return gin.H{
		"data": data,
	}
}

// ListWithPagination creates a paginated list response with pagination info.
func ListWithPagination(data interface{}, total, page, pageSize int64) gin.H {
	return gin.H{
		"data":      data,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	}
}

// ListWithTotal creates a list response with total count.
func ListWithTotal(data interface{}, total int64) gin.H {
	return gin.H{
		"data":  data,
		"total": total,
	}
}
