// Package submission provides submission management services.
package submission

import "errors"

// Service errors
var (
	ErrSubmissionNotFound       = errors.New("submission: submission not found")
	ErrSubmissionLocked         = errors.New("submission: submission is locked and cannot be rejudged")
	ErrSubmissionNotProcessing  = errors.New("submission: submission is not being processed")
	ErrSubmissionNotCompleted   = errors.New("submission: submission is not completed")
	ErrBridgeServerNotAvailable = errors.New("submission: bridge server not available")
	ErrProblemNotFound          = errors.New("submission: problem not found")
)
