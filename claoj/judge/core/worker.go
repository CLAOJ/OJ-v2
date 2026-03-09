package core

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/CLAOJ/claoj/judge/executors"
	"github.com/CLAOJ/claoj/judge/protocol"
)

// JudgeWorker handles grading for a single submission.
type JudgeWorker struct {
	submission *Submission
	executor   executors.Executor
	pm         *protocol.PacketManager
	abort      chan struct{}
	ctx        context.Context
	cancel     context.CancelFunc
}

// NewJudgeWorker creates a new worker for a submission.
func NewJudgeWorker(sub *Submission, exec executors.Executor, pm *protocol.PacketManager) *JudgeWorker {
	ctx, cancel := context.WithCancel(context.Background())
	return &JudgeWorker{
		submission: sub,
		executor:   exec,
		pm:         pm,
		abort:      make(chan struct{}),
		ctx:        ctx,
		cancel:     cancel,
	}
}

// Grade runs the complete grading process.
func (w *JudgeWorker) Grade() (*protocol.GratingResult, error) {
	result := &protocol.GratingResult{
		SubmissionID:    w.submission.ID,
		TestCaseResults: make([]protocol.TestCaseResult, 0),
	}

	// Create temporary directory for this submission
	workDir, err := os.MkdirTemp("", "judge-sub-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(workDir)

	// Step 1: Compile
	log.Printf("Worker %d: Compiling...", w.submission.ID)
	compileStart := time.Now()
	compileResult, err := w.executor.Compile(w.ctx, w.submission.Source, workDir)
	compileTime := time.Since(compileStart)

	if err != nil {
		return nil, err
	}

	if !compileResult.Success {
		w.pm.SendCompileError(w.submission.ID, compileResult.Error)
		return result, nil
	}

	log.Printf("Worker %d: Compiled in %v", w.submission.ID, compileTime)

	// Step 2: Load problem test cases
	problemDir := getProblemDir(w.submission.ProblemID)
	testCases, err := loadTestCases(problemDir)
	if err != nil {
		return nil, err
	}

	log.Printf("Worker %d: Running %d test cases...", w.submission.ID, len(testCases))

	// Step 3: Run test cases
	var totalPoints, maxPoints float64
	var maxMemory float64
	var totalTime float64
	var worstStatus string

	for i, tc := range testCases {
		select {
		case <-w.abort:
			w.pm.SendSubmissionAborted(w.submission.ID)
			return result, nil
		default:
		}

		tcResult, err := w.runTestCase(compileResult.BinaryPath, tc, workDir)
		if err != nil {
			log.Printf("Worker %d: Test case %d error: %v", w.submission.ID, i, err)
			continue
		}

		// Send test case result
		w.pm.SendTestCaseStatus(w.submission.ID, i, tcResult)
		result.TestCaseResults = append(result.TestCaseResults, tcResult)

		// Accumulate stats
		totalPoints += tcResult.Points
		maxPoints += tcResult.TotalPoints
		if tcResult.Memory > maxMemory {
			maxMemory = tcResult.Memory
		}
		totalTime += tcResult.Time

		// Track worst status
		if isWorseStatus(tcResult.Status, worstStatus) {
			worstStatus = tcResult.Status
		}

		// Check for short circuit (first non-AC stops grading)
		if w.submission.ShortCircuit && tcResult.Status != "AC" {
			break
		}
	}

	// Finalize result
	result.Points = totalPoints
	result.TotalPoints = maxPoints
	result.Time = totalTime
	result.Memory = maxMemory
	result.Status = worstStatus

	return result, nil
}

// runTestCase runs a single test case.
func (w *JudgeWorker) runTestCase(binaryPath string, tc TestCase, workDir string) (protocol.TestCaseResult, error) {
	result := protocol.TestCaseResult{
		Position:    tc.Position,
		TotalPoints: tc.Points,
	}

	// Prepare limits
	limits := executors.RunLimits{
		Time:       w.submission.TimeLimit,
		Memory:     w.submission.MemoryLimit,
		OutputSize: 64 * 1024, // 64KB output limit
	}

	// Run the solution
	runStart := time.Now()
	runResult, err := w.executor.Run(w.ctx, tc.Input, limits)
	runTime := time.Since(runStart)

	if err != nil {
		result.Status = "IE"
		result.Feedback = err.Error()
		return result, nil
	}

	result.Time = runTime.Seconds()
	result.Memory = float64(runResult.Memory) / 1024.0 // Convert to KB

	// Check for runtime errors
	if runResult.Status == executors.StatusRuntimeError {
		result.Status = "RTE"
		result.Feedback = runResult.Error
		result.Points = 0
		return result, nil
	}

	// Check for time limit exceeded
	if runResult.Status == executors.StatusTimeLimitExceeded {
		result.Status = "TLE"
		result.Points = 0
		return result, nil
	}

	// Check for memory limit exceeded
	if runResult.Status == executors.StatusMemoryLimitExceeded {
		result.Status = "MLE"
		result.Points = 0
		return result, nil
	}

	// Check output
	checkResult, err := checkOutput(tc.Expected, string(runResult.Output), tc.Checker)
	if err != nil {
		result.Status = "IE"
		result.Feedback = err.Error()
		result.Points = 0
		return result, nil
	}

	result.Status = checkResult.Status
	result.Points = checkResult.Points
	result.Feedback = checkResult.Feedback

	return result, nil
}

// RequestAbort signals the worker to stop grading.
func (w *JudgeWorker) RequestAbort() {
	select {
	case <-w.abort:
		return // Already aborted
	default:
		close(w.abort)
		w.cancel()
	}
}

// getProblemDir returns the path to a problem directory.
func getProblemDir(problemID string) string {
	// Search through problem globs to find the problem
	problemGlobs := []string{
		"/problems/*/" + problemID,
	}

	for _, glob := range problemGlobs {
		matches, _ := filepath.Glob(glob)
		if len(matches) > 0 {
			return matches[0]
		}
	}

	return ""
}

// TestCase represents a single test case.
type TestCase struct {
	Position int
	Input    string
	Expected string
	Points   float64
	Checker  string // Checker type: "standard", "custom", etc.
}

// loadTestCases loads test cases for a problem.
func loadTestCases(problemDir string) ([]TestCase, error) {
	var testCases []TestCase

	// Look for test case files (e.g., 1.in, 1.out, 2.in, 2.out, ...)
	// This is a simplified implementation
	for i := 1; i <= 10; i++ {
		inputPath := filepath.Join(problemDir, "tests", fmt.Sprintf("%d.in", i))
		outputPath := filepath.Join(problemDir, "tests", fmt.Sprintf("%d.out", i))

		inputData, err := os.ReadFile(inputPath)
		if err != nil {
			break // No more test cases
		}

		outputData, _ := os.ReadFile(outputPath)

		testCases = append(testCases, TestCase{
			Position: i - 1, // 0-indexed
			Input:    string(inputData),
			Expected: string(outputData),
			Points:   10.0, // Equal points per case
			Checker:  "standard",
		})
	}

	return testCases, nil
}

// CheckResult represents the result of output checking.
type CheckResult struct {
	Status   string
	Points   float64
	Feedback string
}

// checkOutput compares output with expected using the specified checker.
func checkOutput(expected, output, checkerType string) (*CheckResult, error) {
	// Use appropriate checker
	switch checkerType {
	case "standard", "":
		return checkStandard(expected, output)
	case "custom":
		// TODO: Load custom checker
		return checkStandard(expected, output)
	default:
		return checkStandard(expected, output)
	}
}

// checkStandard performs standard whitespace-insensitive comparison.
func checkStandard(expected, output string) (*CheckResult, error) {
	// Normalize whitespace
	normExpected := normalizeWhitespace(expected)
	normOutput := normalizeWhitespace(output)

	if normExpected == normOutput {
		return &CheckResult{
			Status: "AC",
			Points: 1.0,
		}, nil
	}

	return &CheckResult{
		Status:   "WA",
		Points:   0,
		Feedback: "Output mismatch",
	}, nil
}

// normalizeWhitespace normalizes whitespace for comparison.
func normalizeWhitespace(s string) string {
	// Split on whitespace and rejoin with single spaces
	fields := strings.Fields(s)
	return strings.Join(fields, " ")
}

// isWorseStatus returns true if status a is worse than status b.
func isWorseStatus(a, b string) bool {
	order := map[string]int{
		"AC":  0,
		"SC":  1,
		"W":   2,
		"WA":  2,
		"MLE": 3,
		"TLE": 4,
		"IR":  5,
		"RTE": 6,
		"OLE": 7,
		"CE":  8,
		"IE":  9,
	}

	return order[a] > order[b]
}
