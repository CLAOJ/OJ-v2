// Package problem provides problem management services.
package problem

import (
	"time"

	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/CLAOJ/claoj/sanitization"
	"github.com/CLAOJ/claoj/service/relations"
	"gorm.io/gorm"
)

// ProblemProfile is the admin view of a problem. JSON is flat snake_case;
// the admin edit form and PdfTab read it directly.
type ProblemProfile struct {
	ID                             uint                `json:"id"`
	Code                           string              `json:"code"`
	Name                           string              `json:"name"`
	Source                         string              `json:"source"`
	Description                    string              `json:"description"`
	PdfURL                         string              `json:"pdf_url"`
	GroupID                        uint                `json:"group_id"`
	GroupName                      string              `json:"group_name"`
	TimeLimit                      float64             `json:"time_limit"`
	MemoryLimit                    uint                `json:"memory_limit"`
	ShortCircuit                   bool                `json:"short_circuit"`
	Points                         float64             `json:"points"`
	Partial                        bool                `json:"partial"`
	IsPublic                       bool                `json:"is_public"`
	IsManuallyManaged              bool                `json:"is_manually_managed"`
	Date                           *time.Time          `json:"date"`
	LicenseID                      *uint               `json:"license_id"`
	OgImage                        string              `json:"og_image"`
	Summary                        string              `json:"summary"`
	UserCount                      int                 `json:"user_count"`
	AcRate                         float64             `json:"ac_rate"`
	IsFullMarkup                   bool                `json:"is_full_markup"`
	SubmissionSourceVisibilityMode string              `json:"submission_source_visibility_mode"`
	TestcaseVisibilityMode         string              `json:"testcase_visibility_mode"`
	IsOrganizationPrivate          bool                `json:"is_organization_private"`
	SuggesterID                    *uint               `json:"suggester_id"`
	AuthorIDs                      []uint              `json:"author_ids"`
	CuratorIDs                     []uint              `json:"curator_ids"`
	TesterIDs                      []uint              `json:"tester_ids"`
	TypeIDs                        []uint              `json:"type_ids"`
	AllowedLangIDs                 []uint              `json:"allowed_lang_ids"`
	OrganizationIDs                []uint              `json:"organization_ids"`
	Authors                        []relations.UserRef `json:"authors"`
	Curators                       []relations.UserRef `json:"curators"`
	Testers                        []relations.UserRef `json:"testers"`
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
// Pointer-to-slice fields are full replacement lists (nil = unchanged);
// the Add*/Remove* slices are the legacy incremental form.
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
	GroupID           *uint
	AuthorIDs         *[]uint
	CuratorIDs        *[]uint
	TesterIDs         *[]uint
	TypeIDs           *[]uint
	AllowedLangIDs    *[]uint
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
	Search   string // code or name contains (literal)
	IsPublic *bool
	GroupID  uint
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

	filter := func(q *gorm.DB) *gorm.DB {
		if req.Search != "" {
			p := db.ContainsPattern(req.Search)
			q = q.Where("(judge_problem.code LIKE ? ESCAPE '!' OR judge_problem.name LIKE ? ESCAPE '!')", p, p)
		}
		if req.IsPublic != nil {
			q = q.Where("judge_problem.is_public = ?", *req.IsPublic)
		}
		if req.GroupID != 0 {
			q = q.Where("judge_problem.group_id = ?", req.GroupID)
		}
		return q
	}

	var total int64
	if err := filter(db.DB.Table("judge_problem")).Count(&total).Error; err != nil {
		return nil, err
	}

	if err := filter(db.DB.Table("judge_problem").
		Joins("LEFT JOIN judge_problemgroup ON judge_problemgroup.id = judge_problem.group_id").
		Select("judge_problem.*, judge_problemgroup.name as group_name")).
		Order("judge_problem.date DESC").
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
		Preload("Authors.User").
		Preload("Curators.User").
		Preload("Testers.User").
		Preload("AllowedLangs").
		Preload("Organizations").
		Where("code = ?", req.ProblemCode).First(&problem).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrProblemNotFound
		}
		return nil, err
	}

	profile := problemToProfile(problem, problem.Group.Name)
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

// UpdateProblem updates an existing problem in one transaction.
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
	if req.GroupID != nil {
		var count int64
		if err := db.DB.Model(&models.ProblemGroup{}).Where("id = ?", *req.GroupID).Count(&count).Error; err != nil {
			return nil, err
		}
		if count == 0 {
			return nil, ErrInvalidGroup
		}
		updates["group_id"] = *req.GroupID
	}

	err := db.DB.Transaction(func(tx *gorm.DB) error {
		if len(updates) > 0 {
			if err := tx.Model(&problem).Updates(updates).Error; err != nil {
				return err
			}
		}

		if err := relations.Sync[models.ProblemType](tx, &problem, "Types", req.TypeIDs, req.AddTypeIDs, req.RemoveTypeIDs); err != nil {
			return err
		}
		if err := relations.Sync[models.Profile](tx, &problem, "Authors", req.AuthorIDs, req.AddAuthorIDs, req.RemoveAuthorIDs); err != nil {
			return err
		}
		if err := relations.Sync[models.Profile](tx, &problem, "Curators", req.CuratorIDs, nil, nil); err != nil {
			return err
		}
		if err := relations.Sync[models.Profile](tx, &problem, "Testers", req.TesterIDs, nil, nil); err != nil {
			return err
		}
		return relations.Sync[models.Language](tx, &problem, "AllowedLangs", req.AllowedLangIDs, req.AddLangIDs, req.RemoveLangIDs)
	})
	if err != nil {
		return nil, err
	}

	return s.GetProblem(GetProblemRequest{ProblemCode: problem.Code})
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
	curatorIDs := getProfileIDs(p.Curators)
	testerIDs := getProfileIDs(p.Testers)
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
		AuthorIDs:                      authorIDs,
		CuratorIDs:                     curatorIDs,
		TesterIDs:                      testerIDs,
		Authors:                        relations.UserRefs(p.Authors),
		Curators:                       relations.UserRefs(p.Curators),
		Testers:                        relations.UserRefs(p.Testers),
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
