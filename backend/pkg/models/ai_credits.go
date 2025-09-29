package models

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// AICredits represents the AI credits for a tenant
type AICredits struct {
	BaseModel
	TenantID         uuid.UUID  `gorm:"type:uuid;index;not null;unique;constraint:OnDelete:RESTRICT" json:"tenant_id"`
	TotalCredits     int        `gorm:"default:0" json:"total_credits"`
	UsedCredits      int        `gorm:"default:0" json:"used_credits"`
	RemainingCredits int        `gorm:"default:0" json:"remaining_credits"`
	LastUpdatedBy    *uuid.UUID `gorm:"type:uuid;constraint:OnDelete:SET NULL" json:"last_updated_by,omitempty"`

	// Relationships
	Tenant Tenant `gorm:"foreignKey:TenantID" json:"-"`
}

// AICreditTransaction represents a transaction of AI credits
type AICreditTransaction struct {
	BaseModel
	TenantID      uuid.UUID  `gorm:"type:uuid;index;not null;constraint:OnDelete:RESTRICT" json:"tenant_id"`
	UserID        *uuid.UUID `gorm:"type:uuid;constraint:OnDelete:SET NULL" json:"user_id"` // Nullable for automatic transactions
	Type          string     `gorm:"not null" json:"type"`                                  // 'add', 'use', 'refund'
	Amount        int        `gorm:"not null" json:"amount"`
	Description   string     `gorm:"not null" json:"description"`
	RelatedEntity string     `json:"related_entity,omitempty"` // e.g., 'product', 'campaign'
	RelatedID     *uuid.UUID `gorm:"type:uuid;constraint:OnDelete:SET NULL" json:"related_id,omitempty"`

	// Relationships
	Tenant Tenant `gorm:"foreignKey:TenantID" json:"-"`
	User   *User  `gorm:"foreignKey:UserID" json:"-"`
}

// BeforeCreate hook to calculate remaining credits
func (a *AICredits) BeforeCreate(tx *gorm.DB) error {
	// Generate UUID if not set
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}

	// Calculate remaining credits
	a.RemainingCredits = a.TotalCredits - a.UsedCredits
	return nil
}

// BeforeUpdate hook to calculate remaining credits
func (a *AICredits) BeforeUpdate(tx *gorm.DB) error {
	// Calculate remaining credits
	a.RemainingCredits = a.TotalCredits - a.UsedCredits
	return nil
}

// UseCredits decreases the available credits
func (a *AICredits) UseCredits(amount int) bool {
	if a.RemainingCredits >= amount {
		a.UsedCredits += amount
		a.RemainingCredits = a.TotalCredits - a.UsedCredits
		return true
	}
	return false
}

// AddCredits increases the total credits
func (a *AICredits) AddCredits(amount int) {
	a.TotalCredits += amount
	a.RemainingCredits = a.TotalCredits - a.UsedCredits
}
