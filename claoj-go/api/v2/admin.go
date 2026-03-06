package v2

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/CLAOJ/claoj-go/db"
	"github.com/CLAOJ/claoj-go/jobs"
	"github.com/CLAOJ/claoj-go/models"
	"github.com/CLAOJ/claoj-go/sanitization"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// bridgeServerRef is set by main.go to allow API handlers to access bridge functions
var bridgeServerRef interface {
	Abort(subID uint) error
}

// SetBridgeServer sets the bridge server reference for API handlers
func SetBridgeServer(server interface {
	Abort(subID uint) error
}) {
	bridgeServerRef = server
}

// ============================================================
// ADMIN USER MANAGEMENT
// ============================================================

// AdminUserList - GET /api/v2/admin/users
func AdminUserList(c *gin.Context) {
	page, pageSize := parsePagination(c)

	var profiles []struct {
		models.Profile
		Username     string `gorm:"column:username"`
		Email        string `gorm:"column:email"`
		IsActive     bool   `gorm:"column:is_active"`
		IsStaff      bool   `gorm:"column:is_staff"`
		IsSuperuser  bool   `gorm:"column:is_superuser"`
		DateJoined   time.Time
	}

	if err := db.DB.Table("judge_profile").
		Joins("JOIN auth_user ON auth_user.id = judge_profile.user_id").
		Select("judge_profile.*, auth_user.username, auth_user.email, auth_user.is_active, auth_user.is_staff, auth_user.is_superuser, auth_user.date_joined").
		Order("auth_user.date_joined DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Scan(&profiles).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	type Item struct {
		ID             uint       `json:"id"`
		Username       string     `json:"username"`
		Email          string     `json:"email"`
		Points         float64    `json:"points"`
		PerformancePts float64    `json:"performance_points"`
		ProblemCount   int        `json:"problem_count"`
		Rating         *int       `json:"rating"`
		IsStaff        bool       `json:"is_staff"`
		IsSuperuser    bool       `json:"is_super_user"`
		IsActive       bool       `json:"is_active"`
		IsUnlisted     bool       `json:"is_unlisted"`
		IsMuted        bool       `json:"is_muted"`
		DateJoined     time.Time  `json:"date_joined"`
		LastAccess     time.Time  `json:"last_access"`
		DisplayRank    string     `json:"display_rank"`
		BanReason      *string    `json:"ban_reason"`
	}
	items := make([]Item, len(profiles))
	for i, p := range profiles {
		items[i] = Item{
			ID:             p.ID,
			Username:       p.Username,
			Email:          p.Email,
			Points:         p.Points,
			PerformancePts: p.PerformancePoints,
			ProblemCount:   p.ProblemCount,
			Rating:         p.Rating,
			IsStaff:        p.IsStaff,
			IsSuperuser:    p.IsSuperuser,
			IsActive:       p.IsActive,
			IsUnlisted:     p.IsUnlisted,
			IsMuted:        p.Mute,
			DateJoined:     p.DateJoined,
			LastAccess:     p.LastAccess,
			DisplayRank:    p.DisplayRank,
			BanReason:      p.BanReason,
		}
	}
	c.JSON(http.StatusOK, apiList(items))
}

// AdminUserDetail - GET /api/v2/admin/user/:id
func AdminUserDetail(c *gin.Context) {
	idParam := c.Param("id")
	var profile models.Profile
	if err := db.DB.Preload("User").First(&profile, idParam).Error; err != nil {
		c.JSON(http.StatusNotFound, apiError("user not found"))
		return
	}

	type OrgItem struct {
		ID   uint   `json:"id"`
		Name string `json:"name"`
	}
	orgs := make([]OrgItem, len(profile.Organizations))
	for i, o := range profile.Organizations {
		orgs[i] = OrgItem{o.ID, o.Name}
	}

	c.JSON(http.StatusOK, gin.H{
		"id":                    profile.ID,
		"username":              profile.User.Username,
		"email":                 profile.User.Email,
		"display_name":          profile.UsernameDisplayOverride,
		"about":                 profile.About,
		"points":                profile.Points,
		"performance_points":    profile.PerformancePoints,
		"contribution_points":   profile.ContributionPoints,
		"rating":                profile.Rating,
		"problem_count":         profile.ProblemCount,
		"display_rank":          profile.DisplayRank,
		"is_staff":              profile.User.IsStaff,
		"is_super_user":         profile.User.IsSuperuser,
		"is_active":             profile.User.IsActive,
		"is_unlisted":           profile.IsUnlisted,
		"is_muted":              profile.Mute,
		"is_totp_enabled":       profile.IsTotpEnabled,
		"is_webauthn_enabled":   profile.IsWebauthnEnabled,
		"organizations":         orgs,
		"last_access":           profile.LastAccess,
		"date_joined":           profile.User.DateJoined,
		"ban_reason":            profile.BanReason,
	})
}

// AdminUserUpdate - PATCH /api/v2/admin/user/:id
func AdminUserUpdate(c *gin.Context) {
	idParam := c.Param("id")
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

	var profile models.Profile
	if err := db.DB.Preload("User").First(&profile, idParam).Error; err != nil {
		c.JSON(http.StatusNotFound, apiError("user not found"))
		return
	}

	// Update profile fields
	updates := map[string]interface{}{}
	if input.DisplayName != nil {
		updates["username_display_override"] = *input.DisplayName
	}
	if input.About != nil {
		updates["about"] = *input.About
	}
	if input.IsUnlisted != nil {
		updates["is_unlisted"] = *input.IsUnlisted
	}
	if input.IsMuted != nil {
		updates["mute"] = *input.IsMuted
	}
	if input.DisplayRank != nil {
		updates["display_rank"] = *input.DisplayRank
	}
	if input.BanReason != nil {
		updates["ban_reason"] = *input.BanReason
	}
	if len(updates) > 0 {
		db.DB.Model(&profile).Updates(updates)
	}

	// Update user fields
	userUpdates := map[string]interface{}{}
	if input.IsActive != nil {
		userUpdates["is_active"] = *input.IsActive
	}
	if input.Email != nil {
		userUpdates["email"] = *input.Email
	}
	if len(userUpdates) > 0 {
		db.DB.Model(&profile.User).Updates(userUpdates)
	}

	// Handle organizations
	if len(input.RemoveOrganizationIDs) > 0 {
		db.DB.Model(&profile).Association("Organizations"). Delete(
			&models.Organization{}, "id IN ?", input.RemoveOrganizationIDs)
	}
	if len(input.AddOrganizationIDs) > 0 {
		db.DB.Model(&profile).Association("Organizations").Append(
			&models.Organization{}, "id IN ?", input.AddOrganizationIDs)
	}
	if len(input.RemoveOrganizationAdmin) > 0 {
		db.DB.Model(&profile).Association("Organizations").Delete(
			&models.Organization{}, "id IN ?", input.RemoveOrganizationAdmin)
		db.DB.Model(&profile).Association("Organizations").Append(
			&models.Organization{}, "id IN ?", input.RemoveOrganizationAdmin)
	}

	// Fetch updated profile
	db.DB.Preload("User").Preload("Organizations").First(&profile, profile.ID)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "User updated successfully",
		"user": gin.H{
			"id":        profile.ID,
			"username":  profile.User.Username,
			"email":     profile.User.Email,
			"is_active": profile.User.IsActive,
		},
	})
}

// AdminUserDelete - DELETE /api/v2/admin/user/:id
func AdminUserDelete(c *gin.Context) {
	idParam := c.Param("id")

	var profile models.Profile
	if err := db.DB.First(&profile, idParam).Error; err != nil {
		c.JSON(http.StatusNotFound, apiError("user not found"))
		return
	}

	// Soft delete: deactivate user instead of hard delete
	db.DB.Model(&profile.User).Update("is_active", false)
	db.DB.Model(&profile).Update("is_unlisted", true)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "User deactivated and hidden (soft deleted)",
	})
}

