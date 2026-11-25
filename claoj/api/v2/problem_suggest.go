package v2

import (
	"errors"
	"net/http"
	"time"

	"github.com/CLAOJ/claoj/auth"
	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/CLAOJ/claoj/service/problem"
	"github.com/CLAOJ/claoj/service/relations"
	"github.com/gin-gonic/gin"
)

// SuggestProblemRequest - POST /api/v2/problems/suggest
// The reduced v1 ProblemSuggest form: no authors, no visibility toggle.
type SuggestProblemRequest struct {
	Code                   string  `json:"code" binding:"required"`
	Name                   string  `json:"name" binding:"required"`
	Description            string  `json:"description" binding:"required"`
	Points                 float64 `json:"points" binding:"required"`
	Partial                *bool   `json:"partial"`
	TimeLimit              float64 `json:"time_limit" binding:"required"`
	MemoryLimit            uint    `json:"memory_limit" binding:"required"`
	GroupID                uint    `json:"group_id" binding:"required"`
	TypeIDs                []uint  `json:"type_ids"`
	Source                 string  `json:"source"`
	TestcaseVisibilityMode string  `json:"testcase_visibility_mode"`
}

// SuggestProblemCreate - POST /api/v2/problems/suggest
// @Description Create a hidden problem suggestion (v1 ProblemSuggest). The
// @Description suggester is always the caller and the problem always stays
// @Description private until a reviewer publishes it.
// @Tags Problems
// @Summary Suggest a new problem
// @Accept json
// @Produce json
// @Param request body SuggestProblemRequest true "Suggestion fields"
// @Success 200 {object} map[string]interface{} "Created suggestion"
// @Failure 400 {object} map[string]interface{} "Invalid fields or duplicate code"
// @Failure 403 {object} map[string]interface{} "judge.suggest_new_problem required"
// @Router /problems/suggest [post]
func SuggestProblemCreate(c *gin.Context) {
	// Route-level RequirePerm normally answers first; kept here so the handler
	// is safe to mount anywhere.
	if !auth.HasPerm(c, "judge.suggest_new_problem") {
		c.JSON(http.StatusForbidden, apiUserError("you do not have permission to suggest problems"))
		return
	}

	var req SuggestProblemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, apiUserError(err.Error()))
		return
	}

	suggesterID, ok := auth.CurrentProfileID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, apiUserError("profile not found"))
		return
	}

	var existing int64
	if err := db.DB.Model(&models.Problem{}).Where("code = ?", req.Code).Count(&existing).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}
	if existing > 0 {
		c.JSON(http.StatusBadRequest, apiUserError("another problem with this code already exists"))
		return
	}

	// v1 forces every include_in_problem language onto a suggestion.
	var langIDs []uint
	if err := db.DB.Model(&models.Language{}).
		Where("include_in_problem = ?", true).
		Pluck("id", &langIDs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	partial := true
	if req.Partial != nil {
		partial = *req.Partial
	}
	now := time.Now()

	profile, err := getProblemService().CreateProblem(problem.CreateProblemRequest{
		Code:                   req.Code,
		Name:                   req.Name,
		Description:            req.Description,
		Points:                 req.Points,
		Partial:                partial,
		IsPublic:               false, // forced: suggestions stay hidden
		TimeLimit:              req.TimeLimit,
		MemoryLimit:            req.MemoryLimit,
		GroupID:                req.GroupID,
		TypeIDs:                req.TypeIDs,
		AllowedLangIDs:         langIDs,
		Source:                 req.Source,
		Date:                   &now, // v1 sets the suggestion date on create
		TestcaseVisibilityMode: req.TestcaseVisibilityMode,
		SuggesterID:            &suggesterID,
	})
	if err != nil {
		if errors.Is(err, problem.ErrInvalidVisibilityMode) || errors.Is(err, problem.ErrFieldTooLong) ||
			errors.Is(err, problem.ErrInvalidSuggester) || errors.Is(err, relations.ErrUnknownID) {
			c.JSON(http.StatusBadRequest, apiUserError(err.Error()))
			return
		}
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"problem": gin.H{"id": profile.ID, "code": profile.Code, "name": profile.Name},
	})
}
