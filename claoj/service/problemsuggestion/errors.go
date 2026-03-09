// Package problemsuggestion provides problem suggestion management services.
package problemsuggestion

import "errors"

// Service errors
var (
	ErrSuggestionNotFound    = errors.New("suggestion: suggestion not found")
	ErrSuggestionNotPending  = errors.New("suggestion: suggestion is not pending")
	ErrSuggestionAlreadyApproved = errors.New("suggestion: suggestion already approved")
	ErrSuggestionAlreadyRejected = errors.New("suggestion: suggestion already rejected")
	ErrProblemCodeExists   = errors.New("suggestion: problem code already exists")
	ErrInvalidSuggestionID = errors.New("suggestion: invalid suggestion ID")
	ErrEmptyReason         = errors.New("suggestion: rejection reason cannot be empty")
)
