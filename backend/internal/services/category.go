package services

import (
	"errors"
	"iafarma/internal/repo"
	"iafarma/pkg/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type CategoryService struct {
	categoryRepo *repo.CategoryRepository
}

func NewCategoryService(categoryRepo *repo.CategoryRepository) *CategoryService {
	return &CategoryService{
		categoryRepo: categoryRepo,
	}
}

// CreateCategory creates a new category
func (s *CategoryService) CreateCategory(req *models.CreateCategoryRequest) (*models.Category, error) {
	// Check if category with the same name already exists
	existing, err := s.categoryRepo.FindExistingCategory(req.TenantID, req.Name)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if existing != nil {
		return nil, errors.New("categoria com este nome já existe")
	}

	category := &models.Category{
		BaseTenantModel: models.BaseTenantModel{
			TenantID: req.TenantID,
		},
		Name:        req.Name,
		Description: req.Description,
		ParentID:    req.ParentID,
		Image:       req.Image,
		IsActive:    true,
		SortOrder:   req.SortOrder,
	}

	if err := s.categoryRepo.Create(category); err != nil {
		return nil, err
	}

	return category, nil
}

// UpdateCategory updates an existing category
func (s *CategoryService) UpdateCategory(tenantID, id uuid.UUID, req *models.UpdateCategoryRequest) (*models.Category, error) {
	category, err := s.categoryRepo.GetByID(tenantID, id)
	if err != nil {
		return nil, err
	}

	// Check if another category with the same name exists
	if req.Name != "" && req.Name != category.Name {
		existing, err := s.categoryRepo.FindExistingCategory(tenantID, req.Name)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		if existing != nil && existing.ID != id {
			return nil, errors.New("categoria com este nome já existe")
		}
		category.Name = req.Name
	}

	if req.Description != nil {
		category.Description = *req.Description
	}

	if req.ParentID != nil {
		category.ParentID = req.ParentID
	}

	if req.Image != nil {
		category.Image = *req.Image
	}

	if req.IsActive != nil {
		category.IsActive = *req.IsActive
	}

	if req.SortOrder != nil {
		category.SortOrder = *req.SortOrder
	}

	if err := s.categoryRepo.Update(category); err != nil {
		return nil, err
	}

	return category, nil
}

// DeleteCategory deletes a category
func (s *CategoryService) DeleteCategory(tenantID, id uuid.UUID) error {
	// Check if category exists
	_, err := s.categoryRepo.GetByID(tenantID, id)
	if err != nil {
		return err
	}

	// Check if there are products using this category
	count, err := s.categoryRepo.CountProducts(tenantID, id)
	if err != nil {
		return err
	}

	if count > 0 {
		return errors.New("não é possível excluir categoria que possui produtos")
	}

	return s.categoryRepo.Delete(tenantID, id)
}

// GetCategoryByID gets a category by ID
func (s *CategoryService) GetCategoryByID(tenantID, id uuid.UUID) (*models.Category, error) {
	return s.categoryRepo.GetByID(tenantID, id)
}

// ListCategories lists all categories for a tenant
func (s *CategoryService) ListCategories(tenantID uuid.UUID) ([]models.Category, error) {
	return s.categoryRepo.List(tenantID)
}

// GetRootCategories gets all root categories (without parent) for a tenant
func (s *CategoryService) GetRootCategories(tenantID uuid.UUID) ([]models.Category, error) {
	return s.categoryRepo.GetRootCategories(tenantID)
}

// GetCategoriesByParent gets all categories by parent ID
func (s *CategoryService) GetCategoriesByParent(tenantID, parentID uuid.UUID) ([]models.Category, error) {
	return s.categoryRepo.GetByParent(tenantID, parentID)
}
