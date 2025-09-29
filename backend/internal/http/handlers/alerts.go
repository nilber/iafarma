package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"iafarma/internal/services"
	"iafarma/pkg/models"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

// AlertHandler handles alert-related operations
type AlertHandler struct {
	db           *gorm.DB
	alertService *services.AlertService
}

// NewAlertHandler creates a new alert handler
func NewAlertHandler(db *gorm.DB) *AlertHandler {
	return &AlertHandler{
		db:           db,
		alertService: services.NewAlertService(db),
	}
}

// CreateAlert godoc
// @Summary Create alert
// @Description Create a new alert configuration for a channel
// @Tags alerts
// @Accept json
// @Produce json
// @Param alert body models.Alert true "Alert data"
// @Success 201 {object} models.Alert
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /alerts [post]
// @Security BearerAuth
func (h *AlertHandler) CreateAlert(c echo.Context) error {
	tenantID := c.Get("tenant_id").(uuid.UUID)

	var alert models.Alert
	if err := c.Bind(&alert); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request format",
		})
	}

	// Validate required fields
	if alert.Name == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Name is required",
		})
	}

	if alert.ChannelID == uuid.Nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Channel ID is required",
		})
	}

	// Verify channel belongs to tenant
	var channel models.Channel
	if err := h.db.Where("id = ? AND tenant_id = ?", alert.ChannelID, tenantID).First(&channel).Error; err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Channel not found",
		})
	}

	alert.TenantID = tenantID

	// Usar o nome do alerta como nome do grupo
	if alert.GroupName == "" {
		alert.GroupName = alert.Name
	}

	if err := h.db.Create(&alert).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to create alert",
		})
	}

	// Load channel relationship
	h.db.Preload("Channel").First(&alert, alert.ID)

	// Create WhatsApp group automatically if phones are provided
	if alert.Phones != "" && channel.Session != "" {
		if err := h.alertService.CreateWhatsAppGroup(&alert, channel.Session); err != nil {
			// Log the error but don't fail the alert creation
			c.Logger().Errorf("Failed to create WhatsApp group for alert %s: %v", alert.ID, err)
			// You might want to set a flag or status to indicate group creation failed
		}
	}

	return c.JSON(http.StatusCreated, alert)
}

// GetAlerts godoc
// @Summary Get alerts
// @Description Get all alerts for the current tenant
// @Tags alerts
// @Accept json
// @Produce json
// @Param page query int false "Page number"
// @Param limit query int false "Items per page"
// @Param channel_id query string false "Filter by channel ID"
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]string
// @Router /alerts [get]
// @Security BearerAuth
func (h *AlertHandler) GetAlerts(c echo.Context) error {
	tenantID := c.Get("tenant_id").(uuid.UUID)

	// Pagination
	page, _ := strconv.Atoi(c.QueryParam("page"))
	if page < 1 {
		page = 1
	}

	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	if limit < 1 || limit > 100 {
		limit = 20
	}

	offset := (page - 1) * limit

	// Build query
	query := h.db.Where("tenant_id = ?", tenantID)

	// Filter by channel if specified
	if channelID := c.QueryParam("channel_id"); channelID != "" {
		if id, err := uuid.Parse(channelID); err == nil {
			query = query.Where("channel_id = ?", id)
		}
	}

	// Count total
	var total int64
	query.Model(&models.Alert{}).Count(&total)

	// Get alerts
	var alerts []models.Alert
	err := query.Preload("Channel").
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&alerts).Error

	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to fetch alerts",
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"alerts": alerts,
		"pagination": map[string]interface{}{
			"page":       page,
			"limit":      limit,
			"total":      total,
			"totalPages": (total + int64(limit) - 1) / int64(limit),
		},
	})
}

// GetAlert godoc
// @Summary Get alert by ID
// @Description Get a specific alert by ID
// @Tags alerts
// @Accept json
// @Produce json
// @Param id path string true "Alert ID"
// @Success 200 {object} models.Alert
// @Failure 404 {object} map[string]string
// @Router /alerts/{id} [get]
// @Security BearerAuth
func (h *AlertHandler) GetAlert(c echo.Context) error {
	tenantID := c.Get("tenant_id").(uuid.UUID)
	alertID := c.Param("id")

	var alert models.Alert
	err := h.db.Where("id = ? AND tenant_id = ?", alertID, tenantID).
		Preload("Channel").
		First(&alert).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.JSON(http.StatusNotFound, map[string]string{
				"error": "Alert not found",
			})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to fetch alert",
		})
	}

	return c.JSON(http.StatusOK, alert)
}

