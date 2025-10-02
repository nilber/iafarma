package models

import (
	"crypto/md5"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Customer represents a customer in the system
type Customer struct {
	BaseTenantModel
	Phone     string     `gorm:"not null" json:"phone" validate:"required,numeric"` // Only numbers, no formatting
	Name      string     `json:"name"`
	Email     string     `json:"email"`
	Document  string     `json:"document"` // CPF/CNPJ
	BirthDate *time.Time `json:"birth_date"`
	Gender    string     `json:"gender"`
	Notes     string     `json:"notes"`
	IsActive  bool       `gorm:"default:true" json:"is_active"`
}

// Category represents a product category
type Category struct {
	BaseTenantModel
	Name        string     `gorm:"not null" json:"name" validate:"required"`
	Description string     `json:"description"`
	ParentID    *uuid.UUID `gorm:"type:uuid;constraint:OnDelete:SET NULL" json:"parent_id"`
	Image       string     `json:"image"`
	IsActive    bool       `gorm:"default:true" json:"is_active"`
	SortOrder   int        `gorm:"default:0" json:"sort_order"`
}

// Product represents a product in the catalog
type Product struct {
	BaseTenantModel
	CategoryID        *uuid.UUID `gorm:"type:uuid;constraint:OnDelete:SET NULL" json:"category_id"`
	Name              string     `gorm:"not null" json:"name" validate:"required"`
	Description       string     `json:"description"`
	Price             string     `gorm:"not null" json:"price" validate:"required"`
	SalePrice         string     `json:"sale_price"`
	SKU               string     `gorm:"uniqueIndex:uni_products_tenant_sku;not null" json:"sku"`
	Barcode           string     `json:"barcode"`
	Weight            string     `json:"weight"`     // in grams
	Dimensions        string     `json:"dimensions"` // LxWxH in cm
	Brand             string     `json:"brand"`
	Tags              string     `json:"tags"`
	StockQuantity     int        `gorm:"default:0" json:"stock_quantity"`
	LowStockThreshold int        `gorm:"default:5" json:"low_stock_threshold"`
	SortOrder         int        `gorm:"default:0" json:"sort_order"`
	SearchVector      string     `gorm:"type:tsvector;-" json:"-"`               // Full Text Search vector (não incluir no JSON)
	SearchText        string     `gorm:"type:text;-" json:"-"`                   // Texto combinado para busca semântica
	EmbeddingHash     string     `gorm:"type:varchar(64)" json:"embedding_hash"` // Hash do conteúdo para evitar reprocessamento
}

// ProductVariant represents a product variant
type ProductVariant struct {
	BaseTenantModel
	ProductID         uuid.UUID `gorm:"type:uuid;not null;constraint:OnDelete:RESTRICT" json:"product_id"`
	Name              string    `gorm:"not null" json:"name" validate:"required"`
	SKU               string    `json:"sku"`
	Price             string    `gorm:"not null" json:"price" validate:"required"`
	CompareAtPrice    string    `json:"compare_at_price"` // Original price for discounts
	StockQuantity     int       `gorm:"default:0" json:"stock_quantity"`
	LowStockThreshold int       `gorm:"default:5" json:"low_stock_threshold"`
	IsActive          bool      `gorm:"default:true" json:"is_active"`
}

// ProductMedia represents media files associated with a product
type ProductMedia struct {
	BaseTenantModel
	ProductID uuid.UUID `gorm:"type:uuid;not null;constraint:OnDelete:RESTRICT" json:"product_id"`
	Type      string    `gorm:"not null" json:"type"` // image, video
	URL       string    `gorm:"not null" json:"url"`
	S3Key     string    `json:"s3_key"`
	Alt       string    `json:"alt"`
	SortOrder int       `gorm:"default:0" json:"sort_order"`
}

// ProductCharacteristic represents a characteristic of a product (e.g. size, flavor, border)
type ProductCharacteristic struct {
	BaseTenantModel
	ProductID        uuid.UUID            `gorm:"type:uuid;not null;constraint:OnDelete:RESTRICT" json:"product_id"`
	Title            string               `gorm:"not null" json:"title" validate:"required"`
	IsRequired       bool                 `gorm:"default:false" json:"is_required"`
	IsMultipleChoice bool                 `gorm:"default:false" json:"is_multiple_choice"`
	SortOrder        int                  `gorm:"default:0" json:"sort_order"`
	Items            []CharacteristicItem `gorm:"foreignKey:CharacteristicID;constraint:OnDelete:RESTRICT" json:"items"`
}

// CharacteristicItem represents an item/option within a product characteristic
type CharacteristicItem struct {
	BaseTenantModel
	CharacteristicID uuid.UUID `gorm:"type:uuid;not null;constraint:OnDelete:RESTRICT" json:"characteristic_id"`
	Name             string    `gorm:"not null" json:"name" validate:"required"`
	Price            string    `gorm:"default:'0'" json:"price"` // Additional price for this option
	SortOrder        int       `gorm:"default:0" json:"sort_order"`
}

// Inventory represents product inventory
type Inventory struct {
	BaseTenantModel
	VariantID         uuid.UUID `gorm:"type:uuid;not null;unique;constraint:OnDelete:RESTRICT" json:"variant_id"`
	StockQuantity     int       `gorm:"default:0" json:"stock_quantity"`
	ReservedQuantity  int       `gorm:"default:0" json:"reserved_quantity"`
	LowStockThreshold int       `gorm:"default:5" json:"low_stock_threshold"`
}

// Cart represents a shopping cart
type Cart struct {
	BaseTenantModel
	CustomerID      uuid.UUID  `gorm:"type:uuid;not null;constraint:OnDelete:RESTRICT" json:"customer_id"`
	PaymentMethodID *uuid.UUID `gorm:"type:uuid;constraint:OnDelete:SET NULL" json:"payment_method_id"` // Forma de pagamento selecionada
	Status          string     `gorm:"default:'active'" json:"status"`                                  // active, checkout, completed, abandoned
	ExpiresAt       *time.Time `json:"expires_at"`
	TotalAmount     string     `gorm:"default:'0'" json:"total_amount"`
	ItemsCount      int        `gorm:"default:0" json:"items_count"`
	DiscountCode    string     `json:"discount_code"`
	Observations    string     `json:"observations"` // Observações do carrinho (ex: precisa de troco, sem cebola, etc)
	ChangeFor       string     `json:"change_for"`   // Valor para troco quando pagamento em dinheiro

	// Relations
	Customer      *Customer      `gorm:"foreignKey:CustomerID" json:"customer,omitempty"`
	PaymentMethod *PaymentMethod `gorm:"foreignKey:PaymentMethodID" json:"payment_method,omitempty"`
	Items         []CartItem     `gorm:"foreignKey:CartID" json:"items,omitempty"`
}

// CartItem represents an item in a cart
type CartItem struct {
	BaseTenantModel
	CartID    uuid.UUID  `gorm:"type:uuid;not null;constraint:OnDelete:RESTRICT" json:"cart_id"`
	ProductID *uuid.UUID `gorm:"type:uuid;constraint:OnDelete:SET NULL" json:"product_id"`
	VariantID *uuid.UUID `gorm:"type:uuid;constraint:OnDelete:SET NULL" json:"variant_id"`
	Quantity  int        `gorm:"not null" json:"quantity" validate:"min=1"`
	Price     string     `gorm:"not null" json:"price"`

	// Historical product data for cart integrity
	ProductName        *string `json:"product_name"`
	ProductDescription *string `json:"product_description"`
	ProductSKU         *string `json:"product_sku"`

	// Relations
	Cart       *Cart               `gorm:"foreignKey:CartID" json:"cart,omitempty"`
	Product    *Product            `gorm:"foreignKey:ProductID" json:"product,omitempty"`
	Attributes []CartItemAttribute `gorm:"foreignKey:CartItemID" json:"attributes,omitempty"`
}

// Order represents an order
type Order struct {
	BaseTenantModel
	CustomerID        *uuid.UUID `gorm:"type:uuid;constraint:OnDelete:SET NULL" json:"customer_id"`
	AddressID         *uuid.UUID `gorm:"type:uuid;constraint:OnDelete:SET NULL" json:"address_id"`
	ConversationID    *uuid.UUID `gorm:"type:uuid;constraint:OnDelete:SET NULL" json:"conversation_id"`
	PaymentMethodID   *uuid.UUID `gorm:"type:uuid;constraint:OnDelete:SET NULL" json:"payment_method_id"`
	OrderNumber       string     `gorm:"not null" json:"order_number"`
	Status            string     `gorm:"default:'pending'" json:"status"`
	PaymentStatus     string     `gorm:"default:'pending'" json:"payment_status"`
	FulfillmentStatus string     `gorm:"default:'pending'" json:"fulfillment_status"`
	TotalAmount       string     `gorm:"default:'0'" json:"total_amount"`
	Subtotal          string     `gorm:"default:'0'" json:"subtotal"`
	TaxAmount         string     `gorm:"default:'0'" json:"tax_amount"`
	ShippingAmount    string     `gorm:"default:'0'" json:"shipping_amount"`
	DiscountAmount    string     `gorm:"default:'0'" json:"discount_amount"`
	Currency          string     `gorm:"default:'BRL'" json:"currency"`
	Notes             string     `json:"notes"`
	Observations      string     `json:"observations"` // Campo para observações do cliente (ex: precisa de troco, sem cebola, etc)
	ChangeFor         string     `json:"change_for"`   // Valor para troco quando pagamento em dinheiro
	ShippedAt         *time.Time `json:"shipped_at"`
	DeliveredAt       *time.Time `json:"delivered_at"`

	// Historical customer data for order integrity
	CustomerName     *string `json:"customer_name"`
	CustomerEmail    *string `json:"customer_email"`
	CustomerPhone    *string `json:"customer_phone"`
	CustomerDocument *string `json:"customer_document"`

	// Historical shipping address data
	ShippingName         *string `json:"shipping_name"`
	ShippingStreet       *string `json:"shipping_street"`
	ShippingNumber       *string `json:"shipping_number"`
	ShippingComplement   *string `json:"shipping_complement"`
	ShippingNeighborhood *string `json:"shipping_neighborhood"`
	ShippingCity         *string `json:"shipping_city"`
	ShippingState        *string `json:"shipping_state"`
	ShippingZipcode      *string `json:"shipping_zipcode"`
	ShippingCountry      *string `json:"shipping_country"`

	// Historical billing address data
	BillingName         *string `json:"billing_name"`
	BillingStreet       *string `json:"billing_street"`
	BillingNumber       *string `json:"billing_number"`
	BillingComplement   *string `json:"billing_complement"`
	BillingNeighborhood *string `json:"billing_neighborhood"`
	BillingCity         *string `json:"billing_city"`
	BillingState        *string `json:"billing_state"`
	BillingZipcode      *string `json:"billing_zipcode"`
	BillingCountry      *string `json:"billing_country"`

	// Relations
	Customer      *Customer      `gorm:"foreignKey:CustomerID" json:"customer,omitempty"`
	Address       *Address       `gorm:"foreignKey:AddressID" json:"address,omitempty"`
	Conversation  *Conversation  `gorm:"foreignKey:ConversationID" json:"conversation,omitempty"`
	PaymentMethod *PaymentMethod `gorm:"foreignKey:PaymentMethodID" json:"payment_method,omitempty"`
	Items         []OrderItem    `gorm:"foreignKey:OrderID" json:"items,omitempty"`
	Payments      []Payment      `gorm:"foreignKey:OrderID" json:"payments,omitempty"`
}

// OrderItem represents an item in an order
type OrderItem struct {
	BaseTenantModel
	OrderID   uuid.UUID  `gorm:"type:uuid;not null;constraint:OnDelete:RESTRICT" json:"order_id"`
	ProductID *uuid.UUID `gorm:"type:uuid;constraint:OnDelete:SET NULL" json:"product_id"` // Nullable for historical data integrity
	Quantity  int        `gorm:"not null" json:"quantity"`
	Price     string     `gorm:"not null" json:"price"`
	Total     string     `gorm:"not null" json:"total"`

	// Historical product data for order integrity
	ProductName         *string    `json:"product_name"`
	ProductDescription  *string    `json:"product_description"`
	ProductSKU          *string    `json:"product_sku"`
	ProductCategoryID   *uuid.UUID `gorm:"type:uuid;constraint:OnDelete:SET NULL" json:"product_category_id"`
	ProductCategoryName *string    `json:"product_category_name"`
	UnitPrice           *string    `json:"unit_price"`

	// Relations
	Product    *Product             `gorm:"foreignKey:ProductID" json:"product,omitempty"`
	Attributes []OrderItemAttribute `gorm:"foreignKey:OrderItemID" json:"attributes,omitempty"`
}

// Payment represents a payment
type Payment struct {
	BaseTenantModel
	OrderID     uuid.UUID  `gorm:"type:uuid;not null;constraint:OnDelete:RESTRICT" json:"order_id"`
	Method      string     `gorm:"not null" json:"method"` // credit_card, pix, boleto, etc.
	Status      string     `gorm:"default:'pending'" json:"status"`
	Amount      string     `gorm:"not null" json:"amount"`
	Currency    string     `gorm:"default:'BRL'" json:"currency"`
	ExternalID  string     `json:"external_id"` // Payment gateway ID
	ProcessedAt *time.Time `json:"processed_at"`
	ConfirmedAt *time.Time `json:"confirmed_at"`
}

// Shipment represents a shipment
type Shipment struct {
	BaseTenantModel
	OrderID       uuid.UUID  `gorm:"type:uuid;not null;constraint:OnDelete:RESTRICT" json:"order_id"`
	TrackingCode  string     `json:"tracking_code"`
	Carrier       string     `json:"carrier"`
	Status        string     `gorm:"default:'preparing'" json:"status"`
	ShippedAt     *time.Time `json:"shipped_at"`
	DeliveredAt   *time.Time `json:"delivered_at"`
	EstimatedDate *time.Time `json:"estimated_date"`
	Cost          string     `json:"cost"`
}

// OrderStatusHistory represents order status changes
type OrderStatusHistory struct {
	BaseTenantModel
	OrderID    uuid.UUID `gorm:"type:uuid;not null;constraint:OnDelete:RESTRICT" json:"order_id"`
	FromStatus string    `json:"from_status"`
	ToStatus   string    `gorm:"not null" json:"to_status"`
	Notes      string    `json:"notes"`
	ChangedBy  uuid.UUID `gorm:"type:uuid;constraint:OnDelete:RESTRICT" json:"changed_by"`
}

// Promotion represents promotional campaigns
type Promotion struct {
	BaseTenantModel
	Name               string     `gorm:"not null" json:"name"`
	Description        string     `json:"description"`
	Type               string     `gorm:"not null" json:"type"` // percentage, fixed_amount
	Value              string     `gorm:"not null" json:"value"`
	MinimumOrderAmount string     `json:"minimum_order_amount"`
	UsageLimit         *int       `json:"usage_limit"`
	UsageCount         int        `gorm:"default:0" json:"usage_count"`
	IsActive           bool       `gorm:"default:true" json:"is_active"`
	StartsAt           time.Time  `json:"starts_at"`
	ExpiresAt          *time.Time `json:"expires_at"`
}

// Coupon represents discount coupons
type Coupon struct {
	BaseTenantModel
	Code               string     `gorm:"unique;not null" json:"code"`
	Type               string     `gorm:"not null" json:"type"` // percentage, fixed_amount
	Value              string     `gorm:"not null" json:"value"`
	MinimumOrderAmount string     `json:"minimum_order_amount"`
	UsageLimit         *int       `json:"usage_limit"`
	UsageCount         int        `gorm:"default:0" json:"usage_count"`
	IsActive           bool       `gorm:"default:true" json:"is_active"`
	ExpiresAt          *time.Time `json:"expires_at"`
}

// DomainEvent represents domain events for event sourcing
type DomainEvent struct {
	BaseModel
	TenantID      uuid.UUID `gorm:"type:uuid;constraint:OnDelete:RESTRICT" json:"tenant_id"`
	AggregateID   uuid.UUID `gorm:"type:uuid;not null" json:"aggregate_id"`
	AggregateType string    `gorm:"not null" json:"aggregate_type"`
	EventType     string    `gorm:"not null" json:"event_type"`
	EventData     string    `json:"event_data"`
	Version       int       `gorm:"not null" json:"version"`
}

// Métodos para integração com RAG/Embedding

// GetSearchText retorna o texto combinado para busca semântica
func (p *Product) GetSearchText() string {
	var parts []string

	if p.Name != "" {
		parts = append(parts, p.Name)
	}
	if p.Description != "" {
		parts = append(parts, p.Description)
	}
	if p.Brand != "" {
		parts = append(parts, p.Brand)
	}
	if p.Tags != "" {
		parts = append(parts, p.Tags)
	}
	if p.SKU != "" {
		parts = append(parts, "SKU: "+p.SKU)
	}

	return strings.Join(parts, " ")
}

// GetEmbeddingHash retorna o hash MD5 do conteúdo para verificar se precisa reprocessar
func (p *Product) GetEmbeddingHash() string {
	content := p.GetSearchText()
	hash := md5.Sum([]byte(content))
	return fmt.Sprintf("%x", hash)
}

// NeedsEmbeddingUpdate verifica se o embedding precisa ser atualizado
func (p *Product) NeedsEmbeddingUpdate() bool {
	currentHash := p.GetEmbeddingHash()
	return p.EmbeddingHash != currentHash
}

// GetMetadata retorna os metadados do produto para armazenar junto com o embedding
func (p *Product) GetMetadata() map[string]interface{} {
	return map[string]interface{}{
		"name":                p.Name,
		"description":         p.Description,
		"price":               p.Price,
		"sale_price":          p.SalePrice,
		"sku":                 p.SKU,
		"brand":               p.Brand,
		"tags":                p.Tags,
		"stock_quantity":      p.StockQuantity,
		"low_stock_threshold": p.LowStockThreshold,
		"category_id":         p.CategoryID,
	}
}

// GORM Hooks para atualização automática do embedding
func (p *Product) BeforeCreate(tx *gorm.DB) error {
	// Chamar o hook da estrutura pai primeiro
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}

	// Processar SKU
	if p.SKU == "" {
		p.SKU = ""
	}

	// Atualizar campos de busca
	p.SearchText = p.GetSearchText()
	p.EmbeddingHash = ""

	return nil
}

