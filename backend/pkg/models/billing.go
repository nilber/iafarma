package models

import (
	"time"

	"github.com/google/uuid"
)

// Plan represents subscription plans for tenants
type Plan struct {
	BaseModel
	Name          string  `gorm:"not null;unique" json:"name" validate:"required"`
	Description   string  `gorm:"type:text" json:"description"`
	Price         float64 `gorm:"type:decimal(10,2);not null" json:"price" validate:"required,gte=0"`
	Currency      string  `gorm:"type:varchar(3);not null;default:'BRL'" json:"currency"`
	BillingPeriod string  `gorm:"type:varchar(50);not null;default:'monthly'" json:"billing_period"`

	// Resource Limits
	MaxConversations    int `gorm:"not null" json:"max_conversations" validate:"required,gt=0"`      // Fixed limit
	MaxMessagesPerMonth int `gorm:"not null" json:"max_messages_per_month" validate:"required,gt=0"` // Monthly limit
	MaxProducts         int `gorm:"not null" json:"max_products" validate:"required,gt=0"`           // Fixed limit
	MaxChannels         int `gorm:"not null" json:"max_channels" validate:"required,gt=0"`           // Fixed limit
	MaxCreditsPerMonth  int `gorm:"not null" json:"max_credits_per_month" validate:"required,gt=0"`  // Monthly limit for AI credits

	IsActive  bool   `gorm:"default:true" json:"is_active"`
	IsDefault bool   `gorm:"default:false" json:"is_default"`
	Features  string `gorm:"type:text" json:"features"`   // JSON string with additional features
	StripeURL string `gorm:"type:text" json:"stripe_url"` // Stripe payment link URL
}

// TenantUsage tracks monthly usage for each tenant
type TenantUsage struct {
	BaseModel
	TenantID uuid.UUID `gorm:"type:uuid;not null;index;constraint:OnDelete:RESTRICT" json:"tenant_id"`
	PlanID   uuid.UUID `gorm:"type:uuid;not null;index;constraint:OnDelete:RESTRICT" json:"plan_id"`

	// Usage counters for current billing cycle
	MessagesUsed int `gorm:"default:0" json:"messages_used"`
	CreditsUsed  int `gorm:"default:0" json:"credits_used"`

	// Billing cycle tracking
	BillingCycleStart time.Time `gorm:"not null" json:"billing_cycle_start"`
	BillingCycleEnd   time.Time `gorm:"not null" json:"billing_cycle_end"`

	// Current counts (fixed resources)
	ConversationsCount int `gorm:"default:0" json:"conversations_count"`
	ProductsCount      int `gorm:"default:0" json:"products_count"`
	ChannelsCount      int `gorm:"default:0" json:"channels_count"`

	// Relationships
	Tenant Tenant `gorm:"foreignKey:TenantID" json:"-"`
	Plan   Plan   `gorm:"foreignKey:PlanID" json:"plan"`
}

// TenantPlanHistory tracks plan changes for tenants
type TenantPlanHistory struct {
	BaseModel
	TenantID  uuid.UUID  `gorm:"type:uuid;not null;index;constraint:OnDelete:RESTRICT" json:"tenant_id"`
	OldPlanID *uuid.UUID `gorm:"type:uuid;index;constraint:OnDelete:SET NULL" json:"old_plan_id"`
	NewPlanID uuid.UUID  `gorm:"type:uuid;not null;index;constraint:OnDelete:RESTRICT" json:"new_plan_id"`

	ChangeReason    string    `gorm:"type:text" json:"change_reason"`
	ChangedByUserID uuid.UUID `gorm:"type:uuid;not null;constraint:OnDelete:RESTRICT" json:"changed_by_user_id"`
	EffectiveDate   time.Time `gorm:"not null" json:"effective_date"`

	// Relationships
	Tenant        Tenant `gorm:"foreignKey:TenantID" json:"-"`
	OldPlan       *Plan  `gorm:"foreignKey:OldPlanID" json:"old_plan,omitempty"`
	NewPlan       Plan   `gorm:"foreignKey:NewPlanID" json:"new_plan"`
	ChangedByUser User   `gorm:"foreignKey:ChangedByUserID" json:"changed_by_user"`
}

// UsageAlert represents alerts for usage limits
type UsageAlert struct {
	BaseModel
	TenantID            uuid.UUID `gorm:"type:uuid;not null;index;constraint:OnDelete:RESTRICT" json:"tenant_id"`
	AlertType           string    `gorm:"not null" json:"alert_type"`           // messages, credits, conversations, products, channels
	ThresholdPercentage int       `gorm:"not null" json:"threshold_percentage"` // 80, 90, 100

	CurrentUsage int `gorm:"not null" json:"current_usage"`
	MaxAllowed   int `gorm:"not null" json:"max_allowed"`

	AlertSent bool       `gorm:"default:false" json:"alert_sent"`
	SentAt    *time.Time `json:"sent_at"`

	// Relationships
	Tenant Tenant `gorm:"foreignKey:TenantID" json:"-"`
}

// UsageLimitResult representa o resultado de verificação de limites de uso
type UsageLimitResult struct {
	Allowed         bool   `json:"allowed"`
	RemainingQuota  int    `json:"remaining_quota"`
	TotalLimit      int    `json:"total_limit"`
	CurrentUsage    int    `json:"current_usage"`
	ResourceType    string `json:"resource_type"`
	Message         string `json:"message"`
	RequiresUpgrade bool   `json:"requires_upgrade"`
}
