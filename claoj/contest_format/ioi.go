package contest_format

import (
	"fmt"
	"math"

	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
)

type IOI16ContestFormat struct {
	BaseFormat
}

func (f *IOI16ContestFormat) UpdateParticipation(p *models.ContestParticipation) error {
	var results []struct {
		ProblemID uint
		Batch     int
		MaxPoints float64
		MinDate   int64
	}

	// This query replicates the complex IOI subtask scoring logic:
	// For each problem and batch, find the max points attained across all submissions.
	// We also need the date of the FIRST submission that achieved that max points for the batch.

	// First, find the max points for each (problem, batch)
	err := db.DB.Raw(`
		SELECT q.prob as problem_id, q.batch, q.batch_points as max_points, MIN(q.date_unix) as min_date
		FROM (
			SELECT cp.id as prob, tc.batch, tc.points as batch_points, UNIX_TIMESTAMP(sub.date) as date_unix
			FROM judge_contestproblem cp
			INNER JOIN judge_contestsubmission cs ON cs.problem_id = cp.id AND cs.participation_id = ?
			INNER JOIN judge_submission sub ON sub.id = cs.submission_id AND sub.status = 'D'
			INNER JOIN judge_submissiontestcase tc ON sub.id = tc.submission_id
		) q
		INNER JOIN (
			SELECT q2.prob, q2.batch, MAX(q2.batch_points) as max_batch_points
			FROM (
				SELECT cp.id as prob, tc.batch, tc.points as batch_points
				FROM judge_contestproblem cp
				INNER JOIN judge_contestsubmission cs ON cs.problem_id = cp.id AND cs.participation_id = ?
				INNER JOIN judge_submission sub ON sub.id = cs.submission_id AND sub.status = 'D'
				INNER JOIN judge_submissiontestcase tc ON sub.id = tc.submission_id
			) q2
			GROUP BY q2.prob, q2.batch
		) p ON p.prob = q.prob AND (p.batch = q.batch OR (p.batch IS NULL AND q.batch IS NULL))
		WHERE q.batch_points = p.max_batch_points
		GROUP BY q.prob, q.batch
	`, p.ID, p.ID).Scan(&results).Error

	if err != nil {
		return err
	}

	cumtimeEnabled := false
	if val, ok := f.Config["cumtime"].(bool); ok {
		cumtimeEnabled = val
	}

	formatData := make(models.JSONField)
	var totalScore float64
	var totalCumtime float64

	for _, res := range results {
		probIDStr := fmt.Sprint(res.ProblemID)
		if _, ok := formatData[probIDStr]; !ok {
			formatData[probIDStr] = map[string]interface{}{
				"points": 0.0,
				"time":   0.0,
			}
		}

		d := formatData[probIDStr].(map[string]interface{})
		d["points"] = d["points"].(float64) + res.MaxPoints

		if cumtimeEnabled {
			dt := float64(res.MinDate) - float64(p.RealStart.Unix())
			if dt < 0 {
				dt = 0
			}
			if dt > d["time"].(float64) {
				d["time"] = dt
			}
		}
	}

	for _, dInterface := range formatData {
		d := dInterface.(map[string]interface{})
		points := d["points"].(float64)
		totalScore += points
		if cumtimeEnabled && points > 0 {
			totalCumtime += d["time"].(float64)
		}
	}

	p.Score = math.Round(totalScore*math.Pow10(f.Contest.PointsPrecision)) / math.Pow10(f.Contest.PointsPrecision)
	p.Cumtime = uint(totalCumtime)
	p.FormatData = formatData

	return db.DB.Save(p).Error
}

func NewIOI16Format(contest *models.Contest, config models.JSONField) ContestFormat {
	return &IOI16ContestFormat{
		BaseFormat: BaseFormat{
			Contest: contest,
			Config:  config,
		},
	}
}
