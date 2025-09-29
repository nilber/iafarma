package repo

import (
	"iafarma/pkg/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// CategoryRepository handles product category data access
type CategoryRepository struct {
	db *gorm.DB
}

// NewCategoryRepository creates a new category repository
func NewCategoryRepository(db *gorm.DB) *CategoryRepository {
	return &CategoryRepository{db: db}
}

// GetByID gets a category by ID
func (r *CategoryRepository) GetByID(tenantID, id uuid.UUID) (*models.Category, error) {
	var category models.Category
	if err := r.db.Where("id = ? AND tenant_id = ?", id, tenantID).First(&category).Error; err != nil {
		return nil, err
	}
	return &category, nil
}

// Create creates a new category
func (r *CategoryRepository) Create(category *models.Category) error {
	return r.db.Create(category).Error
}

// Update updates a category
func (r *CategoryRepository) Update(category *models.Category) error {
	return r.db.Save(category).Error
}

// Delete soft deletes a category
func (r *CategoryRepository) Delete(tenantID, id uuid.UUID) error {
	return r.db.Where("id = ? AND tenant_id = ?", id, tenantID).Delete(&models.Category{}).Error
}

// List gets all categories for a tenant, ordered by sort_order
func (r *CategoryRepository) List(tenantID uuid.UUID) ([]models.Category, error) {
	var categories []models.Category
	if err := r.db.Where("tenant_id = ?", tenantID).Order("sort_order ASC, name ASC").Find(&categories).Error; err != nil {
		return nil, err
	}
	return categories, nil
}

// GetRootCategories gets all categories without parent for a tenant
func (r *CategoryRepository) GetRootCategories(tenantID uuid.UUID) ([]models.Category, error) {
	var categories []models.Category
	if err := r.db.Where("tenant_id = ? AND parent_id IS NULL", tenantID).Order("sort_order ASC, name ASC").Find(&categories).Error; err != nil {
		return nil, err
	}
	return categories, nil
}

// GetByParent gets all categories by parent ID
func (r *CategoryRepository) GetByParent(tenantID, parentID uuid.UUID) ([]models.Category, error) {
	var categories []models.Category
	if err := r.db.Where("tenant_id = ? AND parent_id = ?", tenantID, parentID).Order("sort_order ASC, name ASC").Find(&categories).Error; err != nil {
		return nil, err
	}
	return categories, nil
}

// FindExistingCategory finds a category by name (case insensitive)
func (r *CategoryRepository) FindExistingCategory(tenantID uuid.UUID, name string) (*models.Category, error) {
	var category models.Category
	if err := r.db.Where("tenant_id = ? AND LOWER(name) = LOWER(?)", tenantID, name).First(&category).Error; err != nil {
		return nil, err
	}
	return &category, nil
}

// CountProducts counts how many products are in this category
func (r *CategoryRepository) CountProducts(tenantID, categoryID uuid.UUID) (int64, error) {
	var count int64
	if err := r.db.Model(&models.Product{}).Where("tenant_id = ? AND category_id = ?", tenantID, categoryID).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}
