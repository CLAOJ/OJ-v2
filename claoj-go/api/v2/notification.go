package v2

import (
	"net/http"
	"strconv"
	"time"

	"github.com/CLAOJ/claoj-go/db"
	"github.com/CLAOJ/claoj-go/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// NotificationResponse represents a notification in API responses
type NotificationResponse struct {
	ID        uint      `json:"id"`
	Type      string    `json:"type"`
	Title     string    `json:"title"`
	Message   string    `json:"message"`
	Link      string    `json:"link"`
	Read      bool      `json:"read"`
	CreatedAt time.Time `json:"created_at"`
}

// NotificationList - GET /api/v2/notifications
// Lists user notifications with pagination
func NotificationList(c *gin.Context) {
	userID := c.GetUint("userID")

	// Parse pagination params
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	// Parse read filter
	readFilter := c.Query("read")

	offset := (page - 1) * pageSize

	var notifications []models.Notification
	var total int64

	query := db.DB.Where("user_id = ?", userID)
	if readFilter == "true" {
		query = query.Where("read = ?", true)
	} else if readFilter == "false" {
		query = query.Where("read = ?", false)
	}

	query.Model(&models.Notification{}).Count(&total)
	query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&notifications)

	response := make([]NotificationResponse, len(notifications))
	for i, n := range notifications {
		response[i] = NotificationResponse{
			ID:        n.ID,
			Type:      n.Type,
			Title:     n.Title,
			Message:   n.Message,
			Link:      n.Link,
			Read:      n.Read,
			CreatedAt: n.CreatedAt,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"results":     response,
		"count":       len(response),
		"total":       total,
		"page":        page,
		"page_size":   pageSize,
		"total_pages": (total + int64(pageSize) - 1) / int64(pageSize),
	})
}

// NotificationUnreadCount - GET /api/v2/notifications/unread-count
// Returns the count of unread notifications
func NotificationUnreadCount(c *gin.Context) {
	userID := c.GetUint("userID")

	var count int64
	db.DB.Model(&models.Notification{}).Where("user_id = ? AND read = ?", userID, false).Count(&count)

	c.JSON(http.StatusOK, gin.H{
		"unread_count": count,
	})
}

// NotificationMarkRead - POST /api/v2/notifications/:id/read
// Marks a single notification as read
func NotificationMarkRead(c *gin.Context) {
	userID := c.GetUint("userID")
	notificationID := c.Param("id")

	result := db.DB.Model(&models.Notification{}).
		Where("id = ? AND user_id = ?", notificationID, userID).
		Update("read", true)

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, apiError("failed to mark notification as read"))
		return
	}

	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, apiError("notification not found"))
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Notification marked as read"})
}

// NotificationMarkAllRead - POST /api/v2/notifications/read-all
// Marks all notifications as read
func NotificationMarkAllRead(c *gin.Context) {
	userID := c.GetUint("userID")

	result := db.DB.Model(&models.Notification{}).
		Where("user_id = ? AND read = ?", userID, false).
		Update("read", true)

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, apiError("failed to mark notifications as read"))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":       "All notifications marked as read",
		"marked_count":  result.RowsAffected,
	})
}

// NotificationDelete - DELETE /api/v2/notifications/:id
// Deletes a notification
func NotificationDelete(c *gin.Context) {
	userID := c.GetUint("userID")
	notificationID := c.Param("id")

	result := db.DB.Where("id = ? AND user_id = ?", notificationID, userID).Delete(&models.Notification{})

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, apiError("failed to delete notification"))
		return
	}

	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, apiError("notification not found"))
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Notification deleted"})
}

// NotificationPreferencesGet - GET /api/v2/notifications/preferences
// Gets user's notification preferences
func NotificationPreferencesGet(c *gin.Context) {
	userID := c.GetUint("userID")

	var prefs models.NotificationPreference
	err := db.DB.Where("user_id = ?", userID).First(&prefs).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// Return default preferences
			prefs = models.NotificationPreference{
				UserID:                  userID,
				EmailOnSubmissionResult: true,
				EmailOnContestStart:     true,
				EmailOnTicketReply:      true,
				WebOnSubmissionResult:   true,
				WebOnContestStart:       true,
				WebOnTicketReply:        true,
			}
		} else {
			c.JSON(http.StatusInternalServerError, apiError("failed to get preferences"))
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"email_on_submission_result": prefs.EmailOnSubmissionResult,
		"email_on_contest_start":     prefs.EmailOnContestStart,
		"email_on_ticket_reply":      prefs.EmailOnTicketReply,
		"web_on_submission_result":   prefs.WebOnSubmissionResult,
		"web_on_contest_start":       prefs.WebOnContestStart,
		"web_on_ticket_reply":        prefs.WebOnTicketReply,
	})
}

