package v2

import (
	"encoding/json"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm/clause"
)

// ============================================================
// Organization public tabs: /organization/:id/{problems,contests,submissions}
// and the member statistics on the detail endpoint. Reuses
// setupAdminUserCreateDB / seedAccount / authedRouter / doRequest.
// ============================================================

type orgTabsFixture struct {
	org, otherOrg models.Organization

	rootUser, memberUser, authorUser, outsiderUser models.AuthUser
	member, author, outsider                       models.Profile

	orgPublic, orgHidden, otherOrgProblem, loosePublic models.Problem

	orgPastContest, orgFutureContest, orgHiddenContest, otherOrgContest, runningContest models.Contest
}

// setupOrgTabsDB seeds one organization with a public and a hidden problem, a
// past / future / invisible org contest, a running scoreboard-hidden contest, a
// second organization and a couple of submissions.
func setupOrgTabsDB(t *testing.T) orgTabsFixture {
	t.Helper()
	setupAdminUserCreateDB(t)
	require.NoError(t, db.DB.AutoMigrate(
		&models.ProblemGroup{}, &models.Problem{}, &models.Contest{}, &models.ContestTag{},
		&models.Submission{},
	))
	// Not modelled in Go: the per-contest "can see contest submissions" list and
	// the testers-see-submissions flag (both exist in the Django schema).
	require.NoError(t, db.DB.Exec(
		"ALTER TABLE judge_contest ADD COLUMN tester_see_submissions numeric NOT NULL DEFAULT 0").Error)
	require.NoError(t, db.DB.Exec(
		"CREATE TABLE judge_contest_view_contest_submissions (contest_id integer NOT NULL, profile_id integer NOT NULL)").Error)

	rootUser, _ := seedAccount(t, "root", true, true)
	memberUser, member := seedAccount(t, "member", false, false)
	authorUser, author := seedAccount(t, "author", false, false)
	outsiderUser, outsider := seedAccount(t, "outsider", false, false)

	org := models.Organization{Name: "acme", Slug: "acme", ShortName: "ACME", CreationDate: time.Now()}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&org).Error)
	otherOrg := models.Organization{Name: "other", Slug: "other", ShortName: "OTHER", CreationDate: time.Now()}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&otherOrg).Error)
	require.NoError(t, db.DB.Model(&org).Association("Members").Append(&member, &author))

	group := models.ProblemGroup{Name: "g", FullName: "Group"}
	require.NoError(t, db.DB.Create(&group).Error)
	// Distinct, increasing dates keep -date (v1's default order) deterministic.
	problemDate := time.Now().Add(-24 * time.Hour)
	newProblem := func(code string, public bool) models.Problem {
		problemDate = problemDate.Add(time.Minute)
		p := models.Problem{
			Code: code, Name: code, GroupID: group.ID, TimeLimit: 1, MemoryLimit: 256,
			Points: 100, IsPublic: public, IsOrganizationPrivate: true, Date: &problemDate,
		}
		require.NoError(t, db.DB.Omit(clause.Associations).Create(&p).Error)
		return p
	}
	orgPublic := newProblem("orgpub", true)
	require.NoError(t, db.DB.Model(&orgPublic).Association("Organizations").Append(&org))
	orgHidden := newProblem("orghidden", false)
	require.NoError(t, db.DB.Model(&orgHidden).Association("Organizations").Append(&org))
	require.NoError(t, db.DB.Model(&orgHidden).Association("Authors").Append(&author))
	otherOrgProblem := newProblem("otherpub", true)
	require.NoError(t, db.DB.Model(&otherOrgProblem).Association("Organizations").Append(&otherOrg))
	loosePublic := models.Problem{
		Code: "loose", Name: "loose", GroupID: group.ID, TimeLimit: 1, MemoryLimit: 256,
		Points: 100, IsPublic: true,
	}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&loosePublic).Error)

	now := time.Now()
	newContest := func(key string, orgID uint, start, end time.Time, visible bool) models.Contest {
		ct := models.Contest{
			Key: key, Name: key, StartTime: start, EndTime: end, IsVisible: visible,
			IsOrganizationPrivate: true, FormatName: "default",
		}
		require.NoError(t, db.DB.Omit(clause.Associations).Create(&ct).Error)
		if orgID != 0 {
			var o models.Organization
			require.NoError(t, db.DB.First(&o, orgID).Error)
			require.NoError(t, db.DB.Model(&ct).Association("Organizations").Append(&o))
		}
		return ct
	}
	orgPastContest := newContest("orgpast", org.ID, now.Add(-3*time.Hour), now.Add(-time.Hour), true)
	orgFutureContest := newContest("orgfuture", org.ID, now.Add(time.Hour), now.Add(3*time.Hour), true)
	orgHiddenContest := newContest("orghidden", org.ID, now.Add(-3*time.Hour), now.Add(-time.Hour), false)
	otherOrgContest := newContest("otherpast", otherOrg.ID, now.Add(-3*time.Hour), now.Add(-time.Hour), true)
	runningContest := newContest("running", org.ID, now.Add(-time.Hour), now.Add(time.Hour), true)
	require.NoError(t, db.DB.Model(&runningContest).Update("scoreboard_visibility", "C").Error)

	newSubmission := func(p models.Problem, user models.Profile, contestID *uint) models.Submission {
		s := models.Submission{
			UserID: user.ID, ProblemID: p.ID, LanguageID: 1, Date: now,
			Status: "AC", ContestObjectID: contestID,
		}
		require.NoError(t, db.DB.Omit(clause.Associations).Create(&s).Error)
		return s
	}
	// s1 leaks nothing (no contest); s2 is hidden during a running "after
	// contest" scoreboard; s3 is in a visible-scoreboard contest; s4 belongs to
	// the other organization; s5 is the member's own submission in a hidden
	// scoreboard contest and must stay visible to them.
	newSubmission(orgPublic, outsider, nil)
	newSubmission(orgPublic, author, &runningContest.ID)
	newSubmission(orgPublic, author, &orgPastContest.ID)
	newSubmission(otherOrgProblem, author, nil)
	newSubmission(orgPublic, member, &runningContest.ID)

	return orgTabsFixture{
		org: org, otherOrg: otherOrg,
		rootUser: rootUser, memberUser: memberUser, authorUser: authorUser, outsiderUser: outsiderUser,
		member: member, author: author, outsider: outsider,
		orgPublic: orgPublic, orgHidden: orgHidden, otherOrgProblem: otherOrgProblem, loosePublic: loosePublic,
		orgPastContest: orgPastContest, orgFutureContest: orgFutureContest,
		orgHiddenContest: orgHiddenContest, otherOrgContest: otherOrgContest, runningContest: runningContest,
	}
}

