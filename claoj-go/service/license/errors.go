// Package license provides license management services.
package license

import "errors"

// Service errors
var (
	ErrLicenseNotFound  = errors.New("license: license not found")
	ErrInvalidLicenseID = errors.New("license: invalid license ID")
	ErrEmptyKey         = errors.New("license: key cannot be empty")
	ErrEmptyName        = errors.New("license: name cannot be empty")
	ErrEmptyLink        = errors.New("license: link cannot be empty")
	ErrKeyExists        = errors.New("license: key already exists")
)
