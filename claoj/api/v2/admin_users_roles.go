package v2

import (
	"errors"
	"net/http"
	"net/mail"
	"strconv"
	"time"

	"github.com/CLAOJ/claoj/auth"
	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/CLAOJ/claoj/service/user"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ============================================================
// ADMIN USER MANAGEMENT API
// ============================================================

// adminUserGroupItem is the id/name pair the admin user APIs return for a
// Django group.
type adminUserGroupItem struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
}

// AdminUserList - GET /api/v2/admin/users
func AdminUserList(c *gin.Context) {
	page, pageSize := parsePagination(c)

	sortKey := c.DefaultQuery("sort", "date_joined")
	switch sortKey {
	case "username", "date_joined", "last_login", "points":
	default:
		c.JSON(http.StatusBadRequest, apiUserError("invalid sort: must be one of username, date_joined, last_login, points"))
		return
	}
	order := c.DefaultQuery("order", "desc")
	if order != "asc" && order != "desc" {
		c.JSON(http.StatusBadRequest, apiUserError("invalid order: must be asc or desc"))
		return
	}

	req := user.ListUsersRequest{
		Page:     page,
		PageSize: pageSize,
		Search:   c.Query("search"),
		Sort:     sortKey,
		Order:    order,
	}
	for _, f := range []struct {
		name string
		dst  **bool
	}{{"staff", &req.Staff}, {"superuser", &req.Superuser}, {"active", &req.Active}} {
		raw, ok := c.GetQuery(f.name)
		if !ok || raw == "" {
			continue
		}
		v, err := strconv.ParseBool(raw)
		if err != nil {
			c.JSON(http.StatusBadRequest, apiUserError("invalid "+f.name+": must be true or false"))
			return
		}
		*f.dst = &v
	}
	if raw := c.Query("group_id"); raw != "" {
		var groupID uint
		if err := parseUint(raw, &groupID); err != nil {
			c.JSON(http.StatusBadRequest, apiUserError("invalid group_id"))
			return
		}
		req.GroupID = groupID
	}

	resp, err := getUserService().ListUsers(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	type Item struct {
		ID             uint                 `json:"id"`
		Username       string               `json:"username"`
		Email          string               `json:"email"`
		Points         float64              `json:"points"`
		PerformancePts float64              `json:"performance_points"`
		ProblemCount   int                  `json:"problem_count"`
		Rating         *int                 `json:"rating"`
		IsStaff        bool                 `json:"is_staff"`
		IsSuperuser    bool                 `json:"is_super_user"`
		IsActive       bool                 `json:"is_active"`
		IsUnlisted     bool                 `json:"is_unlisted"`
		IsMuted        bool                 `json:"is_muted"`
		DateJoined     time.Time            `json:"date_joined"`
		LastAccess     time.Time            `json:"last_access"`
		LastLogin      *time.Time           `json:"last_login"`
		DisplayRank    string               `json:"display_rank"`
		BanReason      *string              `json:"ban_reason"`
		Groups         []adminUserGroupItem `json:"groups"`
	}
	items := make([]Item, len(resp.Users))
	for i, u := range resp.Users {
		groups := make([]adminUserGroupItem, len(u.Groups))
		for j, g := range u.Groups {
			groups[j] = adminUserGroupItem{ID: g.ID, Name: g.Name}
		}
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
			LastLogin:      u.LastLogin,
			DisplayRank:    u.DisplayRank,
			BanReason:      u.BanReason,
			Groups:         groups,
		}
	}
	c.JSON(http.StatusOK, apiListWithTotal(items, resp.Total))
}

