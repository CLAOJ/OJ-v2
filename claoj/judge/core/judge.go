// Package core implements the main judge coordination logic.
package core

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/CLAOJ/claoj/judge/executors"
	"github.com/CLAOJ/claoj/judge/protocol"
)

// Judge coordinates submission grading.
type Judge struct {
	mu       sync.RWMutex
	cfg      *Config
	pm       *protocol.PacketManager
	executor map[string]executors.Executor
	workers  map[uint]*JudgeWorker // Active workers by submission ID

	currentSubmission *Submission
	gradingLock       sync.Mutex

	problemMtimes map[string]float64 // Problem modification times
	updaterExit   chan struct{}
	updaterSignal chan struct{}
}

// Config holds judge configuration.
type Config struct {
	ServerHost   string
	ServerPort   int
	JudgeName    string
	JudgeKey     string
	APIHost      string
	APIPort      int
	LogFile      string
	NoWatchdog   bool
	SkipSelfTest bool
	ProblemGlobs []string
	TempDir      string
	Runtime      map[string]interface{} // Runtime-specific config
}

// Submission represents a grading request.
type Submission struct {
	ID           uint
	ProblemID    string
	Language     string
	Source       string
	TimeLimit    time.Duration
	MemoryLimit  int64
	ShortCircuit bool
	Meta         map[string]interface{}
}

// DefaultConfig returns default configuration.
func DefaultConfig() *Config {
	return &Config{
		ServerPort: 9999,
		APIHost:    "127.0.0.1",
		TempDir:    os.TempDir(),
		ProblemGlobs: []string{
			"/problems/**/",
		},
		Runtime: make(map[string]interface{}),
	}
}

