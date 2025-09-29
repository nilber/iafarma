package repo

import (
	"iafarma/pkg/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AddressRepository struct {
	db *gorm.DB
}

func NewAddressRepository(db *gorm.DB) *AddressRepository {
	return &AddressRepository{db: db}
}

func (r *AddressRepository) Create(address *models.Address) error {
	return r.db.Create(address).Error
}

func (r *AddressRepository) GetByID(tenantID, addressID uuid.UUID) (*models.Address, error) {
	var address models.Address
	err := r.db.Where("id = ? AND tenant_id = ?", addressID, tenantID).First(&address).Error
	return &address, err
}

func (r *AddressRepository) GetByCustomerID(tenantID, customerID uuid.UUID) ([]models.Address, error) {
	var addresses []models.Address
	err := r.db.Where("customer_id = ? AND tenant_id = ?", customerID, tenantID).
		Order("is_default DESC, created_at ASC").
		Find(&addresses).Error
	return addresses, err
}

func (r *AddressRepository) Update(address *models.Address) error {
	return r.db.Save(address).Error
}

func (r *AddressRepository) Delete(tenantID, addressID uuid.UUID) error {
	return r.db.Where("id = ? AND tenant_id = ?", addressID, tenantID).Delete(&models.Address{}).Error
}

func (r *AddressRepository) SetDefault(tenantID, customerID, addressID uuid.UUID) error {
	// Begin transaction
	tx := r.db.Begin()

	// Remove default from all addresses of this customer
	if err := tx.Model(&models.Address{}).
		Where("customer_id = ? AND tenant_id = ?", customerID, tenantID).
		Update("is_default", false).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Set the selected address as default
	if err := tx.Model(&models.Address{}).
		Where("id = ? AND tenant_id = ?", addressID, tenantID).
		Update("is_default", true).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}
