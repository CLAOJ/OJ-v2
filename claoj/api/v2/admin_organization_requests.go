package v2

import (
	"net/http"
	"strings"
	"time"

	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ============================================================
// ADMIN ORGANIZATION REQUESTS (read-only, v1 OrganizationRequestAdmin)
// ============================================================

// AdminOrganizationRequestList - GET /api/v2/admin/organization-requests
// list_display = username, organization, state, time; the v1 admin has no
// write actions beyond Django's own delete, which v2 does not expose.
func AdminOrganizationRequestList(c *gin.Context) {
	page, pageSize := parsePagination(c)

	state := c.Query("state")
	switch state {
	case "", "P", "A", "R":
	default:
		c.JSON(http.StatusBadRequest, apiUserError("invalid state: must be P, A or R"))
		return
	}
	search := strings.TrimSpace(c.Query("search"))

	// A fresh chain per query: GORM's Count pollutes a reused chain so the
	// following Find would silently skip its Preloads.
	build := func() *gorm.DB {
		query := db.DB.Model(&models.OrganizationRequest{})
		if state != "" {
			query = query.Where("state = ?", state)
		}
		if search != "" {
			pattern := db.ContainsPattern(search)
			query = query.Where(
				`user_id IN (SELECT p.id FROM judge_profile p JOIN auth_user u ON u.id = p.user_id
					WHERE u.username LIKE ? ESCAPE '!')
				OR organization_id IN (SELECT id FROM judge_organization
					WHERE name LIKE ? ESCAPE '!' OR slug LIKE ? ESCAPE '!')`,
				pattern, pattern, pattern)
		}
		return query
	}

	var total int64
	if err := build().Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	var requests []models.OrganizationRequest
	if err := build().
		Preload("Organization").
		Order("time DESC, id DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&requests).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	// Preload("User") is ambiguous here (both organizationrequest.user_id and
	// profile.user_id exist), so resolve usernames in one explicit query.
	profileIDs := make([]uint, len(requests))
	for i, r := range requests {
		profileIDs[i] = r.UserID
	}
	usernames := map[uint]string{}
	if len(profileIDs) > 0 {
		var profiles []models.Profile
		if err := db.DB.Preload("User").Where("id IN ?", profileIDs).Find(&profiles).Error; err != nil {
			c.JSON(http.StatusInternalServerError, apiError(err.Error()))
			return
		}
		for _, p := range profiles {
			usernames[p.ID] = p.User.Username
		}
	}

	type Item struct {
		ID               uint      `json:"id"`
		Username         string    `json:"username"`
		OrganizationID   uint      `json:"organization_id"`
		Organization     string    `json:"organization"`
		OrganizationSlug string    `json:"organization_slug"`
		State            string    `json:"state"`
		Time             time.Time `json:"time"`
		Reason           string    `json:"reason"`
	}
	items := make([]Item, len(requests))
	for i, r := range requests {
		items[i] = Item{
			ID:               r.ID,
			Username:         usernames[r.UserID],
			OrganizationID:   r.OrganizationID,
			Organization:     r.Organization.Name,
			OrganizationSlug: r.Organization.Slug,
			State:            r.State,
			Time:             r.Time,
			Reason:           r.Reason,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"data":      items,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}