// UpdateAlert godoc
// @Summary Update alert
// @Description Update an existing alert
// @Tags alerts
// @Accept json
// @Produce json
// @Param id path string true "Alert ID"
// @Param alert body models.Alert true "Updated alert data"
// @Success 200 {object} models.Alert
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /alerts/{id} [put]
// @Security BearerAuth
func (h *AlertHandler) UpdateAlert(c echo.Context) error {
	tenantID := c.Get("tenant_id").(uuid.UUID)
	alertID := c.Param("id")

	var existingAlert models.Alert
	if err := h.db.Where("id = ? AND tenant_id = ?", alertID, tenantID).Preload("Channel").First(&existingAlert).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.JSON(http.StatusNotFound, map[string]string{
				"error": "Alert not found",
			})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to fetch alert",
		})
	}

	var updateData models.Alert
	if err := c.Bind(&updateData); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request format",
		})
	}

	// Store old phones for comparison
	oldPhones := existingAlert.Phones

	// Update fields
	existingAlert.Name = updateData.Name
	existingAlert.Description = updateData.Description

	// Usar o nome do alerta como nome do grupo
	if updateData.GroupName == "" {
		existingAlert.GroupName = updateData.Name
	} else {
		existingAlert.GroupName = updateData.GroupName
	}

	// Note: Don't update GroupID from request as it should be managed by the system
	// existingAlert.GroupID = updateData.GroupID
	existingAlert.Phones = updateData.Phones
	existingAlert.TriggerOn = updateData.TriggerOn
	existingAlert.IsActive = updateData.IsActive

	// Validate channel change if provided
	if updateData.ChannelID != uuid.Nil {
		var channel models.Channel
		if err := h.db.Where("id = ? AND tenant_id = ?", updateData.ChannelID, tenantID).First(&channel).Error; err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Channel not found",
			})
		}
		existingAlert.ChannelID = updateData.ChannelID
		// Reload channel data
		h.db.Preload("Channel").First(&existingAlert, existingAlert.ID)
	}

	// Sync WhatsApp group participants if phones have changed and group exists
	if oldPhones != existingAlert.Phones && existingAlert.Channel.Session != "" {
		if existingAlert.GroupID != "" {
			// Group exists, sync participants
			if err := h.syncGroupParticipants(oldPhones, existingAlert.Phones, existingAlert.GroupID, existingAlert.Channel.Session); err != nil {
				c.Logger().Errorf("Failed to sync group participants for alert %s: %v", existingAlert.ID, err)
				// Don't fail the update operation, just log the error
			}
		} else if existingAlert.Phones != "" {
			// Group doesn't exist but phones are provided, create the group
			c.Logger().Infof("Creating WhatsApp group for alert %s as it doesn't exist yet", existingAlert.ID)
			if err := h.alertService.CreateWhatsAppGroup(&existingAlert, existingAlert.Channel.Session); err != nil {
				c.Logger().Errorf("Failed to create WhatsApp group for alert %s: %v", existingAlert.ID, err)
				// Don't fail the update operation, just log the error
			} else {
				c.Logger().Infof("Successfully created WhatsApp group for alert %s with ID: %s", existingAlert.ID, existingAlert.GroupID)
			}
		}
	}

	if err := h.db.Save(&existingAlert).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to update alert",
		})
	}

	// Load channel relationship if not already loaded
	if existingAlert.Channel.ID == uuid.Nil {
		h.db.Preload("Channel").First(&existingAlert, existingAlert.ID)
	}

	return c.JSON(http.StatusOK, existingAlert)
}

