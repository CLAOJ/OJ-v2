package v2

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/CLAOJ/claoj/auth"
	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/CLAOJ/claoj/sanitization"
	"github.com/CLAOJ/claoj/scoring"
	"github.com/CLAOJ/claoj/service/organization"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// OrganizationList – GET /api/v2/organizations
func OrganizationList(c *gin.Context) {
	page, pageSize := parsePagination(c)
	search := c.Query("search")

	// Only listed organizations are shown (Django parity:
	// Organization.objects.filter(is_unlisted=False)).
	countQ := db.DB.Model(&models.Organization{}).Where("is_unlisted = ?", false)
	listQ := db.DB.Model(&models.Organization{}).Where("is_unlisted = ?", false)
	if search != "" {
		like := "%" + search + "%"
		countQ = countQ.Where("name LIKE ? OR short_name LIKE ?", like, like)
		listQ = listQ.Where("name LIKE ? OR short_name LIKE ?", like, like)
	}

	var total int64
	if err := countQ.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	var orgs []models.Organization
	if err := listQ.
		Order("name ASC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&orgs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	type Item struct {
		ID          uint    `json:"id"`
		Name        string  `json:"name"`
		Slug        string  `json:"slug"`
		ShortName   string  `json:"short_name"`
		About       string  `json:"about"`
		IsOpen      bool    `json:"is_open"`
		IsUnlisted  bool    `json:"is_unlisted"`
		MemberCount int     `json:"member_count"`
		PP          float64 `json:"performance_points"`
		Logo        string  `json:"logo_override_image"`
	}
	items := make([]Item, len(orgs))
	for i, o := range orgs {
		items[i] = Item{o.ID, o.Name, o.Slug, o.ShortName, o.About, o.IsOpen, o.IsUnlisted, o.MemberCount, o.PerformancePoints, o.LogoOverrideImage}
	}
	c.JSON(http.StatusOK, apiListWithTotal(items, total))
}

