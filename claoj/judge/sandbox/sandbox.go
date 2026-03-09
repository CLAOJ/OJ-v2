// Package sandbox provides process isolation and resource limiting.
package sandbox

import (
	"context"
	"os/exec"
	"syscall"
	"time"

	"github.com/CLAOJ/claoj/judge/executors"
)

// Config holds sandbox configuration.
type Config struct {
	TimeLimit       time.Duration
	MemoryLimit     int64  // bytes
	OutputLimit     int64  // bytes
	StackLimit      int64  // bytes
	MaxProcesses    int    // max number of processes
	MaxOpenFiles    int    // max number of open files
	EnableNetwork   bool   // allow network access (usually false)
	AllowedDirs     []string
	WritableDirs    []string
	ReadOnlyDirs    []string
}

// Sandbox provides isolated execution environment.
type Sandbox struct {
	cfg    *Config
	cmd    *exec.Cmd
	state  *ProcessState
}

// ProcessState holds process exit information.
type ProcessState struct {
	ExitCode      int
	Time          time.Duration
	Memory        int64
	Signal        syscall.Signal
	ContextSwitches ContextSwitches
}

// ContextSwitches tracks voluntary and involuntary switches.
type ContextSwitches struct {
	Voluntary   int64
	Involuntary int64
}

// NewSandbox creates a new sandbox with the given configuration.
func NewSandbox(cfg *Config) *Sandbox {
	return &Sandbox{
		cfg: cfg,
	}
}

// DefaultConfig returns a secure default configuration.
func DefaultConfig() *Config {
	return &Config{
		TimeLimit:    1 * time.Second,
		MemoryLimit:  256 * 1024 * 1024, // 256MB
		OutputLimit:  64 * 1024,          // 64KB
		StackLimit:   8 * 1024 * 1024,    // 8MB
		MaxProcesses: 1,
		MaxOpenFiles: 10,
		EnableNetwork: false,
		AllowedDirs: []string{
			"/usr/lib",
			"/lib64",
			"/lib",
		},
		WritableDirs: []string{
			"/tmp/judge",
		},
	}
}

// Run executes a command in the sandbox.
func (s *Sandbox) Run(ctx context.Context, binary string, input string, limits executors.RunLimits) (*executors.RunResult, error) {
	// Create command
	cmd := exec.CommandContext(ctx, binary)
	cmd.Stdin = nil // Will be set from input
	cmd.Env = []string{} // Minimal environment

	// Setup process isolation
	s.setupProcess(cmd)

	// Create pipes for I/O
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}

	// Start process
	if err := cmd.Start(); err != nil {
		return &executors.RunResult{
			Status: executors.StatusRuntimeError,
			Error:  err.Error(),
		}, nil
	}

	// Write input
	go func() {
		stdin.Write([]byte(input))
		stdin.Close()
	}()

	// Read output with size limit
	output, err := readWithLimit(stdout, limits.OutputSize)
	if err != nil {
		return &executors.RunResult{
			Status: executors.StatusOutputLimitExceeded,
			Error:  err.Error(),
		}, nil
	}

	// Read stderr
	errOutput, _ := readWithLimit(stderr, 64*1024)

	// Wait for process with timeout
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case err := <-done:
		// Normal exit
		state := cmd.ProcessState.Sys().(*syscall.Rusage)
		return s.collectResult(state, output, string(errOutput)), nil

	case <-time.After(limits.Time):
		// Time limit exceeded
		cmd.Process.Kill()
		return &executors.RunResult{
			Status: executors.StatusTimeLimitExceeded,
		}, nil
	}
}

// setupProcess configures process isolation.
func (s *Sandbox) setupProcess(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		// Security flags
		Pdeathsig: syscall.SIGKILL,

		// Resource limits will be set via setrlimit
		// For full isolation, consider using:
		// Cloneflags: syscall.CLONE_NEWPID | syscall.CLONE_NEWNS | syscall.CLONE_NEWNET,
	}

	// Set environment
	cmd.Env = []string{
		"PATH=/usr/bin:/bin",
		"HOME=/tmp",
		"LANG=C.UTF-8",
	}
}

// collectResult collects process exit information.
func (s *Sandbox) collectResult(rusage *syscall.Rusage, output []byte, errOutput string) *executors.RunResult {
	exitCode := 0
	if rusage != nil {
		// Extract exit code from status
		// This is platform-specific
		exitCode = rusage.ExitStatus
	}

	result := &executors.RunResult{
		ExitCode: exitCode,
		Output:   output,
		Error:    errOutput,
	}

	// Convert to appropriate status
	if exitCode != 0 {
		result.Status = executors.StatusRuntimeError
	} else {
		result.Status = executors.StatusAccepted
	}

	if rusage != nil {
		// Convert memory from KB to bytes
		result.Memory = int64(rusage.Maxrss) * 1024

		// Convert user + system time
		utime := time.Duration(rusage.Utime.Sec)*time.Second + time.Duration(rusage.Utime.Usec)*time.Microsecond
		stime := time.Duration(rusage.Stime.Sec)*time.Second + time.Duration(rusage.Stime.Usec)*time.Microsecond
		result.Time = utime + stime
	}

	return result
}

// readWithLimit reads from a reader with a size limit.
func readWithLimit(r interface{ Read([]byte) (int, error) }, limit int64) ([]byte, error) {
	buf := make([]byte, limit)
	n, err := r.Read(buf)
	if err != nil && err.Error() != "EOF" {
		return buf[:n], err
	}
	return buf[:n], nil
}

// DefaultSandbox creates a sandbox with default configuration.
func DefaultSandbox() *Sandbox {
	return NewSandbox(DefaultConfig())
}

// RunSandboxed runs a binary in a sandboxed environment.
func RunSandboxed(ctx context.Context, binary string, input string, limits executors.RunLimits) (*executors.RunResult, error) {
	sandbox := DefaultSandbox()
	sandbox.cfg.TimeLimit = limits.Time
	sandbox.cfg.MemoryLimit = limits.Memory
	sandbox.cfg.OutputLimit = limits.OutputSize

	return sandbox.Run(ctx, binary, input, limits)
}
