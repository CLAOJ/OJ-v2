package models

import "time"

// Notification stores user notifications
type Notification struct {
	ID        uint      `gorm:"primaryKey;column:id"`
	UserID    uint      `gorm:"column:user_id;not null;index"`
	Type      string    `gorm:"column:type;size:20;not null"` // submission, contest, ticket, etc
	Title     string    `gorm:"column:title;size:200;not null"`
	Message   string    `gorm:"column:message;type:longtext;not null"`
	Link      string    `gorm:"column:link;size:500"`
	Read      bool      `gorm:"column:read;not null;default:0"`
	CreatedAt time.Time `gorm:"column:created_at;not null;index"`
}

func (Notification) TableName() string { return "notification" }

// NotificationPreference stores user notification preferences
type NotificationPreference struct {
	ID                       uint `gorm:"primaryKey;column:id"`
	UserID                   uint `gorm:"column:user_id;not null;uniqueIndex"`
	EmailOnSubmissionResult  bool `gorm:"column:email_on_submission_result;not null;default:1"`
	EmailOnContestStart      bool `gorm:"column:email_on_contest_start;not null;default:1"`
	EmailOnTicketReply       bool `gorm:"column:email_on_ticket_reply;not null;default:1"`
	WebOnSubmissionResult    bool `gorm:"column:web_on_submission_result;not null;default:1"`
	WebOnContestStart        bool `gorm:"column:web_on_contest_start;not null;default:1"`
	WebOnTicketReply         bool `gorm:"column:web_on_ticket_reply;not null;default:1"`
}

func (NotificationPreference) TableName() string { return "notification_preference" }
