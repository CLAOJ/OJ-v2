package v2

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/CLAOJ/claoj/auth"
	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
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
		ID        uint   `json:"id"`
		Name      string `json:"name"`
		Online    bool   `json:"online"`
		IsBlocked bool   `json:"is_blocked"`
		AuthKey   string `json:"auth_key"`
		LastIP    string `json:"last_ip"`
	}
	items := make([]Item, len(judges))
	for i, j := range judges {
		ip := ""
		if j.LastIP != nil {
			ip = *j.LastIP
		}
		items[i] = Item{
			ID:        j.ID,
			Name:      j.Name,
			Online:    j.Online,
			IsBlocked: j.IsBlocked,
			AuthKey:   j.AuthKey,
			LastIP:    ip,
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
		c.JSON(http.StatusNotFound, apiError("judge not found"))
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
		c.JSON(http.StatusNotFound, apiError("judge not found"))
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
		c.JSON(http.StatusBadRequest, apiError("invalid judge ID"))
		return
	}

	var judge models.Judge
	if err := db.DB.Preload("Problems").Preload("Runtimes").First(&judge, id).Error; err != nil {
		c.JSON(http.StatusNotFound, apiError("judge not found"))
		return
	}

	// Get runtime versions for this judge
	var runtimeVersions []models.RuntimeVersion
	db.DB.Where("judge_id = ?", id).Preload("Language").Find(&runtimeVersions)

	type ProblemInfo struct {
		Code string `json:"code"`
		Name string `json:"name"`
	}

	type RuntimeInfo struct {
		Key     string `json:"key"`
		Name    string `json:"name"`
		Version string `json:"version"`
	}

	problems := make([]ProblemInfo, len(judge.Problems))
	for i, p := range judge.Problems {
		problems[i] = ProblemInfo{Code: p.Code, Name: p.Name}
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

	c.JSON(http.StatusOK, gin.H{
		"id":          judge.ID,
		"name":        judge.Name,
		"online":      judge.Online,
		"is_blocked":  judge.IsBlocked,
		"is_disabled": judge.IsDisabled,
		"start_time":  startTime,
		"ping":        judge.Ping,
		"load":        judge.Load,
		"description": judge.Description,
		"last_ip":     lastIP,
		"problems":    problems,
		"runtimes":    runtimes,
	})
}

// AdminJudgeEnable - POST /api/v2/admin/judge/:id/enable
func AdminJudgeEnable(c *gin.Context) {
	id, ok := numericParam(c, "id")
	if !ok {
		return
	}

	var judge models.Judge
	if err := db.DB.First(&judge, id).Error; err != nil {
		c.JSON(http.StatusNotFound, apiError("judge not found"))
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
		c.JSON(http.StatusNotFound, apiError("judge not found"))
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
		c.JSON(http.StatusBadRequest, apiError("invalid judge ID"))
		return
	}

	var judge models.Judge
	if err := db.DB.First(&judge, id).Error; err != nil {
		c.JSON(http.StatusNotFound, apiError("judge not found"))
		return
	}

	var input struct {
		Name        *string `json:"name"`
		AuthKey     *string `json:"auth_key"`
		Description *string `json:"description"`
		ProblemIDs  []uint  `json:"problem_ids"`
		RuntimeIDs  []uint  `json:"runtime_ids"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, apiError(err.Error()))
		return
	}

	// name + auth_key are the judge's login; changing them lets the caller
	// impersonate the judge, so it needs the Django change permission.
	if (input.Name != nil || input.AuthKey != nil) && !auth.HasPerm(c, "judge.change_judge") {
		c.JSON(http.StatusForbidden, apiUserError("you do not have permission to rename or re-key a judge"))
		return
	}

	updates := make(map[string]interface{})
	if input.Name != nil {
		name := strings.TrimSpace(*input.Name)
		if name == "" || len(name) > judgeNameMaxLen {
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
		if *input.AuthKey == "" || len(*input.AuthKey) > judgeAuthKeyMaxLen {
			c.JSON(http.StatusBadRequest, apiUserError("auth_key must be 1 to 100 characters"))
			return
		}
		updates["auth_key"] = *input.AuthKey
	}
	if input.Description != nil {
		updates["description"] = *input.Description
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
		ID          uint   `json:"id"`
		Name        string `json:"name"`
		Slug        string `json:"slug"`
		ShortName   string `json:"short_name"`
		IsOpen      bool   `json:"is_open"`
		IsUnlisted  bool   `json:"is_unlisted"`
		MemberCount int    `json:"member_count"`
	}
	items := make([]Item, len(resp.Organizations))
	for i, o := range resp.Organizations {
		items[i] = Item{
			ID:          o.ID,
			Name:        o.Name,
			Slug:        o.Slug,
			ShortName:   o.ShortName,
			IsOpen:      o.IsOpen,
			IsUnlisted:  o.IsUnlisted,
			MemberCount: o.MemberCount,
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"data":      items,
		"total":     resp.Total,
		"page":      resp.Page,
		"page_size": resp.PageSize,
	})
}

// AdminOrganizationCreate - POST /api/v2/admin/organizations
func AdminOrganizationCreate(c *gin.Context) {
	var input struct {
		Name        string  `json:"name" binding:"required"`
		Slug        string  `json:"slug" binding:"required"`
		ShortName   string  `json:"short_name" binding:"required"`
		About       string  `json:"about"`
		IsOpen      bool    `json:"is_open"`
		IsUnlisted  bool    `json:"is_unlisted"`
		Slots       *int    `json:"slots"`
		AccessCode  *string `json:"access_code"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, apiError(err.Error()))
		return
	}

	profile, err := getOrgService().CreateOrganization(organization.CreateOrganizationRequest{
		Name:       input.Name,
		Slug:       input.Slug,
		ShortName:  input.ShortName,
		About:      input.About,
		IsOpen:     input.IsOpen,
		IsUnlisted: input.IsUnlisted,
		Slots:      input.Slots,
		AccessCode: input.AccessCode,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":      true,
		"organization": profile,
	})
}

