package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// BaseTenantModel is the base model for all tenant-scoped entities
type BaseTenantModel struct {
	ID        uuid.UUID       `gorm:"type:uuid;primaryKey" json:"id"`
	TenantID  uuid.UUID       `gorm:"type:uuid;index;not null;constraint:OnDelete:RESTRICT" json:"tenant_id"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
	DeletedAt *gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty" swaggerignore:"true"`
}

// BaseModel is the base model for system-wide entities
type BaseModel struct {
	ID        uuid.UUID       `gorm:"type:uuid;primaryKey" json:"id"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
	DeletedAt *gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty" swaggerignore:"true"`
}

// BeforeCreate hook to generate UUID if not set
func (b *BaseTenantModel) BeforeCreate(tx *gorm.DB) error {
	if b.ID == uuid.Nil {
		b.ID = uuid.New()
	}
	return nil
}

// BeforeCreate hook to generate UUID if not set
func (b *BaseModel) BeforeCreate(tx *gorm.DB) error {
	if b.ID == uuid.Nil {
		b.ID = uuid.New()
	}
	return nil
}

// Tenant represents a company/organization
type Tenant struct {
	BaseModel
	Name         string     `gorm:"not null" json:"name" validate:"required"`
	Domain       string     `json:"domain"`
	PlanID       *uuid.UUID `gorm:"type:uuid;index" json:"plan_id"` // Reference to Plan table
	Plan         string     `gorm:"default:'free'" json:"plan"`     // Keep for backwards compatibility
	Status       string     `gorm:"default:'active'" json:"status"`
	MaxUsers     int        `gorm:"default:5" json:"max_users"`
	MaxMessages  int        `gorm:"default:1000" json:"max_messages"`
	About        string     `gorm:"type:text" json:"about"`
	BusinessType string     `gorm:"default:'sales';check:business_type IN ('sales')" json:"business_type" validate:"required,oneof=sales"`

	// Store Address Fields
	StoreStreet       string `gorm:"column:store_street" json:"store_street"`
	StoreNumber       string `gorm:"column:store_number" json:"store_number"`
	StoreComplement   string `gorm:"column:store_complement" json:"store_complement"`
	StoreNeighborhood string `gorm:"column:store_neighborhood" json:"store_neighborhood"`
	StoreCity         string `gorm:"column:store_city" json:"store_city"`
	StoreState        string `gorm:"column:store_state" json:"store_state"`
	StoreZipCode      string `gorm:"column:store_zip_code" json:"store_zip_code"`
	StoreCountry      string `gorm:"column:store_country;default:'BR'" json:"store_country"`
	StorePhone        string `gorm:"column:store_phone" json:"store_phone"` // Telefone da loja

	// Geolocation Fields
	StoreLatitude  *float64 `gorm:"column:store_latitude;type:decimal(10,8)" json:"store_latitude"`
	StoreLongitude *float64 `gorm:"column:store_longitude;type:decimal(11,8)" json:"store_longitude"`

	// Delivery Configuration
	DeliveryRadiusKm int `gorm:"column:delivery_radius_km;default:0" json:"delivery_radius_km"`

	// Credit System Configuration
	CostPerMessage int `gorm:"column:cost_per_message;default:0;check:cost_per_message >= 0" json:"cost_per_message"` // Custo em créditos por mensagem IA (0 = grátis)

	// AI Customization Configuration
	EnableAIPromptCustomization bool   `gorm:"column:enable_ai_prompt_customization;default:false" json:"enable_ai_prompt_customization"` // Permite customização de prompts IA
	BusinessCategory            string `gorm:"column:business_category;default:'loja'" json:"business_category"`                          // Categoria do negócio (Farmacia, Hamburgeria, Pizzaria, etc.)

	// Store Public Configuration
	IsPublicStore bool    `gorm:"column:is_public_store;default:false" json:"is_public_store"` // Permite acesso público ao catálogo da loja
	Tag           *string `gorm:"column:tag;unique;index" json:"tag"`                          // TAG única para lojas públicas (obrigatório se IsPublicStore=true)

	// Plan relationship
	PlanInfo *Plan `gorm:"foreignKey:PlanID;references:ID" json:"plan_info,omitempty"`
}

// TenantDeliveryZone represents whitelist/blacklist areas for delivery
type TenantDeliveryZone struct {
	BaseModel
	TenantID         uuid.UUID `gorm:"type:uuid;not null;index;constraint:OnDelete:RESTRICT" json:"tenant_id"`
	NeighborhoodName string    `gorm:"not null" json:"neighborhood_name" validate:"required"`
	City             string    `json:"city"`
	State            string    `json:"state"`
	ZoneType         string    `gorm:"not null;check:zone_type IN ('whitelist','blacklist')" json:"zone_type" validate:"required,oneof=whitelist blacklist"`

	// Relationship
	Tenant Tenant `gorm:"foreignKey:TenantID" json:"-"`
}

// User represents a system or tenant user
type User struct {
	BaseModel
	TenantID    *uuid.UUID `gorm:"type:uuid;index;constraint:OnDelete:SET NULL" json:"tenant_id,omitempty"` // null for system admins
	Email       string     `gorm:"unique;not null" json:"email" validate:"required,email"`
	Password    string     `gorm:"not null" json:"-"`
	Name        string     `gorm:"not null" json:"name" validate:"required"`
	Phone       string     `json:"phone"`
	Role        string     `gorm:"not null" json:"role" validate:"required"`
	IsActive    bool       `gorm:"default:true" json:"is_active"`
	LastLoginAt *time.Time `json:"last_login_at"`
}

