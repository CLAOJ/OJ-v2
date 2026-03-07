// Package request provides shared request types for API handlers.
package request

// Pagination holds pagination parameters.
type Pagination struct {
	Page     int `form:"page" binding:"omitempty,min=1"`
	PageSize int `form:"page_size" binding:"omitempty,min=1,max=100"`
}

// DefaultPagination returns default pagination values.
func DefaultPagination() Pagination {
	return Pagination{
		Page:     1,
		PageSize: 20,
	}
}