// AdminUserBan - POST /api/v2/admin/user/:id/ban
func AdminUserBan(c *gin.Context) {
	idParam := c.Param("id")
	var input struct {
		Reason string `json:"reason" binding:"required"`
		Day    int    `json:"day" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, apiError(err.Error()))
		return
	}

	var profile models.Profile
	if err := db.DB.First(&profile, idParam).Error; err != nil {
		c.JSON(http.StatusNotFound, apiError("user not found"))
		return
	}

	db.DB.Model(&profile).Updates(map[string]interface{}{
		"is_unlisted": true,
		"mute":        true,
		"ban_reason":  input.Reason,
	})

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "User banned successfully",
	})
}

// AdminUserUnban - POST /api/v2/admin/user/:id/unban
func AdminUserUnban(c *gin.Context) {
	idParam := c.Param("id")

	var profile models.Profile
	if err := db.DB.First(&profile, idParam).Error; err != nil {
		c.JSON(http.StatusNotFound, apiError("user not found"))
		return
	}

	db.DB.Model(&profile).Updates(map[string]interface{}{
		"is_unlisted": false,
		"mute":        false,
		"ban_reason":  nil,
	})

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "User unbanned successfully",
	})
}

// ============================================================
// ADMIN CONTEST MANAGEMENT
// ============================================================

// AdminContestList - GET /api/v2/admin/contests
func AdminContestList(c *gin.Context) {
	page, pageSize := parsePagination(c)

	var contests []models.Contest

	if err := db.DB.Order("start_time DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&contests).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	total, _ := countRecords(models.Contest{})

	type Item struct {
		ID               uint       `json:"id"`
		Key              string     `json:"key"`
		Name             string     `json:"name"`
		StartTime        time.Time  `json:"start_time"`
		EndTime          time.Time  `json:"end_time"`
		IsVisible        bool       `json:"is_visible"`
		IsRated          bool       `json:"is_rated"`
		UserCount        int        `json:"user_count"`
		FormatName       string     `json:"format_name"`
		IsOrganizationPrivate bool `json:"is_organization_private"`
	}
	items := make([]Item, len(contests))
	for i, c := range contests {
		items[i] = Item{
			ID:                    c.ID,
			Key:                   c.Key,
			Name:                  c.Name,
			StartTime:             c.StartTime,
			EndTime:               c.EndTime,
			IsVisible:             c.IsVisible,
			IsRated:               c.IsRated,
			UserCount:             c.UserCount,
			FormatName:            c.FormatName,
			IsOrganizationPrivate: c.IsOrganizationPrivate,
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"data":      items,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// AdminContestDetail - GET /api/v2/admin/contest/:key
func AdminContestDetail(c *gin.Context) {
	key := c.Param("key")
	var contest models.Contest
	if err := db.DB.Preload("Authors").
		Preload("Curators").
		Preload("Testers").
		Preload("Tags").
		Where("key = ?", key).First(&contest).Error; err != nil {
		c.JSON(http.StatusNotFound, apiError("contest not found"))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"contest": contest,
		"problems": func() []gin.H {
			var cps []models.ContestProblem
			db.DB.Where("contest_id = ?", contest.ID).Find(&cps)
			result := make([]gin.H, len(cps))
			for i, cp := range cps {
				result[i] = gin.H{
					"id":         cp.ID,
					"problem_id": cp.ProblemID,
					"code":       cp.Problem.Code,
					"name":       cp.Problem.Name,
					"points":     cp.Points,
					"order":      cp.Order,
				}
			}
			return result
		}(),
	})
}

// AdminContestCreate - POST /api/v2/admin/contests
func AdminContestCreate(c *gin.Context) {
	var input struct {
		Key              string  `json:"key" binding:"required"`
		Name             string  `json:"name" binding:"required"`
		Description      string  `json:"description" binding:"required"`
		Summary          string  `json:"summary"`
		StartTime        string  `json:"start_time" binding:"required"`
		EndTime          string  `json:"end_time" binding:"required"`
		TimeLimit        *int64  `json:"time_limit"`
		IsVisible        bool    `json:"is_visible"`
		IsRated          bool    `json:"is_rated"`
		FormatName       string  `json:"format_name"`
		FormatConfig     string  `json:"format_config"`
		AccessCode       string  `json:"access_code"`
		HideProblemTags  bool    `json:"hide_problem_tags"`
		RunPretestsOnly  bool    `json:"run_pretests_only"`
		IsOrganizationPrivate bool `json:"is_organization_private"`
		MaxSubmissions   *int    `json:"max_submissions"`
		AuthorIDs        []uint  `json:"author_ids"`
		CuratorIDs       []uint  `json:"curator_ids"`
		TesterIDs        []uint  `json:"tester_ids"`
		ProblemIDs       []uint  `json:"problem_ids"`
		TagIDs           []uint  `json:"tag_ids"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, apiError(err.Error()))
		return
	}

	// Parse timestamps
	startTime, err := parseRFC3339(input.StartTime)
	if err != nil {
		c.JSON(http.StatusBadRequest, apiError("invalid start_time format"))
		return
	}
	endTime, err := parseRFC3339(input.EndTime)
	if err != nil {
		c.JSON(http.StatusBadRequest, apiError("invalid end_time format"))
		return
	}

	contest := models.Contest{
		Key:              input.Key,
		Name:             sanitization.SanitizeTitle(input.Name),
		Description:      sanitization.SanitizeProblemContent(input.Description),
		Summary:          sanitization.SanitizeBlogSummary(input.Summary),
		StartTime:        startTime,
		EndTime:          endTime,
		TimeLimit:        input.TimeLimit,
		IsVisible:        input.IsVisible,
		IsRated:          input.IsRated,
		FormatName:       input.FormatName,
		AccessCode:       input.AccessCode,
		HideProblemTags:  input.HideProblemTags,
		RunPretestsOnly:  input.RunPretestsOnly,
		IsOrganizationPrivate: input.IsOrganizationPrivate,
	}

	if input.FormatConfig != "" {
		contest.FormatConfig = models.JSONField{}
		if err := contest.FormatConfig.Scan(input.FormatConfig); err != nil {
			c.JSON(http.StatusBadRequest, apiError("invalid format_config JSON"))
			return
		}
	}

	if err := db.DB.Create(&contest).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	// Handle many-to-many relations
	if len(input.AuthorIDs) > 0 {
		var authors []models.Profile
		db.DB.Where("id IN ?", input.AuthorIDs).Find(&authors)
		db.DB.Model(&contest).Association("Authors").Append(&authors)
	}
	if len(input.CuratorIDs) > 0 {
		var curators []models.Profile
		db.DB.Where("id IN ?", input.CuratorIDs).Find(&curators)
		db.DB.Model(&contest).Association("Curators").Append(&curators)
	}
	if len(input.TesterIDs) > 0 {
		var testers []models.Profile
		db.DB.Where("id IN ?", input.TesterIDs).Find(&testers)
		db.DB.Model(&contest).Association("Testers").Append(&testers)
	}
	if len(input.ProblemIDs) > 0 {
		var problems []models.Problem
		db.DB.Where("id IN ?", input.ProblemIDs).Find(&problems)

		for i, p := range problems {
			db.DB.Create(&models.ContestProblem{
				ContestID: contest.ID,
				ProblemID: p.ID,
				Points:    100,
				Partial:   true,
				Order:     uint(i + 1),
			})
		}
	}
	if len(input.TagIDs) > 0 {
		var tags []models.ContestTag
		db.DB.Where("id IN ?", input.TagIDs).Find(&tags)
		db.DB.Model(&contest).Association("Tags").Append(&tags)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"contest": gin.H{
			"id":  contest.ID,
			"key": contest.Key,
			"name": contest.Name,
		},
	})
}

// AdminContestUpdate - PATCH /api/v2/admin/contest/:key
func AdminContestUpdate(c *gin.Context) {
	key := c.Param("key")
	var contest models.Contest
	if err := db.DB.Where("key = ?", key).First(&contest).Error; err != nil {
		c.JSON(http.StatusNotFound, apiError("contest not found"))
		return
	}

	var input struct {
		Name                     *string  `json:"name"`
		Description              *string  `json:"description"`
		Summary                  *string  `json:"summary"`
		StartTime                *string  `json:"start_time"`
		EndTime                  *string  `json:"end_time"`
		IsVisible                *bool    `json:"is_visible"`
		IsRated                  *bool    `json:"is_rated"`
		AccessCode               *string  `json:"access_code"`
		HideProblemTags          *bool    `json:"hide_problem_tags"`
		RunPretestsOnly          *bool    `json:"run_pretests_only"`
		IsOrganizationPrivate    *bool    `json:"is_organization_private"`
		AddProblemIDs            []uint   `json:"add_problem_ids"`
		RemoveProblemIDs         []uint   `json:"remove_problem_ids"`
		UpdateTimeLimit          *int64   `json:"update_time_limit"`
		MaxSubmissions           *int     `json:"max_submissions"`
		AddTagIDs                []uint   `json:"add_tag_ids"`
		RemoveTagIDs             []uint   `json:"remove_tag_ids"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, apiError(err.Error()))
		return
	}

	updates := map[string]interface{}{}
	if input.Name != nil {
		updates["name"] = sanitization.SanitizeTitle(*input.Name)
	}
	if input.Description != nil {
		updates["description"] = sanitization.SanitizeProblemContent(*input.Description)
	}
	if input.Summary != nil {
		updates["summary"] = sanitization.SanitizeBlogSummary(*input.Summary)
	}
	if input.IsVisible != nil {
		updates["is_visible"] = *input.IsVisible
	}
	if input.IsRated != nil {
		updates["is_rated"] = *input.IsRated
	}
	if input.AccessCode != nil {
		updates["access_code"] = *input.AccessCode
	}
	if input.HideProblemTags != nil {
		updates["hide_problem_tags"] = *input.HideProblemTags
	}
	if input.RunPretestsOnly != nil {
		updates["run_pretests_only"] = *input.RunPretestsOnly
	}
	if input.IsOrganizationPrivate != nil {
		updates["is_organization_private"] = *input.IsOrganizationPrivate
	}
	if input.UpdateTimeLimit != nil {
		updates["time_limit"] = *input.UpdateTimeLimit
	}
	if input.MaxSubmissions != nil {
		updates["max_submissions"] = *input.MaxSubmissions
	}

	if len(updates) > 0 {
		db.DB.Model(&contest).Updates(updates)
	}

	// Add problems
	if len(input.AddProblemIDs) > 0 {
		var problems []models.Problem
		db.DB.Where("id IN ?", input.AddProblemIDs).Find(&problems)

		var existing []uint
		db.DB.Table("judge_contestproblem").Where("contest_id = ?", contest.ID).Pluck("problem_id", &existing)

		for i, p := range problems {
			if contains(existing, p.ID) {
				continue
			}
			db.DB.Create(&models.ContestProblem{
				ContestID: contest.ID,
				ProblemID: p.ID,
				Points:    100,
				Partial:   true,
				Order:     uint(len(existing) + i + 1),
			})
		}
	}

	// Remove problems
	if len(input.RemoveProblemIDs) > 0 {
		db.DB.Where("contest_id = ? AND problem_id IN ?", contest.ID, input.RemoveProblemIDs).Delete(&models.ContestProblem{})
	}

	// Add tags
	if len(input.AddTagIDs) > 0 {
		var tags []models.ContestTag
		db.DB.Where("id IN ?", input.AddTagIDs).Find(&tags)
		db.DB.Model(&contest).Association("Tags").Append(&tags)
	}

	// Remove tags
	if len(input.RemoveTagIDs) > 0 {
		var tags []models.ContestTag
		db.DB.Where("id IN ?", input.RemoveTagIDs).Find(&tags)
		db.DB.Model(&contest).Association("Tags").Delete(&tags)
	}

	db.DB.Preload("Authors").Preload("Curators").Preload("Testers").Preload("Tags").First(&contest, contest.ID)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"contest": contest,
	})
}

// AdminContestDelete - DELETE /api/v2/admin/contest/:key
func AdminContestDelete(c *gin.Context) {
	key := c.Param("key")
	var contest models.Contest
	if err := db.DB.Where("key = ?", key).First(&contest).Error; err != nil {
		c.JSON(http.StatusNotFound, apiError("contest not found"))
		return
	}

	// Soft delete - hide the contest
	db.DB.Model(&contest).Update("is_visible", false)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Contest hidden (soft deleted)",
	})
}

// AdminContestLock - POST /api/v2/admin/contest/:key/lock
func AdminContestLock(c *gin.Context) {
	key := c.Param("key")
	var contest models.Contest
	if err := db.DB.Where("key = ?", key).First(&contest).Error; err != nil {
		c.JSON(http.StatusNotFound, apiError("contest not found"))
		return
	}

	var input struct {
		LockedAfter *string `json:"locked_after"` // ISO 8601 format, or null to unlock
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, apiError(err.Error()))
		return
	}

	if input.LockedAfter == nil {
		// Unlock - set to NULL
		db.DB.Model(&contest).Update("locked_after", nil)
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "Contest unlocked - submissions are now allowed",
			"locked":  false,
		})
	} else {
		// Lock at specified time
		lockedAt, err := time.Parse(time.RFC3339, *input.LockedAfter)
		if err != nil {
			c.JSON(http.StatusBadRequest, apiError("invalid locked_after format, use ISO 8601"))
			return
		}
		db.DB.Model(&contest).Update("locked_after", lockedAt)
		c.JSON(http.StatusOK, gin.H{
			"success":      true,
			"message":      "Contest lock time updated",
			"locked":       true,
			"locked_after": lockedAt,
		})
	}
}


// AdminContestClone - POST /api/v2/admin/contest/:key/clone
func AdminContestClone(c *gin.Context) {
	key := c.Param("key")
	var sourceContest models.Contest
	if err := db.DB.Where("key = ?", key).First(&sourceContest).Error; err != nil {
		c.JSON(http.StatusNotFound, apiError("contest not found"))
		return
	}

	var input struct {
		NewKey        string `json:"new_key" binding:"required"`
		NewName       string `json:"new_name" binding:"required"`
		CopyProblems  bool   `json:"copy_problems"`
		CopySettings  bool   `json:"copy_settings"`
		NewStartTime  string `json:"new_start_time"`
		NewEndTime    string `json:"new_end_time"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, apiError(err.Error()))
		return
	}

	// Check if new key already exists
	var existing models.Contest
	if err := db.DB.Where("key = ?", input.NewKey).First(&existing).Error; err == nil {
		c.JSON(http.StatusBadRequest, apiError("contest key already exists"))
		return
	}

	// Parse optional new times
	var startTime, endTime time.Time
	if input.NewStartTime != "" {
		var err error
		startTime, err = parseRFC3339(input.NewStartTime)
		if err != nil {
			c.JSON(http.StatusBadRequest, apiError("invalid new_start_time format"))
			return
		}
		endTime, err = parseRFC3339(input.NewEndTime)
		if err != nil {
			c.JSON(http.StatusBadRequest, apiError("invalid new_end_time format"))
			return
		}
	} else {
		// Use original times
		startTime = sourceContest.StartTime
		endTime = sourceContest.EndTime
	}

	// Create new contest with copied settings
	newContest := models.Contest{
		Key:                   input.NewKey,
		Name:                  sanitization.SanitizeTitle(input.NewName),
		Description:           sourceContest.Description,
		Summary:               sourceContest.Summary,
		StartTime:             startTime,
		EndTime:               endTime,
		TimeLimit:             sourceContest.TimeLimit,
		IsVisible:             false, // Start as invisible
		IsRated:               sourceContest.IsRated,
		ScoreboardVisibility:  sourceContest.ScoreboardVisibility,
		ScoreboardCacheTimeout: sourceContest.ScoreboardCacheTimeout,
		UseClarifications:     sourceContest.UseClarifications,
		PushAnnouncements:     sourceContest.PushAnnouncements,
		RatingFloor:           sourceContest.RatingFloor,
		RatingCeiling:         sourceContest.RatingCeiling,
		RateAll:               sourceContest.RateAll,
		IsPrivate:             sourceContest.IsPrivate,
		HideProblemTags:       sourceContest.HideProblemTags,
		HideProblemAuthors:    sourceContest.HideProblemAuthors,
		RunPretestsOnly:       sourceContest.RunPretestsOnly,
		ShowShortDisplay:      sourceContest.ShowShortDisplay,
		IsOrganizationPrivate: sourceContest.IsOrganizationPrivate,
		OgImage:               sourceContest.OgImage,
		LogoOverrideImage:     sourceContest.LogoOverrideImage,
		AccessCode:            sourceContest.AccessCode,
		FormatName:            sourceContest.FormatName,
		FormatConfig:          sourceContest.FormatConfig,
		ProblemLabelScript:    sourceContest.ProblemLabelScript,
		LockedAfter:           nil, // Don't copy lock
		PointsPrecision:       sourceContest.PointsPrecision,
	}

	if err := db.DB.Create(&newContest).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	// Copy authors, curators, testers
	if input.CopySettings {
		var authors []models.Profile
		db.DB.Model(&sourceContest).Association("Authors").Find(&authors)
		if len(authors) > 0 {
			db.DB.Model(&newContest).Association("Authors").Append(&authors)
		}

		var curators []models.Profile
		db.DB.Model(&sourceContest).Association("Curators").Find(&curators)
		if len(curators) > 0 {
			db.DB.Model(&newContest).Association("Curators").Append(&curators)
		}

		var testers []models.Profile
		db.DB.Model(&sourceContest).Association("Testers").Find(&testers)
		if len(testers) > 0 {
			db.DB.Model(&newContest).Association("Testers").Append(&testers)
		}

		var orgs []models.Organization
		db.DB.Model(&sourceContest).Association("Organizations").Find(&orgs)
		if len(orgs) > 0 {
			db.DB.Model(&newContest).Association("Organizations").Append(&orgs)
		}
	}

	// Copy problems
	if input.CopyProblems {
		var contestProblems []models.ContestProblem
		db.DB.Where("contest_id = ?", sourceContest.ID).Order("order ASC").Find(&contestProblems)

		for _, cp := range contestProblems {
			newCP := models.ContestProblem{
				ContestID:            newContest.ID,
				ProblemID:            cp.ProblemID,
				Points:               cp.Points,
				Partial:              cp.Partial,
				IsPretested:          cp.IsPretested,
				Order:                cp.Order,
				OutputPrefixOverride: cp.OutputPrefixOverride,
				MaxSubmissions:       cp.MaxSubmissions,
			}
			db.DB.Create(&newCP)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "contest cloned successfully",
		"new_contest": gin.H{
			"id":  newContest.ID,
			"key": newContest.Key,
			"name": newContest.Name,
		},
	})
}


