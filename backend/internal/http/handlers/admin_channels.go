package handlers

import (
	"net/http"
	"strconv"

	"iafarma/internal/repo"
	"iafarma/internal/services"
	"iafarma/pkg/models"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type AdminChannelHandler struct {
	channelRepo      *repo.ChannelRepository
	planLimitService *services.PlanLimitService
}

func NewAdminChannelHandler(channelRepo *repo.ChannelRepository, planLimitService *services.PlanLimitService) *AdminChannelHandler {
	return &AdminChannelHandler{
		channelRepo:      channelRepo,
		planLimitService: planLimitService,
	}
}

// ListByTenant godoc
// @Summary List channels for a specific tenant (Admin only)
// @Description List all channels for a specific tenant - only accessible by system admin
// @Tags admin,channels
// @Accept json
// @Produce json
// @Param tenant_id path string true "Tenant ID"
// @Param limit query int false "Limit"
// @Param offset query int false "Offset"
// @Success 200 {object} models.ChannelListResponse
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /admin/tenants/{tenant_id}/channels [get]
func (h *AdminChannelHandler) ListByTenant(c echo.Context) error {
	tenantIDStr := c.Param("tenant_id")
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid tenant ID"})
	}

	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	offset, _ := strconv.Atoi(c.QueryParam("offset"))

	if limit <= 0 {
		limit = 20
	}

	result, err := h.channelRepo.ListByTenant(tenantID, limit, offset)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, result)
}

// CreateForTenant godoc
// @Summary Create a channel for a specific tenant (Admin only)
// @Description Create a new channel for a specific tenant - only accessible by system admin
// @Tags admin,channels
// @Accept json
// @Produce json
// @Param tenant_id path string true "Tenant ID"
// @Param channel body models.Channel true "Channel data"
// @Success 201 {object} models.Channel
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /admin/tenants/{tenant_id}/channels [post]
func (h *AdminChannelHandler) CreateForTenant(c echo.Context) error {
	tenantIDStr := c.Param("tenant_id")
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid tenant ID"})
	}

	var channel models.Channel
	if err := c.Bind(&channel); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid channel data"})
	}

	// Set the tenant ID from URL parameter
	channel.TenantID = tenantID

	// Validate channel limit for the tenant
	if err := h.planLimitService.ValidateChannelLimit(tenantID, 1); err != nil {
		return c.JSON(http.StatusForbidden, map[string]string{"error": err.Error()})
	}

	// Create the channel
	if err := h.channelRepo.Create(&channel); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, channel)
}

// UpdateForTenant godoc
// @Summary Update a channel for a specific tenant (Admin only)
// @Description Update a channel for a specific tenant - only accessible by system admin
// @Tags admin,channels
// @Accept json
// @Produce json
// @Param tenant_id path string true "Tenant ID"
// @Param id path string true "Channel ID"
// @Param channel body models.Channel true "Channel data"
// @Success 200 {object} models.Channel
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /admin/tenants/{tenant_id}/channels/{id} [put]
func (h *AdminChannelHandler) UpdateForTenant(c echo.Context) error {
	tenantIDStr := c.Param("tenant_id")
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid tenant ID"})
	}

	channelIDStr := c.Param("id")
	channelID, err := uuid.Parse(channelIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid channel ID"})
	}

	var updates models.Channel
	if err := c.Bind(&updates); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid channel data"})
	}

	// Get existing channel to verify it belongs to the tenant
	existingChannel, err := h.channelRepo.GetByID(channelID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Channel not found"})
	}

	if existingChannel.TenantID != tenantID {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Channel not found for this tenant"})
	}

	// Update the channel
	updates.ID = channelID
	updates.TenantID = tenantID
	if err := h.channelRepo.Update(&updates); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	// Get updated channel
	updatedChannel, err := h.channelRepo.GetByID(channelID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to retrieve updated channel"})
	}

	return c.JSON(http.StatusOK, updatedChannel)
}

// DeleteForTenant godoc
// @Summary Delete a channel for a specific tenant (Admin only)
// @Description Delete a channel for a specific tenant - only accessible by system admin
// @Tags admin,channels
// @Accept json
// @Produce json
// @Param tenant_id path string true "Tenant ID"
// @Param id path string true "Channel ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /admin/tenants/{tenant_id}/channels/{id} [delete]
func (h *AdminChannelHandler) DeleteForTenant(c echo.Context) error {
	tenantIDStr := c.Param("tenant_id")
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid tenant ID"})
	}

	channelIDStr := c.Param("id")
	channelID, err := uuid.Parse(channelIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid channel ID"})
	}

	// Get existing channel to verify it belongs to the tenant
	existingChannel, err := h.channelRepo.GetByID(channelID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Channel not found"})
	}

	if existingChannel.TenantID != tenantID {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Channel not found for this tenant"})
	}

	// Delete the channel
	if err := h.channelRepo.Delete(channelID, tenantID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "Channel deleted successfully"})
}
