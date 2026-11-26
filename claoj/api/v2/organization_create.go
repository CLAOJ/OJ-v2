package v2

import (
	"net/http"

	"github.com/CLAOJ/claoj/auth"
	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/gin-gonic/gin"
)

// ============================================================
// ORGANIZATION-SCOPED CONTENT CREATION
//
// v1 ProblemCreateOrganization, ContestCreateOrganization and
// BlogPostCreateOrganization: the create permission plus organization-admin
// membership (v1 CustomAdminOrganizationMixin, 403 otherwise), with the
// organization forced onto the new object. These are NOT staff routes; the
// staff-only /admin/* create handlers keep their own gates.
// ============================================================

// orgCreateAccess applies the v1 gate: the create permission, and either
// administration of this organization or the matching edit-all bypass. It
// writes the 403 itself.
func orgCreateAccess(c *gin.Context, org models.Organization, perm, editAllPerm string) (models.Profile, bool) {
	_, profile, ok := resolveUserProfile(c)
	if !ok {
		return models.Profile{}, false
	}
	if !auth.HasPerm(c, perm) {
		c.JSON(http.StatusForbidden, apiUserError("you do not have permission to create this in an organization"))
		return models.Profile{}, false
	}
	if !isOrgAdmin(db.DB, org.ID, profile.ID) && !auth.HasPerm(c, editAllPerm) {
		c.JSON(http.StatusForbidden, apiUserError("access denied: not an organization administrator"))
		return models.Profile{}, false
	}
	return *profile, true
}

// withProfileID appends id when the list does not already contain it.
func withProfileID(ids []uint, id uint) []uint {
	for _, existing := range ids {
		if existing == id {
			return ids
		}
	}
	return append(ids, id)
}

// OrganizationProblemCreate - POST /api/v2/organization/:id/problems
func OrganizationProblemCreate(c *gin.Context) {
	org, ok := loadOrganization(c)
	if !ok {
		return
	}
	profile, ok := orgCreateAccess(c, org, "judge.create_organization_problem", "judge.edit_all_problem")
	if !ok {
		return
	}
	var input adminProblemCreateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, apiBindError(err))
		return
	}
	createProblemFromInput(c, input, &org, profile.ID)
}

// OrganizationContestCreate - POST /api/v2/organization/:id/contests
func OrganizationContestCreate(c *gin.Context) {
	org, ok := loadOrganization(c)
	if !ok {
		return
	}
	profile, ok := orgCreateAccess(c, org, "judge.create_private_contest", "judge.edit_all_contest")
	if !ok {
		return
	}
	var input adminContestCreateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, apiBindError(err))
		return
	}
	createContestFromInput(c, input, &org, profile.ID)
}

// OrganizationBlogPostCreate - POST /api/v2/organization/:id/blog-posts
func OrganizationBlogPostCreate(c *gin.Context) {
	org, ok := loadOrganization(c)
	if !ok {
		return
	}
	profile, ok := orgCreateAccess(c, org, "judge.edit_organization_post", "judge.edit_all_post")
	if !ok {
		return
	}
	var req AdminBlogPostCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, apiBindError(err))
		return
	}
	createBlogPostFromInput(c, req, profile.ID, &org)
}
