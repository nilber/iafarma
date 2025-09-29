package repo

import (
	"iafarma/pkg/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type MessageTemplateRepository struct {
	db *gorm.DB
}

func NewMessageTemplateRepository(db *gorm.DB) *MessageTemplateRepository {
	return &MessageTemplateRepository{db: db}
}

// Create creates a new message template
func (r *MessageTemplateRepository) Create(template *models.MessageTemplate) error {
	return r.db.Create(template).Error
}

// GetByID gets a message template by ID
func (r *MessageTemplateRepository) GetByID(id uuid.UUID) (*models.MessageTemplate, error) {
	var template models.MessageTemplate
	err := r.db.Where("id = ?", id).First(&template).Error
	return &template, err
}

// GetByIDAndUser gets a message template by ID and user ID
func (r *MessageTemplateRepository) GetByIDAndUser(id, userID uuid.UUID) (*models.MessageTemplate, error) {
	var template models.MessageTemplate
	err := r.db.Where("id = ? AND user_id = ?", id, userID).First(&template).Error
	return &template, err
}

// ListByUser lists message templates for a specific user
func (r *MessageTemplateRepository) ListByUser(userID uuid.UUID, limit, offset int) ([]*models.MessageTemplate, error) {
	var templates []*models.MessageTemplate
	query := r.db.Where("user_id = ? AND is_active = ?", userID, true).
		Order("usage_count DESC, title ASC")

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	err := query.Find(&templates).Error
	return templates, err
}

// ListByUserAndCategory lists message templates for a specific user and category
func (r *MessageTemplateRepository) ListByUserAndCategory(userID uuid.UUID, category string, limit, offset int) ([]*models.MessageTemplate, error) {
	var templates []*models.MessageTemplate
	query := r.db.Where("user_id = ? AND category = ? AND is_active = ?", userID, category, true).
		Order("usage_count DESC, title ASC")

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	err := query.Find(&templates).Error
	return templates, err
}

// Update updates a message template
func (r *MessageTemplateRepository) Update(template *models.MessageTemplate) error {
	return r.db.Save(template).Error
}

// Delete deletes a message template (soft delete by setting is_active to false)
func (r *MessageTemplateRepository) Delete(id, userID uuid.UUID) error {
	return r.db.Model(&models.MessageTemplate{}).
		Where("id = ? AND user_id = ?", id, userID).
		Update("is_active", false).Error
}

// IncrementUsageCount increments the usage count for a template
func (r *MessageTemplateRepository) IncrementUsageCount(id uuid.UUID) error {
	return r.db.Model(&models.MessageTemplate{}).
		Where("id = ?", id).
		UpdateColumn("usage_count", gorm.Expr("usage_count + 1")).Error
}

// GetCategories gets all categories for a user's templates
func (r *MessageTemplateRepository) GetCategories(userID uuid.UUID) ([]string, error) {
	var categories []string
	err := r.db.Model(&models.MessageTemplate{}).
		Where("user_id = ? AND is_active = ? AND category IS NOT NULL AND category != ''", userID, true).
		Distinct("category").
		Pluck("category", &categories).Error
	return categories, err
}

// Search searches templates by title or content
func (r *MessageTemplateRepository) Search(userID uuid.UUID, searchTerm string, limit, offset int) ([]*models.MessageTemplate, error) {
	var templates []*models.MessageTemplate
	query := r.db.Where("user_id = ? AND is_active = ? AND (title ILIKE ? OR content ILIKE ?)",
		userID, true, "%"+searchTerm+"%", "%"+searchTerm+"%").
		Order("usage_count DESC, title ASC")

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	err := query.Find(&templates).Error
	return templates, err
}
