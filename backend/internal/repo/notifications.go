package repo

import (
	"iafarma/pkg/models"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// NotificationRepository handles notification data access
type NotificationRepository struct {
	db *gorm.DB
}

// NewNotificationRepository creates a new notification repository
func NewNotificationRepository(db *gorm.DB) *NotificationRepository {
	return &NotificationRepository{db: db}
}

// CreateLog creates a new notification log
func (r *NotificationRepository) CreateLog(log *models.NotificationLog) error {
	return r.db.Create(log).Error
}

// IsNotificationSentToday checks if a notification was already sent today
func (r *NotificationRepository) IsNotificationSentToday(tenantID uuid.UUID, notificationType, date string) bool {
	var count int64
	parsedDate, err := time.Parse("2006-01-02", date)
	if err != nil {
		return false
	}

	r.db.Model(&models.NotificationControl{}).
		Where("tenant_id = ? AND notification_type = ? AND reference_date = ? AND sent = ?",
			tenantID, notificationType, parsedDate, true).
		Count(&count)

	return count > 0
}

// MarkNotificationSent marks a notification as sent for a specific date
func (r *NotificationRepository) MarkNotificationSent(tenantID uuid.UUID, notificationType, date string) error {
	parsedDate, err := time.Parse("2006-01-02", date)
	if err != nil {
		return err
	}

	now := time.Now()
	control := models.NotificationControl{
		BaseTenantModel: models.BaseTenantModel{
			TenantID: tenantID,
		},
		NotificationType: notificationType,
		ReferenceDate:    parsedDate,
		Sent:             true,
		SentAt:           &now,
	}

	// Use ON CONFLICT to handle duplicates
	return r.db.Create(&control).Error
}

// GetScheduledNotifications gets active scheduled notifications
func (r *NotificationRepository) GetScheduledNotifications(tenantID *uuid.UUID) ([]models.ScheduledNotification, error) {
	var notifications []models.ScheduledNotification
	query := r.db.Where("is_active = ?", true)

	if tenantID != nil {
		query = query.Where("tenant_id = ?", *tenantID)
	}

	err := query.Preload("Template").Find(&notifications).Error
	return notifications, err
}

// UpdateScheduledNotification updates a scheduled notification
func (r *NotificationRepository) UpdateScheduledNotification(notification *models.ScheduledNotification) error {
	return r.db.Save(notification).Error
}

// GetNotificationLogs gets notification logs with pagination
func (r *NotificationRepository) GetNotificationLogs(tenantID uuid.UUID, limit, offset int) ([]models.NotificationLog, error) {
	var logs []models.NotificationLog
	err := r.db.Where("tenant_id = ?", tenantID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&logs).Error
	return logs, err
}

// GetAllNotificationLogs gets all notification logs for super admin with pagination and filters
func (r *NotificationRepository) GetAllNotificationLogs(tenantID *uuid.UUID, notificationType, status string, limit, offset int) ([]models.NotificationLog, int64, error) {
	var logs []models.NotificationLog
	var total int64

	query := r.db.Model(&models.NotificationLog{})

	// Apply filters
	if tenantID != nil {
		query = query.Where("tenant_id = ?", *tenantID)
	}
	if notificationType != "" {
		query = query.Where("type = ?", notificationType)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}

	// Get total count
	query.Count(&total)

	// Get paginated results with tenant info
	err := query.
		Select("notification_logs.*, tenants.name as tenant_name").
		Joins("LEFT JOIN tenants ON notification_logs.tenant_id = tenants.id").
		Order("notification_logs.created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&logs).Error

	return logs, total, err
}

// GetNotificationStats gets notification statistics for super admin
func (r *NotificationRepository) GetNotificationStats(tenantID *uuid.UUID) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	query := r.db.Model(&models.NotificationLog{})
	if tenantID != nil {
		query = query.Where("tenant_id = ?", *tenantID)
	}

	// Total notifications
	var total int64
	query.Count(&total)
	stats["total"] = total

	// By status
	var statusStats []struct {
		Status string `json:"status"`
		Count  int64  `json:"count"`
	}
	query.Select("status, COUNT(*) as count").Group("status").Find(&statusStats)
	stats["by_status"] = statusStats

	// By type
	var typeStats []struct {
		Type  string `json:"type"`
		Count int64  `json:"count"`
	}
	query.Select("type, COUNT(*) as count").Group("type").Find(&typeStats)
	stats["by_type"] = typeStats

	// Today's notifications
	var todayCount int64
	today := time.Now().Format("2006-01-02")
	queryToday := r.db.Model(&models.NotificationLog{})
	if tenantID != nil {
		queryToday = queryToday.Where("tenant_id = ?", *tenantID)
	}
	queryToday.Where("DATE(created_at) = ?", today).Count(&todayCount)
	stats["today"] = todayCount

	// Failed notifications in last 24h
	var failedCount int64
	last24h := time.Now().Add(-24 * time.Hour)
	queryFailed := r.db.Model(&models.NotificationLog{})
	if tenantID != nil {
		queryFailed = queryFailed.Where("tenant_id = ?", *tenantID)
	}
	queryFailed.Where("status = ? AND created_at >= ?", "failed", last24h).Count(&failedCount)
	stats["failed_24h"] = failedCount

	return stats, nil
}

// GetNotificationLogByID gets a specific notification log
func (r *NotificationRepository) GetNotificationLogByID(id uuid.UUID) (*models.NotificationLog, error) {
	var log models.NotificationLog
	err := r.db.
		Select("notification_logs.*, tenants.name as tenant_name").
		Joins("LEFT JOIN tenants ON notification_logs.tenant_id = tenants.id").
		Where("notification_logs.id = ?", id).
		First(&log).Error

	if err != nil {
		return nil, err
	}
	return &log, nil
}
