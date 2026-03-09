// Package moss provides integration with the MOSS plagiarism detection system
package moss

import (
	"fmt"
	"strconv"
)

// Config holds MOSS configuration
type Config struct {
	UserID      string // MOSS user ID (required)
	Language    string // Programming language (e.g., "java", "cc", "py")
	Comment     string // Optional comment for the submission
	MaxMatches  int    // Maximum number of matches to report
	ShowLongest bool   // Show only longest matches
}

// DefaultLanguageMapping maps CLAOJ language IDs to MOSS languages
var DefaultLanguageMapping = map[string]string{
	"py":    "python",
	"py3":   "python3",
	"c":     "c",
	"cpp":   "cc",
	"cpp14": "cc14",
	"cpp17": "cc17",
	"cpp20": "cc20",
	"java":  "java",
	"js":    "js",
	"go":    "go",
	"rs":    "rust",
	"cs":    "csharp",
	"rb":    "ruby",
	"scala": "scala",
	"kt":    "kotlin",
	"swift": "swift",
	"php":   "php",
	"hs":    "haskell",
	"sql":   "sql",
}

// Submission represents a single submission to MOSS
type Submission struct {
	UserID   string
	FileID   int
	Content  string
	FileName string
}

// Result holds MOSS analysis results
type Result struct {
	SimilarityURL string   `json:"similarity_url"`
	Matches       []Match  `json:"matches,omitempty"`
	SubmissionIDs []uint   `json:"submission_ids"`
	Message       string   `json:"message"`
}

// Match represents a similarity match between two submissions
type Match struct {
	Submission1 uint `json:"submission_1"`
	Submission2 uint `json:"submission_2"`
	Lines       int  `json:"lines"`
	Percentage1 int  `json:"percentage_1"`
	Percentage2 int  `json:"percentage_2"`
}

// SendSubmission sends a submission to MOSS and returns the result URL
// For production, you would connect to moss.stanford.edu:7755
func SendSubmission(config *Config, submissions []*Submission) (*Result, error) {
	if config.UserID == "" {
		return nil, fmt.Errorf("MOSS user ID is required")
	}

	// For now, we return instructions for web-based submission
	return createWebSubmission(config, submissions), nil
}

// createWebSubmission creates a result for web-based MOSS submission
func createWebSubmission(config *Config, submissions []*Submission) *Result {
	instructions := "To analyze these submissions with MOSS:\n"
	instructions += "1. Visit http://moss.stanford.edu/\n"
	instructions += "2. Upload the following submission files:\n"
	for _, sub := range submissions {
		instructions += fmt.Sprintf("   - %s (Submission #%d)\n", sub.FileName, sub.FileID)
	}
	instructions += fmt.Sprintf("3. Select language: %s\n", config.Language)
	instructions += "4. Submit for analysis"

	return &Result{
		SimilarityURL: "http://moss.stanford.edu/",
		SubmissionIDs: extractSubmissionIDs(submissions),
		Message:       instructions,
	}
}

func extractSubmissionIDs(submissions []*Submission) []uint {
	ids := make([]uint, len(submissions))
	for i, sub := range submissions {
		id, _ := strconv.ParseUint(sub.FileName, 10, 32)
		ids[i] = uint(id)
	}
	return ids
}

// GetLanguageCode converts CLAOJ language ID to MOSS language code
func GetLanguageCode(claojLangID string) string {
	if lang, ok := DefaultLanguageMapping[claojLangID]; ok {
		return lang
	}
	return "cc" // default to C++
}

// BuildSubmissions builds MOSS submissions from submission sources
func BuildSubmissions(submissionID uint, source string, additionalSources map[uint]string) []*Submission {
	submissions := make([]*Submission, 0, len(additionalSources)+1)

	submissions = append(submissions, &Submission{
		FileID:   int(submissionID),
		Content:  source,
		FileName: fmt.Sprintf("%d.txt", submissionID),
	})

	for id, src := range additionalSources {
		submissions = append(submissions, &Submission{
			FileID:   int(id),
			Content:  src,
			FileName: fmt.Sprintf("%d.txt", id),
		})
	}

	return submissions
}
