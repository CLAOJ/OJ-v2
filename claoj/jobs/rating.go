package jobs

import (
	"context"
	"encoding/json"
	"log"
	"math"
	"time"

	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/CLAOJ/claoj/scoring"
	"github.com/hibiken/asynq"
	"gorm.io/gorm"
)

const (
	TypeRateContest = "rate_contest"
)

// RatingPayload contains the contest ID to rate.
type RatingPayload struct {
	ContestID uint `json:"contest_id"`
}

// EnqueueRating queues a background task to calculate ratings after a contest ends.
func EnqueueRating(contestID uint) error {
	payload, err := json.Marshal(RatingPayload{ContestID: contestID})
	if err != nil {
		return err
	}

	task := asynq.NewTask(TypeRateContest, payload, asynq.MaxRetry(3), asynq.Timeout(30*time.Minute))
	info, err := Client.Enqueue(task, asynq.Queue("default"))
	if err != nil {
		return err
	}

	log.Printf("jobs: enqueued rating task %s for contest %d", info.ID, contestID)
	return nil
}

// HandleRateContestTask processes the rating calculation.
func HandleRateContestTask(ctx context.Context, t *asynq.Task) error {
	var payload RatingPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return err
	}

	var contest models.Contest
	if err := db.DB.First(&contest, payload.ContestID).Error; err != nil {
		return err
	}

	log.Printf("jobs: rating contest %s (%d)", contest.Key, contest.ID)

	// 1. Fetch participants
	var participations []models.ContestParticipation
	query := db.DB.Where("contest_id = ? AND virtual = 0 AND is_disqualified = 0", contest.ID).
		Order("score DESC, cumtime ASC, tiebreaker ASC")

	if !contest.RateAll {
		query = query.Where("EXISTS (SELECT 1 FROM judge_submission s WHERE s.user_id = judge_contestparticipation.user_id AND s.contest_id = ?)", contest.ID)
	}

	if contest.RatingFloor != nil {
		query = query.Where("EXISTS (SELECT 1 FROM judge_profile p WHERE p.id = judge_contestparticipation.user_id AND (p.rating >= ? OR p.rating IS NULL))", *contest.RatingFloor)
	}
	if contest.RatingCeiling != nil {
		query = query.Where("EXISTS (SELECT 1 FROM judge_profile p WHERE p.id = judge_contestparticipation.user_id AND (p.rating <= ? OR p.rating IS NULL))", *contest.RatingCeiling)
	}

	if err := query.Find(&participations).Error; err != nil {
		return err
	}

	if len(participations) == 0 {
		log.Printf("jobs: no eligible participants for contest %d", contest.ID)
		return nil
	}

	n := len(participations)
	userIDs := make([]uint, n)
	ranking := make([]float64, n)
	oldMeans := make([]float64, n)
	timesRanked := make([]int, n)

	// Calculate ranking (averaging ties)
	rank := 0.0
	delta := 0
	for i := 0; i < n; i++ {
		if i > 0 && (participations[i].Score != participations[i-1].Score ||
			participations[i].Cumtime != participations[i-1].Cumtime ||
			participations[i].Tiebreaker != participations[i-1].Tiebreaker) {
			for j := i - delta; j < i; j++ {
				ranking[j] = rank + float64(delta-1)/2.0
			}
			rank += float64(delta)
			delta = 0
		}
		delta++
		userIDs[i] = participations[i].UserID
	}
	for j := n - delta; j < n; j++ {
		ranking[j] = rank + float64(delta-1)/2.0
	}

	// 2. Fetch historical data
	historicalPs := make([][]float64, n)
	for i, uid := range userIDs {
		var lastRating models.Rating
		err := db.DB.Where("user_id = ?", uid).Order("last_rated DESC").First(&lastRating).Error
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				oldMeans[i] = scoring.MeanInit
				timesRanked[i] = 0
			} else {
				return err
			}
		} else {
			oldMeans[i] = lastRating.Mean
			var count int64
			db.DB.Model(&models.Rating{}).Where("user_id = ?", uid).Count(&count)
			timesRanked[i] = int(count)
		}

		var history []float64
		db.DB.Model(&models.Rating{}).Where("user_id = ?", uid).Order("last_rated DESC").Pluck("performance", &history)
		historicalPs[i] = history
	}

	// 3. Recalculate
	newRatings, newMeans, newPerformances := scoring.RecalculateRatings(ranking, oldMeans, timesRanked, historicalPs)

	// 4. Save results in transaction
	return db.DB.Transaction(func(tx *gorm.DB) error {
		now := time.Now()
		for i := 0; i < n; i++ {
			r := models.Rating{
				UserID:          userIDs[i],
				ContestID:       contest.ID,
				ParticipationID: participations[i].ID,
				Rank:            int(math.Round(ranking[i])),
				RatingVal:       newRatings[i],
				Mean:            newMeans[i],
				Performance:     newPerformances[i],
				LastRated:       now,
			}
			if err := tx.Create(&r).Error; err != nil {
				return err
			}

			// Update User Profile
			if err := tx.Model(&models.Profile{}).Where("id = ?", userIDs[i]).Update("rating", newRatings[i]).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
