package scoring

import (
	"math"

	"github.com/CLAOJ/claoj-go/models"
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

// CalculateProfilePoints recalculates Points, PerformancePoints, and ProblemCount for a user.
func CalculateProfilePoints(db *gorm.DB, profileID uint) error {
	var profile models.Profile
	if err := db.First(&profile, profileID).Error; err != nil {
		return err
	}

	// 1. Get max points per public problem
	type result struct {
		MaxPoints float64
	}
	var maxPointsData []result

	if err := db.Raw(`
		SELECT MAX(s.points) as max_points
		FROM judge_submission s
		JOIN judge_problem p ON s.problem_id = p.id
		WHERE s.user_id = ? AND p.is_public = 1 AND s.points IS NOT NULL
		GROUP BY s.problem_id
		HAVING max_points > 0
		ORDER BY max_points DESC
	`, profileID).Scan(&maxPointsData).Error; err != nil {
		return err
	}

	totalPoints := 0.0
	pp := 0.0
	for i, res := range maxPointsData {
		totalPoints += res.MaxPoints
		if i < PpEntries {
			pp += ppTable[i] * res.MaxPoints
		}
	}

	// 2. Count distinct AC problems
	var solvedCount int64
	if err := db.Raw(`
		SELECT COUNT(DISTINCT s.problem_id)
		FROM judge_submission s
		JOIN judge_problem p ON s.problem_id = p.id
		WHERE s.user_id = ? AND p.is_public = 1 AND s.result = 'AC' AND s.case_points >= s.case_total
	`, profileID).Scan(&solvedCount).Error; err != nil {
		return err
	}

	pp += PpBonusFunction(int(solvedCount))

	// 3. Update profile
	return db.Model(&profile).Updates(map[string]interface{}{
		"points":             totalPoints,
		"problem_count":      int(solvedCount),
		"performance_points": pp,
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
