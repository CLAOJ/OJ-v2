package models

import "time"

// GeneralIssue mirrors judge_generalissue.
type GeneralIssue struct {
	ID       uint   `gorm:"primaryKey;column:id"`
	IssueURL string `gorm:"column:issue_url;size:200;not null"`
}

func (GeneralIssue) TableName() string { return "judge_generalissue" }

// Ticket mirrors judge_ticket.
type Ticket struct {
	ID             uint      `gorm:"primaryKey;column:id"`
	Title          string    `gorm:"column:title;size:100;not null"`
	UserID         uint      `gorm:"column:user_id;not null;index"`
	Time           time.Time `gorm:"column:time;not null"`
	Notes          string    `gorm:"column:notes;type:longtext;not null;default:''"`
	ContentTypeID  uint      `gorm:"column:content_type_id;not null;index"` // references django_content_type
	ObjectID       uint      `gorm:"column:object_id;not null"`             // The ID of the generic linked object
	IsContributive bool      `gorm:"column:is_contributive;not null;default:0"`
	IsOpen         bool      `gorm:"column:is_open;not null;default:1"`

	// relations
	User      Profile         `gorm:"foreignKey:UserID"`
	Messages  []TicketMessage `gorm:"foreignKey:TicketID"`
	Assignees []Profile       `gorm:"many2many:judge_ticket_assignees;joinForeignKey:ticket_id;joinReferences:profile_id"`
}

func (Ticket) TableName() string { return "judge_ticket" }

// TicketMessage mirrors judge_ticketmessage.
type TicketMessage struct {
	ID       uint      `gorm:"primaryKey;column:id"`
	TicketID uint      `gorm:"column:ticket_id;not null;index"`
	UserID   uint      `gorm:"column:user_id;not null;index"`
	Body     string    `gorm:"column:body;type:longtext;not null"`
	Time     time.Time `gorm:"column:time;not null"`

	Ticket Ticket  `gorm:"foreignKey:TicketID"`
	User   Profile `gorm:"foreignKey:UserID"`
}

func (TicketMessage) TableName() string { return "judge_ticketmessage" }
