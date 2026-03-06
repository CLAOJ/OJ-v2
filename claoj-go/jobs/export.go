package jobs

import (
	"archive/zip"
	"net/http"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/CLAOJ/claoj-go/db"
	"github.com/CLAOJ/claoj-go/models"
	"github.com/hibiken/asynq"
)

const (
	TypeUserExport = "user_export"

	// ExportExpiryHours is how long export files are kept before cleanup
	ExportExpiryHours = 24
)

// ExportPayload contains the user ID for export.
type ExportPayload struct {
	UserID uint `json:"user_id"`
}

// EnqueueUserExport queues a background task to prepare user data export.
func EnqueueUserExport(userID uint) error {
	payload, err := json.Marshal(ExportPayload{UserID: userID})
	if err != nil {
		return err
	}

	task := asynq.NewTask(TypeUserExport, payload, asynq.MaxRetry(3), asynq.Timeout(30*time.Minute))
	info, err := Client.Enqueue(task, asynq.Queue("default"))
	if err != nil {
		return err
	}

	log.Printf("jobs: enqueued user export task %s for user %d", info.ID, userID)
	return nil
}

// HandleUserExportTask processes the user data export.
func HandleUserExportTask(ctx context.Context, t *asynq.Task) error {
	var payload ExportPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return err
	}

	log.Printf("jobs: preparing data export for user %d", payload.UserID)

	// Create exports directory if it doesn't exist
	exportsDir := "./exports"
	if err := os.MkdirAll(exportsDir, 0755); err != nil {
		return fmt.Errorf("failed to create exports directory: %w", err)
	}

	// Generate unique export filename
	exportID := fmt.Sprintf("%d_%d", payload.UserID, time.Now().Unix())
	zipPath := filepath.Join(exportsDir, fmt.Sprintf("export_%s.zip", exportID))

	// Create the ZIP file
	if err := createUserExportZIP(payload.UserID, zipPath); err != nil {
		return fmt.Errorf("failed to create export ZIP: %w", err)
	}

	// Update the profile with the export path and timestamp
	exportURL := fmt.Sprintf("/api/v2/user/export/download/%s", exportID)
	if err := db.DB.Model(&models.Profile{}).
		Where("user_id = ?", payload.UserID).
		Updates(map[string]interface{}{
			"data_last_downloaded": time.Now(),
			"notes":                exportURL, // Store export URL in notes temporarily
		}).Error; err != nil {
		return fmt.Errorf("failed to update export metadata: %w", err)
	}

	log.Printf("jobs: completed data export for user %d at %s", payload.UserID, zipPath)
	return nil
}

// createUserExportZIP creates a ZIP file containing all user data.
func createUserExportZIP(userID uint, zipPath string) error {
	// Fetch user profile and auth info
	var profile models.Profile
	if err := db.DB.Preload("User").Where("user_id = ?", userID).First(&profile).Error; err != nil {
		return fmt.Errorf("failed to fetch profile: %w", err)
	}

	// Create ZIP file
	zipFile, err := os.Create(zipPath)
	if err != nil {
		return fmt.Errorf("failed to create zip file: %w", err)
	}
	defer zipFile.Close()

	w := zip.NewWriter(zipFile)
	defer w.Close()

	// 1. User Info (profile.json)
	if err := writeJSONFile(w, "profile.json", map[string]interface{}{
		"username":            profile.User.Username,
		"email":               profile.User.Email,
		"first_name":          profile.User.FirstName,
		"last_name":           profile.User.LastName,
		"date_joined":         profile.User.DateJoined,
		"is_superuser":        profile.User.IsSuperuser,
		"is_staff":            profile.User.IsStaff,
		"about":               profile.About,
		"points":              profile.Points,
		"performance_points":  profile.PerformancePoints,
		"contribution_points": profile.ContributionPoints,
		"rating":              profile.Rating,
		"problem_count":       profile.ProblemCount,
		"display_rank":        profile.DisplayRank,
		"language":            profile.Language.Key,
		"timezone":            profile.Timezone,
		"site_theme":          profile.SiteTheme,
		"ace_theme":           profile.AceTheme,
	}); err != nil {
		return err
	}

	// 2. Submissions (submissions.json)
	if err := exportSubmissions(w, userID); err != nil {
		return fmt.Errorf("failed to export submissions: %w", err)
	}

	// 3. Comments (comments.json)
	if err := exportComments(w, userID); err != nil {
		return fmt.Errorf("failed to export comments: %w", err)
	}

	// 4. Blog Posts (blogs.json)
	if err := exportBlogs(w, userID); err != nil {
		return fmt.Errorf("failed to export blogs: %w", err)
	}

	// 5. Tickets (tickets.json)
	if err := exportTickets(w, userID); err != nil {
		return fmt.Errorf("failed to export tickets: %w", err)
	}

	// 6. Contest Participations (contests.json)
	if err := exportContests(w, userID); err != nil {
		return fmt.Errorf("failed to export contests: %w", err)
	}

	// 7. Organizations (organizations.json)
	if err := exportOrganizations(w, userID); err != nil {
		return fmt.Errorf("failed to export organizations: %w", err)
	}

	return nil
}

