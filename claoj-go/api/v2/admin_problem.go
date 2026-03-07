package v2

import (
	"errors"
	"net/http"

	"github.com/CLAOJ/claoj-go/service/problem"
	"github.com/gin-gonic/gin"
)

// ============================================================
// ADMIN PROBLEM MANAGEMENT API
// ============================================================

// AdminProblemList - GET /api/v2/admin/problems
func AdminProblemList(c *gin.Context) {
	page, pageSize := parsePagination(c)

	resp, err := getProblemService().ListProblems(problem.ListProblemsRequest{
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	type Item struct {
		ID                uint    `json:"id"`
		Code              string  `json:"code"`
		Name              string  `json:"name"`
		Points            float64 `json:"points"`
		Partial           bool    `json:"partial"`
		IsPublic          bool    `json:"is_public"`
		GroupName         string  `json:"group_name"`
		UserCount         int     `json:"user_count"`
		AcRate            float64 `json:"ac_rate"`
		IsManuallyManaged bool    `json:"is_manually_managed"`
	}
	items := make([]Item, len(resp.Problems))
	for i, p := range resp.Problems {
		items[i] = Item{
			ID:                p.ID,
			Code:              p.Code,
			Name:              p.Name,
			Points:            p.Points,
			Partial:           p.Partial,
			IsPublic:          p.IsPublic,
			GroupName:         p.GroupName,
			UserCount:         p.UserCount,
			AcRate:            p.AcRate,
			IsManuallyManaged: p.IsManuallyManaged,
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"data":      items,
		"total":     resp.Total,
		"page":      resp.Page,
		"page_size": resp.PageSize,
	})
}

// AdminProblemDetail - GET /api/v2/admin/problem/:code
func AdminProblemDetail(c *gin.Context) {
	code := c.Param("code")

	profile, err := getProblemService().GetProblem(problem.GetProblemRequest{
		ProblemCode: code,
	})
	if err != nil {
		if errors.Is(err, problem.ErrProblemNotFound) {
			c.JSON(http.StatusNotFound, apiError("problem not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"problem": profile,
	})
}

// AdminProblemCreate - POST /api/v2/admin/problems
func AdminProblemCreate(c *gin.Context) {
	var input struct {
		Code              string  `json:"code" binding:"required"`
		Name              string  `json:"name" binding:"required"`
		Description       string  `json:"description" binding:"required"`
		Points            float64 `json:"points" binding:"required"`
		Partial           bool    `json:"partial"`
		IsPublic          bool    `json:"is_public"`
		TimeLimit         float64 `json:"time_limit" binding:"required"`
		MemoryLimit       uint    `json:"memory_limit" binding:"required"`
		GroupID           uint    `json:"group_id"`
		TypeIDs           []uint  `json:"type_ids"`
		AuthorIDs         []uint  `json:"author_ids"`
		AllowedLangIDs    []uint  `json:"allowed_lang_ids"`
		IsManuallyManaged bool    `json:"is_manually_managed"`
		PdfURL            string  `json:"pdf_url"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, apiError(err.Error()))
		return
	}

	profile, err := getProblemService().CreateProblem(problem.CreateProblemRequest{
		Code:              input.Code,
		Name:              input.Name,
		Description:       input.Description,
		Points:            input.Points,
		Partial:           input.Partial,
		IsPublic:          input.IsPublic,
		TimeLimit:         input.TimeLimit,
		MemoryLimit:       input.MemoryLimit,
		GroupID:           input.GroupID,
		TypeIDs:           input.TypeIDs,
		AuthorIDs:         input.AuthorIDs,
		AllowedLangIDs:    input.AllowedLangIDs,
		IsManuallyManaged: input.IsManuallyManaged,
		PdfURL:            input.PdfURL,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"problem": gin.H{
			"id":   profile.ID,
			"code": profile.Code,
			"name": profile.Name,
		},
	})
}

// AdminProblemUpdate - PATCH /api/v2/admin/problem/:code
func AdminProblemUpdate(c *gin.Context) {
	code := c.Param("code")

	var input struct {
		Name              *string  `json:"name"`
		Description       *string  `json:"description"`
		Points            *float64 `json:"points"`
		Partial           *bool    `json:"partial"`
		IsPublic          *bool    `json:"is_public"`
		TimeLimit         *float64 `json:"time_limit"`
		MemoryLimit       *uint    `json:"memory_limit"`
		IsManuallyManaged *bool    `json:"is_manually_managed"`
		PdfURL            *string  `json:"pdf_url"`
		AddTypeIDs        []uint   `json:"add_type_ids"`
		RemoveTypeIDs     []uint   `json:"remove_type_ids"`
		AddAuthorIDs      []uint   `json:"add_author_ids"`
		RemoveAuthorIDs   []uint   `json:"remove_author_ids"`
		AddLangIDs        []uint   `json:"add_lang_ids"`
		RemoveLangIDs     []uint   `json:"remove_lang_ids"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, apiError(err.Error()))
		return
	}

	profile, err := getProblemService().UpdateProblem(problem.UpdateProblemRequest{
		ProblemCode:       code,
		Name:              input.Name,
		Description:       input.Description,
		Points:            input.Points,
		Partial:           input.Partial,
		IsPublic:          input.IsPublic,
		TimeLimit:         input.TimeLimit,
		MemoryLimit:       input.MemoryLimit,
		IsManuallyManaged: input.IsManuallyManaged,
		PdfURL:            input.PdfURL,
		AddTypeIDs:        input.AddTypeIDs,
		RemoveTypeIDs:     input.RemoveTypeIDs,
		AddAuthorIDs:      input.AddAuthorIDs,
		RemoveAuthorIDs:   input.RemoveAuthorIDs,
		AddLangIDs:        input.AddLangIDs,
		RemoveLangIDs:     input.RemoveLangIDs,
	})
	if err != nil {
		if errors.Is(err, problem.ErrProblemNotFound) {
			c.JSON(http.StatusNotFound, apiError("problem not found"))
			return
		}
		c.JSON(http.StatusBadRequest, apiError(err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"problem": profile,
	})
}

// AdminProblemDelete - DELETE /api/v2/admin/problem/:code
func AdminProblemDelete(c *gin.Context) {
	code := c.Param("code")

	if err := getProblemService().DeleteProblem(problem.DeleteProblemRequest{
		ProblemCode: code,
	}); err != nil {
		if errors.Is(err, problem.ErrProblemNotFound) {
			c.JSON(http.StatusNotFound, apiError("problem not found"))
			return
		}
		c.JSON(http.StatusBadRequest, apiError(err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Problem hidden (soft deleted)",
	})
}
