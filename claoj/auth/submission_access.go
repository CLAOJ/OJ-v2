package auth

import (
	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/gin-gonic/gin"
)

// v1's DMOJ_SUBMISSION_SOURCE_VISIBILITY ("all-solved"), which problems in
// FOLLOW ('F') mode inherit.
const defaultSourceVisibility = "S"

// CanSeeSubmissionDetail mirrors Submission.can_see_detail (v1
// judge/models/submission.py): who may open a submission's source and test
// cases. sub.Problem must be loaded with Authors, Curators and Testers.
func CanSeeSubmissionDetail(c *gin.Context, sub *models.Submission) bool {
	profileID, ok := CurrentProfileID(c)
	if !ok {
		return false
	}
	p := &sub.Problem
	if CanEditProblem(c, p) || HasPerm(c, "judge.view_all_submission") {
		return true
	}
	if !p.IsPublic && p.SuggesterID != nil && HasPerm(c, "judge.suggest_new_problem") {
		return true
	}
	if sub.UserID == profileID {
		return true
	}
	mode := p.SubmissionSourceVisibilityMode
	if mode == "F" || mode == "" {
		mode = defaultSourceVisibility
	}
	tester := profileInList(p.Testers, profileID)
	switch mode {
	case "A":
		return true
	case "S":
		if p.IsPublic || tester {
			var n int64
			db.DB.Model(&models.Submission{}).
				Where("user_id = ? AND problem_id = ? AND result = 'AC' AND points = ?", profileID, p.ID, p.Points).
				Count(&n)
			if n > 0 {
				return true
			}
		}
	case "O":
		if tester {
			return true
		}
	}
	if sub.ContestObjectID != nil {
		var contest models.Contest
		if db.DB.Preload("Authors").Preload("Curators").First(&contest, *sub.ContestObjectID).Error == nil &&
			isContestEditor(&contest, profileID) {
			return true
		}
	}
	return false
}

// CanViewTestcases mirrors Problem.is_testcase_accessible_by (v1
// judge/models/problem.py): 'A' always, 'C' outside contests, 'O' editors only.
// The problem must be loaded with Authors and Curators.
func CanViewTestcases(c *gin.Context, p *models.Problem) bool {
	if p.TestcaseVisibilityMode == "A" {
		return true
	}
	profileID, ok := CurrentProfileID(c)
	if !ok {
		return false
	}
	if CanEditProblem(c, p) {
		return true
	}
	if p.TestcaseVisibilityMode == "C" {
		var profile models.Profile
		return db.DB.Select("current_contest_id").First(&profile, profileID).Error == nil && profile.CurrentContestID == nil
	}
	return false
}
