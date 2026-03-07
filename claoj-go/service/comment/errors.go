// Package comment provides comment management services.
package comment

import "errors"

// Service errors
var (
	ErrCommentNotFound    = errors.New("comment: comment not found")
	ErrUnauthorized       = errors.New("comment: not authorized to edit comment")
	ErrInvalidCommentID   = errors.New("comment: invalid comment ID")
	ErrEmptyBody          = errors.New("comment: body cannot be empty")
	ErrRevisionCreateFail = errors.New("comment: failed to create revision record")
)
