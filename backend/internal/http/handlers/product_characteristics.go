package handlers

import (
	"iafarma/internal/repo"
	"iafarma/internal/utils"
	"iafarma/pkg/models"
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// ProductCharacteristicHandler handles product characteristic operations
type ProductCharacteristicHandler struct {
	characteristicRepo *repo.ProductCharacteristicRepository
	itemRepo           *repo.CharacteristicItemRepository
}

// NewProductCharacteristicHandler creates a new product characteristic handler
func NewProductCharacteristicHandler(characteristicRepo *repo.ProductCharacteristicRepository, itemRepo *repo.CharacteristicItemRepository) *ProductCharacteristicHandler {
	return &ProductCharacteristicHandler{
		characteristicRepo: characteristicRepo,
		itemRepo:           itemRepo,
	}
}

// CreateCharacteristic godoc
// @Summary Create product characteristic
// @Description Create a new characteristic for a product
// @Tags product-characteristics
// @Accept json
// @Produce json
// @Param product_id path string true "Product ID"
// @Param characteristic body models.ProductCharacteristic true "Characteristic data"
// @Success 201 {object} models.ProductCharacteristic
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /products/{product_id}/characteristics [post]
// @Security BearerAuth
func (h *ProductCharacteristicHandler) CreateCharacteristic(c echo.Context) error {
	productID, err := uuid.Parse(c.Param("product_id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid product ID"})
	}

	tenantID := c.Get("tenant_id").(uuid.UUID)

	// SECURITY: Validate product belongs to tenant
	if err := utils.ValidateProductBelongsToTenant(h.characteristicRepo.GetDB(), tenantID, productID); err != nil {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "Product access denied"})
	}

	var characteristic models.ProductCharacteristic
	if err := c.Bind(&characteristic); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}

	// Validate required fields
	if err := c.Validate(&characteristic); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	// Set the tenant ID and product ID
	characteristic.TenantID = tenantID
	characteristic.ProductID = productID

	// Create characteristic
	if err := h.characteristicRepo.Create(&characteristic); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to create characteristic"})
	}

	return c.JSON(http.StatusCreated, characteristic)
}

// GetCharacteristics godoc
// @Summary Get product characteristics
// @Description Get all characteristics for a product
// @Tags product-characteristics
// @Accept json
// @Produce json
// @Param product_id path string true "Product ID"
// @Success 200 {array} models.ProductCharacteristic
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /products/{product_id}/characteristics [get]
// @Security BearerAuth
func (h *ProductCharacteristicHandler) GetCharacteristics(c echo.Context) error {
	productID, err := uuid.Parse(c.Param("product_id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid product ID"})
	}

	tenantID := c.Get("tenant_id").(uuid.UUID)

	characteristics, err := h.characteristicRepo.GetByProductID(tenantID, productID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to get characteristics"})
	}

	return c.JSON(http.StatusOK, characteristics)
}

// GetCharacteristic godoc
// @Summary Get product characteristic by ID
// @Description Get a specific characteristic with its items
// @Tags product-characteristics
// @Accept json
// @Produce json
// @Param product_id path string true "Product ID"
// @Param id path string true "Characteristic ID"
// @Success 200 {object} models.ProductCharacteristic
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /products/{product_id}/characteristics/{id} [get]
// @Security BearerAuth
func (h *ProductCharacteristicHandler) GetCharacteristic(c echo.Context) error {
	characteristicID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid characteristic ID"})
	}

	tenantID := c.Get("tenant_id").(uuid.UUID)

	characteristic, err := h.characteristicRepo.GetByID(tenantID, characteristicID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Characteristic not found"})
	}

	return c.JSON(http.StatusOK, characteristic)
}

// UpdateCharacteristic godoc
// @Summary Update product characteristic
// @Description Update an existing characteristic
// @Tags product-characteristics
// @Accept json
// @Produce json
// @Param product_id path string true "Product ID"
// @Param id path string true "Characteristic ID"
// @Param characteristic body models.ProductCharacteristic true "Characteristic data"
// @Success 200 {object} models.ProductCharacteristic
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /products/{product_id}/characteristics/{id} [put]
// @Security BearerAuth
func (h *ProductCharacteristicHandler) UpdateCharacteristic(c echo.Context) error {
	characteristicID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid characteristic ID"})
	}

	productID, err := uuid.Parse(c.Param("product_id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid product ID"})
	}

	tenantID := c.Get("tenant_id").(uuid.UUID)

	// Check if characteristic exists
	existing, err := h.characteristicRepo.GetByID(tenantID, characteristicID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Characteristic not found"})
	}

	var characteristic models.ProductCharacteristic
	if err := c.Bind(&characteristic); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}

	// Validate required fields
	if err := c.Validate(&characteristic); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	// Keep the original ID, tenant ID, and product ID
	characteristic.BaseTenantModel = existing.BaseTenantModel
	characteristic.BaseTenantModel = existing.BaseTenantModel
	characteristic.ProductID = productID

	if err := h.characteristicRepo.Update(&characteristic); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to update characteristic"})
	}

	return c.JSON(http.StatusOK, characteristic)
}

