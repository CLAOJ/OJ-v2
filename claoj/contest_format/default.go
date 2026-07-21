package contest_format

import (
	"fmt"
	"math"

	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
)

type DefaultContestFormat struct {
	BaseFormat
}

func (f *DefaultContestFormat) UpdateParticipation(p *models.ContestParticipation) error {
	var results []struct {
		ProblemID uint // judge_contestproblem.id
		MaxPoints float64
		MaxTime   float64
	}

	// DMOJ default format: for each contest problem, take the max points and the
	// latest submission time across THIS participation's contest submissions only.
	//
	// This must read judge_contestsubmission (scoped to the participation), NOT
	// judge_submission: the old query counted every submission the user ever made
	// to the problems (including outside the contest) and used the archive-scale
	// judge_submission.points instead of the contest-scale judge_contestsubmission.points,
	// yielding scores that were both inflated in set and ~100x off in magnitude.
	// It is grouped by cs.problem_id, which is the ContestProblem.id that
	// FormatData is keyed by.
	err := db.DB.Raw(`
		SELECT cs.problem_id AS problem_id,
		       MAX(cs.points) AS max_points,
		       MAX(UNIX_TIMESTAMP(sub.date)) AS max_time
		FROM judge_contestsubmission cs
		INNER JOIN judge_submission sub ON sub.id = cs.submission_id
		WHERE cs.participation_id = ?
		GROUP BY cs.problem_id
	`, p.ID).Scan(&results).Error

	if err != nil {
		return err
	}

	start := float64(ParticipationStart(f.Contest, p).Unix())

	var totalPoints float64
	var totalCumtime float64
	formatData := make(models.JSONField)

	for _, res := range results {
		totalPoints += res.MaxPoints
		dt := res.MaxTime - start
		if dt < 0 {
			dt = 0
		}

		if res.MaxPoints > 0 {
			totalCumtime += dt
		}

		formatData[fmt.Sprint(res.ProblemID)] = map[string]interface{}{
			"points": res.MaxPoints,
			"time":   dt,
		}
	}

	p.Score = math.Round(totalPoints*math.Pow10(f.Contest.PointsPrecision)) / math.Pow10(f.Contest.PointsPrecision)
	p.Cumtime = uint(totalCumtime)
	p.Tiebreaker = 0
	p.FormatData = formatData

	return db.DB.Save(p).Error
}

func NewDefaultFormat(contest *models.Contest, config models.JSONField) ContestFormat {
	return &DefaultContestFormat{
		BaseFormat: BaseFormat{
			Contest: contest,
			Config:  config,
		},
	}
}
