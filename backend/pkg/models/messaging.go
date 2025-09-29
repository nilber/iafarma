package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Channel represents a messaging channel (WhatsApp, WebChat, etc.)
type Channel struct {
	BaseTenantModel
	Name         string     `gorm:"not null" json:"name" validate:"required"`
	Type         string     `gorm:"not null" json:"type" validate:"required"`    // whatsapp, webchat, etc.
	Session      string     `gorm:"not null" json:"session" validate:"required"` // session identifier for WhatsApp integration
	Status       string     `gorm:"default:'disconnected'" json:"status"`        // disconnected, connecting, connected
	Config       string     `json:"config"`
	QRCode       string     `json:"qr_code,omitempty"`
	QRExpiresAt  *time.Time `json:"qr_expires_at,omitempty"`
	WebhookURL   string     `json:"webhook_url"`
	WebhookToken string     `json:"webhook_token"`
	IsActive     bool       `gorm:"default:true" json:"is_active"`
}

// Conversation represents a conversation with a customer
type Conversation struct {
	BaseTenantModel
	CustomerID      uuid.UUID  `gorm:"type:uuid;not null;constraint:OnDelete:RESTRICT" json:"customer_id"`
	ChannelID       uuid.UUID  `gorm:"type:uuid;not null;constraint:OnDelete:RESTRICT" json:"channel_id"`
	AssignedAgentID *uuid.UUID `gorm:"type:uuid;constraint:OnDelete:SET NULL" json:"assigned_agent_id"`
	Status          string     `gorm:"default:'open'" json:"status"` // open, closed, waiting
	Priority        string     `gorm:"default:'normal'" json:"priority"`
	IsArchived      bool       `gorm:"default:false" json:"is_archived"`
	IsPinned        bool       `gorm:"default:false" json:"is_pinned"`
	AIEnabled       bool       `gorm:"default:true" json:"ai_enabled"`
	LastMessageAt   *time.Time `json:"last_message_at"`
	UnreadCount     int        `gorm:"default:0" json:"unread_count"`

	// Relations
	Customer      *Customer `gorm:"foreignKey:CustomerID" json:"customer,omitempty"`
	Channel       *Channel  `gorm:"foreignKey:ChannelID" json:"channel,omitempty"`
	AssignedAgent *User     `gorm:"foreignKey:AssignedAgentID" json:"assigned_agent,omitempty"`
}

// Message represents a message in a conversation
type Message struct {
	BaseTenantModel
	ConversationID uuid.UUID  `gorm:"type:uuid;not null;index;constraint:OnDelete:RESTRICT" json:"conversation_id"`
	CustomerID     uuid.UUID  `gorm:"type:uuid;not null;constraint:OnDelete:RESTRICT" json:"customer_id"`
	UserID         *uuid.UUID `gorm:"type:uuid;constraint:OnDelete:SET NULL" json:"user_id"` // null for incoming messages
	UserName       string     `gorm:"size:255" json:"user_name"`                             // name of the user who sent the message
	Type           string     `gorm:"not null;default:'text'" json:"type"`                   // text, image, audio, video, document
	Content        string     `json:"content"`
	Direction      string     `gorm:"not null" json:"direction"`        // in, out
	Status         string     `gorm:"default:'sent'" json:"status"`     // sent, delivered, read, failed
	Source         string     `gorm:"default:'whatsapp'" json:"source"` // whatsapp, chat
	ExternalID     string     `gorm:"index" json:"external_id"`
	MediaURL       string     `json:"media_url,omitempty"`
	MediaType      string     `json:"media_type,omitempty"`
	Filename       string     `json:"filename,omitempty"`
	IsRead         bool       `gorm:"default:false" json:"is_read"`
	IsNote         bool       `gorm:"default:false" json:"is_note"` // true for internal notes, false for regular messages
	DeliveredAt    *time.Time `json:"delivered_at"`
	ReadAt         *time.Time `json:"read_at"`
	WebhookID      string     `gorm:"index" json:"webhook_id"`
	Metadata       string     `json:"metadata"`
	ReplyToID      *uuid.UUID `gorm:"type:uuid;constraint:OnDelete:SET NULL" json:"reply_to_id"`
	ForwardedFrom  string     `json:"forwarded_from"`
	RoutineID      *uuid.UUID `gorm:"type:uuid;constraint:OnDelete:SET NULL" json:"routine_id"` // reference to routine that generated this message

	// Relations
	Conversation *Conversation `gorm:"foreignKey:ConversationID" json:"conversation,omitempty"`
	Customer     *Customer     `gorm:"foreignKey:CustomerID" json:"customer,omitempty"`
}

