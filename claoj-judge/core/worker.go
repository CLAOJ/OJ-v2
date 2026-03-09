package core

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/CLAOJ/claoj-judge/executors"
	"github.com/CLAOJ/claoj-judge/protocol"
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
func (w *JudgeWorker) Grade() (*protocol.GradingResult, error) {
	result := &protocol.GradingResult{
		SubmissionID:    w.submission.ID,
		TestCaseResults: make([]protocol.TestCaseResult, 0),
	}

	// Create temporary directory for this submission
	workDir, err := os.MkdirTemp("", fmt.Sprintf("judge-sub-%d-*", w.submission.ID))
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(workDir)

	log.Printf("Worker %d: Working directory: %s", w.submission.ID, workDir)

	// Step 1: Compile
	log.Printf("Worker %d: Compiling...", w.submission.ID)
	compileStart := time.Now()
	compileResult, err := w.executor.Compile(w.ctx, w.submission.Source, workDir)
	compileTime := time.Since(compileStart)

	if err != nil {
		return nil, fmt.Errorf("compile error: %w", err)
	}

	if !compileResult.Success {
		w.pm.SendCompileError(w.submission.ID, compileResult.Error)
		return result, nil
	}

	log.Printf("Worker %d: Compiled in %v", w.submission.ID, compileTime)

	// Step 2: Load problem test cases
	problemDir := w.getProblemDir(w.submission.ProblemID)
	if problemDir == "" {
		return nil, fmt.Errorf("problem directory not found: %s", w.submission.ProblemID)
	}

	testCases, err := w.loadTestCases(problemDir)
	if err != nil {
		return nil, fmt.Errorf("failed to load test cases: %w", err)
	}

	if len(testCases) == 0 {
		return nil, fmt.Errorf("no test cases found for problem: %s", w.submission.ProblemID)
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
			log.Printf("Worker %d: Aborted", w.submission.ID)
			w.pm.SendSubmissionAborted(w.submission.ID)
			return result, nil
		default:
		}

		tcResult, err := w.runTestCase(compileResult.BinaryPath, tc, workDir, i)
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
			log.Printf("Worker %d: Short circuit at test case %d", w.submission.ID, i)
			break
		}
	}

	// Finalize result
	result.Points = totalPoints
	result.TotalPoints = maxPoints
	result.Time = totalTime
	result.Memory = maxMemory
	result.Status = worstStatus

	log.Printf("Worker %d: Grading complete - %s (%.0f/%.0f points)",
		w.submission.ID, result.Status, result.Points, result.TotalPoints)

	return result, nil
}

// runTestCase runs a single test case.
func (w *JudgeWorker) runTestCase(binaryPath string, tc TestCase, workDir string, index int) (protocol.TestCaseResult, error) {
	result := protocol.TestCaseResult{
		Position:    index,
		TotalPoints: tc.Points,
	}

	// Prepare limits
	limits := executors.RunLimits{
		Time:       w.submission.TimeLimit,
		Memory:     w.submission.MemoryLimit,
		OutputSize: 64 * 1024, // 64KB output limit
	}

	// Read input file
	inputData, err := os.ReadFile(tc.InputPath)
	if err != nil {
		result.Status = "IE"
		result.Feedback = "Failed to read input file"
		return result, nil
	}

	// Run the solution
	runStart := time.Now()
	runResult, err := w.executor.Run(w.ctx, string(inputData), limits)
	runTime := time.Since(runStart)

	if err != nil {
		result.Status = "IE"
		result.Feedback = err.Error()
		return result, nil
	}

	result.Time = runTime.Seconds()
	result.Memory = float64(runResult.Memory) / 1024.0 // Convert to KB

	// Check for various error conditions
	switch runResult.Status {
	case executors.StatusRuntimeError:
		result.Status = "RTE"
		result.Feedback = runResult.Error
		result.Points = 0
		return result, nil
	case executors.StatusTimeLimitExceeded:
		result.Status = "TLE"
		result.Points = 0
		return result, nil
	case executors.StatusMemoryLimitExceeded:
		result.Status = "MLE"
		result.Points = 0
		return result, nil
	case executors.StatusOutputLimitExceeded:
		result.Status = "OLE"
		result.Points = 0
		return result, nil
	}

	// Check output
	expectedData, err := os.ReadFile(tc.OutputPath)
	if err != nil {
		result.Status = "IE"
		result.Feedback = "Failed to read expected output"
		return result, nil
	}

	checkResult := w.checkOutput(string(expectedData), string(runResult.Output), tc.Checker)
	result.Status = checkResult.Status
	result.Points = checkResult.Points
	result.Feedback = checkResult.Feedback

	return result, nil
}

