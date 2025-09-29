package handlers

import (
	"fmt"
	"iafarma/internal/services"
	"iafarma/pkg/models"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
)

type DeliveryHandler struct {
	deliveryService *services.DeliveryService
}

func NewDeliveryHandler(deliveryService *services.DeliveryService) *DeliveryHandler {
	return &DeliveryHandler{
		deliveryService: deliveryService,
	}
}

// UpdateStoreLocationRequest represents the request to update store location
type UpdateStoreLocationRequest struct {
	StoreStreet       string  `json:"store_street" validate:"required"`
	StoreNumber       string  `json:"store_number"`
	StoreNeighborhood string  `json:"store_neighborhood"`
	StoreCity         string  `json:"store_city" validate:"required"`
	StoreState        string  `json:"store_state" validate:"required"`
	StoreZipCode      string  `json:"store_zip_code"`
	StoreCountry      string  `json:"store_country"`
	DeliveryRadiusKm  float64 `json:"delivery_radius_km"`
}

// ManageDeliveryZoneRequest represents the request to manage delivery zones
type ManageDeliveryZoneRequest struct {
	Neighborhood string `json:"neighborhood" validate:"required"`
	City         string `json:"city" validate:"required"`
	State        string `json:"state" validate:"required"`
	ZoneType     string `json:"zone_type" validate:"required,oneof=whitelist blacklist"`
	Action       string `json:"action" validate:"required,oneof=add remove"`
}

// ValidateDeliveryRequest represents the request to validate delivery address
type ValidateDeliveryRequest struct {
	Street       string `json:"street" validate:"required"`
	Number       string `json:"number"`
	Neighborhood string `json:"neighborhood"`
	City         string `json:"city" validate:"required"`
	State        string `json:"state" validate:"required"`
	ZipCode      string `json:"zip_code"`
	Country      string `json:"country"`
}

// UpdateStoreLocation updates the tenant's store location and geocodes it
func (h *DeliveryHandler) UpdateStoreLocation(c echo.Context) error {
	tenantID, ok := c.Get("tenant_id").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "tenant not found"})
	}

	var req UpdateStoreLocationRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	// Build full store address
	var addressParts []string
	addressParts = append(addressParts, req.StoreStreet)
	if req.StoreNumber != "" {
		addressParts = append(addressParts, req.StoreNumber)
	}
	if req.StoreNeighborhood != "" {
		addressParts = append(addressParts, req.StoreNeighborhood)
	}
	addressParts = append(addressParts, req.StoreCity, req.StoreState)
	if req.StoreZipCode != "" {
		addressParts = append(addressParts, req.StoreZipCode)
	}
	if req.StoreCountry != "" {
		addressParts = append(addressParts, req.StoreCountry)
	}

	fullAddress := strings.Join(addressParts, ", ")

	// Prepare update data
	updates := map[string]interface{}{
		"store_street":       req.StoreStreet,
		"store_number":       req.StoreNumber,
		"store_neighborhood": req.StoreNeighborhood,
		"store_city":         req.StoreCity,
		"store_state":        req.StoreState,
		"store_zip_code":     req.StoreZipCode,
		"store_country":      req.StoreCountry,
		"delivery_radius_km": req.DeliveryRadiusKm,
	}

	// Geocode the address and update location
	err := h.deliveryService.UpdateTenantStoreLocation(c.Request().Context(), tenantID, fullAddress, updates)
	if err != nil {
		log.Error().Err(err).Msg("Failed to update store location")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to update store location"})
	}

	// Also update the address fields and delivery radius in the database
	// This would be implemented via the delivery service
	log.Info().
		Str("tenant_id", tenantID.String()).
		Str("address", fullAddress).
		Float64("radius_km", req.DeliveryRadiusKm).
		Msg("Store location updated successfully")

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message":            "Store location updated successfully",
		"address":            fullAddress,
		"delivery_radius_km": req.DeliveryRadiusKm,
	})
}

// ManageDeliveryZone adds or removes neighborhoods from whitelist/blacklist
func (h *DeliveryHandler) ManageDeliveryZone(c echo.Context) error {
	tenantID, ok := c.Get("tenant_id").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "tenant not found"})
	}

	var req ManageDeliveryZoneRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	add := req.Action == "add"
	err := h.deliveryService.ManageDeliveryZone(
		tenantID,
		req.Neighborhood,
		req.City,
		req.State,
		req.ZoneType,
		add,
	)

	if err != nil {
		log.Error().Err(err).Msg("Failed to manage delivery zone")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to manage delivery zone"})
	}

	action := "added to"
	if !add {
		action = "removed from"
	}

	log.Info().
		Str("tenant_id", tenantID.String()).
		Str("neighborhood", req.Neighborhood).
		Str("zone_type", req.ZoneType).
		Str("action", req.Action).
		Msg("Delivery zone updated")

	return c.JSON(http.StatusOK, map[string]string{
		"message": fmt.Sprintf("Neighborhood '%s' %s %s", req.Neighborhood, action, req.ZoneType),
	})
}

// GetDeliveryZones returns all delivery zones for the tenant
func (h *DeliveryHandler) GetDeliveryZones(c echo.Context) error {
	tenantID, ok := c.Get("tenant_id").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "tenant not found"})
	}

	zones, err := h.deliveryService.GetDeliveryZones(tenantID)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get delivery zones")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to get delivery zones"})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"zones": zones,
	})
}

// ValidateDeliveryAddress validates if delivery can be made to an address
func (h *DeliveryHandler) ValidateDeliveryAddress(c echo.Context) error {
	tenantID, ok := c.Get("tenant_id").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "tenant not found"})
	}

	var req ValidateDeliveryRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	// Convert request to models.Address
	address := models.Address{
		Street:       req.Street,
		Number:       req.Number,
		Neighborhood: req.Neighborhood,
		City:         req.City,
		State:        req.State,
		ZipCode:      req.ZipCode,
		Country:      req.Country,
	}

	result, err := h.deliveryService.ValidateDeliveryAddress(c.Request().Context(), tenantID, address)
	if err != nil {
		log.Error().Err(err).Msg("Failed to validate delivery address")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to validate delivery address"})
	}

	return c.JSON(http.StatusOK, result)
}

// GetStoreLocation returns the current store location configuration
func (h *DeliveryHandler) GetStoreLocation(c echo.Context) error {
	tenantID, ok := c.Get("tenant_id").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "tenant not found"})
	}

	tenant, err := h.deliveryService.GetTenantStoreConfiguration(tenantID)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get store configuration")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to get store configuration"})
	}

	response := map[string]interface{}{
		"store_street":       tenant.StoreStreet,
		"store_number":       tenant.StoreNumber,
		"store_neighborhood": tenant.StoreNeighborhood,
		"store_city":         tenant.StoreCity,
		"store_state":        tenant.StoreState,
		"store_zip_code":     tenant.StoreZipCode,
		"store_country":      tenant.StoreCountry,
		"delivery_radius_km": tenant.DeliveryRadiusKm,
	}

	if tenant.StoreLatitude != nil && tenant.StoreLongitude != nil {
		response["store_latitude"] = *tenant.StoreLatitude
		response["store_longitude"] = *tenant.StoreLongitude
		response["coordinates_configured"] = true
	} else {
		response["coordinates_configured"] = false
	}

	return c.JSON(http.StatusOK, response)
}