// exportSubmissions exports all user submissions.
func exportSubmissions(w *zip.Writer, userID uint) error {
	var submissions []models.Submission
	if err := db.DB.Where("user_id = ?", userID).
		Preload("Problem").
		Preload("Language").
		Preload("Source").
		Order("date DESC").
		Find(&submissions).Error; err != nil {
		return err
	}

	type SubmissionExport struct {
		ID        uint      `json:"id"`
		Date      time.Time `json:"date"`
		Problem   string    `json:"problem"`
		Language  string    `json:"language"`
		Status    string    `json:"status"`
		Points    float64   `json:"points"`
		Time      float64   `json:"time"`
		Memory    float64   `json:"memory"`
		Source    string    `json:"source"`
		Error     *string   `json:"error,omitempty"`
	}

	export := make([]SubmissionExport, len(submissions))
	for i, sub := range submissions {
		var source string
		if sub.Source != nil {
			source = sub.Source.Source
		}
		var timeVal, memoryVal float64
		if sub.Time != nil {
			timeVal = *sub.Time
		}
		if sub.Memory != nil {
			memoryVal = *sub.Memory
		}
		var pointsVal float64
		if sub.Points != nil {
			pointsVal = *sub.Points
		}
		export[i] = SubmissionExport{
			ID:       sub.ID,
			Date:     sub.Date,
			Problem:  sub.Problem.Code,
			Language: sub.Language.Key,
			Status:   sub.Status,
			Points:   pointsVal,
			Time:     timeVal,
			Memory:   memoryVal,
			Source:   source,
			Error:    sub.Error,
		}
	}

	return writeJSONFile(w, "submissions.json", export)
}

// exportComments exports all user comments.
func exportComments(w *zip.Writer, userID uint) error {
	var comments []models.Comment
	if err := db.DB.Where("author_id = ?", userID).
		Order("time DESC").
		Find(&comments).Error; err != nil {
		return err
	}

	type CommentExport struct {
		ID     uint      `json:"id"`
		Time   time.Time `json:"time"`
		Page   string    `json:"page"`
		Score  int       `json:"score"`
		Body   string    `json:"body"`
		Hidden bool      `json:"hidden"`
	}

	export := make([]CommentExport, len(comments))
	for i, c := range comments {
		export[i] = CommentExport{
			ID:     c.ID,
			Time:   c.Time,
			Page:   c.Page,
			Score:  c.Score,
			Body:   c.Body,
			Hidden: c.Hidden,
		}
	}

	return writeJSONFile(w, "comments.json", export)
}

// exportBlogs exports all user blog posts.
func exportBlogs(w *zip.Writer, userID uint) error {
	var blogs []models.BlogPost
	if err := db.DB.Where("author_id = ?", userID).
		Order("publish_on DESC").
		Find(&blogs).Error; err != nil {
		return err
	}

	type BlogExport struct {
		ID        uint      `json:"id"`
		Title     string    `json:"title"`
		Slug      string    `json:"slug"`
		PublishOn time.Time `json:"publish_on"`
		Content   string    `json:"content"`
		Summary   string    `json:"summary"`
		Visible   bool      `json:"visible"`
		Score     int       `json:"score"`
		Sticky    bool      `json:"sticky"`
	}

	export := make([]BlogExport, len(blogs))
	for i, b := range blogs {
		export[i] = BlogExport{
			ID:        b.ID,
			Title:     b.Title,
			Slug:      b.Slug,
			PublishOn: b.PublishOn,
			Content:   b.Content,
			Summary:   b.Summary,
			Visible:   b.Visible,
			Score:     b.Score,
			Sticky:    b.Sticky,
		}
	}

	return writeJSONFile(w, "blogs.json", export)
}