func orgTabRequest(t *testing.T, orgID, userID uint, tab string) (int, map[string]any) {
	t.Helper()
	handler := map[string]gin.HandlerFunc{
		"problems":    OrganizationProblemList,
		"contests":    OrganizationContestList,
		"submissions": OrganizationSubmissionList,
	}[tab]
	var router *gin.Engine
	if userID == 0 {
		gin.SetMode(gin.TestMode)
		router = gin.New()
		router.GET("/organization/:id/"+tab, handler)
	} else {
		router = authedRouter(userID, http.MethodGet, "/organization/:id/"+tab, handler)
	}

	target := "/organization/" + strconv.Itoa(int(orgID)) + "/" + tab
	w := doRequest(router, http.MethodGet, target, "")
	var body map[string]any
	if w.Code == http.StatusOK {
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	}
	return w.Code, body
}

func orgCodes(t *testing.T, body map[string]any, field string) []string {
	t.Helper()
	rows, ok := body["data"].([]any)
	require.True(t, ok)
	out := make([]string, len(rows))
	for i, r := range rows {
		out[i] = r.(map[string]any)[field].(string)
	}
	return out
}

// Non-members never get the private lists (v1 CustomOrganizationMixin.dispatch),
// and a missing organization is a 404 rather than a 403.
func TestOrganizationTabs_MembershipGate(t *testing.T) {
	fx := setupOrgTabsDB(t)

	for _, tab := range []string{"problems", "contests", "submissions"} {
		code, _ := orgTabRequest(t, fx.org.ID, 0, tab)
		require.Equal(t, http.StatusForbidden, code, "%s must reject anonymous viewers", tab)

		code, _ = orgTabRequest(t, fx.org.ID, fx.outsiderUser.ID, tab)
		require.Equal(t, http.StatusForbidden, code, "%s must reject non-members", tab)
	}

	code, _ := orgTabRequest(t, 999, fx.memberUser.ID, "problems")
	require.Equal(t, http.StatusNotFound, code)

	code, _ = orgTabRequest(t, fx.org.ID, fx.memberUser.ID, "problems")
	require.Equal(t, http.StatusOK, code)
}

