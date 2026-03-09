package executors

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestCPP17Executor_Compile tests C++17 compilation.
func TestCPP17Executor_Compile(t *testing.T) {
	exec := NewCPP17Executor()

	dir, err := os.MkdirTemp("", "judge-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(dir)

	// Simple valid C++17 program
	source := `#include <iostream>
int main() {
    std::cout << "Hello, World!" << std::endl;
    return 0;
}`

	ctx := context.Background()
	result, err := exec.Compile(ctx, source, dir)
	if err != nil {
		t.Fatalf("Compile failed: %v", err)
	}

	if !result.Success {
		t.Errorf("Compilation failed: %s", result.Error)
	}

	// Verify binary exists
	if _, err := os.Stat(result.BinaryPath); os.IsNotExist(err) {
		t.Errorf("Binary not created at %s", result.BinaryPath)
	}
}

// TestCPP17Executor_CompileError tests compilation error handling.
func TestCPP17Executor_CompileError(t *testing.T) {
	exec := NewCPP17Executor()

	dir, err := os.MkdirTemp("", "judge-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(dir)

	// Invalid C++ code
	source := `int main() { invalid_syntax_error }`

	ctx := context.Background()
	result, err := exec.Compile(ctx, source, dir)
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	if result.Success {
		t.Error("Expected compilation to fail")
	}

	if result.Error == "" {
		t.Error("Expected error message")
	}
}

// TestCPP17Executor_RuntimeVersions tests version detection.
func TestCPP17Executor_RuntimeVersions(t *testing.T) {
	exec := NewCPP17Executor()

	versions := exec.RuntimeVersions()
	if len(versions) == 0 {
		t.Error("Expected at least one version string")
	}

	t.Logf("g++ version: %s", versions[0])
}

// TestC11Executor_Compile tests C11 compilation.
func TestC11Executor_Compile(t *testing.T) {
	exec := NewC11Executor()

	dir, err := os.MkdirTemp("", "judge-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(dir)

	source := `#include <stdio.h>
int main() {
    printf("Hello from C!\n");
    return 0;
}`

	ctx := context.Background()
	result, err := exec.Compile(ctx, source, dir)
	if err != nil {
		t.Fatalf("Compile failed: %v", err)
	}

	if !result.Success {
		t.Errorf("Compilation failed: %s", result.Error)
	}
}

// TestGoExecutor_Compile tests Go compilation.
func TestGoExecutor_Compile(t *testing.T) {
	exec := NewGoExecutor()

	dir, err := os.MkdirTemp("", "judge-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(dir)

	source := `package main
import "fmt"
func main() {
    fmt.Println("Hello from Go!")
}`

	ctx := context.Background()
	result, err := exec.Compile(ctx, source, dir)
	if err != nil {
		t.Fatalf("Compile failed: %v", err)
	}

	if !result.Success {
		t.Errorf("Compilation failed: %s", result.Error)
	}
}

// TestPython3Executor_Compile tests Python syntax checking.
func TestPython3Executor_Compile(t *testing.T) {
	exec := NewPython3Executor()

	dir, err := os.MkdirTemp("", "judge-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(dir)

	// Valid Python
	source := `print("Hello from Python!")`

	ctx := context.Background()
	result, err := exec.Compile(ctx, source, dir)
	if err != nil {
		t.Fatalf("Compile failed: %v", err)
	}

	if !result.Success {
		t.Errorf("Compilation failed: %s", result.Error)
	}
}

// TestPython3Executor_CompileError tests Python syntax error detection.
func TestPython3Executor_CompileError(t *testing.T) {
	exec := NewPython3Executor()

	dir, err := os.MkdirTemp("", "judge-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(dir)

	// Invalid Python
	source := `print("missing close paren`

	ctx := context.Background()
	result, err := exec.Compile(ctx, source, dir)
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	if result.Success {
		t.Error("Expected compilation to fail")
	}
}

// TestNodeJSExecutor_Compile tests Node.js syntax checking.
func TestNodeJSExecutor_Compile(t *testing.T) {
	exec := NewNodeJSExecutor()

	dir, err := os.MkdirTemp("", "judge-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(dir)

	source := `console.log("Hello from Node.js!");`

	ctx := context.Background()
	result, err := exec.Compile(ctx, source, dir)
	if err != nil {
		t.Fatalf("Compile failed: %v", err)
	}

	if !result.Success {
		t.Errorf("Compilation failed: %s", result.Error)
	}
}

// TestRustExecutor_Compile tests Rust compilation.
func TestRustExecutor_Compile(t *testing.T) {
	exec := NewRustExecutor()

	dir, err := os.MkdirTemp("", "judge-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(dir)

	source := `fn main() {
    println!("Hello from Rust!");
}`

	ctx := context.Background()
	result, err := exec.Compile(ctx, source, dir)
	if err != nil {
		t.Fatalf("Compile failed: %v", err)
	}

	if !result.Success {
		t.Errorf("Compilation failed: %s", result.Error)
	}
}

// TestJava8Executor_Compile tests Java compilation.
func TestJava8Executor_Compile(t *testing.T) {
	exec := NewJava8Executor()

	dir, err := os.MkdirTemp("", "judge-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(dir)

	source := `public class Main {
    public static void main(String[] args) {
        System.out.println("Hello from Java!");
    }
}`

	ctx := context.Background()
	result, err := exec.Compile(ctx, source, dir)
	if err != nil {
		t.Fatalf("Compile failed: %v", err)
	}

	if !result.Success {
		t.Errorf("Compilation failed: %s", result.Error)
	}
}

// TestLanguageIdentifiers tests that all executors return correct language IDs.
func TestLanguageIdentifiers(t *testing.T) {
	tests := []struct {
		exec     Executor
		expected string
	}{
		{NewCPP17Executor(), "CPP17"},
		{NewCPP20Executor(), "CPP20"},
		{NewC11Executor(), "C11"},
		{NewPython3Executor(), "PY3"},
		{NewGoExecutor(), "GO"},
		{NewNodeJSExecutor(), "NODEJS"},
		{NewRustExecutor(), "RUST"},
		{NewJava8Executor(), "JAVA8"},
	}

	for _, test := range tests {
		t.Run(test.expected, func(t *testing.T) {
			if got := test.exec.Language(); got != test.expected {
				t.Errorf("Expected %s, got %s", test.expected, got)
			}
		})
	}
}

// BenchmarkCPP17Executor_Compile benchmarks C++ compilation.
func BenchmarkCPP17Executor_Compile(b *testing.B) {
	exec := NewCPP17Executor()
	dir, _ := os.MkdirTemp("", "judge-bench-*")
	defer os.RemoveAll(dir)

	source := `#include <iostream>
int main() { return 0; }`

	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		exec.Compile(ctx, source, dir)
	}
}

// BenchmarkPython3Executor_Compile benchmarks Python syntax checking.
func BenchmarkPython3Executor_Compile(b *testing.B) {
	exec := NewPython3Executor()
	dir, _ := os.MkdirTemp("", "judge-bench-*")
	defer os.RemoveAll(dir)

	source := `print("hello")`

	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		exec.Compile(ctx, source, dir)
	}
}
