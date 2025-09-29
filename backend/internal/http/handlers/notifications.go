package handlers

import (
	"fmt"
	"iafarma/internal/repo"
	"iafarma/internal/services"
	"iafarma/pkg/models"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

// NotificationHandler handles notification-related requests
type NotificationHandler struct {
	db               *gorm.DB
	notificationRepo *repo.NotificationRepository
	tenantRepo       *repo.TenantRepository
	emailService     *services.EmailService
}

// NewNotificationHandler creates a new notification handler
func NewNotificationHandler(db *gorm.DB, emailService *services.EmailService) *NotificationHandler {
	return &NotificationHandler{
		db:               db,
		notificationRepo: repo.NewNotificationRepository(db),
		tenantRepo:       repo.NewTenantRepository(db),
		emailService:     emailService,
	}
}

// AdminListNotifications lists all notifications for super admin
func (h *NotificationHandler) AdminListNotifications(c echo.Context) error {
	// Parse query parameters
	page, _ := strconv.Atoi(c.QueryParam("page"))
	if page < 1 {
		page = 1
	}

	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	if limit < 1 || limit > 100 {
		limit = 20
	}

	offset := (page - 1) * limit

	// Parse filters
	var tenantID *uuid.UUID
	if tenantIDStr := c.QueryParam("tenant_id"); tenantIDStr != "" {
		if id, err := uuid.Parse(tenantIDStr); err == nil {
			tenantID = &id
		}
	}

	notificationType := c.QueryParam("type")
	status := c.QueryParam("status")

	// Get notifications
	logs, total, err := h.notificationRepo.GetAllNotificationLogs(tenantID, notificationType, status, limit, offset)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to get notifications",
		})
	}

	// Calculate pagination
	totalPages := (int(total) + limit - 1) / limit

	return c.JSON(http.StatusOK, map[string]interface{}{
		"data":        logs,
		"total":       total,
		"page":        page,
		"per_page":    limit,
		"total_pages": totalPages,
	})
}

// AdminGetNotificationStats gets notification statistics for super admin
func (h *NotificationHandler) AdminGetNotificationStats(c echo.Context) error {
	// Parse tenant filter
	var tenantID *uuid.UUID
	if tenantIDStr := c.QueryParam("tenant_id"); tenantIDStr != "" {
		if id, err := uuid.Parse(tenantIDStr); err == nil {
			tenantID = &id
		}
	}

	stats, err := h.notificationRepo.GetNotificationStats(tenantID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to get notification statistics",
		})
	}

	return c.JSON(http.StatusOK, stats)
}

// AdminResendNotification resends a notification to specific tenants
func (h *NotificationHandler) AdminResendNotification(c echo.Context) error {
	var request struct {
		NotificationID uuid.UUID   `json:"notification_id" validate:"required"`
		TenantIDs      []uuid.UUID `json:"tenant_ids"`   // Optional: if empty, use original notification's tenant
		ForceResend    bool        `json:"force_resend"` // Ignores "already sent today" check
	}

	if err := c.Bind(&request); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Invalid request format",
		})
	}

	if err := c.Validate(&request); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error":   "Validation failed",
			"details": err.Error(),
		})
	}

	// Get original notification
	originalLog, err := h.notificationRepo.GetNotificationLogByID(request.NotificationID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]interface{}{
			"error": "Notification not found",
		})
	}

	// If no tenant_ids provided, use the original notification's tenant
	targetTenantIDs := request.TenantIDs
	if len(targetTenantIDs) == 0 {
		targetTenantIDs = []uuid.UUID{originalLog.TenantID}
	}

	var results []map[string]interface{}

	// Resend to each tenant
	for _, tenantID := range targetTenantIDs {
		result := map[string]interface{}{
			"tenant_id": tenantID,
			"status":    "success",
		}

		// Get tenant info and admin email
		tenant, err := h.tenantRepo.GetByID(tenantID)
		if err != nil {
			result["status"] = "error"
			result["error"] = "Tenant not found"
			results = append(results, result)
			continue
		}

		// Get tenant admin email
		var adminUser models.User
		err = h.db.Where("tenant_id = ? AND role = ?", tenantID, "tenant_admin").First(&adminUser).Error
		if err != nil {
			result["status"] = "error"
			result["error"] = "No admin user found for tenant"
			results = append(results, result)
			continue
		}

		result["tenant_name"] = tenant.Name

		// Check if already sent today (unless force resend)
		if !request.ForceResend {
			today := time.Now().Format("2006-01-02") // Use current date
			if h.notificationRepo.IsNotificationSentToday(tenantID, originalLog.Type, today) {
				result["status"] = "error"
				result["error"] = "low stock alert already sent today"
				results = append(results, result)
				continue
			}
		}

		// Send email based on notification type
		var emailErr error
		switch originalLog.Type {
		case "daily_sales_report":
			// For daily sales report, we need to generate current data
			emailErr = h.resendDailySalesReport(tenantID, tenant.Name, adminUser.Email)
		case "low_stock_alert":
			// For low stock alert, we need to check current stock
			emailErr = h.resendLowStockAlert(tenantID, tenant.Name, adminUser.Email)
		default:
			// For other notifications, use original content
			emailErr = h.emailService.SendEmail([]string{adminUser.Email}, originalLog.Subject, originalLog.Body)
		}

		if emailErr != nil {
			result["status"] = "error"
			result["error"] = emailErr.Error()

			// Log failed attempt
			h.emailService.LogNotification(
				tenantID,
				originalLog.Type+"_resend",
				adminUser.Email,
				originalLog.Subject,
				originalLog.Body,
				"failed",
				emailErr.Error(),
			)
		} else {
			// Log successful resend
			h.emailService.LogNotification(
				tenantID,
				originalLog.Type+"_resend",
				adminUser.Email,
				originalLog.Subject,
				originalLog.Body,
				"sent",
				"",
			)

			// Mark as sent for today if not force resend
			if !request.ForceResend {
				today := "2024-08-28"
				h.notificationRepo.MarkNotificationSent(tenantID, originalLog.Type, today)
			}
		}

		results = append(results, result)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "Resend operation completed",
		"results": results,
	})
}

