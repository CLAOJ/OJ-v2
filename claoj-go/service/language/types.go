// Package language provides language management services.
package language

// Language represents a programming language with its configuration.
type Language struct {
	ID               uint
	Key              string
	Name             string
	ShortName        *string
	CommonName       string
	Ace              string
	Pygments         string
	Template         string
	Info             string
	Description      string
	Extension        string
	FileOnly         bool
	FileSizeLimit    int
	IncludeInProblem bool
}

// ListLanguagesRequest holds parameters for listing languages.
type ListLanguagesRequest struct {
	Page     int
	PageSize int
}

// ListLanguagesResponse holds the response for listing languages.
type ListLanguagesResponse struct {
	Languages  []Language
	Total      int64
	Page       int
	PageSize   int
}

// GetLanguageRequest holds parameters for getting a language.
type GetLanguageRequest struct {
	LanguageID uint
}

// CreateLanguageRequest holds parameters for creating a language.
type CreateLanguageRequest struct {
	Key              string
	Name             string
	ShortName        *string
	CommonName       string
	Ace              string
	Pygments         string
	Template         string
	Info             string
	Description      string
	Extension        string
	FileOnly         bool
	FileSizeLimit    int
	IncludeInProblem bool
}

// UpdateLanguageRequest holds parameters for updating a language.
type UpdateLanguageRequest struct {
	LanguageID       uint
	Name             *string
	ShortName        *string
	CommonName       *string
	Ace              *string
	Pygments         *string
	Template         *string
	Info             *string
	Description      *string
	Extension        *string
	FileOnly         *bool
	FileSizeLimit    *int
	IncludeInProblem *bool
}

// DeleteLanguageRequest holds parameters for deleting a language.
type DeleteLanguageRequest struct {
	LanguageID uint
}
