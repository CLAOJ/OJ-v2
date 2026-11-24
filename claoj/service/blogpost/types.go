// Package blogpost provides blog post management services.
package blogpost

import "time"

// BlogPost represents a blog post with associated data.
type BlogPost struct {
	ID             uint
	Title          string
	Slug           string
	AuthorIDs      []uint
	PublishOn      time.Time
	Content        string
	Summary        string
	Visible        bool
	Sticky         bool
	Score          int
	GlobalPost     bool
	OgImage        string
	OrganizationID *uint

	// Relations, populated when the query preloads Authors.User/Organization.
	AuthorNames      []string
	OrganizationName *string
}

// ListBlogPostsRequest holds parameters for listing blog posts.
type ListBlogPostsRequest struct {
	Page        int
	PageSize    int
	VisibleOnly bool   // If true, only return visible posts
	Search      string // Matches title OR slug (LIKE)
	Visible     *bool  // If set, filter by visible; overrides nothing else
}

// ListBlogPostsResponse holds the response for listing blog posts.
type ListBlogPostsResponse struct {
	BlogPosts  []BlogPost
	Total      int64
	Page       int
	PageSize   int
}

// GetBlogPostRequest holds parameters for getting a blog post.
type GetBlogPostRequest struct {
	BlogPostID uint
}

// CreateBlogPostRequest holds parameters for creating a blog post.
type CreateBlogPostRequest struct {
	Title        string
	Slug         string
	Content      string
	Summary      string
	AuthorIDs    []uint
	PublishOn    time.Time
	Visible      bool
	Sticky       bool
	GlobalPost   bool
	OgImage      string
	OrganizationID *uint
}

// UpdateBlogPostRequest holds parameters for updating a blog post.
type UpdateBlogPostRequest struct {
	BlogPostID     uint
	Title          *string
	Slug           *string
	Content        *string
	Summary        *string
	AuthorIDs      []uint // If provided, replaces all authors
	PublishOn      *time.Time
	Visible        *bool
	Sticky         *bool
	GlobalPost     *bool
	OgImage        *string
	OrganizationID *uint
	// ClearOrganization sets organization_id to NULL; it wins over a nil
	// OrganizationID, which means "leave unchanged".
	ClearOrganization bool
}

// DeleteBlogPostRequest holds parameters for deleting a blog post.
type DeleteBlogPostRequest struct {
	BlogPostID uint
}

// BlogPostDetail holds detailed blog post information.
type BlogPostDetail struct {
	BlogPost         BlogPost
	AuthorNames      []string
	OrganizationName *string
}
