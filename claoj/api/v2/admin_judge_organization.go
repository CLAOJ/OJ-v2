package v2

import (
	"errors"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/CLAOJ/claoj/auth"
	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/CLAOJ/claoj/scoring"
	"github.com/CLAOJ/claoj/service/organization"
	"github.com/gin-gonic/gin"
)

// ============================================================
// ADMIN JUDGE MANAGEMENT API
// ============================================================

// AdminJudgeList - GET /api/v2/admin/judges
func AdminJudgeList(c *gin.Context) {
	page, pageSize := parsePagination(c)

	var judges []models.Judge

	if err := db.DB.Order("online DESC, name ASC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&judges).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	total, _ := countRecords(models.Judge{})

	type Item struct {
		ID         uint     `json:"id"`
		Name       string   `json:"name"`
		Tier       int      `json:"tier"`
		Online     bool     `json:"online"`
		IsBlocked  bool     `json:"is_blocked"`
		IsDisabled bool     `json:"is_disabled"`
		LastIP     string   `json:"last_ip"`
		Ping       *float64 `json:"ping"`
		Load       *float64 `json:"load"`
	}
	items := make([]Item, len(judges))
	for i, j := range judges {
		ip := ""
		if j.LastIP != nil {
			ip = *j.LastIP
		}
		items[i] = Item{
			ID:         j.ID,
			Name:       j.Name,
			Tier:       j.Tier,
			Online:     j.Online,
			IsBlocked:  j.IsBlocked,
			IsDisabled: j.IsDisabled,
			LastIP:     ip,
			Ping:       j.Ping,
			Load:       j.Load,
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"data":      items,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// AdminJudgeBlock - POST /api/v2/admin/judge/:id/block
func AdminJudgeBlock(c *gin.Context) {
	id, ok := numericParam(c, "id")
	if !ok {
		return
	}

	var judge models.Judge
	if err := db.DB.First(&judge, id).Error; err != nil {
		c.JSON(http.StatusNotFound, apiUserError("judge not found"))
		return
	}

	db.DB.Model(&judge).Update("is_blocked", true)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Judge blocked",
	})
}

// AdminJudgeUnblock - POST /api/v2/admin/judge/:id/unblock
func AdminJudgeUnblock(c *gin.Context) {
	id, ok := numericParam(c, "id")
	if !ok {
		return
	}

	var judge models.Judge
	if err := db.DB.First(&judge, id).Error; err != nil {
		c.JSON(http.StatusNotFound, apiUserError("judge not found"))
		return
	}

	db.DB.Model(&judge).Update("is_blocked", false)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Judge unblocked",
	})
}

