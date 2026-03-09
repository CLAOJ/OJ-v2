// Package organization provides organization management services.
package organization

import "errors"

// Service errors
var (
	ErrOrganizationNotFound = errors.New("organization: organization not found")
	ErrUserNotFound         = errors.New("organization: user not found")
	ErrOrganizationFull     = errors.New("organization: organization is full")
	ErrNotMember            = errors.New("organization: user is not a member")
	ErrNotAdmin             = errors.New("organization: user is not an admin")
)
