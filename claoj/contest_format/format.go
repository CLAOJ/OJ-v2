package contest_format

import (
	"fmt"
	"time"

	"github.com/CLAOJ/claoj/models"
)

// ParticipationStart returns the instant a participation's solve times are
// measured from, mirroring DMOJ's ContestParticipation.start property: for a
// live or spectating participant (virtual <= 0) in a contest with no per-user
// time limit, the shared whole-contest window applies (contest.start_time);
// otherwise the participant's own real_start (join time) is used.
//
// The Go formats previously always used real_start, which produced wrong
// per-problem times and cumtime for the common no-time-limit contest.
func ParticipationStart(contest *models.Contest, p *models.ContestParticipation) time.Time {
	if contest != nil && contest.TimeLimit == nil && p.Virtual <= 0 {
		return contest.StartTime
	}
	return p.RealStart
}

// ContestFormat defines the interface for different contest scoring rules.
type ContestFormat interface {
	// UpdateParticipation calculates and updates scores for a participant.
	UpdateParticipation(p *models.ContestParticipation) error

	// GetLabelForProblem returns the label (e.g., "A", "1") for a problem index.
	GetLabelForProblem(index int) string

	// GetProblemBreakdown returns detailed scoring information per problem.
	GetProblemBreakdown(p *models.ContestParticipation, problems []models.ContestProblem) []interface{}

	// BestSolutionState returns a status string (e.g., "full-score") for a problem result.
	BestSolutionState(points, total float64) string
}

// BaseFormat provides default implementations for common methods.
// Concrete formats embed this and only override what differs.
type BaseFormat struct {
	Contest *models.Contest
	Config  models.JSONField
}

// GetLabelForProblem returns a 1-indexed numeric label ("1", "2", …).
// ICPC overrides this to produce alphabetical labels ("A", "B", …).
func (f *BaseFormat) GetLabelForProblem(index int) string {
	return fmt.Sprint(index + 1)
}

// GetProblemBreakdown returns the per-problem FormatData entries in problem order.
//
// FormatData is keyed by ContestProblem.id (the judge_contestproblem row id),
// matching DMOJ's convention (judge_contestsubmission.problem_id is a FK to
// judge_contestproblem). Keying by cp.ProblemID (the underlying Problem.id)
// silently missed on every lookup, so all breakdown cells came back nil.
func (f *BaseFormat) GetProblemBreakdown(p *models.ContestParticipation, problems []models.ContestProblem) []interface{} {
	breakdown := make([]interface{}, len(problems))
	for i, cp := range problems {
		if data, ok := p.FormatData[fmt.Sprint(cp.ID)]; ok {
			breakdown[i] = data
		} else {
			breakdown[i] = nil
		}
	}
	return breakdown
}

func (f *BaseFormat) BestSolutionState(points, total float64) string {
	if points <= 0 {
		return "failed-score"
	}
	if points >= total {
		return "full-score"
	}
	return "partial-score"
}