// Role represents a role in the system
type Role struct {
	BaseModel
	Name        string `gorm:"unique;not null" json:"name" validate:"required"`
	Description string `json:"description"`
	Scope       string `gorm:"not null" json:"scope"` // 'system' or 'tenant'
	IsSystem    bool   `gorm:"default:false" json:"is_system"`
}

// Permission represents a permission
type Permission struct {
	BaseModel
	Name        string `gorm:"unique;not null" json:"name" validate:"required"`
	Description string `json:"description"`
	Resource    string `gorm:"not null" json:"resource"`
	Action      string `gorm:"not null" json:"action"`
}

// UserRole links users to roles
type UserRole struct {
	BaseModel
	UserID uuid.UUID `gorm:"type:uuid;not null" json:"user_id"`
	RoleID uuid.UUID `gorm:"type:uuid;not null" json:"role_id"`
}

// RolePermission links roles to permissions
type RolePermission struct {
	BaseModel
	RoleID       uuid.UUID `gorm:"type:uuid;not null" json:"role_id"`
	PermissionID uuid.UUID `gorm:"type:uuid;not null" json:"permission_id"`
}

// AuditLog represents an audit trail entry
type AuditLog struct {
	BaseModel
	TenantID   *uuid.UUID `gorm:"type:uuid;index;constraint:OnDelete:SET NULL" json:"tenant_id"`
	UserID     uuid.UUID  `gorm:"type:uuid;not null" json:"user_id"`
	Action     string     `gorm:"not null" json:"action"`
	Resource   string     `gorm:"not null" json:"resource"`
	ResourceID *uuid.UUID `gorm:"type:uuid" json:"resource_id"`
	OldValues  string     `json:"old_values"`
	NewValues  string     `json:"new_values"`
	IPAddress  string     `json:"ip_address"`
	UserAgent  string     `json:"user_agent"`
}

// UpdateProfileRequest represents a request to update user profile
type UpdateProfileRequest struct {
	Name  string `json:"name" validate:"required"`
	Email string `json:"email" validate:"required,email"`
	Phone string `json:"phone"`
}

// ChangePasswordRequest represents a request to change user password
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" validate:"required"`
	NewPassword     string `json:"new_password" validate:"required,min=6"`
}

// PasswordResetToken represents a token for password reset
type PasswordResetToken struct {
	BaseModel
	UserID    uuid.UUID  `gorm:"type:uuid;not null;index" json:"user_id"`
	Token     string     `gorm:"unique;not null" json:"token"`
	ExpiresAt time.Time  `gorm:"not null" json:"expires_at"`
	IsUsed    bool       `gorm:"default:false" json:"is_used"`
	UsedAt    *time.Time `json:"used_at"`

	// Relationship
	User *User `gorm:"foreignKey:UserID;references:ID" json:"user,omitempty"`
}

// ForgotPasswordRequest represents a request to reset password
type ForgotPasswordRequest struct {
	Email string `json:"email" validate:"required,email"`
}

// ResetPasswordRequest represents a request to reset password with token
type ResetPasswordRequest struct {
	Token       string `json:"token" validate:"required"`
	NewPassword string `json:"new_password" validate:"required,min=6"`
}

// AIErrorLog represents errors that occur during AI processing
type AIErrorLog struct {
	BaseTenantModel
	CustomerPhone string     `gorm:"not null" json:"customer_phone"`
	CustomerID    uuid.UUID  `gorm:"type:uuid" json:"customer_id"`
	UserMessage   string     `gorm:"not null" json:"user_message"`    // Mensagem original do usuário
	ToolName      string     `gorm:"not null" json:"tool_name"`       // Nome da ferramenta que falhou
	ToolArgs      string     `json:"tool_args"`                       // Argumentos da ferramenta (JSON)
	ErrorMessage  string     `gorm:"not null" json:"error_message"`   // Mensagem de erro técnica completa
	ErrorType     string     `gorm:"not null" json:"error_type"`      // Tipo do erro (db, api, validation, etc)
	UserResponse  string     `gorm:"not null" json:"user_response"`   // Resposta tratada enviada ao usuário
	Severity      string     `gorm:"default:'error'" json:"severity"` // error, warning, critical
	Resolved      bool       `gorm:"default:false" json:"resolved"`   // Se o erro foi resolvido
	ResolvedAt    *time.Time `json:"resolved_at"`                     // Quando foi resolvido
	ResolvedBy    *uuid.UUID `gorm:"type:uuid" json:"resolved_by"`    // Quem resolveu
	StackTrace    string     `json:"stack_trace"`                     // Stack trace se disponível
}

// TenantSetting represents configuration settings for a tenant
type TenantSetting struct {
	BaseModel
	TenantID     uuid.UUID `gorm:"type:uuid;index;not null;constraint:OnDelete:RESTRICT" json:"tenant_id"`
	SettingKey   string    `gorm:"size:100;not null" json:"setting_key"`
	SettingValue *string   `gorm:"type:text" json:"setting_value"`
	SettingType  string    `gorm:"size:50;default:'string'" json:"setting_type"`
	Description  string    `gorm:"type:text" json:"description"`
	IsActive     bool      `gorm:"default:true" json:"is_active"`
}
