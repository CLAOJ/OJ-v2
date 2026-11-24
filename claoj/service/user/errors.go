// Package user provides user management services.
package user

import "errors"

// Service errors
var (
	ErrInvalidUserID = errors.New("user: invalid user ID")
	ErrInvalidReason = errors.New("user: invalid reason")
	ErrUserNotFound  = errors.New("user: user not found")
	ErrAlreadyBanned = errors.New("user: user is already banned")
	ErrNotBanned     = errors.New("user: user is not banned")
	ErrUsernameTaken = errors.New("user: username already taken")
	ErrEmailTaken    = errors.New("user: email already taken")
)
