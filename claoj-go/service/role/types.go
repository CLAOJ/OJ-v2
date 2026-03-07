// Package role provides role and permission management services.
package role

import "time"

// Role represents a role with associated permissions.
type Role struct {
	ID          uint
	Name        string
	DisplayName string
	Description string
	Color       string
	IsDefault   bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
	PermissionIDs []uint
}

// Permission represents a single permission.
type Permission struct {
	ID          uint
	Code        string
	Name        string
	Description string
	Category    string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// RoleDetail holds detailed role information with permissions.
type RoleDetail struct {
	Role        Role
	Permissions []Permission
}

// ListRolesRequest holds parameters for listing roles.
type ListRolesRequest struct {
	Page     int
	PageSize int
}

// ListRolesResponse holds the response for listing roles.
type ListRolesResponse struct {
	Roles      []Role
	Total      int64
	Page       int
	PageSize   int
}

// GetRoleRequest holds parameters for getting a role.
type GetRoleRequest struct {
	RoleID uint
}

// CreateRoleRequest holds parameters for creating a role.
type CreateRoleRequest struct {
	Name        string
	DisplayName string
	Description string
	Color       string
	IsDefault   bool
	PermissionIDs []uint
}

// UpdateRoleRequest holds parameters for updating a role.
type UpdateRoleRequest struct {
	RoleID      uint
	DisplayName *string
	Description *string
	Color       *string
	IsDefault   *bool
	PermissionIDs []uint // If provided, replaces all existing permissions
}

// DeleteRoleRequest holds parameters for deleting a role.
type DeleteRoleRequest struct {
	RoleID uint
}

// AssignRoleRequest holds parameters for assigning a role to a profile.
type AssignRoleRequest struct {
	ProfileID uint
	RoleID    uint
}

// RemoveRoleRequest holds parameters for removing a role from a profile.
type RemoveRoleRequest struct {
	ProfileID uint
	RoleID    uint
}

// ListPermissionsRequest holds parameters for listing permissions.
type ListPermissionsRequest struct {
	Category string // Optional filter by category
}

// ListPermissionsResponse holds the response for listing permissions.
type ListPermissionsResponse struct {
	Permissions []Permission
	Total       int64
}
