package v2

import (
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/CLAOJ/claoj/sanitization"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type contestAnnouncementItem struct {
	ID          uint      `json:"id"`
	ContestID   uint      `json:"contest_id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Date        time.Time `json:"date"`
}

func newContestAnnouncementItem(a models.ContestAnnouncement) contestAnnouncementItem {
	return contestAnnouncementItem{
		ID:          a.ID,
		ContestID:   a.ContestID,
		Title:       a.Title,
		Description: a.Description,
		Date:        a.Date,
	}
}

type contestAnnouncementRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

// validate returns a user-facing message, or "" when the request matches the
// v1 model constraints (non-blank title <= 100, non-blank description).
func (r *contestAnnouncementRequest) validate() string {
	r.Title = strings.TrimSpace(r.Title)
	if r.Title == "" {
		return "title is required"
	}
	if utf8.RuneCountInString(r.Title) > 100 {
		return "title must be at most 100 characters"
	}
	if strings.TrimSpace(r.Description) == "" {
		return "description is required"
	}
	return ""
}

// announcementContestID resolves :key; the route middleware already checked
// existence and edit permission.
func announcementContestID(c *gin.Context) (uint, bool) {
	var contest models.Contest
	if err := db.DB.Select("id").Where(&models.Contest{Key: c.Param("key")}).First(&contest).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, apiUserError("contest not found"))
			return 0, false
		}
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return 0, false
	}
	return contest.ID, true
}

// AdminContestAnnouncementList – GET /api/v2/admin/contest/:key/announcements
func AdminContestAnnouncementList(c *gin.Context) {
	contestID, ok := announcementContestID(c)
	if !ok {
		return
	}

	var announcements []models.ContestAnnouncement
	if err := db.DB.Where("contest_id = ?", contestID).Order("date DESC, id DESC").Find(&announcements).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	items := make([]contestAnnouncementItem, len(announcements))
	for i, a := range announcements {
		items[i] = newContestAnnouncementItem(a)
	}

	c.JSON(http.StatusOK, apiList(items))
}

// AdminContestAnnouncementCreate – POST /api/v2/admin/contest/:key/announcements
func AdminContestAnnouncementCreate(c *gin.Context) {
	contestID, ok := announcementContestID(c)
	if !ok {
		return
	}

	var req contestAnnouncementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, apiUserError("invalid request body"))
		return
	}
	if msg := req.validate(); msg != "" {
		c.JSON(http.StatusBadRequest, apiUserError(msg))
		return
	}

	// v1 declares date as auto_now_add: the timestamp is always server time.
	announcement := models.ContestAnnouncement{
		ContestID:   contestID,
		Title:       req.Title,
		Description: sanitization.SanitizeProblemContent(req.Description),
		Date:        time.Now(),
	}
	if err := db.DB.Omit("Contest").Create(&announcement).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	c.JSON(http.StatusCreated, newContestAnnouncementItem(announcement))
}

// AdminContestAnnouncementUpdate – PATCH /api/v2/admin/contest/:key/announcements/:id
func AdminContestAnnouncementUpdate(c *gin.Context) {
	contestID, ok := announcementContestID(c)
	if !ok {
		return
	}
	id, ok := numericParam(c, "id")
	if !ok {
		return
	}

	var announcement models.ContestAnnouncement
	if err := db.DB.Where("id = ? AND contest_id = ?", id, contestID).First(&announcement).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, apiUserError("announcement not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	var req contestAnnouncementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, apiUserError("invalid request body"))
		return
	}
	if msg := req.validate(); msg != "" {
		c.JSON(http.StatusBadRequest, apiUserError(msg))
		return
	}

	announcement.Title = req.Title
	announcement.Description = sanitization.SanitizeProblemContent(req.Description)
	if err := db.DB.Model(&announcement).
		Select("title", "description").
		Updates(&announcement).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	c.JSON(http.StatusOK, newContestAnnouncementItem(announcement))
}

// AdminContestAnnouncementDelete – DELETE /api/v2/admin/contest/:key/announcements/:id
func AdminContestAnnouncementDelete(c *gin.Context) {
	contestID, ok := announcementContestID(c)
	if !ok {
		return
	}
	id, ok := numericParam(c, "id")
	if !ok {
		return
	}

	res := db.DB.Where("id = ? AND contest_id = ?", id, contestID).Delete(&models.ContestAnnouncement{})
	if res.Error != nil {
		c.JSON(http.StatusInternalServerError, apiError(res.Error.Error()))
		return
	}
	if res.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, apiUserError("announcement not found"))
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "announcement deleted"})
}
