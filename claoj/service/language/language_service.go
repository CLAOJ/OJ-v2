// Package language provides language management services.
package language

import (
	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// LanguageService provides language management operations.
type LanguageService struct{}

// NewLanguageService creates a new LanguageService instance.
func NewLanguageService() *LanguageService {
	return &LanguageService{}
}

// ListLanguages retrieves a paginated list of languages.
func (s *LanguageService) ListLanguages(req ListLanguagesRequest) (*ListLanguagesResponse, error) {
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 20
	}
	if req.PageSize > 100 {
		req.PageSize = 100
	}

	var languages []models.Language
	query := db.DB.Model(&models.Language{})

	// Get total count
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	// Get paginated results
	if err := query.
		Order("name ASC").
		Offset((req.Page - 1) * req.PageSize).
		Limit(req.PageSize).
		Find(&languages).Error; err != nil {
		return nil, err
	}

	result := make([]Language, len(languages))
	for i, l := range languages {
		result[i] = languageToModel(l)
	}

	return &ListLanguagesResponse{
		Languages:  result,
		Total:      total,
		Page:       req.Page,
		PageSize:   req.PageSize,
	}, nil
}

// GetLanguage retrieves a language by ID.
func (s *LanguageService) GetLanguage(req GetLanguageRequest) (*Language, error) {
	if req.LanguageID == 0 {
		return nil, ErrInvalidLanguageID
	}

	var lang models.Language
	if err := db.DB.First(&lang, req.LanguageID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrLanguageNotFound
		}
		return nil, err
	}

	result := languageToModel(lang)
	return &result, nil
}

// CreateLanguage creates a new language.
func (s *LanguageService) CreateLanguage(req CreateLanguageRequest) (*Language, error) {
	if req.Key == "" {
		return nil, ErrEmptyLanguageKey
	}
	if req.Name == "" {
		return nil, ErrEmptyLanguageName
	}

	// Check if key already exists
	var existing models.Language
	if err := db.DB.Where("`key` = ?", req.Key).First(&existing).Error; err == nil {
		return nil, ErrLanguageKeyExists
	}

	lang := models.Language{
		Key:              req.Key,
		Name:             req.Name,
		ShortName:        req.ShortName,
		CommonName:       req.CommonName,
		Ace:              req.Ace,
		Pygments:         req.Pygments,
		Template:         req.Template,
		Info:             req.Info,
		Description:      req.Description,
		Extension:        req.Extension,
		FileOnly:         req.FileOnly,
		FileSizeLimit:    req.FileSizeLimit,
		IncludeInProblem: req.IncludeInProblem,
	}

	if err := db.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&lang).Error; err != nil {
			return err
		}
		return attachToAllProblems(tx, lang.ID, lang.IncludeInProblem)
	}); err != nil {
		return nil, err
	}

	result := languageToModel(lang)
	return &result, nil
}

// UpdateLanguage updates an existing language.
func (s *LanguageService) UpdateLanguage(req UpdateLanguageRequest) (*Language, error) {
	if req.LanguageID == 0 {
		return nil, ErrInvalidLanguageID
	}

	var lang models.Language
	if err := db.DB.First(&lang, req.LanguageID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrLanguageNotFound
		}
		return nil, err
	}

	updates := make(map[string]interface{})
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.ShortName != nil {
		updates["short_name"] = *req.ShortName
	}
	if req.CommonName != nil {
		updates["common_name"] = *req.CommonName
	}
	if req.Ace != nil {
		updates["ace"] = *req.Ace
	}
	if req.Pygments != nil {
		updates["pygments"] = *req.Pygments
	}
	if req.Template != nil {
		updates["template"] = *req.Template
	}
	if req.Info != nil {
		updates["info"] = *req.Info
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.Extension != nil {
		updates["extension"] = *req.Extension
	}
	if req.FileOnly != nil {
		updates["file_only"] = *req.FileOnly
	}
	if req.FileSizeLimit != nil {
		updates["file_size_limit"] = *req.FileSizeLimit
	}
	if req.IncludeInProblem != nil {
		updates["include_in_problem"] = *req.IncludeInProblem
	}

	if err := db.DB.Transaction(func(tx *gorm.DB) error {
		if len(updates) > 0 {
			if err := tx.Model(&lang).Updates(updates).Error; err != nil {
				return err
			}
		}
		// v1 only adds: turning the flag off never detaches the language. While
		// the flag is set every save re-attaches, which is how problems created
		// after the language are picked up.
		if req.IncludeInProblem != nil && *req.IncludeInProblem {
			return attachToAllProblems(tx, lang.ID, true)
		}
		return nil
	}); err != nil {
		return nil, err
	}

	result := languageToModel(lang)
	return &result, nil
}

// DeleteLanguage deletes a language (only if not in use).
func (s *LanguageService) DeleteLanguage(req DeleteLanguageRequest) error {
	if req.LanguageID == 0 {
		return ErrInvalidLanguageID
	}

	var lang models.Language
	if err := db.DB.First(&lang, req.LanguageID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return ErrLanguageNotFound
		}
		return err
	}

	// Check if language is used in submissions
	var submissionCount int64
	if err := db.DB.Model(&models.Submission{}).Where("language_id = ?", req.LanguageID).Count(&submissionCount).Error; err != nil {
		return err
	}
	if submissionCount > 0 {
		return ErrLanguageInUse
	}

	return db.DB.Delete(&lang).Error
}

// Helper functions

// attachToAllProblems mirrors v1 LanguageAdmin.save_model: a language flagged
// include_in_problem becomes an allowed language of every problem. Executed in
// chunks so a large problem table is not one giant insert, and with ON
// CONFLICT DO NOTHING so re-running it (or re-saving the form) is idempotent.
// There is no detach counterpart: v1 does not remove the language when the
// flag is cleared.
// shortcut: every problem id is loaded into memory before the chunked insert,
// switch to a dialect-specific INSERT ... SELECT past ~100k problems.
func attachToAllProblems(tx *gorm.DB, languageID uint, include bool) error {
	if !include || languageID == 0 {
		return nil
	}
	type allowedLanguage struct {
		ProblemID  uint `gorm:"column:problem_id"`
		LanguageID uint `gorm:"column:language_id"`
	}
	var problemIDs []uint
	if err := tx.Model(&models.Problem{}).Pluck("id", &problemIDs).Error; err != nil {
		return err
	}
	const batchSize = 500
	for start := 0; start < len(problemIDs); start += batchSize {
		end := min(start+batchSize, len(problemIDs))
		rows := make([]allowedLanguage, 0, end-start)
		for _, id := range problemIDs[start:end] {
			rows = append(rows, allowedLanguage{ProblemID: id, LanguageID: languageID})
		}
		if err := tx.Table("judge_problem_allowed_languages").
			Clauses(clause.OnConflict{DoNothing: true}).
			Create(&rows).Error; err != nil {
			return err
		}
	}
	return nil
}

func languageToModel(l models.Language) Language {
	return Language{
		ID:               l.ID,
		Key:              l.Key,
		Name:             l.Name,
		ShortName:        l.ShortName,
		CommonName:       l.CommonName,
		Ace:              l.Ace,
		Pygments:         l.Pygments,
		Template:         l.Template,
		Info:             l.Info,
		Description:      l.Description,
		Extension:        l.Extension,
		FileOnly:         l.FileOnly,
		FileSizeLimit:    l.FileSizeLimit,
		IncludeInProblem: l.IncludeInProblem,
	}
}
