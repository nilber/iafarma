package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// NotificationTemplate represents a notification template
type NotificationTemplate struct {
	BaseModel
	Name      string `gorm:"not null" json:"name" validate:"required"`
	Type      string `gorm:"not null" json:"type"` // 'email', 'sms', 'push'
	Subject   string `json:"subject"`
	Body      string `gorm:"type:text;not null" json:"body" validate:"required"`
	Variables string `gorm:"type:text" json:"variables"` // JSON array of variable names
	IsActive  bool   `gorm:"default:true" json:"is_active"`
}

// ScheduledNotification represents a scheduled notification
type ScheduledNotification struct {
	BaseTenantModel
	Type            string     `gorm:"not null" json:"type"`           // 'daily_sales_report', 'low_stock_alert', 'custom'
	ScheduleType    string     `gorm:"not null" json:"schedule_type"`  // 'daily', 'weekly', 'monthly', 'cron'
	ScheduleValue   string     `gorm:"not null" json:"schedule_value"` // cron expression or simple schedule
	NextRun         *time.Time `json:"next_run"`
	LastRun         *time.Time `json:"last_run"`
	LastRunStatus   string     `json:"last_run_status"`                   // 'success', 'failed', 'skipped'
	RecipientEmails string     `gorm:"type:text" json:"recipient_emails"` // JSON array of emails
	TemplateID      uuid.UUID  `gorm:"type:uuid;constraint:OnDelete:RESTRICT" json:"template_id"`
	IsActive        bool       `gorm:"default:true" json:"is_active"`

	// Relationships
	Template NotificationTemplate `gorm:"foreignKey:TemplateID" json:"template,omitempty"`
}

// NotificationLog represents a log of sent notifications
type NotificationLog struct {
	BaseTenantModel
	ScheduledNotificationID *uuid.UUID `gorm:"type:uuid;constraint:OnDelete:SET NULL" json:"scheduled_notification_id,omitempty"`
	Type                    string     `gorm:"not null" json:"type"`
	Recipient               string     `gorm:"not null" json:"recipient"`
	Subject                 string     `json:"subject"`
	Body                    string     `gorm:"type:text" json:"body"`
	Status                  string     `gorm:"not null" json:"status"` // 'sent', 'failed', 'pending'
	SentAt                  *time.Time `json:"sent_at"`
	ErrorMessage            string     `json:"error_message,omitempty"`

	// Relationships
	ScheduledNotification *ScheduledNotification `gorm:"foreignKey:ScheduledNotificationID" json:"scheduled_notification,omitempty"`
}

// NotificationControl represents control to prevent duplicate notifications
type NotificationControl struct {
	BaseTenantModel
	NotificationType string     `gorm:"not null;index" json:"notification_type"`
	ReferenceDate    time.Time  `gorm:"not null;index" json:"reference_date"` // Date for which notification was sent
	Sent             bool       `gorm:"default:false" json:"sent"`
	SentAt           *time.Time `json:"sent_at"`

	// Composite unique constraint to prevent duplicates
	// UniqueIndex: tenant_id + notification_type + reference_date
}

// BeforeCreate hook for NotificationControl
func (nc *NotificationControl) BeforeCreate(tx *gorm.DB) error {
	// Generate UUID if not set
	if nc.ID == uuid.Nil {
		nc.ID = uuid.New()
	}
	return nil
}
