package v2

import (
	"encoding/json"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm/clause"
)

// ============================================================
// Contest submission surfaces: ?contest= on /submissions, the
// per-problem contest ranking, contest leave/join and owner abort.
// ============================================================

func setupContestSurfacesDB(t *testing.T) {
	t.Helper()
	setupAdminSubmissionDB(t)
	// AutoMigrate(&Contest{}) creates judge_contestproblem as a bare many2many
	// join table, and GORM will not add columns to it afterwards. Recreate it
	// as the real model table in this empty test database.
	require.NoError(t, db.DB.Migrator().DropTable(&models.ContestProblem{}))
	require.NoError(t, db.DB.Migrator().CreateTable(&models.ContestProblem{}))
}

type contestSurfacesFixture struct {
	contest models.Contest
	problem models.Problem
	cp      models.ContestProblem
	lang    models.Language
	alice   models.AuthUser
	aliceP  models.Profile
	bob     models.AuthUser
	bobP    models.Profile
	carol   models.AuthUser
	carolP  models.Profile
}

func seedContestSurfaces(t *testing.T) contestSurfacesFixture {
	t.Helper()
	alice, aliceP := seedAccount(t, "alice", false, false)
	bob, bobP := seedAccount(t, "bob", false, false)
	carol, carolP := seedAccount(t, "carol", false, false)

	group := models.ProblemGroup{Name: "g", FullName: "Group"}
	require.NoError(t, db.DB.Create(&group).Error)
	problem := models.Problem{Code: "p1", Name: "Problem One", GroupID: group.ID, IsPublic: true, TimeLimit: 1, MemoryLimit: 256}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&problem).Error)
	lang := models.Language{Key: "CPP17", Name: "C++17", Extension: "cpp"}
	require.NoError(t, db.DB.Create(&lang).Error)

	now := time.Now()
	contest := models.Contest{
		Key: "c1", Name: "Contest One", Description: "d", Summary: "s",
		StartTime: now.Add(-time.Hour), EndTime: now.Add(time.Hour),
		IsVisible: true, ScoreboardVisibility: "V",
	}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&contest).Error)
	cp := models.ContestProblem{ContestID: contest.ID, ProblemID: problem.ID, Points: 100, Partial: true, Order: 0}
	require.NoError(t, db.DB.Create(&cp).Error)

	return contestSurfacesFixture{
		contest: contest, problem: problem, cp: cp, lang: lang,
		alice: alice, aliceP: aliceP, bob: bob, bobP: bobP, carol: carol, carolP: carolP,
	}
}

// participate creates a live participation for profile and returns it.
func participate(t *testing.T, contestID, profileID uint) models.ContestParticipation {
	t.Helper()
	p := models.ContestParticipation{ContestID: contestID, UserID: profileID, RealStart: time.Now()}
	require.NoError(t, db.DB.Omit("Contest", "User").Create(&p).Error)
	return p
}

// contestSubmission seeds a submission and its judge_contestsubmission link.
func contestSubmission(t *testing.T, profileID, problemID, langID, partID, cpID uint,
	points, runtime float64, result string, archived bool) models.Submission {
	t.Helper()
	r := result
	p := points
	tm := runtime
	sub := models.Submission{
		UserID: profileID, ProblemID: problemID, LanguageID: langID,
		Date: time.Now(), Status: "D", Result: &r, Points: &p, Time: &tm,
		IsArchived: archived,
	}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&sub).Error)
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&models.ContestSubmission{
		SubmissionID: sub.ID, ProblemID: cpID, ParticipationID: partID, Points: points,
	}).Error)
	return sub
}

func plainSubmission(t *testing.T, profileID, problemID, langID uint, result string) models.Submission {
	t.Helper()
	r := result
	s := models.Submission{
		UserID: profileID, ProblemID: problemID, LanguageID: langID,
		Date: time.Now(), Status: "D", Result: &r,
	}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&s).Error)
	return s
}

