package jobs

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/CLAOJ/claoj/cache"
	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
)

// judgeBeatTTL is how long a judge stays online after its last packet. Agents
// send judge-status every 30 s, so three missed beats mark it offline.
const judgeBeatTTL = 90 * time.Second

func judgeBeatKey(id uint) string { return fmt.Sprintf("judge:beat:%d", id) }

// MarkJudgeAlive records a packet from judge id: a Redis beat (shared by every
// backend replica) and online=1, with start_time when it was offline. Each
// agent signs in with its own judge_judge row, so every v2 judge shows up
// under its own name. load is the agent's (judge-status), never set here.
func MarkJudgeAlive(id uint) error {
	if err := cache.Client.Set(cache.Ctx, judgeBeatKey(id), 1, judgeBeatTTL).Err(); err != nil {
		return err
	}
	return db.DB.Model(&models.Judge{}).Where("id = ? AND online = ?", id, false).
		Updates(map[string]interface{}{"online": true, "start_time": time.Now()}).Error
}

// SyncJudgeStatus marks offline the v2 judges (names; v1's rows belong to v1's
// bridge) whose beat expired. Idempotent, so every backend replica may run it.
func SyncJudgeStatus(names []string) error {
	var online []models.Judge
	if err := db.DB.Select("id").Where("name IN ? AND online = ?", names, true).Find(&online).Error; err != nil {
		return err
	}
	for _, j := range online {
		n, err := cache.Client.Exists(cache.Ctx, judgeBeatKey(j.ID)).Result()
		if err != nil {
			return err
		}
		if n == 0 {
			if err := db.DB.Model(&models.Judge{}).Where("id = ?", j.ID).Update("online", false).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

// RunJudgeStatusLoop calls SyncJudgeStatus now and then every interval
// until ctx is done.
func RunJudgeStatusLoop(ctx context.Context, names []string, every time.Duration) {
	ticker := time.NewTicker(every)
	defer ticker.Stop()
	for {
		if err := SyncJudgeStatus(names); err != nil {
			log.Printf("jobs: judge status sync: %v", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
