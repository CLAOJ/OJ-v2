package v2

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/CLAOJ/claoj-go/db"
	"github.com/CLAOJ/claoj-go/models"
	"github.com/CLAOJ/claoj-go/scoring"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// OrganizationList – GET /api/v2/organizations
func OrganizationList(c *gin.Context) {
	page, pageSize := parsePagination(c)
	var orgs []models.Organization

	if err := db.DB.
		Where("is_unlisted = ?", false).
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
		IsOpen      bool    `json:"is_open"`
		MemberCount int     `json:"member_count"`
		PP          float64 `json:"performance_points"`
	}
	items := make([]Item, len(orgs))
	for i, o := range orgs {
		items[i] = Item{o.ID, o.Name, o.Slug, o.ShortName, o.IsOpen, o.MemberCount, o.PerformancePoints}
	}
	c.JSON(http.StatusOK, apiList(items))
}

// OrganizationDetail – GET /api/v2/organization/:id
func OrganizationDetail(c *gin.Context) {
	id := c.Param("id")
	var org models.Organization
	if err := db.DB.First(&org, id).Error; err != nil {
		c.JSON(http.StatusNotFound, apiError("organization not found"))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":                 org.ID,
		"name":               org.Name,
		"slug":               org.Slug,
		"short_name":         org.ShortName,
		"about":              org.About,
		"is_open":            org.IsOpen,
		"member_count":       org.MemberCount,
		"performance_points": org.PerformancePoints,
		"creation_date":      org.CreationDate,
	})
}

// JoinOrganization – POST /api/v2/organization/:id/join
func JoinOrganization(c *gin.Context) {
	id := c.Param("id")
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
	id := c.Param("id")
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
	id := c.Param("id")
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
func OrganizationRequestList(c *gin.Context) {
	id := c.Param("id")
	_, profile, ok := resolveUserProfile(c)
	if !ok {
		return
	}

	if !isOrgAdmin(db.DB, uint(parseInt(id)), profile.ID) {
		c.JSON(http.StatusForbidden, apiError("access denied: not an organization administrator"))
		return
	}

	var reqs []models.OrganizationRequest
	if err := db.DB.Preload("User.User").Where("organization_id = ? AND state = 'P'", id).Find(&reqs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	type Item struct {
		ID       uint      `json:"id"`
		Username string    `json:"username"`
		Time     time.Time `json:"time"`
		Reason   string    `json:"reason"`
	}
	items := make([]Item, len(reqs))
	for i, r := range reqs {
		items[i] = Item{r.ID, r.User.User.Username, r.Time, r.Reason}
	}
	c.JSON(http.StatusOK, apiList(items))
}

// HandleOrganizationRequest – POST /api/v2/organization/request/:rid/handle
func HandleOrganizationRequest(c *gin.Context) {
	rid := c.Param("rid")
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
	id := c.Param("id")
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
