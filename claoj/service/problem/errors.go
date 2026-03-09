// Package problem provides problem management services.
package problem

import "errors"

// Service errors
var (
	ErrProblemNotFound       = errors.New("problem: problem not found")
	ErrProblemCodeExists     = errors.New("problem: problem code already exists")
	ErrClarificationNotFound = errors.New("problem: clarification not found")
)
