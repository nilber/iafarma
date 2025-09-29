package handlers

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"strconv"
	"time"

	"iafarma/internal/ai"
	"iafarma/internal/repo"
	"iafarma/pkg/models"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

// TenantHandler handles tenant management
type TenantHandler struct {
	tenantRepo      *repo.TenantRepository
	db              *gorm.DB
	settingsService *ai.TenantSettingsService
}

// NewTenantHandler creates a new tenant handler
func NewTenantHandler(tenantRepo *repo.TenantRepository, db *gorm.DB) *TenantHandler {
	return &TenantHandler{
		tenantRepo:      tenantRepo,
		db:              db,
		settingsService: ai.NewTenantSettingsService(db),
	}
}

// List godoc
// @Summary List tenants
// @Description Get list of tenants with pagination
// @Tags tenants
// @Accept json
// @Produce json
// @Param limit query int false "Limit" default(20)
// @Param offset query int false "Offset" default(0)
// @Success 200 {object} models.TenantListResponse
// @Failure 500 {object} map[string]string
// @Router /admin/tenants [get]
// @Security BearerAuth
func (h *TenantHandler) List(c echo.Context) error {
	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	offset, _ := strconv.Atoi(c.QueryParam("offset"))

	if limit <= 0 {
		limit = 20
	}

	result, err := h.tenantRepo.List(limit, offset)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, result)
}

// GetByID godoc
// @Summary Get tenant by ID
// @Description Get a specific tenant by ID
// @Tags tenants
// @Accept json
// @Produce json
// @Param id path string true "Tenant ID"
// @Success 200 {object} models.Tenant
// @Failure 404 {object} map[string]string
// @Router /admin/tenants/{id} [get]
// @Security BearerAuth
func (h *TenantHandler) GetByID(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid ID format"})
	}

	tenant, err := h.tenantRepo.GetByID(id)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Tenant not found"})
	}

	return c.JSON(http.StatusOK, tenant)
}

// TenantStats represents tenant statistics
type TenantStats struct {
	TotalTenants       int64 `json:"total_tenants"`
	ActiveTenants      int64 `json:"active_tenants"`
	PaidPlanTenants    int64 `json:"paid_plan_tenants"`
	TotalConversations int64 `json:"total_conversations"`
}

// GetStats godoc
// @Summary Get tenant statistics
// @Description Get aggregated statistics for all tenants
// @Tags tenants
// @Accept json
// @Produce json
// @Success 200 {object} TenantStats
// @Failure 500 {object} map[string]string
// @Router /admin/tenants/stats [get]
// @Security BearerAuth
func (h *TenantHandler) GetStats(c echo.Context) error {
	var stats TenantStats

	// Get total tenants count
	if err := h.db.Model(&models.Tenant{}).Count(&stats.TotalTenants).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to get total tenants count"})
	}

	// Get active tenants count
	if err := h.db.Model(&models.Tenant{}).Where("status = ?", "active").Count(&stats.ActiveTenants).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to get active tenants count"})
	}

	// Get paid plan tenants count (assuming non-free plans are paid)
	if err := h.db.Table("tenants").
		Joins("LEFT JOIN plans ON tenants.plan_id = plans.id").
		Where("plans.price > 0 OR (tenants.plan IS NOT NULL AND tenants.plan != 'free')").
		Count(&stats.PaidPlanTenants).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to get paid plan tenants count"})
	}

	// Get total conversations capacity (sum of max_conversations from all tenant plans)
	var totalConversations sql.NullInt64
	if err := h.db.Table("tenants").
		Joins("LEFT JOIN plans ON tenants.plan_id = plans.id").
		Select("COALESCE(SUM(plans.max_conversations), 0)").
		Scan(&totalConversations).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to get total conversations"})
	}
	stats.TotalConversations = totalConversations.Int64

	return c.JSON(http.StatusOK, stats)
}

// Create godoc
// @Summary Create tenant
// @Description Create a new tenant
// @Tags tenants
// @Accept json
// @Produce json
// @Param tenant body models.Tenant true "Tenant data"
// @Success 201 {object} models.Tenant
// @Failure 400 {object} map[string]string
// @Router /admin/tenants [post]
// @Security BearerAuth
func (h *TenantHandler) Create(c echo.Context) error {
	var tenant models.Tenant
	if err := c.Bind(&tenant); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request"})
	}

	// Validate TAG for public stores
	if tenant.IsPublicStore {
		if tenant.Tag == nil || *tenant.Tag == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "TAG é obrigatório para lojas públicas"})
		}

		// Check if TAG already exists
		existingTenant, _ := h.tenantRepo.GetByTag(*tenant.Tag)
		if existingTenant != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Esta TAG já está sendo utilizada por outra loja"})
		}
	}

	if err := c.Validate(&tenant); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	if err := h.tenantRepo.Create(&tenant); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	// Create default tenant settings
	if err := h.settingsService.CreateDefaultSettings(context.Background(), tenant.ID); err != nil {
		log.Printf("Warning: Failed to create default settings for tenant %s: %v", tenant.ID, err)
		// Don't fail the tenant creation if default settings fail
	}

	return c.JSON(http.StatusCreated, tenant)
}

