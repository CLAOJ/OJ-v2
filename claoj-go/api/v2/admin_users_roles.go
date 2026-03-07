package v2

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/CLAOJ/claoj-go/service/role"
	"github.com/CLAOJ/claoj-go/service/user"
	"github.com/gin-gonic/gin"
)

// ============================================================
// ADMIN USER MANAGEMENT API
// ============================================================

// AdminUserList - GET /api/v2/admin/users
func AdminUserList(c *gin.Context) {
	page, pageSize := parsePagination(c)

	resp, err := getUserService().ListUsers(user.ListUsersRequest{
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	type Item struct {
		ID             uint      `json:"id"`
		Username       string    `json:"username"`
		Email          string    `json:"email"`
		Points         float64   `json:"points"`
		PerformancePts float64   `json:"performance_points"`
		ProblemCount   int       `json:"problem_count"`
		Rating         *int      `json:"rating"`
		IsStaff        bool      `json:"is_staff"`
		IsSuperuser    bool      `json:"is_super_user"`
		IsActive       bool      `json:"is_active"`
		IsUnlisted     bool      `json:"is_unlisted"`
		IsMuted        bool      `json:"is_muted"`
		DateJoined     time.Time `json:"date_joined"`
		LastAccess     time.Time `json:"last_access"`
		DisplayRank    string    `json:"display_rank"`
		BanReason      *string   `json:"ban_reason"`
	}
	items := make([]Item, len(resp.Users))
	for i, u := range resp.Users {
		items[i] = Item{
			ID:             u.ID,
			Username:       u.Username,
			Email:          u.Email,
			Points:         u.Points,
			PerformancePts: u.PerformancePoints,
			ProblemCount:   u.ProblemCount,
			Rating:         u.Rating,
			IsStaff:        u.IsStaff,
			IsSuperuser:    u.IsSuperuser,
			IsActive:       u.IsActive,
			IsUnlisted:     u.IsUnlisted,
			IsMuted:        u.IsMuted,
			DateJoined:     u.DateJoined,
			LastAccess:     u.LastAccess,
			DisplayRank:    u.DisplayRank,
			BanReason:      u.BanReason,
		}
	}
	c.JSON(http.StatusOK, apiList(items))
}

// AdminUserDetail - GET /api/v2/admin/user/:id
func AdminUserDetail(c *gin.Context) {
	idParam := c.Param("id")
	userID, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, apiError("invalid user ID"))
		return
	}

	profile, err := getUserService().GetUser(user.GetUserRequest{
		UserID: uint(userID),
	})
	if err != nil {
		if errors.Is(err, user.ErrUserNotFound) {
			c.JSON(http.StatusNotFound, apiError("user not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	type OrgItem struct {
		ID   uint   `json:"id"`
		Name string `json:"name"`
	}
	orgs := make([]OrgItem, len(profile.OrganizationIDs))
	for i, id := range profile.OrganizationIDs {
		orgs[i] = OrgItem{ID: id}
	}

	c.JSON(http.StatusOK, gin.H{
		"id":                    profile.ID,
		"username":              profile.Username,
		"email":                 profile.Email,
		"display_name":          profile.DisplayName,
		"about":                 profile.About,
		"points":                profile.Points,
		"performance_points":    profile.PerformancePoints,
		"contribution_points":   profile.ContributionPoints,
		"rating":                profile.Rating,
		"problem_count":         profile.ProblemCount,
		"display_rank":          profile.DisplayRank,
		"is_staff":              profile.IsStaff,
		"is_super_user":         profile.IsSuperuser,
		"is_active":             profile.IsActive,
		"is_unlisted":           profile.IsUnlisted,
		"is_muted":              profile.IsMuted,
		"is_totp_enabled":       profile.IsTotpEnabled,
		"is_webauthn_enabled":   profile.IsWebauthnEnabled,
		"organizations":         orgs,
		"last_access":           profile.LastAccess,
		"date_joined":           profile.DateJoined,
		"ban_reason":            profile.BanReason,
	})
}

// AdminUserUpdate - PATCH /api/v2/admin/user/:id
func AdminUserUpdate(c *gin.Context) {
	idParam := c.Param("id")
	userID, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, apiError("invalid user ID"))
		return
	}

	var input struct {
		Email                    *string `json:"email"`
		DisplayName              *string `json:"display_name"`
		About                    *string `json:"about"`
		IsActive                 *bool   `json:"is_active"`
		IsUnlisted               *bool   `json:"is_unlisted"`
		IsMuted                  *bool   `json:"is_muted"`
		DisplayRank              *string `json:"display_rank"`
		BanReason                *string `json:"ban_reason"`
		RemoveOrganizationIDs    []uint  `json:"remove_organization_ids,omitempty"`
		AddOrganizationIDs       []uint  `json:"add_organization_ids,omitempty"`
		RemoveOrganizationAdmin  []uint  `json:"remove_organization_admin,omitempty"`
		AddOrganizationAdmin     []uint  `json:"add_organization_admin,omitempty"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, apiError(err.Error()))
		return
	}

	if err := getUserService().UpdateUser(user.UpdateUserRequest{
		UserID:                  uint(userID),
		Email:                   input.Email,
		DisplayName:             input.DisplayName,
		About:                   input.About,
		IsActive:                input.IsActive,
		IsUnlisted:              input.IsUnlisted,
		IsMuted:                 input.IsMuted,
		DisplayRank:             input.DisplayRank,
		BanReason:               input.BanReason,
		RemoveOrganizationIDs:   input.RemoveOrganizationIDs,
		AddOrganizationIDs:      input.AddOrganizationIDs,
		RemoveOrganizationAdmin: input.RemoveOrganizationAdmin,
		AddOrganizationAdmin:    input.AddOrganizationAdmin,
	}); err != nil {
		if errors.Is(err, user.ErrUserNotFound) {
			c.JSON(http.StatusNotFound, apiError("user not found"))
			return
		}
		c.JSON(http.StatusBadRequest, apiError(err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "User updated successfully",
	})
}

// AdminUserDelete - DELETE /api/v2/admin/user/:id
func AdminUserDelete(c *gin.Context) {
	idParam := c.Param("id")
	userID, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, apiError("invalid user ID"))
		return
	}

	if err := getUserService().DeleteUser(user.DeleteUserRequest{
		UserID: uint(userID),
	}); err != nil {
		if errors.Is(err, user.ErrUserNotFound) {
			c.JSON(http.StatusNotFound, apiError("user not found"))
			return
		}
		c.JSON(http.StatusBadRequest, apiError(err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "User deactivated and hidden (soft deleted)",
	})
}

// AdminUserBan - POST /api/v2/admin/user/:id/ban
func AdminUserBan(c *gin.Context) {
	idParam := c.Param("id")
	userID, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, apiError("invalid user ID"))
		return
	}

	var input struct {
		Reason string `json:"reason" binding:"required"`
		Day    int    `json:"day" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, apiError(err.Error()))
		return
	}

	if err := getUserService().BanUser(user.BanUserRequest{
		UserID: uint(userID),
		Reason: input.Reason,
	}); err != nil {
		if errors.Is(err, user.ErrUserNotFound) {
			c.JSON(http.StatusNotFound, apiError("user not found"))
			return
		}
		c.JSON(http.StatusBadRequest, apiError(err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "User banned successfully",
	})
}

// AdminUserUnban - POST /api/v2/admin/user/:id/unban
func AdminUserUnban(c *gin.Context) {
	idParam := c.Param("id")
	userID, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, apiError("invalid user ID"))
		return
	}

	if err := getUserService().UnbanUser(user.UnbanUserRequest{
		UserID: uint(userID),
	}); err != nil {
		if errors.Is(err, user.ErrUserNotFound) {
			c.JSON(http.StatusNotFound, apiError("user not found"))
			return
		}
		c.JSON(http.StatusBadRequest, apiError(err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "User unbanned successfully",
	})
}