// ============================================================
// ADMIN PROBLEM MANAGEMENT
// ============================================================

// AdminProblemList - GET /api/v2/admin/problems
func AdminProblemList(c *gin.Context) {
	page, pageSize := parsePagination(c)

	var problems []struct {
		models.Problem
		GroupName string `gorm:"column:group_name"`
	}
	query := db.DB.Table("judge_problem").
		Joins("LEFT JOIN judge_problemgroup ON judge_problemgroup.id = judge_problem.group_id").
		Select("judge_problem.*, judge_problemgroup.name as group_name").
		Order("judge_problem.date DESC")

	total, _ := countRecords(models.Problem{})

	if err := query.Offset((page - 1) * pageSize).
		Limit(pageSize).
		Scan(&problems).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	type Item struct {
		ID          uint    `json:"id"`
		Code        string  `json:"code"`
		Name        string  `json:"name"`
		Points      float64 `json:"points"`
		Partial     bool    `json:"partial"`
		IsPublic    bool    `json:"is_public"`
		GroupName   string  `json:"group_name"`
		UserCount   int     `json:"user_count"`
		AcRate      float64 `json:"ac_rate"`
		IsManuallyManaged bool `json:"is_manually_managed"`
	}
	items := make([]Item, len(problems))
	for i, p := range problems {
		items[i] = Item{
			ID:                  p.ID,
			Code:                p.Code,
			Name:                p.Name,
			Points:              p.Points,
			Partial:             p.Partial,
			IsPublic:            p.IsPublic,
			GroupName:           p.GroupName,
			UserCount:           p.UserCount,
			AcRate:              p.AcRate,
			IsManuallyManaged:   p.IsManuallyManaged,
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"data":      items,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// AdminProblemDetail - GET /api/v2/admin/problem/:code
func AdminProblemDetail(c *gin.Context) {
	code := c.Param("code")
	var problem models.Problem
	if err := db.DB.Preload("Group").
		Preload("Types").
		Preload("Authors").
		Where("code = ?", code).First(&problem).Error; err != nil {
		c.JSON(http.StatusNotFound, apiError("problem not found"))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"problem": problem,
		"authors": func() []gin.H {
			result := make([]gin.H, len(problem.Authors))
			for i, a := range problem.Authors {
				result[i] = gin.H{
					"id":       a.ID,
					"username": a.User.Username,
				}
			}
			return result
		}(),
		"types": func() []gin.H {
			result := make([]gin.H, len(problem.Types))
			for i, t := range problem.Types {
				result[i] = gin.H{"id": t.ID, "name": t.FullName}
			}
			return result
		}(),
	})
}

// AdminProblemCreate - POST /api/v2/admin/problems
func AdminProblemCreate(c *gin.Context) {
	var input struct {
		Code              string  `json:"code" binding:"required"`
		Name              string  `json:"name" binding:"required"`
		Description       string  `json:"description" binding:"required"`
		Points            float64 `json:"points" binding:"required"`
		Partial           bool    `json:"partial"`
		IsPublic          bool    `json:"is_public"`
		TimeLimit         float64 `json:"time_limit" binding:"required"`
		MemoryLimit       uint    `json:"memory_limit" binding:"required"`
		GroupID           uint    `json:"group_id"`
		TypeIDs           []uint  `json:"type_ids"`
		AuthorIDs         []uint  `json:"author_ids"`
		AllowedLangIDs    []uint  `json:"allowed_lang_ids"`
		IsManuallyManaged bool    `json:"is_manually_managed"`
		PdfURL            string  `json:"pdf_url"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, apiError(err.Error()))
		return
	}

	problem := models.Problem{
		Code:             input.Code,
		Name:             sanitization.SanitizeTitle(input.Name),
		Description:      sanitization.SanitizeProblemContent(input.Description),
		Points:           input.Points,
		Partial:          input.Partial,
		IsPublic:         input.IsPublic,
		TimeLimit:        input.TimeLimit,
		MemoryLimit:      input.MemoryLimit,
		GroupID:          input.GroupID,
		IsManuallyManaged: input.IsManuallyManaged,
		PdfURL:           input.PdfURL,
	}

	if err := db.DB.Create(&problem).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	// Handle many-to-many relations
	if len(input.TypeIDs) > 0 {
		var types []models.ProblemType
		db.DB.Where("id IN ?", input.TypeIDs).Find(&types)
		db.DB.Model(&problem).Association("Types").Append(&types)
	}
	if len(input.AuthorIDs) > 0 {
		var authors []models.Profile
		db.DB.Where("id IN ?", input.AuthorIDs).Find(&authors)
		db.DB.Model(&problem).Association("Authors").Append(&authors)
	}
	if len(input.AllowedLangIDs) > 0 {
		var langs []models.Language
		db.DB.Where("id IN ?", input.AllowedLangIDs).Find(&langs)
		db.DB.Model(&problem).Association("AllowedLangs").Append(&langs)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"problem": gin.H{
			"id":   problem.ID,
			"code": problem.Code,
			"name": problem.Name,
		},
	})
}

// AdminProblemUpdate - PATCH /api/v2/admin/problem/:code
func AdminProblemUpdate(c *gin.Context) {
	code := c.Param("code")
	var problem models.Problem
	if err := db.DB.Where("code = ?", code).First(&problem).Error; err != nil {
		c.JSON(http.StatusNotFound, apiError("problem not found"))
		return
	}

	var input struct {
		Name              *string  `json:"name"`
		Description       *string  `json:"description"`
		Points            *float64 `json:"points"`
		Partial           *bool    `json:"partial"`
		IsPublic          *bool    `json:"is_public"`
		TimeLimit         *float64 `json:"time_limit"`
		MemoryLimit       *uint    `json:"memory_limit"`
		IsManuallyManaged *bool    `json:"is_manually_managed"`
		PdfURL            *string  `json:"pdf_url"`
		AddTypeIDs        []uint   `json:"add_type_ids"`
		RemoveTypeIDs     []uint   `json:"remove_type_ids"`
		AddAuthorIDs      []uint   `json:"add_author_ids"`
		RemoveAuthorIDs   []uint   `json:"remove_author_ids"`
		AddLangIDs        []uint   `json:"add_lang_ids"`
		RemoveLangIDs     []uint   `json:"remove_lang_ids"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, apiError(err.Error()))
		return
	}

	updates := map[string]interface{}{}
	if input.Name != nil {
		updates["name"] = sanitization.SanitizeTitle(*input.Name)
	}
	if input.Description != nil {
		updates["description"] = sanitization.SanitizeProblemContent(*input.Description)
	}
	if input.Points != nil {
		updates["points"] = *input.Points
	}
	if input.Partial != nil {
		updates["partial"] = *input.Partial
	}
	if input.IsPublic != nil {
		updates["is_public"] = *input.IsPublic
	}
	if input.TimeLimit != nil {
		updates["time_limit"] = *input.TimeLimit
	}
	if input.MemoryLimit != nil {
		updates["memory_limit"] = *input.MemoryLimit
	}
	if input.IsManuallyManaged != nil {
		updates["is_manually_managed"] = *input.IsManuallyManaged
	}
	if input.PdfURL != nil {
		updates["pdf_url"] = *input.PdfURL
	}

	if len(updates) > 0 {
		db.DB.Model(&problem).Updates(updates)
	}

	// Handle type relations
	if len(input.AddTypeIDs) > 0 {
		var types []models.ProblemType
		db.DB.Where("id IN ?", input.AddTypeIDs).Find(&types)
		db.DB.Model(&problem).Association("Types").Append(&types)
	}
	if len(input.RemoveTypeIDs) > 0 {
		var types []models.ProblemType
		db.DB.Where("id IN ?", input.RemoveTypeIDs).Find(&types)
		db.DB.Model(&problem).Association("Types").Delete(&types)
	}

	// Handle author relations
	if len(input.AddAuthorIDs) > 0 {
		var authors []models.Profile
		db.DB.Where("id IN ?", input.AddAuthorIDs).Find(&authors)
		db.DB.Model(&problem).Association("Authors").Append(&authors)
	}
	if len(input.RemoveAuthorIDs) > 0 {
		var authors []models.Profile
		db.DB.Where("id IN ?", input.RemoveAuthorIDs).Find(&authors)
		db.DB.Model(&problem).Association("Authors").Delete(&authors)
	}

	// Handle language relations
	if len(input.AddLangIDs) > 0 {
		var langs []models.Language
		db.DB.Where("id IN ?", input.AddLangIDs).Find(&langs)
		db.DB.Model(&problem).Association("AllowedLangs").Append(&langs)
	}
	if len(input.RemoveLangIDs) > 0 {
		var langs []models.Language
		db.DB.Where("id IN ?", input.RemoveLangIDs).Find(&langs)
		db.DB.Model(&problem).Association("AllowedLangs").Delete(&langs)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"problem": problem,
	})
}

