// Package problemsuggestion provides problem suggestion management services.
package problemsuggestion

import "time"

// ProblemSuggestion represents a problem suggestion with associated data.
type ProblemSuggestion struct {
	ID                   uint
	Code                 string
	Name                 string
	Description          string
	Points               float64
	Partial              bool
	IsPublic             bool
	TimeLimit            float64
	MemoryLimit          uint
	GroupID              uint
	TypeIDs              []uint
	Source               string
	Summary              string
	PdfURL               string
	IsFullMarkup         bool
	ShortCircuit         bool
	SuggesterID          *uint
	SuggestionStatus     string // pending, approved, rejected
	SuggestionNotes      string
	SuggestionReviewedAt *time.Time
	SuggestionReviewedBy *uint
	Date                 *time.Time
}

// ListSuggestionsRequest holds the parameters for listing suggestions.
type ListSuggestionsRequest struct {
	Page     int
	PageSize int
	Status   string // pending, approved, rejected, or empty for all
}

// ListSuggestionsResponse holds the response for listing suggestions.
type ListSuggestionsResponse struct {
	Suggestions []ProblemSuggestion
	Total       int64
	Page        int
	PageSize    int
}

// GetSuggestionRequest holds the parameters for getting a suggestion.
type GetSuggestionRequest struct {
	SuggestionID uint
}

// ApproveSuggestionRequest holds the parameters for approving a suggestion.
type ApproveSuggestionRequest struct {
	SuggestionID uint
	Code         string // Final problem code
	AdminNotes   string
	IsPublic     bool
	MakeFullMarkup bool
	AdminID      uint // ID of the admin approving
}

// RejectSuggestionRequest holds the parameters for rejecting a suggestion.
type RejectSuggestionRequest struct {
	SuggestionID uint
	Reason       string
	AdminNotes   string
	AdminID      uint // ID of the admin rejecting
}

// DeleteSuggestionRequest holds the parameters for deleting a suggestion.
type DeleteSuggestionRequest struct {
	SuggestionID uint
}

// ProblemSuggestionDetail holds detailed suggestion information.
type ProblemSuggestionDetail struct {
	Suggestion        ProblemSuggestion
	SuggesterUsername string
	SuggesterEmail    string
	GroupName         string
	TypeNames         []string
}
