// Package problem provides problem management services.
package problem

import (
	"time"

	"github.com/CLAOJ/claoj-go/db"
	"github.com/CLAOJ/claoj-go/models"
	"github.com/CLAOJ/claoj-go/sanitization"
	"gorm.io/gorm"
)

// ProblemProfile represents a problem with related data.
type ProblemProfile struct {
	ID                             uint
	Code                           string
	Name                           string
	Source                         string
	Description                    string
	PdfURL                         string
	GroupID                        uint
	GroupName                      string
	TimeLimit                      float64
	MemoryLimit                    uint
	ShortCircuit                   bool
	Points                         float64
	Partial                        bool
	IsPublic                       bool
	IsManuallyManaged              bool
	Date                           *time.Time
	LicenseID                      *uint
	OgImage                        string
	Summary                        string
	UserCount                      int
	AcRate                         float64
	IsFullMarkup                   bool
	SubmissionSourceVisibilityMode string
	TestcaseVisibilityMode         string
	IsOrganizationPrivate          bool
	SuggesterID                    *uint
	SuggestionStatus               string
	AuthorIDs                      []uint
	TypeIDs                        []uint
	AllowedLangIDs                 []uint
	OrganizationIDs                []uint
}

// CreateProblemRequest holds the parameters for creating a problem.
type CreateProblemRequest struct {
	Code              string
	Name              string
	Description       string
	Points            float64
	Partial           bool
	IsPublic          bool
	TimeLimit         float64
	MemoryLimit       uint
	GroupID           uint
	TypeIDs           []uint
	AuthorIDs         []uint
	AllowedLangIDs    []uint
	IsManuallyManaged bool
	PdfURL            string
}

// UpdateProblemRequest holds the parameters for updating a problem.
type UpdateProblemRequest struct {
	ProblemCode       string
	Name              *string
	Description       *string
	Points            *float64
	Partial           *bool
	IsPublic          *bool
	TimeLimit         *float64
	MemoryLimit       *uint
	IsManuallyManaged *bool
	PdfURL            *string
	AddTypeIDs        []uint
	RemoveTypeIDs     []uint
	AddAuthorIDs      []uint
	RemoveAuthorIDs   []uint
	AddLangIDs        []uint
	RemoveLangIDs     []uint
}

// DeleteProblemRequest holds the parameters for deleting a problem.
type DeleteProblemRequest struct {
	ProblemCode string
}

// CloneProblemRequest holds the parameters for cloning a problem.
type CloneProblemRequest struct {
	SourceCode string
	NewCode    string
	NewName    string
}

// GetProblemRequest holds the parameters for getting a problem.
type GetProblemRequest struct {
	ProblemCode string
}

// ListProblemsRequest holds the parameters for listing problems.
type ListProblemsRequest struct {
	Page     int
	PageSize int
}

// ListProblemsResponse holds the response for listing problems.
type ListProblemsResponse struct {
	Problems []ProblemProfile
	Total    int64
	Page     int
	PageSize int
}

// ProblemClarificationRequest holds the parameters for creating a problem clarification.
type ProblemClarificationRequest struct {
	ProblemCode string
	Description string
}

// ProblemDataRequest holds the parameters for problem data operations.
type ProblemDataRequest struct {
	ProblemCode string
}

// ProblemPdfRequest holds the parameters for PDF operations.
type ProblemPdfRequest struct {
	ProblemCode string
	PdfURL      string
}

// ProblemService provides problem management operations.
type ProblemService struct{}

// NewProblemService creates a new ProblemService instance.
func NewProblemService() *ProblemService {
	return &ProblemService{}
}

// ListProblems retrieves a paginated list of problems.
func (s *ProblemService) ListProblems(req ListProblemsRequest) (*ListProblemsResponse, error) {
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 20
	}
	if req.PageSize > 100 {
		req.PageSize = 100
	}

	var problems []struct {
		models.Problem
		GroupName string `gorm:"column:group_name"`
	}

	query := db.DB.Table("judge_problem").
		Joins("LEFT JOIN judge_problemgroup ON judge_problemgroup.id = judge_problem.group_id").
		Select("judge_problem.*, judge_problemgroup.name as group_name").
		Order("judge_problem.date DESC")

	// Get total count
	var total int64
	if err := db.DB.Model(&models.Problem{}).Count(&total).Error; err != nil {
		return nil, err
	}

	// Get paginated results
	if err := query.
		Offset((req.Page - 1) * req.PageSize).
		Limit(req.PageSize).
		Scan(&problems).Error; err != nil {
		return nil, err
	}

	result := make([]ProblemProfile, len(problems))
	for i, p := range problems {
		result[i] = problemToProfile(p.Problem, p.GroupName)
	}

	return &ListProblemsResponse{
		Problems: result,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}, nil
}