// AdminProblemDelete - DELETE /api/v2/admin/problem/:code
func AdminProblemDelete(c *gin.Context) {
	code := c.Param("code")
	var problem models.Problem
	if err := db.DB.Where("code = ?", code).First(&problem).Error; err != nil {
		c.JSON(http.StatusNotFound, apiError("problem not found"))
		return
	}

	// Soft delete - hide the problem
	db.DB.Model(&problem).Update("is_public", false)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Problem hidden (soft deleted)",
	})
}

// ============================================================
// ADMIN JUDGE MANAGEMENT
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
		ID       uint   `json:"id"`
		Name     string `json:"name"`
		Online   bool   `json:"online"`
		IsBlocked bool  `json:"is_blocked"`
		AuthKey  string `json:"auth_key"`
		LastIP   string `json:"last_ip"`
	}
	items := make([]Item, len(judges))
	for i, j := range judges {
		ip := ""
		if j.LastIP != nil {
			ip = *j.LastIP
		}
		items[i] = Item{
			ID:       j.ID,
			Name:     j.Name,
			Online:   j.Online,
			IsBlocked: j.IsBlocked,
			AuthKey:  j.AuthKey,
			LastIP:   ip,
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
	id := c.Param("id")

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
	id := c.Param("id")

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
	id := c.Param("id")

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
	id := c.Param("id")

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
		Description *string `json:"description"`
		ProblemIDs  []uint  `json:"problem_ids"`
		RuntimeIDs  []uint  `json:"runtime_ids"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, apiError(err.Error()))
		return
	}

	updates := make(map[string]interface{})
	if input.Description != nil {
		updates["description"] = *input.Description
	}

	if len(updates) > 0 {
		db.DB.Model(&judge).Updates(updates)
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
// ADMIN ORGANIZATION MANAGEMENT
// ============================================================

// AdminOrganizationList - GET /api/v2/admin/organizations
func AdminOrganizationList(c *gin.Context) {
	page, pageSize := parsePagination(c)

	var organizations []models.Organization

	if err := db.DB.Order("name ASC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&organizations).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	total, _ := countRecords(models.Organization{})

	type Item struct {
		ID          uint   `json:"id"`
		Name        string `json:"name"`
		Slug        string `json:"slug"`
		ShortName   string `json:"short_name"`
		IsOpen      bool   `json:"is_open"`
		IsUnlisted  bool   `json:"is_unlisted"`
		MemberCount int    `json:"member_count"`
	}
	items := make([]Item, len(organizations))
	for i, o := range organizations {
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
		"total":     total,
		"page":      page,
		"page_size": pageSize,
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

	org := models.Organization{
		Name:        sanitization.SanitizeTitle(input.Name),
		Slug:        input.Slug,
		ShortName:   sanitization.SanitizeTitle(input.ShortName),
		About:       sanitization.SanitizeBlogContent(input.About),
		IsOpen:      input.IsOpen,
		IsUnlisted:  input.IsUnlisted,
		Slots:       input.Slots,
		AccessCode:  input.AccessCode,
	}

	if err := db.DB.Create(&org).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"organization": org,
	})
}

// AdminOrganizationUpdate - PATCH /api/v2/admin/organization/:id
func AdminOrganizationUpdate(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, apiError("invalid organization ID"))
		return
	}

	var org models.Organization
	if err := db.DB.First(&org, id).Error; err != nil {
		c.JSON(http.StatusNotFound, apiError("organization not found"))
		return
	}

	var input struct {
		Name        *string `json:"name"`
		Slug        *string `json:"slug"`
		ShortName   *string `json:"short_name"`
		About       *string `json:"about"`
		IsOpen      *bool   `json:"is_open"`
		IsUnlisted  *bool   `json:"is_unlisted"`
		Slots       *int    `json:"slots"`
		AccessCode  *string `json:"access_code"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, apiError(err.Error()))
		return
	}

	updates := map[string]interface{}{}
	if input.Name != nil {
		updates["name"] = sanitization.SanitizeTitle(*input.Name)
	}
	if input.Slug != nil {
		updates["slug"] = *input.Slug
	}
	if input.ShortName != nil {
		updates["short_name"] = sanitization.SanitizeTitle(*input.ShortName)
	}
	if input.About != nil {
		updates["about"] = sanitization.SanitizeBlogContent(*input.About)
	}
	if input.IsOpen != nil {
		updates["is_open"] = *input.IsOpen
	}
	if input.IsUnlisted != nil {
		updates["is_unlisted"] = *input.IsUnlisted
	}
	if input.Slots != nil {
		updates["slots"] = *input.Slots
	}
	if input.AccessCode != nil {
		updates["access_code"] = *input.AccessCode
	}

	if len(updates) > 0 {
		db.DB.Model(&org).Updates(updates)
	}

	c.JSON(http.StatusOK, gin.H{
		"success":      true,
		"organization": org,
	})
}

// AdminOrganizationDelete - DELETE /api/v2/admin/organization/:id
func AdminOrganizationDelete(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, apiError("invalid organization ID"))
		return
	}

	var org models.Organization
	if err := db.DB.First(&org, id).Error; err != nil {
		c.JSON(http.StatusNotFound, apiError("organization not found"))
		return
	}

	// Soft delete - mark as unlisted and not open
	db.DB.Model(&org).Updates(map[string]interface{}{
		"is_open":     false,
		"is_unlisted": true,
	})

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Organization hidden (soft deleted)",
	})
}

// ============================================================
// ADMIN SUBMISSION MANAGEMENT
// ============================================================

// AdminSubmissionList - GET /api/v2/admin/submissions
func AdminSubmissionList(c *gin.Context) {
	page, pageSize := parsePagination(c)

	var submissions []struct {
		models.Submission
		Username     string `gorm:"column:username"`
		ProblemCode  string `gorm:"column:problem_code"`
		LanguageName string `gorm:"column:language_name"`
	}

	if err := db.DB.Table("judge_submission").
		Joins("JOIN auth_user ON auth_user.id = judge_submission.user_id").
		Joins("JOIN judge_problem ON judge_problem.id = judge_submission.problem_id").
		Joins("JOIN judge_language ON judge_language.id = judge_submission.language_id").
		Select("judge_submission.*, auth_user.username, judge_problem.code as problem_code, judge_language.name as language_name").
		Order("judge_submission.date DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Scan(&submissions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	total, _ := countRecords(models.Submission{})

	type Item struct {
		ID           uint       `json:"id"`
		Username     string     `json:"user"`
		ProblemCode  string     `json:"problem"`
		LanguageName string     `json:"language"`
		Status       string     `json:"status"`
		Result       *string    `json:"result"`
		Score        *float64   `json:"score"`
		Time         *float64   `json:"time"`
		Memory       *float64   `json:"memory"`
		Date         time.Time  `json:"date"`
		IsPretested  bool       `json:"is_pretested"`
	}
	items := make([]Item, len(submissions))
	for i, s := range submissions {
		items[i] = Item{
			ID:           s.ID,
			Username:     s.Username,
			ProblemCode:  s.ProblemCode,
			LanguageName: s.LanguageName,
			Status:       s.Status,
			Result:       s.Result,
			Score:        s.Points,
			Time:         s.Time,
			Memory:       s.Memory,
			Date:         s.Date,
			IsPretested:  s.IsPretested,
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"data":      items,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// AdminSubmissionRejudge - POST /api/v2/admin/submission/:id/rejudge
func AdminSubmissionRejudge(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, apiError("invalid submission ID"))
		return
	}

	var sub models.Submission
	if err := db.DB.Preload("Problem").First(&sub, id).Error; err != nil {
		c.JSON(http.StatusNotFound, apiError("submission not found"))
		return
	}

	// Check if submission is locked
	if sub.LockedAfter != nil {
		c.JSON(http.StatusBadRequest, apiError("submission is locked and cannot be rejudged"))
		return
	}

	// Reset submission state for rejudge
	db.DB.Model(&sub).Updates(map[string]interface{}{
		"status":           "QU",
		"result":           nil,
		"points":           nil,
		"time":             nil,
		"memory":           nil,
		"current_testcase": 0,
		"case_points":      0,
		"case_total":       0,
	})

	// Clear test cases
	db.DB.Where("submission_id = ?", id).Delete(&models.SubmissionTestCase{})

	// Enqueue for rejudging
	if err := jobs.EnqueueJudgeSubmission(uint(id)); err != nil {
		// Return success anyway since queuing failed is not critical
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "Submission reset for rejudge (queue not available)",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Submission queued for rejudge",
	})
}

// AdminSubmissionAbort - POST /api/v2/admin/submission/:id/abort
func AdminSubmissionAbort(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, apiError("invalid submission ID"))
		return
	}

	var sub models.Submission
	if err := db.DB.First(&sub, id).Error; err != nil {
		c.JSON(http.StatusNotFound, apiError("submission not found"))
		return
	}

	// Only allow aborting submissions that are being processed
	if sub.Status != "P" && sub.Status != "G" {
		c.JSON(http.StatusBadRequest, apiError("submission is not being processed"))
		return
	}

	// Get the bridge server from global state
	if bridgeServerRef == nil {
		c.JSON(http.StatusInternalServerError, apiError("bridge server not available"))
		return
	}

	// Send abort command to judge
	if err := bridgeServerRef.Abort(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, apiError(fmt.Sprintf("failed to abort: %v", err)))
		return
	}

	// Update submission status to aborted
	db.DB.Model(&sub).Updates(map[string]interface{}{
		"status": "AB",
		"result": "AB",
	})

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Submission abort signal sent",
	})
}

