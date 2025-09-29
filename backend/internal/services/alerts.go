package services

import (
	"fmt"
	"strings"

	"iafarma/internal/zapplus"
	"iafarma/pkg/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// AlertService handles alert notifications
type AlertService struct {
	db *gorm.DB
}

// NewAlertService creates a new alert service
func NewAlertService(db *gorm.DB) *AlertService {
	return &AlertService{db: db}
}

// CreateWhatsAppGroup creates a WhatsApp group via ZapPlus integration
func (s *AlertService) CreateWhatsAppGroup(alert *models.Alert, channelSession string) error {
	if alert.Phones == "" {
		return fmt.Errorf("no phones specified for group creation")
	}

	// Parse phones from comma-separated string
	phoneList := strings.Split(alert.Phones, ",")
	var participants []string
	for _, phone := range phoneList {
		cleanPhone := strings.TrimSpace(phone)
		if cleanPhone != "" {
			participants = append(participants, cleanPhone)
		}
	}

	if len(participants) == 0 {
		return fmt.Errorf("no valid phones found")
	}

	// Use centralized ZapPlus client
	client := zapplus.GetClient()
	response, err := client.CreateGroup(channelSession, alert.GroupName, participants)
	if err != nil {
		return fmt.Errorf("failed to create group: %w", err)
	}

	// Save the group serialized ID for sending messages
	alert.GroupID = response.GID.Serialized
	if err := s.db.Save(alert).Error; err != nil {
		return fmt.Errorf("failed to save group ID: %w", err)
	}

	// Enable messaging for all participants (disable admin-only mode)
	if err := client.SetGroupMessagesAdminOnly(channelSession, response.GID.Serialized, false); err != nil {
		fmt.Printf("Warning: Failed to enable group messaging: %v\n", err)
		// Don't fail the entire operation for this
	}

	return nil
}

// SendOrderAlert sends an order notification to the alert group
func (s *AlertService) SendOrderAlert(tenantID uuid.UUID, order *models.Order, customerPhone string) error {
	// Use the centralized notification service for sending order alerts
	notificationService := zapplus.NewNotificationService(s.db)
	return notificationService.SendOrderAlert(tenantID, order, customerPhone)
}

// AddParticipantToGroup adds a participant to an existing WhatsApp group
func (s *AlertService) AddParticipantToGroup(groupID, phone, session string) error {
	// Use centralized ZapPlus client
	client := zapplus.GetClient()
	return client.AddParticipantToGroup(session, groupID, phone)
}

// RemoveParticipantFromGroup removes a participant from an existing WhatsApp group
func (s *AlertService) RemoveParticipantFromGroup(groupID, phone, session string) error {
	// Use centralized ZapPlus client
	client := zapplus.GetClient()
	return client.RemoveParticipantFromGroup(session, groupID, phone)
}

// GetActiveAlertsForChannel returns active alerts for a specific channel
func (s *AlertService) GetActiveAlertsForChannel(tenantID, channelID uuid.UUID) ([]models.Alert, error) {
	var alerts []models.Alert
	err := s.db.Where("channel_id = ? AND tenant_id = ? AND is_active = ?", channelID, tenantID, true).
		Preload("Channel").
		Find(&alerts).Error
	return alerts, err
}

// DeleteWhatsAppGroup deletes a WhatsApp group via ZapPlus API
func (s *AlertService) DeleteWhatsAppGroup(groupID, channelSession string) error {
	if groupID == "" || channelSession == "" {
		return fmt.Errorf("group ID and channel session are required")
	}

	// Use centralized ZapPlus client
	client := zapplus.GetClient()
	return client.DeleteGroup(channelSession, groupID)
}
