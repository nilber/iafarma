package repo

import (
	"iafarma/pkg/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ProductCharacteristicRepository handles product characteristics data access
type ProductCharacteristicRepository struct {
	db *gorm.DB
}

// NewProductCharacteristicRepository creates a new product characteristic repository
func NewProductCharacteristicRepository(db *gorm.DB) *ProductCharacteristicRepository {
	return &ProductCharacteristicRepository{db: db}
}

// GetDB returns the database instance for direct queries
func (r *ProductCharacteristicRepository) GetDB() *gorm.DB {
	return r.db
}

// Create creates a new product characteristic
func (r *ProductCharacteristicRepository) Create(characteristic *models.ProductCharacteristic) error {
	return r.db.Create(characteristic).Error
}

// GetByID gets a characteristic by ID
func (r *ProductCharacteristicRepository) GetByID(tenantID, id uuid.UUID) (*models.ProductCharacteristic, error) {
	var characteristic models.ProductCharacteristic
	err := r.db.Preload("Items").Where("id = ? AND tenant_id = ?", id, tenantID).First(&characteristic).Error
	if err != nil {
		return nil, err
	}
	return &characteristic, nil
}

// GetByProductID gets all characteristics for a product
func (r *ProductCharacteristicRepository) GetByProductID(tenantID, productID uuid.UUID) ([]models.ProductCharacteristic, error) {
	var characteristics []models.ProductCharacteristic
	err := r.db.Preload("Items", func(db *gorm.DB) *gorm.DB {
		return db.Order("sort_order ASC")
	}).Where("product_id = ? AND tenant_id = ?", productID, tenantID).Order("sort_order ASC").Find(&characteristics).Error
	return characteristics, err
}

// Update updates a characteristic
func (r *ProductCharacteristicRepository) Update(characteristic *models.ProductCharacteristic) error {
	return r.db.Save(characteristic).Error
}

// Delete deletes a characteristic
func (r *ProductCharacteristicRepository) Delete(tenantID, id uuid.UUID) error {
	return r.db.Where("id = ? AND tenant_id = ?", id, tenantID).Delete(&models.ProductCharacteristic{}).Error
}

// CharacteristicItemRepository handles characteristic items data access
type CharacteristicItemRepository struct {
	db *gorm.DB
}

// NewCharacteristicItemRepository creates a new characteristic item repository
func NewCharacteristicItemRepository(db *gorm.DB) *CharacteristicItemRepository {
	return &CharacteristicItemRepository{db: db}
}

// Create creates a new characteristic item
func (r *CharacteristicItemRepository) Create(item *models.CharacteristicItem) error {
	return r.db.Create(item).Error
}

// GetByID gets an item by ID
func (r *CharacteristicItemRepository) GetByID(tenantID, id uuid.UUID) (*models.CharacteristicItem, error) {
	var item models.CharacteristicItem
	err := r.db.Where("id = ? AND tenant_id = ?", id, tenantID).First(&item).Error
	if err != nil {
		return nil, err
	}
	return &item, nil
}

// GetByCharacteristicID gets all items for a characteristic
func (r *CharacteristicItemRepository) GetByCharacteristicID(tenantID, characteristicID uuid.UUID) ([]models.CharacteristicItem, error) {
	var items []models.CharacteristicItem
	err := r.db.Where("characteristic_id = ? AND tenant_id = ?", characteristicID, tenantID).Order("sort_order ASC").Find(&items).Error
	return items, err
}

// Update updates an item
func (r *CharacteristicItemRepository) Update(item *models.CharacteristicItem) error {
	return r.db.Save(item).Error
}

// Delete deletes an item
func (r *CharacteristicItemRepository) Delete(tenantID, id uuid.UUID) error {
	return r.db.Where("id = ? AND tenant_id = ?", id, tenantID).Delete(&models.CharacteristicItem{}).Error
}
