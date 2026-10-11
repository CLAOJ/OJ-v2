// Package problem provides problem management services.
package problem

import "errors"

// Service errors
var (
	ErrProblemNotFound       = errors.New("problem: problem not found")
	ErrProblemCodeExists     = errors.New("problem: problem code already exists")
	ErrInvalidGroup          = errors.New("problem: group does not exist")
	ErrClarificationNotFound = errors.New("problem: clarification not found")
)
