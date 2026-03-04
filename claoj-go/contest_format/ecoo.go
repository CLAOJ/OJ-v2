package contest_format

import (
	"fmt"
	"math"
	"time"

	"github.com/CLAOJ/claoj-go/db"
	"github.com/CLAOJ/claoj-go/models"
)

type ECOOContestFormat struct {
	BaseFormat
}

func (f *ECOOContestFormat) UpdateParticipation(p *models.ContestParticipation) error {
	var results []struct {
		ProblemID     uint
		ProblemPoints float64
		Points        float64
		DateUnix      int64
		SubCount      int64
	}

	// ECOO uses the score on the LAST non-CE/IE submission.
	// Plus bonuses for first AC and time.
	err := db.DB.Raw(`
		SELECT cp.problem_id, pr.points as problem_points, cs.points, UNIX_TIMESTAMP(sub.date) as date_unix,
		       (SELECT COUNT(*) FROM judge_contestsubmission cs2 
		        JOIN judge_submission sub2 ON sub2.id = cs2.submission_id 
		        WHERE cs2.problem_id = cp.id AND cs2.participation_id = ? AND sub2.result NOT IN ('IE', 'CE')) as sub_count
		FROM judge_contestproblem cp
		JOIN judge_problem pr ON pr.id = cp.problem_id
		JOIN judge_contestsubmission cs ON cs.problem_id = cp.id AND cs.participation_id = ?
		JOIN judge_submission sub ON sub.id = cs.submission_id
		WHERE sub.date = (
			SELECT MAX(sub3.date)
			FROM judge_contestsubmission cs3
			JOIN judge_submission sub3 ON sub3.id = cs3.submission_id
			WHERE cs3.problem_id = cp.id AND cs3.participation_id = ? AND sub3.result NOT IN ('IE', 'CE')
		)
	`, p.ID, p.ID, p.ID).Scan(&results).Error

	if err != nil {
		return err
	}

	firstAcBonus := 10.0
	if val, ok := f.Config["first_ac_bonus"].(float64); ok {
		firstAcBonus = val
	}
	timeBonus := 5.0
	if val, ok := f.Config["time_bonus"].(float64); ok {
		timeBonus = val
	}
	cumtimeEnabled := false
	if val, ok := f.Config["cumtime"].(bool); ok {
		cumtimeEnabled = val
	}

	formatData := make(models.JSONField)
	var totalScore float64
	var totalCumtime float64

	for _, res := range results {
		dt := float64(res.DateUnix) - float64(p.RealStart.Unix())
		if dt < 0 {
			dt = 0
		}

		bonus := 0.0
		if res.Points > 0 {
			// First AC bonus: only if exactly 1 submission and it's full points
			if res.SubCount == 1 && res.Points >= res.ProblemPoints {
				bonus += firstAcBonus
			}
			// Time bonus: for every X minutes before contest end
			if timeBonus > 0 && p.Contest.EndTime.After(p.RealStart) {
				// We need the participant's individual end time if they have one,
				// but usually Participation.RealStart + Contest.TimeLimit
				// Simplified here to use p.Contest.EndTime as in Django if not specified.
				// In DMOJ, it actually uses participation.end_time.
				// Let's assume the window end is Participation.RealStart + TimeLimit if TimeLimit exists.
				windowEnd := p.Contest.EndTime
				if p.Contest.TimeLimit != nil {
					windowEnd = p.RealStart.Add(time.Duration(*p.Contest.TimeLimit) * time.Microsecond)
				}

				rem := windowEnd.Unix() - res.DateUnix
				if rem > 0 {
					bonus += math.Floor(float64(rem) / 60.0 / timeBonus)
				}
			}
		}

		formatData[fmt.Sprint(res.ProblemID)] = map[string]interface{}{
			"points": res.Points,
			"time":   dt,
			"bonus":  bonus,
		}

		totalScore += res.Points + bonus
		if cumtimeEnabled {
			totalCumtime += dt
		}
	}

	p.Score = math.Round(totalScore*math.Pow10(f.Contest.PointsPrecision)) / math.Pow10(f.Contest.PointsPrecision)
	p.Cumtime = uint(totalCumtime)
	p.FormatData = formatData

	return db.DB.Save(p).Error
}

func NewECOOFormat(contest *models.Contest, config models.JSONField) ContestFormat {
	return &ECOOContestFormat{
		BaseFormat: BaseFormat{
			Contest: contest,
			Config:  config,
		},
	}
}
