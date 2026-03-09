package sandbox

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/CLAOJ/claoj/judge/executors"
)

// TestSandbox_BasicExecution tests basic program execution.
func TestSandbox_BasicExecution(t *testing.T) {
	// Create a simple test program
	testDir, err := os.MkdirTemp("", "sandbox-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(testDir)

	// Create a simple C program
	sourcePath := filepath.Join(testDir, "hello.c")
	source := `#include <stdio.h>
int main() {
    printf("Hello, World!\n");
    return 0;
}`
	if err := os.WriteFile(sourcePath, []byte(source), 0644); err != nil {
		t.Fatalf("Failed to write source: %v", err)
	}

	// Compile
	binaryPath := filepath.Join(testDir, "hello")
	cmd := exec.Command("gcc", "-O2", "-o", binaryPath, sourcePath)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("Compilation failed: %v\n%s", err, output)
	}

	// Run in sandbox
	sandbox := DefaultSandbox()
	ctx := context.Background()
	limits := executors.RunLimits{
		Time:       2 * time.Second,
		Memory:     256 * 1024 * 1024,
		OutputSize: 64 * 1024,
	}

	result, err := sandbox.Run(ctx, binaryPath, "", limits)
	if err != nil {
		t.Fatalf("Sandbox run failed: %v", err)
	}

	if result.Status != executors.StatusAccepted {
		t.Errorf("Expected AC, got %s", result.Status)
	}

	expectedOutput := "Hello, World!\n"
	if string(result.Output) != expectedOutput {
		t.Errorf("Expected output %q, got %q", expectedOutput, string(result.Output))
	}
}

// TestSandbox_TimeLimit tests time limit enforcement.
func TestSandbox_TimeLimit(t *testing.T) {
	testDir, err := os.MkdirTemp("", "sandbox-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(testDir)

	// Create infinite loop program
	sourcePath := filepath.Join(testDir, "infinite.c")
	source := `int main() {
    while (1) { }
    return 0;
}`
	if err := os.WriteFile(sourcePath, []byte(source), 0644); err != nil {
		t.Fatalf("Failed to write source: %v", err)
	}

	// Compile
	binaryPath := filepath.Join(testDir, "infinite")
	cmd := exec.Command("gcc", "-O2", "-o", binaryPath, sourcePath)
	cmd.CombinedOutput()

	// Run with short time limit
	sandbox := DefaultSandbox()
	ctx := context.Background()
	limits := executors.RunLimits{
		Time:       500 * time.Millisecond,
		Memory:     256 * 1024 * 1024,
		OutputSize: 64 * 1024,
	}

	result, err := sandbox.Run(ctx, binaryPath, "", limits)
	if err != nil {
		t.Fatalf("Sandbox run failed: %v", err)
	}

	if result.Status != executors.StatusTimeLimitExceeded {
		t.Errorf("Expected TLE, got %s", result.Status)
	}
}

// TestSandbox_MemoryLimit tests memory limit enforcement.
func TestSandbox_MemoryLimit(t *testing.T) {
	testDir, err := os.MkdirTemp("", "sandbox-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(testDir)

	// Create memory allocation program
	sourcePath := filepath.Join(testDir, "alloc.c")
	source := `#include <stdlib.h>
#include <string.h>
int main() {
    char *buf = malloc(100 * 1024 * 1024); // 100MB
    memset(buf, 0, 100 * 1024 * 1024);
    return 0;
}`
	if err := os.WriteFile(sourcePath, []byte(source), 0644); err != nil {
		t.Fatalf("Failed to write source: %v", err)
	}

	// Compile
	binaryPath := filepath.Join(testDir, "alloc")
	cmd := exec.Command("gcc", "-O2", "-o", binaryPath, sourcePath)
	cmd.CombinedOutput()

	// Run with low memory limit
	sandbox := DefaultSandbox()
	ctx := context.Background()
	limits := executors.RunLimits{
		Time:       2 * time.Second,
		Memory:     32 * 1024 * 1024, // 32MB limit
		OutputSize: 64 * 1024,
	}

	result, err := sandbox.Run(ctx, binaryPath, "", limits)
	if err != nil {
		t.Fatalf("Sandbox run failed: %v", err)
	}

	// Should either get MLE or RTE (killed by OOM)
	if result.Status != executors.StatusMemoryLimitExceeded &&
		result.Status != executors.StatusRuntimeError {
		t.Errorf("Expected MLE or RTE, got %s", result.Status)
	}
}

// TestSandbox_OutputLimit tests output limit enforcement.
func TestSandbox_OutputLimit(t *testing.T) {
	testDir, err := os.MkdirTemp("", "sandbox-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(testDir)

	// Create verbose output program
	sourcePath := filepath.Join(testDir, "verbose.c")
	source := `#include <stdio.h>
int main() {
    for (int i = 0; i < 10000; i++) {
        printf("Line %d\n", i);
    }
    return 0;
}`
	if err := os.WriteFile(sourcePath, []byte(source), 0644); err != nil {
		t.Fatalf("Failed to write source: %v", err)
	}

	// Compile
	binaryPath := filepath.Join(testDir, "verbose")
	cmd := exec.Command("gcc", "-O2", "-o", binaryPath, sourcePath)
	cmd.CombinedOutput()

	// Run with small output limit
	sandbox := DefaultSandbox()
	ctx := context.Background()
	limits := executors.RunLimits{
		Time:       2 * time.Second,
		Memory:     256 * 1024 * 1024,
		OutputSize: 1024, // 1KB
	}

	result, err := sandbox.Run(ctx, binaryPath, "", limits)
	if err != nil {
		t.Fatalf("Sandbox run failed: %v", err)
	}

	if result.Status != executors.StatusOutputLimitExceeded {
		t.Errorf("Expected OLE, got %s", result.Status)
	}
}

// TestDefaultConfig tests that default config is secure.
func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.EnableNetwork {
		t.Error("Network should be disabled by default")
	}

	if cfg.MaxProcesses != 1 {
		t.Errorf("Expected MaxProcesses=1, got %d", cfg.MaxProcesses)
	}

	if cfg.MemoryLimit <= 0 {
		t.Error("Memory limit should be positive")
	}
}

// TestRunSandboxed tests the convenience function.
func TestRunSandboxed(t *testing.T) {
	testDir, err := os.MkdirTemp("", "sandbox-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(testDir)

	// Create simple program
	sourcePath := filepath.Join(testDir, "simple.c")
	source := `#include <stdio.h>
int main() {
    printf("OK\n");
    return 0;
}`
	os.WriteFile(sourcePath, []byte(source), 0644)

	binaryPath := filepath.Join(testDir, "simple")
	exec.Command("gcc", "-O2", "-o", binaryPath, sourcePath).CombinedOutput()

	ctx := context.Background()
	limits := executors.RunLimits{
		Time:       1 * time.Second,
		Memory:     256 * 1024 * 1024,
		OutputSize: 64 * 1024,
	}

	result, err := RunSandboxed(ctx, binaryPath, "", limits)
	if err != nil {
		t.Fatalf("RunSandboxed failed: %v", err)
	}

	if result.Status != executors.StatusAccepted {
		t.Errorf("Expected AC, got %s", result.Status)
	}
}