// AdminUserDetail - GET /api/v2/admin/user/:id
//
// :id is a judge_profile.id (the convention of this handler family). The
// response deliberately carries no password hash, TOTP key, scratch codes,
// API token or other secret column.
func AdminUserDetail(c *gin.Context) {
	idParam := c.Param("id")
	userID, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, apiUserError("invalid user ID"))
		return
	}

	profile, err := getUserService().GetUser(user.GetUserRequest{
		UserID: uint(userID),
	})
	if err != nil {
		if errors.Is(err, user.ErrUserNotFound) {
			c.JSON(http.StatusNotFound, apiUserError("user not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	groups := make([]adminUserGroupItem, 0)
	var groupRows []struct {
		ID   uint
		Name string
	}
	if err := db.DB.Table("auth_user_groups").
		Select("auth_group.id AS id, auth_group.name AS name").
		Joins("JOIN auth_group ON auth_group.id = auth_user_groups.group_id").
		Where("auth_user_groups.user_id = ?", profile.UserID).
		Order("auth_group.name ASC").
		Scan(&groupRows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}
	groupIDs := make([]uint, len(groupRows))
	for i, g := range groupRows {
		groups = append(groups, adminUserGroupItem{ID: g.ID, Name: g.Name})
		groupIDs[i] = g.ID
	}

	type PermItem struct {
		ID       uint   `json:"id"`
		Codename string `json:"codename"`
	}
	permissions := make([]PermItem, 0)
	permissionIDs := make([]uint, 0)
	var permRows []struct {
		ID       uint
		Codename string
		AppLabel string
	}
	if err := db.DB.Table("auth_user_user_permissions").
		Select("auth_permission.id AS id, auth_permission.codename AS codename, django_content_type.app_label AS app_label").
		Joins("JOIN auth_permission ON auth_permission.id = auth_user_user_permissions.permission_id").
		Joins("JOIN django_content_type ON django_content_type.id = auth_permission.content_type_id").
		Where("auth_user_user_permissions.user_id = ?", profile.UserID).
		Order("django_content_type.app_label ASC, auth_permission.codename ASC").
		Scan(&permRows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}
	for _, p := range permRows {
		permissions = append(permissions, PermItem{ID: p.ID, Codename: p.AppLabel + "." + p.Codename})
		permissionIDs = append(permissionIDs, p.ID)
	}

	// judge_webauthncredential.user_id is a judge_profile.id FK.
	var credCount int64
	if err := db.DB.Model(&models.WebAuthnCredential{}).Where("user_id = ?", profile.ID).Count(&credCount).Error; err != nil {
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
		"id":                        profile.ID,
		"username":                  profile.Username,
		"email":                     profile.Email,
		"first_name":                profile.FirstName,
		"last_name":                 profile.LastName,
		"display_name":              profile.DisplayName,
		"about":                     profile.About,
		"timezone":                  profile.Timezone,
		"language_id":               profile.LanguageID,
		"ace_theme":                 profile.AceTheme,
		"site_theme":                profile.SiteTheme,
		"notes":                     profile.Notes,
		"user_script":               profile.UserScript,
		"username_display_override": profile.DisplayName,
		"points":                    profile.Points,
		"performance_points":        profile.PerformancePoints,
		"contribution_points":       profile.ContributionPoints,
		"rating":                    profile.Rating,
		"problem_count":             profile.ProblemCount,
		"display_rank":              profile.DisplayRank,
		"is_staff":                  profile.IsStaff,
		"is_super_user":             profile.IsSuperuser,
		"is_superuser":              profile.IsSuperuser,
		"is_active":                 profile.IsActive,
		"is_unlisted":               profile.IsUnlisted,
		"mute":                      profile.IsMuted,
		"is_muted":                  profile.IsMuted,
		"ban_reason":                profile.BanReason,
		"is_totp_enabled":           profile.IsTotpEnabled,
		"is_webauthn_enabled":       profile.IsWebauthnEnabled,
		"webauthn_credential_count": credCount,
		"groups":                    groups,
		"group_ids":                 groupIDs,
		"permissions":               permissions,
		"permission_ids":            permissionIDs,
		"organizations":             orgs,
		"last_access":               profile.LastAccess,
		"last_login":                profile.LastLogin,
		"date_joined":               profile.DateJoined,
	})
}

// AdminUserUpdate - PATCH /api/v2/admin/user/:id
func AdminUserUpdate(c *gin.Context) {
	idParam := c.Param("id")
	userID, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, apiUserError("invalid user ID"))
		return
	}

	var input struct {
		Username  *string `json:"username"`
		FirstName *string `json:"first_name"`
		LastName  *string `json:"last_name"`
		Email     *string `json:"email"`
		// DisplayName is the legacy name for UsernameDisplayOverride.
		DisplayName             *string `json:"display_name"`
		UsernameDisplayOverride *string `json:"username_display_override"`
		About                   *string `json:"about"`
		Notes                   *string `json:"notes"`
		UserScript              *string `json:"user_script"`
		Timezone                *string `json:"timezone"`
		LanguageID              *uint   `json:"language_id"`
		AceTheme                *string `json:"ace_theme"`
		SiteTheme               *string `json:"site_theme"`
		// Mute is the profile column; IsMuted is the legacy API name.
		Mute                    *bool   `json:"mute"`
		IsMuted                 *bool   `json:"is_muted"`
		IsActive                *bool   `json:"is_active"`
		IsUnlisted              *bool   `json:"is_unlisted"`
		DisplayRank             *string `json:"display_rank"`
		BanReason               *string `json:"ban_reason"`
		RemoveOrganizationIDs   []uint  `json:"remove_organization_ids,omitempty"`
		AddOrganizationIDs      []uint  `json:"add_organization_ids,omitempty"`
		RemoveOrganizationAdmin []uint  `json:"remove_organization_admin,omitempty"`
		AddOrganizationAdmin    []uint  `json:"add_organization_admin,omitempty"`
		IsStaff                 *bool   `json:"is_staff"`
		IsSuperuser             *bool   `json:"is_superuser"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, apiBindError(err))
		return
	}

	if input.UsernameDisplayOverride == nil {
		input.UsernameDisplayOverride = input.DisplayName
	}
	if input.Mute == nil {
		input.Mute = input.IsMuted
	}

	var target models.Profile
	if err := db.DB.Preload("User").First(&target, uint(userID)).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, apiUserError("user not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	// Only a *changed* username/email is validated: a legacy account may hold
	// values that predate these rules (a "." in the username, a display-name
	// email), and echoing them back unchanged must not block saving the rest of
	// the form.
	if input.Username != nil && *input.Username != target.User.Username {
		if !usernameRegex.MatchString(*input.Username) || len(*input.Username) > 30 {
			c.JSON(http.StatusBadRequest, apiUserError("invalid username: must be 1-30 chars, alphanumeric or underscore"))
			return
		}
	}
	if input.Email != nil && *input.Email != target.User.Email {
		addr, err := mail.ParseAddress(*input.Email)
		if err != nil || addr.Address != *input.Email {
			c.JSON(http.StatusBadRequest, apiUserError("invalid email address"))
			return
		}
	}

	// v1 parity: appointing or removing an organization administrator requires
	// authority over that organization (org admin, edit_all_organization, or
	// superuser). Checked per targeted org id.
	orgAdminTargets := append(append([]uint{}, input.AddOrganizationAdmin...), input.RemoveOrganizationAdmin...)
	for _, orgID := range orgAdminTargets {
		if !auth.CanEditOrganization(c, orgID) {
			c.JSON(http.StatusForbidden, apiUserError("you do not have permission to manage administrators of organization "+strconv.FormatUint(uint64(orgID), 10)))
			return
		}
	}

	// Self-demotion and self-deactivation lock the caller out of the admin
	// they are using, so they are refused rather than silently applied.
	callerID := c.GetUint("user_id")
	callerIsSuperuser := auth.GetAccess(c).IsSuperuser
	if target.UserID == callerID {
		if input.IsActive != nil && !*input.IsActive {
			c.JSON(http.StatusBadRequest, apiUserError("you cannot deactivate your own account"))
			return
		}
		if input.IsStaff != nil && !*input.IsStaff {
			c.JSON(http.StatusBadRequest, apiUserError("you cannot remove your own staff status"))
			return
		}
		if input.IsSuperuser != nil && !*input.IsSuperuser {
			c.JSON(http.StatusBadRequest, apiUserError("you cannot remove your own superuser status"))
			return
		}
	}

	// Editing another staff/superuser account is a takeover primitive
	// (rename, deactivate, email swap), so it is superuser-only. Any change to
	// is_staff/is_superuser/is_active on someone else is likewise
	// superuser-only, including deactivating a regular user.
	if target.UserID != callerID && !callerIsSuperuser {
		if target.User.IsStaff || target.User.IsSuperuser {
			c.JSON(http.StatusForbidden, apiUserError("superuser access required to edit a staff account"))
			return
		}
		if input.IsStaff != nil || input.IsSuperuser != nil || input.IsActive != nil {
			c.JSON(http.StatusForbidden, apiUserError("superuser access required to change account status"))
			return
		}
	}

	// Changing the email of a staff/superuser account is covered by the
	// takeover gate above; self-service email changes need no elevation.

	// is_staff / is_superuser changes are superuser-only; the write itself
	// lands in auth_user (a different primary key than judge_profile.id,
	// resolved above) inside the service transaction.
	if (input.IsStaff != nil || input.IsSuperuser != nil) && !callerIsSuperuser {
		c.JSON(http.StatusForbidden, apiUserError("superuser access required to change staff/superuser status"))
		return
	}

	if input.LanguageID != nil {
		var count int64
		if err := db.DB.Model(&models.Language{}).Where("id = ?", *input.LanguageID).Count(&count).Error; err != nil {
			c.JSON(http.StatusInternalServerError, apiError(err.Error()))
			return
		}
		if count == 0 {
			c.JSON(http.StatusBadRequest, apiUserError("language not found"))
			return
		}
	}

	if err := getUserService().UpdateUser(user.UpdateUserRequest{
		UserID:                  uint(userID),
		Username:                input.Username,
		FirstName:               input.FirstName,
		LastName:                input.LastName,
		Email:                   input.Email,
		DisplayName:             input.UsernameDisplayOverride,
		About:                   input.About,
		Notes:                   input.Notes,
		UserScript:              input.UserScript,
		Timezone:                input.Timezone,
		LanguageID:              input.LanguageID,
		AceTheme:                input.AceTheme,
		SiteTheme:               input.SiteTheme,
		IsActive:                input.IsActive,
		IsStaff:                 input.IsStaff,
		IsSuperuser:             input.IsSuperuser,
		IsUnlisted:              input.IsUnlisted,
		IsMuted:                 input.Mute,
		DisplayRank:             input.DisplayRank,
		BanReason:               input.BanReason,
		RemoveOrganizationIDs:   input.RemoveOrganizationIDs,
		AddOrganizationIDs:      input.AddOrganizationIDs,
		RemoveOrganizationAdmin: input.RemoveOrganizationAdmin,
		AddOrganizationAdmin:    input.AddOrganizationAdmin,
	}); err != nil {
		if errors.Is(err, user.ErrUserNotFound) {
			c.JSON(http.StatusNotFound, apiUserError("user not found"))
			return
		}
		if errors.Is(err, user.ErrUsernameTaken) || errors.Is(err, user.ErrEmailTaken) {
			c.JSON(http.StatusConflict, apiUserError(err.Error()))
			return
		}
		// Anything else is an unexpected write failure; never echo DB text.
		c.JSON(http.StatusInternalServerError, apiError("failed to update user: "+err.Error()))
		return
	}

	// Only a real status change warrants the cache bump and the session
	// revocation; the editor sends the current value on every save.
	activeChanged := input.IsActive != nil && *input.IsActive != target.User.IsActive
	if input.IsStaff != nil || input.IsSuperuser != nil || activeChanged {
		// Role and account-status changes must invalidate the cached
		// permission snapshot.
		auth.BumpPermVersion()
	}
	if activeChanged {
		// A deactivated account must not keep sessions opened before the
		// change (and a reactivated one should start clean).
		revokeUserSessions(target.UserID)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "User updated successfully",
	})
}

// authorizeAccountAction loads the judge_profile behind :id and applies the
// staff/superuser takeover rule shared by ban, unban and delete: a
// non-superuser may not act on a different staff or superuser account.
// Responses are written on failure.
func authorizeAccountAction(c *gin.Context, profileID uint) (models.Profile, bool) {
	var target models.Profile
	if err := db.DB.Preload("User").First(&target, profileID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, apiUserError("user not found"))
			return target, false
		}
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return target, false
	}
	if target.UserID != c.GetUint("user_id") && !auth.GetAccess(c).IsSuperuser &&
		(target.User.IsStaff || target.User.IsSuperuser) {
		c.JSON(http.StatusForbidden, apiUserError("superuser access required to manage a staff account"))
		return target, false
	}
	return target, true
}

// AdminUserDelete - DELETE /api/v2/admin/user/:id
func AdminUserDelete(c *gin.Context) {
	idParam := c.Param("id")
	userID, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, apiUserError("invalid user ID"))
		return
	}

	target, ok := authorizeAccountAction(c, uint(userID))
	if !ok {
		return
	}
	if target.UserID == c.GetUint("user_id") {
		c.JSON(http.StatusBadRequest, apiUserError("you cannot delete your own account"))
		return
	}

	if err := getUserService().DeleteUser(user.DeleteUserRequest{
		UserID: uint(userID),
	}); err != nil {
		if errors.Is(err, user.ErrUserNotFound) {
			c.JSON(http.StatusNotFound, apiUserError("user not found"))
			return
		}
		c.JSON(http.StatusBadRequest, apiClientError(err))
		return
	}

	// Soft deletion deactivates the account: end its sessions and drop the
	// cached permission snapshot immediately.
	auth.BumpPermVersion()
	revokeUserSessions(target.UserID)

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
		c.JSON(http.StatusBadRequest, apiUserError("invalid user ID"))
		return
	}

	var input struct {
		Reason string `json:"reason" binding:"required"`
		// Accepted for older clients and ignored: judge_profile has no
		// ban-expiry column, so every ban lasts until an unban.
		Day int `json:"day"`
	}

	target, ok := authorizeAccountAction(c, uint(userID))
	if !ok {
		return
	}
	if target.UserID == c.GetUint("user_id") {
		c.JSON(http.StatusBadRequest, apiUserError("you cannot ban your own account"))
		return
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, apiBindError(err))
		return
	}

	if err := getUserService().BanUser(user.BanUserRequest{
		UserID: uint(userID),
		Reason: input.Reason,
	}); err != nil {
		if errors.Is(err, user.ErrUserNotFound) {
			c.JSON(http.StatusNotFound, apiUserError("user not found"))
			return
		}
		c.JSON(http.StatusBadRequest, apiClientError(err))
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
		c.JSON(http.StatusBadRequest, apiUserError("invalid user ID"))
		return
	}

	if _, ok := authorizeAccountAction(c, uint(userID)); !ok {
		return
	}

	if err := getUserService().UnbanUser(user.UnbanUserRequest{
		UserID: uint(userID),
	}); err != nil {
		if errors.Is(err, user.ErrUserNotFound) {
			c.JSON(http.StatusNotFound, apiUserError("user not found"))
			return
		}
		c.JSON(http.StatusBadRequest, apiClientError(err))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "User unbanned successfully",
	})
}