// Update godoc
// @Summary Update tenant
// @Description Update an existing tenant
// @Tags tenants
// @Accept json
// @Produce json
// @Param id path string true "Tenant ID"
// @Param tenant body models.Tenant true "Tenant data"
// @Success 200 {object} models.Tenant
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /admin/tenants/{id} [put]
// @Security BearerAuth
func (h *TenantHandler) Update(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid ID format"})
	}

	// Check if tenant exists
	existingTenant, err := h.tenantRepo.GetByID(id)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Tenant not found"})
	}

	// Define the update request structure
	type UpdateRequest struct {
		Name                        string     `json:"name"`
		Domain                      string     `json:"domain"`
		Plan                        string     `json:"plan"`    // Keep for backwards compatibility
		PlanID                      *uuid.UUID `json:"plan_id"` // New field using UUID
		Status                      string     `json:"status"`
		BusinessType                string     `json:"business_type"`
		BusinessCategory            string     `json:"business_category"`              // Categoria do negócio (Farmacia, Hamburgeria, etc.)
		CostPerMessage              *int       `json:"cost_per_message"`               // Custo em créditos por mensagem IA
		EnableAIPromptCustomization *bool      `json:"enable_ai_prompt_customization"` // Permite customização de prompts IA
		IsPublicStore               *bool      `json:"is_public_store"`                // Permite acesso público ao catálogo da loja
		Tag                         *string    `json:"tag"`                            // TAG única para lojas públicas
		About                       string     `json:"about"`
		// MaxUsers and MaxMessages removed - now come from plan relationship
	}

	var updateData UpdateRequest
	if err := c.Bind(&updateData); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request"})
	}

	// Update only provided fields (non-zero values)
	if updateData.Name != "" {
		existingTenant.Name = updateData.Name
	}
	if updateData.Domain != "" {
		existingTenant.Domain = updateData.Domain
	}
	if updateData.Plan != "" {
		existingTenant.Plan = updateData.Plan
	}
	if updateData.PlanID != nil {
		existingTenant.PlanID = updateData.PlanID
	}
	if updateData.Status != "" {
		existingTenant.Status = updateData.Status
	}
	if updateData.BusinessType != "" {
		existingTenant.BusinessType = updateData.BusinessType
	}
	if updateData.BusinessCategory != "" {
		existingTenant.BusinessCategory = updateData.BusinessCategory
	}
	if updateData.CostPerMessage != nil {
		existingTenant.CostPerMessage = *updateData.CostPerMessage
	}
	if updateData.EnableAIPromptCustomization != nil {
		existingTenant.EnableAIPromptCustomization = *updateData.EnableAIPromptCustomization
	}
	if updateData.IsPublicStore != nil {
		existingTenant.IsPublicStore = *updateData.IsPublicStore
	}
	if updateData.Tag != nil {
		existingTenant.Tag = updateData.Tag
	}
	if updateData.About != "" {
		existingTenant.About = updateData.About
	}

	// Validate TAG for public stores
	if existingTenant.IsPublicStore {
		if existingTenant.Tag == nil || *existingTenant.Tag == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "TAG é obrigatório para lojas públicas"})
		}

		// Check if TAG already exists (but not for the current tenant)
		existingByTag, _ := h.tenantRepo.GetByTag(*existingTenant.Tag)
		if existingByTag != nil && existingByTag.ID != existingTenant.ID {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Esta TAG já está sendo utilizada por outra loja"})
		}
	}

	if err := h.tenantRepo.Update(existingTenant); err != nil {
		log.Printf("❌ Error updating tenant: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	log.Printf("✅ Tenant updated successfully: %+v", existingTenant)
	return c.JSON(http.StatusOK, existingTenant)
}

// Delete godoc
// @Summary Delete tenant
// @Description Delete a tenant and all its associated data (system_admin only)
// @Tags tenants
// @Accept json
// @Produce json
// @Param id path string true "Tenant ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /admin/tenants/{id} [delete]
// @Security BearerAuth
func (h *TenantHandler) Delete(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid ID format"})
	}

	// Check if tenant exists
	_, err = h.tenantRepo.GetByID(id)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Tenant not found"})
	}

	// Delete the tenant (should cascade delete all related data)
	if err := h.tenantRepo.Delete(id); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "Tenant deleted successfully"})
}

