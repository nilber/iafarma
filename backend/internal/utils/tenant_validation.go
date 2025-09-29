package utils

import (
	"errors"
	"fmt"

	"iafarma/pkg/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// TenantValidationError represents a tenant validation error
type TenantValidationError struct {
	ResourceType string
	ResourceID   uuid.UUID
	TenantID     uuid.UUID
}

func (e *TenantValidationError) Error() string {
	return fmt.Sprintf("%s with ID %s not found or access denied for tenant %s",
		e.ResourceType, e.ResourceID, e.TenantID)
}

// ValidateCustomerBelongsToTenant validates that a customer belongs to the specified tenant
func ValidateCustomerBelongsToTenant(db *gorm.DB, tenantID, customerID uuid.UUID) error {
	if customerID == uuid.Nil {
		return nil // Allow nil customers (optional fields)
	}

	var count int64
	if err := db.Model(&models.Customer{}).
		Where("id = ? AND tenant_id = ?", customerID, tenantID).
		Count(&count).Error; err != nil {
		return fmt.Errorf("failed to validate customer: %w", err)
	}

	if count == 0 {
		return &TenantValidationError{
			ResourceType: "customer",
			ResourceID:   customerID,
			TenantID:     tenantID,
		}
	}

	return nil
}

// ValidateProductBelongsToTenant validates that a product belongs to the specified tenant
func ValidateProductBelongsToTenant(db *gorm.DB, tenantID, productID uuid.UUID) error {
	if productID == uuid.Nil {
		return nil // Allow nil products (optional fields)
	}

	var count int64
	if err := db.Model(&models.Product{}).
		Where("id = ? AND tenant_id = ?", productID, tenantID).
		Count(&count).Error; err != nil {
		return fmt.Errorf("failed to validate product: %w", err)
	}

	if count == 0 {
		return &TenantValidationError{
			ResourceType: "product",
			ResourceID:   productID,
			TenantID:     tenantID,
		}
	}

	return nil
}

// ValidateCategoryBelongsToTenant validates that a category belongs to the specified tenant
func ValidateCategoryBelongsToTenant(db *gorm.DB, tenantID, categoryID uuid.UUID) error {
	if categoryID == uuid.Nil {
		return nil // Allow nil categories (optional fields)
	}

	var count int64
	if err := db.Model(&models.Category{}).
		Where("id = ? AND tenant_id = ?", categoryID, tenantID).
		Count(&count).Error; err != nil {
		return fmt.Errorf("failed to validate category: %w", err)
	}

	if count == 0 {
		return &TenantValidationError{
			ResourceType: "category",
			ResourceID:   categoryID,
			TenantID:     tenantID,
		}
	}

	return nil
}

// ValidateAddressBelongsToTenant validates that an address belongs to the specified tenant
func ValidateAddressBelongsToTenant(db *gorm.DB, tenantID, addressID uuid.UUID) error {
	if addressID == uuid.Nil {
		return nil // Allow nil addresses (optional fields)
	}

	var count int64
	if err := db.Model(&models.Address{}).
		Where("id = ? AND tenant_id = ?", addressID, tenantID).
		Count(&count).Error; err != nil {
		return fmt.Errorf("failed to validate address: %w", err)
	}

	if count == 0 {
		return &TenantValidationError{
			ResourceType: "address",
			ResourceID:   addressID,
			TenantID:     tenantID,
		}
	}

	return nil
}

// ValidateOrderBelongsToTenant validates that an order belongs to the specified tenant
func ValidateOrderBelongsToTenant(db *gorm.DB, tenantID, orderID uuid.UUID) error {
	if orderID == uuid.Nil {
		return nil // Allow nil orders (optional fields)
	}

	var count int64
	if err := db.Model(&models.Order{}).
		Where("id = ? AND tenant_id = ?", orderID, tenantID).
		Count(&count).Error; err != nil {
		return fmt.Errorf("failed to validate order: %w", err)
	}

	if count == 0 {
		return &TenantValidationError{
			ResourceType: "order",
			ResourceID:   orderID,
			TenantID:     tenantID,
		}
	}

	return nil
}

// ValidateConversationBelongsToTenant validates that a conversation belongs to the specified tenant
func ValidateConversationBelongsToTenant(db *gorm.DB, tenantID, conversationID uuid.UUID) error {
	if conversationID == uuid.Nil {
		return nil // Allow nil conversations (optional fields)
	}

	var count int64
	if err := db.Model(&models.Conversation{}).
		Where("id = ? AND tenant_id = ?", conversationID, tenantID).
		Count(&count).Error; err != nil {
		return fmt.Errorf("failed to validate conversation: %w", err)
	}

	if count == 0 {
		return &TenantValidationError{
			ResourceType: "conversation",
			ResourceID:   conversationID,
			TenantID:     tenantID,
		}
	}

	return nil
}

// ValidatePaymentMethodBelongsToTenant validates that a payment method belongs to the specified tenant
func ValidatePaymentMethodBelongsToTenant(db *gorm.DB, tenantID, paymentMethodID uuid.UUID) error {
	if paymentMethodID == uuid.Nil {
		return nil // Allow nil payment methods (optional fields)
	}

	var count int64
	if err := db.Model(&models.PaymentMethod{}).
		Where("id = ? AND tenant_id = ?", paymentMethodID, tenantID).
		Count(&count).Error; err != nil {
		return fmt.Errorf("failed to validate payment method: %w", err)
	}

	if count == 0 {
		return &TenantValidationError{
			ResourceType: "payment_method",
			ResourceID:   paymentMethodID,
			TenantID:     tenantID,
		}
	}

	return nil
}

// ValidateCartBelongsToTenant validates that a cart belongs to the specified tenant
func ValidateCartBelongsToTenant(db *gorm.DB, tenantID, cartID uuid.UUID) error {
	if cartID == uuid.Nil {
		return nil // Allow nil carts (optional fields)
	}

	var count int64
	if err := db.Model(&models.Cart{}).
		Where("id = ? AND tenant_id = ?", cartID, tenantID).
		Count(&count).Error; err != nil {
		return fmt.Errorf("failed to validate cart: %w", err)
	}

	if count == 0 {
		return &TenantValidationError{
			ResourceType: "cart",
			ResourceID:   cartID,
			TenantID:     tenantID,
		}
	}

	return nil
}

// ValidateAddressBelongsToCustomer validates that an address belongs to a specific customer within a tenant
func ValidateAddressBelongsToCustomer(db *gorm.DB, tenantID, customerID, addressID uuid.UUID) error {
	if addressID == uuid.Nil {
		return nil // Allow nil addresses (optional fields)
	}

	var count int64
	if err := db.Model(&models.Address{}).
		Where("id = ? AND customer_id = ? AND tenant_id = ?", addressID, customerID, tenantID).
		Count(&count).Error; err != nil {
		return fmt.Errorf("failed to validate address for customer: %w", err)
	}

	if count == 0 {
		return &TenantValidationError{
			ResourceType: "customer_address",
			ResourceID:   addressID,
			TenantID:     tenantID,
		}
	}

	return nil
}

// ValidateCartBelongsToCustomer validates that a cart belongs to a specific customer within a tenant
func ValidateCartBelongsToCustomer(db *gorm.DB, tenantID, customerID, cartID uuid.UUID) error {
	if cartID == uuid.Nil {
		return nil // Allow nil carts (optional fields)
	}

	var count int64
	if err := db.Model(&models.Cart{}).
		Where("id = ? AND customer_id = ? AND tenant_id = ?", cartID, customerID, tenantID).
		Count(&count).Error; err != nil {
		return fmt.Errorf("failed to validate cart for customer: %w", err)
	}

	if count == 0 {
		return &TenantValidationError{
			ResourceType: "customer_cart",
			ResourceID:   cartID,
			TenantID:     tenantID,
		}
	}

	return nil
}

// ValidateOrderBelongsToCustomer validates that an order belongs to a specific customer within a tenant
func ValidateOrderBelongsToCustomer(db *gorm.DB, tenantID, customerID, orderID uuid.UUID) error {
	if orderID == uuid.Nil {
		return nil // Allow nil orders (optional fields)
	}

	var count int64
	if err := db.Model(&models.Order{}).
		Where("id = ? AND customer_id = ? AND tenant_id = ?", orderID, customerID, tenantID).
		Count(&count).Error; err != nil {
		return fmt.Errorf("failed to validate order for customer: %w", err)
	}

	if count == 0 {
		return &TenantValidationError{
			ResourceType: "customer_order",
			ResourceID:   orderID,
			TenantID:     tenantID,
		}
	}

	return nil
}

// IsTenantValidationError checks if an error is a tenant validation error
func IsTenantValidationError(err error) bool {
	var tenantErr *TenantValidationError
	return errors.As(err, &tenantErr)
}