// LoadConfig loads configuration from a YAML file.
func LoadConfig(path string) (*Config, error) {
	// Expand ~ to home directory
	if len(path) > 0 && path[0] == '~' {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		path = filepath.Join(home, path[1:])
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// NewJudge creates a new judge instance.
func NewJudge(cfg *Config) (*Judge, error) {
	j := &Judge{
		cfg:           cfg,
		executor:      make(map[string]executors.Executor),
		workers:       make(map[uint]*JudgeWorker),
		problemMtimes: make(map[string]float64),
		updaterExit:   make(chan struct{}),
		updaterSignal: make(chan struct{}, 1),
	}

	// Initialize executors
	if err := j.loadExecutors(); err != nil {
		return nil, fmt.Errorf("failed to load executors: %w", err)
	}

	// Scan problems
	if err := j.scanProblems(); err != nil {
		log.Printf("Warning: Problem scan failed: %v", err)
	}

	return j, nil
}

// loadExecutors initializes all language executors.
func (j *Judge) loadExecutors() error {
	// Register all supported executors
	execList := []executors.Executor{
		executors.NewCPP17Executor(),
		executors.NewCPP20Executor(),
		executors.NewC11Executor(),
		executors.NewPython3Executor(),
		executors.NewPython2Executor(),
		executors.NewJava8Executor(),
		executors.NewJava11Executor(),
		executors.NewGoExecutor(),
		executors.NewRustExecutor(),
		executors.NewNodeJSExecutor(),
		// Add more executors as they are implemented
	}

	for _, exec := range execList {
		j.executor[exec.Language()] = exec
		log.Printf("Loaded executor: %s", exec.Language())
	}

	return nil
}

// scanProblems scans problem directories for available problems.
func (j *Judge) scanProblems() error {
	for _, glob := range j.cfg.ProblemGlobs {
		matches, err := filepath.Glob(glob)
		if err != nil {
			return err
		}

		for _, match := range matches {
			initPath := filepath.Join(match, "init.yml")
			if _, err := os.Stat(initPath); err == nil {
				info, err := os.Stat(match)
				if err != nil {
					continue
				}
				problemID := filepath.Base(match)
				j.problemMtimes[problemID] = float64(info.ModTime().Unix())
				log.Printf("Found problem: %s (mtime: %v)", problemID, info.ModTime())
			}
		}
	}

	return nil
}

// SetPacketManager sets the packet manager for network communication.
func (j *Judge) SetPacketManager(pm *protocol.PacketManager) {
	j.pm = pm
}

// Listen starts the judge's main loop.
func (j *Judge) Listen() error {
	if j.pm == nil {
		return fmt.Errorf("packet manager not set")
	}

	// Send supported problems to server
	j.updateProblems()

	// Start updater thread
	go j.updaterThread()

	// Start packet reading loop
	return j.pm.Run(j)
}

// updaterThread periodically sends problem updates to the server.
func (j *Judge) updaterThread() {
	for {
		select {
		case <-j.updaterSignal:
			j.updateProblems()
			time.Sleep(3 * time.Second)
		case <-j.updaterExit:
			return
		}
	}
}

// updateProblems sends the current problem list to the server.
func (j *Judge) updateProblems() {
	j.scanProblems() // Re-scan for changes
	j.pm.SendSupportedProblems(j.problemMtimes)
}

// BeginGrading starts grading a submission.
func (j *Judge) BeginGrading(sub *Submission) error {
	j.gradingLock.Lock()
	defer j.gradingLock.Unlock()

	if j.currentSubmission != nil {
		return fmt.Errorf("already grading submission %d", j.currentSubmission.ID)
	}

	j.currentSubmission = sub
	log.Printf("Starting grading: sub=%d problem=%s language=%s",
		sub.ID, sub.ProblemID, sub.Language)

	// Get executor for this language
	exec, ok := j.executor[sub.Language]
	if !ok {
		return fmt.Errorf("unknown language: %s", sub.Language)
	}

	// Create worker for this submission
	worker := NewJudgeWorker(sub, exec, j.pm)
	j.workers[sub.ID] = worker

	// Send grading begin packet
	j.pm.SendGradingBegin(sub.ID)

	// Run grading in goroutine
	go func() {
		result, err := worker.Grade()
		if err != nil {
			log.Printf("Grading error: %v", err)
			j.pm.SendInternalError(sub.ID, err.Error())
		} else {
			j.pm.SendGradingEnd(sub.ID, result)
		}

		// Cleanup
		j.gradingLock.Lock()
		j.currentSubmission = nil
		delete(j.workers, sub.ID)
		j.gradingLock.Unlock()
	}()

	return nil
}

// AbortGrading aborts the current grading.
func (j *Judge) AbortGrading(submissionID uint) error {
	j.gradingLock.Lock()
	defer j.gradingLock.Unlock()

	if j.currentSubmission == nil {
		log.Printf("Nothing to abort")
		return nil
	}

	if j.currentSubmission.ID != submissionID {
		log.Printf("Cannot abort %d, currently grading %d",
			submissionID, j.currentSubmission.ID)
		return nil
	}

	log.Printf("Aborting grading: sub=%d", submissionID)

	if worker, ok := j.workers[submissionID]; ok {
		worker.RequestAbort()
	}

	return nil
}

// GetLoad returns the current judge load (0.0 - 1.0).
func (j *Judge) GetLoad() float64 {
	// Simple load calculation based on number of active workers
	j.mu.RLock()
	defer j.mu.RUnlock()

	// For now, return 0 if idle, 1 if busy
	if len(j.workers) > 0 {
		return 1.0
	}
	return 0.0
}

// Shutdown gracefully shuts down the judge.
func (j *Judge) Shutdown() error {
	close(j.updaterExit)
	j.pm.Close()
	return nil
}

// GetSupportedProblems returns the list of supported problems.
func (j *Judge) GetSupportedProblems() map[string]float64 {
	return j.problemMtimes
}

// GetRuntimeVersions returns version info for all executors.
func (j *Judge) GetRuntimeVersions() map[string][]string {
	versions := make(map[string][]string)
	for lang, exec := range j.executor {
		versions[lang] = exec.RuntimeVersions()
	}
	return versions
}