// MessageMedia represents media attachments
type MessageMedia struct {
	BaseTenantModel
	MessageID    uuid.UUID `gorm:"type:uuid;not null;constraint:OnDelete:RESTRICT" json:"message_id"`
	Type         string    `gorm:"not null" json:"type"` // image, audio, video, document
	FileName     string    `json:"file_name"`
	MimeType     string    `json:"mime_type"`
	Size         int64     `json:"size"`
	S3Key        string    `gorm:"not null" json:"s3_key"`
	URL          string    `json:"url"`
	ThumbnailS3  string    `json:"thumbnail_s3"`
	ThumbnailURL string    `json:"thumbnail_url"`
	Duration     *int      `json:"duration"` // for audio/video in seconds
	Width        *int      `json:"width"`    // for images/video
	Height       *int      `json:"height"`   // for images/video
}

// Tag represents a tag for conversations
type Tag struct {
	BaseTenantModel
	Name        string `gorm:"not null" json:"name" validate:"required"`
	Color       string `gorm:"default:'#6B7280'" json:"color"`
	Description string `json:"description"`
}

// ConversationTag links conversations to tags
type ConversationTag struct {
	BaseTenantModel
	ConversationID string `gorm:"not null" json:"conversation_id"`
	TagID          string `gorm:"not null" json:"tag_id"`
}

// QuickReply represents a quick reply template
type QuickReply struct {
	BaseTenantModel
	Title      string `gorm:"not null" json:"title" validate:"required"`
	Content    string `gorm:"not null" json:"content" validate:"required"`
	Category   string `json:"category"`
	IsActive   bool   `gorm:"default:true" json:"is_active"`
	UsageCount int    `gorm:"default:0" json:"usage_count"`
}