// UpdateProfile godoc
// @Summary Update tenant profile
// @Description Update the current tenant's profile (self-update)
// @Tags tenants
// @Accept json
// @Produce json
// @Param tenant body map[string]interface{} true "Tenant profile data"
// @Success 200 {object} models.Tenant
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /tenant/profile [put]
// @Security BearerAuth
func (h *TenantHandler) UpdateProfile(c echo.Context) error {
	tenantID, ok := c.Get("tenant_id").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Tenant ID required"})
	}

	// Check if tenant exists
	existingTenant, err := h.tenantRepo.GetByID(tenantID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Tenant not found"})
	}

	// Use a map for partial updates
	var updateData map[string]interface{}
	if err := c.Bind(&updateData); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request"})
	}

	// Update only allowed fields for self-update
	if name, exists := updateData["name"]; exists {
		existingTenant.Name = name.(string)
	}
	if about, exists := updateData["about"]; exists {
		existingTenant.About = about.(string)
	}
	// Store information fields
	if storePhone, exists := updateData["store_phone"]; exists {
		existingTenant.StorePhone = storePhone.(string)
	}
	if storeStreet, exists := updateData["store_street"]; exists {
		existingTenant.StoreStreet = storeStreet.(string)
	}
	if storeNumber, exists := updateData["store_number"]; exists {
		existingTenant.StoreNumber = storeNumber.(string)
	}
	if storeComplement, exists := updateData["store_complement"]; exists {
		existingTenant.StoreComplement = storeComplement.(string)
	}
	if storeNeighborhood, exists := updateData["store_neighborhood"]; exists {
		existingTenant.StoreNeighborhood = storeNeighborhood.(string)
	}
	if storeCity, exists := updateData["store_city"]; exists {
		existingTenant.StoreCity = storeCity.(string)
	}
	if storeState, exists := updateData["store_state"]; exists {
		existingTenant.StoreState = storeState.(string)
	}
	if storeZipCode, exists := updateData["store_zip_code"]; exists {
		existingTenant.StoreZipCode = storeZipCode.(string)
	}
	if businessCategory, exists := updateData["business_category"]; exists {
		existingTenant.BusinessCategory = businessCategory.(string)
	}

	if err := h.tenantRepo.Update(existingTenant); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, existingTenant)
}

// GetProfile godoc
// @Summary Get tenant profile
// @Description Get the current tenant's profile
// @Tags tenants
// @Accept json
// @Produce json
// @Success 200 {object} models.Tenant
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /tenant/profile [get]
// @Security BearerAuth
func (h *TenantHandler) GetProfile(c echo.Context) error {
	tenantID, ok := c.Get("tenant_id").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Tenant ID required"})
	}

	tenant, err := h.tenantRepo.GetByID(tenantID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Tenant not found"})
	}

	return c.JSON(http.StatusOK, tenant)
}

