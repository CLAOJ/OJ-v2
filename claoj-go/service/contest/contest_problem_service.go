// Package contest provides contest management services.
package contest

import (
	"github.com/CLAOJ/claoj-go/db"
	"github.com/CLAOJ/claoj-go/models"
	"gorm.io/gorm"
)

// ContestProblemService provides contest problem management operations.
type ContestProblemService struct{}

// NewContestProblemService creates a new ContestProblemService instance.
func NewContestProblemService() *ContestProblemService {
	return &ContestProblemService{}
}

// ContestProblemInfo holds contest problem information.
type ContestProblemInfo struct {
	ID        uint
	ProblemID uint
	Code      string
	Name      string
	Points    int
	Order     uint
	Partial   bool
}

// GetContestProblems retrieves all problems for a contest.
func (s *ContestProblemService) GetContestProblems(contestID uint) ([]ContestProblemInfo, error) {
	var contestProblems []models.ContestProblem
	if err := db.DB.Where("contest_id = ?", contestID).Order("order ASC").Find(&contestProblems).Error; err != nil {
		return nil, err
	}

	problems := make([]ContestProblemInfo, len(contestProblems))
	for i, cp := range contestProblems {
		problems[i] = ContestProblemInfo{
			ID:        cp.ID,
			ProblemID: cp.ProblemID,
			Code:      cp.Problem.Code,
			Name:      cp.Problem.Name,
			Points:    cp.Points,
			Order:     cp.Order,
			Partial:   cp.Partial,
		}
	}

	return problems, nil
}

// AddProblemsToContest adds problems to a contest.
func (s *ContestProblemService) AddProblemsToContest(contestID uint, problemIDs []uint) error {
	if len(problemIDs) == 0 {
		return nil
	}

	var problems []models.Problem
	if err := db.DB.Where("id IN ?", problemIDs).Find(&problems).Error; err != nil {
		return err
	}

	// Get existing problem IDs to avoid duplicates
	var existing []uint
	if err := db.DB.Table("judge_contestproblem").
		Where("contest_id = ?", contestID).
		Pluck("problem_id", &existing).Error; err != nil {
		return err
	}

	// Get current max order
	var maxOrder uint
	if err := db.DB.Table("judge_contestproblem").
		Where("contest_id = ?", contestID).
		Select("COALESCE(MAX(order), 0)").Scan(&maxOrder).Error; err != nil {
		return err
	}

	order := maxOrder
	for _, p := range problems {
		if containsUint(existing, p.ID) {
			continue
		}
		order++
		cp := models.ContestProblem{
			ContestID: contestID,
			ProblemID: p.ID,
			Points:    100,
			Partial:   true,
			Order:     order,
		}
		if err := db.DB.Create(&cp).Error; err != nil {
			return err
		}
	}

	return nil
}

// RemoveProblemsFromContest removes problems from a contest.
func (s *ContestProblemService) RemoveProblemsFromContest(contestID uint, problemIDs []uint) error {
	if len(problemIDs) == 0 {
		return nil
	}

	return db.DB.Where("contest_id = ? AND problem_id IN ?", contestID, problemIDs).
		Delete(&models.ContestProblem{}).Error
}

// CopyProblemsToContest copies all problems from a source contest to a target contest.
func (s *ContestProblemService) CopyProblemsToContest(sourceContestID, targetContestID uint) error {
	var contestProblems []models.ContestProblem
	if err := db.DB.Where("contest_id = ?", sourceContestID).
		Order("order ASC").
		Find(&contestProblems).Error; err != nil {
		return err
	}

	for _, cp := range contestProblems {
		newCP := models.ContestProblem{
			ContestID:            targetContestID,
			ProblemID:            cp.ProblemID,
			Points:               cp.Points,
			Partial:              cp.Partial,
			IsPretested:          cp.IsPretested,
			Order:                cp.Order,
			OutputPrefixOverride: cp.OutputPrefixOverride,
			MaxSubmissions:       cp.MaxSubmissions,
		}
		if err := db.DB.Create(&newCP).Error; err != nil {
			return err
		}
	}

	return nil
}

// UpdateProblemConfig updates the configuration for a specific contest problem.
func (s *ContestProblemService) UpdateProblemConfig(contestID, problemID uint, points int, partial bool, order uint) error {
	updates := make(map[string]interface{})
	if points > 0 {
		updates["points"] = points
	}
	updates["partial"] = partial
	if order > 0 {
		updates["order"] = order
	}

	if len(updates) == 0 {
		return nil
	}

	return db.DB.Model(&models.ContestProblem{}).
		Where("contest_id = ? AND problem_id = ?", contestID, problemID).
		Updates(updates).Error
}

// GetProblemConfig retrieves the configuration for a specific contest problem.
func (s *ContestProblemService) GetProblemConfig(contestID, problemID uint) (*ContestProblemInfo, error) {
	var cp models.ContestProblem
	if err := db.DB.Where("contest_id = ? AND problem_id = ?", contestID, problemID).
		Preload("Problem").
		First(&cp).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrContestNotFound
		}
		return nil, err
	}

	return &ContestProblemInfo{
		ID:        cp.ID,
		ProblemID: cp.ProblemID,
		Code:      cp.Problem.Code,
		Name:      cp.Problem.Name,
		Points:    cp.Points,
		Order:     cp.Order,
		Partial:   cp.Partial,
	}, nil
}

// containsUint checks if a uint slice contains a value.
func containsUint(slice []uint, item uint) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
