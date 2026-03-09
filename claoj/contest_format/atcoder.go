package contest_format

import (
	"fmt"
	"math"

	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
)

type AtCoderContestFormat struct {
	BaseFormat
}

func (f *AtCoderContestFormat) UpdateParticipation(p *models.ContestParticipation) error {
	var results []struct {
		ProblemID uint
		Points    float64
		MinTime   int64
	}

	// AtCoder uses max points and the time of the FIRST submission attaining those points.
	err := db.DB.Raw(`
		SELECT q.prob as problem_id, q.max_points as points, MIN(UNIX_TIMESTAMP(sub.date)) as min_time
		FROM (
			SELECT cp.id as prob, MAX(cs.points) as max_points
			FROM judge_contestproblem cp
			INNER JOIN judge_contestsubmission cs ON cs.problem_id = cp.id AND cs.participation_id = ?
			GROUP BY cp.id
		) q
		INNER JOIN judge_contestsubmission cs2 ON cs2.problem_id = q.prob AND cs2.participation_id = ? AND cs2.points = q.max_points
		INNER JOIN judge_submission sub ON sub.id = cs2.submission_id
		GROUP BY q.prob
	`, p.ID, p.ID).Scan(&results).Error

	if err != nil {
		return err
	}

	penaltyMinutes := 5.0
	if val, ok := f.Config["penalty"].(float64); ok {
		penaltyMinutes = val
	}

	formatData := make(models.JSONField)
	var totalScore float64
	var maxDt float64
	var totalPenalty float64

	for _, res := range results {
		dt := float64(res.MinTime) - float64(p.RealStart.Unix())
		if dt < 0 {
			dt = 0
		}

		var penaltyCount int64
		db.DB.Model(&models.ContestSubmission{}).
			Joins("JOIN judge_submission ON judge_submission.id = judge_contestsubmission.submission_id").
			Where("judge_contestsubmission.participation_id = ? AND judge_contestsubmission.problem_id = ?", p.ID, res.ProblemID).
			Where("judge_submission.date <= FROM_UNIXTIME(?)", res.MinTime).
			Where("judge_submission.result NOT IN ('CE', 'IE') AND judge_submission.result IS NOT NULL").
			Count(&penaltyCount)

		actualPenalty := penaltyCount
		if res.Points > 0 {
			actualPenalty = penaltyCount - 1
		}
		if actualPenalty < 0 {
			actualPenalty = 0
		}

		formatData[fmt.Sprint(res.ProblemID)] = map[string]interface{}{
			"points":  res.Points,
			"time":    dt,
			"penalty": float64(actualPenalty),
		}

		totalScore += res.Points
		if res.Points > 0 {
			if dt > maxDt {
				maxDt = dt
			}
			totalPenalty += float64(actualPenalty) * penaltyMinutes * 60
		}
	}

	p.Score = math.Round(totalScore*math.Pow10(f.Contest.PointsPrecision)) / math.Pow10(f.Contest.PointsPrecision)
	p.Cumtime = uint(maxDt + totalPenalty)
	p.Tiebreaker = 0
	p.FormatData = formatData

	return db.DB.Save(p).Error
}

func NewAtCoderFormat(contest *models.Contest, config models.JSONField) ContestFormat {
	return &AtCoderContestFormat{
		BaseFormat: BaseFormat{
			Contest: contest,
			Config:  config,
		},
	}
}