// GetSystemStats godoc
// @Summary Get system statistics
// @Description Get global system statistics for admin dashboard
// @Tags admin
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]string
// @Router /admin/stats [get]
// @Security BearerAuth
func (h *TenantHandler) GetSystemStats(c echo.Context) error {
	db := h.tenantRepo.GetDB()

	var stats map[string]interface{}

	// Get total tenants
	var totalTenants int64
	if err := db.Model(&models.Tenant{}).Count(&totalTenants).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to get tenant count: " + err.Error(),
		})
	}

	// For now, assume all tenants are active (we can add an "active" field later)
	activeTenants := totalTenants

	// Get total users across all tenants
	var totalUsers int64
	if err := db.Model(&models.User{}).Count(&totalUsers).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to get user count: " + err.Error(),
		})
	}

	// For now, assume all users are active (we can add an "active" field later)
	activeUsers := totalUsers

	// Get total messages if the table exists
	var totalMessages int64
	if err := db.Raw("SELECT COUNT(*) FROM messages").Scan(&totalMessages).Error; err != nil {
		// If messages table doesn't exist or error, set to 0
		totalMessages = 0
	}

	// Get total conversations if the table exists
	var totalConversations int64
	if err := db.Raw("SELECT COUNT(*) FROM conversations").Scan(&totalConversations).Error; err != nil {
		// If conversations table doesn't exist or error, set to 0
		totalConversations = 0
	}

	// Get tenants by plan using Plan table relationship
	var tenantsByPlanResults []struct {
		PlanName string
		Count    int64
	}

	if err := db.Table("tenants").
		Joins("LEFT JOIN plans ON tenants.plan_id = plans.id").
		Select("COALESCE(plans.name, 'Sem Plano') as plan_name, COUNT(*) as count").
		Group("plans.name").
		Scan(&tenantsByPlanResults).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to get tenants by plan: " + err.Error(),
		})
	}

	// Convert results to map
	tenantsByPlan := make(map[string]int)
	for _, result := range tenantsByPlanResults {
		tenantsByPlan[result.PlanName] = int(result.Count)
	}

	// Get recent activity (this month)
	recentActivity := map[string]int{
		"new_tenants_this_month": 0,
		"new_users_this_month":   0,
		"messages_this_month":    0,
	}

	// Try to get tenants created this month
	var newTenantsThisMonth int64
	if err := db.Model(&models.Tenant{}).
		Where("DATE_TRUNC('month', created_at) = DATE_TRUNC('month', CURRENT_DATE)").
		Count(&newTenantsThisMonth).Error; err == nil {
		recentActivity["new_tenants_this_month"] = int(newTenantsThisMonth)
	}

	// Try to get users created this month
	var newUsersThisMonth int64
	if err := db.Model(&models.User{}).
		Where("DATE_TRUNC('month', created_at) = DATE_TRUNC('month', CURRENT_DATE)").
		Count(&newUsersThisMonth).Error; err == nil {
		recentActivity["new_users_this_month"] = int(newUsersThisMonth)
	}

	// Try to get messages from this month
	var messagesThisMonth int64
	if err := db.Raw(`
		SELECT COUNT(*) 
		FROM messages 
		WHERE DATE_TRUNC('month', created_at) = DATE_TRUNC('month', CURRENT_DATE)
	`).Scan(&messagesThisMonth).Error; err == nil {
		recentActivity["messages_this_month"] = int(messagesThisMonth)
	}

	stats = map[string]interface{}{
		"total_tenants":       totalTenants,
		"active_tenants":      activeTenants,
		"total_users":         totalUsers,
		"active_users":        activeUsers,
		"total_messages":      totalMessages,
		"total_conversations": totalConversations,
		"tenants_by_plan":     tenantsByPlan,
		"recent_activity":     recentActivity,
	}

	return c.JSON(http.StatusOK, stats)
}

// GetTenantStats godoc
// @Summary Get tenant statistics
// @Description Get conversation and message statistics for a specific tenant
// @Tags tenants
// @Accept json
// @Produce json
// @Param id path string true "Tenant ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /admin/tenants/{id}/stats [get]
// @Security BearerAuth
func (h *TenantHandler) GetTenantStats(c echo.Context) error {
	tenantID := c.Param("id")
	if tenantID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Tenant ID is required"})
	}

	tenantUUID, err := uuid.Parse(tenantID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid tenant ID format"})
	}

	// Verify tenant exists
	var tenant models.Tenant
	if err := h.tenantRepo.GetDB().First(&tenant, "id = ?", tenantUUID).Error; err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Tenant not found"})
	}

	// Count total conversations
	var totalConversations int64
	if err := h.tenantRepo.GetDB().Model(&models.Conversation{}).Where("tenant_id = ?", tenantUUID).Count(&totalConversations).Error; err != nil {
		log.Printf("Error counting conversations: %v", err)
		totalConversations = 0
	}

	// Count active conversations (not archived)
	var activeConversations int64
	if err := h.tenantRepo.GetDB().Model(&models.Conversation{}).Where("tenant_id = ? AND is_archived = false", tenantUUID).Count(&activeConversations).Error; err != nil {
		log.Printf("Error counting active conversations: %v", err)
		activeConversations = 0
	}

	// Count total messages
	var totalMessages int64
	if err := h.tenantRepo.GetDB().Model(&models.Message{}).Where("tenant_id = ?", tenantUUID).Count(&totalMessages).Error; err != nil {
		log.Printf("Error counting total messages: %v", err)
		totalMessages = 0
	}

	// Count messages from current month
	now := time.Now()
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	var messagesThisMonth int64
	if err := h.tenantRepo.GetDB().Model(&models.Message{}).Where("tenant_id = ? AND created_at >= ?", tenantUUID, startOfMonth).Count(&messagesThisMonth).Error; err != nil {
		log.Printf("Error counting messages this month: %v", err)
		messagesThisMonth = 0
	}

	stats := map[string]interface{}{
		"total_conversations":    totalConversations,
		"active_conversations":   activeConversations,
		"total_messages":         totalMessages,
		"messages_current_month": messagesThisMonth,
	}

	return c.JSON(http.StatusOK, stats)
}
