package repository

import "errors"

// Common repository errors
var (
	ErrNotFound          = errors.New("repository: resource not found")
	ErrAlreadyExists     = errors.New("repository: resource already exists")
	ErrInvalidID         = errors.New("repository: invalid ID")
	ErrDuplicateUsername = errors.New("repository: duplicate username")
	ErrDuplicateEmail    = errors.New("repository: duplicate email")
	ErrDuplicateCode     = errors.New("repository: duplicate code")
)