// AdminJudgeDetail - GET /api/v2/admin/judge/:id
func AdminJudgeDetail(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, apiUserError("invalid judge ID"))
		return
	}

	var judge models.Judge
	if err := db.DB.Preload("Problems").Preload("Runtimes").First(&judge, id).Error; err != nil {
		c.JSON(http.StatusNotFound, apiUserError("judge not found"))
		return
	}

	// Get runtime versions for this judge
	var runtimeVersions []models.RuntimeVersion
	db.DB.Where("judge_id = ?", id).Preload("Language").Find(&runtimeVersions)

	type ProblemInfo struct {
		ID   uint   `json:"id"`
		Code string `json:"code"`
		Name string `json:"name"`
	}

	type RuntimeInfo struct {
		Key     string `json:"key"`
		Name    string `json:"name"`
		Version string `json:"version"`
	}

	type AssignedRuntime struct {
		ID   uint   `json:"id"`
		Key  string `json:"key"`
		Name string `json:"name"`
	}

	problems := make([]ProblemInfo, len(judge.Problems))
	problemIDs := make([]uint, len(judge.Problems))
	for i, p := range judge.Problems {
		problems[i] = ProblemInfo{ID: p.ID, Code: p.Code, Name: p.Name}
		problemIDs[i] = p.ID
	}

	// runtime_ids are the Language rows the judge is assigned (what the edit
	// form writes); runtimes below report the versions the agent announced.
	assigned := make([]AssignedRuntime, len(judge.Runtimes))
	runtimeIDs := make([]uint, len(judge.Runtimes))
	for i, r := range judge.Runtimes {
		assigned[i] = AssignedRuntime{ID: r.ID, Key: r.Key, Name: r.Name}
		runtimeIDs[i] = r.ID
	}

	runtimes := make([]RuntimeInfo, len(runtimeVersions))
	for i, r := range runtimeVersions {
		runtimes[i] = RuntimeInfo{
			Key:     r.Language.Key,
			Name:    r.Language.Name,
			Version: r.Version,
		}
	}

	lastIP := ""
	if judge.LastIP != nil {
		lastIP = *judge.LastIP
	}

	startTime := ""
	if judge.StartTime != nil {
		startTime = judge.StartTime.Format(time.RFC3339)
	}

	body := gin.H{
		"id":                judge.ID,
		"name":              judge.Name,
		"tier":              judge.Tier,
		"online":            judge.Online,
		"is_blocked":        judge.IsBlocked,
		"is_disabled":       judge.IsDisabled,
		"start_time":        startTime,
		"ping":              judge.Ping,
		"load":              judge.Load,
		"description":       judge.Description,
		"last_ip":           lastIP,
		"problems":          problems,
		"problem_ids":       problemIDs,
		"runtimes":          runtimes,
		"runtime_ids":       runtimeIDs,
		"assigned_runtimes": assigned,
	}
	// v1 shows the auth key only on the change form, which needs change_judge:
	// it is the judge's login, so it is not part of the plain read payload.
	if auth.HasPerm(c, "judge.change_judge") {
		body["auth_key"] = judge.AuthKey
	}
	c.JSON(http.StatusOK, body)
}

// AdminJudgeEnable - POST /api/v2/admin/judge/:id/enable
func AdminJudgeEnable(c *gin.Context) {
	id, ok := numericParam(c, "id")
	if !ok {
		return
	}

	var judge models.Judge
	if err := db.DB.First(&judge, id).Error; err != nil {
		c.JSON(http.StatusNotFound, apiUserError("judge not found"))
		return
	}

	db.DB.Model(&judge).Update("is_disabled", false)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Judge enabled",
	})
}

// AdminJudgeDisable - POST /api/v2/admin/judge/:id/disable
func AdminJudgeDisable(c *gin.Context) {
	id, ok := numericParam(c, "id")
	if !ok {
		return
	}

	var judge models.Judge
	if err := db.DB.First(&judge, id).Error; err != nil {
		c.JSON(http.StatusNotFound, apiUserError("judge not found"))
		return
	}

	db.DB.Model(&judge).Update("is_disabled", true)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Judge disabled",
	})
}

