// Package problemgroup provides problem group management services.
package problemgroup

import "errors"

// Service errors
var (
	ErrGroupNotFound      = errors.New("group: group not found")
	ErrInvalidGroupID     = errors.New("group: invalid group ID")
	ErrEmptyGroupName     = errors.New("group: group name cannot be empty")
	ErrEmptyGroupFullName = errors.New("group: group full name cannot be empty")
	ErrGroupNameExists    = errors.New("group: group name already exists")
)