// DeleteCharacteristic godoc
// @Summary Delete product characteristic
// @Description Delete a characteristic and all its items
// @Tags product-characteristics
// @Accept json
// @Produce json
// @Param product_id path string true "Product ID"
// @Param id path string true "Characteristic ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /products/{product_id}/characteristics/{id} [delete]
// @Security BearerAuth
func (h *ProductCharacteristicHandler) DeleteCharacteristic(c echo.Context) error {
	characteristicID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid characteristic ID"})
	}

	tenantID := c.Get("tenant_id").(uuid.UUID)

	// Check if characteristic exists
	_, err = h.characteristicRepo.GetByID(tenantID, characteristicID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Characteristic not found"})
	}

	if err := h.characteristicRepo.Delete(tenantID, characteristicID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to delete characteristic"})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "Characteristic deleted successfully"})
}

// CreateCharacteristicItem godoc
// @Summary Create characteristic item
// @Description Create a new item for a characteristic
// @Tags characteristic-items
// @Accept json
// @Produce json
// @Param characteristic_id path string true "Characteristic ID"
// @Param item body models.CharacteristicItem true "Item data"
// @Success 201 {object} models.CharacteristicItem
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /characteristics/{characteristic_id}/items [post]
// @Security BearerAuth
func (h *ProductCharacteristicHandler) CreateCharacteristicItem(c echo.Context) error {
	characteristicID, err := uuid.Parse(c.Param("characteristic_id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid characteristic ID"})
	}

	tenantID := c.Get("tenant_id").(uuid.UUID)

	var item models.CharacteristicItem
	if err := c.Bind(&item); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}

	// Validate required fields
	if err := c.Validate(&item); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	// Set the tenant ID and characteristic ID
	item.TenantID = tenantID
	item.CharacteristicID = characteristicID

	// Create item
	if err := h.itemRepo.Create(&item); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to create item"})
	}

	return c.JSON(http.StatusCreated, item)
}

// GetCharacteristicItems godoc
// @Summary Get characteristic items
// @Description Get all items for a characteristic
// @Tags characteristic-items
// @Accept json
// @Produce json
// @Param characteristic_id path string true "Characteristic ID"
// @Success 200 {array} models.CharacteristicItem
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /characteristics/{characteristic_id}/items [get]
// @Security BearerAuth
func (h *ProductCharacteristicHandler) GetCharacteristicItems(c echo.Context) error {
	characteristicID, err := uuid.Parse(c.Param("characteristic_id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid characteristic ID"})
	}

	tenantID := c.Get("tenant_id").(uuid.UUID)

	items, err := h.itemRepo.GetByCharacteristicID(tenantID, characteristicID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to get items"})
	}

	return c.JSON(http.StatusOK, items)
}

// UpdateCharacteristicItem godoc
// @Summary Update characteristic item
// @Description Update an existing characteristic item
// @Tags characteristic-items
// @Accept json
// @Produce json
// @Param characteristic_id path string true "Characteristic ID"
// @Param id path string true "Item ID"
// @Param item body models.CharacteristicItem true "Item data"
// @Success 200 {object} models.CharacteristicItem
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /characteristics/{characteristic_id}/items/{id} [put]
// @Security BearerAuth
func (h *ProductCharacteristicHandler) UpdateCharacteristicItem(c echo.Context) error {
	itemID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid item ID"})
	}

	characteristicID, err := uuid.Parse(c.Param("characteristic_id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid characteristic ID"})
	}

	tenantID := c.Get("tenant_id").(uuid.UUID)

	// Check if item exists
	existing, err := h.itemRepo.GetByID(tenantID, itemID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Item not found"})
	}

	var item models.CharacteristicItem
	if err := c.Bind(&item); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}

	// Validate required fields
	if err := c.Validate(&item); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	// Keep the original ID, tenant ID, and characteristic ID
	item.BaseTenantModel = existing.BaseTenantModel
	item.BaseTenantModel = existing.BaseTenantModel
	item.CharacteristicID = characteristicID

	if err := h.itemRepo.Update(&item); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to update item"})
	}

	return c.JSON(http.StatusOK, item)
}

// DeleteCharacteristicItem godoc
// @Summary Delete characteristic item
// @Description Delete a characteristic item
// @Tags characteristic-items
// @Accept json
// @Produce json
// @Param characteristic_id path string true "Characteristic ID"
// @Param id path string true "Item ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /characteristics/{characteristic_id}/items/{id} [delete]
// @Security BearerAuth
func (h *ProductCharacteristicHandler) DeleteCharacteristicItem(c echo.Context) error {
	itemID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid item ID"})
	}

	tenantID := c.Get("tenant_id").(uuid.UUID)

	// Check if item exists
	_, err = h.itemRepo.GetByID(tenantID, itemID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Item not found"})
	}

	if err := h.itemRepo.Delete(tenantID, itemID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to delete item"})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "Item deleted successfully"})
}
