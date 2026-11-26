package v2

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/CLAOJ/claoj/auth"
	"github.com/CLAOJ/claoj/config"
	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/CLAOJ/claoj/service/submission"
	"github.com/gin-gonic/gin"
	"github.com/pmezard/go-difflib/difflib"
	"gorm.io/gorm"
)

// SubmissionList – GET /api/v2/submissions
// Supports ?user=username, ?problem=code, ?result=, ?language=,
// ?organization=, ?contest=key and ?best=1. The verdict and language filters
// accept repeated params (?result=AC&result=WA) or one comma-separated value
// (?result=AC,WA); `organization` is an organization id, matching v1's
// `user__organizations=<pk>` filter (membership, not problem ownership).
// `contest` keeps only submissions linked to that contest (v1's in-contest
// submission lists) and applies v1's ForceContestMixin visibility rules.
// `best=1` keeps one row per user (v1 RankedSubmissions: highest points, then
// fastest time), dropping zero-point rows and unlisted profiles.
func SubmissionList(c *gin.Context) {
	page, pageSize := parsePagination(c)
	userFilter := c.Query("user")
	problemFilter := c.Query("problem")
	resultFilters := queryValues(c, "result")
	langFilters := queryValues(c, "language")
	orgFilter := strings.TrimSpace(c.Query("organization"))
	contestFilter := strings.TrimSpace(c.Query("contest"))

	// Only expose submissions to problems the requesting user may see, so
	// organization-private problems do not leak via the submission list.
	visExpr, visArgs := auth.VisibleProblemFilterFor(c, "pr")
	q := db.DB.Model(&models.Submission{}).
		Joins("JOIN judge_profile jp ON jp.id = judge_submission.user_id").
		Joins("JOIN auth_user au ON au.id = jp.user_id").
		Joins("JOIN judge_problem pr ON pr.id = judge_submission.problem_id").
		Joins("JOIN judge_language l ON l.id = judge_submission.language_id").
		Where(visExpr, visArgs...)

	if userFilter != "" {
		q = q.Where("au.username = ?", userFilter)
	}
	if problemFilter != "" {
		q = q.Where("pr.code = ?", problemFilter)
	}
	if len(resultFilters) > 0 {
		q = q.Where("judge_submission.result IN ?", resultFilters)
	}
	if len(langFilters) > 0 {
		q = q.Where("l.key IN ?", langFilters)
	}
	if orgFilter != "" {
		orgID, err := strconv.ParseUint(orgFilter, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, apiUserError("invalid organization id"))
			return
		}
		// Organization membership only; the problem visibility filter above
		// still decides which submissions (and org-private problems) are listed.
		q = q.Where("EXISTS (SELECT 1 FROM judge_profile_organizations po "+
			"WHERE po.profile_id = jp.id AND po.organization_id = ?)", orgID)
	}

	if contestFilter != "" {
		var ct models.Contest
		if err := db.DB.Where("`key` = ?", contestFilter).First(&ct).Error; err != nil {
			c.JSON(http.StatusNotFound, apiUserError("contest not found"))
			return
		}
		// v1 ForceContestMixin.access_check: hidden contests 404, and a contest
		// that has not started is not readable without see_private_contest.
		if !auth.CanViewContest(c, &ct) ||
			(time.Now().Before(ct.StartTime) && !auth.HasPerm(c, "judge.see_private_contest")) {
			c.JSON(http.StatusNotFound, apiUserError("contest not found"))
			return
		}
		q = q.
			Joins("JOIN judge_contestsubmission jcs ON jcs.submission_id = judge_submission.id").
			Joins("JOIN judge_contestparticipation jcp ON jcp.id = jcs.participation_id").
			Where("jcp.contest_id = ?", ct.ID)
		// v1 SubmissionsListBase: while the scoreboard is withheld, everyone
		// (participant or not) only sees their own submissions. A non-public
		// contest scoreboard must never leak other competitors' rows.
		if !auth.CanSeeContestScoreboard(c, &ct) {
			if profileID, ok := auth.CurrentProfileID(c); ok {
				q = q.Where("jcp.user_id = ?", profileID)
			} else {
				q = q.Where("1 = 0")
			}
		}
	}

	best := c.Query("best") == "1"
	if best {
		q = q.Where("jp.is_unlisted = ?", false).
			Where("judge_submission.points > ?", 0).
			Where("judge_submission.is_archived = ?", false)
		// Rank the already-filtered rows per submitter and keep the winner, so
		// the pager's COUNT and the page fetch both see one row per user.
		ranked := q.Session(&gorm.Session{}).
			Select("judge_submission.id, ROW_NUMBER() OVER (" +
				"PARTITION BY judge_submission.user_id " +
				"ORDER BY judge_submission.points DESC, judge_submission.time ASC, judge_submission.id ASC" +
				") AS best_rank")
		q = q.Joins("JOIN (?) AS best_rows ON best_rows.id = judge_submission.id", ranked).
			Where("best_rows.best_rank = 1")
	}

	// v1 AllSubmissions uses InfinitePaginationMixin: the unscoped list never
	// counts, because COUNT(*) over every submission with these joins costs
	// seconds. Lists scoped to a user/problem/contest/organization stay small
	// and keep the exact total, as v1's DiggPaginator lists did.
	countTotal := userFilter != "" || problemFilter != "" || contestFilter != "" || orgFilter != "" || best
	writeSubmissionList(c, q, page, pageSize, best, countTotal)
}

