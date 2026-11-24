package v2

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/CLAOJ/claoj/auth"
	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/CLAOJ/claoj/service/submission"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ============================================================
// ADMIN SUBMISSION MANAGEMENT API
// ============================================================

// AdminSubmissionList - GET /api/v2/admin/submissions
func AdminSubmissionList(c *gin.Context) {
	page, pageSize := parsePagination(c)

	resp, err := getSubmissionService().ListSubmissions(submission.ListSubmissionsRequest{
		Page:        page,
		PageSize:    pageSize,
		Username:    strings.TrimSpace(c.Query("user")),
		ProblemCode: strings.TrimSpace(c.Query("problem")),
		Result:      c.Query("result"),
		Status:      c.Query("status"),
		Language:    strings.TrimSpace(c.Query("language")),
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	type Item struct {
		ID           uint      `json:"id"`
		Username     string    `json:"user"`
		ProblemCode  string    `json:"problem"`
		ProblemName  string    `json:"problem_name"`
		LanguageName string    `json:"language"`
		Status       string    `json:"status"`
		Result       *string   `json:"result"`
		Score        *float64  `json:"score"`
		Time         *float64  `json:"time"`
		Memory       *float64  `json:"memory"`
		Date         time.Time `json:"date"`
		IsPretested  bool      `json:"is_pretested"`
	}
	items := make([]Item, len(resp.Submissions))
	for i, s := range resp.Submissions {
		date, _ := time.Parse(time.RFC3339, s.Date)
		items[i] = Item{
			ID:           s.ID,
			Username:     s.Username,
			ProblemCode:  s.ProblemCode,
			ProblemName:  s.ProblemName,
			LanguageName: s.LanguageName,
			Status:       s.Status,
			Result:       s.Result,
			Score:        s.Points,
			Time:         s.Time,
			Memory:       s.Memory,
			Date:         date,
			IsPretested:  s.IsPretested,
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"data":      items,
		"total":     resp.Total,
		"page":      resp.Page,
		"page_size": resp.PageSize,
	})
}

// AdminSubmissionRejudge - POST /api/v2/admin/submission/:id/rejudge
func AdminSubmissionRejudge(c *gin.Context) {
	idParam := c.Param("id")
	submissionID, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, apiError("invalid submission ID"))
		return
	}

	// Django parity: rejudging a submission requires being able to rejudge its problem.
	var sub models.Submission
	if err := db.DB.Preload("Problem.Authors").Preload("Problem.Curators").First(&sub, submissionID).Error; err != nil {
		c.JSON(http.StatusNotFound, apiError("submission not found"))
		return
	}
	if !auth.CanRejudge(c, &sub.Problem) {
		c.JSON(http.StatusForbidden, apiError("you do not have permission to rejudge this submission"))
		return
	}

	if err := getSubmissionService().Rejudge(submission.RejudgeRequest{
		SubmissionID: uint(submissionID),
	}); err != nil {
		if errors.Is(err, submission.ErrSubmissionNotFound) {
			c.JSON(http.StatusNotFound, apiError("submission not found"))
			return
		}
		if errors.Is(err, submission.ErrSubmissionLocked) {
			c.JSON(http.StatusBadRequest, apiError("submission is locked and cannot be rejudged"))
			return
		}
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Submission queued for rejudge",
	})
}

// AdminSubmissionAbort - POST /api/v2/admin/submission/:id/abort
func AdminSubmissionAbort(c *gin.Context) {
	idParam := c.Param("id")
	submissionID, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, apiError("invalid submission ID"))
		return
	}

	// Django parity: aborting another user's submission requires judge.abort_any_submission.
	if !auth.HasPerm(c, "judge.abort_any_submission") {
		c.JSON(http.StatusForbidden, apiError("you do not have permission to abort submissions"))
		return
	}

	if err := getSubmissionService().Abort(submission.AbortRequest{
		SubmissionID: uint(submissionID),
	}); err != nil {
		if errors.Is(err, submission.ErrSubmissionNotFound) {
			c.JSON(http.StatusNotFound, apiError("submission not found"))
			return
		}
		if errors.Is(err, submission.ErrSubmissionNotProcessing) {
			c.JSON(http.StatusBadRequest, apiError("submission is not being processed"))
			return
		}
		if errors.Is(err, submission.ErrBridgeServerNotAvailable) {
			c.JSON(http.StatusInternalServerError, apiError("bridge server not available"))
			return
		}
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Submission abort signal sent",
	})
}