// AdminTriggerNotification manually triggers a notification type for specific tenants
func (h *NotificationHandler) AdminTriggerNotification(c echo.Context) error {
	var request struct {
		Type      string      `json:"type" validate:"required"` // "daily_sales_report" or "low_stock_alert"
		TenantIDs []uuid.UUID `json:"tenant_ids" validate:"required"`
		ForceRun  bool        `json:"force_run"` // Ignores "already sent today" check
	}

	if err := c.Bind(&request); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Invalid request format",
		})
	}

	if err := c.Validate(&request); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error":   "Validation failed",
			"details": err.Error(),
		})
	}

	// Validate notification type
	if request.Type != "daily_sales_report" && request.Type != "low_stock_alert" {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Invalid notification type. Supported types: daily_sales_report, low_stock_alert",
		})
	}

	var results []map[string]interface{}

	// Trigger for each tenant
	for _, tenantID := range request.TenantIDs {
		result := map[string]interface{}{
			"tenant_id": tenantID,
			"status":    "success",
		}

		// Get tenant info and admin email
		tenant, err := h.tenantRepo.GetByID(tenantID)
		if err != nil {
			result["status"] = "error"
			result["error"] = "Tenant not found"
			results = append(results, result)
			continue
		}

		// Get tenant admin email
		var adminUser models.User
		err = h.db.Where("tenant_id = ? AND role = ?", tenantID, "tenant_admin").First(&adminUser).Error
		if err != nil {
			result["status"] = "error"
			result["error"] = "No admin user found for tenant"
			results = append(results, result)
			continue
		}

		result["tenant_name"] = tenant.Name

		// Check if already sent today (unless force run)
		if !request.ForceRun {
			today := time.Now().Format("2006-01-02") // Use current date
			if h.notificationRepo.IsNotificationSentToday(tenantID, request.Type, today) {
				result["status"] = "error"
				result["error"] = fmt.Sprintf("%s already sent today", request.Type)
				results = append(results, result)
				continue
			}
		}

		// Trigger the appropriate notification
		var emailErr error
		switch request.Type {
		case "daily_sales_report":
			emailErr = h.resendDailySalesReport(tenantID, tenant.Name, adminUser.Email)
		case "low_stock_alert":
			emailErr = h.resendLowStockAlert(tenantID, tenant.Name, adminUser.Email)
		}

		if emailErr != nil {
			result["status"] = "error"
			result["error"] = emailErr.Error()
		} else {
			// Mark as sent for today if not force run
			if !request.ForceRun {
				today := time.Now().Format("2006-01-02")
				h.notificationRepo.MarkNotificationSent(tenantID, request.Type, today)
			}
		}

		results = append(results, result)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "Notification trigger completed",
		"results": results,
	})
}