// submissionListRow is the public submission row shape, shared by the global
// list and the organization submissions tab.
type submissionListRow struct {
	ID          uint      `json:"id"`
	Problem     string    `json:"problem"`
	ProblemName string    `json:"problem_name"`
	User        string    `json:"user"`
	Date        time.Time `json:"date"`
	Language    string    `json:"language"`
	Status      string    `json:"status"`
	Result      *string   `json:"result"`
	Points      *float64  `json:"points"`
	Time        *float64  `json:"time"`
	Memory      *float64  `json:"memory"`
	CasePoints  float64   `json:"case_points"`
	CaseTotal   float64   `json:"case_total"`
}

// writeSubmissionList pages a submission query (which must already carry the
// judge_profile/ auth_user / judge_problem / judge_language joins) and writes
// the list response. `best` switches the order to v1's best-solutions ranking.
// countTotal=false skips the COUNT and answers {data, has_more} instead of
// {data, total}, fetching one extra row to learn whether a next page exists.
func writeSubmissionList(c *gin.Context, q *gorm.DB, page, pageSize int, best, countTotal bool) {
	var total int64
	if countTotal {
		if err := q.Session(&gorm.Session{}).Count(&total).Error; err != nil {
			c.JSON(http.StatusInternalServerError, apiError(err.Error()))
			return
		}
	}
	limit := pageSize
	if !countTotal {
		limit++
	}

	var rows []struct {
		models.Submission
		Username    string
		ProblemCode string
		ProblemName string
		LangKey     string
	}

	order := "judge_submission.date DESC"
	if best {
		order = "judge_submission.points DESC, judge_submission.time ASC, judge_submission.id ASC"
	}

	if err := q.Session(&gorm.Session{}).
		Select("judge_submission.*, au.username, pr.code as problem_code, pr.name as problem_name, l.key as lang_key").
		Order(order).
		Offset((page - 1) * pageSize).
		Limit(limit).
		Find(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}
	hasMore := len(rows) > pageSize
	if hasMore {
		rows = rows[:pageSize]
	}

	items := make([]submissionListRow, len(rows))
	for i, r := range rows {
		items[i] = submissionListRow{
			ID:          r.Submission.ID,
			Problem:     r.ProblemCode,
			ProblemName: r.ProblemName,
			User:        r.Username,
			Date:        r.Submission.Date,
			Language:    r.LangKey,
			Status:      r.Submission.Status,
			Result:      r.Submission.Result,
			Points:      r.Submission.Points,
			Time:        r.Submission.Time,
			Memory:      r.Submission.Memory,
			CasePoints:  r.Submission.CasePoints,
			CaseTotal:   r.Submission.CaseTotal,
		}
	}
	if !countTotal {
		c.JSON(http.StatusOK, gin.H{"data": items, "has_more": hasMore})
		return
	}
	c.JSON(http.StatusOK, apiListWithTotal(items, total))
}