// AdminSubmissionBatchRejudge - POST /api/v2/admin/submissions/batch-rejudge
func AdminSubmissionBatchRejudge(c *gin.Context) {
	var req struct {
		SubmissionIDs []uint `json:"submission_ids"`
		Filters       *struct {
			UserID      *uint   `json:"user_id"`
			Username    string  `json:"username"`
			ProblemID   *uint   `json:"problem_id"`
			ProblemCode string  `json:"problem_code"`
			LanguageID  *uint   `json:"language_id"`
			Language    string  `json:"language"`
			Status      string  `json:"status"`
			Result      string  `json:"result"`
			FromDate    string  `json:"from_date"`
			ToDate      string  `json:"to_date"`
		} `json:"filters"`
		DryRun bool `json:"dry_run"`
	}

	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, apiError("invalid request body"))
		return
	}

	// Build query for matching submissions
	query := db.DB.Model(&models.Submission{})

	// Apply filters if provided
	if req.Filters != nil {
		if req.Filters.UserID != nil && *req.Filters.UserID > 0 {
			query = query.Where("user_id = ?", *req.Filters.UserID)
		}
		if req.Filters.Username != "" {
			var profile models.Profile
			if err := db.DB.Joins("JOIN auth_user ON auth_user.id = judge_profile.user_id").
				Where("auth_user.username = ?", req.Filters.Username).
				First(&profile).Error; err == nil {
				query = query.Where("user_id = ?", profile.UserID)
			}
		}
		if req.Filters.ProblemID != nil && *req.Filters.ProblemID > 0 {
			query = query.Where("problem_id = ?", *req.Filters.ProblemID)
		}
		if req.Filters.ProblemCode != "" {
			var problem models.Problem
			if err := db.DB.Where("code = ?", req.Filters.ProblemCode).First(&problem).Error; err == nil {
				query = query.Where("problem_id = ?", problem.ID)
			}
		}
		if req.Filters.LanguageID != nil && *req.Filters.LanguageID > 0 {
			query = query.Where("language_id = ?", *req.Filters.LanguageID)
		}
		if req.Filters.Language != "" {
			var lang models.Language
			if err := db.DB.Where("name LIKE ?", "%"+req.Filters.Language+"%").First(&lang).Error; err == nil {
				query = query.Where("language_id = ?", lang.ID)
			}
		}
		if req.Filters.Status != "" {
			query = query.Where("status = ?", req.Filters.Status)
		}
		if req.Filters.Result != "" {
			query = query.Where("result = ?", req.Filters.Result)
		}
		if req.Filters.FromDate != "" {
			query = query.Where("date >= ?", req.Filters.FromDate)
		}
		if req.Filters.ToDate != "" {
			query = query.Where("date <= ?", req.Filters.ToDate)
		}
	}

	// If specific submission IDs provided, filter by them
	if len(req.SubmissionIDs) > 0 {
		query = query.Where("id IN ?", req.SubmissionIDs)
	}

	// Count matching submissions
	var count int64
	if err := query.Count(&count).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	// If dry run, just return the count
	if req.DryRun {
		c.JSON(http.StatusOK, gin.H{
			"count": count,
			"message": fmt.Sprintf("%d submissions would be rejudged", count),
		})
		return
	}

	// Get all matching submission IDs
	var submissionIDs []uint
	if err := query.Pluck("id", &submissionIDs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	// Reset all matching submissions
	for _, subID := range submissionIDs {
		var sub models.Submission
		if err := db.DB.First(&sub, subID).Error; err != nil {
			continue // Skip if not found
		}

		// Skip locked submissions
		if sub.LockedAfter != nil {
			continue
		}

		// Reset submission state
		db.DB.Model(&sub).Updates(map[string]interface{}{
			"status":           "QU",
			"result":           nil,
			"points":           nil,
			"time":             nil,
			"memory":           nil,
			"current_testcase": 0,
			"case_points":      0,
			"case_total":       0,
		})

		// Clear test cases
		db.DB.Where("submission_id = ?", subID).Delete(&models.SubmissionTestCase{})

		// Enqueue for rejudging
		jobs.EnqueueJudgeSubmission(subID)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"count":   len(submissionIDs),
		"message": fmt.Sprintf("%d submissions queued for rejudge", len(submissionIDs)),
	})
}

// ============================================================
// ADMIN ROLES & PERMISSIONS MANAGEMENT
// ============================================================

// AdminRoleList - GET /api/v2/admin/roles
func AdminRoleList(c *gin.Context) {
	var roles []models.Role
	if err := db.DB.Preload("Permissions").Order("name ASC").Find(&roles).Error; err != nil {
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

	items := make([]RoleItem, len(roles))
	for i, r := range roles {
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

	var role models.Role
	if err := db.DB.Preload("Permissions").First(&role, id).Error; err != nil {
		c.JSON(http.StatusNotFound, apiError("role not found"))
		return
	}

	c.JSON(http.StatusOK, role)
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

	role := models.Role{
		Name:        input.Name,
		DisplayName: input.DisplayName,
		Description: input.Description,
		Color:       input.Color,
		IsDefault:   input.IsDefault,
	}

	if err := db.DB.Create(&role).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	// Associate permissions
	if len(input.PermissionIDs) > 0 {
		var permissions []models.Permission
		if err := db.DB.Where("id IN ?", input.PermissionIDs).Find(&permissions).Error; err == nil {
			db.DB.Model(&role).Association("Permissions").Append(&permissions)
		}
	}

	// Reload role with permissions
	db.DB.Preload("Permissions").First(&role, role.ID)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"role":    role,
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

	var role models.Role
	if err := db.DB.First(&role, id).Error; err != nil {
		c.JSON(http.StatusNotFound, apiError("role not found"))
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

	updates := map[string]interface{}{}
	if input.DisplayName != "" {
		updates["display_name"] = input.DisplayName
	}
	if input.Description != "" {
		updates["description"] = input.Description
	}
	if input.Color != "" {
		updates["color"] = input.Color
	}
	updates["is_default"] = input.IsDefault

	if len(updates) > 0 {
		db.DB.Model(&role).Updates(updates)
	}

	// Update permissions
	if input.PermissionIDs != nil {
		var permissions []models.Permission
		if err := db.DB.Where("id IN ?", input.PermissionIDs).Find(&permissions).Error; err == nil {
			db.DB.Model(&role).Association("Permissions").Clear()
			db.DB.Model(&role).Association("Permissions").Append(&permissions)
		}
	}

	// Reload role with permissions
	db.DB.Preload("Permissions").First(&role, role.ID)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"role":    role,
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

	var role models.Role
	if err := db.DB.First(&role, id).Error; err != nil {
		c.JSON(http.StatusNotFound, apiError("role not found"))
		return
	}

	// Prevent deletion of default roles
	if role.IsDefault {
		c.JSON(http.StatusBadRequest, apiError("cannot delete default role"))
		return
	}

	if err := db.DB.Delete(&role).Error; err != nil {
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
	var permissions []models.Permission
	if err := db.DB.Order("category ASC, code ASC").Find(&permissions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": permissions,
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

	var profile models.Profile
	if err := db.DB.First(&profile, profileID).Error; err != nil {
		c.JSON(http.StatusNotFound, apiError("profile not found"))
		return
	}

	var role models.Role
	if err := db.DB.First(&role, input.RoleID).Error; err != nil {
		c.JSON(http.StatusNotFound, apiError("role not found"))
		return
	}

	if err := db.DB.Model(&profile).Association("Roles").Append(&role); err != nil {
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

	var profile models.Profile
	if err := db.DB.First(&profile, profileID).Error; err != nil {
		c.JSON(http.StatusNotFound, apiError("profile not found"))
		return
	}

	var role models.Role
	if err := db.DB.First(&role, roleID).Error; err != nil {
		c.JSON(http.StatusNotFound, apiError("role not found"))
		return
	}

	if err := db.DB.Model(&profile).Association("Roles").Delete(&role); err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Role removed successfully",
	})
}

// ============================================================
// HELPER FUNCTIONS
// ============================================================

func parseRFC3339(s string) (time.Time, error) {
	return time.Parse(time.RFC3339, s)
}

func countRecords(model interface{}) (int64, error) {
	var count int64
	return count, db.DB.Model(model).Count(&count).Error
}

func contains(slice []uint, item uint) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// ============================================================
// SUBMISSION RESCORE ENDPOINTS
// ============================================================

// AdminSubmissionRescore - POST /admin/submission/:id/rescore
// Rescores a single submission (recalculates points based on current test cases)
func AdminSubmissionRescore(c *gin.Context) {
	user, _, ok := resolveUserProfile(c)
	if !ok {
		return
	}

	if !user.IsSuperuser {
		c.JSON(http.StatusForbidden, apiError("admin access required"))
		return
	}

	submissionIDStr := c.Param("id")
	var submissionID uint
	if err := parseUint(submissionIDStr, &submissionID); err != nil {
		c.JSON(http.StatusBadRequest, apiError("invalid submission ID"))
		return
	}

	var submission models.Submission
	if err := db.DB.First(&submission, submissionID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, apiError("submission not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, apiError("database error"))
		return
	}

	// Requeue the submission for rejudging
	if err := jobs.EnqueueJudgeSubmission(submission.ID); err != nil {
		c.JSON(http.StatusInternalServerError, apiError("failed to requeue submission"))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "submission rescored successfully",
		"submission_id": submission.ID,
	})
}

// AdminSubmissionBatchRescoreRequest - request body for batch rescore
type AdminSubmissionBatchRescoreRequest struct {
	SubmissionIDs []uint `json:"submission_ids"`
	ProblemID     *uint  `json:"problem_id"`
	UserID        *uint  `json:"user_id"`
	DryRun        bool   `json:"dry_run"`
}

// AdminSubmissionBatchRescore - POST /admin/submissions/batch-rescore
// Rescores multiple submissions based on filters or specific IDs
func AdminSubmissionBatchRescore(c *gin.Context) {
	user, _, ok := resolveUserProfile(c)
	if !ok {
		return
	}

	if !user.IsSuperuser {
		c.JSON(http.StatusForbidden, apiError("admin access required"))
		return
	}

	var req AdminSubmissionBatchRescoreRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, apiError(err.Error()))
		return
	}

	// Build query based on filters
	query := db.DB.Model(&models.Submission{})

	if len(req.SubmissionIDs) > 0 {
		query = query.Where("id IN ?", req.SubmissionIDs)
	} else {
		// Use filters if no specific IDs provided
		if req.ProblemID != nil {
			query = query.Where("problem_id = ?", *req.ProblemID)
		}
		if req.UserID != nil {
			query = query.Where("user_id = ?", *req.UserID)
		}
	}

	// Dry run - just count
	if req.DryRun {
		var count int64
		if err := query.Count(&count).Error; err != nil {
			c.JSON(http.StatusInternalServerError, apiError("database error"))
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"count": count,
			"message": "dry run complete",
		})
		return
	}

	// Get all submissions to rescore
	var submissions []models.Submission
	if err := query.Find(&submissions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError("database error"))
		return
	}

	// Enqueue each submission for rejudging
	rescored := 0
	for _, sub := range submissions {
		if err := jobs.EnqueueJudgeSubmission(sub.ID); err == nil {
			rescored++
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   "batch rescore initiated",
		"rescored":  rescored,
		"total":     len(submissions),
	})
}