// AdminOrganizationUpdate - PATCH /api/v2/admin/organization/:id
func AdminOrganizationUpdate(c *gin.Context) {
	idParam := c.Param("id")
	organizationID, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, apiError("invalid organization ID"))
		return
	}

	var input struct {
		Name       *string `json:"name"`
		Slug       *string `json:"slug"`
		ShortName  *string `json:"short_name"`
		About      *string `json:"about"`
		IsOpen     *bool   `json:"is_open"`
		IsUnlisted *bool   `json:"is_unlisted"`
		Slots      *int    `json:"slots"`
		AccessCode *string `json:"access_code"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, apiError(err.Error()))
		return
	}

	profile, err := getOrgService().UpdateOrganization(organization.UpdateOrganizationRequest{
		OrganizationID: uint(organizationID),
		Name:           input.Name,
		Slug:           input.Slug,
		ShortName:      input.ShortName,
		About:          input.About,
		IsOpen:         input.IsOpen,
		IsUnlisted:     input.IsUnlisted,
		Slots:          input.Slots,
		AccessCode:     input.AccessCode,
	})
	if err != nil {
		if errors.Is(err, organization.ErrOrganizationNotFound) {
			c.JSON(http.StatusNotFound, apiError("organization not found"))
			return
		}
		c.JSON(http.StatusBadRequest, apiError(err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":      true,
		"organization": profile,
	})
}

// AdminOrganizationDelete - DELETE /api/v2/admin/organization/:id
func AdminOrganizationDelete(c *gin.Context) {
	idParam := c.Param("id")
	organizationID, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, apiError("invalid organization ID"))
		return
	}

	if err := getOrgService().DeleteOrganization(organization.DeleteOrganizationRequest{
		OrganizationID: uint(organizationID),
	}); err != nil {
		if errors.Is(err, organization.ErrOrganizationNotFound) {
			c.JSON(http.StatusNotFound, apiError("organization not found"))
			return
		}
		c.JSON(http.StatusBadRequest, apiError(err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Organization hidden (soft deleted)",
	})
}
