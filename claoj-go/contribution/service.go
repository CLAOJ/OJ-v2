// Package contribution provides contribution point calculation services.
package contribution

import (
	"github.com/CLAOJ/claoj-go/db"
	"github.com/CLAOJ/claoj-go/models"
)

// ContributionConfig holds the point values for different activities.
var ContributionConfig = struct {
	CommentPointPerScore int // Points per comment score point
	BlogPointPerScore    int // Points per blog post score point
	TicketPointBase      int // Base points for contributive ticket
	ProblemSuggestion    int // Points for approved problem suggestion
}{
	CommentPointPerScore: 1,
	BlogPointPerScore:    1,
	TicketPointBase:      5,
	ProblemSuggestion:    20,
}

// CalculateProfileContributionPoints calculates the total contribution points for a profile.
// Contribution points come from:
// - Comments: sum of comment scores * config multiplier
// - Blogs: sum of blog post scores * config multiplier
// - Tickets: base points for each contributive ticket created by the user
// - Problem Suggestions: points for each approved problem suggestion
func CalculateProfileContributionPoints(profileID uint) (int, error) {
	total := 0

	// Calculate comment contribution (sum of all comment scores)
	var commentScore struct {
		Total int
	}
	err := db.DB.Table("judge_comment").
		Select("COALESCE(SUM(score), 0) as total").
		Where("author_id = ? AND hidden = ?", profileID, false).
		Scan(&commentScore).Error
	if err != nil {
		return 0, err
	}
	total += commentScore.Total * ContributionConfig.CommentPointPerScore

	// Calculate blog contribution (sum of all blog post scores)
	var blogScore struct {
		Total int
	}
	err = db.DB.Table("judge_blogpost").
		Select("COALESCE(SUM(score), 0) as total").
		Where("author_id = ?", profileID).
		Scan(&blogScore).Error
	if err != nil {
		return 0, err
	}
	total += blogScore.Total * ContributionConfig.BlogPointPerScore

	// Calculate ticket contribution (contributive tickets)
	var contributiveTicketCount int64
	err = db.DB.Model(&models.Ticket{}).
		Where("user_id = ? AND is_contributive = ?", profileID, true).
		Count(&contributiveTicketCount).Error
	if err != nil {
		return 0, err
	}
	total += int(contributiveTicketCount) * ContributionConfig.TicketPointBase

	// Calculate problem suggestion contribution (approved suggestions)
	var approvedSuggestionCount int64
	err = db.DB.Model(&models.Problem{}).
		Where("suggester_id = ? AND suggestion_status = ?", profileID, "approved").
		Count(&approvedSuggestionCount).Error
	if err != nil {
		return 0, err
	}
	total += int(approvedSuggestionCount) * ContributionConfig.ProblemSuggestion

	return total, nil
}

// UpdateProfileContributionPoints calculates and saves the contribution points for a profile.
func UpdateProfileContributionPoints(profileID uint) error {
	points, err := CalculateProfileContributionPoints(profileID)
	if err != nil {
		return err
	}

	return db.DB.Model(&models.Profile{}).
		Where("id = ?", profileID).
		Update("contribution_points", points).Error
}

// UpdateAllProfileContributionPoints recalculates contribution points for all profiles.
// This is useful for migration or periodic recalculation.
func UpdateAllProfileContributionPoints() error {
	var profiles []models.Profile
	if err := db.DB.Select("id").Find(&profiles).Error; err != nil {
		return err
	}

	for _, p := range profiles {
		if err := UpdateProfileContributionPoints(p.ID); err != nil {
			// Continue with other profiles even if one fails
			continue
		}
	}

	return nil
}