// AdminProblemRescoreAll - POST /admin/problem/:code/rescore-all
// Rescores all submissions for a specific problem
func AdminProblemRescoreAll(c *gin.Context) {
	user, _, ok := resolveUserProfile(c)
	if !ok {
		return
	}

	if !user.IsSuperuser {
		c.JSON(http.StatusForbidden, apiError("admin access required"))
		return
	}

	code := c.Param("code")

	var problem models.Problem
	if err := db.DB.Where("code = ?", code).First(&problem).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, apiError("problem not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, apiError("database error"))
		return
	}

	// Get all submissions for this problem
	var submissions []models.Submission
	if err := db.DB.Where("problem_id = ?", problem.ID).Find(&submissions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError("database error"))
		return
	}

	// Enqueue each submission for rejudging
	rescored := 0
	for _, sub := range submissions {
		if err := jobs.EnqueueJudgeSubmission(sub.ID); err == nil {
			rescored++
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   "problem rescore initiated",
		"rescored":  rescored,
		"total":     len(submissions),
		"problem_code": problem.Code,
	})
}

// ============================================================
// ADMIN COMMENT MANAGEMENT
// ============================================================

// AdminCommentList - GET /api/v2/admin/comments
// List all comments with filtering and pagination
func AdminCommentList(c *gin.Context) {
	page, pageSize := parsePagination(c)
	search := c.Query("search")
	hidden := c.Query("hidden") // "true", "false", or empty for all

	query := db.DB.Model(&models.Comment{}).
		Preload("Author.User").
		Joins("LEFT JOIN judge_profile ON judge_profile.id = judge_comment.author_id").
		Joins("LEFT JOIN auth_user ON auth_user.id = judge_profile.user_id")

	if search != "" {
		query = query.Where("judge_comment.body LIKE ? OR auth_user.username LIKE ? OR judge_comment.page LIKE ?",
			"%"+search+"%", "%"+search+"%", "%"+search+"%")
	}

	if hidden != "" {
		if hidden == "true" {
			query = query.Where("judge_comment.hidden = ?", true)
		} else if hidden == "false" {
			query = query.Where("judge_comment.hidden = ?", false)
		}
	}

	// Get total count
	var total int64
	if err := query.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	// Get comments
	var comments []models.Comment
	if err := query.
		Order("judge_comment.time DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&comments).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	type CommentItem struct {
		ID        uint      `json:"id"`
		AuthorID  uint      `json:"author_id"`
		Username  string    `json:"username"`
		Page      string    `json:"page"`
		Body      string    `json:"body"`
		Score     int       `json:"score"`
		Hidden    bool      `json:"hidden"`
		Time      time.Time `json:"time"`
		ParentID  *uint     `json:"parent_id,omitempty"`
	}

	items := make([]CommentItem, len(comments))
	for i, cm := range comments {
		username := ""
		if cm.Author.User.Username != "" {
			username = cm.Author.User.Username
		}
		items[i] = CommentItem{
			ID:       cm.ID,
			AuthorID: cm.AuthorID,
			Username: username,
			Page:     cm.Page,
			Body:     cm.Body,
			Score:    cm.Score,
			Hidden:   cm.Hidden,
			Time:     cm.Time,
			ParentID: cm.ParentID,
		}
	}

	c.JSON(http.StatusOK, apiListWithTotal(items, total))
}

