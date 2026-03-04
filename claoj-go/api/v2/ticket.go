package v2

import (
	"errors"
	"net/http"
	"time"

	"github.com/CLAOJ/claoj-go/db"
	"github.com/CLAOJ/claoj-go/models"
	"github.com/CLAOJ/claoj-go/sanitization"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// TicketList – GET /api/v2/tickets
// Returns a list of tickets created by the authenticated user
func TicketList(c *gin.Context) {
	user, profile, ok := resolveUserProfile(c)
	if !ok {
		return
	}

	page, pageSize := parsePagination(c)
	var tickets []models.Ticket

	// Admins can see all tickets, regular users see only their own
	query := db.DB.Model(&models.Ticket{})
	if !user.IsSuperuser {
		query = query.Where("user_id = ?", profile.ID)
	}

	if err := query.
		Order("time DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&tickets).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	type Item struct {
		ID      uint      `json:"id"`
		Title   string    `json:"title"`
		IsOpen  bool      `json:"is_open"`
		Created time.Time `json:"created"`
	}
	items := make([]Item, len(tickets))
	for i, t := range tickets {
		items[i] = Item{t.ID, t.Title, t.IsOpen, t.Time}
	}

	c.JSON(http.StatusOK, apiList(items))
}

type TicketCreateRequest struct {
	Title         string `json:"title" binding:"required"`
	Body          string `json:"body" binding:"required"`
	ContentTypeID uint   `json:"content_type_id"` // Generic foreign key references
	ObjectID      uint   `json:"object_id"`
}

// TicketCreate - POST /api/v2/tickets
// Creates a new ticket
func TicketCreate(c *gin.Context) {
	_, profile, ok := resolveUserProfile(c)
	if !ok {
		return
	}

	var req TicketCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := db.DB.Transaction(func(tx *gorm.DB) error {
		now := time.Now()

		ticket := models.Ticket{
			Title:          sanitization.SanitizeTitle(req.Title),
			UserID:         profile.ID,
			Time:           now,
			ContentTypeID:  req.ContentTypeID,
			ObjectID:       req.ObjectID,
			IsContributive: false,
			IsOpen:         true,
		}

		if err := tx.Create(&ticket).Error; err != nil {
			return err
		}

		message := models.TicketMessage{
			TicketID: ticket.ID,
			UserID:   profile.ID,
			Body:     sanitization.SanitizeTicketBody(req.Body),
			Time:     now,
		}

		if err := tx.Create(&message).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create ticket"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "ticket created successfully"})
}

// TicketDetail - GET /api/v2/ticket/:id
// Gets the contents of a ticket and its messages
func TicketDetail(c *gin.Context) {
	ticketID := c.Param("id")
	user, profile, ok := resolveUserProfile(c)
	if !ok {
		return
	}

	var ticket models.Ticket
	query := db.DB.Preload("Messages.User.User").Preload("User.User")

	// Access control
	if !user.IsSuperuser {
		query = query.Where("user_id = ?", profile.ID)
	}

	if err := query.First(&ticket, ticketID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "ticket not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}

	type MsgItem struct {
		ID       uint      `json:"id"`
		Body     string    `json:"body"`
		Time     time.Time `json:"time"`
		Username string    `json:"username"`
	}

	messages := make([]MsgItem, len(ticket.Messages))
	for i, m := range ticket.Messages {
		messages[i] = MsgItem{
			ID:       m.ID,
			Body:     m.Body,
			Time:     m.Time,
			Username: m.User.User.Username,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"id":       ticket.ID,
		"title":    ticket.Title,
		"creator":  ticket.User.User.Username,
		"is_open":  ticket.IsOpen,
		"created":  ticket.Time,
		"messages": messages,
	})
}