// SubmissionDetail – GET /api/v2/submission/:id
func SubmissionDetail(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, apiUserError("invalid submission id"))
		return
	}

	var sub models.Submission
	if err := db.DB.
		Preload("Problem.Authors").
		Preload("Problem.Curators").
		Preload("Problem.Testers").
		Preload("Language").
		Preload("Source").
		Preload("TestCases", "1=1 ORDER BY `case` ASC").
		Where("id = ?", id).First(&sub).Error; err != nil {
		c.JSON(http.StatusNotFound, apiUserError("submission not found"))
		return
	}

	// Check problem is public
	if !sub.Problem.IsPublic {
		c.JSON(http.StatusNotFound, apiUserError("submission not found"))
		return
	}

	// v1 (SubmissionDetailBase) shows nothing of a submission (source, test
	// cases) to someone who fails Submission.can_see_detail.
	if !auth.CanSeeSubmissionDetail(c, &sub) {
		c.JSON(http.StatusForbidden, apiUserError("you do not have permission to view this submission"))
		return
	}

	// Resolve the submitter's username via auth_user. Preload("User.User")
	// does not populate the nested auth_user, so it must be joined explicitly.
	var username string
	db.DB.Table("judge_profile").
		Select("auth_user.username").
		Joins("JOIN auth_user ON auth_user.id = judge_profile.user_id").
		Where("judge_profile.id = ?", sub.UserID).
		Scan(&username)

	source := ""
	if sub.Source != nil {
		// File-only submissions store the uploaded file's public URL in
		// source. v1 (judge/views/submission.py) gives that URL only to the
		// submitter, so anonymous viewers get no source at all.
		if !sub.Language.FileOnly {
			source = sub.Source.Source
		} else {
			access := auth.GetAccess(c)
			profileID, _ := auth.CurrentProfileID(c)
			if (access.IsSuperuser && access.IsActive) || (profileID != 0 && profileID == sub.UserID) {
				source = sub.Source.Source
			}
		}
	}

	type TestCaseItem struct {
		Case     int      `json:"case"`
		Status   string   `json:"status"`
		Time     *float64 `json:"time"`
		Memory   *float64 `json:"memory"`
		Points   *float64 `json:"points"`
		Total    *float64 `json:"total"`
		Feedback string   `json:"feedback"`
		// Test data, only when the viewer may see it (v1 status-testcases.html):
		// input/answer cut to 64 bytes, the program's output, the checker log.
		Input            *string `json:"input,omitempty"`
		Answer           *string `json:"answer,omitempty"`
		Output           *string `json:"output,omitempty"`
		ExtendedFeedback string  `json:"extended_feedback,omitempty"`
	}
	canViewTest := auth.CanViewTestcases(c, &sub.Problem)
	var data map[int]caseData
	var prefix *int
	if canViewTest {
		data = problemCaseData(&sub.Problem)
		prefix = viewerOutputPrefix(c, &sub)
	}
	showData := prefix == nil || *prefix > 0
	testCases := make([]TestCaseItem, len(sub.TestCases))
	for i, tc := range sub.TestCases {
		item := TestCaseItem{Case: tc.Case, Status: tc.Status, Time: tc.Time, Memory: tc.Memory, Points: tc.Points, Total: tc.Total, Feedback: tc.Feedback}
		if canViewTest {
			item.ExtendedFeedback = tc.ExtendedFeedback
			if d, ok := data[tc.Case]; ok && showData {
				in, ans, out := d.Input, truncatePrefix(d.Answer, prefix), truncatePrefix(tc.Output, prefix)
				item.Input, item.Answer, item.Output = &in, &ans, &out
			}
		}
		testCases[i] = item
	}

	c.JSON(http.StatusOK, gin.H{
		"id":           sub.ID,
		"problem":      sub.Problem.Code,
		"problem_name": sub.Problem.Name,
		// The problem's full score, so the page can show 1.5/1.5 like v1
		// instead of the test-case point totals.
		"problem_points":     sub.Problem.Points,
		"user":               username,
		"date":               sub.Date,
		"language":           sub.Language.Key,
		"status":             sub.Status,
		"result":             sub.Result,
		"points":             sub.Points,
		"time":               sub.Time,
		"memory":             sub.Memory,
		"source":             source,
		"case_points":        sub.CasePoints,
		"case_total":         sub.CaseTotal,
		"test_cases":         testCases,
		"can_view_testcases": canViewTest,
		// Websocket channel with live grading events; unguessable, so only
		// viewers of this page learn it.
		"event_channel": SubmissionEventChannel(sub.ID),
	})
}

// SubmissionEventChannel is the per-submission websocket channel the bridge
// publishes grading progress to.
func SubmissionEventChannel(id uint) string {
	m := hmac.New(sha256.New, []byte(config.C.App.SecretKey))
	fmt.Fprintf(m, "submission-event:%d", id)
	return "sub_" + hex.EncodeToString(m.Sum(nil))[:32]
}

