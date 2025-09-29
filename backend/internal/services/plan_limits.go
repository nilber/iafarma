package services

import (
	"fmt"
	"iafarma/pkg/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// PlanLimitService handles plan limit validations
type PlanLimitService struct {
	db *gorm.DB
}

// NewPlanLimitService creates a new plan limit service
func NewPlanLimitService(db *gorm.DB) *PlanLimitService {
	return &PlanLimitService{db: db}
}

// PlanLimits represents current plan limits for a tenant
type PlanLimits struct {
	MaxProducts      int `json:"max_products"`
	MaxConversations int `json:"max_conversations"`
	MaxChannels      int `json:"max_channels"`
}

// TenantUsage represents current usage for a tenant
type TenantUsage struct {
	ProductCount      int `json:"product_count"`
	ConversationCount int `json:"conversation_count"`
	ChannelCount      int `json:"channel_count"`
}

// PlanLimitError represents a plan limit validation error
type PlanLimitError struct {
	Type     string `json:"type"`
	Current  int    `json:"current"`
	Limit    int    `json:"limit"`
	Required int    `json:"required"`
	Message  string `json:"message"`
}

func (e *PlanLimitError) Error() string {
	return e.Message
}

// GetTenantPlanLimits gets the plan limits for a tenant
func (s *PlanLimitService) GetTenantPlanLimits(tenantID uuid.UUID) (*PlanLimits, error) {
	var tenant models.Tenant
	err := s.db.Preload("PlanInfo").Where("id = ?", tenantID).First(&tenant).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant: %w", err)
	}

	if tenant.PlanInfo == nil {
		return nil, fmt.Errorf("tenant has no plan configured")
	}

	return &PlanLimits{
		MaxProducts:      tenant.PlanInfo.MaxProducts,
		MaxConversations: tenant.PlanInfo.MaxConversations,
		MaxChannels:      tenant.PlanInfo.MaxChannels,
	}, nil
}

// GetTenantUsage gets the current usage for a tenant
func (s *PlanLimitService) GetTenantUsage(tenantID uuid.UUID) (*TenantUsage, error) {
	usage := &TenantUsage{}

	// Count products
	var productCount int64
	err := s.db.Model(&models.Product{}).Where("tenant_id = ?", tenantID).Count(&productCount).Error
	if err != nil {
		return nil, fmt.Errorf("failed to count products: %w", err)
	}
	usage.ProductCount = int(productCount)

	// Count conversations (customers)
	var conversationCount int64
	err = s.db.Model(&models.Customer{}).Where("tenant_id = ?", tenantID).Count(&conversationCount).Error
	if err != nil {
		return nil, fmt.Errorf("failed to count conversations: %w", err)
	}
	usage.ConversationCount = int(conversationCount)

	// Count channels
	var channelCount int64
	err = s.db.Model(&models.Channel{}).Where("tenant_id = ?", tenantID).Count(&channelCount).Error
	if err != nil {
		return nil, fmt.Errorf("failed to count channels: %w", err)
	}
	usage.ChannelCount = int(channelCount)

	return usage, nil
}

// ValidateProductLimit validates if a tenant can add more products
func (s *PlanLimitService) ValidateProductLimit(tenantID uuid.UUID, additionalProducts int) error {
	limits, err := s.GetTenantPlanLimits(tenantID)
	if err != nil {
		return err
	}

	usage, err := s.GetTenantUsage(tenantID)
	if err != nil {
		return err
	}

	if usage.ProductCount+additionalProducts > limits.MaxProducts {
		return &PlanLimitError{
			Type:     "products",
			Current:  usage.ProductCount,
			Limit:    limits.MaxProducts,
			Required: additionalProducts,
			Message: fmt.Sprintf("Limite de produtos atingido. Plano atual permite %d produtos, você já tem %d. Tentando adicionar %d produtos.",
				limits.MaxProducts, usage.ProductCount, additionalProducts),
		}
	}

	return nil
}

// ValidateConversationLimit validates if a tenant can add more conversations
// Returns an error only if it's a hard limit, otherwise returns nil with a warning flag
func (s *PlanLimitService) ValidateConversationLimit(tenantID uuid.UUID) (*PlanLimitError, error) {
	limits, err := s.GetTenantPlanLimits(tenantID)
	if err != nil {
		return nil, err
	}

	usage, err := s.GetTenantUsage(tenantID)
	if err != nil {
		return nil, err
	}

	if usage.ConversationCount >= limits.MaxConversations {
		return &PlanLimitError{
			Type:    "conversations",
			Current: usage.ConversationCount,
			Limit:   limits.MaxConversations,
			Message: fmt.Sprintf("Limite de conversas atingido. Plano atual permite %d conversas, você já tem %d. Considere atualizar seu plano para não perder novos atendimentos.",
				limits.MaxConversations, usage.ConversationCount),
		}, nil
	}

	return nil, nil
}

// GetProductsAvailable returns how many products can still be added
func (s *PlanLimitService) GetProductsAvailable(tenantID uuid.UUID) (int, error) {
	limits, err := s.GetTenantPlanLimits(tenantID)
	if err != nil {
		return 0, err
	}

	usage, err := s.GetTenantUsage(tenantID)
	if err != nil {
		return 0, err
	}

	available := limits.MaxProducts - usage.ProductCount
	if available < 0 {
		available = 0
	}

	return available, nil
}

// ValidateChannelLimit validates if a tenant can add more channels
func (s *PlanLimitService) ValidateChannelLimit(tenantID uuid.UUID, additionalChannels int) error {
	limits, err := s.GetTenantPlanLimits(tenantID)
	if err != nil {
		return err
	}

	usage, err := s.GetTenantUsage(tenantID)
	if err != nil {
		return err
	}

	if usage.ChannelCount+additionalChannels > limits.MaxChannels {
		return &PlanLimitError{
			Type:     "channels",
			Current:  usage.ChannelCount,
			Limit:    limits.MaxChannels,
			Required: additionalChannels,
			Message: fmt.Sprintf("Limite de canais atingido. Plano atual permite %d canais, você já tem %d.",
				limits.MaxChannels, usage.ChannelCount),
		}
	}

	return nil
}

// GetTenantLimitStatus returns comprehensive status of tenant limits
func (s *PlanLimitService) GetTenantLimitStatus(tenantID uuid.UUID) (map[string]interface{}, error) {
	limits, err := s.GetTenantPlanLimits(tenantID)
	if err != nil {
		return nil, err
	}

	usage, err := s.GetTenantUsage(tenantID)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"limits": limits,
		"usage":  usage,
		"available": map[string]int{
			"products":      max(0, limits.MaxProducts-usage.ProductCount),
			"conversations": max(0, limits.MaxConversations-usage.ConversationCount),
			"channels":      max(0, limits.MaxChannels-usage.ChannelCount),
		},
		"warnings": map[string]bool{
			"products_near_limit":      float64(usage.ProductCount)/float64(limits.MaxProducts) >= 0.8,
			"conversations_near_limit": float64(usage.ConversationCount)/float64(limits.MaxConversations) >= 0.8,
		},
	}, nil
}

// Helper function for max
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
