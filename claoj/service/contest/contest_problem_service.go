// Package contest provides contest management services.
package contest

import (
	"fmt"

	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"gorm.io/gorm"
)

// ContestProblemService provides contest problem management operations.
type ContestProblemService struct{}

// NewContestProblemService creates a new ContestProblemService instance.
func NewContestProblemService() *ContestProblemService {
	return &ContestProblemService{}
}

// ContestProblemInfo is one contest problem row as the admin sees it.
type ContestProblemInfo struct {
	ID          uint   `json:"id"`
	ProblemID   uint   `json:"problem_id"`
	Code        string `json:"code"`
	Name        string `json:"name"`
	Points      int    `json:"points"`
	Partial     bool   `json:"partial"`
	IsPretested bool   `json:"is_pretested"`
	Order       uint   `json:"order"`
}

// ContestProblemInput is one row of a full contest problem list. Its
// position in the list is its order. Omitted fields keep the existing
// row's value, or the defaults (100 points, partial) for a new row.
type ContestProblemInput struct {
	ProblemID   uint  `json:"problem_id"`
	Points      *int  `json:"points"`
	Partial     *bool `json:"partial"`
	IsPretested *bool `json:"is_pretested"`
}

// GetContestProblems retrieves all problems for a contest, in order.
func (s *ContestProblemService) GetContestProblems(contestID uint) ([]ContestProblemInfo, error) {
	var contestProblems []models.ContestProblem
	if err := db.DB.Preload("Problem").Where("contest_id = ?", contestID).Order("`order` ASC").Find(&contestProblems).Error; err != nil {
		return nil, err
	}

	problems := make([]ContestProblemInfo, len(contestProblems))
	for i, cp := range contestProblems {
		problems[i] = ContestProblemInfo{
			ID:          cp.ID,
			ProblemID:   cp.ProblemID,
			Code:        cp.Problem.Code,
			Name:        cp.Problem.Name,
			Points:      cp.Points,
			Partial:     cp.Partial,
			IsPretested: cp.IsPretested,
			Order:       cp.Order,
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
		Select("COALESCE(MAX(`order`), 0)").Scan(&maxOrder).Error; err != nil {
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
			Points:    defaultContestProblemPoints,
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
		Order("`order` ASC").
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

const defaultContestProblemPoints = 100

// ReplaceContestProblems makes the contest's problem rows exactly items,
// in that order. Existing rows are updated in place (their ids are what
// judge_contestsubmission references); unlisted rows are deleted unless
// they already have contest submissions. Run inside the caller's tx.
func (s *ContestProblemService) ReplaceContestProblems(tx *gorm.DB, contestID uint, items []ContestProblemInput) error {
	listed := make(map[uint]bool, len(items))
	ids := make([]uint, 0, len(items))
	for _, it := range items {
		if listed[it.ProblemID] {
			return fmt.Errorf("%w: problem %d is listed twice", ErrUnknownProblem, it.ProblemID)
		}
		if it.Points != nil && *it.Points < 0 {
			return ErrInvalidPoints
		}
		listed[it.ProblemID] = true
		ids = append(ids, it.ProblemID)
	}
	if len(ids) > 0 {
		var n int64
		if err := tx.Model(&models.Problem{}).Where("id IN ?", ids).Count(&n).Error; err != nil {
			return err
		}
		if int(n) != len(ids) {
			return ErrUnknownProblem
		}
	}

	var existing []models.ContestProblem
	if err := tx.Preload("Problem").Where("contest_id = ?", contestID).Find(&existing).Error; err != nil {
		return err
	}
	byProblem := make(map[uint]models.ContestProblem, len(existing))
	for _, cp := range existing {
		byProblem[cp.ProblemID] = cp
	}

	for _, cp := range existing {
		if listed[cp.ProblemID] {
			continue
		}
		var subs int64
		if err := tx.Model(&models.ContestSubmission{}).Where("problem_id = ?", cp.ID).Count(&subs).Error; err != nil {
			return err
		}
		if subs > 0 {
			return fmt.Errorf("%w: %s", ErrContestProblemHasSubmissions, cp.Problem.Code)
		}
		if err := tx.Delete(&models.ContestProblem{}, cp.ID).Error; err != nil {
			return err
		}
	}

	for i, it := range items {
		cp, ok := byProblem[it.ProblemID]
		if !ok {
			cp = models.ContestProblem{
				ContestID: contestID,
				ProblemID: it.ProblemID,
				Points:    defaultContestProblemPoints,
				Partial:   true,
			}
		}
		cp.Order = uint(i + 1)
		if it.Points != nil {
			cp.Points = *it.Points
		}
		if it.Partial != nil {
			cp.Partial = *it.Partial
		}
		if it.IsPretested != nil {
			cp.IsPretested = *it.IsPretested
		}
		cp.Problem = models.Problem{}
		cp.Contest = models.Contest{}
		if err := tx.Omit("Problem", "Contest").Save(&cp).Error; err != nil {
			return err
		}
	}
	return nil
}

// mergeLegacyProblemEdits turns add/remove id lists into a full ordered
// list (current order, removals dropped, additions appended) so the legacy
// form gets the same submission protection as the full list.
func (s *ContestProblemService) mergeLegacyProblemEdits(tx *gorm.DB, contestID uint, add, remove []uint) ([]ContestProblemInput, error) {
	var current []models.ContestProblem
	if err := tx.Where("contest_id = ?", contestID).Order("`order` ASC").Find(&current).Error; err != nil {
		return nil, err
	}
	removed := make(map[uint]bool, len(remove))
	for _, id := range remove {
		removed[id] = true
	}
	present := make(map[uint]bool, len(current)+len(add))
	items := make([]ContestProblemInput, 0, len(current)+len(add))
	for _, cp := range current {
		if removed[cp.ProblemID] {
			continue
		}
		present[cp.ProblemID] = true
		items = append(items, ContestProblemInput{ProblemID: cp.ProblemID})
	}
	for _, id := range add {
		if !present[id] && !removed[id] {
			present[id] = true
			items = append(items, ContestProblemInput{ProblemID: id})
		}
	}
	return items, nil
}