// v1 ProblemListOrganization.get_filter: members only see the organization's
// public problems; hidden ones appear only for their editors or for holders of
// judge.see_private_problem (superusers).
func TestOrganizationProblems_Scoping(t *testing.T) {
	fx := setupOrgTabsDB(t)

	_, body := orgTabRequest(t, fx.org.ID, fx.memberUser.ID, "problems")
	require.Equal(t, []string{"orgpub"}, orgCodes(t, body, "code"))
	require.EqualValues(t, 1, body["total"])

	_, body = orgTabRequest(t, fx.org.ID, fx.authorUser.ID, "problems")
	require.Equal(t, []string{"orghidden", "orgpub"}, orgCodes(t, body, "code"))
	require.EqualValues(t, 2, body["total"])

	_, body = orgTabRequest(t, fx.org.ID, fx.rootUser.ID, "problems")
	require.EqualValues(t, 2, body["total"])

	// The total counts every matching row, not just the page.
	gin.SetMode(gin.TestMode)
	router := authedRouter(fx.authorUser.ID, http.MethodGet, "/organization/:id/problems", OrganizationProblemList)
	w := doRequest(router, http.MethodGet,
		"/organization/"+strconv.Itoa(int(fx.org.ID))+"/problems?page=1&page_size=1", "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var paged map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &paged))
	require.EqualValues(t, 2, paged["total"])
	require.Len(t, paged["data"].([]any), 1)
}

// v1 ContestListOrganization: visible org contests that already ended, nothing
// from other organizations and nothing upcoming; privileged viewers also see
// invisible ones.
func TestOrganizationContests_Scoping(t *testing.T) {
	fx := setupOrgTabsDB(t)

	_, body := orgTabRequest(t, fx.org.ID, fx.memberUser.ID, "contests")
	require.Equal(t, []string{"orgpast"}, orgCodes(t, body, "key"))
	require.EqualValues(t, 1, body["total"])

	_, body = orgTabRequest(t, fx.org.ID, fx.rootUser.ID, "contests")
	require.ElementsMatch(t, []string{"orgpast", "orghidden"}, orgCodes(t, body, "key"))
	require.EqualValues(t, 2, body["total"])
}

// v1 SubmissionListOrganization + SubmissionsListBase: org problems only, and a
// non-privileged member still loses in-contest submissions that the contest's
// scoreboard hides during the contest (their own stay visible).
func TestOrganizationSubmissions_Scoping(t *testing.T) {
	fx := setupOrgTabsDB(t)

	code, body := orgTabRequest(t, fx.org.ID, fx.memberUser.ID, "submissions")
	require.Equal(t, http.StatusOK, code)
	rows := body["data"].([]any)
	require.EqualValues(t, 3, body["total"], "outsider, visible-scoreboard contest, and own hidden-contest row")
	for _, r := range rows {
		require.Equal(t, "orgpub", r.(map[string]any)["problem"])
	}

	_, body = orgTabRequest(t, fx.org.ID, fx.authorUser.ID, "submissions")
	require.EqualValues(t, 3, body["total"], "gains their own running-contest row but loses the member's")

	// A viewer who may see private contests gets the unfiltered list, but the
	// organization constraint still applies (the other organization's
	// submission stays out).
	_, body = orgTabRequest(t, fx.org.ID, fx.rootUser.ID, "submissions")
	require.EqualValues(t, 4, body["total"])
}

// v1 SubmissionListOrganization inherits Problem.get_visible_problems, so a
// hidden organization problem's submissions stay out of a plain member's list
// while its author still sees them.
func TestOrganizationSubmissions_HiddenProblem(t *testing.T) {
	fx := setupOrgTabsDB(t)
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&models.Submission{
		UserID: fx.author.ID, ProblemID: fx.orgHidden.ID, LanguageID: 1, Date: time.Now(), Status: "AC",
	}).Error)

	_, body := orgTabRequest(t, fx.org.ID, fx.memberUser.ID, "submissions")
	require.EqualValues(t, 3, body["total"], "the hidden problem's row must not leak")
	for _, r := range body["data"].([]any) {
		require.Equal(t, "orgpub", r.(map[string]any)["problem"])
	}

	_, body = orgTabRequest(t, fx.org.ID, fx.authorUser.ID, "submissions")
	require.EqualValues(t, 4, body["total"])
	var hiddenRows int
	for _, r := range body["data"].([]any) {
		if r.(map[string]any)["problem"] == "orghidden" {
			hiddenRows++
		}
	}
	require.Equal(t, 1, hiddenRows, "the problem's author still sees it")
}

