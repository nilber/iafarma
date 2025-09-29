package handlers

import (
	"net/http"

	"iafarma/internal/services"

	"github.com/labstack/echo/v4"
)

// ChannelMonitorHandler handles channel monitoring API endpoints
type ChannelMonitorHandler struct {
	monitorService *services.ChannelMonitorService
}

// NewChannelMonitorHandler creates a new channel monitor handler
func NewChannelMonitorHandler(monitorService *services.ChannelMonitorService) *ChannelMonitorHandler {
	return &ChannelMonitorHandler{
		monitorService: monitorService,
	}
}

// GetMonitoringStatus godoc
// @Summary Get channel monitoring status
// @Description Get the current status of the channel monitoring service
// @Tags monitoring
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]string
// @Router /monitoring/channels/status [get]
// @Security BearerAuth
func (h *ChannelMonitorHandler) GetMonitoringStatus(c echo.Context) error {
	status := h.monitorService.GetMonitoringStatus()
	return c.JSON(http.StatusOK, status)
}
