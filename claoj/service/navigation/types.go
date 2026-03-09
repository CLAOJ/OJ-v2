// Package navigation provides navigation bar management services.
package navigation

// NavigationEntry represents a navigation bar entry.
type NavigationEntry struct {
	ID       uint
	Key      string
	Label    string
	Path     string
	ParentID *uint
	Order    int
	Lft      int
	Rght     int
	TreeID   int
	Level    int
}

// ListNavigationRequest holds parameters for listing navigation entries.
type ListNavigationRequest struct {
	Page     int
	PageSize int
	TreeID   *int // Optional filter by tree ID
}

// ListNavigationResponse holds the response for listing navigation entries.
type ListNavigationResponse struct {
	Entries  []NavigationEntry
	Total    int64
	Page     int
	PageSize int
}

// GetNavigationRequest holds parameters for getting a navigation entry.
type GetNavigationRequest struct {
	NavID uint
}

// CreateNavigationRequest holds parameters for creating a navigation entry.
type CreateNavigationRequest struct {
	Key      string
	Label    string
	Path     string
	ParentID *uint
	Order    int
}

// UpdateNavigationRequest holds parameters for updating a navigation entry.
type UpdateNavigationRequest struct {
	NavID    uint
	Label    *string
	Path     *string
	ParentID *uint
	Order    *int
}

// DeleteNavigationRequest holds parameters for deleting a navigation entry.
type DeleteNavigationRequest struct {
	NavID uint
}