// viewerOutputPrefix is v1's prefix_length: a contest submission viewed by
// someone currently in a contest shows only output_prefix_override bytes of
// answer/output (0 hides the data). nil = no limit.
func viewerOutputPrefix(c *gin.Context, sub *models.Submission) *int {
	if sub.ContestObjectID == nil {
		return nil
	}
	profileID, ok := auth.CurrentProfileID(c)
	if !ok {
		return nil
	}
	var profile models.Profile
	if db.DB.Select("current_contest_id").First(&profile, profileID).Error != nil || profile.CurrentContestID == nil {
		return nil
	}
	var cp models.ContestProblem
	if db.DB.Where("contest_id = ? AND problem_id = ?", *sub.ContestObjectID, sub.ProblemID).First(&cp).Error != nil {
		return nil
	}
	return cp.OutputPrefixOverride
}

// SubmissionResubmitSource – GET /api/v2/submission/:id/resubmit
// The prefill for v1's /problem/<p>/resubmit/<submission>: source + language
// of an earlier submission. Only the owner may copy it; judge.resubmit_other
// may copy anyone's (v1 ProblemSubmit.dispatch). File-only languages have no
// source to prefill and 404 like v1.
func SubmissionResubmitSource(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, apiUserError("invalid submission id"))
		return
	}
	_, profile, ok := resolveUserProfile(c)
	if !ok {
		return
	}

	var sub models.Submission
	if err := db.DB.Preload("Source").Preload("Language").First(&sub, id).Error; err != nil {
		c.JSON(http.StatusNotFound, apiUserError("submission not found"))
		return
	}
	if sub.Language.FileOnly {
		c.JSON(http.StatusNotFound, apiUserError("submission not found"))
		return
	}
	if sub.UserID != profile.ID && !auth.HasPerm(c, "judge.resubmit_other") {
		c.JSON(http.StatusForbidden, apiUserError("not allowed to resubmit this submission"))
		return
	}
	source := ""
	if sub.Source != nil {
		source = sub.Source.Source
	}
	c.JSON(http.StatusOK, gin.H{
		"submission": sub.ID,
		"problem":    sub.ProblemID,
		"language":   sub.Language.Key,
		"source":     source,
	})
}