// GetProblem retrieves a problem by code with full details.
func (s *ProblemService) GetProblem(req GetProblemRequest) (*ProblemProfile, error) {
	var problem models.Problem
	if err := db.DB.Preload("Group").
		Preload("Types").
		Preload("Authors").
		Where("code = ?", req.ProblemCode).First(&problem).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrProblemNotFound
		}
		return nil, err
	}

	groupName := problem.Group.Name
	profile := problemToProfile(problem, groupName)
	return &profile, nil
}

// CreateProblem creates a new problem.
func (s *ProblemService) CreateProblem(req CreateProblemRequest) (*ProblemProfile, error) {
	problem := models.Problem{
		Code:              req.Code,
		Name:              sanitization.SanitizeTitle(req.Name),
		Description:       sanitization.SanitizeProblemContent(req.Description),
		Points:            req.Points,
		Partial:           req.Partial,
		IsPublic:          req.IsPublic,
		TimeLimit:         req.TimeLimit,
		MemoryLimit:       req.MemoryLimit,
		GroupID:           req.GroupID,
		IsManuallyManaged: req.IsManuallyManaged,
		PdfURL:            req.PdfURL,
	}

	if err := db.DB.Create(&problem).Error; err != nil {
		return nil, err
	}

	// Handle many-to-many relations
	if len(req.TypeIDs) > 0 {
		var types []models.ProblemType
		db.DB.Where("id IN ?", req.TypeIDs).Find(&types)
		db.DB.Model(&problem).Association("Types").Append(&types)
	}
	if len(req.AuthorIDs) > 0 {
		var authors []models.Profile
		db.DB.Where("id IN ?", req.AuthorIDs).Find(&authors)
		db.DB.Model(&problem).Association("Authors").Append(&authors)
	}
	if len(req.AllowedLangIDs) > 0 {
		var langs []models.Language
		db.DB.Where("id IN ?", req.AllowedLangIDs).Find(&langs)
		db.DB.Model(&problem).Association("AllowedLangs").Append(&langs)
	}

	// Get group name
	var groupName string
	if req.GroupID > 0 {
		var group models.ProblemGroup
		if err := db.DB.First(&group, req.GroupID).Error; err == nil {
			groupName = group.Name
		}
	}

	profile := problemToProfile(problem, groupName)
	return &profile, nil
}

// UpdateProblem updates an existing problem.
func (s *ProblemService) UpdateProblem(req UpdateProblemRequest) (*ProblemProfile, error) {
	var problem models.Problem
	if err := db.DB.Where("code = ?", req.ProblemCode).First(&problem).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrProblemNotFound
		}
		return nil, err
	}

	updates := make(map[string]interface{})
	if req.Name != nil {
		updates["name"] = sanitization.SanitizeTitle(*req.Name)
	}
	if req.Description != nil {
		updates["description"] = sanitization.SanitizeProblemContent(*req.Description)
	}
	if req.Points != nil {
		updates["points"] = *req.Points
	}
	if req.Partial != nil {
		updates["partial"] = *req.Partial
	}
	if req.IsPublic != nil {
		updates["is_public"] = *req.IsPublic
	}
	if req.TimeLimit != nil {
		updates["time_limit"] = *req.TimeLimit
	}
	if req.MemoryLimit != nil {
		updates["memory_limit"] = *req.MemoryLimit
	}
	if req.IsManuallyManaged != nil {
		updates["is_manually_managed"] = *req.IsManuallyManaged
	}
	if req.PdfURL != nil {
		updates["pdf_url"] = *req.PdfURL
	}

	if len(updates) > 0 {
		if err := db.DB.Model(&problem).Updates(updates).Error; err != nil {
			return nil, err
		}
	}

	// Handle type relations
	if len(req.AddTypeIDs) > 0 {
		var types []models.ProblemType
		db.DB.Where("id IN ?", req.AddTypeIDs).Find(&types)
		db.DB.Model(&problem).Association("Types").Append(&types)
	}
	if len(req.RemoveTypeIDs) > 0 {
		var types []models.ProblemType
		db.DB.Where("id IN ?", req.RemoveTypeIDs).Find(&types)
		db.DB.Model(&problem).Association("Types").Delete(&types)
	}

	// Handle author relations
	if len(req.AddAuthorIDs) > 0 {
		var authors []models.Profile
		db.DB.Where("id IN ?", req.AddAuthorIDs).Find(&authors)
		db.DB.Model(&problem).Association("Authors").Append(&authors)
	}
	if len(req.RemoveAuthorIDs) > 0 {
		var authors []models.Profile
		db.DB.Where("id IN ?", req.RemoveAuthorIDs).Find(&authors)
		db.DB.Model(&problem).Association("Authors").Delete(&authors)
	}

	// Handle language relations
	if len(req.AddLangIDs) > 0 {
		var langs []models.Language
		db.DB.Where("id IN ?", req.AddLangIDs).Find(&langs)
		db.DB.Model(&problem).Association("AllowedLangs").Append(&langs)
	}
	if len(req.RemoveLangIDs) > 0 {
		var langs []models.Language
		db.DB.Where("id IN ?", req.RemoveLangIDs).Find(&langs)
		db.DB.Model(&problem).Association("AllowedLangs").Delete(&langs)
	}

	// Reload problem with relations
	db.DB.Preload("Group").Preload("Types").Preload("Authors").First(&problem, problem.ID)

	groupName := problem.Group.Name
	profile := problemToProfile(problem, groupName)
	return &profile, nil
}