// syncGroupParticipants compares old and new phone lists and syncs WhatsApp group participants
func (h *AlertHandler) syncGroupParticipants(oldPhones, newPhones, groupID, session string) error {
	fmt.Printf("🔄 DEBUG syncGroupParticipants - Starting sync\n")
	fmt.Printf("🔄 DEBUG - Old phones: %s\n", oldPhones)
	fmt.Printf("🔄 DEBUG - New phones: %s\n", newPhones)
	fmt.Printf("🔄 DEBUG - Group ID: %s\n", groupID)
	fmt.Printf("🔄 DEBUG - Session: %s\n", session)

	// Parse phone lists
	oldPhoneList := parsePhoneList(oldPhones)
	newPhoneList := parsePhoneList(newPhones)

	fmt.Printf("🔄 DEBUG - Old phone list: %v\n", oldPhoneList)
	fmt.Printf("🔄 DEBUG - New phone list: %v\n", newPhoneList)

	// Find phones to add (in new but not in old)
	phonesToAdd := findPhoneDifference(newPhoneList, oldPhoneList)

	// Find phones to remove (in old but not in new)
	phonesToRemove := findPhoneDifference(oldPhoneList, newPhoneList)

	fmt.Printf("🔄 DEBUG - Phones to add: %v\n", phonesToAdd)
	fmt.Printf("🔄 DEBUG - Phones to remove: %v\n", phonesToRemove)

	// Add new participants
	for _, phone := range phonesToAdd {
		fmt.Printf("➕ DEBUG - Adding participant: %s\n", phone)
		if err := h.alertService.AddParticipantToGroup(groupID, phone, session); err != nil {
			fmt.Printf("❌ DEBUG - Failed to add participant %s: %v\n", phone, err)
			return fmt.Errorf("failed to add participant %s: %w", phone, err)
		}
		fmt.Printf("✅ DEBUG - Successfully added participant: %s\n", phone)
	}

	// Remove old participants
	for _, phone := range phonesToRemove {
		fmt.Printf("➖ DEBUG - Removing participant: %s\n", phone)
		if err := h.alertService.RemoveParticipantFromGroup(groupID, phone, session); err != nil {
			fmt.Printf("❌ DEBUG - Failed to remove participant %s: %v\n", phone, err)
			return fmt.Errorf("failed to remove participant %s: %w", phone, err)
		}
		fmt.Printf("✅ DEBUG - Successfully removed participant: %s\n", phone)
	}

	fmt.Printf("🔄 DEBUG syncGroupParticipants - Completed sync\n")
	return nil
}

// parsePhoneList parses a comma-separated phone list into a slice
func parsePhoneList(phones string) []string {
	if phones == "" {
		return []string{}
	}

	var phoneList []string
	for _, phone := range strings.Split(phones, ",") {
		cleanPhone := strings.TrimSpace(phone)
		if cleanPhone != "" {
			phoneList = append(phoneList, cleanPhone)
		}
	}
	return phoneList
}

// findPhoneDifference returns phones that are in list1 but not in list2
func findPhoneDifference(list1, list2 []string) []string {
	phoneMap := make(map[string]bool)
	for _, phone := range list2 {
		phoneMap[phone] = true
	}

	var difference []string
	for _, phone := range list1 {
		if !phoneMap[phone] {
			difference = append(difference, phone)
		}
	}
	return difference
}

// DeleteAlert godoc
// @Summary Delete alert
// @Description Delete an alert
// @Tags alerts
// @Accept json
// @Produce json
// @Param id path string true "Alert ID"
// @Success 204
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /alerts/{id} [delete]
// @Security BearerAuth
func (h *AlertHandler) DeleteAlert(c echo.Context) error {
	tenantID := c.Get("tenant_id").(uuid.UUID)
	alertID := c.Param("id")

	var alert models.Alert
	if err := h.db.Where("id = ? AND tenant_id = ?", alertID, tenantID).Preload("Channel").First(&alert).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.JSON(http.StatusNotFound, map[string]string{
				"error": "Alert not found",
			})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to fetch alert",
		})
	}

	// Delete WhatsApp group if it exists
	if alert.GroupID != "" && alert.Channel.Session != "" {
		if err := h.alertService.DeleteWhatsAppGroup(alert.GroupID, alert.Channel.Session); err != nil {
			// Log the error but don't fail the alert deletion
			c.Logger().Errorf("Failed to delete WhatsApp group %s for alert %s: %v", alert.GroupID, alert.ID, err)
		}
	}

	if err := h.db.Delete(&alert).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to delete alert",
		})
	}

	return c.NoContent(http.StatusNoContent)
}