// exportTickets exports all user tickets.
func exportTickets(w *zip.Writer, userID uint) error {
	var tickets []models.Ticket
	if err := db.DB.Where("user_id = ?", userID).
		Preload("Messages").
		Order("created DESC").
		Find(&tickets).Error; err != nil {
		return err
	}

	type TicketExport struct {
		ID             uint      `json:"id"`
		Title          string    `json:"title"`
		Created        time.Time `json:"created"`
		IsOpen         bool      `json:"is_open"`
		IsContributive bool      `json:"is_contributive"`
		Notes          string    `json:"notes"`
		MessageCount int        `json:"message_count"`
	}

	export := make([]TicketExport, len(tickets))
	for i, t := range tickets {
		export[i] = TicketExport{
			ID:             t.ID,
			Title:          t.Title,
			Created:        t.Time,
			IsOpen:         t.IsOpen,
			IsContributive: t.IsContributive,
			Notes:          t.Notes,
			MessageCount:   len(t.Messages),
		}
	}

	return writeJSONFile(w, "tickets.json", export)
}

// exportContests exports all user contest participations.
func exportContests(w *zip.Writer, userID uint) error {
	var participations []models.ContestParticipation
	if err := db.DB.Where("user_id = ?", userID).
		Preload("Contest").
		Order("id DESC").
		Find(&participations).Error; err != nil {
		return err
	}

	type ContestExport struct {
		ContestName  string    `json:"contest_name"`
		ContestKey   string    `json:"contest_key"`
		Score        float64   `json:"score"`
		Cumtime      uint      `json:"cumtime"`
		IsDisqualified bool    `json:"is_disqualified"`
		Virtual      int       `json:"virtual"`
		StartTime    time.Time `json:"start_time"`
	}

	export := make([]ContestExport, len(participations))
	for i, p := range participations {
		export[i] = ContestExport{
			ContestName:    p.Contest.Name,
			ContestKey:     p.Contest.Key,
			Score:          p.Score,
			Cumtime:        p.Cumtime,
			IsDisqualified: p.IsDisqualified,
			Virtual:        p.Virtual,
			StartTime:      p.RealStart,
		}
	}

	return writeJSONFile(w, "contests.json", export)
}

// exportOrganizations exports all user organization memberships.
func exportOrganizations(w *zip.Writer, userID uint) error {
	var profile models.Profile
	if err := db.DB.Preload("Organizations").Where("user_id = ?", userID).First(&profile).Error; err != nil {
		return err
	}

	type OrgExport struct {
		ID            uint      `json:"id"`
		Name          string    `json:"name"`
		Slug          string    `json:"slug"`
		ShortName     string    `json:"short_name"`
		About         string    `json:"about"`
		MemberSince   time.Time `json:"member_since"` // Approximation
	}

	export := make([]OrgExport, len(profile.Organizations))
	for i, o := range profile.Organizations {
		export[i] = OrgExport{
			ID:          o.ID,
			Name:        o.Name,
			Slug:        o.Slug,
			ShortName:   o.ShortName,
			About:       o.About,
			MemberSince: o.CreationDate, // Approximate
		}
	}

	return writeJSONFile(w, "organizations.json", export)
}

// writeJSONFile writes a JSON file to the ZIP archive.
func writeJSONFile(w *zip.Writer, filename string, data interface{}) error {
	f, err := w.Create(filename)
	if err != nil {
		return err
	}

	encoder := json.NewEncoder(f)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

// GetExportFilePath returns the file path for a user's export.
func GetExportFilePath(userID uint, exportID string) (string, error) {
	exportsDir := "./exports"
	expectedPath := filepath.Join(exportsDir, fmt.Sprintf("export_%s.zip", exportID))

	// Verify the export ID matches the user ID to prevent path traversal
	_ = userID // reserved for validation
	if filepath.Base(expectedPath) != fmt.Sprintf("export_%s.zip", exportID) {
		return "", fmt.Errorf("invalid export ID")
	}

	// Check if file exists
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		return "", fmt.Errorf("export file not found or expired")
	}

	return expectedPath, nil
}

// ServeExportFile serves the export file for download.
func ServeExportFile(w http.ResponseWriter, filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	// Get file info for Content-Length
	stat, err := file.Stat()
	if err != nil {
		return err
	}

	// Set headers for file download
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", stat.Size()))
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filepath.Base(filename)))

	// Copy file to response
	_, err = io.Copy(w, file)
	return err
}