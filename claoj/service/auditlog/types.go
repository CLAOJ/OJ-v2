// Package auditlog provides audit log management services.
package auditlog

import "time"

// AuditLogEntry represents an audit log entry.
type AuditLogEntry struct {
	ID        uint
	UserID    uint
	Username  string
	Action    string // CREATE, UPDATE, DELETE, LOGIN, LOGOUT, etc.
	Resource  string // User, Problem, Contest, etc.
	ResourceID string
	IPAddress string
	UserAgent string
	Details   string // JSON-encoded details
	Status    string // success, failure, blocked
	CreatedAt time.Time
}

// ListAuditLogsRequest holds parameters for listing audit logs.
type ListAuditLogsRequest struct {
	Page       int
	PageSize   int
	UserID     *uint  // Optional filter by user ID
	Action     string // Optional filter by action (e.g., CREATE, UPDATE, DELETE)
	Resource   string // Optional filter by resource
	Status     string // Optional filter by status
	DateFrom   *time.Time
	DateTo     *time.Time
}

// ListAuditLogsResponse holds the response for listing audit logs.
type ListAuditLogsResponse struct {
	Logs     []AuditLogEntry
	Total    int64
	Page     int
	PageSize int
}

// GetAuditLogRequest holds parameters for getting an audit log entry.
type GetAuditLogRequest struct {
	LogID uint
}
