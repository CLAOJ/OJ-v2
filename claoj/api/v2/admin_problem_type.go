package v2

import (
	"errors"
	"net/http"
	"strings"

	"github.com/CLAOJ/claoj/service/problemtype"
	"github.com/gin-gonic/gin"
)

// ============================================================
// ADMIN PROBLEM TYPE MANAGEMENT API
// ============================================================

// AdminProblemTypeList - GET /api/v2/admin/problem-types
// List all problem types
func AdminProblemTypeList(c *gin.Context) {
	resp, err := getProblemTypeService().ListTypes(problemtype.ListTypesRequest{
		Search: strings.TrimSpace(c.Query("search")),
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	type ProblemTypeItem struct {
		ID           uint   `json:"id"`
		Name         string `json:"name"`
		FullName     string `json:"full_name"`
		ProblemCount int64  `json:"problem_count"`
	}

	items := make([]ProblemTypeItem, len(resp.Types))
	for i, t := range resp.Types {
		items[i] = ProblemTypeItem{
			ID:           t.ID,
			Name:         t.Name,
			FullName:     t.FullName,
			ProblemCount: t.ProblemCount,
		}
	}

	c.JSON(http.StatusOK, apiListWithTotal(items, resp.Total))
}

// AdminProblemTypeDetail - GET /api/v2/admin/problem-type/:id
// Get problem type detail
func AdminProblemTypeDetail(c *gin.Context) {
	idStr := c.Param("id")
	var id uint
	if err := parseUint(idStr, &id); err != nil {
		c.JSON(http.StatusBadRequest, apiError("invalid problem type id"))
		return
	}

	ptype, err := getProblemTypeService().GetType(problemtype.GetTypeRequest{
		TypeID: id,
	})
	if err != nil {
		if errors.Is(err, problemtype.ErrTypeNotFound) {
			c.JSON(http.StatusNotFound, apiError("problem type not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	problems, err := getProblemTypeService().GetTypeProblems(ptype.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}
	refs := make([]gin.H, len(problems))
	for i, p := range problems {
		refs[i] = gin.H{"id": p.ID, "code": p.Code, "name": p.Name}
	}

	problemIDs := ptype.ProblemIDs
	if problemIDs == nil {
		problemIDs = []uint{}
	}

	c.JSON(http.StatusOK, gin.H{
		"id":            ptype.ID,
		"name":          ptype.Name,
		"full_name":     ptype.FullName,
		"problem_ids":   problemIDs,
		"problem_count": ptype.ProblemCount,
		"problems":      refs,
	})
}

// AdminProblemTypeCreateRequest - POST /api/v2/admin/problem-types
type AdminProblemTypeCreateRequest struct {
	Name       string  `json:"name" binding:"required"`
	FullName   string  `json:"full_name" binding:"required"`
	ProblemIDs *[]uint `json:"problem_ids"`
}

// AdminProblemTypeCreate - POST /api/v2/admin/problem-types
// Create a new problem type
func AdminProblemTypeCreate(c *gin.Context) {
	user, _, ok := resolveUserProfile(c)
	if !ok {
		return
	}

	if !user.IsSuperuser {
		c.JSON(http.StatusForbidden, apiError("admin access required"))
		return
	}

	var req AdminProblemTypeCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, apiError(err.Error()))
		return
	}

	ptype, err := getProblemTypeService().CreateType(problemtype.CreateTypeRequest{
		Name:       req.Name,
		FullName:   req.FullName,
		ProblemIDs: req.ProblemIDs,
	})
	if err != nil {
		if errors.Is(err, problemtype.ErrTypeNameExists) {
			c.JSON(http.StatusBadRequest, apiError("problem type name already exists"))
			return
		}
		if errors.Is(err, problemtype.ErrEmptyTypeName) {
			c.JSON(http.StatusBadRequest, apiError("problem type name cannot be empty"))
			return
		}
		if errors.Is(err, problemtype.ErrEmptyTypeFullName) {
			c.JSON(http.StatusBadRequest, apiError("problem type full name cannot be empty"))
			return
		}
		if errors.Is(err, problemtype.ErrUnknownProblemIDs) {
			c.JSON(http.StatusBadRequest, apiUserError(err.Error()))
			return
		}
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "problem type created",
		"type":    gin.H{"id": ptype.ID, "name": ptype.Name},
	})
}

// AdminProblemTypeUpdateRequest - PATCH /api/v2/admin/problem-type/:id
type AdminProblemTypeUpdateRequest struct {
	FullName   *string `json:"full_name"`
	ProblemIDs *[]uint `json:"problem_ids"`
}

// AdminProblemTypeUpdate - PATCH /api/v2/admin/problem-type/:id
// Update a problem type
func AdminProblemTypeUpdate(c *gin.Context) {
	user, _, ok := resolveUserProfile(c)
	if !ok {
		return
	}

	if !user.IsSuperuser {
		c.JSON(http.StatusForbidden, apiError("admin access required"))
		return
	}

	idStr := c.Param("id")
	var id uint
	if err := parseUint(idStr, &id); err != nil {
		c.JSON(http.StatusBadRequest, apiError("invalid problem type id"))
		return
	}

	var req AdminProblemTypeUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, apiError(err.Error()))
		return
	}

	ptype, err := getProblemTypeService().UpdateType(problemtype.UpdateTypeRequest{
		TypeID:     id,
		FullName:   req.FullName,
		ProblemIDs: req.ProblemIDs,
	})
	if err != nil {
		if errors.Is(err, problemtype.ErrTypeNotFound) {
			c.JSON(http.StatusNotFound, apiError("problem type not found"))
			return
		}
		if errors.Is(err, problemtype.ErrUnknownProblemIDs) {
			c.JSON(http.StatusBadRequest, apiUserError(err.Error()))
			return
		}
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "problem type updated",
		"type":    gin.H{"id": ptype.ID, "name": ptype.Name},
	})
}

// AdminProblemTypeDelete - DELETE /api/v2/admin/problem-type/:id
// Delete a problem type
func AdminProblemTypeDelete(c *gin.Context) {
	user, _, ok := resolveUserProfile(c)
	if !ok {
		return
	}

	if !user.IsSuperuser {
		c.JSON(http.StatusForbidden, apiError("admin access required"))
		return
	}

	idStr := c.Param("id")
	var id uint
	if err := parseUint(idStr, &id); err != nil {
		c.JSON(http.StatusBadRequest, apiError("invalid problem type id"))
		return
	}

	if err := getProblemTypeService().DeleteType(problemtype.DeleteTypeRequest{
		TypeID: id,
	}); err != nil {
		if errors.Is(err, problemtype.ErrTypeNotFound) {
			c.JSON(http.StatusNotFound, apiError("problem type not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "problem type deleted",
	})
}