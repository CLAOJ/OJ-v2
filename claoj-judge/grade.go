package main

import (
	"context"
	"encoding/json"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/hibiken/asynq"
)

// terminal packets end a submission on tiervnoj's side.
var terminal = map[string]bool{
	"grading-end":           true,
	"compile-error":         true,
	"internal-error":        true,
	"submission-terminated": true,
}

// Agent grades one asynq task at a time on the local tiervnoj child.
type Agent struct {
	child     *Child
	post      func(Packet) error
	stopping  atomic.Bool   // set on SIGTERM: asynq re-queues the task, so don't abort it
	abortWait time.Duration // how long to wait for tiervnoj to confirm an abort
}

// ProcessTask implements asynq.Handler.
func (a *Agent) ProcessTask(ctx context.Context, t *asynq.Task) error {
	var req Packet
	if err := json.Unmarshal(t.Payload(), &req); err != nil {
		return fmt.Errorf("bad judge task payload: %v: %w", err, asynq.SkipRetry)
	}
	retried, _ := asynq.GetRetryCount(ctx)
	maxRetry, _ := asynq.GetMaxRetry(ctx)
	// A failed attempt is not necessarily a crash: asynq counts every handler
	// error and lease expiry, so SIGTERM mid-grade, an errNoChild reconnect
	// gap and a >5min backend outage all land here too. On the last attempt
	// the retry budget is exhausted — report IE instead of retrying forever.
	return a.grade(ctx, req, retried > 0 && retried >= maxRetry)
}

func (a *Agent) grade(ctx context.Context, req Packet, giveUp bool) error {
	id := req.SubmissionID()
	if giveUp {
		return a.post(Packet{"name": "internal-error", "submission-id": id, "message": "judge task failed after exhausting its retries"})
	}
	if err := a.child.Send(req); err != nil {
		return err
	}

	done := ctx.Done()
	var abortTimeout <-chan time.Time
	for {
		select {
		case p := <-a.child.Events():
			if p.SubmissionID() != id {
				continue // late packet from an earlier submission
			}
			if err := a.post(p); err != nil {
				return err
			}
			if terminal[p.Name()] {
				return nil
			}
		case <-done:
			if a.stopping.Load() {
				return ctx.Err() // shutting down: asynq re-queues for another judge
			}
			done = nil // aborted via CancelProcessing (or task timeout)
			if err := a.child.Send(Packet{"name": "terminate-submission"}); err != nil {
				return a.post(Packet{"name": "submission-terminated", "submission-id": id})
			}
			abortTimeout = time.After(a.abortWait)
		case <-abortTimeout:
			// tiervnoj never confirmed; record the abort ourselves
			return a.post(Packet{"name": "submission-terminated", "submission-id": id})
		}
	}
}
