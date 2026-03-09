// Package problemtype provides problem type management services.
package problemtype

import "errors"

// Service errors
var (
	ErrTypeNotFound       = errors.New("type: type not found")
	ErrInvalidTypeID      = errors.New("type: invalid type ID")
	ErrEmptyTypeName      = errors.New("type: type name cannot be empty")
	ErrEmptyTypeFullName  = errors.New("type: type full name cannot be empty")
	ErrTypeNameExists     = errors.New("type: type name already exists")
)
