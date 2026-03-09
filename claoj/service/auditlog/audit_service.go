// Package auditlog provides audit log management services.
package auditlog

import (
	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"gorm.io/gorm"
)

// AuditLogService provides audit log management operations.
type AuditLogService struct{}

// NewAuditLogService creates a new AuditLogService instance.
func NewAuditLogService() *AuditLogService {
	return &AuditLogService{}
}

// ListAuditLogs retrieves a paginated list of audit logs with optional filters.
func (s *AuditLogService) ListAuditLogs(req ListAuditLogsRequest) (*ListAuditLogsResponse, error) {
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 20
	}
	if req.PageSize > 100 {
		req.PageSize = 100
	}

	var logs []models.AuditLog
	query := db.DB.Model(&models.AuditLog{})

	// Apply filters
	if req.UserID != nil {
		query = query.Where("user_id = ?", *req.UserID)
	}
	if req.Action != "" {
		query = query.Where("action = ?", req.Action)
	}
	if req.Resource != "" {
		query = query.Where("resource = ?", req.Resource)
	}
	if req.Status != "" {
		query = query.Where("status = ?", req.Status)
	}
	if req.DateFrom != nil {
		query = query.Where("created_at >= ?", *req.DateFrom)
	}
	if req.DateTo != nil {
		query = query.Where("created_at <= ?", *req.DateTo)
	}

	// Get total count
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	// Get paginated results
	if err := query.
		Order("created_at DESC").
		Offset((req.Page - 1) * req.PageSize).
		Limit(req.PageSize).
		Find(&logs).Error; err != nil {
		return nil, err
	}

	result := make([]AuditLogEntry, len(logs))
	for i, l := range logs {
		result[i] = auditLogToModel(l)
	}

	return &ListAuditLogsResponse{
		Logs:     result,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}, nil
}

// GetAuditLog retrieves an audit log entry by ID.
func (s *AuditLogService) GetAuditLog(req GetAuditLogRequest) (*AuditLogEntry, error) {
	if req.LogID == 0 {
		return nil, ErrInvalidLogID
	}

	var log models.AuditLog
	if err := db.DB.First(&log, req.LogID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrLogNotFound
		}
		return nil, err
	}

	result := auditLogToModel(log)
	return &result, nil
}

// Helper functions

func auditLogToModel(l models.AuditLog) AuditLogEntry {
	return AuditLogEntry{
		ID:        l.ID,
		UserID:    l.UserID,
		Username:  l.Username,
		Action:    l.Action,
		Resource:  l.Resource,
		ResourceID: l.ResourceID,
		IPAddress: l.IPAddress,
		UserAgent: l.UserAgent,
		Details:   l.Details,
		Status:    l.Status,
		CreatedAt: l.CreatedAt,
	}
}
