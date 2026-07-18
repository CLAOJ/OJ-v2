// Package group provides Django-group (role) management services.
package group

import "errors"

// Service errors
var (
	ErrGroupNotFound   = errors.New("group: group not found")
	ErrGroupNameExists = errors.New("group: group name already exists")
	ErrEmptyGroupName  = errors.New("group: group name cannot be empty")
	ErrUserNotFound    = errors.New("group: user not found")
)
