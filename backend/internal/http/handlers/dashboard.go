package handlers

import (
	"fmt"
	"net/http"

	"iafarma/internal/repo"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

// DashboardHandler handles dashboard-related endpoints
type DashboardHandler struct {
	messageRepo *repo.MessageRepository
	db          *gorm.DB
}

// NewDashboardHandler creates a new dashboard handler
func NewDashboardHandler(messageRepo *repo.MessageRepository, db *gorm.DB) *DashboardHandler {
	return &DashboardHandler{
		messageRepo: messageRepo,
		db:          db,
	}
}

// GetUnreadMessages godoc
// @Summary Get count of unread messages
// @Description Get the total count of unread messages for the authenticated user's tenant
// @Tags dashboard
// @Produce json
// @Success 200 {object} map[string]int64
// @Failure 401 {object} map[string]string
// @Router /dashboard/unread-messages [get]
func (h *DashboardHandler) GetUnreadMessages(c echo.Context) error {
	tenantIDRaw := c.Get("tenant_id")
	// fmt.Printf("DEBUG Dashboard - tenantIDRaw: %v, type: %T\n", tenantIDRaw, tenantIDRaw)

	if tenantIDRaw == nil {
		fmt.Println("DEBUG Dashboard - Tenant ID not found")
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Tenant ID not found"})
	}

	tenantID, ok := tenantIDRaw.(uuid.UUID)
	if !ok {
		fmt.Printf("DEBUG Dashboard - Invalid tenant ID format. Value: %v, Type: %T\n", tenantIDRaw, tenantIDRaw)
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid tenant ID format"})
	}

	// fmt.Printf("DEBUG Dashboard - Using tenant ID: %s\n", tenantID.String())

	count, err := h.messageRepo.GetUnreadCountByTenant(tenantID)
	if err != nil {
		// fmt.Printf("DEBUG Dashboard - Error getting unread count: %v\n", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to get unread messages count"})
	}

	// fmt.Printf("DEBUG Dashboard - Unread count: %d\n", count)
	return c.JSON(http.StatusOK, map[string]int64{"unread_count": count})
}

// DashboardStats represents dashboard statistics
type DashboardStats struct {
	TotalProducts     int64 `json:"total_products"`
	ProductsOnSale    int64 `json:"products_on_sale"`
	TotalCustomers    int64 `json:"total_customers"`
	ActiveCustomers   int64 `json:"active_customers"`
	TotalOrders       int64 `json:"total_orders"`
	PendingOrders     int64 `json:"pending_orders"`
	TotalChannels     int64 `json:"total_channels"`
	ActiveChannels    int64 `json:"active_channels"`
	ConnectedChannels int64 `json:"connected_channels"`
}

// GetStats godoc
// @Summary Get dashboard statistics
// @Description Get comprehensive statistics for the dashboard
// @Tags dashboard
// @Produce json
// @Success 200 {object} DashboardStats
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /dashboard/stats [get]
func (h *DashboardHandler) GetStats(c echo.Context) error {
	tenantIDRaw := c.Get("tenant_id")
	if tenantIDRaw == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Tenant ID not found"})
	}

	tenantID, ok := tenantIDRaw.(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid tenant ID format"})
	}

	stats := DashboardStats{}

	// Get product statistics
	if err := h.db.Raw("SELECT COUNT(*) FROM products WHERE tenant_id = ? AND deleted_at IS NULL", tenantID).Scan(&stats.TotalProducts).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to get product count"})
	}

	if err := h.db.Raw("SELECT COUNT(*) FROM products WHERE tenant_id = ? AND deleted_at IS NULL AND sale_price IS NOT NULL AND sale_price != '' AND sale_price != '0'", tenantID).Scan(&stats.ProductsOnSale).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to get products on sale count"})
	}

	// Get customer statistics
	if err := h.db.Raw("SELECT COUNT(*) FROM customers WHERE tenant_id = ? AND deleted_at IS NULL", tenantID).Scan(&stats.TotalCustomers).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to get customer count"})
	}

	if err := h.db.Raw("SELECT COUNT(*) FROM customers WHERE tenant_id = ? AND deleted_at IS NULL AND is_active = true", tenantID).Scan(&stats.ActiveCustomers).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to get active customer count"})
	}

	// Get order statistics
	if err := h.db.Raw("SELECT COUNT(*) FROM orders WHERE tenant_id = ? AND deleted_at IS NULL", tenantID).Scan(&stats.TotalOrders).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to get order count"})
	}

	if err := h.db.Raw("SELECT COUNT(*) FROM orders WHERE tenant_id = ? AND deleted_at IS NULL AND status IN ('pending', 'processing')", tenantID).Scan(&stats.PendingOrders).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to get pending order count"})
	}

	// Get channel statistics
	if err := h.db.Raw("SELECT COUNT(*) FROM channels WHERE tenant_id = ? AND deleted_at IS NULL", tenantID).Scan(&stats.TotalChannels).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to get channel count"})
	}

	if err := h.db.Raw("SELECT COUNT(*) FROM channels WHERE tenant_id = ? AND deleted_at IS NULL AND is_active = true", tenantID).Scan(&stats.ActiveChannels).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to get active channel count"})
	}

	if err := h.db.Raw("SELECT COUNT(*) FROM channels WHERE tenant_id = ? AND deleted_at IS NULL AND status = 'connected'", tenantID).Scan(&stats.ConnectedChannels).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to get connected channel count"})
	}

	return c.JSON(http.StatusOK, stats)
}

