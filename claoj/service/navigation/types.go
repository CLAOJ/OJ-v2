// Package navigation provides navigation bar management services.
package navigation

// NavigationEntry represents a navigation bar entry.
type NavigationEntry struct {
	ID          uint
	Key         string
	Label       string
	Path        string
	Regex       string
	ParentID    *uint
	ParentLabel string
	Order       int
	Lft         int
	Rght        int
	TreeID      int
	Level       int
}

// PublicNavigationItem is one node of the public navbar tree.
type PublicNavigationItem struct {
	Key      string
	Label    string
	Path     string
	Regex    string
	Children []PublicNavigationItem
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
	Regex    string
	ParentID *uint
	Order    int
}

// UpdateNavigationRequest holds parameters for updating a navigation entry.
// SetParent distinguishes "leave the parent alone" from "move to root" (nil).
type UpdateNavigationRequest struct {
	NavID     uint
	Label     *string
	Path      *string
	Regex     *string
	ParentID  *uint
	SetParent bool
	Order     *int
}

// DeleteNavigationRequest holds parameters for deleting a navigation entry.
type DeleteNavigationRequest struct {
	NavID uint
}