// submissionListAs calls /submissions as userID (0 = anonymous).
func submissionListAs(t *testing.T, userID uint, query string) (int, []map[string]any, float64) {
	t.Helper()
	router := authedRouter(userID, http.MethodGet, "/submissions", SubmissionList)
	w := doRequest(router, http.MethodGet, "/submissions"+query, "")
	if w.Code != http.StatusOK {
		return w.Code, nil, 0
	}
	var body struct {
		Data  []map[string]any `json:"data"`
		Total float64          `json:"total"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	return w.Code, body.Data, body.Total
}

// TestSubmissionList_ContestFilter_AccessMatrix covers v1's in-contest
// submission lists: only that contest's submissions, the whole set once the
// scoreboard is visible, own rows only while it is withheld, and 404 before
// the contest starts unless the viewer may see private contests.
func TestSubmissionList_ContestFilter_AccessMatrix(t *testing.T) {
	setupContestSurfacesDB(t)
	f := seedContestSurfaces(t)
	alicePart := participate(t, f.contest.ID, f.aliceP.ID)
	bobPart := participate(t, f.contest.ID, f.bobP.ID)

	contestSubmission(t, f.aliceP.ID, f.problem.ID, f.lang.ID, alicePart.ID, f.cp.ID, 100, 1, "AC", false)
	contestSubmission(t, f.aliceP.ID, f.problem.ID, f.lang.ID, alicePart.ID, f.cp.ID, 50, 2, "WA", false)
	contestSubmission(t, f.bobP.ID, f.problem.ID, f.lang.ID, bobPart.ID, f.cp.ID, 100, 1, "AC", false)
	// Carol never joined: her submission is not a contest submission and must
	// stay out of the filtered list.
	plainSubmission(t, f.carolP.ID, f.problem.ID, f.lang.ID, "AC")

	t.Run("scoreboard visible shows every contest submission", func(t *testing.T) {
		code, rows, total := submissionListAs(t, 0, "?contest=c1")
		require.Equal(t, http.StatusOK, code)
		require.Equal(t, 3.0, total)
		require.Len(t, rows, 3)
	})

	t.Run("plain list keeps non-contest rows", func(t *testing.T) {
		_, _, total := submissionListAs(t, 0, "?problem=p1")
		require.Equal(t, 4.0, total)
	})

	t.Run("other filters still apply", func(t *testing.T) {
		_, rows, total := submissionListAs(t, 0, "?contest=c1&result=AC")
		require.Equal(t, 2.0, total)
		for _, r := range rows {
			require.Equal(t, "AC", r["result"])
		}
		_, _, total = submissionListAs(t, 0, "?contest=c1&user=alice")
		require.Equal(t, 2.0, total)
		_, _, total = submissionListAs(t, 0, "?contest=c1&problem=p1&best=1")
		require.Equal(t, 2.0, total, "best mode dedupes per user inside the contest")
	})

	t.Run("hidden scoreboard restricts everyone to their own rows", func(t *testing.T) {
		require.NoError(t, db.DB.Model(&models.Contest{}).Where("id = ?", f.contest.ID).
			Update("scoreboard_visibility", "C").Error)

		_, _, total := submissionListAs(t, 0, "?contest=c1")
		require.Equal(t, 0.0, total, "anonymous visitors see nothing")

		_, _, total = submissionListAs(t, f.alice.ID, "?contest=c1")
		require.Equal(t, 2.0, total)

		_, _, total = submissionListAs(t, f.bob.ID, "?contest=c1")
		require.Equal(t, 1.0, total)

		_, _, total = submissionListAs(t, f.carol.ID, "?contest=c1")
		require.Equal(t, 0.0, total, "non-participants see no peer rows")

		// Privileged viewers bypass the hidden scoreboard (v1 permission).
		judge, _ := seedAccount(t, "judge", false, false)
		grantContestPerm(t, judge.ID, "see_private_contest")
		_, _, total = submissionListAs(t, judge.ID, "?contest=c1")
		require.Equal(t, 3.0, total)
	})

	t.Run("not started hides the list unless privileged", func(t *testing.T) {
		require.NoError(t, db.DB.Model(&models.Contest{}).Where("id = ?", f.contest.ID).
			Update("start_time", time.Now().Add(2*time.Hour)).Error)

		code, _, _ := submissionListAs(t, 0, "?contest=c1")
		require.Equal(t, http.StatusNotFound, code)

		privileged, _ := seedAccount(t, "privileged", false, false)
		grantContestPerm(t, privileged.ID, "see_private_contest")
		code, _, _ = submissionListAs(t, privileged.ID, "?contest=c1")
		require.Equal(t, http.StatusOK, code)
	})

	t.Run("unknown contest is 404", func(t *testing.T) {
		code, _, _ := submissionListAs(t, 0, "?contest=nope")
		require.Equal(t, http.StatusNotFound, code)
	})
}

type problemRankPayload struct {
	Contest          string           `json:"contest"`
	Problem          map[string]any   `json:"problem"`
	Data             []map[string]any `json:"data"`
	Total            float64          `json:"total"`
	ScoreboardHidden bool             `json:"scoreboard_hidden"`
}

func getProblemRank(t *testing.T, userID uint, target string) (int, problemRankPayload) {
	t.Helper()
	router := authedRouter(userID, http.MethodGet, "/contest/:key/rank/:problem", ContestProblemRank)
	w := doRequest(router, http.MethodGet, target, "")
	var body problemRankPayload
	if w.Code == http.StatusOK {
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	}
	return w.Code, body
}

// TestContestProblemRank_OrderAndExclusions covers v1 ContestRankedSubmission:
// best row per user by contest points then runtime, positive points only,
// unlisted/archived/disqualified rows out.
func TestContestProblemRank_OrderAndExclusions(t *testing.T) {
	setupContestSurfacesDB(t)
	f := seedContestSurfaces(t)
	alicePart := participate(t, f.contest.ID, f.aliceP.ID)
	bobPart := participate(t, f.contest.ID, f.bobP.ID)
	_, daveP := seedAccount(t, "dave", false, false)
	davePart := participate(t, f.contest.ID, daveP.ID)
	_, frankP := seedAccount(t, "frank", false, false)
	frankPart := participate(t, f.contest.ID, frankP.ID)
	carolPart := participate(t, f.contest.ID, f.carolP.ID)
	require.NoError(t, db.DB.Model(&models.Profile{}).Where("id = ?", f.carolP.ID).
		Update("is_unlisted", true).Error)

	contestSubmission(t, f.aliceP.ID, f.problem.ID, f.lang.ID, alicePart.ID, f.cp.ID, 50, 1, "AC", false)
	contestSubmission(t, f.aliceP.ID, f.problem.ID, f.lang.ID, alicePart.ID, f.cp.ID, 100, 3, "AC", false)
	bestAlice := contestSubmission(t, f.aliceP.ID, f.problem.ID, f.lang.ID, alicePart.ID, f.cp.ID, 100, 2, "AC", false)
	bestBob := contestSubmission(t, f.bobP.ID, f.problem.ID, f.lang.ID, bobPart.ID, f.cp.ID, 100, 1.5, "AC", false)
	// Slower tied row for Bob: loses the per-user dedupe.
	contestSubmission(t, f.bobP.ID, f.problem.ID, f.lang.ID, bobPart.ID, f.cp.ID, 100, 2.5, "AC", false)
	// Excluded: zero points, unlisted profile, archived submission.
	contestSubmission(t, daveP.ID, f.problem.ID, f.lang.ID, davePart.ID, f.cp.ID, 0, 0.5, "WA", false)
	contestSubmission(t, f.carolP.ID, f.problem.ID, f.lang.ID, carolPart.ID, f.cp.ID, 100, 0.5, "AC", false)
	contestSubmission(t, frankP.ID, f.problem.ID, f.lang.ID, frankPart.ID, f.cp.ID, 100, 0.4, "AC", true)

	// A submission to the same problem from another contest must not leak in.
	other := models.Contest{Key: "c2", Name: "Other", Description: "d", StartTime: time.Now().Add(-time.Hour), EndTime: time.Now().Add(time.Hour), IsVisible: true}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&other).Error)
	c2p := models.ContestProblem{ContestID: other.ID, ProblemID: f.problem.ID, Points: 100, Partial: true}
	require.NoError(t, db.DB.Create(&c2p).Error)
	otherPart := participate(t, other.ID, f.aliceP.ID)
	contestSubmission(t, f.aliceP.ID, f.problem.ID, f.lang.ID, otherPart.ID, c2p.ID, 100, 0.1, "AC", false)

	code, body := getProblemRank(t, 0, "/contest/c1/rank/p1")
	require.Equal(t, http.StatusOK, code)
	require.False(t, body.ScoreboardHidden)
	require.Equal(t, "p1", body.Problem["code"])
	require.Equal(t, "1", body.Problem["label"])
	require.Equal(t, "Problem One", body.Problem["name"])
	require.Equal(t, 2.0, body.Total)
	require.Len(t, body.Data, 2)
	require.Equal(t, "bob", body.Data[0]["user"])
	require.Equal(t, float64(bestBob.ID), body.Data[0]["id"])
	require.Equal(t, float64(1), body.Data[0]["rank"])
	require.Equal(t, 100.0, body.Data[0]["points"])
	require.Equal(t, "alice", body.Data[1]["user"])
	require.Equal(t, float64(bestAlice.ID), body.Data[1]["id"])
	require.Equal(t, float64(2), body.Data[1]["rank"])

	t.Run("unknown or foreign problem is 404", func(t *testing.T) {
		code, _ := getProblemRank(t, 0, "/contest/c1/rank/nope")
		require.Equal(t, http.StatusNotFound, code)

		foreign := models.Problem{Code: "p9", Name: "Foreign", GroupID: f.problem.GroupID, IsPublic: true}
		require.NoError(t, db.DB.Omit(clause.Associations).Create(&foreign).Error)
		code, _ = getProblemRank(t, 0, "/contest/c1/rank/p9")
		require.Equal(t, http.StatusNotFound, code)
	})

	t.Run("hidden scoreboard hides peers and keeps own rows", func(t *testing.T) {
		require.NoError(t, db.DB.Model(&models.Contest{}).Where("id = ?", f.contest.ID).
			Update("scoreboard_visibility", "C").Error)

		_, body := getProblemRank(t, 0, "/contest/c1/rank/p1")
		require.True(t, body.ScoreboardHidden)
		require.Empty(t, body.Data)

		_, body = getProblemRank(t, f.bob.ID, "/contest/c1/rank/p1")
		require.False(t, body.ScoreboardHidden)
		require.Len(t, body.Data, 1)
		require.Equal(t, "bob", body.Data[0]["user"])

		nobody, _ := seedAccount(t, "nobody", false, false)
		_, hiddenForNonParticipant := getProblemRank(t, nobody.ID, "/contest/c1/rank/p1")
		require.True(t, hiddenForNonParticipant.ScoreboardHidden, "non-participants get nothing")
	})

	t.Run("not started is 404 for anonymous", func(t *testing.T) {
		require.NoError(t, db.DB.Model(&models.Contest{}).Where("id = ?", f.contest.ID).
			Update("start_time", time.Now().Add(2*time.Hour)).Error)
		code, _ := getProblemRank(t, 0, "/contest/c1/rank/p1")
		require.Equal(t, http.StatusNotFound, code)
	})
}

// TestContestLeave detaches the current contest pointer without destroying the
// participation (v1 ContestLeave + Profile.remove_contest).
func TestContestLeave(t *testing.T) {
	setupContestSurfacesDB(t)
	f := seedContestSurfaces(t)
	part := participate(t, f.contest.ID, f.aliceP.ID)
	require.NoError(t, db.DB.Model(&models.Profile{}).Where("id = ?", f.aliceP.ID).
		Update("current_contest_id", part.ID).Error)

	router := authedRouter(f.alice.ID, http.MethodPost, "/contest/:key/leave", ContestLeave)
	w := doRequest(router, http.MethodPost, "/contest/c1/leave", "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var profile models.Profile
	require.NoError(t, db.DB.First(&profile, f.aliceP.ID).Error)
	require.Nil(t, profile.CurrentContestID, "leave clears the pointer")

	var participationCount int64
	db.DB.Model(&models.ContestParticipation{}).
		Where("id = ? AND contest_id = ?", part.ID, f.contest.ID).Count(&participationCount)
	require.Equal(t, int64(1), participationCount, "the scoreboard row is kept")

	// Already out: 404 (v1 "You are not in contest").
	w = doRequest(router, http.MethodPost, "/contest/c1/leave", "")
	require.Equal(t, http.StatusNotFound, w.Code)

	// A pointer into another contest does not let the user leave this one.
	other := models.Contest{Key: "c2", Name: "Other", Description: "d", StartTime: time.Now().Add(-time.Hour), EndTime: time.Now().Add(time.Hour), IsVisible: true}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&other).Error)
	otherPart := participate(t, other.ID, f.aliceP.ID)
	require.NoError(t, db.DB.Model(&models.Profile{}).Where("id = ?", f.aliceP.ID).
		Update("current_contest_id", otherPart.ID).Error)
	w = doRequest(router, http.MethodPost, "/contest/c1/leave", "")
	require.Equal(t, http.StatusNotFound, w.Code)
}

// TestContestJoinResumesParticipation: v1's join re-points current_contest at
// an existing participation instead of refusing the re-join.
func TestContestJoinResumesParticipation(t *testing.T) {
	setupContestSurfacesDB(t)
	f := seedContestSurfaces(t)
	part := participate(t, f.contest.ID, f.aliceP.ID)

	router := authedRouter(f.alice.ID, http.MethodPost, "/contest/:key/join", ContestJoin)
	w := doRequest(router, http.MethodPost, "/contest/c1/join", "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var profile models.Profile
	require.NoError(t, db.DB.First(&profile, f.aliceP.ID).Error)
	require.NotNil(t, profile.CurrentContestID)
	require.Equal(t, part.ID, *profile.CurrentContestID)

	var count int64
	db.DB.Model(&models.ContestParticipation{}).Where("contest_id = ?", f.contest.ID).Count(&count)
	require.Equal(t, int64(1), count, "re-join must not create a second participation")
}

// fakeAbortBridge records abort signals sent to the judge.
type fakeAbortBridge struct {
	ids []uint
}

func (f *fakeAbortBridge) Abort(subID uint) error {
	f.ids = append(f.ids, subID)
	return nil
}

// TestSubmissionAbort_Ownership covers v1 abort_submission: the owner may abort
// their queued submission, nobody else may, and a rejudged submission is only
// abortable by judge.abort_any_submission holders.
func TestSubmissionAbort_Ownership(t *testing.T) {
	setupAdminSubmissionDB(t)
	bridge := &fakeAbortBridge{}
	oldBridge := bridgeServerRef
	bridgeServerRef = bridge
	t.Cleanup(func() { bridgeServerRef = oldBridge; submissionService = nil })

	owner, ownerP := seedAccount(t, "owner", false, false)
	stranger, _ := seedAccount(t, "stranger", false, false)
	staff, _ := seedAccount(t, "staff", true, false)
	grantSubmissionPerm(t, staff.ID, "abort_any_submission")

	problem := models.Problem{Code: "p1", Name: "Problem", IsPublic: true}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&problem).Error)
	lang := models.Language{Key: "CPP17", Name: "C++17"}
	require.NoError(t, db.DB.Create(&lang).Error)
	newSub := func(status string) models.Submission {
		s := models.Submission{UserID: ownerP.ID, ProblemID: problem.ID, LanguageID: lang.ID, Date: time.Now(), Status: status}
		require.NoError(t, db.DB.Omit(clause.Associations).Create(&s).Error)
		return s
	}

	sub := newSub("QU")
	router := authedRouter(owner.ID, http.MethodPost, "/submission/:id/abort", SubmissionAbort)
	w := doRequest(router, http.MethodPost, "/submission/"+strconv.Itoa(int(sub.ID))+"/abort", "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.Equal(t, []uint{sub.ID}, bridge.ids)

	var reloaded models.Submission
	require.NoError(t, db.DB.First(&reloaded, sub.ID).Error)
	require.Equal(t, "AB", reloaded.Status)
	require.NotNil(t, reloaded.Result)
	require.Equal(t, "AB", *reloaded.Result)

	// Another user is refused.
	other := newSub("QU")
	strangerRouter := authedRouter(stranger.ID, http.MethodPost, "/submission/:id/abort", SubmissionAbort)
	w = doRequest(strangerRouter, http.MethodPost, "/submission/"+strconv.Itoa(int(other.ID))+"/abort", "")
	require.Equal(t, http.StatusForbidden, w.Code)

	// A rejudged submission is refused for its owner but allowed for staff.
	rejudged := newSub("QU")
	require.NoError(t, db.DB.Model(&models.Submission{}).Where("id = ?", rejudged.ID).
		Update("rejudged_date", time.Now()).Error)
	w = doRequest(router, http.MethodPost, "/submission/"+strconv.Itoa(int(rejudged.ID))+"/abort", "")
	require.Equal(t, http.StatusForbidden, w.Code)

	staffRouter := authedRouter(staff.ID, http.MethodPost, "/submission/:id/abort", SubmissionAbort)
	w = doRequest(staffRouter, http.MethodPost, "/submission/"+strconv.Itoa(int(rejudged.ID))+"/abort", "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	// Anonymous is refused (the route is login-gated; direct call has no user).
	anonRouter := authedRouter(0, http.MethodPost, "/submission/:id/abort", SubmissionAbort)
	w = doRequest(anonRouter, http.MethodPost, "/submission/"+strconv.Itoa(int(other.ID))+"/abort", "")
	require.Equal(t, http.StatusForbidden, w.Code)
}

// TestContestLeaveResponseShape keeps the leave/join JSON minimal and stable.
func TestContestLeaveResponseShape(t *testing.T) {
	setupContestSurfacesDB(t)
	f := seedContestSurfaces(t)
	part := participate(t, f.contest.ID, f.aliceP.ID)
	require.NoError(t, db.DB.Model(&models.Profile{}).Where("id = ?", f.aliceP.ID).
		Update("current_contest_id", part.ID).Error)

	router := authedRouter(f.alice.ID, http.MethodPost, "/contest/:key/leave", ContestLeave)
	w := doRequest(router, http.MethodPost, "/contest/c1/leave", "")
	require.Equal(t, http.StatusOK, w.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Equal(t, "successfully left contest", body["message"])
}
