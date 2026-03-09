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
}

// NewCPP17Executor creates a C++17 executor.
func NewCPP17Executor() *CPPExecutor {
	return &CPPExecutor{
		Compiler: "g++",
		Standard: "-std=c++17",
	}
}

// NewCPP20Executor creates a C++20 executor.
func NewCPP20Executor() *CPPExecutor {
	return &CPPExecutor{
		Compiler: "g++",
		Standard: "-std=c++20",
	}
}

func (e *CPPExecutor) Language() string {
	if strings.Contains(e.Standard, "c++20") {
		return "CPP20"
	}
	return "CPP17"
}

func (e *CPPExecutor) Compile(ctx context.Context, source string, dir string) (*CompileResult, error) {
	sourcePath := filepath.Join(dir, "solution.cpp")
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

// Python3Executor handles Python 3 execution.
type Python3Executor struct {
	baseExecutor
	Interpreter string
}

// NewPython3Executor creates a Python 3 executor.
func NewPython3Executor() *Python3Executor {
	return &Python3Executor{
		Interpreter: "python3",
	}
}

// NewPython2Executor creates a Python 2 executor.
func NewPython2Executor() *Python3Executor {
	return &Python3Executor{
		Interpreter: "python2",
	}
}

func (e *Python3Executor) Language() string {
	if e.Interpreter == "python2" {
		return "PY2"
	}
	return "PY3"
}

func (e *Python3Executor) Compile(ctx context.Context, source string, dir string) (*CompileResult, error) {
	sourcePath := filepath.Join(dir, "solution.py")
	if err := os.WriteFile(sourcePath, []byte(source), 0644); err != nil {
		return nil, err
	}

	e.binaryPath = sourcePath

	cmd := exec.CommandContext(ctx, e.Interpreter, "-m", "py_compile", sourcePath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return &CompileResult{
			Success: false,
			Error:   string(output),
		}, nil
	}

	return &CompileResult{
		Success:    true,
		BinaryPath: sourcePath,
	}, nil
}

func (e *Python3Executor) Run(ctx context.Context, input string, limits RunLimits) (*RunResult, error) {
	return runSandboxed(ctx, e.Interpreter, input, limits)
}

func (e *Python3Executor) RuntimeVersions() []string {
	cmd := exec.Command(e.Interpreter, "--version")
	output, _ := cmd.CombinedOutput()
	return []string{strings.TrimSpace(string(output))}
}

// JavaExecutor handles Java compilation and execution.
type JavaExecutor struct {
	baseExecutor
	Compiler  string
	Runner    string
	ClassName string
}

// NewJava8Executor creates a Java 8 executor.
func NewJava8Executor() *JavaExecutor {
	return &JavaExecutor{
		Compiler:  "javac",
		Runner:    "java",
		ClassName: "Main",
	}
}

func (e *JavaExecutor) Language() string {
	return "JAVA8"
}

func (e *JavaExecutor) Compile(ctx context.Context, source string, dir string) (*CompileResult, error) {
	sourcePath := filepath.Join(dir, "Main.java")
	if err := os.WriteFile(sourcePath, []byte(source), 0644); err != nil {
		return nil, err
	}

	cmd := exec.CommandContext(ctx, e.Compiler,
		"-encoding", "UTF-8",
		"-d", dir,
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

	classPath := filepath.Join(dir, "Main.class")
	if _, err := os.Stat(classPath); os.IsNotExist(err) {
		return &CompileResult{
			Success: false,
			Error:   "Class file not created",
			Time:    compileTime,
		}, nil
	}

	e.binaryPath = dir
	return &CompileResult{
		Success:    true,
		BinaryPath: dir,
		Time:       compileTime,
	}, nil
}

func (e *JavaExecutor) Run(ctx context.Context, input string, limits RunLimits) (*RunResult, error) {
	// For Java, we need to run the java command
	return runSandboxed(ctx, e.Runner+" "+e.ClassName, input, limits)
}

func (e *JavaExecutor) RuntimeVersions() []string {
	cmd := exec.Command(e.Compiler, "-version")
	output, _ := cmd.CombinedOutput()
	return []string{strings.TrimSpace(string(output))}
}

// GoExecutor handles Go compilation and execution.
type GoExecutor struct {
	baseExecutor
	Compiler string
}

// NewGoExecutor creates a Go executor.
func NewGoExecutor() *GoExecutor {
	return &GoExecutor{
		Compiler: "go",
	}
}

func (e *GoExecutor) Language() string {
	return "GO"
}

func (e *GoExecutor) Compile(ctx context.Context, source string, dir string) (*CompileResult, error) {
	sourcePath := filepath.Join(dir, "solution.go")
	if err := os.WriteFile(sourcePath, []byte(source), 0644); err != nil {
		return nil, err
	}

	binaryPath := filepath.Join(dir, "solution")
	e.binaryPath = binaryPath

	cmd := exec.CommandContext(ctx, e.Compiler, "build", "-o", binaryPath, sourcePath)
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0")

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

func (e *GoExecutor) Run(ctx context.Context, input string, limits RunLimits) (*RunResult, error) {
	return runSandboxed(ctx, e.binaryPath, input, limits)
}

func (e *GoExecutor) RuntimeVersions() []string {
	cmd := exec.Command(e.Compiler, "version")
	output, _ := cmd.Output()
	return []string{strings.TrimSpace(string(output))}
}

// NodeJSExecutor handles Node.js execution.
type NodeJSExecutor struct {
	baseExecutor
	Runtime string
}

// NewNodeJSExecutor creates a Node.js executor.
func NewNodeJSExecutor() *NodeJSExecutor {
	return &NodeJSExecutor{
		Runtime: "node",
	}
}

func (e *NodeJSExecutor) Language() string {
	return "NODEJS"
}

func (e *NodeJSExecutor) Compile(ctx context.Context, source string, dir string) (*CompileResult, error) {
	sourcePath := filepath.Join(dir, "solution.js")
	if err := os.WriteFile(sourcePath, []byte(source), 0644); err != nil {
		return nil, err
	}

	e.binaryPath = sourcePath

	cmd := exec.CommandContext(ctx, e.Runtime, "--check", sourcePath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return &CompileResult{
			Success: false,
			Error:   string(output),
		}, nil
	}

	return &CompileResult{
		Success:    true,
		BinaryPath: sourcePath,
	}, nil
}

func (e *NodeJSExecutor) Run(ctx context.Context, input string, limits RunLimits) (*RunResult, error) {
	return runSandboxed(ctx, e.Runtime, input, limits)
}

func (e *NodeJSExecutor) RuntimeVersions() []string {
	cmd := exec.Command(e.Runtime, "--version")
	output, _ := cmd.Output()
	return []string{strings.TrimSpace(string(output))}
}

// RustExecutor handles Rust compilation and execution.
type RustExecutor struct {
	baseExecutor
	Compiler string
}

// NewRustExecutor creates a Rust executor.
func NewRustExecutor() *RustExecutor {
	return &RustExecutor{
		Compiler: "rustc",
	}
}

func (e *RustExecutor) Language() string {
	return "RUST"
}

func (e *RustExecutor) Compile(ctx context.Context, source string, dir string) (*CompileResult, error) {
	sourcePath := filepath.Join(dir, "solution.rs")
	if err := os.WriteFile(sourcePath, []byte(source), 0644); err != nil {
		return nil, err
	}

	binaryPath := filepath.Join(dir, "solution")
	e.binaryPath = binaryPath

	cmd := exec.CommandContext(ctx, e.Compiler, "-O", "-o", binaryPath, sourcePath)

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

func (e *RustExecutor) Run(ctx context.Context, input string, limits RunLimits) (*RunResult, error) {
	return runSandboxed(ctx, e.binaryPath, input, limits)
}

func (e *RustExecutor) RuntimeVersions() []string {
	cmd := exec.Command(e.Compiler, "--version")
	output, _ := cmd.Output()
	return []string{strings.TrimSpace(string(output))}
}