// OrganizationDetail – GET /api/v2/organization/:id
// ?sort=points|problem_count|rating|performance_points&order=asc|desc orders the
// member table (v1 OrganizationUsers: all four sorts default to descending).
func OrganizationDetail(c *gin.Context) {
	id, ok := numericParam(c, "id")
	if !ok {
		return
	}
	var org models.Organization
	if err := db.DB.First(&org, id).Error; err != nil {
		c.JSON(http.StatusNotFound, apiError("organization not found"))
		return
	}

	type Member struct {
		ID                uint    `json:"id"`
		Username          string  `json:"username"`
		DisplayName       string  `json:"display_name"`
		Role              string  `json:"role"`
		JoinedAt          string  `json:"joined_at"`
		Points            float64 `json:"points"`
		ProblemCount      int     `json:"problem_count"`
		Rating            *int    `json:"rating"`
		PerformancePoints float64 `json:"performance_points"`
	}

	// One joined query for the member table: the stats live on judge_profile,
	// so nothing else has to be loaded. Unknown sort columns fall back to the
	// v1 default (-performance_points).
	memberSorts := map[string]string{
		"points":             "judge_profile.points",
		"problem_count":      "judge_profile.problem_count",
		"rating":             "COALESCE(judge_profile.rating, 0)",
		"performance_points": "judge_profile.performance_points",
	}
	sortExpr, ok := memberSorts[c.DefaultQuery("sort", "performance_points")]
	if !ok {
		sortExpr = memberSorts["performance_points"]
	}
	orderDir := "DESC"
	if c.Query("order") == "asc" {
		orderDir = "ASC"
	}

	var memberRows []struct {
		models.Profile
		Username string
	}
	if err := db.DB.Model(&models.Profile{}).
		Select("judge_profile.*, au.username").
		Joins("JOIN judge_profile_organizations po ON po.profile_id = judge_profile.id").
		Joins("JOIN auth_user au ON au.id = judge_profile.user_id").
		Where("po.organization_id = ?", org.ID).
		Where("judge_profile.is_unlisted = ?", false).
		Order(sortExpr + " " + orderDir).
		Order("judge_profile.id ASC").
		Find(&memberRows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	var adminRows []struct {
		models.Profile
		Username string
	}
	if err := db.DB.Model(&models.Profile{}).
		Select("judge_profile.*, au.username").
		Joins("JOIN judge_organization_admins oa ON oa.profile_id = judge_profile.id").
		Joins("JOIN auth_user au ON au.id = judge_profile.user_id").
		Where("oa.organization_id = ?", org.ID).
		Where("judge_profile.is_unlisted = ?", false).
		Order("judge_profile.performance_points DESC").
		Order("judge_profile.id ASC").
		Find(&adminRows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	// Members who are also admins are labelled with the "admin" role.
	adminIDs := make(map[uint]bool, len(adminRows))
	for _, a := range adminRows {
		adminIDs[a.ID] = true
	}

	toMember := func(p models.Profile, username string) Member {
		role := "member"
		if adminIDs[p.ID] {
			role = "admin"
		}
		return Member{
			ID:          p.ID,
			Username:    username,
			DisplayName: p.UsernameDisplayOverride,
			Role:        role,
			// judge_profile_organizations has no join timestamp; left empty.
			JoinedAt:          "",
			Points:            p.Points,
			ProblemCount:      p.ProblemCount,
			Rating:            p.Rating,
			PerformancePoints: p.PerformancePoints,
		}
	}
	members := make([]Member, len(memberRows))
	for i, m := range memberRows {
		members[i] = toMember(m.Profile, m.Username)
	}
	admins := make([]Member, len(adminRows))
	for i, a := range adminRows {
		admins[i] = toMember(a.Profile, a.Username)
	}

	// The requesting user's own profile id, so the client can tell whether the
	// viewer is one of this organization's admins. 0 for anonymous requests.
	viewerProfileID := uint(0)
	if pid, ok := auth.CurrentProfileID(c); ok {
		viewerProfileID = pid
	}

	// Viewer-scoped state for the organization page. Membership cannot be read
	// off the member table: unlisted profiles are filtered out of it, and the
	// join/request UI must not offer to join an organization you are in.
	type viewerRequestInfo struct {
		ID     uint      `json:"id"`
		State  string    `json:"state"`
		Time   time.Time `json:"time"`
		Reason string    `json:"reason"`
	}
	viewerIsMember, viewerIsAdmin := false, false
	var viewerRequest *viewerRequestInfo
	if viewerProfileID > 0 {
		var count int64
		db.DB.Table("judge_profile_organizations").
			Where("profile_id = ? AND organization_id = ?", viewerProfileID, org.ID).
			Count(&count)
		viewerIsMember = count > 0
		viewerIsAdmin = isOrgAdmin(db.DB, org.ID, viewerProfileID)

		var req models.OrganizationRequest
		if err := db.DB.Where("user_id = ? AND organization_id = ?", viewerProfileID, org.ID).
			Order("time DESC, id DESC").First(&req).Error; err == nil {
			viewerRequest = &viewerRequestInfo{req.ID, req.State, req.Time, req.Reason}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"id":                  org.ID,
		"name":                org.Name,
		"slug":                org.Slug,
		"short_name":          org.ShortName,
		"about":               org.About,
		"is_open":             org.IsOpen,
		"is_unlisted":         org.IsUnlisted,
		"member_count":        org.MemberCount,
		"performance_points":  org.PerformancePoints,
		"logo_override_image": org.LogoOverrideImage,
		"creation_date":       org.CreationDate,
		"user_id":             viewerProfileID,
		"viewer_is_member":    viewerIsMember,
		"viewer_is_admin":     viewerIsAdmin,
		"viewer_request":      viewerRequest,
		"members":             members,
		"admins":              admins,
	})
}

// PublicOrganizationCreateRequest is the v1 CreateOrganization form: about,
// logo_override_image, admins (plus the organization's own name/slug, which
// the v2 form collects explicitly).
type PublicOrganizationCreateRequest struct {
	Name              string `json:"name" binding:"required"`
	Slug              string `json:"slug" binding:"required"`
	ShortName         string `json:"short_name"`
	About             string `json:"about"`
	LogoOverrideImage string `json:"logo_override_image"`
	Admins            []uint `json:"admins"`
}

// organizationAdminLimit mirrors v1 settings.CLAOJ_ORGANIZATION_ADMIN_LIMIT:
// how many organizations one profile may administer before creating more is
// refused. v1's spam_organization escape hatch can only ever resolve true for
// a superuser (no such Django permission exists), so superusers are exempt.
const organizationAdminLimit = 3

// PublicOrganizationCreate – POST /api/v2/organizations
// v1 CreateOrganization: judge.add_organization, the admin-of limit, and the
// caller becomes an administrator of the new organization.
func PublicOrganizationCreate(c *gin.Context) {
	_, profile, ok := resolveUserProfile(c)
	if !ok {
		return
	}
	if !auth.HasPerm(c, "judge.add_organization") {
		c.JSON(http.StatusForbidden, apiUserError("you are not allowed to create new organizations"))
		return
	}

	if !auth.GetAccess(c).IsSuperuser {
		var adminOf int64
		db.DB.Table("judge_organization_admins").Where("profile_id = ?", profile.ID).Count(&adminOf)
		if adminOf >= organizationAdminLimit {
			c.JSON(http.StatusForbidden, apiUserError("you are already an administrator of "+
				strconv.Itoa(int(adminOf))+" organizations; the limit is "+
				strconv.Itoa(organizationAdminLimit)))
			return
		}
	}

	var req PublicOrganizationCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, apiUserError(err.Error()))
		return
	}

	// v1 sets the short name to the slug's first 20 characters.
	shortName := req.ShortName
	if shortName == "" {
		shortName = req.Slug
	}
	if len(shortName) > 20 {
		shortName = shortName[:20]
	}

	adminIDs := []uint{profile.ID}
	for _, id := range req.Admins {
		if id != profile.ID {
			adminIDs = append(adminIDs, id)
		}
	}
	missing, err := missingProfileIDs(adminIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}
	if len(missing) > 0 {
		c.JSON(http.StatusBadRequest, apiUserError(fmt.Sprintf("unknown profile IDs: %v", missing)))
		return
	}

	created, err := getOrgService().CreateOrganization(organization.CreateOrganizationRequest{
		Name:      req.Name,
		Slug:      req.Slug,
		ShortName: shortName,
		About:     req.About,
		// v1 defaults a new organization to unlisted and closed.
		IsUnlisted:        true,
		LogoOverrideImage: req.LogoOverrideImage,
		AdminIDs:          adminIDs,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "organization created",
		"organization": gin.H{
			"id":   created.ID,
			"name": created.Name,
			"slug": created.Slug,
		},
	})
}