// DeleteProblem performs a soft delete by hiding the problem.
func (s *ProblemService) DeleteProblem(req DeleteProblemRequest) error {
	var problem models.Problem
	if err := db.DB.Where("code = ?", req.ProblemCode).First(&problem).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return ErrProblemNotFound
		}
		return err
	}

	return db.DB.Model(&problem).Update("is_public", false).Error
}

// CloneProblem creates a copy of an existing problem.
func (s *ProblemService) CloneProblem(req CloneProblemRequest) (*ProblemProfile, error) {
	// Get source problem
	var sourceProblem models.Problem
	if err := db.DB.Where("code = ?", req.SourceCode).First(&sourceProblem).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrProblemNotFound
		}
		return nil, err
	}

	// Check if new code already exists
	var existing models.Problem
	if err := db.DB.Where("code = ?", req.NewCode).First(&existing).Error; err == nil {
		return nil, ErrProblemCodeExists
	}

	// Create new problem
	newProblem := models.Problem{
		Code:                           req.NewCode,
		Name:                           sanitization.SanitizeTitle(req.NewName),
		Description:                    sourceProblem.Description,
		Points:                         sourceProblem.Points,
		Partial:                        sourceProblem.Partial,
		IsPublic:                       false, // Start as hidden
		TimeLimit:                      sourceProblem.TimeLimit,
		MemoryLimit:                    sourceProblem.MemoryLimit,
		GroupID:                        sourceProblem.GroupID,
		ShortCircuit:                   sourceProblem.ShortCircuit,
		IsManuallyManaged:              sourceProblem.IsManuallyManaged,
		LicenseID:                      sourceProblem.LicenseID,
		OgImage:                        sourceProblem.OgImage,
		Summary:                        sourceProblem.Summary,
		IsFullMarkup:                   sourceProblem.IsFullMarkup,
		SubmissionSourceVisibilityMode: sourceProblem.SubmissionSourceVisibilityMode,
		TestcaseVisibilityMode:         sourceProblem.TestcaseVisibilityMode,
		IsOrganizationPrivate:          sourceProblem.IsOrganizationPrivate,
	}

	if err := db.DB.Create(&newProblem).Error; err != nil {
		return nil, err
	}

	// Copy authors, types, allowed langs
	copyProblemAssociations(db.DB, sourceProblem.ID, newProblem.ID)

	// Reload with relations
	db.DB.Preload("Group").Preload("Types").Preload("Authors").First(&newProblem, newProblem.ID)

	groupName := newProblem.Group.Name
	profile := problemToProfile(newProblem, groupName)
	return &profile, nil
}

// CreateClarification creates a new problem clarification.
func (s *ProblemService) CreateClarification(problemCode, description string) error {
	var problem models.Problem
	if err := db.DB.Where("code = ?", problemCode).First(&problem).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return ErrProblemNotFound
		}
		return err
	}

	clarification := models.ProblemClarification{
		ProblemID:   problem.ID,
		Description: sanitization.SanitizeProblemContent(description),
		Date:        time.Now(),
	}

	return db.DB.Create(&clarification).Error
}

// DeleteClarification deletes a problem clarification.
func (s *ProblemService) DeleteClarification(clarificationID uint) error {
	var clarification models.ProblemClarification
	if err := db.DB.First(&clarification, clarificationID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return ErrClarificationNotFound
		}
		return err
	}

	return db.DB.Delete(&clarification).Error
}

// UpdatePdfURL updates the PDF URL for a problem.
func (s *ProblemService) UpdatePdfURL(problemCode, pdfURL string) error {
	var problem models.Problem
	if err := db.DB.Where("code = ?", problemCode).First(&problem).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return ErrProblemNotFound
		}
		return err
	}

	return db.DB.Model(&problem).Update("pdf_url", pdfURL).Error
}

