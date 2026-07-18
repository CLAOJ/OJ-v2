// claoj/auth/access.go
package auth

import (
	"time"

	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/gin-gonic/gin"
)

// CurrentProfileID resolves the request user's judge_profile id.
func CurrentProfileID(c *gin.Context) (uint, bool) {
	userID, ok := c.Get("user_id")
	if !ok {
		return 0, false
	}
	if v, ok := c.Get("current_profile_id"); ok {
		return v.(uint), true
	}
	var profile models.Profile
	if err := db.DB.Select("id").Where("user_id = ?", userID.(uint)).First(&profile).Error; err != nil {
		return 0, false
	}
	c.Set("current_profile_id", profile.ID)
	return profile.ID, true
}

func profileInList(profiles []models.Profile, profileID uint) bool {
	for _, p := range profiles {
		if p.ID == profileID {
			return true
		}
	}
	return false
}

// problemEditorIDs mirrors Problem.editor_ids (OJ problem.py:374-381):
// authors ∪ curators ∪ suggester.
func isProblemEditor(problem *models.Problem, profileID uint) bool {
	if profileInList(problem.Authors, profileID) || profileInList(problem.Curators, profileID) {
		return true
	}
	return problem.SuggesterID != nil && *problem.SuggesterID == profileID
}

// CanEditProblem mirrors Problem.is_editable_by (OJ problem.py:229-238).
// The problem must be loaded with Preload("Authors").Preload("Curators").
func CanEditProblem(c *gin.Context, problem *models.Problem) bool {
	access := GetAccess(c)
	if access.IsSuperuser && access.IsActive {
		return true
	}
	if !access.HasPerm("judge.edit_own_problem") {
		return false
	}
	if access.HasPerm("judge.edit_all_problem") {
		return true
	}
	if access.HasPerm("judge.edit_public_problem") && problem.IsPublic {
		return true
	}
	profileID, ok := CurrentProfileID(c)
	return ok && isProblemEditor(problem, profileID)
}

// CanViewProblem mirrors the codename-relevant parts of Problem.is_accessible_by
// (OJ problem.py:240-284): public problems are visible; hidden ones require
// see_private_problem, editability, or being a tester.
// Load with Preload("Authors").Preload("Curators").Preload("Testers").
func CanViewProblem(c *gin.Context, problem *models.Problem) bool {
	if problem.IsPublic {
		return true
	}
	access := GetAccess(c)
	if access.IsSuperuser && access.IsActive {
		return true
	}
	if access.HasPerm("judge.see_private_problem") {
		return true
	}
	if CanEditProblem(c, problem) {
		return true
	}
	profileID, ok := CurrentProfileID(c)
	return ok && profileInList(problem.Testers, profileID)
}

// contestEditorIDs mirrors Contest.editor_ids: authors ∪ curators.
func isContestEditor(contest *models.Contest, profileID uint) bool {
	return profileInList(contest.Authors, profileID) || profileInList(contest.Curators, profileID)
}

// CanEditContest mirrors Contest.is_editable_by (OJ contest.py:380-389).
// Load with Preload("Authors").Preload("Curators").
func CanEditContest(c *gin.Context, contest *models.Contest) bool {
	access := GetAccess(c)
	if access.IsSuperuser && access.IsActive {
		return true
	}
	if access.HasPerm("judge.edit_all_contest") {
		return true
	}
	if !access.HasPerm("judge.edit_own_contest") {
		return false
	}
	profileID, ok := CurrentProfileID(c)
	return ok && isContestEditor(contest, profileID)
}

// CanViewContest mirrors the codename-relevant parts of Contest.access_check
// (OJ contest.py:321-370). Load with Preload("Authors").Preload("Curators").Preload("Testers").
func CanViewContest(c *gin.Context, contest *models.Contest) bool {
	if contest.IsVisible {
		return true
	}
	access := GetAccess(c)
	if access.IsSuperuser && access.IsActive {
		return true
	}
	if access.HasPerm("judge.see_private_contest") || access.HasPerm("judge.edit_all_contest") {
		return true
	}
	profileID, ok := CurrentProfileID(c)
	if !ok {
		return false
	}
	return isContestEditor(contest, profileID) || profileInList(contest.Testers, profileID)
}

// CanViewSolution mirrors Solution.is_accessible_by (OJ problem.py:630-637).
func CanViewSolution(c *gin.Context, solution *models.Solution, problem *models.Problem) bool {
	if solution.IsPublic && solution.PublishOn != nil && solution.PublishOn.Before(time.Now()) {
		return true
	}
	if HasPerm(c, "judge.see_private_solution") {
		return true
	}
	return CanEditProblem(c, problem)
}

// CanRejudge mirrors Problem.is_rejudgeable_by (OJ problem.py:286-287).
func CanRejudge(c *gin.Context, problem *models.Problem) bool {
	access := GetAccess(c)
	if access.IsSuperuser && access.IsActive {
		return true
	}
	return access.HasPerm("judge.rejudge_submission") && CanEditProblem(c, problem)
}
