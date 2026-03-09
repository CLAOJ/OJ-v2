// Package comment provides comment management services.
package comment

import "time"

// Comment represents a comment with associated data.
type Comment struct {
	ID       uint
	AuthorID uint
	Time     time.Time
	Page     string
	Score    int
	Body     string
	Hidden   bool
	ParentID *uint
}

// CommentRevision represents a revision of a comment.
type CommentRevision struct {
	ID        uint
	CommentID uint
	EditorID  uint
	Time      time.Time
	Body      string
	Reason    string
}

// UpdateCommentRequest holds the parameters for updating a comment.
type UpdateCommentRequest struct {
	CommentID uint
	Body      *string
	Hidden    *bool
	Reason    string // Reason for editing (for revision tracking)
	EditorID  uint   // ID of the user making the change
}

// DeleteCommentRequest holds the parameters for deleting a comment.
type DeleteCommentRequest struct {
	CommentID uint
}

// GetCommentRequest holds the parameters for getting a comment.
type GetCommentRequest struct {
	CommentID uint
}

// ListCommentsRequest holds the parameters for listing comments.
type ListCommentsRequest struct {
	Page     int
	PageSize int
	PageFilter string // Filter by page (e.g., problem page)
}

// ListCommentsResponse holds the response for listing comments.
type ListCommentsResponse struct {
	Comments []Comment
	Total    int64
	Page     int
	PageSize int
}

// CommentDetailResponse holds the full comment detail response.
type CommentDetailResponse struct {
	Comment Comment
	Revisions []CommentRevision
}
