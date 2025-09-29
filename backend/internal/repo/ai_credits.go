package repo

import (
	"iafarma/pkg/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// AICreditRepository handles AI credits data access
type AICreditRepository struct {
	db *gorm.DB
}

// NewAICreditRepository creates a new AI credit repository
func NewAICreditRepository(db *gorm.DB) *AICreditRepository {
	return &AICreditRepository{db: db}
}

// GetByTenantID gets AI credits by tenant ID
func (r *AICreditRepository) GetByTenantID(tenantID uuid.UUID) (*models.AICredits, error) {
	var credits models.AICredits
	err := r.db.Where("tenant_id = ?", tenantID).First(&credits).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// Create initial credits record with 0 credits
			credits = models.AICredits{
				TenantID:         tenantID,
				TotalCredits:     0,
				UsedCredits:      0,
				RemainingCredits: 0,
			}
			if createErr := r.db.Create(&credits).Error; createErr != nil {
				return nil, createErr
			}
			return &credits, nil
		}
		return nil, err
	}
	return &credits, nil
}

// Update updates AI credits
func (r *AICreditRepository) Update(credits *models.AICredits) error {
	return r.db.Save(credits).Error
}

// CreateTransaction creates a new AI credit transaction
func (r *AICreditRepository) CreateTransaction(transaction *models.AICreditTransaction) error {
	return r.db.Create(transaction).Error
}

// GetTransactionsByTenantID gets transactions by tenant ID with pagination
func (r *AICreditRepository) GetTransactionsByTenantID(tenantID uuid.UUID, limit, offset int) ([]models.AICreditTransaction, error) {
	var transactions []models.AICreditTransaction
	err := r.db.Where("tenant_id = ?", tenantID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&transactions).Error
	return transactions, err
}

// UseCredits uses credits for a tenant atomically
func (r *AICreditRepository) UseCredits(tenantID uuid.UUID, userID *uuid.UUID, amount int, description string, relatedEntity string, relatedID *uuid.UUID) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// Get current credits for update
		var credits models.AICredits
		if err := tx.Where("tenant_id = ?", tenantID).First(&credits).Error; err != nil {
			return err
		}

		// Check if there are enough credits
		if credits.RemainingCredits < amount {
			return gorm.ErrCheckConstraintViolated // You can create a custom error
		}

		// Use credits
		if !credits.UseCredits(amount) {
			return gorm.ErrCheckConstraintViolated
		}

		// Update credits
		if err := tx.Save(&credits).Error; err != nil {
			return err
		}

		// Create transaction record
		transaction := models.AICreditTransaction{
			TenantID:      tenantID,
			UserID:        userID,
			Type:          "use",
			Amount:        amount,
			Description:   description,
			RelatedEntity: relatedEntity,
			RelatedID:     relatedID,
		}

		return tx.Create(&transaction).Error
	})
}

// AddCredits adds credits for a tenant atomically
func (r *AICreditRepository) AddCredits(tenantID uuid.UUID, userID *uuid.UUID, amount int, description string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// Get current credits for update
		var credits models.AICredits
		if err := tx.Where("tenant_id = ?", tenantID).First(&credits).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				// Create new credits record
				credits = models.AICredits{
					TenantID:         tenantID,
					TotalCredits:     0,
					UsedCredits:      0,
					RemainingCredits: 0,
				}
			} else {
				return err
			}
		}

		// Add credits
		credits.AddCredits(amount)
		credits.LastUpdatedBy = userID

		// Update credits
		if err := tx.Save(&credits).Error; err != nil {
			return err
		}

		// Create transaction record
		transaction := models.AICreditTransaction{
			TenantID:    tenantID,
			UserID:      userID,
			Type:        "add",
			Amount:      amount,
			Description: description,
		}

		return tx.Create(&transaction).Error
	})
}
