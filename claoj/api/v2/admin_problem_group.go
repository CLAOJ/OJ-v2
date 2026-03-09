package v2

import (
	"errors"
	"net/http"

	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/CLAOJ/claoj/service/problemgroup"
	"github.com/gin-gonic/gin"
)

// ============================================================
// ADMIN PROBLEM GROUP MANAGEMENT API
// ============================================================

// AdminProblemGroupList - GET /api/v2/admin/problem-groups
// List all problem groups
func AdminProblemGroupList(c *gin.Context) {
	resp, err := getProblemGroupService().ListGroups()
	if err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	type ProblemGroupItem struct {
		ID       uint   `json:"id"`
		Name     string `json:"name"`
		FullName string `json:"full_name"`
	}

	items := make([]ProblemGroupItem, len(resp.Groups))
	for i, g := range resp.Groups {
		items[i] = ProblemGroupItem{
			ID:       g.ID,
			Name:     g.Name,
			FullName: g.FullName,
		}
	}

	c.JSON(http.StatusOK, apiListWithTotal(items, resp.Total))
}

// AdminProblemGroupDetail - GET /api/v2/admin/problem-group/:id
// Get problem group detail
func AdminProblemGroupDetail(c *gin.Context) {
	idStr := c.Param("id")
	var id uint
	if err := parseUint(idStr, &id); err != nil {
		c.JSON(http.StatusBadRequest, apiError("invalid problem group id"))
		return
	}

	group, err := getProblemGroupService().GetGroup(problemgroup.GetGroupRequest{
		GroupID: id,
	})
	if err != nil {
		if errors.Is(err, problemgroup.ErrGroupNotFound) {
			c.JSON(http.StatusNotFound, apiError("problem group not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":        group.ID,
		"name":      group.Name,
		"full_name": group.FullName,
	})
}

// AdminProblemGroupCreateRequest - POST /api/v2/admin/problem-groups
type AdminProblemGroupCreateRequest struct {
	Name     string `json:"name" binding:"required"`
	FullName string `json:"full_name" binding:"required"`
}

// AdminProblemGroupCreate - POST /api/v2/admin/problem-groups
// Create a new problem group
func AdminProblemGroupCreate(c *gin.Context) {
	user, _, ok := resolveUserProfile(c)
	if !ok {
		return
	}

	if !user.IsSuperuser {
		c.JSON(http.StatusForbidden, apiError("admin access required"))
		return
	}

	var req AdminProblemGroupCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, apiError(err.Error()))
		return
	}

	group, err := getProblemGroupService().CreateGroup(problemgroup.CreateGroupRequest{
		Name:     req.Name,
		FullName: req.FullName,
	})
	if err != nil {
		if errors.Is(err, problemgroup.ErrGroupNameExists) {
			c.JSON(http.StatusBadRequest, apiError("problem group name already exists"))
			return
		}
		if errors.Is(err, problemgroup.ErrEmptyGroupName) {
			c.JSON(http.StatusBadRequest, apiError("problem group name cannot be empty"))
			return
		}
		if errors.Is(err, problemgroup.ErrEmptyGroupFullName) {
			c.JSON(http.StatusBadRequest, apiError("problem group full name cannot be empty"))
			return
		}
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "problem group created",
		"group":   gin.H{"id": group.ID, "name": group.Name},
	})
}

// AdminProblemGroupUpdateRequest - PATCH /api/v2/admin/problem-group/:id
type AdminProblemGroupUpdateRequest struct {
	FullName *string `json:"full_name"`
}

// AdminProblemGroupUpdate - PATCH /api/v2/admin/problem-group/:id
// Update a problem group
func AdminProblemGroupUpdate(c *gin.Context) {
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
		c.JSON(http.StatusBadRequest, apiError("invalid problem group id"))
		return
	}

	var req AdminProblemGroupUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, apiError(err.Error()))
		return
	}

	group, err := getProblemGroupService().UpdateGroup(problemgroup.UpdateGroupRequest{
		GroupID:  id,
		FullName: req.FullName,
	})
	if err != nil {
		if errors.Is(err, problemgroup.ErrGroupNotFound) {
			c.JSON(http.StatusNotFound, apiError("problem group not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "problem group updated",
		"group":   gin.H{"id": group.ID, "name": group.Name},
	})
}

// AdminProblemGroupDelete - DELETE /api/v2/admin/problem-group/:id
// Delete a problem group
func AdminProblemGroupDelete(c *gin.Context) {
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
		c.JSON(http.StatusBadRequest, apiError("invalid problem group id"))
		return
	}

	// Check if group is used by problems
	var problemCount int64
	db.DB.Model(&models.Problem{}).Where("group_id = ?", id).Count(&problemCount)
	if problemCount > 0 {
		c.JSON(http.StatusBadRequest, apiError("cannot delete problem group used by problems"))
		return
	}

	if err := getProblemGroupService().DeleteGroup(problemgroup.DeleteGroupRequest{
		GroupID: id,
	}); err != nil {
		if errors.Is(err, problemgroup.ErrGroupNotFound) {
			c.JSON(http.StatusNotFound, apiError("problem group not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "problem group deleted",
	})
}
