// claoj/auth/problem_visibility.go
package auth

import (
	"github.com/CLAOJ/claoj/db"
	"github.com/gin-gonic/gin"
)

// VisibleProblemFilter returns a parenthesized SQL boolean expression (and its
// bind arguments) restricting judge_problem rows to those the request user is
// allowed to see. It mirrors Problem.get_visible_problems
// (OJ/judge/models/problem.py:309-348).
//
// The surrounding query must expose the problem table as "judge_problem".
// Use VisibleProblemFilterFor when it is aliased. The result is parenthesized,
// so it is safe to AND with other conditions.
func VisibleProblemFilter(c *gin.Context) (string, []any) {
	return VisibleProblemFilterFor(c, "judge_problem")
}

// VisibleProblemFilterFor is VisibleProblemFilter with an explicit name for the
// judge_problem table/alias used by the surrounding query (e.g. "pr" when the
// query does JOIN judge_problem pr ...).
func VisibleProblemFilterFor(c *gin.Context, t string) (string, []any) {
	access := GetAccess(c)
	// Users who may see every problem, including non-public ones.
	if (access.IsSuperuser && access.IsActive) ||
		access.HasPerm("judge.see_private_problem") ||
		access.HasPerm("judge.edit_all_problem") {
		return "(1 = 1)", nil
	}

	authorCuratorTester := "EXISTS (SELECT 1 FROM judge_problem_authors a WHERE a.problem_id = " + t + ".id AND a.profile_id = ?) " +
		"OR EXISTS (SELECT 1 FROM judge_problem_curators cu WHERE cu.problem_id = " + t + ".id AND cu.profile_id = ?) " +
		"OR EXISTS (SELECT 1 FROM judge_problem_testers te WHERE te.problem_id = " + t + ".id AND te.profile_id = ?)"

	profileID, ok := CurrentProfileID(c)
	if !ok {
		// Anonymous users only see fully public problems.
		return "(" + t + ".is_public = ? AND " + t + ".is_organization_private = ?)",
			[]any{true, false}
	}

	// Users who may see all organization-private problems (or edit public ones)
	// are not constrained by organization membership; everyone else must be a
	// member of one of an organization-private problem's organizations.
	var publicPart string
	args := []any{}
	if access.HasPerm("judge.see_organization_problem") || access.HasPerm("judge.edit_public_problem") {
		publicPart = t + ".is_public = ?"
		args = append(args, true)
	} else {
		orgMember := "EXISTS (SELECT 1 FROM judge_problem_organizations po " +
			"JOIN judge_profile_organizations pu ON pu.organization_id = po.organization_id " +
			"WHERE po.problem_id = " + t + ".id AND pu.profile_id = ?)"
		publicPart = t + ".is_public = ? AND (" +
			t + ".is_organization_private = ? OR (" + t + ".is_organization_private = ? AND " + orgMember + "))"
		args = append(args, true, false, true, profileID)
	}

	// Authors, curators, and testers always see their own problems (even hidden).
	expr := "((" + publicPart + ") OR " + authorCuratorTester + ")"
	args = append(args, profileID, profileID, profileID)
	return expr, args
}

// userInProblemOrg reports whether the profile belongs to any organization the
// problem is private to (judge_problem_organizations ∩ the user's
// judge_profile_organizations).
func userInProblemOrg(problemID, profileID uint) bool {
	var count int64
	db.DB.Table("judge_problem_organizations AS po").
		Joins("JOIN judge_profile_organizations AS pu ON pu.organization_id = po.organization_id").
		Where("po.problem_id = ? AND pu.profile_id = ?", problemID, profileID).
		Count(&count)
	return count > 0
}