// v1 OrganizationUsers hides unlisted profiles from the member table but keeps
// the organization's stored member_count.
func TestOrganizationDetail_UnlistedMembers(t *testing.T) {
	fx := setupOrgTabsDB(t)
	require.NoError(t, db.DB.Model(&models.Profile{}).
		Where("id = ?", fx.author.ID).Update("is_unlisted", true).Error)
	require.NoError(t, db.DB.Model(&models.Organization{}).
		Where("id = ?", fx.org.ID).Update("member_count", 2).Error)

	router := authedRouter(fx.memberUser.ID, http.MethodGet, "/organization/:id", OrganizationDetail)
	w := doRequest(router, http.MethodGet, "/organization/"+strconv.Itoa(int(fx.org.ID)), "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var body struct {
		Members     []map[string]any `json:"members"`
		MemberCount int              `json:"member_count"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.EqualValues(t, 2, body.MemberCount)
	require.Len(t, body.Members, 1)
	require.Equal(t, "member", body.Members[0]["username"])
}

// The problems tab accepts v1's sort allow-list (default -date) and ignores
// unknown columns.
func TestOrganizationProblems_Sort(t *testing.T) {
	fx := setupOrgTabsDB(t)

	problemCodes := func(query string) []string {
		router := authedRouter(fx.authorUser.ID, http.MethodGet, "/organization/:id/problems", OrganizationProblemList)
		w := doRequest(router, http.MethodGet,
			"/organization/"+strconv.Itoa(int(fx.org.ID))+"/problems"+query, "")
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())
		var body map[string]any
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
		return orgCodes(t, body, "code")
	}

	require.Equal(t, []string{"orgpub", "orghidden"}, problemCodes("?sort=name&order=desc"))
	require.Equal(t, []string{"orghidden", "orgpub"}, problemCodes("?sort=name&order=asc"))
	require.Equal(t, []string{"orghidden", "orgpub"}, problemCodes("?sort=password"))
}

// The detail endpoint carries the member stats (v1 OrganizationUsers columns)
// and orders them like v1: -performance_points by default, all four sortable.
func TestOrganizationDetail_MemberStats(t *testing.T) {
	fx := setupOrgTabsDB(t)

	require.NoError(t, db.DB.Model(&models.Profile{}).Where("id = ?", fx.member.ID).Updates(map[string]any{
		"points": 10.0, "problem_count": 2, "rating": 1500, "performance_points": 20.0,
	}).Error)
	require.NoError(t, db.DB.Model(&models.Profile{}).Where("id = ?", fx.author.ID).Updates(map[string]any{
		"points": 30.0, "problem_count": 5, "rating": 1800, "performance_points": 40.0,
	}).Error)

	members := func(query string) []map[string]any {
		router := authedRouter(fx.memberUser.ID, http.MethodGet, "/organization/:id", OrganizationDetail)
		w := doRequest(router, http.MethodGet, "/organization/"+strconv.Itoa(int(fx.org.ID))+query, "")
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())
		var body struct {
			Members []map[string]any `json:"members"`
			Admins  []map[string]any `json:"admins"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
		return body.Members
	}

	rows := members("")
	require.Len(t, rows, 2)
	require.Equal(t, "author", rows[0]["username"], "default sort is -performance_points")
	require.EqualValues(t, 30, rows[0]["points"])
	require.EqualValues(t, 5, rows[0]["problem_count"])
	require.EqualValues(t, 1800, rows[0]["rating"])
	require.EqualValues(t, 40, rows[0]["performance_points"])
	require.Equal(t, "member", rows[0]["role"])
	require.Equal(t, "member", rows[1]["username"])

	rows = members("?sort=points&order=asc")
	require.Equal(t, "member", rows[0]["username"], "ascending points puts the lower score first")

	rows = members("?sort=problem_count&order=desc")
	require.Equal(t, "author", rows[0]["username"])

	// An unknown sort column falls back to the default instead of erroring.
	rows = members("?sort=password")
	require.Equal(t, "author", rows[0]["username"])
}