// checkOutput compares output with expected.
func (w *JudgeWorker) checkOutput(expected, output, checkerType string) *checkResult {
	switch checkerType {
	case "standard", "":
		return w.checkStandard(expected, output)
	case "custom":
		// TODO: Implement custom checker
		return w.checkStandard(expected, output)
	default:
		return w.checkStandard(expected, output)
	}
}

// checkStandard performs standard whitespace-insensitive comparison.
func (w *JudgeWorker) checkStandard(expected, output string) *checkResult {
	// Normalize whitespace
	normExpected := normalizeWhitespace(expected)
	normOutput := normalizeWhitespace(output)

	if normExpected == normOutput {
		return &checkResult{
			Status: "AC",
			Points: 1.0,
		}
	}

	return &checkResult{
		Status:   "WA",
		Points:   0,
		Feedback: "Output mismatch",
	}
}

// RequestAbort signals the worker to stop grading.
func (w *JudgeWorker) RequestAbort() {
	select {
	case <-w.abort:
		return
	default:
		close(w.abort)
		w.cancel()
	}
}

// getProblemDir returns the path to a problem directory.
func (w *JudgeWorker) getProblemDir(problemID string) string {
	problemGlobs := []string{
		"/problems/*/" + problemID,
		"/problems/" + problemID,
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
	Position  int
	InputPath string
	OutputPath string
	Points    float64
	Checker   string
}

// loadTestCases loads test cases for a problem.
func (w *JudgeWorker) loadTestCases(problemDir string) ([]TestCase, error) {
	var testCases []TestCase

	// Look for test case directories
	testsDir := filepath.Join(problemDir, "tests")
	if _, err := os.Stat(testsDir); os.IsNotExist(err) {
		// Try alternative test directory locations
		testsDir = filepath.Join(problemDir, "test_cases")
		if _, err := os.Stat(testsDir); os.IsNotExist(err) {
			return nil, fmt.Errorf("no tests directory found")
		}
	}

	// Find all .in files
	for i := 1; i <= 100; i++ {
		inputPath := filepath.Join(testsDir, fmt.Sprintf("%d.in", i))
		outputPath := filepath.Join(testsDir, fmt.Sprintf("%d.out", i))

		if _, err := os.Stat(inputPath); os.IsNotExist(err) {
			// Also try .dat extension
			inputPath = filepath.Join(testsDir, fmt.Sprintf("%d.dat", i))
			if _, err := os.Stat(inputPath); os.IsNotExist(err) {
				continue
			}
		}

		if _, err := os.Stat(outputPath); os.IsNotExist(err) {
			outputPath = filepath.Join(testsDir, fmt.Sprintf("%d.ans", i))
		}

		testCases = append(testCases, TestCase{
			Position:   len(testCases),
			InputPath:  inputPath,
			OutputPath: outputPath,
			Points:     10.0,
			Checker:    "standard",
		})
	}

	return testCases, nil
}

// checkResult represents the result of output checking.
type checkResult struct {
	Status   string
	Points   float64
	Feedback string
}

// normalizeWhitespace normalizes whitespace for comparison.
func normalizeWhitespace(s string) string {
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
