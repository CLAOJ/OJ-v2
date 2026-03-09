package v2

import (
	"errors"
	"net/http"
	"time"

	"github.com/CLAOJ/claoj-go/db"
	"github.com/CLAOJ/claoj-go/jobs"
	"github.com/CLAOJ/claoj-go/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type SubmitRequest struct {
	Language   string `json:"language" binding:"required"`
	Source     string `json:"source" binding:"required"`
	ContestKey string `json:"contest_key"` // Optional
}

// Submit – POST /api/v2/problem/:code/submit
// @Description Submit a solution for a problem. Requires authentication.
// @Tags Problems
// @Summary Submit solution
// @Accept json
// @Produce json
// @Param code path string true "Problem code"
// @Param request body SubmitRequest true "Submission details"
// @Success 200 {object} map[string]interface{} "Submission queued"
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Problem not accessible"
// @Failure 404 {object} map[string]string "Problem not found"
// @Router /problem/{code}/submit [post]
func Submit(c *gin.Context) {
	code := c.Param("code")

	uid, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	userID := uid.(uint)

	var req SubmitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 1. Fetch Problem
	var problem models.Problem
	if err := db.DB.Where("code = ?", code).First(&problem).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "problem not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}

	if !problem.IsPublic {
		c.JSON(http.StatusForbidden, gin.H{"error": "problem is not public"})
		return
	}

	// 2. Fetch Language
	var lang models.Language
	if err := db.DB.Where("key = ?", req.Language).First(&lang).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid language"})
		return
	}

	// 3. Validate Source Length
	if len(req.Source) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "source cannot be empty"})
		return
	}
	// Note: normally there's a 65536 byte hard limit from Django, or custom per language
	if len(req.Source) > 65536 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "source code too long"})
		return
	}

	// 4. Create the Submission record within a transaction
	var sub models.Submission
	err := db.DB.Transaction(func(tx *gorm.DB) error {
		now := time.Now()
		statusStr := "QU"
		zeroFloat := 0.0

		sub = models.Submission{
			UserID:      userID,
			ProblemID:   problem.ID,
			LanguageID:  lang.ID,
			Date:        now,
			Status:      "QU",
			Result:      &statusStr,
			JudgedOnID:  nil,
			IsPretested: false,
			Time:        &zeroFloat,
			Memory:      &zeroFloat,
			Points:      &zeroFloat,
		}

		if err := tx.Create(&sub).Error; err != nil {
			return err
		}

		source := models.SubmissionSource{
			SubmissionID: sub.ID,
			Source:       req.Source,
		}
		if err := tx.Create(&source).Error; err != nil {
			return err
		}

		// 4b. If this is a contest submission, link it
		if req.ContestKey != "" {
			var ct models.Contest
			if err := tx.Where("key = ?", req.ContestKey).First(&ct).Error; err != nil {
				return errors.New("invalid contest")
			}

			if !ct.IsVisible {
				return errors.New("contest is not visible")
			}

			if now.Before(ct.StartTime) || now.After(ct.EndTime) {
				return errors.New("contest is not active")
			}

			// Check if contest is locked
			if ct.LockedAfter != nil && now.After(*ct.LockedAfter) {
				return errors.New("contest submissions are locked")
			}

			// We need the user's Profile ID to link to ContestParticipation
			var profile models.Profile
			if err := tx.Where("user_id = ?", userID).First(&profile).Error; err != nil {
				return errors.New("user profile not found")
			}

			// Check contest-level submission limit
			if ct.MaxSubmissions != nil {
				var submissionCount int64
				if err := tx.Table("judge_contestsubmission").
					Joins("JOIN judge_contestparticipation ON judge_contestsubmission.participation_id = judge_contestparticipation.id").
					Where("judge_contestparticipation.contest_id = ? AND judge_contestparticipation.user_id = ?", ct.ID, profile.ID).
					Count(&submissionCount).Error; err != nil {
					return err
				}
				if submissionCount >= int64(*ct.MaxSubmissions) {
					return errors.New("contest submission limit reached")
				}
			}

			var part models.ContestParticipation
			if err := tx.Where("contest_id = ? AND user_id = ?", ct.ID, profile.ID).First(&part).Error; err != nil {
				return errors.New("user has not joined the contest")
			}

			// Notice that ContestSubmission links to ContestProblem, not Problem directly
			var cprob models.ContestProblem
			if err := tx.Where("contest_id = ? AND problem_id = ?", ct.ID, problem.ID).First(&cprob).Error; err != nil {
				return errors.New("problem is not in this contest")
			}

			// Check per-problem submission limit
			if cprob.MaxSubmissions != nil {
				var problemSubmissionCount int64
				if err := tx.Table("judge_contestsubmission").
					Joins("JOIN judge_contestparticipation ON judge_contestsubmission.participation_id = judge_contestparticipation.id").
					Where("judge_contestparticipation.contest_id = ? AND judge_contestparticipation.user_id = ? AND judge_contestsubmission.problem_id = ?", ct.ID, profile.ID, cprob.ID).
					Count(&problemSubmissionCount).Error; err != nil {
					return err
				}
				if problemSubmissionCount >= int64(*cprob.MaxSubmissions) {
					return errors.New("problem submission limit reached")
				}
			}

			csub := models.ContestSubmission{
				SubmissionID:    sub.ID,
				ParticipationID: part.ID,
				ProblemID:       cprob.ID,
				Points:          0, // will be updated by Go bridge on_grading_end
				IsPretest:       false,
			}
			if err := tx.Create(&csub).Error; err != nil {
				return err
			}

			// Optionally link the Submission's ContestObjectID
			if err := tx.Model(&sub).UpdateColumn("contest_object_id", ct.ID).Error; err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 5. Enqueue the asynchronous grading task
	if err := jobs.EnqueueJudgeSubmission(sub.ID); err != nil {
		// Log error, but submission was already created.
		// Realistically we might want to display a warning.
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":         "submission saved but failed to enqueue judge task",
			"submission_id": sub.ID,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":       "submitted successfully",
		"submission_id": sub.ID,
	})
}