// AdminSubmissionBatchRejudge - POST /api/v2/admin/submissions/batch-rejudge
func AdminSubmissionBatchRejudge(c *gin.Context) {
	var req struct {
		SubmissionIDs []uint `json:"submission_ids"`
		Filters       *struct {
			UserID      *uint  `json:"user_id"`
			Username    string `json:"username"`
			ProblemID   *uint  `json:"problem_id"`
			ProblemCode string `json:"problem_code"`
			LanguageID  *uint  `json:"language_id"`
			Language    string `json:"language"`
			Status      string `json:"status"`
			Result      string `json:"result"`
			FromDate    string `json:"from_date"`
			ToDate      string `json:"to_date"`
		} `json:"filters"`
		DryRun bool `json:"dry_run"`
	}

	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, apiError("invalid request body"))
		return
	}

	// Django parity: bulk rejudge requires judge.rejudge_submission_lot.
	if !auth.HasPerm(c, "judge.rejudge_submission_lot") {
		c.JSON(http.StatusForbidden, apiError("you do not have permission to batch-rejudge submissions"))
		return
	}

	filters := &submission.RejudgeFilters{}
	if req.Filters != nil {
		filters.UserID = req.Filters.UserID
		filters.Username = req.Filters.Username
		filters.ProblemID = req.Filters.ProblemID
		filters.ProblemCode = req.Filters.ProblemCode
		filters.LanguageID = req.Filters.LanguageID
		filters.Language = req.Filters.Language
		filters.Status = req.Filters.Status
		filters.Result = req.Filters.Result
		filters.FromDate = req.Filters.FromDate
		filters.ToDate = req.Filters.ToDate
	}

	resp, err := getSubmissionService().BatchRejudge(submission.BatchRejudgeRequest{
		SubmissionIDs: req.SubmissionIDs,
		Filters:       filters,
		DryRun:        req.DryRun,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	if req.DryRun {
		c.JSON(http.StatusOK, gin.H{
			"count":   resp.Count,
			"message": fmt.Sprintf("%d submissions would be rejudged", resp.Count),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"count":   resp.Count,
		"message": fmt.Sprintf("%d submissions queued for rejudge", resp.Count),
	})
}

// AdminSubmissionRescore - POST /api/v2/admin/submission/:id/rescore
// Rescores a single submission (recalculates points based on current test cases)
func AdminSubmissionRescore(c *gin.Context) {
	submissionIDStr := c.Param("id")
	var submissionID uint
	if err := parseUint(submissionIDStr, &submissionID); err != nil {
		c.JSON(http.StatusBadRequest, apiError("invalid submission ID"))
		return
	}

	// Django parity: rescoring a submission requires being able to rejudge its problem.
	var sub models.Submission
	if err := db.DB.Preload("Problem.Authors").Preload("Problem.Curators").First(&sub, submissionID).Error; err != nil {
		c.JSON(http.StatusNotFound, apiError("submission not found"))
		return
	}
	if !auth.CanRejudge(c, &sub.Problem) {
		c.JSON(http.StatusForbidden, apiError("admin access required"))
		return
	}

	if err := getSubmissionService().Rescore(submission.RescoreRequest{
		SubmissionID: submissionID,
	}); err != nil {
		if errors.Is(err, submission.ErrSubmissionNotFound) {
			c.JSON(http.StatusNotFound, apiError("submission not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, apiError("failed to requeue submission"))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":       "submission rescored successfully",
		"submission_id": submissionID,
	})
}

// AdminSubmissionBatchRescoreRequest - request body for batch rescore
type AdminSubmissionBatchRescoreRequest struct {
	SubmissionIDs []uint `json:"submission_ids"`
	ProblemID     *uint  `json:"problem_id"`
	UserID        *uint  `json:"user_id"`
	DryRun        bool   `json:"dry_run"`
}

// AdminSubmissionBatchRescore - POST /api/v2/admin/submissions/batch-rescore
// Rescores multiple submissions based on filters or specific IDs
func AdminSubmissionBatchRescore(c *gin.Context) {
	// Django parity: a bulk rescore is a "lot" operation.
	if !auth.HasPerm(c, "judge.rejudge_submission_lot") {
		c.JSON(http.StatusForbidden, apiError("admin access required"))
		return
	}

	var req AdminSubmissionBatchRescoreRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, apiError(err.Error()))
		return
	}

	resp, err := getSubmissionService().BatchRescore(submission.BatchRescoreRequest{
		SubmissionIDs: req.SubmissionIDs,
		ProblemID:     req.ProblemID,
		UserID:        req.UserID,
		DryRun:        req.DryRun,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	if req.DryRun {
		c.JSON(http.StatusOK, gin.H{
			"count":   resp.Total,
			"message": "dry run complete",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "batch rescore initiated",
		"rescored": resp.Rescored,
		"total":    resp.Total,
	})
}

// AdminProblemRescoreAll - POST /api/v2/admin/problem/:code/rescore-all
// Rescores all submissions for a specific problem
func AdminProblemRescoreAll(c *gin.Context) {
	code := c.Param("code")

	// Django parity: rescoring all submissions for a problem is a bulk rejudge
	// scoped to that problem.
	var problem models.Problem
	if err := db.DB.Preload("Authors").Preload("Curators").Where("code = ?", code).First(&problem).Error; err != nil {
		c.JSON(http.StatusNotFound, apiError("problem not found"))
		return
	}
	if !auth.CanRejudge(c, &problem) || !auth.HasPerm(c, "judge.rejudge_submission_lot") {
		c.JSON(http.StatusForbidden, apiError("admin access required"))
		return
	}

	resp, err := getSubmissionService().RescoreAll(submission.RescoreAllRequest{
		ProblemCode: code,
	})
	if err != nil {
		if errors.Is(err, submission.ErrProblemNotFound) {
			c.JSON(http.StatusNotFound, apiError("problem not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      "problem rescore initiated",
		"rescored":     resp.Rescored,
		"total":        resp.Total,
		"problem_code": code,
	})
}

// AdminSubmissionDelete - DELETE /api/v2/admin/submission/:id
// Permanently deletes a finished submission and recalculates the points it fed.
func AdminSubmissionDelete(c *gin.Context) {
	var submissionID uint
	if err := parseUint(c.Param("id"), &submissionID); err != nil {
		c.JSON(http.StatusBadRequest, apiUserError("invalid submission ID"))
		return
	}

	// Django parity: without judge.edit_all_problem a user may only delete
	// submissions of problems they author or curate.
	var sub models.Submission
	if err := db.DB.Preload("Problem.Authors").Preload("Problem.Curators").First(&sub, submissionID).Error; err != nil {
		c.JSON(http.StatusNotFound, apiError("submission not found"))
		return
	}
	if !auth.CanEditProblem(c, &sub.Problem) {
		c.JSON(http.StatusForbidden, apiUserError("you do not have permission to delete this submission"))
		return
	}

	if err := getSubmissionService().Delete(submission.DeleteRequest{SubmissionID: submissionID}); err != nil {
		switch {
		case errors.Is(err, submission.ErrSubmissionNotFound):
			c.JSON(http.StatusNotFound, apiError("submission not found"))
		case errors.Is(err, submission.ErrSubmissionInProgress):
			c.JSON(http.StatusConflict, apiUserError("submission is still being judged; abort it first"))
		default:
			c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Submission deleted",
	})
}

// AdminSubmissionDetail - GET /api/v2/admin/submission/:id
// Staff-readable detail: names, every numeric field, lock/archive state,
// source text and per-case results. No secrets are involved.
func AdminSubmissionDetail(c *gin.Context) {
	var submissionID uint
	if err := parseUint(c.Param("id"), &submissionID); err != nil {
		c.JSON(http.StatusBadRequest, apiUserError("invalid submission ID"))
		return
	}

	var sub models.Submission
	if err := db.DB.
		Preload("Problem").
		Preload("Language").
		Preload("JudgedOn").
		Preload("Source").
		Preload("TestCases", "1=1 ORDER BY `case` ASC").
		First(&sub, submissionID).Error; err != nil {
		c.JSON(http.StatusNotFound, apiError("submission not found"))
		return
	}

	var username string
	db.DB.Table("judge_profile").
		Select("auth_user.username").
		Joins("JOIN auth_user ON auth_user.id = judge_profile.user_id").
		Where("judge_profile.id = ?", sub.UserID).
		Scan(&username)

	source := ""
	if sub.Source != nil {
		source = sub.Source.Source
	}

	judgedOn := ""
	if sub.JudgedOn != nil {
		judgedOn = sub.JudgedOn.Name
	}

	var contestKey, contestName string
	if sub.ContestObjectID != nil {
		var contest models.Contest
		if err := db.DB.First(&contest, *sub.ContestObjectID).Error; err == nil {
			contestKey, contestName = contest.Key, contest.Name
		}
	}

	type TestCaseItem struct {
		ID       uint     `json:"id"`
		Case     int      `json:"case"`
		Status   string   `json:"status"`
		Time     *float64 `json:"time"`
		Memory   *float64 `json:"memory"`
		Points   *float64 `json:"points"`
		Total    *float64 `json:"total"`
		Feedback string   `json:"feedback"`
	}
	testCases := make([]TestCaseItem, len(sub.TestCases))
	for i, tc := range sub.TestCases {
		testCases[i] = TestCaseItem{tc.ID, tc.Case, tc.Status, tc.Time, tc.Memory, tc.Points, tc.Total, tc.Feedback}
	}

	c.JSON(http.StatusOK, gin.H{
		"id":                sub.ID,
		"user":              username,
		"problem":           sub.Problem.Code,
		"problem_name":      sub.Problem.Name,
		"language_id":       sub.LanguageID,
		"language":          sub.Language.Key,
		"language_name":     sub.Language.Name,
		"date":              sub.Date,
		"time":              sub.Time,
		"memory":            sub.Memory,
		"points":            sub.Points,
		"status":            sub.Status,
		"result":            sub.Result,
		"error":             sub.Error,
		"case_points":       sub.CasePoints,
		"case_total":        sub.CaseTotal,
		"current_testcase":  sub.CurrentTestcase,
		"batch":             sub.Batch,
		"is_pretested":      sub.IsPretested,
		"judged_on_id":      sub.JudgedOnID,
		"judged_on":         judgedOn,
		"judged_date":       sub.JudgedDate,
		"rejudged_date":     sub.RejudgedDate,
		"locked_after":      sub.LockedAfter,
		"is_locked":         sub.LockedAfter != nil && sub.LockedAfter.Before(time.Now()),
		"is_archived":       sub.IsArchived,
		"contest_object_id": sub.ContestObjectID,
		"contest_key":       contestKey,
		"contest_name":      contestName,
		"source":            source,
		"test_cases":        testCases,
	})
}

// AdminSubmissionUpdateRequest - PATCH /api/v2/admin/submission/:id
// Only staff-editable fields; user, problem and date are deliberately absent.
// Nullable fields are raw so an explicit null clears them while an omitted key
// leaves them unchanged.
type AdminSubmissionUpdateRequest struct {
	LanguageID      *uint           `json:"language_id"`
	Status          *string         `json:"status"`
	Result          json.RawMessage `json:"result"`
	Points          *float64        `json:"points"`
	CasePoints      *float64        `json:"case_points"`
	CaseTotal       *float64        `json:"case_total"`
	JudgedOnID      json.RawMessage `json:"judged_on_id"`
	JudgedDate      json.RawMessage `json:"judged_date"`
	Error           *string         `json:"error"`
	ContestObjectID json.RawMessage `json:"contest_object_id"`
	Source          *string         `json:"source"`
	LockedAfter     json.RawMessage `json:"locked_after"`
	IsArchived      *bool           `json:"is_archived"`
}

// parseNullableUint interprets a raw PATCH field: absent means unchanged, null
// clears, a positive number sets, anything else is invalid.
func parseNullableUint(raw json.RawMessage) (id *uint, clear, ok bool) {
	if len(raw) == 0 {
		return nil, false, true
	}
	if string(raw) == "null" {
		return nil, true, true
	}
	var v uint
	if err := json.Unmarshal(raw, &v); err != nil || v == 0 {
		return nil, false, false
	}
	return &v, false, true
}

// parseNullableString interprets a raw string field with the same absent /
// null / value rules as parseNullableUint.
func parseNullableString(raw json.RawMessage) (s *string, clear, ok bool) {
	if len(raw) == 0 {
		return nil, false, true
	}
	if string(raw) == "null" {
		return nil, true, true
	}
	var v string
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil, false, false
	}
	return &v, false, true
}

// parseAdminTime accepts RFC3339 and the datetime-local shapes the admin form
// sends, plus a bare date.
func parseAdminTime(raw json.RawMessage) (t *time.Time, clear, ok bool) {
	if len(raw) == 0 {
		return nil, false, true
	}
	if string(raw) == "null" {
		return nil, true, true
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil || s == "" {
		return nil, false, false
	}
	for _, layout := range []string{time.RFC3339, "2006-01-02T15:04:05", "2006-01-02T15:04", "2006-01-02"} {
		if parsed, err := time.Parse(layout, s); err == nil {
			return &parsed, false, true
		}
	}
	return nil, false, false
}

// AdminSubmissionUpdate - PATCH /api/v2/admin/submission/:id
func AdminSubmissionUpdate(c *gin.Context) {
	if !auth.HasPerm(c, "judge.change_submission") {
		c.JSON(http.StatusForbidden, apiUserError("you do not have permission to edit submissions"))
		return
	}

	var submissionID uint
	if err := parseUint(c.Param("id"), &submissionID); err != nil {
		c.JSON(http.StatusBadRequest, apiUserError("invalid submission ID"))
		return
	}

	var sub models.Submission
	if err := db.DB.First(&sub, submissionID).Error; err != nil {
		c.JSON(http.StatusNotFound, apiError("submission not found"))
		return
	}

	var req AdminSubmissionUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, apiUserError("invalid request body"))
		return
	}

	// The public submit path enforces maxSourceBytes; an admin edit must not be a
	// way around it.
	if req.Source != nil && len(*req.Source) > maxSourceBytes {
		c.JSON(http.StatusBadRequest, apiUserError("source code too long"))
		return
	}

	for _, f := range []struct {
		name string
		v    *float64
	}{
		{"points", req.Points},
		{"case_points", req.CasePoints},
		{"case_total", req.CaseTotal},
	} {
		if f.v != nil && (*f.v < 0 || *f.v > 100000) {
			c.JSON(http.StatusBadRequest, apiUserError("invalid "+f.name))
			return
		}
	}

	updates := map[string]interface{}{}

	if req.LanguageID != nil {
		var lang models.Language
		if *req.LanguageID == 0 || db.DB.Select("id").First(&lang, *req.LanguageID).Error != nil {
			c.JSON(http.StatusBadRequest, apiUserError("unknown language"))
			return
		}
		updates["language_id"] = *req.LanguageID
	}

	if req.Status != nil {
		if _, ok := models.SubmissionStatus[*req.Status]; !ok {
			c.JSON(http.StatusBadRequest, apiUserError("unknown submission status"))
			return
		}
		updates["status"] = *req.Status
	}

	if len(req.Result) > 0 {
		result, clear, ok := parseNullableString(req.Result)
		if !ok {
			c.JSON(http.StatusBadRequest, apiUserError("invalid result"))
			return
		}
		if clear {
			updates["result"] = nil
		} else {
			if _, ok := models.SubmissionResult[*result]; !ok {
				c.JSON(http.StatusBadRequest, apiUserError("unknown submission result"))
				return
			}
			updates["result"] = *result
		}
	}

	if req.Points != nil {
		updates["points"] = *req.Points
	}
	if req.CasePoints != nil {
		updates["case_points"] = *req.CasePoints
	}
	if req.CaseTotal != nil {
		updates["case_total"] = *req.CaseTotal
	}

	if len(req.JudgedOnID) > 0 {
		judgeID, clear, ok := parseNullableUint(req.JudgedOnID)
		if !ok {
			c.JSON(http.StatusBadRequest, apiUserError("invalid judged_on_id"))
			return
		}
		if clear {
			updates["judged_on_id"] = nil
		} else {
			var judge models.Judge
			if db.DB.Select("id").First(&judge, *judgeID).Error != nil {
				c.JSON(http.StatusBadRequest, apiUserError("unknown judge"))
				return
			}
			updates["judged_on_id"] = *judgeID
		}
	}

	if len(req.JudgedDate) > 0 {
		when, clear, ok := parseAdminTime(req.JudgedDate)
		if !ok {
			c.JSON(http.StatusBadRequest, apiUserError("invalid judged_date"))
			return
		}
		if clear {
			updates["judged_date"] = nil
		} else {
			updates["judged_date"] = *when
		}
	}

	if req.Error != nil {
		if *req.Error == "" {
			updates["error"] = nil
		} else {
			updates["error"] = *req.Error
		}
	}

	if len(req.ContestObjectID) > 0 {
		contestID, clear, ok := parseNullableUint(req.ContestObjectID)
		if !ok {
			c.JSON(http.StatusBadRequest, apiUserError("invalid contest_object_id"))
			return
		}
		if clear {
			updates["contest_object_id"] = nil
		} else {
			var contest models.Contest
			if db.DB.Select("id").First(&contest, *contestID).Error != nil {
				c.JSON(http.StatusBadRequest, apiUserError("unknown contest"))
				return
			}
			updates["contest_object_id"] = *contestID
		}
	}

	if len(req.LockedAfter) > 0 {
		if !auth.HasPerm(c, "judge.lock_submission") {
			c.JSON(http.StatusForbidden, apiUserError("you do not have permission to lock submissions"))
			return
		}
		when, clear, ok := parseAdminTime(req.LockedAfter)
		if !ok {
			c.JSON(http.StatusBadRequest, apiUserError("invalid locked_after"))
			return
		}
		if clear {
			updates["locked_after"] = nil
		} else {
			updates["locked_after"] = *when
		}
	}

	if req.IsArchived != nil {
		if !auth.HasPerm(c, "judge.archive_submission") {
			c.JSON(http.StatusForbidden, apiUserError("you do not have permission to archive submissions"))
			return
		}
		updates["is_archived"] = *req.IsArchived
	}

	if err := db.DB.Transaction(func(tx *gorm.DB) error {
		if len(updates) > 0 {
			if err := tx.Model(&models.Submission{}).Where("id = ?", sub.ID).Updates(updates).Error; err != nil {
				return err
			}
		}
		if req.Source == nil {
			return nil
		}
		var src models.SubmissionSource
		err := tx.Where("submission_id = ?", sub.ID).First(&src).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return tx.Create(&models.SubmissionSource{SubmissionID: sub.ID, Source: *req.Source}).Error
		}
		if err != nil {
			return err
		}
		return tx.Model(&src).Update("source", *req.Source).Error
	}); err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Submission updated",
	})
}

// AdminSubmissionBulkDelete - POST /api/v2/admin/submissions/bulk-delete
func AdminSubmissionBulkDelete(c *gin.Context) {
	if !auth.HasPerm(c, "judge.delete_submission") {
		c.JSON(http.StatusForbidden, apiUserError("you do not have permission to delete submissions"))
		return
	}

	var req struct {
		IDs []uint `json:"ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, apiUserError("invalid request body"))
		return
	}
	if len(req.IDs) == 0 {
		c.JSON(http.StatusBadRequest, apiUserError("no submissions selected"))
		return
	}
	if len(req.IDs) > 200 {
		c.JSON(http.StatusBadRequest, apiUserError("at most 200 submissions can be deleted at once"))
		return
	}
	for _, id := range req.IDs {
		if id == 0 {
			c.JSON(http.StatusBadRequest, apiUserError("invalid submission ID"))
			return
		}
	}

	deleted, err := getSubmissionService().BulkDelete(submission.BulkDeleteRequest{SubmissionIDs: req.IDs})
	if err != nil {
		if errors.Is(err, submission.ErrSubmissionInProgress) {
			c.JSON(http.StatusConflict, apiUserError("one or more submissions are still being judged; abort them first"))
			return
		}
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"deleted": deleted,
		"message": fmt.Sprintf("%d submissions deleted", deleted),
	})
}
