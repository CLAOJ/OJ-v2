package models

import (
	"time"
)

// AuditLog represents an audit log entry for tracking admin actions
type AuditLog struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"not null;index" json:"user_id"`
	Username  string    `gorm:"size:150;not null" json:"username"`
	Action    string    `gorm:"size:50;not null;index" json:"action"` // CREATE, UPDATE, DELETE, LOGIN, LOGOUT, etc.
	Resource  string    `gorm:"size:100;index" json:"resource"`       // User, Problem, Contest, etc.
	ResourceID string   `gorm:"size:100" json:"resource_id"`          // ID or key of the affected resource
	IPAddress string    `gorm:"size:45" json:"ip_address"`            // IPv4 or IPv6
	UserAgent string    `gorm:"size:500" json:"user_agent"`
	Details   string    `gorm:"type:text" json:"details"`             // JSON-encoded details
	Status    string    `gorm:"size:20;default:'success'" json:"status"` // success, failure, blocked
	CreatedAt time.Time `gorm:"autoCreateTime;index" json:"created_at"`
}

// TableName specifies the table name for audit logs
func (AuditLog) TableName() string {
	return "audit_log"
}