// NotificationPreferencesUpdate - PATCH /api/v2/notifications/preferences
// Updates user's notification preferences
type NotificationPreferencesRequest struct {
	EmailOnSubmissionResult *bool `json:"email_on_submission_result,omitempty"`
	EmailOnContestStart     *bool `json:"email_on_contest_start,omitempty"`
	EmailOnTicketReply      *bool `json:"email_on_ticket_reply,omitempty"`
	WebOnSubmissionResult   *bool `json:"web_on_submission_result,omitempty"`
	WebOnContestStart       *bool `json:"web_on_contest_start,omitempty"`
	WebOnTicketReply        *bool `json:"web_on_ticket_reply,omitempty"`
}

func NotificationPreferencesUpdate(c *gin.Context) {
	userID := c.GetUint("userID")

	var req NotificationPreferencesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, apiError(err.Error()))
		return
	}

	var prefs models.NotificationPreference
	err := db.DB.Where("user_id = ?", userID).First(&prefs).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// Create new preferences
			prefs = models.NotificationPreference{
				UserID:                  userID,
				EmailOnSubmissionResult: true,
				EmailOnContestStart:     true,
				EmailOnTicketReply:      true,
				WebOnSubmissionResult:   true,
				WebOnContestStart:       true,
				WebOnTicketReply:        true,
			}
		} else {
			c.JSON(http.StatusInternalServerError, apiError("failed to get preferences"))
			return
		}
	}

	// Update fields if provided
	if req.EmailOnSubmissionResult != nil {
		prefs.EmailOnSubmissionResult = *req.EmailOnSubmissionResult
	}
	if req.EmailOnContestStart != nil {
		prefs.EmailOnContestStart = *req.EmailOnContestStart
	}
	if req.EmailOnTicketReply != nil {
		prefs.EmailOnTicketReply = *req.EmailOnTicketReply
	}
	if req.WebOnSubmissionResult != nil {
		prefs.WebOnSubmissionResult = *req.WebOnSubmissionResult
	}
	if req.WebOnContestStart != nil {
		prefs.WebOnContestStart = *req.WebOnContestStart
	}
	if req.WebOnTicketReply != nil {
		prefs.WebOnTicketReply = *req.WebOnTicketReply
	}

	if err := db.DB.Save(&prefs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError("failed to update preferences"))
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Preferences updated successfully"})
}

// CreateNotification creates a new notification for a user
func CreateNotification(userID uint, notifType, title, message, link string) (*models.Notification, error) {
	notification := models.Notification{
		UserID:    userID,
		Type:      notifType,
		Title:     title,
		Message:   message,
		Link:      link,
		Read:      false,
		CreatedAt: time.Now(),
	}

	if err := db.DB.Create(&notification).Error; err != nil {
		return nil, err
	}

	return &notification, nil
}

// CreateSubmissionResultNotification creates a notification for submission result
func CreateSubmissionResultNotification(userID uint, submissionID int, problemCode, problemName, verdict string) (*models.Notification, error) {
	title := "Submission Result"
	message := "Your submission for " + problemName + " (" + problemCode + ") received verdict: " + verdict
	link := "/submissions/" + strconv.Itoa(submissionID)

	return CreateNotification(userID, "submission", title, message, link)
}

// CreateContestStartNotification creates a notification for contest start
func CreateContestStartNotification(userID uint, contestKey, contestName string) (*models.Notification, error) {
	title := "Contest Starting"
	message := "The contest \"" + contestName + "\" is starting soon!"
	link := "/contests/" + contestKey

	return CreateNotification(userID, "contest", title, message, link)
}

// CreateTicketReplyNotification creates a notification for ticket reply
func CreateTicketReplyNotification(userID uint, ticketID int, ticketTitle string) (*models.Notification, error) {
	title := "Ticket Reply"
	message := "Your ticket \"" + ticketTitle + "\" has received a reply"
	link := "/tickets/" + strconv.Itoa(ticketID)

	return CreateNotification(userID, "ticket", title, message, link)
}
