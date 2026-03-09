package jobs

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/CLAOJ/claoj/scoring"
	"github.com/hibiken/asynq"
)

const (
	TypeRescoreProblem = "rescore_problem"
)

// RescorePayload contains the problem ID to rescore.
type RescorePayload struct {
	ProblemID uint `json:"problem_id"`
}

// EnqueueRescore queues a background task to recalculate problem points across users.
func EnqueueRescore(problemID uint) error {
	payload, err := json.Marshal(RescorePayload{ProblemID: problemID})
	if err != nil {
		return err
	}

	task := asynq.NewTask(TypeRescoreProblem, payload, asynq.MaxRetry(2), asynq.Timeout(10*time.Minute))
	info, err := Client.Enqueue(task, asynq.Queue("default"))
	if err != nil {
		return err
	}

	log.Printf("jobs: enqueued rescore task %s for problem %d", info.ID, problemID)
	return nil
}

// HandleRescoreTask processes the score recalculation.
func HandleRescoreTask(ctx context.Context, t *asynq.Task) error {
	var payload RescorePayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return err
	}

	database := db.DB
	log.Printf("jobs: running rescore for problem %d", payload.ProblemID)

	// 1. Find all users who have submissions for this problem
	var userIDs []uint
	if err := database.Model(&models.Submission{}).Where("problem_id = ?", payload.ProblemID).Distinct().Pluck("user_id", &userIDs).Error; err != nil {
		return err
	}

	// 2. Recalculate points for each user
	for _, uid := range userIDs {
		if err := scoring.CalculateProfilePoints(database, uid); err != nil {
			log.Printf("jobs: failed to recalculate points for user %d: %v", uid, err)
			continue
		}
	}

	// 3. Recalculate affected organizations
	if len(userIDs) > 0 {
		var orgIDs []uint
		database.Table("judge_profile_organizations").Where("profile_id IN ?", userIDs).Distinct().Pluck("organization_id", &orgIDs)
		for _, oid := range orgIDs {
			if err := scoring.CalculateOrganizationPoints(database, oid); err != nil {
				log.Printf("jobs: failed to recalculate points for org %d: %v", oid, err)
			}
		}
	}

	return nil
}
