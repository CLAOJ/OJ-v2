package v2

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/CLAOJ/claoj/sanitization"
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
	search := c.Query("search")
	status := c.Query("status") // "open", "closed", or ""
	assigned := c.Query("assigned") // "true", "false", or ""

	var tickets []models.Ticket

	// Admins can see all tickets, regular users see only their own
	query := db.DB.Model(&models.Ticket{}).Preload("Assignees")

	if !user.IsSuperuser {
		query = query.Where("user_id = ?", profile.ID)
	}

	// Apply filters
	if search != "" {
		query = query.Where("title LIKE ? OR notes LIKE ?", "%"+search+"%", "%"+search+"%")
	}
	if status == "open" {
		query = query.Where("is_open = ?", true)
	} else if status == "closed" {
		query = query.Where("is_open = ?", false)
	}
	if assigned == "true" {
		query = query.Joins("JOIN judge_ticket_assignees ON judge_ticket_assignees.ticket_id = judge_ticket.id").
			Where("judge_ticket_assignees.profile_id IS NOT NULL").
			Group("judge_ticket.id")
	} else if assigned == "false" {
		query = query.Where("NOT EXISTS (SELECT 1 FROM judge_ticket_assignees WHERE judge_ticket_assignees.ticket_id = judge_ticket.id)")
	}

	if err := query.
		Order("time DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&tickets).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	// Get total count with filters
	var total int64
	countQuery := db.DB.Model(&models.Ticket{})
	if !user.IsSuperuser {
		countQuery = countQuery.Where("user_id = ?", profile.ID)
	}
	if search != "" {
		countQuery = countQuery.Where("title LIKE ? OR notes LIKE ?", "%"+search+"%", "%"+search+"%")
	}
	if status == "open" {
		countQuery = countQuery.Where("is_open = ?", true)
	} else if status == "closed" {
		countQuery = countQuery.Where("is_open = ?", false)
	}
	countQuery.Count(&total)

	type Item struct {
		ID           uint      `json:"id"`
		Title        string    `json:"title"`
		IsOpen       bool      `json:"is_open"`
		IsContributive bool    `json:"is_contributive"`
		Created      time.Time `json:"created"`
		Assignees    []string  `json:"assignees"`
	}
	items := make([]Item, len(tickets))
	for i, t := range tickets {
		assigneeNames := make([]string, len(t.Assignees))
		for j, a := range t.Assignees {
			assigneeNames[j] = a.User.Username
		}
		items[i] = Item{t.ID, t.Title, t.IsOpen, t.IsContributive, t.Time, assigneeNames}
	}

	c.JSON(http.StatusOK, gin.H{
		"items": items,
		"total": total,
	})
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

	// Preload assignees for admin users
	if user.IsSuperuser {
		db.DB.Preload("Assignees.User").First(&ticket, ticketID)
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

	// Build linked item info (problem or contest)
	var linkedItem map[string]interface{}
	if ticket.ContentTypeID > 0 && ticket.ObjectID > 0 {
		// Content type 1 = Problem, 2 = Contest (adjust based on your django_content_type IDs)
		if ticket.ContentTypeID == 1 {
			var problem models.Problem
			if err := db.DB.First(&problem, ticket.ObjectID).Error; err == nil {
				linkedItem = map[string]interface{}{
					"type": "problem",
					"code": problem.Code,
					"name": problem.Name,
				}
			}
		} else if ticket.ContentTypeID == 2 {
			var contest models.Contest
			if err := db.DB.First(&contest, ticket.ObjectID).Error; err == nil {
				linkedItem = map[string]interface{}{
					"type": "contest",
					"key":  contest.Key,
					"name": contest.Name,
				}
			}
		}
	}

	assigneeUsernames := make([]string, len(ticket.Assignees))
	for i, a := range ticket.Assignees {
		assigneeUsernames[i] = a.User.Username
	}

	response := gin.H{
		"id":             ticket.ID,
		"title":          ticket.Title,
		"creator":        ticket.User.User.Username,
		"is_open":        ticket.IsOpen,
		"is_contributive": ticket.IsContributive,
		"notes":          ticket.Notes,
		"created":        ticket.Time,
		"messages":       messages,
		"assignees":      assigneeUsernames,
	}
	if linkedItem != nil {
		response["linked_item"] = linkedItem
	}

	c.JSON(http.StatusOK, response)
}

// ============================================================
// ADMIN TICKET MANAGEMENT ENDPOINTS
// ============================================================

// AdminTicketList - GET /admin/tickets
// Lists all tickets with filtering (admin only)
func AdminTicketList(c *gin.Context) {
	page, pageSize := parsePagination(c)
	search := c.Query("search")
	status := c.Query("status")
	assigned := c.Query("assigned")
	isContributive := c.Query("is_contributive")

	var tickets []models.Ticket
	query := db.DB.Model(&models.Ticket{}).Preload("User.User").Preload("Assignees.User")

	if search != "" {
		query = query.Where("title LIKE ? OR notes LIKE ?", "%"+search+"%", "%"+search+"%")
	}
	if status == "open" {
		query = query.Where("is_open = ?", true)
	} else if status == "closed" {
		query = query.Where("is_open = ?", false)
	}
	if assigned == "true" {
		query = query.Joins("JOIN judge_ticket_assignees ON judge_ticket_assignees.ticket_id = judge_ticket.id").
			Where("judge_ticket_assignees.profile_id IS NOT NULL").
			Group("judge_ticket.id")
	} else if assigned == "false" {
		query = query.Where("NOT EXISTS (SELECT 1 FROM judge_ticket_assignees WHERE judge_ticket_assignees.ticket_id = judge_ticket.id)")
	}
	if isContributive == "true" {
		query = query.Where("is_contributive = ?", true)
	} else if isContributive == "false" {
		query = query.Where("is_contributive = ?", false)
	}

	if err := query.Order("time DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&tickets).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var total int64
	db.DB.Model(&models.Ticket{}).Count(&total)

	type Item struct {
		ID             uint        `json:"id"`
		Title          string      `json:"title"`
		IsOpen         bool        `json:"is_open"`
		IsContributive bool        `json:"is_contributive"`
		Created        time.Time   `json:"created"`
		User           string      `json:"user"`
		Assignees      []string    `json:"assignees"`
		Notes          string      `json:"notes"`
	}
	items := make([]Item, len(tickets))
	for i, t := range tickets {
		assigneeNames := make([]string, len(t.Assignees))
		for j, a := range t.Assignees {
			assigneeNames[j] = a.User.Username
		}
		items[i] = Item{
			ID:             t.ID,
			Title:          t.Title,
			IsOpen:         t.IsOpen,
			IsContributive: t.IsContributive,
			Created:        t.Time,
			User:           t.User.User.Username,
			Assignees:      assigneeNames,
			Notes:          t.Notes,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"items": items,
		"total": total,
	})
}

// AdminTicketDetail - GET /admin/ticket/:id
// Gets full ticket details including notes and assignees (admin only)
func AdminTicketDetail(c *gin.Context) {
	ticketID := c.Param("id")

	var ticket models.Ticket
	if err := db.DB.Preload("User.User").Preload("Assignees.User").Preload("Messages.User.User").First(&ticket, ticketID).Error; err != nil {
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

	assigneeUsernames := make([]string, len(ticket.Assignees))
	for i, a := range ticket.Assignees {
		assigneeUsernames[i] = a.User.Username
	}

	c.JSON(http.StatusOK, gin.H{
		"id":              ticket.ID,
		"title":           ticket.Title,
		"creator":         ticket.User.User.Username,
		"is_open":         ticket.IsOpen,
		"is_contributive": ticket.IsContributive,
		"notes":           ticket.Notes,
		"created":         ticket.Time,
		"assignees":       assigneeUsernames,
		"messages":        messages,
	})
}

type AdminTicketAssignRequest struct {
	ProfileIDs []uint `json:"profile_ids"`
}

// AdminTicketAssign - POST /admin/ticket/:id/assign
// Assigns staff members to a ticket (admin only)
func AdminTicketAssign(c *gin.Context) {
	ticketIDStr := c.Param("id")
	ticketID, err := strconv.ParseUint(ticketIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ticket ID"})
		return
	}

	var ticket models.Ticket
	if err := db.DB.First(&ticket, ticketID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "ticket not found"})
		return
	}

	var req AdminTicketAssignRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Replace existing assignees with new ones
	if err := db.DB.Model(&ticket).Association("Assignees").Replace(req.ProfileIDs); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to assign"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "ticket assigned successfully"})
}

// AdminTicketToggleOpen - POST /admin/ticket/:id/toggle
// Toggles the open/closed status of a ticket (admin only)
func AdminTicketToggleOpen(c *gin.Context) {
	ticketIDStr := c.Param("id")
	ticketID, err := strconv.ParseUint(ticketIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ticket ID"})
		return
	}

	var ticket models.Ticket
	if err := db.DB.First(&ticket, ticketID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "ticket not found"})
		return
	}

	ticket.IsOpen = !ticket.IsOpen
	if err := db.DB.Save(&ticket).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update ticket"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "ticket status updated", "is_open": ticket.IsOpen})
}

