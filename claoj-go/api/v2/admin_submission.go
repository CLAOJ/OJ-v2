package v2

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/CLAOJ/claoj-go/service/submission"
	"github.com/gin-gonic/gin"
)

// ============================================================
// ADMIN SUBMISSION MANAGEMENT API
// ============================================================

// AdminSubmissionList - GET /api/v2/admin/submissions
func AdminSubmissionList(c *gin.Context) {
	page, pageSize := parsePagination(c)

	resp, err := getSubmissionService().ListSubmissions(submission.ListSubmissionsRequest{
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	type Item struct {
		ID           uint      `json:"id"`
		Username     string    `json:"user"`
		ProblemCode  string    `json:"problem"`
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
			UserID      *uint   `json:"user_id"`
			Username    string  `json:"username"`
			ProblemID   *uint   `json:"problem_id"`
			ProblemCode string  `json:"problem_code"`
			LanguageID  *uint   `json:"language_id"`
			Language    string  `json:"language"`
			Status      string  `json:"status"`
			Result      string  `json:"result"`
			FromDate    string  `json:"from_date"`
			ToDate      string  `json:"to_date"`
		} `json:"filters"`
		DryRun bool `json:"dry_run"`
	}

	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, apiError("invalid request body"))
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
	user, _, ok := resolveUserProfile(c)
	if !ok {
		return
	}

	if !user.IsSuperuser {
		c.JSON(http.StatusForbidden, apiError("admin access required"))
		return
	}

	submissionIDStr := c.Param("id")
	var submissionID uint
	if err := parseUint(submissionIDStr, &submissionID); err != nil {
		c.JSON(http.StatusBadRequest, apiError("invalid submission ID"))
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
	user, _, ok := resolveUserProfile(c)
	if !ok {
		return
	}

	if !user.IsSuperuser {
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
	user, _, ok := resolveUserProfile(c)
	if !ok {
		return
	}

	if !user.IsSuperuser {
		c.JSON(http.StatusForbidden, apiError("admin access required"))
		return
	}

	code := c.Param("code")

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
		"message":        "problem rescore initiated",
		"rescored":       resp.Rescored,
		"total":          resp.Total,
		"problem_code":   code,
	})
}
