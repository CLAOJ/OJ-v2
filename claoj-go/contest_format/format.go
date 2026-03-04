package contest_format

import (
	"fmt"

	"github.com/CLAOJ/claoj-go/models"
)

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
func (f *BaseFormat) GetProblemBreakdown(p *models.ContestParticipation, problems []models.ContestProblem) []interface{} {
	breakdown := make([]interface{}, len(problems))
	for i, cp := range problems {
		if data, ok := p.FormatData[fmt.Sprint(cp.ProblemID)]; ok {
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
