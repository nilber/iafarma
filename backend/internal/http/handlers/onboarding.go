package handlers

import (
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"

	"iafarma/pkg/models"
)

// OnboardingHandler handles onboarding status endpoints
type OnboardingHandler struct {
	db *gorm.DB
}

// NewOnboardingHandler creates a new onboarding handler
func NewOnboardingHandler(db *gorm.DB) *OnboardingHandler {
	return &OnboardingHandler{db: db}
}

// OnboardingStatus represents the onboarding status response
type OnboardingStatus struct {
	IsCompleted     bool             `json:"is_completed"`
	CompletionRate  float64          `json:"completion_rate"`
	Items           []OnboardingItem `json:"items"`
	TenantCreatedAt time.Time        `json:"tenant_created_at"`
}

// OnboardingItem represents a single onboarding checklist item
type OnboardingItem struct {
	ID          string     `json:"id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	IsCompleted bool       `json:"is_completed"`
	ActionURL   string     `json:"action_url"`
	Priority    int        `json:"priority"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}

// GetOnboardingStatus returns the onboarding status for the current tenant
func (h *OnboardingHandler) GetOnboardingStatus(c echo.Context) error {
	// Get tenant ID from JWT context
	tenantID, ok := c.Get("tenant_id").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid tenant"})
	}

	// Get tenant info
	var tenant models.Tenant
	if err := h.db.First(&tenant, tenantID).Error; err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Tenant not found"})
	}

	status := h.checkOnboardingStatus(tenant)
	return c.JSON(http.StatusOK, status)
}

// checkOnboardingStatus checks the completion status of all onboarding items
func (h *OnboardingHandler) checkOnboardingStatus(tenant models.Tenant) OnboardingStatus {
	items := []OnboardingItem{
		h.checkStoreConfiguration(tenant),
		h.checkProducts(tenant),
		h.checkPaymentMethods(tenant),
		h.checkConnectedChannel(tenant),
	}

	// Calculate completion rate
	completedCount := 0
	for _, item := range items {
		if item.IsCompleted {
			completedCount++
		}
	}

	completionRate := float64(completedCount) / float64(len(items)) * 100
	isCompleted := completedCount == len(items)

	return OnboardingStatus{
		IsCompleted:     isCompleted,
		CompletionRate:  completionRate,
		Items:           items,
		TenantCreatedAt: tenant.CreatedAt,
	}
}

// checkStoreConfiguration checks if store information is properly configured
func (h *OnboardingHandler) checkStoreConfiguration(tenant models.Tenant) OnboardingItem {
	item := OnboardingItem{
		ID:          "store_config",
		Title:       "Configurar Dados da Loja",
		Description: "Configure telefone, sobre a loja, endereço e horário de funcionamento",
		ActionURL:   "/settings?tab=store",
		Priority:    1,
		IsCompleted: false,
	}

	// Check if basic store info is configured (not default values)
	// Consider it configured if it was updated more than 1 minute after tenant creation
	tenantCreation := tenant.CreatedAt
	storeUpdateTime := tenant.UpdatedAt

	// Check if essential fields are filled and not default
	hasPhone := tenant.StorePhone != ""
	hasAbout := tenant.About != ""
	hasAddress := tenant.StoreStreet != "" && tenant.StoreCity != "" && tenant.StoreState != ""

	// Check if data was updated after creation (indicating user configuration)
	wasConfiguredByUser := storeUpdateTime.After(tenantCreation.Add(1 * time.Minute))

	if hasPhone && hasAbout && hasAddress && wasConfiguredByUser {
		item.IsCompleted = true
		item.CompletedAt = &storeUpdateTime
	}

	return item
}

