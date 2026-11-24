package v2

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/CLAOJ/claoj/auth"
	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/CLAOJ/claoj/sanitization"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// TicketList – GET /api/v2/tickets
// Returns a list of tickets created by the authenticated user
func TicketList(c *gin.Context) {
	_, profile, ok := resolveUserProfile(c)
	if !ok {
		return
	}

	page, pageSize := parsePagination(c)
	search := c.Query("search")
	status := c.Query("status")     // "open", "closed", or ""
	assigned := c.Query("assigned") // "true", "false", or ""

	var tickets []models.Ticket

	// Admins can see all tickets, regular users see only their own
	query := db.DB.Model(&models.Ticket{}).Preload("Assignees")

	if !auth.HasPerm(c, "judge.change_ticket") {
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

	// Count from the same filtered statement rather than a hand-rebuilt copy,
	// which previously drifted: it never applied the `assigned` filter, so the
	// pager over-reported pages whenever that filter was active.
	var total int64
	if err := query.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	if err := query.Session(&gorm.Session{}).
		Order("time DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&tickets).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	type Item struct {
		ID             uint      `json:"id"`
		Title          string    `json:"title"`
		IsOpen         bool      `json:"is_open"`
		IsContributive bool      `json:"is_contributive"`
		Created        time.Time `json:"created"`
		Assignees      []string  `json:"assignees"`
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
	ticketID, ok := numericParam(c, "id")
	if !ok {
		return
	}
	_, profile, ok := resolveUserProfile(c)
	if !ok {
		return
	}

	var ticket models.Ticket
	query := db.DB.Preload("Messages.User.User").Preload("User.User")

	// Access control
	if !auth.HasPerm(c, "judge.change_ticket") {
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

	// Preload assignees for ticket moderators
	if auth.HasPerm(c, "judge.change_ticket") {
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
	linked := resolveTicketLinkedItem(ticket.ContentTypeID, ticket.ObjectID)
	var linkedItem map[string]interface{}
	if linked.Type != "" {
		linkedItem = map[string]interface{}{
			"type": linked.Type,
			"name": linked.Name,
		}
		if linked.Type == "problem" {
			linkedItem["code"] = linked.Code
		} else {
			linkedItem["key"] = linked.Code
		}
	}

	assigneeUsernames := make([]string, len(ticket.Assignees))
	for i, a := range ticket.Assignees {
		assigneeUsernames[i] = a.User.Username
	}

	response := gin.H{
		"id":              ticket.ID,
		"title":           ticket.Title,
		"creator":         ticket.User.User.Username,
		"is_open":         ticket.IsOpen,
		"is_contributive": ticket.IsContributive,
		"notes":           ticket.Notes,
		"created":         ticket.Time,
		"messages":        messages,
		"assignees":       assigneeUsernames,
	}
	if linkedItem != nil {
		response["linked_item"] = linkedItem
	}

	c.JSON(http.StatusOK, response)
}

// ============================================================
// ADMIN TICKET MANAGEMENT ENDPOINTS
// ============================================================

// ticketLinkedItem is a ticket's generic reference to a problem or contest.
// Type is "" when the ticket is unlinked or the reference is orphaned.
type ticketLinkedItem struct {
	Type string // "problem" | "contest" | ""
	ID   uint
	Code string // problem code or contest key
	Name string
}

// LinkedLabel renders the v1 admin list_display label (code + name).
func (l ticketLinkedItem) LinkedLabel() string {
	if l.Type == "" {
		return ""
	}
	return l.Code + " – " + l.Name
}

// LinkedPath is the public site path of the linked item.
func (l ticketLinkedItem) LinkedPath() string {
	switch l.Type {
	case "problem":
		return "/problem/" + l.Code
	case "contest":
		return "/contest/" + l.Code
	}
	return ""
}

// resolveTicketLinkedItem loads the object referenced by a ticket's
// content_type/object_id pair. The type comes from django_content_type's
// app_label/model: ids depend on migration order, so they must never be
// hardcoded.
func resolveTicketLinkedItem(contentTypeID, objectID uint) ticketLinkedItem {
	if contentTypeID == 0 || objectID == 0 {
		return ticketLinkedItem{}
	}
	var ct models.DjangoContentType
	if err := db.DB.First(&ct, contentTypeID).Error; err != nil || ct.AppLabel != "judge" {
		return ticketLinkedItem{}
	}
	switch ct.Model {
	case "problem":
		var problem models.Problem
		if err := db.DB.First(&problem, objectID).Error; err != nil {
			return ticketLinkedItem{}
		}
		return ticketLinkedItem{Type: "problem", ID: problem.ID, Code: problem.Code, Name: problem.Name}
	case "contest":
		var contest models.Contest
		if err := db.DB.First(&contest, objectID).Error; err != nil {
			return ticketLinkedItem{}
		}
		return ticketLinkedItem{Type: "contest", ID: contest.ID, Code: contest.Key, Name: contest.Name}
	}
	return ticketLinkedItem{}
}

// ticketContentTypeID maps a public linked type name to its
// django_content_type row.
func ticketContentTypeID(linkedType string) (uint, bool) {
	if linkedType != "problem" && linkedType != "contest" {
		return 0, false
	}
	var ct models.DjangoContentType
	if err := db.DB.Where("app_label = ? AND model = ?", "judge", linkedType).First(&ct).Error; err != nil {
		return 0, false
	}
	return ct.ID, true
}

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
		query = query.Where("EXISTS (SELECT 1 FROM judge_ticket_assignees WHERE judge_ticket_assignees.ticket_id = judge_ticket.id)")
	} else if assigned == "false" {
		query = query.Where("NOT EXISTS (SELECT 1 FROM judge_ticket_assignees WHERE judge_ticket_assignees.ticket_id = judge_ticket.id)")
	}
	if isContributive == "true" {
		query = query.Where("is_contributive = ?", true)
	} else if isContributive == "false" {
		query = query.Where("is_contributive = ?", false)
	}

	// Count with the same filters as the rows (it previously ignored them).
	var total int64
	if err := query.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if err := query.Order("time DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&tickets).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	type Item struct {
		ID             uint      `json:"id"`
		Title          string    `json:"title"`
		IsOpen         bool      `json:"is_open"`
		IsContributive bool      `json:"is_contributive"`
		Created        time.Time `json:"created"`
		User           string    `json:"user"`
		Assignees      []string  `json:"assignees"`
		Notes          string    `json:"notes"`
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
	ticketID, ok := numericParam(c, "id")
	if !ok {
		return
	}

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

	linked := resolveTicketLinkedItem(ticket.ContentTypeID, ticket.ObjectID)

	c.JSON(http.StatusOK, gin.H{
		"id":              ticket.ID,
		"title":           ticket.Title,
		"creator":         ticket.User.User.Username,
		"creator_id":      ticket.User.ID,
		"is_open":         ticket.IsOpen,
		"is_contributive": ticket.IsContributive,
		"notes":           ticket.Notes,
		"created":         ticket.Time,
		"assignees":       assigneeUsernames,
		"messages":        messages,
		"linked_type":     linked.Type,
		"linked_id":       linked.ID,
		"linked_label":    linked.LinkedLabel(),
		"linked_path":     linked.LinkedPath(),
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

type AdminTicketUpdateRequest struct {
	Title      *string `json:"title"`
	UserID     *uint   `json:"user_id"`     // judge_profile id
	LinkedType *string `json:"linked_type"` // "problem" | "contest" | ""
	LinkedID   *uint   `json:"linked_id"`
}

// AdminTicketUpdate - PATCH /admin/ticket/:id
// Edits the v1-editable ticket fields (title, creator, linked item). `time`
// stays read-only, so the request struct has no such field.
func AdminTicketUpdate(c *gin.Context) {
	ticketID, ok := numericParam(c, "id")
	if !ok {
		return
	}

	var req AdminTicketUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var ticket models.Ticket
	if err := db.DB.First(&ticket, ticketID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "ticket not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}

	if req.Title != nil {
		title := sanitization.SanitizeTitle(*req.Title)
		if title == "" {
			c.JSON(http.StatusBadRequest, apiUserError("title is required"))
			return
		}
		ticket.Title = title
	}

	if req.UserID != nil {
		var profile models.Profile
		if err := db.DB.Select("id").First(&profile, *req.UserID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusBadRequest, apiUserError("user not found"))
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
			return
		}
		// judge_ticket.user_id is a judge_profile id, not an auth_user id.
		ticket.UserID = profile.ID
	}

	if req.LinkedType != nil || req.LinkedID != nil {
		if req.LinkedType == nil || req.LinkedID == nil {
			c.JSON(http.StatusBadRequest, apiUserError("linked_type and linked_id must be sent together"))
			return
		}
		linkedType, linkedID := *req.LinkedType, *req.LinkedID
		if linkedType == "" {
			if linkedID != 0 {
				c.JSON(http.StatusBadRequest, apiUserError("linked_id must be 0 when linked_type is empty"))
				return
			}
			ticket.ContentTypeID, ticket.ObjectID = 0, 0
		} else {
			if linkedID == 0 {
				c.JSON(http.StatusBadRequest, apiUserError("linked_id is required with linked_type"))
				return
			}
			contentTypeID, known := ticketContentTypeID(linkedType)
			if !known {
				c.JSON(http.StatusBadRequest, apiUserError("unknown linked_type"))
				return
			}
			if resolveTicketLinkedItem(contentTypeID, linkedID).Type == "" {
				c.JSON(http.StatusBadRequest, apiUserError("linked item not found"))
				return
			}
			ticket.ContentTypeID, ticket.ObjectID = contentTypeID, linkedID
		}
	}

	if err := db.DB.Save(&ticket).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update ticket"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "ticket updated"})
}

type AdminTicketMessageUpdateRequest struct {
	Body *string `json:"body"`
}

// loadTicketMessage resolves the :id/:messageId pair, writing 404 when the
// ticket or a message that does not belong to it is missing.
func loadTicketMessage(c *gin.Context, ticketID, messageID string) (*models.TicketMessage, bool) {
	var message models.TicketMessage
	if err := db.DB.Where("ticket_id = ?", ticketID).First(&message, messageID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "message not found"})
			return nil, false
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return nil, false
	}
	return &message, true
}

// AdminTicketMessageUpdate - PATCH /admin/ticket/:id/message/:messageId
func AdminTicketMessageUpdate(c *gin.Context) {
	ticketID, ok := numericParam(c, "id")
	if !ok {
		return
	}
	messageID, ok := numericParam(c, "messageId")
	if !ok {
		return
	}

	var ticket models.Ticket
	if err := db.DB.Select("id").First(&ticket, ticketID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "ticket not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}

	var req AdminTicketMessageUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	message, ok := loadTicketMessage(c, ticketID, messageID)
	if !ok {
		return
	}

	if req.Body != nil {
		body := sanitization.SanitizeTicketBody(*req.Body)
		if body == "" {
			c.JSON(http.StatusBadRequest, apiUserError("body is required"))
			return
		}
		message.Body = body
	}

	if err := db.DB.Save(message).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update message"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "message updated"})
}

// AdminTicketMessageDelete - DELETE /admin/ticket/:id/message/:messageId
func AdminTicketMessageDelete(c *gin.Context) {
	ticketID, ok := numericParam(c, "id")
	if !ok {
		return
	}
	messageID, ok := numericParam(c, "messageId")
	if !ok {
		return
	}

	var ticket models.Ticket
	if err := db.DB.Select("id").First(&ticket, ticketID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "ticket not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}

	message, ok := loadTicketMessage(c, ticketID, messageID)
	if !ok {
		return
	}

	if err := db.DB.Delete(message).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete message"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "message deleted"})
}

// AdminTicketDelete - DELETE /admin/ticket/:id
// Permanently deletes a ticket with its messages and assignee links (admin only)
func AdminTicketDelete(c *gin.Context) {
	ticketID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ticket ID"})
		return
	}

	var ticket models.Ticket
	if err := db.DB.First(&ticket, ticketID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "ticket not found"})
		return
	}

	err = db.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("ticket_id = ?", ticket.ID).Delete(&models.TicketMessage{}).Error; err != nil {
			return err
		}
		if err := tx.Model(&ticket).Association("Assignees").Clear(); err != nil {
			return err
		}
		return tx.Delete(&ticket).Error
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete ticket"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "ticket deleted"})
}
