package v2

import (
	"errors"
	"net/http"

	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/CLAOJ/claoj/service/group"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ============================================================
// ADMIN DJANGO GROUP (ROLE) & PERMISSION MANAGEMENT API
//
// Writes rows into Django's own auth_group / auth_group_permissions /
// auth_user_groups tables, exactly like Django admin does.
// ============================================================

// AdminGroupList - GET /api/admin/groups
func AdminGroupList(c *gin.Context) {
	groups, err := getGroupService().ListGroups()
	if err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	type Item struct {
		ID              uint   `json:"id"`
		Name            string `json:"name"`
		UserCount       int    `json:"user_count"`
		PermissionCount int    `json:"permission_count"`
	}
	items := make([]Item, len(groups))
	for i, g := range groups {
		items[i] = Item{ID: g.ID, Name: g.Name, UserCount: g.UserCount, PermissionCount: g.PermissionCount}
	}
	c.JSON(http.StatusOK, apiList(items))
}

// AdminGroupDetail - GET /api/admin/group/:id
func AdminGroupDetail(c *gin.Context) {
	var id uint
	if err := parseUint(c.Param("id"), &id); err != nil {
		c.JSON(http.StatusBadRequest, apiError("invalid group id"))
		return
	}

	detail, err := getGroupService().GetGroup(id)
	if err != nil {
		if errors.Is(err, group.ErrGroupNotFound) {
			c.JSON(http.StatusNotFound, apiError("group not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	type UserItem struct {
		ID       uint   `json:"id"`
		Username string `json:"username"`
	}
	users := make([]UserItem, len(detail.Users))
	for i, u := range detail.Users {
		users[i] = UserItem{ID: u.ID, Username: u.Username}
	}

	c.JSON(http.StatusOK, gin.H{
		"id":             detail.ID,
		"name":           detail.Name,
		"permission_ids": detail.PermissionIDs,
		"users":          users,
	})
}

// AdminGroupCreate - POST /api/admin/groups
func AdminGroupCreate(c *gin.Context) {
	var input struct {
		Name          string `json:"name" binding:"required"`
		PermissionIDs []uint `json:"permission_ids"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, apiError(err.Error()))
		return
	}

	created, err := getGroupService().CreateGroup(input.Name, input.PermissionIDs)
	if err != nil {
		if errors.Is(err, group.ErrGroupNameExists) {
			c.JSON(http.StatusBadRequest, apiError("group name already exists"))
			return
		}
		if errors.Is(err, group.ErrEmptyGroupName) {
			c.JSON(http.StatusBadRequest, apiError("group name cannot be empty"))
			return
		}
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	c.JSON(http.StatusCreated, gin.H{"id": created.ID, "name": created.Name})
}

// AdminGroupUpdate - PATCH /api/admin/group/:id
func AdminGroupUpdate(c *gin.Context) {
	var id uint
	if err := parseUint(c.Param("id"), &id); err != nil {
		c.JSON(http.StatusBadRequest, apiError("invalid group id"))
		return
	}

	var input struct {
		Name          *string `json:"name"`
		PermissionIDs *[]uint `json:"permission_ids"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, apiError(err.Error()))
		return
	}

	if err := getGroupService().UpdateGroup(id, input.Name, input.PermissionIDs); err != nil {
		if errors.Is(err, group.ErrGroupNotFound) {
			c.JSON(http.StatusNotFound, apiError("group not found"))
			return
		}
		if errors.Is(err, group.ErrGroupNameExists) {
			c.JSON(http.StatusBadRequest, apiError("group name already exists"))
			return
		}
		if errors.Is(err, group.ErrEmptyGroupName) {
			c.JSON(http.StatusBadRequest, apiError("group name cannot be empty"))
			return
		}
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// AdminGroupDelete - DELETE /api/admin/group/:id
func AdminGroupDelete(c *gin.Context) {
	var id uint
	if err := parseUint(c.Param("id"), &id); err != nil {
		c.JSON(http.StatusBadRequest, apiError("invalid group id"))
		return
	}

	if err := getGroupService().DeleteGroup(id); err != nil {
		if errors.Is(err, group.ErrGroupNotFound) {
			c.JSON(http.StatusNotFound, apiError("group not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	c.Status(http.StatusNoContent)
}

// AdminPermissionList - GET /api/admin/permissions
func AdminPermissionList(c *gin.Context) {
	perms, err := getGroupService().ListPermissions()
	if err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	type Item struct {
		ID       uint   `json:"id"`
		Codename string `json:"codename"`
		Name     string `json:"name"`
		AppLabel string `json:"app_label"`
		Model    string `json:"model"`
	}
	items := make([]Item, len(perms))
	for i, p := range perms {
		items[i] = Item{ID: p.ID, Codename: p.Codename, Name: p.Name, AppLabel: p.AppLabel, Model: p.Model}
	}
	c.JSON(http.StatusOK, apiList(items))
}

// resolveAuthUserID resolves the :id route param used across the
// /admin/user/:id/... family (a judge_profile.id, per the established
// convention in admin_users_roles.go) to the corresponding auth_user.id --
// the identity Django's auth_user_groups join table actually stores.
// Returns (authUserID, true) on success; on failure it has already written
// the error response and the caller must return.
func resolveAuthUserID(c *gin.Context, profileID uint) (uint, bool) {
	var profile models.Profile
	if err := db.DB.Select("user_id").First(&profile, profileID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, apiError("user not found"))
			return 0, false
		}
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return 0, false
	}
	return profile.UserID, true
}

// AdminUserAddGroup - POST /api/admin/user/:id/groups
func AdminUserAddGroup(c *gin.Context) {
	var profileID uint
	if err := parseUint(c.Param("id"), &profileID); err != nil {
		c.JSON(http.StatusBadRequest, apiError("invalid user id"))
		return
	}

	var input struct {
		GroupID uint `json:"group_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, apiError(err.Error()))
		return
	}

	authUserID, ok := resolveAuthUserID(c, profileID)
	if !ok {
		return
	}

	if err := getGroupService().AddUserToGroup(authUserID, input.GroupID); err != nil {
		if errors.Is(err, group.ErrGroupNotFound) {
			c.JSON(http.StatusNotFound, apiError("group not found"))
			return
		}
		if errors.Is(err, group.ErrUserNotFound) {
			c.JSON(http.StatusNotFound, apiError("user not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	c.Status(http.StatusNoContent)
}

// AdminUserRemoveGroup - DELETE /api/admin/user/:id/groups/:groupId
func AdminUserRemoveGroup(c *gin.Context) {
	var profileID uint
	if err := parseUint(c.Param("id"), &profileID); err != nil {
		c.JSON(http.StatusBadRequest, apiError("invalid user id"))
		return
	}
	var groupID uint
	if err := parseUint(c.Param("groupId"), &groupID); err != nil {
		c.JSON(http.StatusBadRequest, apiError("invalid group id"))
		return
	}

	authUserID, ok := resolveAuthUserID(c, profileID)
	if !ok {
		return
	}

	if err := getGroupService().RemoveUserFromGroup(authUserID, groupID); err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	c.Status(http.StatusNoContent)
}
