// Package problem provides problem management services.
package problem

import "errors"

// Service errors
var (
	ErrProblemNotFound       = errors.New("problem: problem not found")
	ErrProblemCodeExists     = errors.New("problem: problem code already exists")
	ErrInvalidGroup          = errors.New("problem: group does not exist")
	ErrInvalidLicense        = errors.New("problem: license does not exist")
	ErrInvalidSuggester      = errors.New("problem: suggester does not exist")
	ErrFieldTooLong          = errors.New("problem: field value too long")
	ErrClarificationNotFound = errors.New("problem: clarification not found")
	ErrInvalidVisibilityMode = errors.New("problem: invalid visibility mode")
)
