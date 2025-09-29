package handlers

import (
	"iafarma/internal/services"
	"iafarma/pkg/models"
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type CategoryHandler struct {
	categoryService *services.CategoryService
}

func NewCategoryHandler(categoryService *services.CategoryService) *CategoryHandler {
	return &CategoryHandler{
		categoryService: categoryService,
	}
}

// List godoc
// @Summary List categories
// @Description Get all categories for a tenant, ordered by sort_order
// @Tags categories
// @Produce json
// @Success 200 {array} models.Category
// @Failure 500 {object} map[string]string
// @Router /categories [get]
// @Security BearerAuth
func (h *CategoryHandler) List(c echo.Context) error {
	tenantID := c.Get("tenant_id").(uuid.UUID)

	categories, err := h.categoryService.ListCategories(tenantID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to fetch categories"})
	}

	return c.JSON(http.StatusOK, categories)
}

// GetByID godoc
// @Summary Get category by ID
// @Description Get a specific category by ID
// @Tags categories
// @Produce json
// @Param id path string true "Category ID"
// @Success 200 {object} models.Category
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /categories/{id} [get]
// @Security BearerAuth
func (h *CategoryHandler) GetByID(c echo.Context) error {
	tenantID := c.Get("tenant_id").(uuid.UUID)

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid category ID"})
	}

	category, err := h.categoryService.GetCategoryByID(tenantID, id)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "category not found"})
	}

	return c.JSON(http.StatusOK, category)
}

// Create godoc
// @Summary Create category
// @Description Create a new category
// @Tags categories
// @Accept json
// @Produce json
// @Param category body models.CreateCategoryRequest true "Category data"
// @Success 201 {object} models.Category
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /categories [post]
// @Security BearerAuth
func (h *CategoryHandler) Create(c echo.Context) error {
	tenantID := c.Get("tenant_id").(uuid.UUID)

	var req models.CreateCategoryRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	req.TenantID = tenantID

	category, err := h.categoryService.CreateCategory(&req)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, category)
}

// Update godoc
// @Summary Update category
// @Description Update an existing category
// @Tags categories
// @Accept json
// @Produce json
// @Param id path string true "Category ID"
// @Param category body models.UpdateCategoryRequest true "Category data"
// @Success 200 {object} models.Category
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /categories/{id} [put]
// @Security BearerAuth
func (h *CategoryHandler) Update(c echo.Context) error {
	tenantID := c.Get("tenant_id").(uuid.UUID)

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid category ID"})
	}

	var req models.UpdateCategoryRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	category, err := h.categoryService.UpdateCategory(tenantID, id, &req)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, category)
}

// Delete godoc
// @Summary Delete category
// @Description Delete a category
// @Tags categories
// @Param id path string true "Category ID"
// @Success 204 "No Content"
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /categories/{id} [delete]
// @Security BearerAuth
func (h *CategoryHandler) Delete(c echo.Context) error {
	tenantID := c.Get("tenant_id").(uuid.UUID)

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid category ID"})
	}

	if err := h.categoryService.DeleteCategory(tenantID, id); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	return c.NoContent(http.StatusNoContent)
}

// GetRootCategories godoc
// @Summary Get root categories
// @Description Get all categories without parent for a tenant
// @Tags categories
// @Produce json
// @Success 200 {array} models.Category
// @Failure 500 {object} map[string]string
// @Router /categories/root [get]
// @Security BearerAuth
func (h *CategoryHandler) GetRootCategories(c echo.Context) error {
	tenantID := c.Get("tenant_id").(uuid.UUID)

	categories, err := h.categoryService.GetRootCategories(tenantID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to fetch root categories"})
	}

	return c.JSON(http.StatusOK, categories)
}

// GetByParent godoc
// @Summary Get categories by parent
// @Description Get all categories by parent ID
// @Tags categories
// @Produce json
// @Param parent_id path string true "Parent Category ID"
// @Success 200 {array} models.Category
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /categories/parent/{parent_id} [get]
// @Security BearerAuth
func (h *CategoryHandler) GetByParent(c echo.Context) error {
	tenantID := c.Get("tenant_id").(uuid.UUID)

	parentID, err := uuid.Parse(c.Param("parent_id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid parent ID"})
	}

	categories, err := h.categoryService.GetCategoriesByParent(tenantID, parentID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to fetch categories"})
	}

	return c.JSON(http.StatusOK, categories)
}

// RegisterRoutes registers category routes
func (h *CategoryHandler) RegisterRoutes(e *echo.Group) {
	categoryGroup := e.Group("/categories")

	categoryGroup.GET("", h.List)
	categoryGroup.GET("/root", h.GetRootCategories)
	categoryGroup.GET("/:id", h.GetByID)
	categoryGroup.POST("", h.Create)
	categoryGroup.PUT("/:id", h.Update)
	categoryGroup.DELETE("/:id", h.Delete)
	categoryGroup.GET("/parent/:parent_id", h.GetByParent)
}
