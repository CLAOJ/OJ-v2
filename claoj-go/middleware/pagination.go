package middleware

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const (
	// DefaultPage is the default page number
	DefaultPage = 1
	// DefaultPageSize is the default page size
	DefaultPageSize = 100
	// MaxPageSize is the maximum allowed page size
	MaxPageSize = 1000
	// PageQueryKey is the query parameter key for page number
	PageQueryKey = "page"
	// PageSizeQueryKey is the query parameter key for page size
	PageSizeQueryKey = "page_size"
	// PageContextKey is the context key for page number
	PageContextKey = "pagination_page"
	// PageSizeContextKey is the context key for page size
	PageSizeContextKey = "pagination_page_size"
	// OffsetContextKey is the context key for calculated offset
	OffsetContextKey = "pagination_offset"
)

// PaginationConfig holds pagination configuration
type PaginationConfig struct {
	DefaultPage     int
	DefaultPageSize int
	MaxPageSize     int
	PageKey         string
	PageSizeKey     string
}

// DefaultPaginationConfig returns the default pagination configuration
func DefaultPaginationConfig() PaginationConfig {
	return PaginationConfig{
		DefaultPage:     DefaultPage,
		DefaultPageSize: DefaultPageSize,
		MaxPageSize:     MaxPageSize,
		PageKey:         PageQueryKey,
		PageSizeKey:     PageSizeQueryKey,
	}
}

// Pagination returns a middleware that parses pagination parameters from query string
// and stores them in the context.
//
// Query parameters:
//   - page: page number (default: 1, minimum: 1)
//   - page_size: page size (default: 100, min: 1, max: 1000)
//
// Context values stored:
//   - pagination_page: int - the page number
//   - pagination_page_size: int - the page size
//   - pagination_offset: int - calculated offset (page-1)*page_size
func Pagination() gin.HandlerFunc {
	return PaginationWithConfig(DefaultPaginationConfig())
}

// PaginationWithConfig returns a pagination middleware with custom configuration
func PaginationWithConfig(config PaginationConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		page := parseIntParam(c, config.PageKey, config.DefaultPage)
		pageSize := parseIntParam(c, config.PageSizeKey, config.DefaultPageSize)

		// Validate page number
		if page < 1 {
			page = config.DefaultPage
		}

		// Validate page size
		if pageSize < 1 {
			pageSize = config.DefaultPageSize
		} else if pageSize > config.MaxPageSize {
			pageSize = config.MaxPageSize
		}

		offset := (page - 1) * pageSize

		// Store in context
		c.Set(PageContextKey, page)
		c.Set(PageSizeContextKey, pageSize)
		c.Set(OffsetContextKey, offset)

		c.Next()
	}
}

// parseIntParam parses an integer from query parameters with a default value
func parseIntParam(c *gin.Context, key string, defaultValue int) int {
	valueStr := c.DefaultQuery(key, strconv.Itoa(defaultValue))
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}
	return value
}

// GetPage retrieves the page number from context
func GetPage(c *gin.Context) int {
	if v, exists := c.Get(PageContextKey); exists {
		if page, ok := v.(int); ok {
			return page
		}
	}
	return DefaultPage
}

// GetPageSize retrieves the page size from context
func GetPageSize(c *gin.Context) int {
	if v, exists := c.Get(PageSizeContextKey); exists {
		if pageSize, ok := v.(int); ok {
			return pageSize
		}
	}
	return DefaultPageSize
}

// GetOffset retrieves the calculated offset from context
func GetOffset(c *gin.Context) int {
	if v, exists := c.Get(OffsetContextKey); exists {
		if offset, ok := v.(int); ok {
			return offset
		}
	}
	return 0
}

// PaginationParams holds parsed pagination parameters
type PaginationParams struct {
	Page     int
	PageSize int
	Offset   int
}

// GetPaginationParams retrieves all pagination parameters from context
func GetPaginationParams(c *gin.Context) PaginationParams {
	return PaginationParams{
		Page:     GetPage(c),
		PageSize: GetPageSize(c),
		Offset:   GetOffset(c),
	}
}

// ApplyPagination applies pagination to a GORM query
// Usage: db.Model(&User{}).Scopes(middleware.ApplyPagination).Find(&users)
// Note: For proper pagination, use Paginate helper function instead
func ApplyPagination(db *gorm.DB) *gorm.DB {
	// When used as a GORM scope, gin context is not available
	// Use default values - this function is mainly for compatibility
	// Recommended: use Paginate(db, c) helper instead
	return db.Offset(0).Limit(DefaultPageSize)
}

// Paginate applies pagination to a GORM query using values from gin context
// Usage: db.Model(&User{}).Scopes(middleware.Paginate(c)).Find(&users)
func Paginate(c *gin.Context) func(*gorm.DB) *gorm.DB {
	offset := GetOffset(c)
	pageSize := GetPageSize(c)

	return func(db *gorm.DB) *gorm.DB {
		return db.Offset(offset).Limit(pageSize)
	}
}

// PaginatedResponse represents a paginated API response
type PaginatedResponse struct {
	Data interface{} `json:"data"`
	// Total is optional, include when total count is known
	Total int64 `json:"total,omitempty"`
	// Page is the current page number
	Page int `json:"page,omitempty"`
	// PageSize is the current page size
	PageSize int `json:"page_size,omitempty"`
	// HasMore indicates if there are more pages
	HasMore bool `json:"has_more,omitempty"`
}

// NewPaginatedResponse creates a new paginated response
func NewPaginatedResponse(data interface{}) PaginatedResponse {
	return PaginatedResponse{
		Data:     data,
		Page:     DefaultPage,
		PageSize: DefaultPageSize,
	}
}

// PaginatedResponseWithMeta creates a paginated response with metadata
func PaginatedResponseWithMeta(c *gin.Context, data interface{}, total int64) PaginatedResponse {
	page := GetPage(c)
	pageSize := GetPageSize(c)
	hasMore := total > int64(page*pageSize)

	return PaginatedResponse{
		Data:     data,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
		HasMore:  hasMore,
	}
}

// SendPaginatedResponse sends a paginated JSON response
func SendPaginatedResponse(c *gin.Context, statusCode int, data interface{}, total int64) {
	c.JSON(statusCode, PaginatedResponseWithMeta(c, data, total))
}

// SendPaginatedResponseNoTotal sends a paginated JSON response without total count
func SendPaginatedResponseNoTotal(c *gin.Context, statusCode int, data interface{}) {
	c.JSON(statusCode, NewPaginatedResponse(data))
}

// ValidatePagination validates pagination parameters and returns an error if invalid
func ValidatePagination(c *gin.Context) bool {
	pageStr := c.Query(PageQueryKey)
	pageSizeStr := c.Query(PageSizeQueryKey)

	// Check for non-numeric values
	if pageStr != "" {
		if _, err := strconv.Atoi(pageStr); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid page parameter"})
			return false
		}
	}

	if pageSizeStr != "" {
		if _, err := strconv.Atoi(pageSizeStr); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid page_size parameter"})
			return false
		}
	}

	return true
}
