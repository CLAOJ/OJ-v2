// Package navigation provides navigation bar management services.
package navigation

import "errors"

// Service errors
var (
	ErrNavNotFound      = errors.New("navigation: entry not found")
	ErrInvalidNavID     = errors.New("navigation: invalid navigation ID")
	ErrEmptyKey         = errors.New("navigation: key cannot be empty")
	ErrEmptyLabel       = errors.New("navigation: label cannot be empty")
	ErrEmptyPath        = errors.New("navigation: path cannot be empty")
	ErrKeyExists        = errors.New("navigation: key already exists")
)
