package jobs

import (
	"context"
	"log"
	"time"

	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/hibiken/asynq"
)

// listServers is a seam overridden in tests.
var listServers = func() ([]*asynq.ServerInfo, error) { return Inspector.Servers() }

// SyncJudgeStatus mirrors asynq's worker heartbeats into the shared v2
// judge_judge row: online if any server consumes queue "judge", load =
// busy slots / total slots, start_time = oldest such server. Every backend
// replica may run it; the writes are idempotent.
func SyncJudgeStatus(judgeName string) error {
	servers, err := listServers()
	if err != nil {
		return err
	}

	var judges, busy, slots int
	var started time.Time
	for _, s := range servers {
		if _, ok := s.Queues[QueueJudge]; !ok {
			continue
		}
		judges++
		busy += len(s.ActiveWorkers)
		slots += s.Concurrency
		if started.IsZero() || s.Started.Before(started) {
			started = s.Started
		}
	}

	updates := map[string]interface{}{"online": judges > 0}
	if judges > 0 && slots > 0 {
		updates["load"] = float64(busy) / float64(slots)
		updates["start_time"] = started
	}
	return db.DB.Model(&models.Judge{}).Where("name = ?", judgeName).Updates(updates).Error
}

// RunJudgeStatusLoop calls SyncJudgeStatus now and then every interval
// until ctx is done.
func RunJudgeStatusLoop(ctx context.Context, judgeName string, every time.Duration) {
	ticker := time.NewTicker(every)
	defer ticker.Stop()
	for {
		if err := SyncJudgeStatus(judgeName); err != nil {
			log.Printf("jobs: judge status sync: %v", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