// AdminJudgeUpdate - PATCH /api/v2/admin/judge/:id
func AdminJudgeUpdate(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, apiUserError("invalid judge ID"))
		return
	}

	var judge models.Judge
	if err := db.DB.First(&judge, id).Error; err != nil {
		c.JSON(http.StatusNotFound, apiUserError("judge not found"))
		return
	}

	var input struct {
		Name        *string `json:"name"`
		AuthKey     *string `json:"auth_key"`
		Tier        *int    `json:"tier"`
		Description *string `json:"description"`
		ProblemIDs  []uint  `json:"problem_ids"`
		RuntimeIDs  []uint  `json:"runtime_ids"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, apiBindError(err))
		return
	}

	// name + auth_key are the judge's login; changing them lets the caller
	// impersonate the judge, so it needs the Django change permission.
	if (input.Name != nil || input.AuthKey != nil) && !auth.HasPerm(c, "judge.change_judge") {
		c.JSON(http.StatusForbidden, apiUserError("you do not have permission to rename or re-key a judge"))
		return
	}

	// An online judge keeps its old identity; renaming or re-keying it would
	// make every later event fail authentication.
	if (input.Name != nil || input.AuthKey != nil) && judge.Online {
		c.JSON(http.StatusConflict, apiUserError("judge is online; stop the judge agent before renaming or re-keying it"))
		return
	}

	updates := make(map[string]interface{})
	if input.Name != nil {
		name := strings.TrimSpace(*input.Name)
		if name == "" || utf8.RuneCountInString(name) > judgeNameMaxLen {
			c.JSON(http.StatusBadRequest, apiUserError("name is required and must be at most 50 characters"))
			return
		}
		taken, err := judgeNameTaken(name, judge.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, apiError(err.Error()))
			return
		}
		if taken {
			c.JSON(http.StatusConflict, apiUserError("a judge with this name already exists"))
			return
		}
		updates["name"] = name
	}
	if input.AuthKey != nil {
		if *input.AuthKey == "" || utf8.RuneCountInString(*input.AuthKey) > judgeAuthKeyMaxLen {
			c.JSON(http.StatusBadRequest, apiUserError("auth_key must be 1 to 100 characters"))
			return
		}
		updates["auth_key"] = *input.AuthKey
	}
	if input.Description != nil {
		updates["description"] = *input.Description
	}
	// v1's PositiveIntegerField: 0 and negatives are rejected; model.Tier is a
	// signed 32-bit int, so anything above that would wrap on insert.
	if input.Tier != nil {
		if *input.Tier < 1 || *input.Tier > math.MaxInt32 {
			c.JSON(http.StatusBadRequest, apiUserError("tier must be an integer from 1 to 2147483647"))
			return
		}
		updates["tier"] = *input.Tier
	}

	if len(updates) > 0 {
		if err := db.DB.Model(&judge).Updates(updates).Error; err != nil {
			c.JSON(http.StatusInternalServerError, apiError(err.Error()))
			return
		}
	}

	// Update problem assignments
	if input.ProblemIDs != nil {
		var problems []models.Problem
		db.DB.Where("id IN ?", input.ProblemIDs).Find(&problems)
		db.DB.Model(&judge).Association("Problems").Replace(&problems)
	}

	// Update runtime assignments
	if input.RuntimeIDs != nil {
		var runtimes []models.Language
		db.DB.Where("id IN ?", input.RuntimeIDs).Find(&runtimes)
		db.DB.Model(&judge).Association("Runtimes").Replace(&runtimes)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
	})
}

// ============================================================
// ADMIN ORGANIZATION MANAGEMENT API
// ============================================================

// AdminOrganizationList - GET /api/v2/admin/organizations
func AdminOrganizationList(c *gin.Context) {
	page, pageSize := parsePagination(c)

	resp, err := getOrgService().ListOrganizations(organization.ListOrganizationsRequest{
		Page:     page,
		PageSize: pageSize,
		Search:   strings.TrimSpace(c.Query("search")),
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	type Item struct {
		ID                uint    `json:"id"`
		Name              string  `json:"name"`
		Slug              string  `json:"slug"`
		ShortName         string  `json:"short_name"`
		IsOpen            bool    `json:"is_open"`
		IsUnlisted        bool    `json:"is_unlisted"`
		LogoOverrideImage string  `json:"logo_override_image"`
		MemberCount       int     `json:"member_count"`
		PerformancePoints float64 `json:"performance_points"`
	}
	items := make([]Item, len(resp.Organizations))
	for i, o := range resp.Organizations {
		items[i] = Item{
			ID:                o.ID,
			Name:              o.Name,
			Slug:              o.Slug,
			ShortName:         o.ShortName,
			IsOpen:            o.IsOpen,
			IsUnlisted:        o.IsUnlisted,
			LogoOverrideImage: o.LogoOverrideImage,
			MemberCount:       o.MemberCount,
			PerformancePoints: o.PerformancePoints,
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"data":      items,
		"total":     resp.Total,
		"page":      resp.Page,
		"page_size": resp.PageSize,
	})
}

// AdminOrganizationDetail - GET /api/v2/admin/organization/:id
func AdminOrganizationDetail(c *gin.Context) {
	idParam := c.Param("id")
	organizationID, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, apiUserError("invalid organization ID"))
		return
	}

	profile, err := getOrgService().GetOrganization(organization.GetOrganizationRequest{
		OrganizationID: uint(organizationID),
	})
	if err != nil {
		if errors.Is(err, organization.ErrOrganizationNotFound) {
			c.JSON(http.StatusNotFound, apiUserError("organization not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	c.JSON(http.StatusOK, organizationDetailBody(profile))
}

// organizationDetailBody is the flat admin organization payload shared by
// detail, create and update so all three return one shape.
func organizationDetailBody(profile *organization.OrganizationProfile) gin.H {
	return gin.H{
		"id":                  profile.ID,
		"name":                profile.Name,
		"slug":                profile.Slug,
		"short_name":          profile.ShortName,
		"about":               profile.About,
		"is_open":             profile.IsOpen,
		"is_unlisted":         profile.IsUnlisted,
		"slots":               profile.Slots,
		"access_code":         profile.AccessCode,
		"logo_override_image": profile.LogoOverrideImage,
		"creation_date":       profile.CreationDate.Format(time.RFC3339),
		"member_count":        profile.MemberCount,
		"performance_points":  profile.PerformancePoints,
		"admins":              profile.Admins,
	}
}

// AdminOrganizationCreate - POST /api/v2/admin/organizations
func AdminOrganizationCreate(c *gin.Context) {
	var input struct {
		Name              string  `json:"name" binding:"required"`
		Slug              string  `json:"slug" binding:"required"`
		ShortName         string  `json:"short_name" binding:"required"`
		About             string  `json:"about"`
		IsOpen            bool    `json:"is_open"`
		IsUnlisted        bool    `json:"is_unlisted"`
		Slots             *int    `json:"slots"`
		AccessCode        *string `json:"access_code"`
		LogoOverrideImage string  `json:"logo_override_image"`
		Admins            []uint  `json:"admins"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, apiBindError(err))
		return
	}

	// v1 keeps admins/is_open/slots readonly on the form without this perm.
	if (input.IsOpen || input.Slots != nil || len(input.Admins) > 0) &&
		!auth.HasPerm(c, "judge.organization_admin") {
		c.JSON(http.StatusForbidden, apiUserError("setting administrators, open status or slots requires the judge.organization_admin permission"))
		return
	}
	missing, err := missingProfileIDs(input.Admins)
	if err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}
	if len(missing) > 0 {
		c.JSON(http.StatusBadRequest, apiUserError(fmt.Sprintf("unknown profile IDs: %v", missing)))
		return
	}

	profile, err := getOrgService().CreateOrganization(organization.CreateOrganizationRequest{
		Name:              input.Name,
		Slug:              input.Slug,
		ShortName:         input.ShortName,
		About:             input.About,
		IsOpen:            input.IsOpen,
		IsUnlisted:        input.IsUnlisted,
		Slots:             input.Slots,
		AccessCode:        input.AccessCode,
		LogoOverrideImage: input.LogoOverrideImage,
		AdminIDs:          input.Admins,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	c.JSON(http.StatusOK, organizationDetailBody(profile))
}

// AdminOrganizationUpdate - PATCH /api/v2/admin/organization/:id
func AdminOrganizationUpdate(c *gin.Context) {
	idParam := c.Param("id")
	organizationID, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, apiUserError("invalid organization ID"))
		return
	}

	var input struct {
		Name              *string `json:"name"`
		Slug              *string `json:"slug"`
		ShortName         *string `json:"short_name"`
		About             *string `json:"about"`
		IsOpen            *bool   `json:"is_open"`
		IsUnlisted        *bool   `json:"is_unlisted"`
		Slots             *int    `json:"slots"`
		ClearSlots        bool    `json:"clear_slots"` // explicit null: *int cannot tell absent from null
		AccessCode        *string `json:"access_code"`
		LogoOverrideImage *string `json:"logo_override_image"`
		Admins            *[]uint `json:"admins"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, apiBindError(err))
		return
	}

	var org models.Organization
	if err := db.DB.Preload("Admins").First(&org, organizationID).Error; err != nil {
		c.JSON(http.StatusNotFound, apiUserError("organization not found"))
		return
	}

	// Unchanged (echoed) values pass; only an actual change is gated, matching
	// v1 where these fields are readonly without judge.organization_admin.
	restrictedChange := (input.IsOpen != nil && *input.IsOpen != org.IsOpen) ||
		(input.Slots != nil && (org.Slots == nil || *org.Slots != *input.Slots)) ||
		(input.ClearSlots && org.Slots != nil) ||
		(input.Admins != nil && !sameIDs(orgAdminIDs(org.Admins), *input.Admins))
	if restrictedChange && !auth.HasPerm(c, "judge.organization_admin") {
		c.JSON(http.StatusForbidden, apiUserError("changing administrators, open status or slots requires the judge.organization_admin permission"))
		return
	}
	if input.Admins != nil {
		missing, err := missingProfileIDs(*input.Admins)
		if err != nil {
			c.JSON(http.StatusInternalServerError, apiError(err.Error()))
			return
		}
		if len(missing) > 0 {
			c.JSON(http.StatusBadRequest, apiUserError(fmt.Sprintf("unknown profile IDs: %v", missing)))
			return
		}
	}

	profile, err := getOrgService().UpdateOrganization(organization.UpdateOrganizationRequest{
		OrganizationID:    uint(organizationID),
		Name:              input.Name,
		Slug:              input.Slug,
		ShortName:         input.ShortName,
		About:             input.About,
		IsOpen:            input.IsOpen,
		IsUnlisted:        input.IsUnlisted,
		Slots:             input.Slots,
		ClearSlots:        input.ClearSlots,
		AccessCode:        input.AccessCode,
		LogoOverrideImage: input.LogoOverrideImage,
		AdminIDs:          input.Admins,
	})
	if err != nil {
		if errors.Is(err, organization.ErrOrganizationNotFound) {
			c.JSON(http.StatusNotFound, apiUserError("organization not found"))
			return
		}
		c.JSON(http.StatusBadRequest, apiClientError(err))
		return
	}

	c.JSON(http.StatusOK, organizationDetailBody(profile))
}

// AdminOrganizationRecalculate - POST /api/v2/admin/organization/:id/recalculate
func AdminOrganizationRecalculate(c *gin.Context) {
	idParam := c.Param("id")
	organizationID, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, apiUserError("invalid organization ID"))
		return
	}

	var org models.Organization
	if err := db.DB.First(&org, organizationID).Error; err != nil {
		c.JSON(http.StatusNotFound, apiUserError("organization not found"))
		return
	}

	if err := scoring.CalculateOrganizationPoints(db.DB, uint(organizationID)); err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}
	if err := db.DB.Model(&org).Select("performance_points").First(&org, organizationID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{"performance_points": org.PerformancePoints})
}

// missingProfileIDs lists the requested profile ids that have no row.
func missingProfileIDs(ids []uint) ([]uint, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var found []uint
	if err := db.DB.Model(&models.Profile{}).Where("id IN ?", ids).Pluck("id", &found).Error; err != nil {
		return nil, err
	}
	seen := make(map[uint]bool, len(found))
	for _, id := range found {
		seen[id] = true
	}
	var missing []uint
	for _, id := range ids {
		if !seen[id] {
			missing = append(missing, id)
		}
	}
	return missing, nil
}

func orgAdminIDs(admins []models.Profile) []uint {
	ids := make([]uint, len(admins))
	for i, p := range admins {
		ids[i] = p.ID
	}
	return ids
}

// sameIDs compares two id lists as sets.
func sameIDs(a, b []uint) bool {
	as := make(map[uint]bool, len(a))
	for _, id := range a {
		as[id] = true
	}
	bs := make(map[uint]bool, len(b))
	for _, id := range b {
		bs[id] = true
	}
	if len(as) != len(bs) {
		return false
	}
	for id := range as {
		if !bs[id] {
			return false
		}
	}
	return true
}

// AdminOrganizationDelete - DELETE /api/v2/admin/organization/:id
func AdminOrganizationDelete(c *gin.Context) {
	idParam := c.Param("id")
	organizationID, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, apiUserError("invalid organization ID"))
		return
	}

	if err := getOrgService().DeleteOrganization(organization.DeleteOrganizationRequest{
		OrganizationID: uint(organizationID),
	}); err != nil {
		if errors.Is(err, organization.ErrOrganizationNotFound) {
			c.JSON(http.StatusNotFound, apiUserError("organization not found"))
			return
		}
		c.JSON(http.StatusBadRequest, apiClientError(err))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Organization hidden (soft deleted)",
	})
}
