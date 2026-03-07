package v2

import (
	"errors"
	"net/http"
	"time"

	"github.com/CLAOJ/claoj-go/service/auditlog"
	"github.com/CLAOJ/claoj-go/service/miscconfig"
	"github.com/gin-gonic/gin"
)

// ============================================================
// ADMIN MISC CONFIG API
// ============================================================

// AdminMiscConfigList - GET /api/v2/admin/misc-configs
func AdminMiscConfigList(c *gin.Context) {
	page, pageSize := parsePagination(c)

	resp, err := getMiscConfigService().ListConfig(miscconfig.ListConfigRequest{
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	type Item struct {
		ID    uint   `json:"id"`
		Key   string `json:"key"`
		Value string `json:"value"`
	}

	items := make([]Item, len(resp.Configs))
	for i, mc := range resp.Configs {
		items[i] = Item{
			ID:    mc.ID,
			Key:   mc.Key,
			Value: mc.Value,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  items,
		"total": resp.Total,
	})
}

// AdminMiscConfigDetail - GET /api/v2/admin/misc-config/:id
func AdminMiscConfigDetail(c *gin.Context) {
	idStr := c.Param("id")
	var id uint
	if err := parseUint(idStr, &id); err != nil {
		c.JSON(http.StatusBadRequest, apiError("invalid config id"))
		return
	}

	mc, err := getMiscConfigService().GetConfig(miscconfig.GetConfigRequest{
		ConfigID: id,
	})
	if err != nil {
		if errors.Is(err, miscconfig.ErrConfigNotFound) {
			c.JSON(http.StatusNotFound, apiError("config not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	c.JSON(http.StatusOK, mc)
}

// AdminMiscConfigCreate - POST /api/v2/admin/misc-configs
func AdminMiscConfigCreate(c *gin.Context) {
	var req struct {
		Key   string `json:"key" binding:"required"`
		Value string `json:"value"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	mc, err := getMiscConfigService().CreateConfig(miscconfig.CreateConfigRequest{
		Key:   req.Key,
		Value: req.Value,
	})
	if err != nil {
		if errors.Is(err, miscconfig.ErrKeyExists) {
			c.JSON(http.StatusConflict, gin.H{"error": "config key already exists"})
			return
		}
		if errors.Is(err, miscconfig.ErrEmptyKey) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "config key cannot be empty"})
			return
		}
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	c.JSON(http.StatusCreated, mc)
}

// AdminMiscConfigUpdate - PATCH /api/v2/admin/misc-config/:id
func AdminMiscConfigUpdate(c *gin.Context) {
	idStr := c.Param("id")
	var id uint
	if err := parseUint(idStr, &id); err != nil {
		c.JSON(http.StatusBadRequest, apiError("invalid config id"))
		return
	}

	var req struct {
		Value string `json:"value"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	mc, err := getMiscConfigService().UpdateConfig(miscconfig.UpdateConfigRequest{
		ConfigID: id,
		Value:    req.Value,
	})
	if err != nil {
		if errors.Is(err, miscconfig.ErrConfigNotFound) {
			c.JSON(http.StatusNotFound, apiError("config not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	c.JSON(http.StatusOK, mc)
}

// AdminMiscConfigDelete - DELETE /api/v2/admin/misc-config/:id
func AdminMiscConfigDelete(c *gin.Context) {
	idStr := c.Param("id")
	var id uint
	if err := parseUint(idStr, &id); err != nil {
		c.JSON(http.StatusBadRequest, apiError("invalid config id"))
		return
	}

	if err := getMiscConfigService().DeleteConfig(miscconfig.DeleteConfigRequest{
		ConfigID: id,
	}); err != nil {
		if errors.Is(err, miscconfig.ErrConfigNotFound) {
			c.JSON(http.StatusNotFound, apiError("config not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "config deleted",
	})
}

// ============================================================
// ADMIN AUDIT LOG API
// ============================================================

// AdminAuditLogList - GET /api/v2/admin/audit-logs
func AdminAuditLogList(c *gin.Context) {
	page, pageSize := parsePagination(c)

	// Get filter parameters
	action := c.Query("action")
	resource := c.Query("resource")
	userIDStr := c.Query("user_id")
	status := c.Query("status")
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	// Build request
	req := auditlog.ListAuditLogsRequest{
		Page:     page,
		PageSize: pageSize,
		Action:   action,
		Resource: resource,
		Status:   status,
	}

	// Parse optional filters
	if userIDStr != "" {
		var userID uint
		if err := parseUint(userIDStr, &userID); err == nil {
			req.UserID = &userID
		}
	}
	if startDate != "" {
		if t, err := time.Parse("2006-01-02", startDate); err == nil {
			req.DateFrom = &t
		}
	}
	if endDate != "" {
		if t, err := time.Parse("2006-01-02", endDate); err == nil {
			req.DateTo = &t
		}
	}

	resp, err := getAuditLogService().ListAuditLogs(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"results":   resp.Logs,
		"total":     resp.Total,
		"page":      resp.Page,
		"page_size": resp.PageSize,
	})
}

// AdminAuditLogDetail - GET /api/v2/admin/audit-log/:id
func AdminAuditLogDetail(c *gin.Context) {
	idStr := c.Param("id")
	var id uint
	if err := parseUint(idStr, &id); err != nil {
		c.JSON(http.StatusBadRequest, apiError("invalid log id"))
		return
	}

	log, err := getAuditLogService().GetAuditLog(auditlog.GetAuditLogRequest{
		LogID: id,
	})
	if err != nil {
		if errors.Is(err, auditlog.ErrLogNotFound) {
			c.JSON(http.StatusNotFound, apiError("log not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	c.JSON(http.StatusOK, log)
}