// AdminCommentUpdate - PATCH /api/v2/admin/comment/:id
// Admin update comment (body, hidden status)
func AdminCommentUpdate(c *gin.Context) {
	user, profile, ok := resolveUserProfile(c)
	if !ok {
		return
	}

	if !user.IsStaff && !user.IsSuperuser {
		c.JSON(http.StatusForbidden, apiError("admin access required"))
		return
	}

	commentIDStr := c.Param("id")
	var commentID uint
	if err := parseUint(commentIDStr, &commentID); err != nil {
		c.JSON(http.StatusBadRequest, apiError("invalid comment id"))
		return
	}

	var comment models.Comment
	if err := db.DB.First(&comment, commentID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, apiError("comment not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, apiError("database error"))
		return
	}

	var req struct {
		Body   *string `json:"body,omitempty"`
		Hidden *bool   `json:"hidden,omitempty"`
		Reason string  `json:"reason,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, apiError(err.Error()))
		return
	}

	// Create revision if body is being changed
	if req.Body != nil && *req.Body != comment.Body {
		revision := models.CommentRevision{
			CommentID: comment.ID,
			EditorID:  profile.ID,
			Time:      time.Now(),
			Body:      comment.Body, // Save old body
			Reason:    req.Reason,
		}
		db.DB.Create(&revision)
		comment.Body = sanitization.SanitizeComment(*req.Body)
	}

	if req.Hidden != nil {
		comment.Hidden = *req.Hidden
	}

	if err := db.DB.Save(&comment).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError("failed to update comment"))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "comment updated",
		"comment": gin.H{
			"id":     comment.ID,
			"body":   comment.Body,
			"hidden": comment.Hidden,
		},
	})
}

// AdminCommentDelete - DELETE /api/v2/admin/comment/:id
// Hard delete a comment (admin only)
func AdminCommentDelete(c *gin.Context) {
	user, _, ok := resolveUserProfile(c)
	if !ok {
		return
	}

	if !user.IsStaff && !user.IsSuperuser {
		c.JSON(http.StatusForbidden, apiError("admin access required"))
		return
	}

	commentIDStr := c.Param("id")
	var commentID uint
	if err := parseUint(commentIDStr, &commentID); err != nil {
		c.JSON(http.StatusBadRequest, apiError("invalid comment id"))
		return
	}

	var comment models.Comment
	if err := db.DB.First(&comment, commentID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, apiError("comment not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, apiError("database error"))
		return
	}

	// Hard delete the comment
	if err := db.DB.Delete(&comment).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError("failed to delete comment"))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "comment deleted",
	})
}

// ============================================================
// ADMIN LANGUAGE MANAGEMENT
// ============================================================

// AdminLanguageList - GET /api/v2/admin/languages
// List all languages
func AdminLanguageList(c *gin.Context) {
	page, pageSize := parsePagination(c)

	var total int64
	if err := db.DB.Model(&models.Language{}).Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	var languages []models.Language
	if err := db.DB.
		Order("key ASC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&languages).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	type LanguageItem struct {
		ID               uint    `json:"id"`
		Key              string  `json:"key"`
		Name             string  `json:"name"`
		ShortName        *string `json:"short_name"`
		CommonName       string  `json:"common_name"`
		Ace              string  `json:"ace"`
		Pygments         string  `json:"pygments"`
		Extension        string  `json:"extension"`
		FileOnly         bool    `json:"file_only"`
		FileSizeLimit    int     `json:"file_size_limit"`
		IncludeInProblem bool    `json:"include_in_problem"`
		Info             string  `json:"info"`
	}

	items := make([]LanguageItem, len(languages))
	for i, lang := range languages {
		items[i] = LanguageItem{
			ID:               lang.ID,
			Key:              lang.Key,
			Name:             lang.Name,
			ShortName:        lang.ShortName,
			CommonName:       lang.CommonName,
			Ace:              lang.Ace,
			Pygments:         lang.Pygments,
			Extension:        lang.Extension,
			FileOnly:         lang.FileOnly,
			FileSizeLimit:    lang.FileSizeLimit,
			IncludeInProblem: lang.IncludeInProblem,
			Info:             lang.Info,
		}
	}

	c.JSON(http.StatusOK, apiListWithTotal(items, total))
}

// AdminLanguageDetail - GET /api/v2/admin/language/:id
// Get language detail
func AdminLanguageDetail(c *gin.Context) {
	idStr := c.Param("id")
	var id uint
	if err := parseUint(idStr, &id); err != nil {
		c.JSON(http.StatusBadRequest, apiError("invalid language id"))
		return
	}

	var lang models.Language
	if err := db.DB.First(&lang, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, apiError("language not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, apiError("database error"))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":                 lang.ID,
		"key":                lang.Key,
		"name":               lang.Name,
		"short_name":         lang.ShortName,
		"common_name":        lang.CommonName,
		"ace":                lang.Ace,
		"pygments":           lang.Pygments,
		"template":           lang.Template,
		"description":        lang.Description,
		"extension":          lang.Extension,
		"file_only":          lang.FileOnly,
		"file_size_limit":    lang.FileSizeLimit,
		"include_in_problem": lang.IncludeInProblem,
		"info":               lang.Info,
	})
}

// AdminLanguageCreateRequest - POST /api/v2/admin/languages
type AdminLanguageCreateRequest struct {
	Key              string  `json:"key" binding:"required"`
	Name             string  `json:"name" binding:"required"`
	ShortName        *string `json:"short_name"`
	CommonName       string  `json:"common_name" binding:"required"`
	Ace              string  `json:"ace" binding:"required"`
	Pygments         string  `json:"pygments" binding:"required"`
	Template         string  `json:"template"`
	Description      string  `json:"description"`
	Extension        string  `json:"extension" binding:"required"`
	FileOnly         bool    `json:"file_only"`
	FileSizeLimit    int     `json:"file_size_limit"`
	IncludeInProblem bool    `json:"include_in_problem"`
	Info             string  `json:"info"`
}

// AdminLanguageCreate - POST /api/v2/admin/languages
// Create a new language
func AdminLanguageCreate(c *gin.Context) {
	user, _, ok := resolveUserProfile(c)
	if !ok {
		return
	}

	if !user.IsSuperuser {
		c.JSON(http.StatusForbidden, apiError("admin access required"))
		return
	}

	var req AdminLanguageCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, apiError(err.Error()))
		return
	}

	// Check if key already exists
	var existing models.Language
	if err := db.DB.Where("key = ?", req.Key).First(&existing).Error; err == nil {
		c.JSON(http.StatusBadRequest, apiError("language key already exists"))
		return
	}

	lang := models.Language{
		Key:              req.Key,
		Name:             req.Name,
		ShortName:        req.ShortName,
		CommonName:       req.CommonName,
		Ace:              req.Ace,
		Pygments:         req.Pygments,
		Template:         req.Template,
		Description:      req.Description,
		Extension:        req.Extension,
		FileOnly:         req.FileOnly,
		FileSizeLimit:    req.FileSizeLimit,
		IncludeInProblem: req.IncludeInProblem,
		Info:             req.Info,
	}

	if err := db.DB.Create(&lang).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError("failed to create language"))
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":  "language created",
		"language": gin.H{"id": lang.ID, "key": lang.Key},
	})
}

// AdminLanguageUpdateRequest - PATCH /api/v2/admin/language/:id
type AdminLanguageUpdateRequest struct {
	Name             *string `json:"name"`
	ShortName        *string `json:"short_name"`
	CommonName       *string `json:"common_name"`
	Ace              *string `json:"ace"`
	Pygments         *string `json:"pygments"`
	Template         *string `json:"template"`
	Description      *string `json:"description"`
	Extension        *string `json:"extension"`
	FileOnly         *bool   `json:"file_only"`
	FileSizeLimit    *int    `json:"file_size_limit"`
	IncludeInProblem *bool   `json:"include_in_problem"`
	Info             *string `json:"info"`
}

// AdminLanguageUpdate - PATCH /api/v2/admin/language/:id
// Update a language
func AdminLanguageUpdate(c *gin.Context) {
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
		c.JSON(http.StatusBadRequest, apiError("invalid language id"))
		return
	}

	var lang models.Language
	if err := db.DB.First(&lang, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, apiError("language not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, apiError("database error"))
		return
	}

	var req AdminLanguageUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, apiError(err.Error()))
		return
	}

	// Update fields if provided
	if req.Name != nil {
		lang.Name = *req.Name
	}
	if req.ShortName != nil {
		lang.ShortName = req.ShortName
	}
	if req.CommonName != nil {
		lang.CommonName = *req.CommonName
	}
	if req.Ace != nil {
		lang.Ace = *req.Ace
	}
	if req.Pygments != nil {
		lang.Pygments = *req.Pygments
	}
	if req.Template != nil {
		lang.Template = *req.Template
	}
	if req.Description != nil {
		lang.Description = *req.Description
	}
	if req.Extension != nil {
		lang.Extension = *req.Extension
	}
	if req.FileOnly != nil {
		lang.FileOnly = *req.FileOnly
	}
	if req.FileSizeLimit != nil {
		lang.FileSizeLimit = *req.FileSizeLimit
	}
	if req.IncludeInProblem != nil {
		lang.IncludeInProblem = *req.IncludeInProblem
	}
	if req.Info != nil {
		lang.Info = *req.Info
	}

	if err := db.DB.Save(&lang).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError("failed to update language"))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "language updated",
		"language": gin.H{"id": lang.ID, "key": lang.Key},
	})
}

// AdminLanguageDelete - DELETE /api/v2/admin/language/:id
// Delete a language
func AdminLanguageDelete(c *gin.Context) {
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
		c.JSON(http.StatusBadRequest, apiError("invalid language id"))
		return
	}

	var lang models.Language
	if err := db.DB.First(&lang, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, apiError("language not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, apiError("database error"))
		return
	}

	// Check if language is used in submissions
	var submissionCount int64
	db.DB.Model(&models.Submission{}).Where("language_id = ?", id).Count(&submissionCount)
	if submissionCount > 0 {
		c.JSON(http.StatusBadRequest, apiError("cannot delete language with existing submissions"))
		return
	}

	if err := db.DB.Delete(&lang).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError("failed to delete language"))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "language deleted",
	})
}

// ============================================================
// ADMIN BLOG POST MANAGEMENT
// ============================================================

// AdminBlogPostList - GET /api/v2/admin/blog-posts
// List all blog posts
func AdminBlogPostList(c *gin.Context) {
	page, pageSize := parsePagination(c)

	var total int64
	if err := db.DB.Model(&models.BlogPost{}).Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	var posts []models.BlogPost
	if err := db.DB.
		Preload("Authors.User").
		Preload("Organization").
		Order("publish_on DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&posts).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	type BlogPostItem struct {
		ID           uint      `json:"id"`
		Title        string    `json:"title"`
		Slug         string    `json:"slug"`
		AuthorNames  []string  `json:"author_names"`
		PublishOn    time.Time `json:"publish_on"`
		Visible      bool      `json:"visible"`
		Sticky       bool      `json:"sticky"`
		GlobalPost   bool      `json:"global_post"`
		Organization *string   `json:"organization,omitempty"`
		Score        int       `json:"score"`
	}

	items := make([]BlogPostItem, len(posts))
	for i, post := range posts {
		authorNames := make([]string, len(post.Authors))
		for j, author := range post.Authors {
			authorNames[j] = author.User.Username
		}
		var orgName *string
		if post.Organization != nil {
			orgName = &post.Organization.Name
		}
		items[i] = BlogPostItem{
			ID:           post.ID,
			Title:        post.Title,
			Slug:         post.Slug,
			AuthorNames:  authorNames,
			PublishOn:    post.PublishOn,
			Visible:      post.Visible,
			Sticky:       post.Sticky,
			GlobalPost:   post.GlobalPost,
			Organization: orgName,
			Score:        post.Score,
		}
	}

	c.JSON(http.StatusOK, apiListWithTotal(items, total))
}

// AdminBlogPostDetail - GET /api/v2/admin/blog-post/:id
// Get blog post detail
func AdminBlogPostDetail(c *gin.Context) {
	idStr := c.Param("id")
	var id uint
	if err := parseUint(idStr, &id); err != nil {
		c.JSON(http.StatusBadRequest, apiError("invalid blog post id"))
		return
	}

	var post models.BlogPost
	if err := db.DB.
		Preload("Authors.User").
		Preload("Organization").
		First(&post, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, apiError("blog post not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, apiError("database error"))
		return
	}

	authorIDs := make([]uint, len(post.Authors))
	authorNames := make([]string, len(post.Authors))
	for i, author := range post.Authors {
		authorIDs[i] = author.ID
		authorNames[i] = author.User.Username
	}

	var orgID *uint
	var orgName *string
	if post.Organization != nil {
		orgID = &post.Organization.ID
		orgName = &post.Organization.Name
	}

	c.JSON(http.StatusOK, gin.H{
		"id":           post.ID,
		"title":        post.Title,
		"slug":         post.Slug,
		"content":      post.Content,
		"summary":      post.Summary,
		"author_ids":   authorIDs,
		"author_names": authorNames,
		"publish_on":   post.PublishOn,
		"visible":      post.Visible,
		"sticky":       post.Sticky,
		"global_post":  post.GlobalPost,
		"og_image":     post.OgImage,
		"organization_id": orgID,
		"organization_name": orgName,
		"score":        post.Score,
	})
}

// AdminBlogPostCreateRequest - POST /api/v2/admin/blog-posts
type AdminBlogPostCreateRequest struct {
	Title        string    `json:"title" binding:"required"`
	Slug         string    `json:"slug" binding:"required"`
	Content      string    `json:"content" binding:"required"`
	Summary      string    `json:"summary" binding:"required"`
	AuthorIDs    []uint    `json:"author_ids"`
	PublishOn    time.Time `json:"publish_on" binding:"required"`
	Visible      bool      `json:"visible"`
	Sticky       bool      `json:"sticky"`
	GlobalPost   bool      `json:"global_post"`
	OgImage      string    `json:"og_image"`
	OrganizationID *uint   `json:"organization_id"`
}

// AdminBlogPostCreate - POST /api/v2/admin/blog-posts
// Create a new blog post
func AdminBlogPostCreate(c *gin.Context) {
	user, profile, ok := resolveUserProfile(c)
	if !ok {
		return
	}

	if !user.IsSuperuser && !user.IsStaff {
		c.JSON(http.StatusForbidden, apiError("admin access required"))
		return
	}

	var req AdminBlogPostCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, apiError(err.Error()))
		return
	}

	// Check if slug already exists
	var existing models.BlogPost
	if err := db.DB.Where("slug = ?", req.Slug).First(&existing).Error; err == nil {
		c.JSON(http.StatusBadRequest, apiError("slug already exists"))
		return
	}

	post := models.BlogPost{
		Title:          req.Title,
		Slug:           req.Slug,
		Content:        req.Content,
		Summary:        req.Summary,
		PublishOn:      req.PublishOn,
		Visible:        req.Visible,
		Sticky:         req.Sticky,
		GlobalPost:     req.GlobalPost,
		OgImage:        req.OgImage,
		OrganizationID: req.OrganizationID,
	}

	// Set default author if none provided
	if len(req.AuthorIDs) == 0 {
		post.AuthorID = profile.ID
	} else {
		post.AuthorID = req.AuthorIDs[0]
	}

	if err := db.DB.Create(&post).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError("failed to create blog post"))
		return
	}

	// Set authors many-to-many
	if len(req.AuthorIDs) > 0 {
		var authors []models.Profile
		db.DB.Where("id IN ?", req.AuthorIDs).Find(&authors)
		post.Authors = authors
		db.DB.Save(&post)
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "blog post created",
		"blog_post": gin.H{"id": post.ID, "slug": post.Slug},
	})
}

// AdminBlogPostUpdateRequest - PATCH /api/v2/admin/blog-post/:id
type AdminBlogPostUpdateRequest struct {
	Title          *string    `json:"title"`
	Slug           *string    `json:"slug"`
	Content        *string    `json:"content"`
	Summary        *string    `json:"summary"`
	AuthorIDs      []uint     `json:"author_ids"`
	PublishOn      *time.Time `json:"publish_on"`
	Visible        *bool      `json:"visible"`
	Sticky         *bool      `json:"sticky"`
	GlobalPost     *bool      `json:"global_post"`
	OgImage        *string    `json:"og_image"`
	OrganizationID *uint      `json:"organization_id"`
}

// AdminBlogPostUpdate - PATCH /api/v2/admin/blog-post/:id
// Update a blog post
func AdminBlogPostUpdate(c *gin.Context) {
	user, _, ok := resolveUserProfile(c)
	if !ok {
		return
	}

	if !user.IsSuperuser && !user.IsStaff {
		c.JSON(http.StatusForbidden, apiError("admin access required"))
		return
	}

	idStr := c.Param("id")
	var id uint
	if err := parseUint(idStr, &id); err != nil {
		c.JSON(http.StatusBadRequest, apiError("invalid blog post id"))
		return
	}

	var post models.BlogPost
	if err := db.DB.First(&post, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, apiError("blog post not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, apiError("database error"))
		return
	}

	var req AdminBlogPostUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, apiError(err.Error()))
		return
	}

	// Update fields if provided
	if req.Title != nil {
		post.Title = *req.Title
	}
	if req.Slug != nil {
		// Check if new slug conflicts
		var existing models.BlogPost
		if err := db.DB.Where("slug = ? AND id != ?", *req.Slug, id).First(&existing).Error; err == nil {
			c.JSON(http.StatusBadRequest, apiError("slug already exists"))
			return
		}
		post.Slug = *req.Slug
	}
	if req.Content != nil {
		post.Content = *req.Content
	}
	if req.Summary != nil {
		post.Summary = *req.Summary
	}
	if req.PublishOn != nil {
		post.PublishOn = *req.PublishOn
	}
	if req.Visible != nil {
		post.Visible = *req.Visible
	}
	if req.Sticky != nil {
		post.Sticky = *req.Sticky
	}
	if req.GlobalPost != nil {
		post.GlobalPost = *req.GlobalPost
	}
	if req.OgImage != nil {
		post.OgImage = *req.OgImage
	}
	if req.OrganizationID != nil {
		post.OrganizationID = req.OrganizationID
	}

	if err := db.DB.Save(&post).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError("failed to update blog post"))
		return
	}

	// Update authors many-to-many
	if req.AuthorIDs != nil {
		var authors []models.Profile
		db.DB.Where("id IN ?", req.AuthorIDs).Find(&authors)
		post.Authors = authors
		db.DB.Save(&post)
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "blog post updated",
		"blog_post": gin.H{"id": post.ID, "slug": post.Slug},
	})
}

// AdminBlogPostDelete - DELETE /api/v2/admin/blog-post/:id
// Delete a blog post
func AdminBlogPostDelete(c *gin.Context) {
	user, _, ok := resolveUserProfile(c)
	if !ok {
		return
	}

	if !user.IsSuperuser && !user.IsStaff {
		c.JSON(http.StatusForbidden, apiError("admin access required"))
		return
	}

	idStr := c.Param("id")
	var id uint
	if err := parseUint(idStr, &id); err != nil {
		c.JSON(http.StatusBadRequest, apiError("invalid blog post id"))
		return
	}

	var post models.BlogPost
	if err := db.DB.First(&post, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, apiError("blog post not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, apiError("database error"))
		return
	}

	if err := db.DB.Delete(&post).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError("failed to delete blog post"))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "blog post deleted",
	})
}

// ============================================================
// ADMIN LICENSE MANAGEMENT
// ============================================================

// AdminLicenseList - GET /api/v2/admin/licenses
// List all licenses
func AdminLicenseList(c *gin.Context) {
	page, pageSize := parsePagination(c)

	var total int64
	if err := db.DB.Model(&models.License{}).Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	var licenses []models.License
	if err := db.DB.
		Order("key ASC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&licenses).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	type LicenseItem struct {
		ID      uint   `json:"id"`
		Key     string `json:"key"`
		Name    string `json:"name"`
		Link    string `json:"link"`
		Display string `json:"display"`
		Icon    string `json:"icon"`
	}

	items := make([]LicenseItem, len(licenses))
	for i, lic := range licenses {
		items[i] = LicenseItem{
			ID:      lic.ID,
			Key:     lic.Key,
			Name:    lic.Name,
			Link:    lic.Link,
			Display: lic.Display,
			Icon:    lic.Icon,
		}
	}

	c.JSON(http.StatusOK, apiListWithTotal(items, total))
}

// AdminLicenseDetail - GET /api/v2/admin/license/:id
// Get license detail
func AdminLicenseDetail(c *gin.Context) {
	idStr := c.Param("id")
	var id uint
	if err := parseUint(idStr, &id); err != nil {
		c.JSON(http.StatusBadRequest, apiError("invalid license id"))
		return
	}

	var lic models.License
	if err := db.DB.First(&lic, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, apiError("license not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, apiError("database error"))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":      lic.ID,
		"key":     lic.Key,
		"name":    lic.Name,
		"link":    lic.Link,
		"display": lic.Display,
		"icon":    lic.Icon,
		"text":    lic.Text,
	})
}

// AdminLicenseCreateRequest - POST /api/v2/admin/licenses
type AdminLicenseCreateRequest struct {
	Key     string `json:"key" binding:"required"`
	Link    string `json:"link" binding:"required"`
	Name    string `json:"name" binding:"required"`
	Display string `json:"display"`
	Icon    string `json:"icon"`
	Text    string `json:"text"`
}

// AdminLicenseCreate - POST /api/v2/admin/licenses
// Create a new license
func AdminLicenseCreate(c *gin.Context) {
	user, _, ok := resolveUserProfile(c)
	if !ok {
		return
	}

	if !user.IsSuperuser {
		c.JSON(http.StatusForbidden, apiError("admin access required"))
		return
	}

	var req AdminLicenseCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, apiError(err.Error()))
		return
	}

	// Check if key already exists
	var existing models.License
	if err := db.DB.Where("key = ?", req.Key).First(&existing).Error; err == nil {
		c.JSON(http.StatusBadRequest, apiError("license key already exists"))
		return
	}

	lic := models.License{
		Key:     req.Key,
		Link:    req.Link,
		Name:    req.Name,
		Display: req.Display,
		Icon:    req.Icon,
		Text:    req.Text,
	}

	if err := db.DB.Create(&lic).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError("failed to create license"))
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "license created",
		"license": gin.H{"id": lic.ID, "key": lic.Key},
	})
}

// AdminLicenseUpdateRequest - PATCH /api/v2/admin/license/:id
type AdminLicenseUpdateRequest struct {
	Link    *string `json:"link"`
	Name    *string `json:"name"`
	Display *string `json:"display"`
	Icon    *string `json:"icon"`
	Text    *string `json:"text"`
}

// AdminLicenseUpdate - PATCH /api/v2/admin/license/:id
// Update a license
func AdminLicenseUpdate(c *gin.Context) {
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
		c.JSON(http.StatusBadRequest, apiError("invalid license id"))
		return
	}

	var lic models.License
	if err := db.DB.First(&lic, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, apiError("license not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, apiError("database error"))
		return
	}

	var req AdminLicenseUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, apiError(err.Error()))
		return
	}

	// Update fields if provided
	if req.Link != nil {
		lic.Link = *req.Link
	}
	if req.Name != nil {
		lic.Name = *req.Name
	}
	if req.Display != nil {
		lic.Display = *req.Display
	}
	if req.Icon != nil {
		lic.Icon = *req.Icon
	}
	if req.Text != nil {
		lic.Text = *req.Text
	}

	if err := db.DB.Save(&lic).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError("failed to update license"))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "license updated",
		"license": gin.H{"id": lic.ID, "key": lic.Key},
	})
}

// AdminLicenseDelete - DELETE /api/v2/admin/license/:id
// Delete a license
func AdminLicenseDelete(c *gin.Context) {
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
		c.JSON(http.StatusBadRequest, apiError("invalid license id"))
		return
	}

	var lic models.License
	if err := db.DB.First(&lic, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, apiError("license not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, apiError("database error"))
		return
	}

	// Check if license is used by problems
	var problemCount int64
	db.DB.Model(&models.Problem{}).Where("license_id = ?", id).Count(&problemCount)
	if problemCount > 0 {
		c.JSON(http.StatusBadRequest, apiError("cannot delete license used by problems"))
		return
	}

	if err := db.DB.Delete(&lic).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError("failed to delete license"))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "license deleted",
	})
}

// ============================================================
// ADMIN PROBLEM GROUP MANAGEMENT
// ============================================================

// AdminProblemGroupList - GET /api/v2/admin/problem-groups
// List all problem groups
func AdminProblemGroupList(c *gin.Context) {
	page, pageSize := parsePagination(c)

	var total int64
	if err := db.DB.Model(&models.ProblemGroup{}).Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	var groups []models.ProblemGroup
	if err := db.DB.
		Order("name ASC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&groups).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	type ProblemGroupItem struct {
		ID       uint   `json:"id"`
		Name     string `json:"name"`
		FullName string `json:"full_name"`
	}

	items := make([]ProblemGroupItem, len(groups))
	for i, g := range groups {
		items[i] = ProblemGroupItem{
			ID:       g.ID,
			Name:     g.Name,
			FullName: g.FullName,
		}
	}

	c.JSON(http.StatusOK, apiListWithTotal(items, total))
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

	var group models.ProblemGroup
	if err := db.DB.First(&group, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, apiError("problem group not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, apiError("database error"))
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

	// Check if name already exists
	var existing models.ProblemGroup
	if err := db.DB.Where("name = ?", req.Name).First(&existing).Error; err == nil {
		c.JSON(http.StatusBadRequest, apiError("problem group name already exists"))
		return
	}

	group := models.ProblemGroup{
		Name:     req.Name,
		FullName: req.FullName,
	}

	if err := db.DB.Create(&group).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError("failed to create problem group"))
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "problem group created",
		"group": gin.H{"id": group.ID, "name": group.Name},
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

	var group models.ProblemGroup
	if err := db.DB.First(&group, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, apiError("problem group not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, apiError("database error"))
		return
	}

	var req AdminProblemGroupUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, apiError(err.Error()))
		return
	}

	if req.FullName != nil {
		group.FullName = *req.FullName
	}

	if err := db.DB.Save(&group).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError("failed to update problem group"))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "problem group updated",
		"group": gin.H{"id": group.ID, "name": group.Name},
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

	var group models.ProblemGroup
	if err := db.DB.First(&group, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, apiError("problem group not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, apiError("database error"))
		return
	}

	// Check if group is used by problems
	var problemCount int64
	db.DB.Model(&models.Problem{}).Where("group_id = ?", id).Count(&problemCount)
	if problemCount > 0 {
		c.JSON(http.StatusBadRequest, apiError("cannot delete problem group used by problems"))
		return
	}

	if err := db.DB.Delete(&group).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError("failed to delete problem group"))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "problem group deleted",
	})
}

// ============================================================
// ADMIN PROBLEM TYPE MANAGEMENT
// ============================================================

// AdminProblemTypeList - GET /api/v2/admin/problem-types
// List all problem types
func AdminProblemTypeList(c *gin.Context) {
	page, pageSize := parsePagination(c)

	var total int64
	if err := db.DB.Model(&models.ProblemType{}).Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	var types []models.ProblemType
	if err := db.DB.
		Order("name ASC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&types).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	type ProblemTypeItem struct {
		ID       uint   `json:"id"`
		Name     string `json:"name"`
		FullName string `json:"full_name"`
	}

	items := make([]ProblemTypeItem, len(types))
	for i, t := range types {
		items[i] = ProblemTypeItem{
			ID:       t.ID,
			Name:     t.Name,
			FullName: t.FullName,
		}
	}

	c.JSON(http.StatusOK, apiListWithTotal(items, total))
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

	var ptype models.ProblemType
	if err := db.DB.First(&ptype, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, apiError("problem type not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, apiError("database error"))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":        ptype.ID,
		"name":      ptype.Name,
		"full_name": ptype.FullName,
	})
}

// AdminProblemTypeCreateRequest - POST /api/v2/admin/problem-types
type AdminProblemTypeCreateRequest struct {
	Name     string `json:"name" binding:"required"`
	FullName string `json:"full_name" binding:"required"`
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

	// Check if name already exists
	var existing models.ProblemType
	if err := db.DB.Where("name = ?", req.Name).First(&existing).Error; err == nil {
		c.JSON(http.StatusBadRequest, apiError("problem type name already exists"))
		return
	}

	ptype := models.ProblemType{
		Name:     req.Name,
		FullName: req.FullName,
	}

	if err := db.DB.Create(&ptype).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError("failed to create problem type"))
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "problem type created",
		"type": gin.H{"id": ptype.ID, "name": ptype.Name},
	})
}

// AdminProblemTypeUpdateRequest - PATCH /api/v2/admin/problem-type/:id
type AdminProblemTypeUpdateRequest struct {
	FullName *string `json:"full_name"`
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

	var ptype models.ProblemType
	if err := db.DB.First(&ptype, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, apiError("problem type not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, apiError("database error"))
		return
	}

	var req AdminProblemTypeUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, apiError(err.Error()))
		return
	}

	if req.FullName != nil {
		ptype.FullName = *req.FullName
	}

	if err := db.DB.Save(&ptype).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError("failed to update problem type"))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "problem type updated",
		"type": gin.H{"id": ptype.ID, "name": ptype.Name},
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

	var ptype models.ProblemType
	if err := db.DB.First(&ptype, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, apiError("problem type not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, apiError("database error"))
		return
	}

	if err := db.DB.Delete(&ptype).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError("failed to delete problem type"))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "problem type deleted",
	})
}
