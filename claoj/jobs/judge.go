package jobs

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/CLAOJ/claoj/config"
	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/hibiken/asynq"
)

const (
	TypeJudgeSubmission = "judge_submission"
	// QueueJudge is consumed only by judge agents (claoj-judge), never by
	// the backend's own asynq worker.
	QueueJudge    = "judge"
	judgeMaxRetry = 2
)

// enqueueFunc and the inspector funcs below are seams overridden in tests.
var enqueueFunc = defaultEnqueue

var (
	getTaskInfo      = func(queue, id string) (*asynq.TaskInfo, error) { return Inspector.GetTaskInfo(queue, id) }
	deleteTask       = func(queue, id string) error { return Inspector.DeleteTask(queue, id) }
	cancelProcessing = func(id string) error { return Inspector.CancelProcessing(id) }
)

func defaultEnqueue(task *asynq.Task, opts ...asynq.Option) (*asynq.TaskInfo, error) {
	return Client.Enqueue(task, opts...)
}

// SetMockEnqueue sets a mock enqueue function for testing.
func SetMockEnqueue(fn func(*asynq.Task, ...asynq.Option) (*asynq.TaskInfo, error)) {
	enqueueFunc = fn
}

func judgeTaskID(subID uint) string { return fmt.Sprintf("sub-%d", subID) }

// BuildSubmissionRequest builds the DMOJ "submission-request" packet a judge
// grades from. Ported from the old bridge.Server.Submit.
func BuildSubmissionRequest(subID uint) (map[string]interface{}, error) {
	var sub models.Submission
	if err := db.DB.Preload("Problem").Preload("Language").First(&sub, subID).Error; err != nil {
		return nil, fmt.Errorf("submission %d not found: %w", subID, err)
	}

	var source models.SubmissionSource
	if err := db.DB.Where("submission_id = ?", subID).First(&source).Error; err != nil && !sub.Language.FileOnly {
		return nil, fmt.Errorf("submission %d source missing: %w", subID, err)
	}

	timeLimit, memLimit := sub.Problem.TimeLimit, sub.Problem.MemoryLimit
	var override models.LanguageLimit
	if err := db.DB.Where("problem_id = ? AND language_id = ?", sub.ProblemID, sub.LanguageID).First(&override).Error; err == nil {
		timeLimit, memLimit = override.TimeLimit, uint(override.MemoryLimit)
	}

	src := source.Source
	if sub.Language.FileOnly {
		// e.g. /media/submissions/file.zip -> https://site/media/...
		src = config.C.App.SiteFullURL + src
	}

	return map[string]interface{}{
		"name":          "submission-request",
		"submission-id": subID,
		"problem-id":    sub.Problem.Code,
		"language":      sub.Language.Key,
		"source":        src,
		"time-limit":    timeLimit,
		"memory-limit":  memLimit,
		"short-circuit": sub.Problem.ShortCircuit,
		"meta": map[string]interface{}{
			"pretests-only":   sub.IsPretested,
			"in-contest":      0, // stub, as before
			"attempt-no":      1, // stub, as before
			"user":            sub.UserID,
			"file-only":       sub.Language.FileOnly,
			"file-size-limit": sub.Language.FileSizeLimit,
		},
	}, nil
}

// EnqueueJudgeSubmission queues a submission for any judge agent to grade.
// The task ID makes it idempotent: re-enqueueing a submission that is
// still queued or being graded is a no-op.
func EnqueueJudgeSubmission(subID uint) error {
	pkt, err := BuildSubmissionRequest(subID)
	if err != nil {
		return err
	}
	payload, err := json.Marshal(pkt)
	if err != nil {
		return err
	}

	task := asynq.NewTask(TypeJudgeSubmission, payload)
	id := judgeTaskID(subID)
	opts := []asynq.Option{
		asynq.Queue(QueueJudge),
		asynq.TaskID(id),
		asynq.MaxRetry(judgeMaxRetry),
		asynq.Timeout(30 * time.Minute),
	}

	_, err = enqueueFunc(task, opts...)
	if errors.Is(err, asynq.ErrTaskIDConflict) {
		info, ierr := getTaskInfo(QueueJudge, id)
		switch {
		case errors.Is(ierr, asynq.ErrTaskNotFound):
			// finished between the two calls: just enqueue again
		case ierr != nil:
			return ierr
		case info.State != asynq.TaskStateArchived && info.State != asynq.TaskStateCompleted:
			return nil // already queued or being graded
		default:
			if err := deleteTask(QueueJudge, id); err != nil {
				return err
			}
		}
		_, err = enqueueFunc(task, opts...)
	}
	if err != nil {
		return err
	}

	log.Printf("jobs: queued judge task %s", id)
	return nil
}

// AbortJudgeSubmission drops a queued judge task, or cancels the one a
// judge is grading (the agent then sends terminate-submission to tiervnoj).
func AbortJudgeSubmission(subID uint) error {
	id := judgeTaskID(subID)
	info, err := getTaskInfo(QueueJudge, id)
	if errors.Is(err, asynq.ErrTaskNotFound) || errors.Is(err, asynq.ErrQueueNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if info.State == asynq.TaskStateActive {
		return cancelProcessing(id)
	}
	return deleteTask(QueueJudge, id)
}

// JudgeAborter satisfies submission.BridgeServer / v2.SetBridgeServer.
type JudgeAborter struct{}

func (JudgeAborter) Abort(subID uint) error { return AbortJudgeSubmission(subID) }
