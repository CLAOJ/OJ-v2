package v2

import (
	"errors"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/CLAOJ/claoj/auth"
	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/jobs"
	"github.com/CLAOJ/claoj/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// maxSourceBytes is Django's hard per-submission source limit.
const maxSourceBytes = 65536

// maxSubmissionUploadBytes caps the whole multipart body of a file submission.
// The real limit is the language's file_size_limit (MB) applied to the file
// itself; this ceiling only stops an unlimited upload from being buffered
// before the language (and therefore its limit) is known.
const maxSubmissionUploadBytes = 32 << 20

type SubmitRequest struct {
	Language   string `json:"language" binding:"required"`
	Source     string `json:"source" binding:"required"`
	ContestKey string `json:"contest_key"` // Optional
}

// Submit – POST /api/v2/problem/:code/submit
// @Description Submit a solution for a problem. Requires authentication. Text
// @Description languages post JSON {language, source}; file-only languages
// @Description post multipart/form-data with fields language, contest_key and
// @Description one submission_file whose extension matches the language.
// @Tags Problems
// @Summary Submit solution
// @Accept json
// @Accept mpfd
// @Produce json
// @Param code path string true "Problem code"
// @Param request body SubmitRequest false "Submission details (JSON)"
// @Param language formData string false "Language key (multipart)"
// @Param contest_key formData string false "Contest key (multipart)"
// @Param contest query string false "Contest key (query param, as the web client sends it)"
// @Param submission_file formData file false "Source file (file-only languages)"
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
	var form *multipart.Form
	// file_only languages submit an uploaded file (v1 ProblemSubmitForm), all
	// other languages keep the JSON {language, source} path.
	isUpload := strings.HasPrefix(c.ContentType(), "multipart/form-data")
	if isUpload {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxSubmissionUploadBytes)
		parsed, err := c.MultipartForm()
		if err != nil {
			var tooLarge *http.MaxBytesError
			if errors.As(err, &tooLarge) {
				c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "the uploaded file is too large"})
			} else {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid multipart body"})
			}
			return
		}
		form = parsed
		req.Language = firstFormValue(form, "language")
		req.ContestKey = firstFormValue(form, "contest_key")
	} else if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// The web client opens the problem page with ?contest=<key> and submits
	// there, so the key arrives as a query param rather than a form/JSON field.
	// Without this fallback those submissions were never linked to the contest.
	if req.ContestKey == "" {
		req.ContestKey = c.Query("contest")
	}

	// 1. Fetch Problem (editor/tester associations loaded for the access check)
	var problem models.Problem
	if err := db.DB.Preload("Authors").Preload("Curators").Preload("Testers").
		Where("code = ?", code).First(&problem).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "problem not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}

	// Only allow submitting to a problem the user may access (Django parity:
	// Problem.is_accessible_by). This blocks organization-private problems for
	// non-members, not just non-public ones.
	if !auth.CanViewProblem(c, &problem) {
		c.JSON(http.StatusForbidden, gin.H{"error": "you do not have access to this problem"})
		return
	}

	// 2. Fetch Language
	var lang models.Language
	if err := db.DB.Where("`key` = ?", req.Language).First(&lang).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid language"})
		return
	}

	// 3. Validate the source for the language's submission mode
	sourceValue := req.Source
	storedPath := ""
	if isUpload {
		if !lang.FileOnly {
			c.JSON(http.StatusBadRequest, gin.H{"error": "this language does not accept file submissions"})
			return
		}
		files := form.File["submission_file"]
		if len(files) != 1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "exactly one submission_file is required"})
			return
		}
		publicPath, err := saveSubmissionFile(c, files[0], &lang)
		if err != nil {
			var userErr submissionFileError
			if errors.As(err, &userErr) {
				c.JSON(http.StatusBadRequest, gin.H{"error": userErr.Error()})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to store the submitted file"})
			}
			return
		}
		sourceValue = publicPath
		storedPath = filepath.Join(submissionFileDir, filepath.Base(publicPath))
	} else {
		if lang.FileOnly {
			c.JSON(http.StatusBadRequest, gin.H{"error": "this language requires a file submission"})
			return
		}
		if len(req.Source) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "source cannot be empty"})
			return
		}
		// Note: normally there's a 65536 byte hard limit from Django, or custom per language
		if len(req.Source) > maxSourceBytes {
			c.JSON(http.StatusBadRequest, gin.H{"error": "source code too long"})
			return
		}
	}

	// The stored file is only kept once its source row is committed; every
	// failure before that point removes the orphan.
	fileKept := false
	if storedPath != "" {
		defer func() {
			if !fileKept {
				os.Remove(storedPath)
			}
		}()
	}

	// 4. Create the Submission record within a transaction
	var sub models.Submission
	err := db.DB.Transaction(func(tx *gorm.DB) error {
		now := time.Now()
		statusStr := "QU"
		zeroFloat := 0.0

		// judge_submission.user_id is a FK to judge_profile.id, NOT
		// auth_user.id — the two id sequences diverge on real data, so
		// resolve the profile first.
		var submitter models.Profile
		if err := tx.Where("user_id = ?", userID).First(&submitter).Error; err != nil {
			return errors.New("user profile not found")
		}

		sub = models.Submission{
			UserID:      submitter.ID,
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
			Source:       sourceValue,
		}
		if err := tx.Create(&source).Error; err != nil {
			return err
		}

		// 4b. If this is a contest submission, link it
		if req.ContestKey != "" {
			var ct models.Contest
			if err := tx.Where("`key` = ?", req.ContestKey).First(&ct).Error; err != nil {
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
	fileKept = true

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

// firstFormValue reads the first value of a multipart form field.
func firstFormValue(form *multipart.Form, key string) string {
	if values := form.Value[key]; len(values) > 0 {
		return values[0]
	}
	return ""
}
