package ai

import (
	"iafarma/pkg/models"

	"github.com/google/uuid"
)

// AlertServiceWrapper implements AlertServiceInterface to avoid circular dependencies
type AlertServiceWrapper struct {
	sendOrderAlertFunc        func(tenantID uuid.UUID, order *models.Order, customerPhone string) error
	sendHumanSupportAlertFunc func(tenantID uuid.UUID, customerID uuid.UUID, customerPhone string, reason string) error
}

func (a *AlertServiceWrapper) SendOrderAlert(tenantID uuid.UUID, order *models.Order, customerPhone string) error {
	if a.sendOrderAlertFunc != nil {
		return a.sendOrderAlertFunc(tenantID, order, customerPhone)
	}
	return nil
}

func (a *AlertServiceWrapper) SendHumanSupportAlert(tenantID uuid.UUID, customerID uuid.UUID, customerPhone string, reason string) error {
	if a.sendHumanSupportAlertFunc != nil {
		return a.sendHumanSupportAlertFunc(tenantID, customerID, customerPhone, reason)
	}
	return nil
}

// SetSendOrderAlertFunc sets the alert function after creation
func (a *AlertServiceWrapper) SetSendOrderAlertFunc(fn func(tenantID uuid.UUID, order *models.Order, customerPhone string) error) {
	a.sendOrderAlertFunc = fn
}

// SetSendHumanSupportAlertFunc sets the human support alert function after creation
func (a *AlertServiceWrapper) SetSendHumanSupportAlertFunc(fn func(tenantID uuid.UUID, customerID uuid.UUID, customerPhone string, reason string) error) {
	a.sendHumanSupportAlertFunc = fn
}

// NewAlertServiceWrapper creates a new alert service wrapper
func NewAlertServiceWrapper(sendOrderAlertFunc func(tenantID uuid.UUID, order *models.Order, customerPhone string) error) *AlertServiceWrapper {
	return &AlertServiceWrapper{
		sendOrderAlertFunc: sendOrderAlertFunc,
	}
}