// AdminTicketSetContributive - POST /admin/ticket/:id/set-contributive
// Sets the contributive flag on a ticket (admin only)
func AdminTicketSetContributive(c *gin.Context) {
	ticketIDStr := c.Param("id")
	ticketID, err := strconv.ParseUint(ticketIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ticket ID"})
		return
	}

	var req struct {
		IsContributive bool `json:"is_contributive"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var ticket models.Ticket
	if err := db.DB.First(&ticket, ticketID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "ticket not found"})
		return
	}

	ticket.IsContributive = req.IsContributive
	if err := db.DB.Save(&ticket).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update ticket"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "ticket contributive status updated", "is_contributive": ticket.IsContributive})
}

// AdminTicketUpdateNotes - PATCH /admin/ticket/:id/notes
// Updates internal notes on a ticket (admin only)
func AdminTicketUpdateNotes(c *gin.Context) {
	ticketIDStr := c.Param("id")
	ticketID, err := strconv.ParseUint(ticketIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ticket ID"})
		return
	}

	var req struct {
		Notes string `json:"notes"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var ticket models.Ticket
	if err := db.DB.First(&ticket, ticketID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "ticket not found"})
		return
	}

	ticket.Notes = req.Notes
	if err := db.DB.Save(&ticket).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update notes"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "notes updated"})
}
