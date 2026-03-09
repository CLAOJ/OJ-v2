// Package core implements the main judge coordination logic.
package core

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/CLAOJ/claoj-judge/config"
	"github.com/CLAOJ/claoj-judge/executors"
	"github.com/CLAOJ/claoj-judge/protocol"
)

// Judge coordinates submission grading.
type Judge struct {
	mu       sync.RWMutex
	cfg      *config.Config
	pm       *protocol.PacketManager
	executor map[string]executors.Executor
	workers  map[uint]*JudgeWorker

	currentSubmission *Submission
	gradingLock       sync.Mutex

	problemMtimes map[string]float64
	updaterExit   chan struct{}
	updaterSignal chan struct{}

	running bool
}

// Submission represents a grading request.
type Submission struct {
	ID           uint
	ProblemID    string
	StorageNamespace *string
	Language     string
	Source       string
	TimeLimit    time.Duration
	MemoryLimit  int64
	ShortCircuit bool
	Meta         map[string]interface{}
}

// NewJudge creates a new judge instance.
func NewJudge(cfg *config.Config) (*Judge, error) {
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
	execList := []executors.Executor{
		executors.NewCPP17Executor(),
		executors.NewCPP20Executor(),
		executors.NewC11Executor(),
		executors.NewPython3Executor(),
		executors.NewPython2Executor(),
		executors.NewJava8Executor(),
		executors.NewGoExecutor(),
		executors.NewNodeJSExecutor(),
		executors.NewRustExecutor(),
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
				log.Printf("Found problem: %s", problemID)
			}
		}
	}

	return nil
}

// Run starts the judge's main loop.
func (j *Judge) Run() error {
	// Create packet manager and connect to backend
	pm, err := protocol.NewPacketManager(j.cfg)
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}
	j.pm = pm

	// Perform handshake
	problems := j.GetSupportedProblems()
	runtimes := j.GetRuntimeVersions()

	if err := pm.Handshake(problems, runtimes); err != nil {
		return fmt.Errorf("handshake failed: %w", err)
	}

	log.Printf("Handshake successful, waiting for submissions...")

	j.running = true

	// Start updater goroutine
	go j.updaterThread()

	// Start problem watcher if enabled
	if !j.cfg.NoWatchdog {
		go j.watchProblems()
	}

	// Run packet reading loop (this blocks)
	return pm.Run(j)
}

// updaterThread periodically sends problem updates to the server.
func (j *Judge) updaterThread() {
	for {
		select {
		case <-j.updaterSignal:
			time.Sleep(3 * time.Second)
			j.updateProblems()
		case <-j.updaterExit:
			return
		}
	}
}

// updateProblems sends the current problem list to the server.
func (j *Judge) updateProblems() {
	j.scanProblems()
	j.pm.SendSupportedProblems(j.problemMtimes)
	log.Printf("Updated problem list: %d problems", len(j.problemMtimes))
}

// watchProblems watches problem directories for changes.
func (j *Judge) watchProblems() {
	// Simple polling-based watcher
	// TODO: Implement proper fsnotify-based watching
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	lastMtimes := make(map[string]float64)
	for k, v := range j.problemMtimes {
		lastMtimes[k] = v
	}

	for range ticker.C {
		if !j.running {
			return
		}

		j.scanProblems()

		// Check for changes
		changed := false
		if len(j.problemMtimes) != len(lastMtimes) {
			changed = true
		} else {
			for k, v := range j.problemMtimes {
				if v != lastMtimes[k] {
					changed = true
					break
				}
			}
		}

		if changed {
			log.Printf("Problem changes detected!")
			j.updaterSignal <- struct{}{}

			// Update lastMtimes
			lastMtimes = make(map[string]float64)
			for k, v := range j.problemMtimes {
				lastMtimes[k] = v
			}
		}
	}
}

// HandlePacket handles incoming packets from the server.
func (j *Judge) HandlePacket(packet protocol.Packet) error {
	name, _ := packet["name"].(string)

	switch name {
	case "ping":
		when, _ := packet["when"].(float64)
		j.pm.SendPingResponse(when, j.GetLoad())
		return nil

	case "submission-request":
		return j.handleSubmissionRequest(packet)

	case "terminate-submission":
		subID, _ := packet["submission-id"].(float64)
		return j.AbortGrading(uint(subID))

	case "disconnect":
		log.Println("Received disconnect request, shutting down...")
		j.Shutdown()
		os.Exit(0)
		return nil

	default:
		log.Printf("Unknown packet: %s", name)
		return nil
	}
}

// handleSubmissionRequest processes a submission request.
func (j *Judge) handleSubmissionRequest(packet protocol.Packet) error {
	subID, _ := packet["submission-id"].(float64)
	problemID, _ := packet["problem-id"].(string)
	language, _ := packet["language"].(string)
	source, _ := packet["source"].(string)
	timeLimit, _ := packet["time-limit"].(float64)
	memoryLimit, _ := packet["memory-limit"].(float64)
	shortCircuit, _ := packet["short-circuit"].(bool)

	var storageNS *string
	if ns, ok := packet["storage-namespace"].(string); ok {
		storageNS = &ns
	}

	sub := &Submission{
		ID:               uint(subID),
		ProblemID:        problemID,
		StorageNamespace: storageNS,
		Language:         language,
		Source:           source,
		TimeLimit:        time.Duration(timeLimit * float64(time.Second)),
		MemoryLimit:      int64(memoryLimit),
		ShortCircuit:     shortCircuit,
		Meta:             make(map[string]interface{}),
	}

	// Acknowledge submission
	j.pm.SendSubmissionAcknowledged(sub.ID)

	// Start grading
	return j.BeginGrading(sub)
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
	j.pm.SendGradingBegin(sub.ID, false)

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
	j.mu.RLock()
	defer j.mu.RUnlock()

	if len(j.workers) > 0 {
		return 1.0
	}
	return 0.0
}

// Shutdown gracefully shuts down the judge.
func (j *Judge) Shutdown() {
	j.running = false
	close(j.updaterExit)
	if j.pm != nil {
		j.pm.Close()
	}
	log.Println("Judge shut down complete")
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

// IsRunning returns whether the judge is running.
func (j *Judge) IsRunning() bool {
	return j.running
}
