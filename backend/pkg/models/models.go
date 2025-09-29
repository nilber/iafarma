package models

// PaginationResult represents paginated results
type PaginationResult[T any] struct {
	Data       []T   `json:"data"`
	Total      int64 `json:"total"`
	Page       int   `json:"page"`
	PerPage    int   `json:"per_page"`
	TotalPages int   `json:"total_pages"`
}

// Internal repository types (with GORM dependencies)

// ChannelWithConversationCount represents a channel with conversation count for internal use
type ChannelWithConversationCount struct {
	Channel
	ConversationCount int64 `json:"conversation_count"`
}

// Swagger-specific types (non-generic to avoid swag parsing issues)

// SwaggerChannel represents a channel for swagger docs (without GORM dependencies)
type SwaggerChannel struct {
	ID            string `json:"id"`
	TenantID      string `json:"tenant_id"`
	Name          string `json:"name"`
	Type          string `json:"type"`
	PhoneNumber   string `json:"phone_number,omitempty"`
	InstanceID    string `json:"instance_id,omitempty"`
	AccessToken   string `json:"access_token,omitempty"`
	BusinessPhone string `json:"business_phone,omitempty"`
	WebhookURL    string `json:"webhook_url,omitempty"`
	WebhookToken  string `json:"webhook_token,omitempty"`
	IsActive      bool   `json:"is_active"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
}

// SwaggerChannelWithConversationCount represents a channel with conversation count for swagger docs
type SwaggerChannelWithConversationCount struct {
	SwaggerChannel
	ConversationCount int64 `json:"conversation_count"`
}

// ChannelListResponse represents paginated channel results for Swagger docs
type ChannelListResponse struct {
	Data       []SwaggerChannelWithConversationCount `json:"data"`
	Total      int64                                 `json:"total"`
	Page       int                                   `json:"page"`
	PerPage    int                                   `json:"per_page"`
	TotalPages int                                   `json:"total_pages"`
}

// SwaggerProduct represents a product for swagger docs (without GORM dependencies)
type SwaggerProduct struct {
	ID          string `json:"id"`
	TenantID    string `json:"tenant_id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Price       int64  `json:"price"`
	SKU         string `json:"sku,omitempty"`
	IsActive    bool   `json:"is_active"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

// ProductListResponse represents paginated product results for Swagger docs
type ProductListResponse struct {
	Data       []SwaggerProduct `json:"data"`
	Total      int64            `json:"total"`
	Page       int              `json:"page"`
	PerPage    int              `json:"per_page"`
	TotalPages int              `json:"total_pages"`
}

// SwaggerCustomer represents a customer for swagger docs (without GORM dependencies)
type SwaggerCustomer struct {
	ID        string `json:"id"`
	TenantID  string `json:"tenant_id"`
	Name      string `json:"name"`
	Email     string `json:"email,omitempty"`
	Phone     string `json:"phone,omitempty"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// CustomerListResponse represents paginated customer results for Swagger docs
type CustomerListResponse struct {
	Data       []SwaggerCustomer `json:"data"`
	Total      int64             `json:"total"`
	Page       int               `json:"page"`
	PerPage    int               `json:"per_page"`
	TotalPages int               `json:"total_pages"`
}

// SwaggerOrder represents an order for swagger docs (without GORM dependencies)
type SwaggerOrder struct {
	ID            string `json:"id"`
	TenantID      string `json:"tenant_id"`
	CustomerID    string `json:"customer_id"`
	Status        string `json:"status"`
	TotalAmount   int64  `json:"total_amount"`
	PaymentStatus string `json:"payment_status"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
}

// SwaggerOrderWithCustomer represents an order with customer information for Swagger docs
type SwaggerOrderWithCustomer struct {
	SwaggerOrder
	CustomerName      string `json:"customer_name"`
	CustomerEmail     string `json:"customer_email"`
	PaymentMethodName string `json:"payment_method_name"`
	ItemsCount        int    `json:"items_count"`
}

// OrderListResponse represents paginated order results for Swagger docs
type OrderListResponse struct {
	Data       []SwaggerOrderWithCustomer `json:"data"`
	Total      int64                      `json:"total"`
	Page       int                        `json:"page"`
	PerPage    int                        `json:"per_page"`
	TotalPages int                        `json:"total_pages"`
}

// SwaggerTenant represents a tenant for swagger docs (without GORM dependencies)
type SwaggerTenant struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Domain    string `json:"domain,omitempty"`
	Plan      string `json:"plan"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// SwaggerTenantWithAdmin represents a tenant with admin email information for Swagger docs
type SwaggerTenantWithAdmin struct {
	SwaggerTenant
	AdminEmail string `json:"admin_email"`
}

// TenantListResponse represents paginated tenant results for Swagger docs
type TenantListResponse struct {
	Data       []SwaggerTenantWithAdmin `json:"data"`
	Total      int64                    `json:"total"`
	Page       int                      `json:"page"`
	PerPage    int                      `json:"per_page"`
	TotalPages int                      `json:"total_pages"`
}

// GetAllModels returns all models for GORM AutoMigrate
func GetAllModels() []interface{} {
	return []interface{}{
		// Core models
		&Tenant{},
		&TenantDeliveryZone{},
		&User{},
		&Role{},

		// Sales models
		&Customer{},
		&Category{},
		&Product{},
		&ProductVariant{},
		&ProductMedia{},
		&ProductCharacteristic{},
		&CharacteristicItem{},
		&Inventory{},
		&Cart{},
		&CartItem{},
		&CartItemAttribute{},
		&PaymentMethod{},
		&Order{},
		&OrderItem{},
		&OrderItemAttribute{},
		&Payment{},
		&Shipment{},
		&OrderStatusHistory{},
		&Promotion{},
		&Coupon{},
		&DomainEvent{},

		// Address models
		&Address{},
		&MunicipioBrasileiro{},

		// Messaging models
		&Channel{},
		&Conversation{},
		&ConversationUser{},
		&Message{},
		&MessageMedia{},
		&Tag{},
		&ConversationTag{},
		&QuickReply{},
		&MessageTemplate{},
		&SLAPolicy{},
		&AgentAssignment{},
		&Alert{},
		&ConversationMemory{},

		// Notification models
		&NotificationTemplate{},
		&ScheduledNotification{},
		&NotificationLog{},
		&NotificationControl{},

		// Import Job models
		&ImportJob{},

		// AI Credits models
		&AICredits{},
		&AICreditTransaction{},

		// Billing models
		&Plan{},
		&TenantUsage{},
		&UsageAlert{},

		// System models
		&AIErrorLog{},
		&TenantSetting{},

		// Password reset tokens
		&PasswordResetToken{},
	}
}
