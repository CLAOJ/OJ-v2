package v2

import (
	"errors"
	"net/http"

	"github.com/CLAOJ/claoj/auth"
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
	if !auth.GetAccess(c).IsSuperuser {
		c.JSON(http.StatusForbidden, apiError("superuser access required"))
		return
	}

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
		if errors.Is(err, group.ErrUnknownPermissionID) {
			c.JSON(http.StatusBadRequest, apiError("one or more permission IDs do not exist"))
			return
		}
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	c.JSON(http.StatusCreated, gin.H{"id": created.ID, "name": created.Name})
}

// AdminGroupUpdate - PATCH /api/admin/group/:id
func AdminGroupUpdate(c *gin.Context) {
	if !auth.GetAccess(c).IsSuperuser {
		c.JSON(http.StatusForbidden, apiError("superuser access required"))
		return
	}

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
		if errors.Is(err, group.ErrUnknownPermissionID) {
			c.JSON(http.StatusBadRequest, apiError("one or more permission IDs do not exist"))
			return
		}
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// AdminGroupDelete - DELETE /api/admin/group/:id
func AdminGroupDelete(c *gin.Context) {
	if !auth.GetAccess(c).IsSuperuser {
		c.JSON(http.StatusForbidden, apiError("superuser access required"))
		return
	}

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
	if !auth.GetAccess(c).IsSuperuser {
		c.JSON(http.StatusForbidden, apiError("superuser access required"))
		return
	}

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
	if !auth.GetAccess(c).IsSuperuser {
		c.JSON(http.StatusForbidden, apiError("superuser access required"))
		return
	}

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
		if errors.Is(err, group.ErrGroupNotFound) {
			c.JSON(http.StatusNotFound, apiError("group not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	c.Status(http.StatusNoContent)
}

// AdminGroupMembers - GET /api/admin/group/:id/members
//
// Paginated member list (search by username or email). Readable by any
// staff member; membership mutations stay superuser-only.
func AdminGroupMembers(c *gin.Context) {
	var groupID uint
	if err := parseUint(c.Param("id"), &groupID); err != nil {
		c.JSON(http.StatusBadRequest, apiError("invalid group id"))
		return
	}
	page, pageSize := parsePagination(c)

	resp, err := getGroupService().ListMembers(group.ListMembersRequest{
		GroupID:  groupID,
		Page:     page,
		PageSize: pageSize,
		Search:   c.Query("search"),
	})
	if err != nil {
		if errors.Is(err, group.ErrGroupNotFound) {
			c.JSON(http.StatusNotFound, apiError("group not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	type Item struct {
		ID       uint    `json:"id"`
		Username string  `json:"username"`
		Email    string  `json:"email"`
		IsActive bool    `json:"is_active"`
		Points   float64 `json:"points"`
	}
	items := make([]Item, len(resp.Members))
	for i, m := range resp.Members {
		items[i] = Item{ID: m.ID, Username: m.Username, Email: m.Email, IsActive: m.IsActive, Points: m.Points}
	}

	out := PaginationResponseWithPage(items, resp.Total, resp.Page, resp.PageSize)
	out["group_id"] = resp.GroupID
	out["group_name"] = resp.GroupName
	c.JSON(http.StatusOK, out)
}

// AdminGroupAddMember - POST /api/admin/group/:id/members
//
// Body: {"user_id": <judge_profile.id>}. Additive rather than a PUT replace:
// replacing the set would delete and recreate every membership row, which is
// racy for large groups and wipes members added concurrently.
func AdminGroupAddMember(c *gin.Context) {
	if !auth.GetAccess(c).IsSuperuser {
		c.JSON(http.StatusForbidden, apiError("superuser access required"))
		return
	}

	var groupID uint
	if err := parseUint(c.Param("id"), &groupID); err != nil {
		c.JSON(http.StatusBadRequest, apiError("invalid group id"))
		return
	}

	var input struct {
		// UserID is a judge_profile.id (the id the admin user APIs and
		// pickers return), not an auth_user.id.
		UserID uint `json:"user_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, apiError(err.Error()))
		return
	}

	authUserID, ok := resolveAuthUserID(c, input.UserID)
	if !ok {
		return
	}

	if err := getGroupService().AddUserToGroup(authUserID, groupID); err != nil {
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

// AdminGroupRemoveMember - DELETE /api/admin/group/:id/members/:userId
//
// :userId is a judge_profile.id, like AdminGroupAddMember's body.
func AdminGroupRemoveMember(c *gin.Context) {
	if !auth.GetAccess(c).IsSuperuser {
		c.JSON(http.StatusForbidden, apiError("superuser access required"))
		return
	}

	var groupID uint
	if err := parseUint(c.Param("id"), &groupID); err != nil {
		c.JSON(http.StatusBadRequest, apiError("invalid group id"))
		return
	}
	var profileID uint
	if err := parseUint(c.Param("userId"), &profileID); err != nil {
		c.JSON(http.StatusBadRequest, apiError("invalid user id"))
		return
	}

	authUserID, ok := resolveAuthUserID(c, profileID)
	if !ok {
		return
	}

	if err := getGroupService().RemoveUserFromGroup(authUserID, groupID); err != nil {
		if errors.Is(err, group.ErrGroupNotFound) {
			c.JSON(http.StatusNotFound, apiError("group not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	c.Status(http.StatusNoContent)
}
