// Package executors provides language execution support.
package executors

import (
	"context"
	"errors"
	"time"
)

// ErrNotImplemented indicates a method is not implemented.
var ErrNotImplemented = errors.New("not implemented")

// Executor defines the interface for language runners.
type Executor interface {
	// Language returns the language identifier.
	Language() string

	// Compile compiles source code.
	Compile(ctx context.Context, source string, dir string) (*CompileResult, error)

	// Run executes compiled code.
	Run(ctx context.Context, input string, limits RunLimits) (*RunResult, error)

	// RuntimeVersions returns version information.
	RuntimeVersions() []string
}

// CompileResult contains compilation results.
type CompileResult struct {
	Success    bool
	BinaryPath string
	Error      string
	Warnings   string
	Time       time.Duration
	Memory     int64
}

// RunLimits specifies execution limits.
type RunLimits struct {
	Time       time.Duration
	Memory     int64
	OutputSize int64
}

// RunResult contains execution results.
type RunResult struct {
	Status     RunStatus
	ExitCode   int
	Time       time.Duration
	Memory     int64
	Output     []byte
	Error      string
	Signal     int
}

// RunStatus represents execution status.
type RunStatus int

const (
	StatusAccepted RunStatus = iota
	StatusWrongAnswer
	StatusTimeLimitExceeded
	StatusMemoryLimitExceeded
	StatusOutputLimitExceeded
	StatusRuntimeError
	StatusInternalError
	StatusCompileError
)

// String returns string representation of RunStatus.
func (s RunStatus) String() string {
	switch s {
	case StatusAccepted:
		return "AC"
	case StatusWrongAnswer:
		return "WA"
	case StatusTimeLimitExceeded:
		return "TLE"
	case StatusMemoryLimitExceeded:
		return "MLE"
	case StatusOutputLimitExceeded:
		return "OLE"
	case StatusRuntimeError:
		return "RTE"
	case StatusInternalError:
		return "IE"
	case StatusCompileError:
		return "CE"
	default:
		return "Unknown"
	}
}

// baseExecutor provides common functionality for all executors.
type baseExecutor struct {
	binaryPath string
	workDir    string
}
