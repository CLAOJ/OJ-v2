package jobs

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/hibiken/asynq"
)

// Task names
const (
	TypeJudgeSubmission = "judge_submission"
)

// BridgeRouter is an interface we inject so `jobs` doesn't strictly depend on `bridge.Server`
// but can still call `Submit(subID)`.
type BridgeRouter interface {
	Submit(subID uint) error
}

var globalBridge BridgeRouter

// SetBridge lets main.go inject the bridge router for the worker to use.
func SetBridge(b BridgeRouter) {
	globalBridge = b
}

// JudgeSubmissionPayload contains the info for a submission-request job.
type JudgeSubmissionPayload struct {
	SubmissionID uint `json:"submission_id"`
}

// EnqueueJudgeSubmission puts a submission into the queue for grading.
func EnqueueJudgeSubmission(subID uint) error {
	payload, err := json.Marshal(JudgeSubmissionPayload{SubmissionID: subID})
	if err != nil {
		return err
	}

	task := asynq.NewTask(TypeJudgeSubmission, payload, asynq.MaxRetry(3), asynq.Timeout(5*time.Minute))
	info, err := Client.Enqueue(task, asynq.Queue("critical"))
	if err != nil {
		return err
	}

	log.Printf("jobs: enqueued judge task %s for sub %d", info.ID, subID)
	return nil
}

// HandleJudgeSubmissionTask is the worker function that actually runs the task.
func HandleJudgeSubmissionTask(ctx context.Context, t *asynq.Task) error {
	var payload JudgeSubmissionPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return err
	}

	if globalBridge == nil {
		log.Printf("jobs: globalBridge not set, ignoring judge task for sub %d", payload.SubmissionID)
		return nil
	}

	return globalBridge.Submit(payload.SubmissionID)
}
