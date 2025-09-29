package repo

import (
	"iafarma/pkg/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ChannelRepository handles channel data access
type ChannelRepository struct {
	db *gorm.DB
}

// NewChannelRepository creates a new channel repository
func NewChannelRepository(db *gorm.DB) *ChannelRepository {
	return &ChannelRepository{db: db}
}

// GetByID gets a channel by ID
func (r *ChannelRepository) GetByID(id uuid.UUID) (*models.Channel, error) {
	var channel models.Channel
	err := r.db.Where("id = ?", id).First(&channel).Error
	if err != nil {
		return nil, err
	}
	return &channel, nil
}

// GetByIDAndTenant gets a channel by ID and tenant ID for security
func (r *ChannelRepository) GetByIDAndTenant(id, tenantID uuid.UUID) (*models.Channel, error) {
	var channel models.Channel
	err := r.db.Where("id = ? AND tenant_id = ?", id, tenantID).First(&channel).Error
	if err != nil {
		return nil, err
	}
	return &channel, nil
}

// Create creates a new channel
func (r *ChannelRepository) Create(channel *models.Channel) error {
	return r.db.Create(channel).Error
}

// GetBySession gets a channel by session (globally, not tenant-specific)
func (r *ChannelRepository) GetBySession(session string) (*models.Channel, error) {
	var channel models.Channel
	err := r.db.Where("session = ?", session).First(&channel).Error
	if err != nil {
		return nil, err
	}
	return &channel, nil
}

// SessionExists checks if a session already exists globally
func (r *ChannelRepository) SessionExists(session string) (bool, error) {
	var count int64
	err := r.db.Model(&models.Channel{}).Where("session = ?", session).Count(&count).Error
	return count > 0, err
}

// Update updates a channel
func (r *ChannelRepository) Update(channel *models.Channel) error {
	return r.db.Save(channel).Error
}

// Delete deletes a channel (soft delete)
func (r *ChannelRepository) Delete(id, tenantID uuid.UUID) error {
	result := r.db.Where("id = ? AND tenant_id = ?", id, tenantID).Delete(&models.Channel{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// HardDelete permanently deletes a channel (for admin purposes)
func (r *ChannelRepository) HardDelete(id, tenantID uuid.UUID) error {
	result := r.db.Unscoped().Where("id = ? AND tenant_id = ?", id, tenantID).Delete(&models.Channel{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// MigrateConversations migrates all conversations from source channel to destination channel
func (r *ChannelRepository) MigrateConversations(sourceChannelID, destinationChannelID, tenantID uuid.UUID) (int64, error) {
	// Use a transaction to ensure data consistency
	tx := r.db.Begin()
	if tx.Error != nil {
		return 0, tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Count conversations to be migrated
	var count int64
	err := tx.Model(&models.Conversation{}).
		Where("channel_id = ? AND tenant_id = ?", sourceChannelID, tenantID).
		Count(&count).Error
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	// Update all conversations from source channel to destination channel
	result := tx.Model(&models.Conversation{}).
		Where("channel_id = ? AND tenant_id = ?", sourceChannelID, tenantID).
		Update("channel_id", destinationChannelID)

	if result.Error != nil {
		tx.Rollback()
		return 0, result.Error
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		return 0, err
	}

	return count, nil
}

// ListByTenant lists channels for a specific tenant with pagination and conversation count
func (r *ChannelRepository) ListByTenant(tenantID uuid.UUID, limit, offset int) (models.PaginationResult[models.ChannelWithConversationCount], error) {
	var channels []models.ChannelWithConversationCount
	var total int64

	// Get total count for tenant
	if err := r.db.Model(&models.Channel{}).Where("tenant_id = ?", tenantID).Count(&total).Error; err != nil {
		return models.PaginationResult[models.ChannelWithConversationCount]{}, err
	}

	// Get channels with conversation count using a join query
	query := `
		SELECT 
			c.*,
			COALESCE(COUNT(conv.id), 0) as conversation_count
		FROM channels c
		LEFT JOIN conversations conv ON c.id = conv.channel_id AND conv.tenant_id = c.tenant_id
		WHERE c.tenant_id = ? AND c.deleted_at IS NULL
		GROUP BY c.id
		ORDER BY c.created_at DESC
		LIMIT ? OFFSET ?
	`

	err := r.db.Raw(query, tenantID, limit, offset).Scan(&channels).Error
	if err != nil {
		return models.PaginationResult[models.ChannelWithConversationCount]{}, err
	}

	page := (offset / limit) + 1
	if limit == 0 {
		page = 1
	}

	totalPages := int((total + int64(limit) - 1) / int64(limit))
	if limit == 0 {
		totalPages = 1
	}

	return models.PaginationResult[models.ChannelWithConversationCount]{
		Data:       channels,
		Total:      total,
		Page:       page,
		PerPage:    limit,
		TotalPages: totalPages,
	}, nil
}

// CountConversationsByChannel counts conversations for a specific channel
func (r *ChannelRepository) CountConversationsByChannel(channelID, tenantID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.Model(&models.Conversation{}).
		Where("channel_id = ? AND tenant_id = ?", channelID, tenantID).
		Count(&count).Error
	return count, err
}

// MessageRepository handles message data access
type MessageRepository struct {
	db *gorm.DB
}

// NewMessageRepository creates a new message repository
func NewMessageRepository(db *gorm.DB) *MessageRepository {
	return &MessageRepository{db: db}
}

// GetByID gets a message by ID
func (r *MessageRepository) GetByID(id uuid.UUID) (*models.Message, error) {
	var message models.Message
	err := r.db.Preload("Customer").Preload("User").Preload("Media").
		Where("id = ?", id).First(&message).Error
	if err != nil {
		return nil, err
	}
	return &message, nil
}

// Create creates a new message
func (r *MessageRepository) Create(message *models.Message) error {
	return r.db.Create(message).Error
}

// Update updates a message
func (r *MessageRepository) Update(message *models.Message) error {
	return r.db.Save(message).Error
}

// ListByConversation lists messages by conversation ID
func (r *MessageRepository) ListByConversation(conversationID uuid.UUID, limit, offset int) ([]models.Message, error) {
	var messages []models.Message
	err := r.db.Where("conversation_id = ?", conversationID).
		Order("created_at DESC").
		Limit(limit).Offset(offset).
		Find(&messages).Error
	return messages, err
}

// GetUnreadCountByTenant gets the total count of unread messages for a tenant
func (r *MessageRepository) GetUnreadCountByTenant(tenantID uuid.UUID) (int64, error) {
	var count int64
	// fmt.Printf("DEBUG GetUnreadCountByTenant - Tenant ID: %s\n", tenantID)

	err := r.db.Table("conversations").
		Where("tenant_id = ?", tenantID).
		Select("COALESCE(SUM(unread_count), 0)").
		Scan(&count).Error

	// fmt.Printf("DEBUG GetUnreadCountByTenant - Count: %d, Error: %v\n", count, err)
	return count, err
}
