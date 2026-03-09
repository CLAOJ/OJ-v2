// Package miscconfig provides miscellaneous configuration management services.
package miscconfig

import "errors"

// Service errors
var (
	ErrConfigNotFound  = errors.New("config: configuration not found")
	ErrInvalidConfigID = errors.New("config: invalid configuration ID")
	ErrEmptyKey        = errors.New("config: key cannot be empty")
	ErrKeyExists       = errors.New("config: key already exists")
)
