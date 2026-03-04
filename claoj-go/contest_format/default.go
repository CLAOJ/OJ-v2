package contest_format

import (
	"fmt"
	"math"

	"github.com/CLAOJ/claoj-go/db"
	"github.com/CLAOJ/claoj-go/models"
)

type DefaultContestFormat struct {
	BaseFormat
}

func (f *DefaultContestFormat) UpdateParticipation(p *models.ContestParticipation) error {
	var results []struct {
		ProblemID uint
		MaxPoints float64
		MaxTime   float64
	}

	// Fetch max points and latest time for each problem for this user in this contest
	err := db.DB.Model(&models.Submission{}).
		Select("problem_id, MAX(points) as max_points, MAX(UNIX_TIMESTAMP(date)) as max_time").
		Where("user_id = ? AND problem_id IN (?)", p.UserID,
			db.DB.Model(&models.ContestProblem{}).Select("problem_id").Where("contest_id = ?", p.ContestID),
		).
		Group("problem_id").
		Scan(&results).Error

	if err != nil {
		return err
	}

	var totalPoints float64
	var totalCumtime float64
	formatData := make(models.JSONField)

	for _, res := range results {
		totalPoints += res.MaxPoints
		// cumtime in DMOJ/CLAOJ is usually sum of (submission_time - contest_start) for non-zero points
		// res.MaxTime is absolute unix timestamp
		dt := res.MaxTime - float64(p.RealStart.Unix())
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