// Helper function to resend daily sales report
func (h *NotificationHandler) resendDailySalesReport(tenantID uuid.UUID, tenantName, tenantEmail string) error {
	// This would need to be implemented to generate current sales data
	// For now, we'll use the email service's existing method
	reportData := map[string]interface{}{
		"Date":          "28/08/2024",
		"TotalOrders":   0,
		"TotalRevenue":  "0,00",
		"TotalProducts": 0,
		"TopProducts":   []interface{}{},
	}

	return h.emailService.SendDailySalesReport(tenantID, tenantName, tenantEmail, reportData)
}

// Helper function to resend low stock alert
func (h *NotificationHandler) resendLowStockAlert(tenantID uuid.UUID, tenantName, tenantEmail string) error {
	// Get current low stock products for this tenant
	var lowStockProducts []models.Product
	err := h.db.Where("tenant_id = ? AND stock_quantity <= low_stock_threshold", tenantID).Find(&lowStockProducts).Error
	if err != nil {
		return fmt.Errorf("failed to fetch low stock products: %w", err)
	}

	// Convert to map format for email template
	lowStockData := make([]map[string]interface{}, len(lowStockProducts))
	for i, product := range lowStockProducts {
		lowStockData[i] = map[string]interface{}{
			"name":          product.Name,
			"current_stock": product.StockQuantity,
			"threshold":     product.LowStockThreshold,
			"sku":           product.SKU,
		}
	}

	// Render email template directly (bypass daily check for forced resend)
	subject := fmt.Sprintf("🔄 [REENVIO] Alerta de Estoque Baixo - %s", tenantName)

	var body string
	if len(lowStockData) > 0 {
		// Create simple HTML template for low stock alert
		body = fmt.Sprintf(`
<html>
<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333;">
<h2 style="color: #e74c3c;">🔄 [REENVIO] Alerta de Estoque Baixo</h2>
<p>Olá, <strong>%s</strong>!</p>
<p>Este é um reenvio da notificação. Os seguintes produtos estão com estoque abaixo do limite:</p>
<table style="width: 100%%; border-collapse: collapse; margin: 20px 0;">
<thead>
<tr style="background-color: #f8f9fa;">
<th style="border: 1px solid #ddd; padding: 12px; text-align: left;">Produto</th>
<th style="border: 1px solid #ddd; padding: 12px; text-align: left;">SKU</th>
<th style="border: 1px solid #ddd; padding: 12px; text-align: left;">Estoque Atual</th>
<th style="border: 1px solid #ddd; padding: 12px; text-align: left;">Limite Mínimo</th>
</tr>
</thead>
<tbody>`, tenantName)

		for _, product := range lowStockData {
			body += fmt.Sprintf(`
<tr>
<td style="border: 1px solid #ddd; padding: 12px;">%v</td>
<td style="border: 1px solid #ddd; padding: 12px;">%v</td>
<td style="border: 1px solid #ddd; padding: 12px; color: #e74c3c; font-weight: bold;">%v</td>
<td style="border: 1px solid #ddd; padding: 12px;">%v</td>
</tr>`, product["name"], product["sku"], product["current_stock"], product["threshold"])
		}

		body += `
</tbody>
</table>
<p>Por favor, considere reabastecer estes produtos.</p>
<p>Atenciosamente,<br>Sistema IAFarma</p>
</body>
</html>`
	} else {
		// If no current low stock products, send informational resend
		body = fmt.Sprintf(`
<html>
<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333;">
<h2 style="color: #28a745;">🔄 [REENVIO] Alerta de Estoque Baixo</h2>
<p>Olá, <strong>%s</strong>!</p>
<p>Este é um reenvio forçado da notificação de estoque baixo.</p>
<p><strong>Status atual:</strong> Nenhum produto com estoque abaixo do limite encontrado no momento.</p>
<p>Se você recebeu esta notificação originalmente com produtos em falta, eles podem ter sido reabastecidos desde então.</p>
<p>Atenciosamente,<br>Sistema IAFarma</p>
</body>
</html>`, tenantName)
	}

	// Send email directly
	err = h.emailService.SendEmail([]string{tenantEmail}, subject, body)
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	// Log resend notification
	h.emailService.LogNotification(tenantID, "low_stock_alert_resend", tenantEmail, subject, body, "sent", "")

	return nil
}