// ConversationCounts represents conversation counts by category for filters
type ConversationCounts struct {
	Novas         int64 `json:"novas"`          // Conversations without assigned agent with unread messages
	EmAtendimento int64 `json:"em_atendimento"` // Conversations with assigned agent with unread messages
	Minhas        int64 `json:"minhas"`         // Conversations assigned to current user with unread messages
	Arquivadas    int64 `json:"arquivadas"`     // Archived conversations with unread messages
}

// GetConversationCounts godoc
// @Summary Get conversation counts by category with unread messages
// @Description Get counts of conversations with unread messages for each filter category (novas, em_atendimento, minhas, arquivadas)
// @Tags dashboard
// @Produce json
// @Success 200 {object} ConversationCounts
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /dashboard/conversation-counts [get]
func (h *DashboardHandler) GetConversationCounts(c echo.Context) error {
	tenantIDRaw := c.Get("tenant_id")
	if tenantIDRaw == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Tenant ID not found"})
	}

	tenantID, ok := tenantIDRaw.(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid tenant ID format"})
	}

	// Get user ID for "minhas" count
	userIDRaw := c.Get("user_id")
	if userIDRaw == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "User ID not found"})
	}

	userID, ok := userIDRaw.(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid user ID format"})
	}

	counts := ConversationCounts{}

	// Novas: conversations without assigned agent, not archived, and with unread messages
	if err := h.db.Raw(`
		SELECT COUNT(*) 
		FROM conversations 
		WHERE tenant_id = ? 
		AND deleted_at IS NULL 
		AND is_archived = false 
		AND assigned_agent_id IS NULL
		AND unread_count > 0
	`, tenantID).Scan(&counts.Novas).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to get novas count"})
	}

	// Em Atendimento: conversations with assigned agent, not archived, and with unread messages
	if err := h.db.Raw(`
		SELECT COUNT(*) 
		FROM conversations 
		WHERE tenant_id = ? 
		AND deleted_at IS NULL 
		AND is_archived = false 
		AND assigned_agent_id IS NOT NULL
		AND unread_count > 0
	`, tenantID).Scan(&counts.EmAtendimento).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to get em_atendimento count"})
	}

	// Minhas: conversations assigned to current user, not archived, and with unread messages
	if err := h.db.Raw(`
		SELECT COUNT(*) 
		FROM conversations 
		WHERE tenant_id = ? 
		AND deleted_at IS NULL 
		AND is_archived = false 
		AND assigned_agent_id = ?
		AND unread_count > 0
	`, tenantID, userID).Scan(&counts.Minhas).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to get minhas count"})
	}

	// Arquivadas: archived conversations with unread messages
	if err := h.db.Raw(`
		SELECT COUNT(*) 
		FROM conversations 
		WHERE tenant_id = ? 
		AND deleted_at IS NULL 
		AND is_archived = true
		AND unread_count > 0
	`, tenantID).Scan(&counts.Arquivadas).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to get arquivadas count"})
	}

	return c.JSON(http.StatusOK, counts)
}
