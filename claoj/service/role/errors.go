// Package role provides role and permission management services.
package role

import "errors"

// Service errors
var (
	ErrRoleNotFound          = errors.New("role: role not found")
	ErrPermissionNotFound    = errors.New("role: permission not found")
	ErrInvalidRoleID         = errors.New("role: invalid role ID")
	ErrInvalidPermissionID   = errors.New("role: invalid permission ID")
	ErrCannotDeleteDefault   = errors.New("role: cannot delete default role")
	ErrEmptyRoleName         = errors.New("role: role name cannot be empty")
	ErrEmptyDisplayName      = errors.New("role: display name cannot be empty")
	ErrRoleNameExists        = errors.New("role: role name already exists")
)