// SubmissionAbort - POST /api/v2/submission/:id/abort
// The owner may abort their own queued/running submission (v1
// /submission/<id>/abort); holders of judge.abort_any_submission may abort
// anyone's. v1 refuses an owner's abort once the submission was rejudged.
func SubmissionAbort(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, apiUserError("invalid submission id"))
		return
	}

	var sub models.Submission
	if err := db.DB.Select("id", "user_id", "rejudged_date").First(&sub, id).Error; err != nil {
		c.JSON(http.StatusNotFound, apiUserError("submission not found"))
		return
	}

	canAbortAny := auth.HasPerm(c, "judge.abort_any_submission")
	profileID, _ := auth.CurrentProfileID(c)
	if !canAbortAny && (profileID == 0 || profileID != sub.UserID || sub.RejudgedDate != nil) {
		c.JSON(http.StatusForbidden, apiUserError("you do not have permission to abort this submission"))
		return
	}

	if err := getSubmissionService().Abort(submission.AbortRequest{SubmissionID: uint(id)}); err != nil {
		switch {
		case errors.Is(err, submission.ErrSubmissionNotFound):
			c.JSON(http.StatusNotFound, apiUserError("submission not found"))
		case errors.Is(err, submission.ErrSubmissionNotProcessing):
			c.JSON(http.StatusBadRequest, apiUserError("submission is not being processed"))
		case errors.Is(err, submission.ErrBridgeServerNotAvailable):
			c.JSON(http.StatusInternalServerError, apiUserError("bridge server not available"))
		default:
			c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Submission abort signal sent"})
}

// SubmissionDiff - GET /api/v2/submissions/:id1/diff/:id2
// Compares two submissions and returns the unified diff
func SubmissionDiff(c *gin.Context) {
	id1, err := strconv.ParseUint(c.Param("id1"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, apiUserError("invalid submission id"))
		return
	}
	id2, err := strconv.ParseUint(c.Param("id2"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, apiUserError("invalid submission id"))
		return
	}

	// Get both submissions
	var sub1, sub2 models.Submission
	if err := db.DB.Preload("Source").Where("id = ?", id1).First(&sub1).Error; err != nil {
		c.JSON(http.StatusNotFound, apiUserError("submission 1 not found"))
		return
	}
	if err := db.DB.Preload("Source").Where("id = ?", id2).First(&sub2).Error; err != nil {
		c.JSON(http.StatusNotFound, apiUserError("submission 2 not found"))
		return
	}

	// Check permissions: both submissions must be accessible
	// Public submissions or user's own submissions.
	// judge_submission.user_id is a judge_profile.id FK — compare against
	// the viewer's profile id, not their auth_user id.
	profileID := uint(0)
	if pid, ok := auth.CurrentProfileID(c); ok {
		profileID = pid
	}

	// Django parity: seeing any submission's detail requires judge.view_all_submission.
	isAdmin := auth.HasPerm(c, "judge.view_all_submission")

	// Check access to submission 1
	if !sub1.Problem.IsPublic && sub1.UserID != profileID && !isAdmin {
		c.JSON(http.StatusForbidden, apiUserError("access denied to submission 1"))
		return
	}
	// Check access to submission 2
	if !sub2.Problem.IsPublic && sub2.UserID != profileID && !isAdmin {
		c.JSON(http.StatusForbidden, apiUserError("access denied to submission 2"))
		return
	}

	source1 := ""
	source2 := ""
	if sub1.Source != nil {
		source1 = sub1.Source.Source
	}
	if sub2.Source != nil {
		source2 = sub2.Source.Source
	}

	// Generate unified diff
	lines1 := strings.Split(source1, "\n")
	lines2 := strings.Split(source2, "\n")

	diffResult, err := difflib.GetUnifiedDiffString(difflib.UnifiedDiff{
		A:        lines1,
		B:        lines2,
		FromFile: "Submission #" + strconv.FormatUint(id1, 10),
		ToFile:   "Submission #" + strconv.FormatUint(id2, 10),
		Context:  3,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, apiUserError("failed to generate diff"))
		return
	}

	// Generate line-by-line diff for frontend rendering
	diffLines := []gin.H{}
	diffObj := difflib.NewMatcher(lines1, lines2)
	opCodes := diffObj.GetGroupedOpCodes(3)

	for _, op := range opCodes {
		tag := op[0]
		for i := int(tag.I1); i < int(tag.I2); i++ {
			diffLines = append(diffLines, gin.H{
				"type":    "delete",
				"line":    i + 1,
				"content": lines1[i],
			})
		}
		for i := int(tag.J1); i < int(tag.J2); i++ {
			diffLines = append(diffLines, gin.H{
				"type":    "add",
				"line":    i + 1,
				"content": lines2[i],
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"submission1": gin.H{
			"id":       sub1.ID,
			"problem":  sub1.Problem.Code,
			"user":     getUserUsername(sub1.UserID),
			"date":     sub1.Date,
			"language": getLanguageKey(sub1.LanguageID),
			"result":   sub1.Result,
			"points":   sub1.Points,
		},
		"submission2": gin.H{
			"id":       sub2.ID,
			"problem":  sub2.Problem.Code,
			"user":     getUserUsername(sub2.UserID),
			"date":     sub2.Date,
			"language": getLanguageKey(sub2.LanguageID),
			"result":   sub2.Result,
			"points":   sub2.Points,
		},
		"unified_diff": diffResult,
		"diff_lines":   diffLines,
		"stats": gin.H{
			"additions": countAdditions(diffLines),
			"deletions": countDeletions(diffLines),
		},
	})
}

// Helper functions for SubmissionDiff
// getUserUsername resolves a judge_submission.user_id — a judge_profile.id FK,
// not an auth_user.id — to the owning auth user's username.
func getUserUsername(profileID uint) string {
	var profile models.Profile
	if err := db.DB.Preload("User").First(&profile, profileID).Error; err != nil {
		return "unknown"
	}
	return profile.User.Username
}

func getLanguageKey(languageID uint) string {
	var lang models.Language
	if err := db.DB.Where("id = ?", languageID).First(&lang).Error; err != nil {
		return "unknown"
	}
	return lang.Key
}

func countAdditions(diffLines []gin.H) int {
	count := 0
	for _, line := range diffLines {
		if line["type"] == "add" {
			count++
		}
	}
	return count
}

func countDeletions(diffLines []gin.H) int {
	count := 0
	for _, line := range diffLines {
		if line["type"] == "delete" {
			count++
		}
	}
	return count
}
