package v2

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/CLAOJ/claoj/service/user"
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
