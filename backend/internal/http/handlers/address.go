package handlers

import (
	"net/http"

	"iafarma/internal/repo"
	"iafarma/pkg/models"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type AddressHandler struct {
	addressRepo  *repo.AddressRepository
	customerRepo *repo.CustomerRepository
}

func NewAddressHandler(addressRepo *repo.AddressRepository, customerRepo *repo.CustomerRepository) *AddressHandler {
	return &AddressHandler{
		addressRepo:  addressRepo,
		customerRepo: customerRepo,
	}
}

// CreateAddress creates a new address for a customer
func (h *AddressHandler) CreateAddress(c echo.Context) error {
	tenantID := c.Get("tenant_id").(uuid.UUID)

	var req models.CreateAddressRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}

	// Validate that customer exists and belongs to tenant
	_, err := h.customerRepo.GetByID(tenantID, req.CustomerID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Customer not found"})
	}

	// If this is set as default, unset other defaults first
	if req.IsDefault {
		addresses, _ := h.addressRepo.GetByCustomerID(tenantID, req.CustomerID)
		for _, addr := range addresses {
			if addr.IsDefault {
				addr.IsDefault = false
				h.addressRepo.Update(&addr)
			}
		}
	}

	address := &models.Address{
		CustomerID:   req.CustomerID,
		Label:        req.Label,
		Street:       req.Street,
		Number:       req.Number,
		Complement:   req.Complement,
		Neighborhood: req.Neighborhood,
		City:         req.City,
		State:        req.State,
		ZipCode:      req.ZipCode,
		Country:      req.Country,
		IsDefault:    req.IsDefault,
	}

	// Set tenant ID
	address.TenantID = tenantID

	if address.Country == "" {
		address.Country = "BR"
	}

	if err := h.addressRepo.Create(address); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to create address"})
	}

	return c.JSON(http.StatusCreated, address)
}

// GetAddressesByCustomer returns all addresses for a customer
func (h *AddressHandler) GetAddressesByCustomer(c echo.Context) error {
	tenantID := c.Get("tenant_id").(uuid.UUID)
	customerIDStr := c.Param("customer_id")

	customerID, err := uuid.Parse(customerIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid customer ID"})
	}

	// Validate that customer exists and belongs to tenant
	_, err = h.customerRepo.GetByID(tenantID, customerID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Customer not found"})
	}

	addresses, err := h.addressRepo.GetByCustomerID(tenantID, customerID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to get addresses"})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"addresses": addresses,
		"total":     len(addresses),
	})
}

// GetAddress returns a specific address
func (h *AddressHandler) GetAddress(c echo.Context) error {
	tenantID := c.Get("tenant_id").(uuid.UUID)
	addressIDStr := c.Param("id")

	addressID, err := uuid.Parse(addressIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid address ID"})
	}

	address, err := h.addressRepo.GetByID(tenantID, addressID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Address not found"})
	}

	return c.JSON(http.StatusOK, address)
}

// UpdateAddress updates an existing address
func (h *AddressHandler) UpdateAddress(c echo.Context) error {
	tenantID := c.Get("tenant_id").(uuid.UUID)
	addressIDStr := c.Param("id")

	addressID, err := uuid.Parse(addressIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid address ID"})
	}

	address, err := h.addressRepo.GetByID(tenantID, addressID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Address not found"})
	}

	var req models.UpdateAddressRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}

	// Update fields if provided
	if req.Label != nil {
		address.Label = *req.Label
	}
	if req.Street != nil {
		address.Street = *req.Street
	}
	if req.Number != nil {
		address.Number = *req.Number
	}
	if req.Complement != nil {
		address.Complement = *req.Complement
	}
	if req.Neighborhood != nil {
		address.Neighborhood = *req.Neighborhood
	}
	if req.City != nil {
		address.City = *req.City
	}
	if req.State != nil {
		address.State = *req.State
	}
	if req.ZipCode != nil {
		address.ZipCode = *req.ZipCode
	}
	if req.Country != nil {
		address.Country = *req.Country
	}
	if req.IsDefault != nil {
		address.IsDefault = *req.IsDefault
	}

	// Update the address first
	if err := h.addressRepo.Update(address); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to update address"})
	}

	// If setting as default, unset other defaults after updating
	if req.IsDefault != nil && *req.IsDefault {
		h.addressRepo.SetDefault(tenantID, address.CustomerID, addressID)
	}

	return c.JSON(http.StatusOK, address)
}

// DeleteAddress deletes an address
func (h *AddressHandler) DeleteAddress(c echo.Context) error {
	tenantID := c.Get("tenant_id").(uuid.UUID)
	addressIDStr := c.Param("id")

	addressID, err := uuid.Parse(addressIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid address ID"})
	}

	// Check if address exists
	_, err = h.addressRepo.GetByID(tenantID, addressID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Address not found"})
	}

	if err := h.addressRepo.Delete(tenantID, addressID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to delete address"})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "Address deleted successfully"})
}

// SetDefaultAddress sets an address as default for a customer
func (h *AddressHandler) SetDefaultAddress(c echo.Context) error {
	tenantID := c.Get("tenant_id").(uuid.UUID)
	addressIDStr := c.Param("id")

	addressID, err := uuid.Parse(addressIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid address ID"})
	}

	address, err := h.addressRepo.GetByID(tenantID, addressID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Address not found"})
	}

	if err := h.addressRepo.SetDefault(tenantID, address.CustomerID, addressID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to set default address"})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "Address set as default successfully"})
}