func (p *Product) BeforeUpdate(tx *gorm.DB) error {
	// Processar SKU
	if p.SKU == "" {
		p.SKU = ""
	}

	// Atualizar campos de busca se necessário
	currentSearchText := p.GetSearchText()
	if p.SearchText != currentSearchText {
		p.SearchText = currentSearchText
		p.EmbeddingHash = "" // Forçar reatualização do embedding
	}

	return nil
}

// Category Request Models
type CreateCategoryRequest struct {
	TenantID    uuid.UUID  `json:"tenant_id"`
	Name        string     `json:"name" validate:"required"`
	Description string     `json:"description"`
	ParentID    *uuid.UUID `json:"parent_id"`
	Image       string     `json:"image"`
	SortOrder   int        `json:"sort_order"`
}

type UpdateCategoryRequest struct {
	Name        string     `json:"name"`
	Description *string    `json:"description"`
	ParentID    *uuid.UUID `json:"parent_id"`
	Image       *string    `json:"image"`
	IsActive    *bool      `json:"is_active"`
	SortOrder   *int       `json:"sort_order"`
}

// CartItemAttribute represents selected attributes for cart items
type CartItemAttribute struct {
	BaseTenantModel
	CartItemID    uuid.UUID `gorm:"type:uuid;not null" json:"cart_item_id"`
	AttributeID   uuid.UUID `gorm:"type:uuid;not null" json:"attribute_id"`
	OptionID      uuid.UUID `gorm:"type:uuid;not null" json:"option_id"`
	AttributeName string    `json:"attribute_name"`
	OptionName    string    `json:"option_name"`
	OptionPrice   string    `gorm:"default:'0'" json:"option_price"`

	// Relations
	CartItem *CartItem `gorm:"foreignKey:CartItemID" json:"cart_item,omitempty"`
}

// OrderItemAttribute represents selected attributes for order items
type OrderItemAttribute struct {
	BaseTenantModel
	OrderItemID   uuid.UUID `gorm:"type:uuid;not null" json:"order_item_id"`
	AttributeID   uuid.UUID `gorm:"type:uuid;not null" json:"attribute_id"`
	OptionID      uuid.UUID `gorm:"type:uuid;not null" json:"option_id"`
	AttributeName string    `gorm:"not null" json:"attribute_name"`
	OptionName    string    `gorm:"not null" json:"option_name"`
	OptionPrice   string    `gorm:"default:'0'" json:"option_price"`

	// Relations
	OrderItem *OrderItem `gorm:"foreignKey:OrderItemID" json:"order_item,omitempty"`
}

// PaymentMethod represents a payment method available for orders
type PaymentMethod struct {
	BaseTenantModel
	Name     string `gorm:"not null" json:"name" validate:"required"`
	IsActive bool   `gorm:"default:true" json:"is_active"`
}
