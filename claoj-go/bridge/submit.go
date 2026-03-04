package bridge

import (
	"errors"
	"fmt"
	"log"

	"github.com/CLAOJ/claoj-go/config"
	"github.com/CLAOJ/claoj-go/db"
	"github.com/CLAOJ/claoj-go/models"
)

// Submit sends a submission grading request to the appropriate judge.
// Ported from judge_handler.py's submit() method.
func (s *Server) Submit(subID uint) error {
	var sub models.Submission
	if err := db.DB.Preload("Problem").Preload("Language").Preload("User").Where("id = ?", subID).First(&sub).Error; err != nil {
		return fmt.Errorf("submission not found: %d", subID)
	}

	var source models.SubmissionSource
	if err := db.DB.Where("submission_id = ?", subID).First(&source).Error; err != nil && !sub.Language.FileOnly {
		return fmt.Errorf("submission source missing: %d", subID)
	}

	// 1. Find an available judge
	s.manager.RLock()
	var selected *Handler
	for _, handler := range s.manager.judges {
		if handler.problems[sub.Problem.Code] && handler.executors[sub.Language.Key] != nil && !handler.working {
			selected = handler
			break
		}
	}
	s.manager.RUnlock()

	if selected == nil {
		return errors.New("no available judge for this problem/language")
	}

	// 2. Fetch Time/Memory limits (LanguageLimit overrides)
	timeLimit := sub.Problem.TimeLimit
	memLimit := sub.Problem.MemoryLimit
	var override models.LanguageLimit
	if err := db.DB.Where("problem_id = ? AND language_id = ?", sub.ProblemID, sub.LanguageID).First(&override).Error; err == nil {
		timeLimit = override.TimeLimit
		memLimit = uint(override.MemoryLimit)
	}

	// 3. Mark as working
	selected.working = true
	selected.workingSub = subID

	// 4. Construct source payload (File_only = URLs)
	sourcePayload := source.Source
	if sub.Language.FileOnly {
		// e.g. /media/submissions/file.zip -> https://site.com/media/...
		sourcePayload = config.C.App.SiteFullURL + sourcePayload
	}

	// 5. Send packet
	pkt := Packet{
		"name":          "submission-request",
		"submission-id": subID,
		"problem-id":    sub.Problem.Code,
		"language":      sub.Language.Key,
		"source":        sourcePayload,
		"time-limit":    timeLimit,
		"memory-limit":  memLimit,
		"short-circuit": sub.Problem.ShortCircuit,
		"meta": map[string]interface{}{
			"pretests-only":   sub.IsPretested,
			"in-contest":      0, // stub, needs contest participation lookup
			"attempt-no":      1, // stub
			"user":            sub.UserID,
			"file-only":       sub.Language.FileOnly,
			"file-size-limit": sub.Language.FileSizeLimit,
		},
	}

	log.Printf("bridge: dispatching sub %d to %s", subID, selected.name)

	if err := selected.conn.WritePacket(pkt); err != nil {
		selected.cleanup()
		return fmt.Errorf("failed to write to judge: %w", err)
	}

	return nil
}

// GetManager exposes the manager for manual testing if needed
func (s *Server) GetManager() *Manager {
	return s.manager
}
