package scoring

import (
	"math"

	"github.com/CLAOJ/claoj/models"
	"gorm.io/gorm"
)

const (
	PpStep    = 0.95
	PpEntries = 100
)

func PpBonusFunction(n int) float64 {
	return 300 * (1 - math.Pow(0.997, float64(n)))
}

var ppTable []float64

func init() {
	ppTable = make([]float64, PpEntries)
	for i := 0; i < PpEntries; i++ {
		ppTable[i] = math.Pow(PpStep, float64(i))
	}
}

// ppVisibility is v1's public-problem rule (Problem.get_public_problems) plus
// its unarchived-submission rule (performance_points.get_pp_breakdown).
const ppVisibility = "pr.is_public = 1 AND pr.is_organization_private = 0 AND s.is_archived = 0"

// PPProblem is one problem's entry in a performance-points breakdown.
type PPProblem struct {
	Code         string  `json:"code"`
	Name         string  `json:"name"`
	Points       float64 `json:"points"`
	Weight       float64 `json:"weight"`
	Contribution float64 `json:"contribution"`
}

// PPResult is a profile's performance-points calculation.
type PPResult struct {
	// Problems holds at most PpEntries rows: only those can contribute.
	Problems          []PPProblem `json:"problems"`
	TotalPoints       float64     `json:"points"`
	SolvedCount       int         `json:"solved_count"`
	BonusPoints       float64     `json:"bonus_points"`
	WeightedSum       float64     `json:"weighted_sum"`
	PerformancePoints float64     `json:"performance_points"`
}

// CalculatePP computes a profile's points from scratch. Both the stored
// judge_profile columns and the /pp-breakdown endpoint use this function, so
// the header and the breakdown can never disagree.
func CalculatePP(db *gorm.DB, profileID uint) (*PPResult, error) {
	// Best unarchived submission per visible public problem, highest first.
	var rows []struct {
		Code      string  `gorm:"column:code"`
		Name      string  `gorm:"column:name"`
		MaxPoints float64 `gorm:"column:max_points"`
	}
	if err := db.Raw(`
		SELECT pr.code, pr.name, MAX(s.points) AS max_points
		FROM judge_submission s
		JOIN judge_problem pr ON s.problem_id = pr.id
		WHERE s.user_id = ? AND `+ppVisibility+` AND s.points IS NOT NULL
		GROUP BY pr.id, pr.code, pr.name
		HAVING max_points > 0
		ORDER BY max_points DESC, pr.id ASC
	`, profileID).Scan(&rows).Error; err != nil {
		return nil, err
	}

	res := &PPResult{TotalPoints: 0, PerformancePoints: 0}
	if len(rows) > PpEntries {
		res.Problems = make([]PPProblem, PpEntries)
	} else {
		res.Problems = make([]PPProblem, len(rows))
	}
	for i, row := range rows {
		res.TotalPoints += row.MaxPoints
		if i >= PpEntries {
			continue
		}
		weight := ppTable[i]
		contribution := weight * row.MaxPoints
		res.WeightedSum += contribution
		res.Problems[i] = PPProblem{
			Code:         row.Code,
			Name:         row.Name,
			Points:       row.MaxPoints,
			Weight:       weight,
			Contribution: contribution,
		}
	}

	// Distinct solved problems (fully accepted, unarchived).
	var solvedCount int64
	if err := db.Raw(`
		SELECT COUNT(DISTINCT s.problem_id)
		FROM judge_submission s
		JOIN judge_problem pr ON s.problem_id = pr.id
		WHERE s.user_id = ? AND `+ppVisibility+`
		  AND s.result = 'AC' AND s.case_points >= s.case_total
	`, profileID).Scan(&solvedCount).Error; err != nil {
		return nil, err
	}
	res.SolvedCount = int(solvedCount)
	res.BonusPoints = PpBonusFunction(res.SolvedCount)
	res.PerformancePoints = res.WeightedSum + res.BonusPoints

	return res, nil
}

// CalculateProfilePoints recalculates Points, PerformancePoints, and ProblemCount for a user.
func CalculateProfilePoints(db *gorm.DB, profileID uint) error {
	var profile models.Profile
	if err := db.First(&profile, profileID).Error; err != nil {
		return err
	}

	res, err := CalculatePP(db, profileID)
	if err != nil {
		return err
	}

	// 3. Update profile
	return db.Model(&profile).Updates(map[string]interface{}{
		"points":             res.TotalPoints,
		"problem_count":      res.SolvedCount,
		"performance_points": res.PerformancePoints,
	}).Error
}

// CalculateOrganizationPoints recalculates PerformancePoints for an organization based on its members.
func CalculateOrganizationPoints(db *gorm.DB, orgID uint) error {
	var org models.Organization
	if err := db.First(&org, orgID).Error; err != nil {
		return err
	}

	var memberPPs []float64
	if err := db.Raw(`
		SELECT p.performance_points
		FROM judge_profile p
		JOIN judge_profile_organizations po ON p.id = po.profile_id
		WHERE po.organization_id = ? AND p.performance_points > 0
		ORDER BY p.performance_points DESC
	`, orgID).Scan(&memberPPs).Error; err != nil {
		return err
	}

	totalPP := 0.0
	// For organizations, the formula uses the same PP tables but usually has a different scale if configured.
	// In settings.py, CLAOJ_ORG_PP_SCALE = 1, CLAOJ_ORG_PP_STEP = 0.95
	for i, mpp := range memberPPs {
		if i < PpEntries {
			totalPP += math.Pow(PpStep, float64(i)) * mpp
		}
	}

	return db.Model(&org).Update("performance_points", totalPP).Error
}