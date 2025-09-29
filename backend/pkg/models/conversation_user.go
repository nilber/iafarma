package models

import (
	"time"

	"github.com/google/uuid"
)

// ConversationUser represents a user assigned to a conversation with WhatsApp group proxy
type ConversationUser struct {
	ID              uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	TenantID        uuid.UUID `gorm:"type:uuid;not null;index;constraint:OnDelete:RESTRICT" json:"tenant_id"`
	ConversationID  uuid.UUID `gorm:"type:uuid;not null;index;constraint:OnDelete:RESTRICT" json:"conversation_id"`
	UserID          uuid.UUID `gorm:"type:uuid;not null;index;constraint:OnDelete:RESTRICT" json:"user_id"`
	WhatsappGroupID string    `gorm:"type:varchar(255);not null;uniqueIndex" json:"whatsapp_group_id"`
	CreatedAt       time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt       time.Time `gorm:"autoUpdateTime" json:"updated_at"`

	// Relationships
	Tenant       Tenant       `gorm:"foreignKey:TenantID;constraint:OnDelete:RESTRICT" json:"tenant,omitempty"`
	Conversation Conversation `gorm:"foreignKey:ConversationID;constraint:OnDelete:RESTRICT" json:"conversation,omitempty"`
	User         User         `gorm:"foreignKey:UserID;constraint:OnDelete:RESTRICT" json:"user,omitempty"`
}

// TableName returns the table name for ConversationUser
func (ConversationUser) TableName() string {
	return "conversation_users"
}