// ============================================================
// ADMIN ROLES & PERMISSIONS MANAGEMENT API
// ============================================================

// AdminRoleList - GET /api/v2/admin/roles
func AdminRoleList(c *gin.Context) {
	resp, err := getRoleService().ListRoles(role.ListRolesRequest{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	type RoleItem struct {
		ID          uint   `json:"id"`
		Name        string `json:"name"`
		DisplayName string `json:"display_name"`
		Description string `json:"description"`
		Color       string `json:"color"`
		IsDefault   bool   `json:"is_default"`
	}

	items := make([]RoleItem, len(resp.Roles))
	for i, r := range resp.Roles {
		items[i] = RoleItem{
			ID:          r.ID,
			Name:        r.Name,
			DisplayName: r.DisplayName,
			Description: r.Description,
			Color:       r.Color,
			IsDefault:   r.IsDefault,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"data": items,
	})
}

// AdminRoleDetail - GET /api/v2/admin/role/:id
func AdminRoleDetail(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, apiError("invalid role ID"))
		return
	}

	detail, err := getRoleService().GetRole(role.GetRoleRequest{RoleID: uint(id)})
	if err != nil {
		if errors.Is(err, role.ErrRoleNotFound) {
			c.JSON(http.StatusNotFound, apiError("role not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	c.JSON(http.StatusOK, detail)
}

// AdminRoleCreate - POST /api/v2/admin/roles
func AdminRoleCreate(c *gin.Context) {
	var input struct {
		Name        string `json:"name" binding:"required"`
		DisplayName string `json:"display_name" binding:"required"`
		Description string `json:"description"`
		Color       string `json:"color"`
		IsDefault   bool   `json:"is_default"`
		PermissionIDs []uint `json:"permission_ids"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, apiError(err.Error()))
		return
	}

	r, err := getRoleService().CreateRole(role.CreateRoleRequest{
		Name:        input.Name,
		DisplayName: input.DisplayName,
		Description: input.Description,
		Color:       input.Color,
		IsDefault:   input.IsDefault,
		PermissionIDs: input.PermissionIDs,
	})
	if err != nil {
		if errors.Is(err, role.ErrRoleNameExists) {
			c.JSON(http.StatusBadRequest, apiError("role name already exists"))
			return
		}
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"role":    r,
	})
}

// AdminRoleUpdate - PATCH /api/v2/admin/role/:id
func AdminRoleUpdate(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, apiError("invalid role ID"))
		return
	}

	var input struct {
		DisplayName  string `json:"display_name"`
		Description  string `json:"description"`
		Color        string `json:"color"`
		IsDefault    bool   `json:"is_default"`
		PermissionIDs []uint `json:"permission_ids"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, apiError(err.Error()))
		return
	}

	r, err := getRoleService().UpdateRole(role.UpdateRoleRequest{
		RoleID:      uint(id),
		DisplayName: &input.DisplayName,
		Description: &input.Description,
		Color:       &input.Color,
		IsDefault:   &input.IsDefault,
		PermissionIDs: input.PermissionIDs,
	})
	if err != nil {
		if errors.Is(err, role.ErrRoleNotFound) {
			c.JSON(http.StatusNotFound, apiError("role not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"role":    r,
	})
}

// AdminRoleDelete - DELETE /api/v2/admin/role/:id
func AdminRoleDelete(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, apiError("invalid role ID"))
		return
	}

	if err := getRoleService().DeleteRole(role.DeleteRoleRequest{RoleID: uint(id)}); err != nil {
		if errors.Is(err, role.ErrRoleNotFound) {
			c.JSON(http.StatusNotFound, apiError("role not found"))
			return
		}
		if errors.Is(err, role.ErrCannotDeleteDefault) {
			c.JSON(http.StatusBadRequest, apiError("cannot delete default role"))
			return
		}
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Role deleted",
	})
}

// AdminPermissionList - GET /api/v2/admin/permissions
func AdminPermissionList(c *gin.Context) {
	resp, err := getRoleService().ListPermissions(role.ListPermissionsRequest{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": resp.Permissions,
	})
}

// AdminProfileAssignRole - POST /api/v2/admin/profile/:id/roles
func AdminProfileAssignRole(c *gin.Context) {
	profileIDParam := c.Param("id")
	profileID, err := strconv.ParseUint(profileIDParam, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, apiError("invalid profile ID"))
		return
	}

	var input struct {
		RoleID uint `json:"role_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, apiError(err.Error()))
		return
	}

	if err := getRoleService().AssignRole(role.AssignRoleRequest{
		ProfileID: uint(profileID),
		RoleID:    input.RoleID,
	}); err != nil {
		if errors.Is(err, role.ErrRoleNotFound) {
			c.JSON(http.StatusNotFound, apiError("role not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Role assigned successfully",
	})
}

// AdminProfileRemoveRole - DELETE /api/v2/admin/profile/:id/roles/:roleId
func AdminProfileRemoveRole(c *gin.Context) {
	profileIDParam := c.Param("id")
	profileID, err := strconv.ParseUint(profileIDParam, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, apiError("invalid profile ID"))
		return
	}

	roleIDParam := c.Param("roleId")
	roleID, err := strconv.ParseUint(roleIDParam, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, apiError("invalid role ID"))
		return
	}

	if err := getRoleService().RemoveRole(role.RemoveRoleRequest{
		ProfileID: uint(profileID),
		RoleID:    uint(roleID),
	}); err != nil {
		if errors.Is(err, role.ErrRoleNotFound) {
			c.JSON(http.StatusNotFound, apiError("role not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Role removed successfully",
	})
}
