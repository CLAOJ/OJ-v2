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

// CanViewProblem mirrors Problem.is_accessible_by (OJ problem.py:240-284). A
// public problem that is private to organizations is only visible to members of
// those organizations (or holders of see_organization_problem); hidden problems
// require see_private_problem, editability, or being a tester.
// Load with Preload("Authors").Preload("Curators").Preload("Testers").
func CanViewProblem(c *gin.Context, problem *models.Problem) bool {
	access := GetAccess(c)

	// Public problems are visible unless they are restricted to organizations.
	if problem.IsPublic {
		if !problem.IsOrganizationPrivate {
			return true
		}
		if access.HasPerm("judge.see_organization_problem") {
			return true
		}
		if profileID, ok := CurrentProfileID(c); ok && userInProblemOrg(problem.ID, profileID) {
			return true
		}
		// Not an organization member — editors, testers, or see_private_problem
		// below may still grant access.
	}

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

// CanViewContest mirrors Contest.access_check (OJ contest.py:321-370). It fully
// enforces organization-private and private-contestant restrictions: a merely
// visible contest is NOT automatically viewable if it is limited to an
// organization or a private contestant list. Membership is resolved from the
// database by contest id, so the caller only needs the contest's id and its
// is_visible / is_organization_private / is_private flags populated (a plain
// row load is enough — no association preloads required).
func CanViewContest(c *gin.Context, contest *models.Contest) bool {
	access := GetAccess(c)
	if access.IsSuperuser && access.IsActive {
		return true
	}
	if access.HasPerm("judge.see_private_contest") || access.HasPerm("judge.edit_all_contest") {
		return true
	}

	profileID, authed := CurrentProfileID(c)

	// Authors, curators and testers may always view their contest, even while
	// it is hidden (the editor/tester short-circuits in access_check).
	if authed && (contestHasMember(contest.ID, "judge_contest_authors", profileID) ||
		contestHasMember(contest.ID, "judge_contest_curators", profileID) ||
		contestHasMember(contest.ID, "judge_contest_testers", profileID)) {
		return true
	}

	// Everyone else needs the contest to be publicly visible first.
	if !contest.IsVisible {
		return false
	}

	// A visible contest with no privacy restrictions is open to all.
	if !contest.IsOrganizationPrivate && !contest.IsPrivate {
		return true
	}

	// Restricted contests require an authenticated member of every restriction
	// that applies (organization membership and/or the private-contestant list).
	if !authed {
		return false
	}
	if contest.IsOrganizationPrivate && !userInContestOrg(contest.ID, profileID) {
		return false
	}
	if contest.IsPrivate && !contestHasMember(contest.ID, "judge_contest_private_contestants", profileID) {
		return false
	}
	return true
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

// userIsOrgAdmin reports whether profileID is an administrator of the
// organization (membership in judge_organization_admins). joinTable columns
// are the Django defaults for Organization.admins.
func userIsOrgAdmin(orgID, profileID uint) bool {
	var count int64
	db.DB.Table("judge_organization_admins").
		Where("organization_id = ? AND profile_id = ?", orgID, profileID).
		Count(&count)
	return count > 0
}

// CanEditOrganization mirrors DMOJ organization-edit authority
// (OJ/judge/models/profile.py:81 + org mixins): superuser, OR
// judge.edit_all_organization, OR the judge.organization_admin bypass, OR being
// an administrator of this specific organization.
func CanEditOrganization(c *gin.Context, orgID uint) bool {
	access := GetAccess(c)
	if access.IsSuperuser && access.IsActive {
		return true
	}
	if access.HasPerm("judge.edit_all_organization") {
		return true
	}
	if access.HasPerm("judge.organization_admin") {
		return true
	}
	profileID, ok := CurrentProfileID(c)
	return ok && userIsOrgAdmin(orgID, profileID)
}