// MessageTemplate represents a user's personal message template with variables
type MessageTemplate struct {
	BaseTenantModel
	UserID      uuid.UUID `gorm:"type:uuid;not null;index;constraint:OnDelete:RESTRICT" json:"user_id"`
	Title       string    `gorm:"not null" json:"title" validate:"required"`
	Content     string    `gorm:"not null;type:text" json:"content" validate:"required"`
	Variables   string    `gorm:"type:text" json:"variables"` // JSON array of variable names
	Category    string    `json:"category"`
	IsActive    bool      `gorm:"default:true" json:"is_active"`
	UsageCount  int       `gorm:"default:0" json:"usage_count"`
	Description string    `json:"description"`

	// Relations
	User *User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

// SLAPolicy represents SLA policies for conversations
type SLAPolicy struct {
	BaseTenantModel
	Name               string        `gorm:"not null" json:"name" validate:"required"`
	FirstResponseTime  time.Duration `json:"first_response_time"` // in minutes
	ResolutionTime     time.Duration `json:"resolution_time"`     // in minutes
	BusinessHoursOnly  bool          `gorm:"default:false" json:"business_hours_only"`
	BusinessHoursStart string        `gorm:"default:'09:00'" json:"business_hours_start"`
	BusinessHoursEnd   string        `gorm:"default:'18:00'" json:"business_hours_end"`
	BusinessDays       string        `gorm:"default:'1,2,3,4,5'" json:"business_days"` // 1=Monday, 7=Sunday
	IsActive           bool          `gorm:"default:true" json:"is_active"`
	Priority           string        `gorm:"default:'normal'" json:"priority"`
}

// AgentAssignment represents agent assignments and queues
type AgentAssignment struct {
	BaseTenantModel
	UserID        uuid.UUID `gorm:"type:uuid;not null;constraint:OnDelete:RESTRICT" json:"user_id"`
	ChannelID     uuid.UUID `gorm:"type:uuid;not null;constraint:OnDelete:RESTRICT" json:"channel_id"`
	MaxConcurrent int       `gorm:"default:5" json:"max_concurrent"`
	IsActive      bool      `gorm:"default:true" json:"is_active"`
	CurrentLoad   int       `gorm:"default:0" json:"current_load"`
}

// Alert represents an alert configuration for a channel
type Alert struct {
	BaseTenantModel
	Name        string    `gorm:"not null" json:"name" validate:"required"`
	Description string    `json:"description"`
	ChannelID   uuid.UUID `gorm:"type:uuid;not null;constraint:OnDelete:RESTRICT" json:"channel_id"`
	GroupName   string    `gorm:"not null" json:"group_name" validate:"required"` // Nome do grupo do WhatsApp no ZapPlus
	GroupID     string    `json:"group_id"`                                       // ID do grupo retornado pela API
	Phones      string    `json:"phones"`                                         // Lista de telefones separados por vírgula
	TriggerOn   string    `gorm:"default:'order_created'" json:"trigger_on"`      // order_created, message_received, etc.
	IsActive    bool      `gorm:"default:true" json:"is_active"`

	// Relations
	Channel *Channel `gorm:"foreignKey:ChannelID" json:"channel,omitempty"`
}

// ProductReference stores product info with sequential number for conversation memory
type ProductReference struct {
	SequentialID int    `json:"sequential_id"`
	ProductID    string `json:"product_id"`
	Name         string `json:"name"`
	Price        string `json:"price"`
	SalePrice    string `json:"sale_price"`
	Description  string `json:"description"`
}

// ConversationHistoryItem represents a single message in conversation history
type ConversationHistoryItem struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// JSONB custom types for PostgreSQL
type ProductReferenceList []ProductReference
type ConversationHistoryList []ConversationHistoryItem

// Implement driver.Valuer interface for JSONB
func (p ProductReferenceList) Value() (driver.Value, error) {
	return json.Marshal(p)
}

func (p *ProductReferenceList) Scan(value interface{}) error {
	if value == nil {
		*p = ProductReferenceList{}
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}

	return json.Unmarshal(bytes, p)
}

func (c ConversationHistoryList) Value() (driver.Value, error) {
	return json.Marshal(c)
}

func (c *ConversationHistoryList) Scan(value interface{}) error {
	if value == nil {
		*c = ConversationHistoryList{}
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}

	return json.Unmarshal(bytes, c)
}

// ConversationMemory represents persistent conversation memory
type ConversationMemory struct {
	ID                  uuid.UUID               `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	TenantID            uuid.UUID               `gorm:"type:uuid;not null;uniqueIndex:idx_conversation_memory_tenant_phone;constraint:OnDelete:RESTRICT" json:"tenant_id"`
	CustomerPhone       string                  `gorm:"size:50;not null;uniqueIndex:idx_conversation_memory_tenant_phone" json:"customer_phone"`
	ProductList         ProductReferenceList    `gorm:"type:jsonb;default:'[]'" json:"product_list"`
	ConversationHistory ConversationHistoryList `gorm:"type:jsonb;default:'[]'" json:"conversation_history"`
	SequentialNumber    int                     `gorm:"default:0" json:"sequential_number"`
	CreatedAt           time.Time               `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt           time.Time               `gorm:"not null;default:CURRENT_TIMESTAMP" json:"updated_at"`
}

// AIResponse represents a response from AI services
type AIResponse struct {
	Message string `json:"message"`
	Type    string `json:"type"` // sales
}
