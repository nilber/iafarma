package handlers

import (
	"net/http"
	"strconv"
	"time"

	"iafarma/pkg/models"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

type ErrorLogHandler struct {
	DB *gorm.DB
}

func NewErrorLogHandler(db *gorm.DB) *ErrorLogHandler {
	return &ErrorLogHandler{DB: db}
}

type ErrorLogResponse struct {
	ID            string     `json:"id"`
	TenantID      string     `json:"tenant_id"`
	CustomerPhone string     `json:"customer_phone"`
	CustomerID    *string    `json:"customer_id"`
	UserMessage   string     `json:"user_message"`
	ToolName      string     `json:"tool_name"`
	ToolArgs      string     `json:"tool_args"`
	ErrorMessage  string     `json:"error_message"`
	ErrorType     string     `json:"error_type"`
	UserResponse  string     `json:"user_response"`
	Severity      string     `json:"severity"`
	Resolved      bool       `json:"resolved"`
	ResolvedAt    *time.Time `json:"resolved_at"`
	ResolvedBy    *string    `json:"resolved_by"`
	StackTrace    string     `json:"stack_trace"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

type ErrorStatsResponse struct {
	TotalErrors      int64       `json:"total_errors"`
	UnresolvedErrors int64       `json:"unresolved_errors"`
	CriticalErrors   int64       `json:"critical_errors"`
	ErrorsByType     []TypeCount `json:"errors_by_type"`
}

type TypeCount struct {
	Type  string `json:"type"`
	Count int64  `json:"count"`
}

// GetErrorLogs returns paginated list of error logs for a tenant
func (h *ErrorLogHandler) GetErrorLogs(c echo.Context) error {
	// Para super admin, tenant_id é opcional (pode ver logs de todos os tenants)
	// Para admins normais, tenant_id vem do JWT middleware
	tenantID := c.Request().Header.Get("X-Tenant-ID")

	// Se não tem tenant_id no header, tenta pegar do query param (para super admin)
	if tenantID == "" {
		tenantID = c.QueryParam("tenant_id")
	}

	// Para super admin, se não especificou tenant, lista de todos
	userRole := c.Get("user_role")
	if userRole != "system_admin" && tenantID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Tenant ID is required"})
	}

	// Parse query parameters
	page, _ := strconv.Atoi(c.QueryParam("page"))
	if page < 1 {
		page = 1
	}

	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	if limit < 1 || limit > 100 {
		limit = 20
	}

	errorType := c.QueryParam("error_type")
	severity := c.QueryParam("severity")
	resolved := c.QueryParam("resolved")

	offset := (page - 1) * limit

	// Build query
	var query *gorm.DB
	if tenantID != "" {
		query = h.DB.Where("tenant_id = ?", tenantID)
	} else {
		// Super admin vendo todos os logs
		query = h.DB.Model(&models.AIErrorLog{})
	}

	if errorType != "" {
		query = query.Where("error_type = ?", errorType)
	}

	if severity != "" {
		query = query.Where("severity = ?", severity)
	}

	if resolved != "" {
		resolvedBool := resolved == "true"
		query = query.Where("resolved = ?", resolvedBool)
	}

	// Get total count
	var total int64
	if err := query.Model(&models.AIErrorLog{}).Count(&total).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to count error logs"})
	}

	// Get error logs
	var errorLogs []models.AIErrorLog
	if err := query.Order("created_at DESC").Offset(offset).Limit(limit).Find(&errorLogs).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to get error logs"})
	}

	// Convert to response format
	response := make([]ErrorLogResponse, len(errorLogs))
	for i, log := range errorLogs {
		var customerID *string
		if log.CustomerID != (uuid.UUID{}) {
			customerIDStr := log.CustomerID.String()
			customerID = &customerIDStr
		}

		var resolvedBy *string
		if log.ResolvedBy != nil {
			resolvedByStr := log.ResolvedBy.String()
			resolvedBy = &resolvedByStr
		}

		response[i] = ErrorLogResponse{
			ID:            log.ID.String(),
			TenantID:      log.TenantID.String(),
			CustomerPhone: log.CustomerPhone,
			CustomerID:    customerID,
			UserMessage:   log.UserMessage,
			ToolName:      log.ToolName,
			ToolArgs:      log.ToolArgs,
			ErrorMessage:  log.ErrorMessage,
			ErrorType:     log.ErrorType,
			UserResponse:  log.UserResponse,
			Severity:      log.Severity,
			Resolved:      log.Resolved,
			ResolvedAt:    log.ResolvedAt,
			ResolvedBy:    resolvedBy,
			StackTrace:    log.StackTrace,
			CreatedAt:     log.CreatedAt,
			UpdatedAt:     log.UpdatedAt,
		}
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"error_logs": response,
		"pagination": map[string]interface{}{
			"page":  page,
			"limit": limit,
			"total": total,
			"pages": (total + int64(limit) - 1) / int64(limit),
		},
	})
}

// GetErrorStats returns error statistics for a tenant
func (h *ErrorLogHandler) GetErrorStats(c echo.Context) error {
	// Para super admin, tenant_id é opcional
	tenantID := c.Request().Header.Get("X-Tenant-ID")

	// Se não tem tenant_id no header, tenta pegar do query param (para super admin)
	if tenantID == "" {
		tenantID = c.QueryParam("tenant_id")
	}

	// Para super admin, se não especificou tenant, mostra stats agregadas
	userRole := c.Get("user_role")
	if userRole != "system_admin" && tenantID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Tenant ID is required"})
	}

	var stats ErrorStatsResponse

	var query *gorm.DB
	if tenantID != "" {
		query = h.DB.Where("tenant_id = ?", tenantID)
	} else {
		// Super admin vendo stats de todos os tenants
		query = h.DB.Model(&models.AIErrorLog{})
	}

	// Total errors
	if err := query.Model(&models.AIErrorLog{}).Count(&stats.TotalErrors).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to get total errors"})
	}

	// Unresolved errors
	var unresolvedQuery *gorm.DB
	if tenantID != "" {
		unresolvedQuery = h.DB.Model(&models.AIErrorLog{}).Where("tenant_id = ? AND resolved = false", tenantID)
	} else {
		unresolvedQuery = h.DB.Model(&models.AIErrorLog{}).Where("resolved = false")
	}
	if err := unresolvedQuery.Count(&stats.UnresolvedErrors).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to get unresolved errors"})
	}

	// Critical errors
	var criticalQuery *gorm.DB
	if tenantID != "" {
		criticalQuery = h.DB.Model(&models.AIErrorLog{}).Where("tenant_id = ? AND severity = 'critical'", tenantID)
	} else {
		criticalQuery = h.DB.Model(&models.AIErrorLog{}).Where("severity = 'critical'")
	}
	if err := criticalQuery.Count(&stats.CriticalErrors).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to get critical errors"})
	}

	// Errors by type
	var typeCounts []TypeCount
	var typeQuery *gorm.DB
	if tenantID != "" {
		typeQuery = h.DB.Model(&models.AIErrorLog{}).
			Select("error_type as type, count(*) as count").
			Where("tenant_id = ?", tenantID).
			Group("error_type")
	} else {
		typeQuery = h.DB.Model(&models.AIErrorLog{}).
			Select("error_type as type, count(*) as count").
			Group("error_type")
	}
	if err := typeQuery.Find(&typeCounts).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to get errors by type"})
	}

	stats.ErrorsByType = typeCounts

	return c.JSON(http.StatusOK, stats)
}

// ResolveError marks an error as resolved
func (h *ErrorLogHandler) ResolveError(c echo.Context) error {
	// Para super admin, tenant_id é opcional se especificar o ID completo do erro
	tenantID := c.Request().Header.Get("X-Tenant-ID")

	userRole := c.Get("user_role")
	if userRole != "system_admin" && tenantID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Tenant ID is required"})
	}

	errorID := c.Param("id")
	if errorID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Error ID is required"})
	}

	// Update error as resolved
	now := time.Now()
	var result *gorm.DB
	if tenantID != "" {
		result = h.DB.Model(&models.AIErrorLog{}).
			Where("id = ? AND tenant_id = ?", errorID, tenantID).
			Updates(map[string]interface{}{
				"resolved":    true,
				"resolved_at": &now,
				"updated_at":  now,
			})
	} else {
		// Super admin pode resolver qualquer erro
		result = h.DB.Model(&models.AIErrorLog{}).
			Where("id = ?", errorID).
			Updates(map[string]interface{}{
				"resolved":    true,
				"resolved_at": &now,
				"updated_at":  now,
			})
	}

	if result.Error != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to resolve error"})
	}

	if result.RowsAffected == 0 {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Error not found"})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "Error resolved successfully"})
}
