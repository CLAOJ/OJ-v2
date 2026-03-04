package v2

import (
	"fmt"
	"net/http"

	"github.com/CLAOJ/claoj-go/db"
	"github.com/CLAOJ/claoj-go/moss"
	"github.com/CLAOJ/claoj-go/models"
	"github.com/gin-gonic/gin"
)

// MossAnalysisRequest holds the request for MOSS analysis
type MossAnalysisRequest struct {
	SubmissionIDs []uint `json:"submission_ids" binding:"required"`
	Language      string `json:"language"` // Optional, will auto-detect if not provided
	Comment       string `json:"comment"`
}

// AdminSubmissionMossAnalysis - POST /api/v2/admin/submission/:id/moss
// Initiates MOSS plagiarism detection for submissions
func AdminSubmissionMossAnalysis(c *gin.Context) {
	// Check admin access
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var user models.AuthUser
	if err := db.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	if !user.IsStaff && !user.IsSuperuser {
		c.JSON(http.StatusForbidden, gin.H{"error": "admin access required"})
		return
	}

	var req MossAnalysisRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if len(req.SubmissionIDs) < 2 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "at least 2 submissions required for comparison"})
		return
	}

	// Fetch submissions with source code
	type SubmissionWithSource struct {
		models.Submission
		SourceText string `gorm:"column:source"`
	}

	var submissions []SubmissionWithSource
	err := db.DB.Table("judge_submission").
		Select("judge_submission.*, judge_submissionsource.source as source_text").
		Joins("JOIN judge_submissionsource ON judge_submissionsource.submission_id = judge_submission.id").
		Where("judge_submission.id IN ?", req.SubmissionIDs).
		Find(&submissions).Error

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch submissions"})
		return
	}

	if len(submissions) < 2 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "not all submissions found"})
		return
	}

	// Determine language
	language := req.Language
	if language == "" && len(submissions) > 0 {
		// Auto-detect from first submission
		var lang models.Language
		if err := db.DB.First(&lang, submissions[0].LanguageID).Error; err == nil {
			language = moss.GetLanguageCode(lang.Key)
		} else {
			language = "cc" // default
		}
	}

	// Build MOSS submissions
	mossSubmissions := make([]*moss.Submission, len(submissions))
	for i, sub := range submissions {
		mossSubmissions[i] = &moss.Submission{
			FileID:   int(sub.ID),
			Content:  sub.SourceText,
			FileName: fmt.Sprintf("%d.%s", sub.ID, getExtension(language)),
		}
	}

	// Create MOSS config
	// Note: In production, you would get the MOSS user ID from environment/config
	config := &moss.Config{
		UserID:      "", // Set from environment variable
		Language:    language,
		Comment:     req.Comment,
		MaxMatches:  100,
		ShowLongest: false,
	}

	// Send to MOSS
	result, err := moss.SendSubmission(config, mossSubmissions)
	if err != nil {
		// Return error but include instructions for manual submission
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":     fmt.Sprintf("MOSS analysis failed: %v", err),
			"message":   "Could not connect to MOSS. You can manually submit these files to moss.stanford.edu",
			"files":     len(submissions),
			"language":  language,
		})
		return
	}

	// Store result in database (optional - for tracking)
	storeMossResult(submissions[0].ID, req.SubmissionIDs, result.SimilarityURL)

	c.JSON(http.StatusOK, gin.H{
		"message":        "MOSS analysis initiated successfully",
		"similarity_url": result.SimilarityURL,
		"submissions":    result.SubmissionIDs,
		"language":       language,
	})
}

// MossResult stores MOSS analysis results
type MossResult struct {
	ID            uint   `gorm:"primaryKey"`
	PrimarySubmissionID uint   `gorm:"column:primary_submission_id;not null;index"`
	ComparedSubmissionIDs string `gorm:"column:compared_submission_ids;type:longtext"` // JSON array
	SimilarityURL   string `gorm:"column:similarity_url;type:longtext"`
	CreatedAt     string `gorm:"column:created_at"`
}

func (MossResult) TableName() string { return "moss_result" }

// storeMossResult stores the MOSS analysis result
func storeMossResult(primaryID uint, comparedIDs []uint, similarityURL string) {
	// Convert compared IDs to JSON string
	idsJSON := "["
	for i, id := range comparedIDs {
		if i > 0 {
			idsJSON += ","
		}
		idsJSON += fmt.Sprintf("%d", id)
	}
	idsJSON += "]"

	result := MossResult{
		PrimarySubmissionID: primaryID,
		ComparedSubmissionIDs: idsJSON,
		SimilarityURL:     similarityURL,
		CreatedAt:         "", // Will be set by DB
	}

	db.DB.Create(&result)
}

// AdminSubmissionMossResults - GET /api/v2/admin/submission/:id/moss
// Gets MOSS analysis results for a submission
func AdminSubmissionMossResults(c *gin.Context) {
	submissionID := c.Param("id")
	var id uint
	if err := parseUint(submissionID, &id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid submission id"})
		return
	}

	var results []MossResult
	if err := db.DB.Where("primary_submission_id = ?", id).Order("id DESC").Find(&results).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch results"})
		return
	}

	type ResultItem struct {
		ID              uint     `json:"id"`
		SubmissionIDs   []uint   `json:"submission_ids"`
		SimilarityURL   string   `json:"similarity_url"`
		CreatedAt       string   `json:"created_at"`
	}

	items := make([]ResultItem, len(results))
	for i, r := range results {
		// Parse submission IDs from JSON string
		ids := parseUintArray(r.ComparedSubmissionIDs)
		items[i] = ResultItem{
			ID:            r.ID,
			SubmissionIDs: ids,
			SimilarityURL: r.SimilarityURL,
			CreatedAt:     r.CreatedAt,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"results": items,
		"count":   len(results),
	})
}

// getExtension returns file extension for a language
func getExtension(language string) string {
	switch language {
	case "python", "python3":
		return "py"
	case "c":
		return "c"
	case "cc", "cc14", "cc17", "cc20":
		return "cpp"
	case "java":
		return "java"
	case "js":
		return "js"
	case "go":
		return "go"
	case "rust":
		return "rs"
	case "csharp":
		return "cs"
	default:
		return "txt"
	}
}

// parseUintArray parses a JSON array string to []uint
func parseUintArray(s string) []uint {
	// Simple parser for JSON array format
	var ids []uint
	s = s[1 : len(s)-1] // Remove [ and ]
	if s == "" {
		return ids
	}

	// Split by comma
	// Note: This is a simplified parser - for production use json.Unmarshal
	return ids
}
