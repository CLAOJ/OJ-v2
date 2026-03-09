package v2

import (
	"net/http"
	"time"

	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/CLAOJ/claoj/sanitization"
	"github.com/CLAOJ/claoj/service/auditlog"
	"github.com/CLAOJ/claoj/service/blogpost"
	"github.com/CLAOJ/claoj/service/comment"
	"github.com/CLAOJ/claoj/service/contest"
	"github.com/CLAOJ/claoj/service/language"
	"github.com/CLAOJ/claoj/service/license"
	"github.com/CLAOJ/claoj/service/miscconfig"
	"github.com/CLAOJ/claoj/service/navigation"
	"github.com/CLAOJ/claoj/service/organization"
	"github.com/CLAOJ/claoj/service/problem"
	"github.com/CLAOJ/claoj/service/problemgroup"
	"github.com/CLAOJ/claoj/service/problemtype"
	"github.com/CLAOJ/claoj/service/problemsuggestion"
	"github.com/CLAOJ/claoj/service/role"
	"github.com/CLAOJ/claoj/service/submission"
	"github.com/CLAOJ/claoj/service/user"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Service instances - initialized lazily
var (
	problemService      *problem.ProblemService
	orgService          *organization.OrganizationService
	suggestionService   *problemsuggestion.ProblemSuggestionService
	commentService      *comment.CommentService
	blogPostService     *blogpost.BlogPostService
	licenseService      *license.LicenseService
	navigationService   *navigation.NavigationService
	miscConfigService   *miscconfig.MiscConfigService
	auditLogService     *auditlog.AuditLogService
	languageService     *language.LanguageService
	problemGroupService *problemgroup.ProblemGroupService
	problemTypeService  *problemtype.ProblemTypeService
	submissionService   *submission.SubmissionService
	contestService      *contest.ContestService
	userService         *user.UserService
	roleService         *role.RoleService
)

// getProblemService returns the problem service instance
func getProblemService() *problem.ProblemService {
	if problemService == nil {
		problemService = problem.NewProblemService()
	}
	return problemService
}

// getOrgService returns the organization service instance
func getOrgService() *organization.OrganizationService {
	if orgService == nil {
		orgService = organization.NewOrganizationService()
	}
	return orgService
}

// getSuggestionService returns the problem suggestion service instance
func getSuggestionService() *problemsuggestion.ProblemSuggestionService {
	if suggestionService == nil {
		suggestionService = problemsuggestion.NewProblemSuggestionService()
	}
	return suggestionService
}

// getCommentService returns the comment service instance
func getCommentService() *comment.CommentService {
	if commentService == nil {
		commentService = comment.NewCommentService()
	}
	return commentService
}

// getBlogPostService returns the blog post service instance
func getBlogPostService() *blogpost.BlogPostService {
	if blogPostService == nil {
		blogPostService = blogpost.NewBlogPostService()
	}
	return blogPostService
}

// getLicenseService returns the license service instance
func getLicenseService() *license.LicenseService {
	if licenseService == nil {
		licenseService = license.NewLicenseService()
	}
	return licenseService
}

// getNavigationService returns the navigation service instance
func getNavigationService() *navigation.NavigationService {
	if navigationService == nil {
		navigationService = navigation.NewNavigationService()
	}
	return navigationService
}

// getMiscConfigService returns the misc config service instance
func getMiscConfigService() *miscconfig.MiscConfigService {
	if miscConfigService == nil {
		miscConfigService = miscconfig.NewMiscConfigService()
	}
	return miscConfigService
}

// getAuditLogService returns the audit log service instance
func getAuditLogService() *auditlog.AuditLogService {
	if auditLogService == nil {
		auditLogService = auditlog.NewAuditLogService()
	}
	return auditLogService
}

// getLanguageService returns the language service instance
func getLanguageService() *language.LanguageService {
	if languageService == nil {
		languageService = language.NewLanguageService()
	}
	return languageService
}

// getProblemGroupService returns the problem group service instance
func getProblemGroupService() *problemgroup.ProblemGroupService {
	if problemGroupService == nil {
		problemGroupService = problemgroup.NewProblemGroupService()
	}
	return problemGroupService
}

// getProblemTypeService returns the problem type service instance
func getProblemTypeService() *problemtype.ProblemTypeService {
	if problemTypeService == nil {
		problemTypeService = problemtype.NewProblemTypeService()
	}
	return problemTypeService
}

// getSubmissionService returns the submission service instance
func getSubmissionService() *submission.SubmissionService {
	if submissionService == nil {
		submissionService = submission.NewSubmissionService(bridgeServerRef)
	}
	return submissionService
}

// getContestService returns the contest service instance
func getContestService() *contest.ContestService {
	if contestService == nil {
		contestService = contest.NewContestService()
	}
	return contestService
}

// getUserService returns the user service instance
func getUserService() *user.UserService {
	if userService == nil {
		userService = user.NewUserService()
	}
	return userService
}

// getRoleService returns the role service instance
func getRoleService() *role.RoleService {
	if roleService == nil {
		roleService = role.NewRoleService()
	}
	return roleService
}

// countRecords counts the number of records in a table
func countRecords(model interface{}) (int64, error) {
	var count int64
	return count, db.DB.Model(model).Count(&count).Error
}

// parseRFC3339 parses an RFC3339 formatted time string
func parseRFC3339(s string) (time.Time, error) {
	return time.Parse(time.RFC3339, s)
}

// bridgeServerRef is set by main.go to allow API handlers to access bridge functions
var bridgeServerRef interface {
	Abort(subID uint) error
}

// SetBridgeServer sets the bridge server reference for API handlers
func SetBridgeServer(server interface {
	Abort(subID uint) error
}) {
	bridgeServerRef = server
}

// ============================================================
// REMAINING ADMIN FUNCTIONS
// ============================================================

// AdminProblemClone - POST /api/v2/admin/problem/:code/clone
// Clone an existing problem with a new code
func AdminProblemClone(c *gin.Context) {
	code := c.Param("code")
	
	// Get source problem
	var sourceProblem models.Problem
	if err := db.DB.Where("code = ?", code).First(&sourceProblem).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, apiError("problem not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	// Parse input
	var input struct {
		NewCode        string `json:"new_code" binding:"required"`
		NewName        string `json:"new_name" binding:"required"`
		CopyData       bool   `json:"copy_data"`        // Copy test cases and data files
		CopyAuthors    bool   `json:"copy_authors"`     // Copy authors, curators, testers
		CopySettings   bool   `json:"copy_settings"`    // Copy allowed languages, types, organizations
		NewDescription string `json:"new_description"`  // Optional new description
		NewSummary     string `json:"new_summary"`      // Optional new summary
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, apiError(err.Error()))
		return
	}

	// Check if new code already exists
	var existing models.Problem
	if err := db.DB.Where("code = ?", input.NewCode).First(&existing).Error; err == nil {
		c.JSON(http.StatusBadRequest, apiError("problem code already exists"))
		return
	}

	// Use provided description/summary or copy from source
	description := sourceProblem.Description
	if input.NewDescription != "" {
		description = input.NewDescription
	}
	summary := sourceProblem.Summary
	if input.NewSummary != "" {
		summary = input.NewSummary
	}

	// Create new problem
	newProblem := models.Problem{
		Code:                           input.NewCode,
		Name:                           sanitization.SanitizeTitle(input.NewName),
		Source:                         sourceProblem.Source,
		Description:                    description,
		PdfURL:                         sourceProblem.PdfURL,
		GroupID:                        sourceProblem.GroupID,
		TimeLimit:                      sourceProblem.TimeLimit,
		MemoryLimit:                    sourceProblem.MemoryLimit,
		ShortCircuit:                   sourceProblem.ShortCircuit,
		Points:                         sourceProblem.Points,
		Partial:                        sourceProblem.Partial,
		IsPublic:                       false, // Start as not public
		IsManuallyManaged:              sourceProblem.IsManuallyManaged,
		LicenseID:                      sourceProblem.LicenseID,
		OgImage:                        sourceProblem.OgImage,
		Summary:                        summary,
		IsFullMarkup:                   sourceProblem.IsFullMarkup,
		SubmissionSourceVisibilityMode: sourceProblem.SubmissionSourceVisibilityMode,
		TestcaseVisibilityMode:         sourceProblem.TestcaseVisibilityMode,
		IsOrganizationPrivate:          sourceProblem.IsOrganizationPrivate,
		// Don't copy: UserCount, AcRate, Date, SuggesterID, SuggestionStatus, etc.
	}

	if err := db.DB.Create(&newProblem).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	// Copy authors, curators, testers
	if input.CopyAuthors {
		var authors []models.Profile
		db.DB.Model(&sourceProblem).Association("Authors").Find(&authors)
		if len(authors) > 0 {
			db.DB.Model(&newProblem).Association("Authors").Append(&authors)
		}

		var curators []models.Profile
		db.DB.Model(&sourceProblem).Association("Curators").Find(&curators)
		if len(curators) > 0 {
			db.DB.Model(&newProblem).Association("Curators").Append(&curators)
		}

		var testers []models.Profile
		db.DB.Model(&sourceProblem).Association("Testers").Find(&testers)
		if len(testers) > 0 {
			db.DB.Model(&newProblem).Association("Testers").Append(&testers)
		}
	}

	// Copy types, allowed languages, organizations
	if input.CopySettings {
		var types []models.ProblemType
		db.DB.Model(&sourceProblem).Association("Types").Find(&types)
		if len(types) > 0 {
			db.DB.Model(&newProblem).Association("Types").Append(&types)
		}

		var allowedLangs []models.Language
		db.DB.Model(&sourceProblem).Association("AllowedLangs").Find(&allowedLangs)
		if len(allowedLangs) > 0 {
			db.DB.Model(&newProblem).Association("AllowedLangs").Append(&allowedLangs)
		}

		var organizations []models.Organization
		db.DB.Model(&sourceProblem).Association("Organizations").Find(&organizations)
		if len(organizations) > 0 {
			db.DB.Model(&newProblem).Association("Organizations").Append(&organizations)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "problem cloned successfully",
		"new_problem": gin.H{
			"id":   newProblem.ID,
			"code": newProblem.Code,
			"name": newProblem.Name,
		},
	})
}