// checkProducts checks if at least one product is registered
func (h *OnboardingHandler) checkProducts(tenant models.Tenant) OnboardingItem {
	item := OnboardingItem{
		ID:          "products",
		Title:       "Cadastrar Produtos",
		Description: "Cadastre pelo menos um produto em seu catálogo",
		ActionURL:   "/products",
		Priority:    2,
		IsCompleted: false,
	}

	// Check if there's at least one product (products don't have status column)
	var productCount int64
	err := h.db.Model(&models.Product{}).
		Where("tenant_id = ?", tenant.ID).
		Count(&productCount).Error

	if err == nil && productCount > 0 {
		item.IsCompleted = true

		// Get the most recent product creation time
		var product models.Product
		if err := h.db.Where("tenant_id = ?", tenant.ID).
			Order("created_at DESC").First(&product).Error; err == nil {
			item.CompletedAt = &product.CreatedAt
		}
	}

	return item
}

// checkPaymentMethods checks if at least one payment method is registered
func (h *OnboardingHandler) checkPaymentMethods(tenant models.Tenant) OnboardingItem {
	item := OnboardingItem{
		ID:          "payment_methods",
		Title:       "Cadastrar Métodos de Pagamento",
		Description: "Configure pelo menos um método de pagamento para aceitar pedidos",
		ActionURL:   "/payment-methods",
		Priority:    3,
		IsCompleted: false,
	}

	// Check if there's at least one payment method
	var paymentMethodCount int64
	err := h.db.Model(&models.PaymentMethod{}).
		Where("tenant_id = ? AND is_active = ?", tenant.ID, true).
		Count(&paymentMethodCount).Error

	if err == nil && paymentMethodCount > 0 {
		item.IsCompleted = true

		// Get the most recent payment method creation time
		var paymentMethod models.PaymentMethod
		if err := h.db.Where("tenant_id = ? AND is_active = ?", tenant.ID, true).
			Order("created_at DESC").First(&paymentMethod).Error; err == nil {
			item.CompletedAt = &paymentMethod.CreatedAt
		}
	}

	return item
}

// checkConnectedChannel checks if at least one channel is connected (any type)
func (h *OnboardingHandler) checkConnectedChannel(tenant models.Tenant) OnboardingItem {
	item := OnboardingItem{
		ID:          "channel_connection",
		Title:       "Conectar ao WhatsApp",
		Description: "Pelo menos um canal deve estar conectado para o sistema funcionar",
		ActionURL:   "/whatsapp/connection",
		Priority:    4,
		IsCompleted: false,
	}

	// Check if there's at least one channel connected (any type)
	var channelCount int64
	err := h.db.Model(&models.Channel{}).
		Where("tenant_id = ? AND status = ?", tenant.ID, "connected").
		Count(&channelCount).Error

	if err == nil && channelCount > 0 {
		item.IsCompleted = true

		// Get the most recent connected channel time
		var channel models.Channel
		if err := h.db.Where("tenant_id = ? AND status = ?", tenant.ID, "connected").
			Order("updated_at DESC").First(&channel).Error; err == nil {
			item.CompletedAt = &channel.UpdatedAt
		}
	}

	return item
}

// CompleteOnboardingItem marks a specific onboarding item as completed manually
func (h *OnboardingHandler) CompleteOnboardingItem(c echo.Context) error {
	_, ok := c.Get("tenant_id").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid tenant"})
	}

	itemID := c.Param("item_id")
	if itemID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Item ID is required"})
	}

	// In the future, we could store completion status in a separate table
	// For now, we'll rely on the automatic detection logic

	return c.JSON(http.StatusOK, map[string]string{"message": "Item completion tracked automatically"})
}

// DismissOnboarding allows tenant to dismiss the onboarding modal temporarily
func (h *OnboardingHandler) DismissOnboarding(c echo.Context) error {
	tenantID, ok := c.Get("tenant_id").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid tenant"})
	}

	// Update tenant to mark onboarding as dismissed
	// We could add a dismissed_at field to track this
	now := time.Now()
	err := h.db.Model(&models.Tenant{}).
		Where("id = ?", tenantID).
		Update("updated_at", now).Error

	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to dismiss onboarding"})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "Onboarding dismissed"})
}
