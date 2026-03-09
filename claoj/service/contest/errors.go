// Package contest provides contest management services.
package contest

import "errors"

// Service errors
var (
	ErrContestNotFound       = errors.New("contest: contest not found")
	ErrContestKeyExists      = errors.New("contest: contest key already exists")
	ErrInvalidFormatConfig   = errors.New("contest: invalid format config JSON")
	ErrInvalidLockedAfter    = errors.New("contest: invalid locked_after format, use ISO 8601")
	ErrInvalidStartTime      = errors.New("contest: invalid new_start_time format")
	ErrInvalidEndTime        = errors.New("contest: invalid new_end_time format")
	ErrParticipationNotFound = errors.New("contest: participation not found")
	ErrTagNotFound           = errors.New("contest: tag not found")
)