// ClearPdfURL clears the PDF URL for a problem.
func (s *ProblemService) ClearPdfURL(problemCode string) error {
	var problem models.Problem
	if err := db.DB.Where("code = ?", problemCode).First(&problem).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return ErrProblemNotFound
		}
		return err
	}

	return db.DB.Model(&problem).Update("pdf_url", "").Error
}

// Helper functions

func problemToProfile(p models.Problem, groupName string) ProblemProfile {
	authorIDs := getProfileIDs(p.Authors)
	typeIDs := getTypeIDs(p.Types)
	langIDs := getLangIDs(p.AllowedLangs)
	orgIDs := getOrgIDs(p.Organizations)

	return ProblemProfile{
		ID:                             p.ID,
		Code:                           p.Code,
		Name:                           p.Name,
		Source:                         p.Source,
		Description:                    p.Description,
		PdfURL:                         p.PdfURL,
		GroupID:                        p.GroupID,
		GroupName:                      groupName,
		TimeLimit:                      p.TimeLimit,
		MemoryLimit:                    p.MemoryLimit,
		ShortCircuit:                   p.ShortCircuit,
		Points:                         p.Points,
		Partial:                        p.Partial,
		IsPublic:                       p.IsPublic,
		IsManuallyManaged:              p.IsManuallyManaged,
		Date:                           p.Date,
		LicenseID:                      p.LicenseID,
		OgImage:                        p.OgImage,
		Summary:                        p.Summary,
		UserCount:                      p.UserCount,
		AcRate:                         p.AcRate,
		IsFullMarkup:                   p.IsFullMarkup,
		SubmissionSourceVisibilityMode: p.SubmissionSourceVisibilityMode,
		TestcaseVisibilityMode:         p.TestcaseVisibilityMode,
		IsOrganizationPrivate:          p.IsOrganizationPrivate,
		SuggesterID:                    p.SuggesterID,
		SuggestionStatus:               p.SuggestionStatus,
		AuthorIDs:                      authorIDs,
		TypeIDs:                        typeIDs,
		AllowedLangIDs:                 langIDs,
		OrganizationIDs:                orgIDs,
	}
}

func getProfileIDs(profiles []models.Profile) []uint {
	ids := make([]uint, len(profiles))
	for i, p := range profiles {
		ids[i] = p.ID
	}
	return ids
}

func getTypeIDs(types []models.ProblemType) []uint {
	ids := make([]uint, len(types))
	for i, t := range types {
		ids[i] = t.ID
	}
	return ids
}

func getLangIDs(langs []models.Language) []uint {
	ids := make([]uint, len(langs))
	for i, l := range langs {
		ids[i] = l.ID
	}
	return ids
}

func getOrgIDs(orgs []models.Organization) []uint {
	ids := make([]uint, len(orgs))
	for i, o := range orgs {
		ids[i] = o.ID
	}
	return ids
}

func copyProblemAssociations(tx *gorm.DB, sourceID, targetID uint) {
	var authors, testers, curators []models.Profile
	var types []models.ProblemType
	var langs []models.Language
	var orgs []models.Organization

	tx.Model(&models.Problem{}).Where("id = ?", sourceID).Association("Authors").Find(&authors)
	tx.Model(&models.Problem{}).Where("id = ?", sourceID).Association("Testers").Find(&testers)
	tx.Model(&models.Problem{}).Where("id = ?", sourceID).Association("Curators").Find(&curators)
	tx.Model(&models.Problem{}).Where("id = ?", sourceID).Association("Types").Find(&types)
	tx.Model(&models.Problem{}).Where("id = ?", sourceID).Association("AllowedLangs").Find(&langs)
	tx.Model(&models.Problem{}).Where("id = ?", sourceID).Association("Organizations").Find(&orgs)

	if len(authors) > 0 {
		tx.Model(&models.Problem{}).Where("id = ?", targetID).Association("Authors").Append(&authors)
	}
	if len(testers) > 0 {
		tx.Model(&models.Problem{}).Where("id = ?", targetID).Association("Testers").Append(&testers)
	}
	if len(curators) > 0 {
		tx.Model(&models.Problem{}).Where("id = ?", targetID).Association("Curators").Append(&curators)
	}
	if len(types) > 0 {
		tx.Model(&models.Problem{}).Where("id = ?", targetID).Association("Types").Append(&types)
	}
	if len(langs) > 0 {
		tx.Model(&models.Problem{}).Where("id = ?", targetID).Association("AllowedLangs").Append(&langs)
	}
	if len(orgs) > 0 {
		tx.Model(&models.Problem{}).Where("id = ?", targetID).Association("Organizations").Append(&orgs)
	}
}
