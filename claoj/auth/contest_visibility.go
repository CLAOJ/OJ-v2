// claoj/auth/contest_visibility.go
package auth

import (
	"github.com/CLAOJ/claoj/db"
	"github.com/gin-gonic/gin"
)

// VisibleContestFilter returns a parenthesized SQL boolean expression (and its
// bind arguments) restricting judge_contest rows to those the request user is
// allowed to see. It mirrors Contest.get_visible_contests
// (OJ/judge/models/contest.py:391-417).
//
// The surrounding query must expose the contest table as "judge_contest"
// (e.g. db.DB.Model(&models.Contest{}) or "... FROM judge_contest ..."). When
// the contest table is aliased, use VisibleContestFilterFor instead. The result
// is wrapped in parentheses, so it is safe to AND with other conditions.
func VisibleContestFilter(c *gin.Context) (string, []any) {
	return VisibleContestFilterFor(c, "judge_contest")
}

// VisibleContestFilterFor is VisibleContestFilter with an explicit name for the
// judge_contest table/alias used by the surrounding query (e.g. "jc" when the
// query does JOIN judge_contest jc ...). Column references and the correlated
// subqueries are qualified with t, so t must match the table/alias in scope.
func VisibleContestFilterFor(c *gin.Context, t string) (string, []any) {
	access := GetAccess(c)
	if (access.IsSuperuser && access.IsActive) ||
		access.HasPerm("judge.see_private_contest") ||
		access.HasPerm("judge.edit_all_contest") {
		return "(1 = 1)", nil
	}

	// Correlated EXISTS fragments, each referencing the outer contest row by
	// t.id and binding a single judge_profile id argument.
	orgMember := "EXISTS (SELECT 1 FROM judge_contest_organizations co " +
		"JOIN judge_profile_organizations po ON po.organization_id = co.organization_id " +
		"WHERE co.contest_id = " + t + ".id AND po.profile_id = ?)"
	privateContestant := "EXISTS (SELECT 1 FROM judge_contest_private_contestants pc " +
		"WHERE pc.contest_id = " + t + ".id AND pc.profile_id = ?)"
	editorOrTester := "EXISTS (SELECT 1 FROM judge_contest_authors a WHERE a.contest_id = " + t + ".id AND a.profile_id = ?) " +
		"OR EXISTS (SELECT 1 FROM judge_contest_curators cu WHERE cu.contest_id = " + t + ".id AND cu.profile_id = ?) " +
		"OR EXISTS (SELECT 1 FROM judge_contest_testers te WHERE te.contest_id = " + t + ".id AND te.profile_id = ?)"

	profileID, ok := CurrentProfileID(c)
	if !ok {
		// Anonymous users only see visible, fully public contests.
		return "(" + t + ".is_visible = ? AND " + t + ".is_organization_private = ? AND " + t + ".is_private = ?)",
			[]any{true, false, false}
	}

	// Authenticated, non-privileged user: mirror the Q(...) disjunction of
	// get_visible_contests. A visible contest is shown when it is fully public,
	// or the user satisfies the private / organization membership it requires;
	// additionally, authors/curators/testers always see their own contests
	// (even while hidden).
	expr := "(" +
		"(" + t + ".is_visible = ? AND (" +
		"(" + t + ".is_organization_private = ? AND " + t + ".is_private = ?) " +
		"OR (" + t + ".is_organization_private = ? AND " + t + ".is_private = ? AND " + privateContestant + ") " +
		"OR (" + t + ".is_organization_private = ? AND " + t + ".is_private = ? AND " + orgMember + ") " +
		"OR (" + t + ".is_organization_private = ? AND " + t + ".is_private = ? AND " + orgMember + " AND " + privateContestant + ")" +
		")) " +
		"OR " + editorOrTester +
		")"

	args := []any{
		true,                             // is_visible
		false, false,                     // fully public
		false, true, profileID,           // private only (+ private contestant)
		true, false, profileID,           // organization private only (+ org member)
		true, true, profileID, profileID, // both (+ org member, + private contestant)
		profileID, profileID, profileID,  // author / curator / tester
	}
	return expr, args
}

// contestHasMember reports whether profileID appears in one of a contest's
// profile join tables (judge_contest_authors / _curators / _testers /
// _private_contestants). joinTable must be a trusted constant, never user input.
func contestHasMember(contestID uint, joinTable string, profileID uint) bool {
	var count int64
	db.DB.Table(joinTable).
		Where("contest_id = ? AND profile_id = ?", contestID, profileID).
		Count(&count)
	return count > 0
}

// userInContestOrg reports whether the profile belongs to any organization the
// contest is private to (judge_contest_organizations ∩ the user's
// judge_profile_organizations).
func userInContestOrg(contestID, profileID uint) bool {
	var count int64
	db.DB.Table("judge_contest_organizations AS co").
		Joins("JOIN judge_profile_organizations AS po ON po.organization_id = co.organization_id").
		Where("co.contest_id = ? AND po.profile_id = ?", contestID, profileID).
		Count(&count)
	return count > 0
}
