package executors

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// CPPExecutor handles C++ compilation and execution.
type CPPExecutor struct {
	baseExecutor
	Compiler string
	Standard string
	Flags    []string
}

// NewCPP17Executor creates a C++17 executor.
func NewCPP17Executor() *CPPExecutor {
	return &CPPExecutor{
		Compiler: "g++",
		Standard: "-std=c++17",
		Flags: []string{
			"-O2",
			"-Wall",
			"-static",
			"-o",
		},
	}
}

// NewCPP20Executor creates a C++20 executor.
func NewCPP20Executor() *CPPExecutor {
	return &CPPExecutor{
		Compiler: "g++",
		Standard: "-std=c++20",
		Flags: []string{
			"-O2",
			"-Wall",
			"-static",
			"-o",
		},
	}
}

func (e *CPPExecutor) Language() string {
	if strings.Contains(e.Standard, "c++20") {
		return "CPP20"
	}
	return "CPP17"
}

func (e *CPPExecutor) Compile(ctx context.Context, source string, dir string) (*CompileResult, error) {
	// Write source file
	sourcePath := filepath.Join(dir, "solution.cpp")
	if err := os.WriteFile(sourcePath, []byte(source), 0644); err != nil {
		return nil, err
	}

	binaryPath := filepath.Join(dir, "solution")
	e.binaryPath = binaryPath

	// Build command
	args := []string{
		e.Standard,
		"-O2",
		"-Wall",
		"-static",
		"-o", binaryPath,
		sourcePath,
	}

	cmd := exec.CommandContext(ctx, e.Compiler, args...)
	start := time.Now()
	output, err := cmd.CombinedOutput()
	compileTime := time.Since(start)

	if err != nil {
		return &CompileResult{
			Success: false,
			Error:   string(output),
			Time:    compileTime,
		}, nil
	}

	// Verify binary exists
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		return &CompileResult{
			Success: false,
			Error:   "Binary not created",
			Time:    compileTime,
		}, nil
	}

	return &CompileResult{
		Success:    true,
		BinaryPath: binaryPath,
		Time:       compileTime,
	}, nil
}

func (e *CPPExecutor) Run(ctx context.Context, input string, limits RunLimits) (*RunResult, error) {
	return runSandboxed(ctx, e.binaryPath, input, limits)
}

func (e *CPPExecutor) RuntimeVersions() []string {
	cmd := exec.Command(e.Compiler, "--version")
	output, err := cmd.Output()
	if err != nil {
		return []string{"unknown"}
	}

	// Extract version number
	re := regexp.MustCompile(`g\+\+.*?(\d+\.\d+\.\d+)`)
	matches := re.FindStringSubmatch(string(output))
	if len(matches) > 1 {
		return []string{"g++ " + matches[1]}
	}

	return []string{strings.TrimSpace(string(output))}
}

// C11Executor handles C11 compilation and execution.
type C11Executor struct {
	baseExecutor
	Compiler string
	Standard string
}

// NewC11Executor creates a C11 executor.
func NewC11Executor() *C11Executor {
	return &C11Executor{
		Compiler: "gcc",
		Standard: "-std=c11",
	}
}

func (e *C11Executor) Language() string {
	return "C11"
}

func (e *C11Executor) Compile(ctx context.Context, source string, dir string) (*CompileResult, error) {
	sourcePath := filepath.Join(dir, "solution.c")
	if err := os.WriteFile(sourcePath, []byte(source), 0644); err != nil {
		return nil, err
	}

	binaryPath := filepath.Join(dir, "solution")
	e.binaryPath = binaryPath

	cmd := exec.CommandContext(ctx, e.Compiler,
		e.Standard,
		"-O2",
		"-Wall",
		"-static",
		"-o", binaryPath,
		sourcePath,
	)

	start := time.Now()
	output, err := cmd.CombinedOutput()
	compileTime := time.Since(start)

	if err != nil {
		return &CompileResult{
			Success: false,
			Error:   string(output),
			Time:    compileTime,
		}, nil
	}

	return &CompileResult{
		Success:    true,
		BinaryPath: binaryPath,
		Time:       compileTime,
	}, nil
}

func (e *C11Executor) Run(ctx context.Context, input string, limits RunLimits) (*RunResult, error) {
	return runSandboxed(ctx, e.binaryPath, input, limits)
}

func (e *C11Executor) RuntimeVersions() []string {
	cmd := exec.Command(e.Compiler, "--version")
	output, _ := cmd.Output()
	return []string{strings.TrimSpace(string(output))}
}
