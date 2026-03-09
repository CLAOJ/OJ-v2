// Package blogpost provides blog post management services.
package blogpost

import "errors"

// Service errors
var (
	ErrBlogPostNotFound   = errors.New("blogpost: blog post not found")
	ErrInvalidBlogPostID  = errors.New("blogpost: invalid blog post ID")
	ErrEmptyTitle         = errors.New("blogpost: title cannot be empty")
	ErrEmptySlug          = errors.New("blogpost: slug cannot be empty")
	ErrEmptyContent       = errors.New("blogpost: content cannot be empty")
	ErrSlugExists         = errors.New("blogpost: slug already exists")
)
