// Package auditlog provides audit log management services.
package auditlog

import "errors"

// Service errors
var (
	ErrLogNotFound    = errors.New("auditlog: log entry not found")
	ErrInvalidLogID   = errors.New("auditlog: invalid log ID")
)