// UpdateOrganization – PATCH /api/v2/organization/:id
// Lets an organization's own administrators edit its settings (distinct from
// the site-admin endpoint at /admin/organization/:id).
func UpdateOrganization(c *gin.Context) {
	id, ok := numericParam(c, "id")
	if !ok {
		return
	}
	_, profile, ok := resolveUserProfile(c)
	if !ok {
		return
	}

	var org models.Organization
	if err := db.DB.First(&org, id).Error; err != nil {
		c.JSON(http.StatusNotFound, apiError("organization not found"))
		return
	}

	if !isOrgAdmin(db.DB, org.ID, profile.ID) {
		c.JSON(http.StatusForbidden, apiError("access denied: not an organization administrator"))
		return
	}

	var req struct {
		Name       *string `json:"name"`
		ShortName  *string `json:"short_name"`
		About      *string `json:"about"`
		IsOpen     *bool   `json:"is_open"`
		IsUnlisted *bool   `json:"is_unlisted"`
		Slots      *int    `json:"slots"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, apiError(err.Error()))
		return
	}

	updates := map[string]any{}
	if req.Name != nil {
		updates["name"] = sanitization.SanitizeTitle(*req.Name)
	}
	if req.ShortName != nil {
		updates["short_name"] = sanitization.SanitizeTitle(*req.ShortName)
	}
	if req.About != nil {
		updates["about"] = sanitization.SanitizeBlogContent(*req.About)
	}
	if req.IsOpen != nil {
		updates["is_open"] = *req.IsOpen
	}
	if req.IsUnlisted != nil {
		updates["is_unlisted"] = *req.IsUnlisted
	}
	if req.Slots != nil {
		// A zero (or negative) slot count means "unlimited" (NULL).
		if *req.Slots > 0 {
			updates["slots"] = *req.Slots
		} else {
			updates["slots"] = gorm.Expr("NULL")
		}
	}

	if len(updates) > 0 {
		if err := db.DB.Model(&org).Updates(updates).Error; err != nil {
			c.JSON(http.StatusInternalServerError, apiError(err.Error()))
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "organization updated"})
}

// JoinOrganization – POST /api/v2/organization/:id/join
func JoinOrganization(c *gin.Context) {
	id, ok := numericParam(c, "id")
	if !ok {
		return
	}
	_, profile, ok := resolveUserProfile(c)
	if !ok {
		return
	}

	err := db.DB.Transaction(func(tx *gorm.DB) error {
		var org models.Organization
		if err := tx.First(&org, id).Error; err != nil {
			return errors.New("organization not found")
		}

		if !org.IsOpen {
			return errors.New("this organization is not open; please request to join")
		}

		// Check if already in
		var count int64
		tx.Table("judge_profile_organizations").
			Where("profile_id = ? AND organization_id = ?", profile.ID, org.ID).
			Count(&count)
		if count > 0 {
			return errors.New("you are already a member of this organization")
		}

		// Add member
		if err := tx.Exec("INSERT INTO judge_profile_organizations (profile_id, organization_id) VALUES (?, ?)", profile.ID, org.ID).Error; err != nil {
			return err
		}

		// Update stats
		return onOrganizationMemberChange(tx, org.ID)
	})

	if err != nil {
		c.JSON(http.StatusBadRequest, apiError(err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "joined successfully"})
}

// LeaveOrganization – POST /api/v2/organization/:id/leave
func LeaveOrganization(c *gin.Context) {
	id, ok := numericParam(c, "id")
	if !ok {
		return
	}
	_, profile, ok := resolveUserProfile(c)
	if !ok {
		return
	}

	err := db.DB.Transaction(func(tx *gorm.DB) error {
		var org models.Organization
		if err := tx.First(&org, id).Error; err != nil {
			return errors.New("organization not found")
		}

		// Cannot leave if admin
		if isOrgAdmin(tx, org.ID, profile.ID) {
			return errors.New("admins cannot leave their organization; transfer ownership or delete it")
		}

		// Remove member
		res := tx.Exec("DELETE FROM judge_profile_organizations WHERE profile_id = ? AND organization_id = ?", profile.ID, org.ID)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return errors.New("you are not a member of this organization")
		}

		// Update stats
		return onOrganizationMemberChange(tx, org.ID)
	})

	if err != nil {
		c.JSON(http.StatusBadRequest, apiError(err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "left successfully"})
}

// RequestJoinOrganization – POST /api/v2/organization/:id/request
func RequestJoinOrganization(c *gin.Context) {
	id, ok := numericParam(c, "id")
	if !ok {
		return
	}
	_, profile, ok := resolveUserProfile(c)
	if !ok {
		return
	}

	var reqBody struct {
		Reason string `json:"reason" binding:"required"`
	}
	if err := c.ShouldBindJSON(&reqBody); err != nil {
		c.JSON(http.StatusBadRequest, apiError(err.Error()))
		return
	}

	err := db.DB.Transaction(func(tx *gorm.DB) error {
		var org models.Organization
		if err := tx.First(&org, id).Error; err != nil {
			return errors.New("organization not found")
		}

		if org.IsOpen {
			return errors.New("this organization is open; you can join directly")
		}

		// Check if already in
		var count int64
		tx.Table("judge_profile_organizations").
			Where("profile_id = ? AND organization_id = ?", profile.ID, org.ID).
			Count(&count)
		if count > 0 {
			return errors.New("you are already a member of this organization")
		}

		// Check for existing pending request
		tx.Model(&models.OrganizationRequest{}).
			Where("user_id = ? AND organization_id = ? AND state = 'P'", profile.ID, org.ID).
			Count(&count)
		if count > 0 {
			return errors.New("you already have a pending request to join this organization")
		}

		// Create request
		req := models.OrganizationRequest{
			UserID:         profile.ID,
			OrganizationID: org.ID,
			Time:           time.Now(),
			State:          "P",
			Reason:         reqBody.Reason,
		}
		return tx.Create(&req).Error
	})

	if err != nil {
		c.JSON(http.StatusBadRequest, apiError(err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "request sent successfully"})
}

// OrganizationRequestList – GET /api/v2/organization/:id/requests (Admin only)
// ?state=P (default, pending) | A | R | log (approved+rejected) | all mirrors
// v1 OrganizationRequestView / OrganizationRequestLog.
func OrganizationRequestList(c *gin.Context) {
	id, ok := numericParam(c, "id")
	if !ok {
		return
	}
	if _, ok := loadOrganization(c); !ok {
		return
	}
	_, profile, ok := resolveUserProfile(c)
	if !ok {
		return
	}

	if !isOrgAdmin(db.DB, uint(parseInt(id)), profile.ID) {
		c.JSON(http.StatusForbidden, apiError("access denied: not an organization administrator"))
		return
	}

	// Usernames are resolved with an explicit JOIN: request.user_id is a
	// judge_profile id and Preload("User.User") guesses has-one here (same
	// trap as judge_ticket/contest participation).
	q := db.DB.Table("judge_organizationrequest r").
		Select("r.id, r.user_id, au.username, r.time, r.reason, r.state").
		Joins("JOIN judge_profile p ON p.id = r.user_id").
		Joins("JOIN auth_user au ON au.id = p.user_id").
		Where("r.organization_id = ?", id)
	switch state := c.DefaultQuery("state", "P"); state {
	case "P", "A", "R":
		q = q.Where("r.state = ?", state)
	case "log":
		q = q.Where("r.state IN ?", []string{"A", "R"})
	case "all":
	default:
		c.JSON(http.StatusBadRequest, apiUserError("invalid state"))
		return
	}

	var items []struct {
		ID       uint      `json:"id"`
		UserID   uint      `json:"user_id"`
		Username string    `json:"username"`
		Time     time.Time `json:"time"`
		Reason   string    `json:"reason"`
		State    string    `json:"state"`
	}
	if err := q.Order("r.time DESC, r.id DESC").Scan(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}
	c.JSON(http.StatusOK, apiList(items))
}

// HandleOrganizationRequest – POST /api/v2/organization/request/:rid/handle
func HandleOrganizationRequest(c *gin.Context) {
	rid, ok := numericParam(c, "rid")
	if !ok {
		return
	}
	_, profile, ok := resolveUserProfile(c)
	if !ok {
		return
	}

	var reqBody struct {
		Action string `json:"action" binding:"required"` // "approve" or "reject"
	}
	if err := c.ShouldBindJSON(&reqBody); err != nil {
		c.JSON(http.StatusBadRequest, apiError(err.Error()))
		return
	}

	err := db.DB.Transaction(func(tx *gorm.DB) error {
		var req models.OrganizationRequest
		if err := tx.Preload("Organization").First(&req, rid).Error; err != nil {
			return errors.New("request not found")
		}

		if !isOrgAdmin(tx, req.OrganizationID, profile.ID) {
			return errors.New("access denied: not an organization administrator")
		}

		if req.State != "P" {
			return errors.New("request already handled")
		}

		if reqBody.Action == "approve" {
			req.State = "A"
			// Check slots
			if req.Organization.Slots != nil {
				if req.Organization.MemberCount >= *req.Organization.Slots {
					return errors.New("organization is full")
				}
			}

			// Add member
			if err := tx.Exec("INSERT INTO judge_profile_organizations (profile_id, organization_id) VALUES (?, ?)", req.UserID, req.OrganizationID).Error; err != nil {
				return err
			}
			if err := onOrganizationMemberChange(tx, req.OrganizationID); err != nil {
				return err
			}
		} else if reqBody.Action == "reject" {
			req.State = "R"
		} else {
			return errors.New("invalid action")
		}

		return tx.Save(&req).Error
	})

	if err != nil {
		c.JSON(http.StatusBadRequest, apiError(err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "request handled successfully"})
}

// KickUser – POST /api/v2/organization/:id/kick
func KickUser(c *gin.Context) {
	id, ok := numericParam(c, "id")
	if !ok {
		return
	}
	_, profile, ok := resolveUserProfile(c)
	if !ok {
		return
	}

	var reqBody struct {
		Username string `json:"username" binding:"required"`
	}
	if err := c.ShouldBindJSON(&reqBody); err != nil {
		c.JSON(http.StatusBadRequest, apiError(err.Error()))
		return
	}

	err := db.DB.Transaction(func(tx *gorm.DB) error {
		var org models.Organization
		if err := tx.First(&org, id).Error; err != nil {
			return errors.New("organization not found")
		}

		if !isOrgAdmin(tx, org.ID, profile.ID) {
			return errors.New("access denied: not an organization administrator")
		}

		var target models.Profile
		if err := tx.Table("judge_profile").
			Joins("JOIN auth_user ON auth_user.id = judge_profile.user_id").
			Where("auth_user.username = ?", reqBody.Username).
			First(&target).Error; err != nil {
			return errors.New("target user not found")
		}

		if isOrgAdmin(tx, org.ID, target.ID) {
			return errors.New("cannot kick an administrator")
		}

		res := tx.Exec("DELETE FROM judge_profile_organizations WHERE profile_id = ? AND organization_id = ?", target.ID, org.ID)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return errors.New("user is not a member of this organization")
		}

		return onOrganizationMemberChange(tx, org.ID)
	})

	if err != nil {
		c.JSON(http.StatusBadRequest, apiError(err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "user kicked successfully"})
}

// ============================================================
// ORGANIZATION PUBLIC TABS
//
// v1 gates all three lists behind CustomOrganizationMixin.dispatch: an
// organization member, or a holder of the view's bypass permissions. The row
// filters below then mirror ProblemListOrganization.get_filter,
// ContestListOrganization._get_queryset and SubmissionListOrganization.
// ============================================================

// loadOrganization resolves the :id path param, writing 400/404 itself.
func loadOrganization(c *gin.Context) (models.Organization, bool) {
	id, ok := numericParam(c, "id")
	if !ok {
		return models.Organization{}, false
	}
	var org models.Organization
	if err := db.DB.First(&org, id).Error; err != nil {
		c.JSON(http.StatusNotFound, apiError("organization not found"))
		return models.Organization{}, false
	}
	return org, true
}

// orgTabAccess reports whether the viewer may read one of the organization's
// private tabs: a member, a superuser, or a holder of one of the bypass
// permissions (v1 CustomOrganizationMixin.permission_bypass).
func orgTabAccess(c *gin.Context, orgID uint, bypass ...string) (uint, bool) {
	profileID, _ := auth.CurrentProfileID(c)
	if profileID > 0 {
		var count int64
		db.DB.Table("judge_profile_organizations").
			Where("profile_id = ? AND organization_id = ?", profileID, orgID).
			Count(&count)
		if count > 0 {
			return profileID, true
		}
	}
	access := auth.GetAccess(c)
	if access.IsSuperuser && access.IsActive {
		return profileID, true
	}
	for _, perm := range bypass {
		if access.HasPerm(perm) {
			return profileID, true
		}
	}
	return profileID, false
}

func orgForbidden(c *gin.Context) {
	c.JSON(http.StatusForbidden, apiUserError("you must join the organization to view its private data"))
}

// OrganizationProblemList – GET /api/v2/organization/:id/problems
// v1 ProblemListOrganization.get_filter: everything in the organization when
// the viewer holds judge.see_private_problem, otherwise the public problems plus
// the ones they author/curate/test.
func OrganizationProblemList(c *gin.Context) {
	org, ok := loadOrganization(c)
	if !ok {
		return
	}
	profileID, allowed := orgTabAccess(c, org.ID, "judge.see_organization_problem", "judge.edit_all_problem")
	if !allowed {
		orgForbidden(c)
		return
	}
	page, pageSize := parsePagination(c)

	// v1 lists the organization's problems newest first; ?sort= keeps the same
	// allow-list as the problem table (unknown columns fall back to -date).
	problemSorts := map[string]string{
		"code":       "judge_problem.code",
		"name":       "judge_problem.name",
		"points":     "judge_problem.points",
		"date":       "judge_problem.date",
		"user_count": "judge_problem.user_count",
	}
	sortExpr, valid := problemSorts[c.DefaultQuery("sort", "date")]
	if !valid {
		sortExpr = problemSorts["date"]
	}
	orderDir := "DESC"
	if c.Query("order") == "asc" {
		orderDir = "ASC"
	}

	visExpr, visArgs := auth.VisibleProblemFilter(c)
	q := db.DB.Model(&models.Problem{}).
		Where(visExpr, visArgs...).
		Where("judge_problem.id IN (SELECT problem_id FROM judge_problem_organizations WHERE organization_id = ?)", org.ID)

	// ?search= narrows by code/name. The org contest form's problem picker
	// reads the org's problems through this same member-gated list.
	if search := strings.TrimSpace(c.Query("search")); search != "" {
		like := "%" + search + "%"
		q = q.Where("(judge_problem.code LIKE ? OR judge_problem.name LIKE ?)", like, like)
	}

	var total int64
	if err := q.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	var problems []models.Problem
	if err := q.Session(&gorm.Session{}).
		Preload("Group").
		Select("judge_problem.id, judge_problem.code, judge_problem.name, judge_problem.points, judge_problem.partial, judge_problem.is_public, judge_problem.user_count, judge_problem.ac_rate, judge_problem.group_id, judge_problem.date").
		Order(sortExpr + " " + orderDir + ", judge_problem.id ASC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&problems).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	c.JSON(http.StatusOK, apiListWithTotal(problemListItems(problems, profileID), total))
}

// OrganizationContestList – GET /api/v2/organization/:id/contests
// v1 ContestListOrganization: visible contests of the organization (for
// non-privileged viewers), past ones only, newest first.
func OrganizationContestList(c *gin.Context) {
	org, ok := loadOrganization(c)
	if !ok {
		return
	}
	if _, allowed := orgTabAccess(c, org.ID, "judge.see_private_contest", "judge.edit_all_contest"); !allowed {
		orgForbidden(c)
		return
	}
	page, pageSize := parsePagination(c)

	visExpr, visArgs := auth.VisibleContestFilter(c)
	q := db.DB.Model(&models.Contest{}).
		Where(visExpr, visArgs...).
		Where("is_organization_private = ?", true).
		Where("id IN (SELECT contest_id FROM judge_contest_organizations WHERE organization_id = ?)", org.ID).
		Where("end_time < ?", time.Now())

	var total int64
	if err := q.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	var contests []models.Contest
	if err := q.Session(&gorm.Session{}).
		Select("judge_contest.id, judge_contest.key, judge_contest.name, judge_contest.start_time, judge_contest.end_time, judge_contest.is_rated, judge_contest.format_name, judge_contest.user_count").
		Order("start_time DESC, id DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&contests).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	type Item struct {
		Key       string    `json:"key"`
		Name      string    `json:"name"`
		StartTime time.Time `json:"start_time"`
		EndTime   time.Time `json:"end_time"`
		IsRated   bool      `json:"is_rated"`
		Format    string    `json:"format"`
		UserCount int       `json:"user_count"`
	}
	items := make([]Item, len(contests))
	for i, ct := range contests {
		items[i] = Item{ct.Key, ct.Name, ct.StartTime, ct.EndTime, ct.IsRated, ct.FormatName, ct.UserCount}
	}
	c.JSON(http.StatusOK, apiListWithTotal(items, total))
}

// orgSubmissionVisibilitySQL is v1 SubmissionsListBase._get_queryset's rule for
// viewers without judge.see_private_contest: their own submissions, submissions
// outside any contest, and submissions of contests whose scoreboard/submissions
// they may see. judge_contest_view_contest_submissions is the per-contest
// "can see contest submissions" list.
const orgSubmissionVisibilitySQL = "(judge_submission.user_id = ?" +
	" OR judge_submission.contest_object_id IS NULL" +
	" OR judge_submission.contest_object_id IN (SELECT c.id FROM judge_contest c WHERE" +
	" EXISTS (SELECT 1 FROM judge_contest_authors a WHERE a.contest_id = c.id AND a.profile_id = ?)" +
	" OR EXISTS (SELECT 1 FROM judge_contest_curators cu WHERE cu.contest_id = c.id AND cu.profile_id = ?)" +
	" OR (c.tester_see_submissions = ? AND EXISTS (SELECT 1 FROM judge_contest_testers te WHERE te.contest_id = c.id AND te.profile_id = ?))" +
	" OR EXISTS (SELECT 1 FROM judge_contest_view_contest_submissions vcs WHERE vcs.contest_id = c.id AND vcs.profile_id = ?)" +
	" OR c.scoreboard_visibility = ?" +
	" OR (c.end_time < ? AND c.scoreboard_visibility IN (?, ?))))"

// OrganizationSubmissionList – GET /api/v2/organization/:id/submissions
// v1 SubmissionListOrganization: submissions to this organization's problems,
// with the base contest visibility rules applied.
func OrganizationSubmissionList(c *gin.Context) {
	org, ok := loadOrganization(c)
	if !ok {
		return
	}
	profileID, allowed := orgTabAccess(c, org.ID, "judge.view_all_submission")
	if !allowed {
		orgForbidden(c)
		return
	}
	page, pageSize := parsePagination(c)

	q := db.DB.Model(&models.Submission{}).
		Joins("JOIN judge_profile jp ON jp.id = judge_submission.user_id").
		Joins("JOIN auth_user au ON au.id = jp.user_id").
		Joins("JOIN judge_problem pr ON pr.id = judge_submission.problem_id").
		Joins("JOIN judge_language l ON l.id = judge_submission.language_id").
		Where("pr.id IN (SELECT problem_id FROM judge_problem_organizations WHERE organization_id = ?)", org.ID)

	// Problem.get_visible_problems: a submission is only listed when its problem
	// is visible to the viewer (hidden org problems appear for their authors).
	visExpr, visArgs := auth.VisibleProblemFilterFor(c, "pr")
	q = q.Where(visExpr, visArgs...)

	access := auth.GetAccess(c)
	if !access.HasPerm("judge.see_private_contest") {
		q = q.Where(orgSubmissionVisibilitySQL,
			profileID, profileID, profileID, true, profileID, profileID,
			"V", time.Now(), "C", "P")
	}

	writeSubmissionList(c, q, page, pageSize, false, true)
}

// helpers

func isOrgAdmin(tx *gorm.DB, orgID, profileID uint) bool {
	var count int64
	tx.Table("judge_organization_admins").
		Where("organization_id = ? AND profile_id = ?", orgID, profileID).
		Count(&count)
	return count > 0
}

func onOrganizationMemberChange(tx *gorm.DB, orgID uint) error {
	var count int64
	if err := tx.Table("judge_profile_organizations").Where("organization_id = ?", orgID).Count(&count).Error; err != nil {
		return err
	}
	if err := tx.Model(&models.Organization{}).Where("id = ?", orgID).Update("member_count", int(count)).Error; err != nil {
		return err
	}
	return scoring.CalculateOrganizationPoints(tx, orgID)
}

func parseInt(s string) int {
	val, _ := strconv.Atoi(s)
	return val
}
