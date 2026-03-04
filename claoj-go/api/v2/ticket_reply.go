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

type TicketMessageRequest struct {
	Body string `json:"body" binding:"required"`
}

// TicketReply - POST /api/v2/ticket/:id/message
// Adds a message to an existing ticket
func TicketReply(c *gin.Context) {
	ticketID := c.Param("id")
	user, profile, ok := resolveUserProfile(c)
	if !ok {
		return
	}

	var ticket models.Ticket
	query := db.DB.Model(&models.Ticket{})

	// Access control
	if !user.IsSuperuser {
		query = query.Where("user_id = ?", profile.ID)
	}

	var req TicketMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := query.First(&ticket, ticketID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "ticket not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}

	if !ticket.IsOpen {
		c.JSON(http.StatusForbidden, gin.H{"error": "ticket is closed"})
		return
	}

	message := models.TicketMessage{
		TicketID: ticket.ID,
		UserID:   profile.ID,
		Body:     sanitization.SanitizeTicketBody(req.Body),
		Time:     time.Now(),
	}

	if err := db.DB.Create(&message).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to post message"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "reply posted"})
}
