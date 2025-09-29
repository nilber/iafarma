package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"iafarma/internal/repo"
	"iafarma/internal/services"
	"iafarma/pkg/models"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

// MigrationResponse represents the response for conversation migration
type MigrationResponse struct {
	MigratedCount int    `json:"migrated_count"`
	Message       string `json:"message"`
}

// ChannelHandler handles channel operations
type ChannelHandler struct {
	channelRepo      *repo.ChannelRepository
	planLimitService *services.PlanLimitService
}

// NewChannelHandler creates a new channel handler
func NewChannelHandler(channelRepo *repo.ChannelRepository, planLimitService *services.PlanLimitService) *ChannelHandler {
	return &ChannelHandler{
		channelRepo:      channelRepo,
		planLimitService: planLimitService,
	}
}

// List godoc
// @Summary List channels
// @Description Get a list of channels with pagination
// @Tags channels
// @Accept json
// @Produce json
// @Param limit query int false "Limit" default(20)
// @Param offset query int false "Offset" default(0)
// @Success 200 {object} models.ChannelListResponse
// @Failure 500 {object} map[string]string
// @Router /channels [get]
// @Security BearerAuth
func (h *ChannelHandler) List(c echo.Context) error {
	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	offset, _ := strconv.Atoi(c.QueryParam("offset"))

	if limit <= 0 {
		limit = 20
	}

	tenantID, ok := c.Get("tenant_id").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "tenant_id not found in context"})
	}

	result, err := h.channelRepo.ListByTenant(tenantID, limit, offset)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, result)
}

// GetByID godoc
// @Summary Get channel by ID
// @Description Get a channel by its ID
// @Tags channels
// @Accept json
// @Produce json
// @Param id path string true "Channel ID"
// @Success 200 {object} models.Channel
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /channels/{id} [get]
// @Security BearerAuth
func (h *ChannelHandler) GetByID(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid ID format"})
	}

	tenantID, ok := c.Get("tenant_id").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "tenant_id not found in context"})
	}

	channel, err := h.channelRepo.GetByIDAndTenant(id, tenantID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Channel not found"})
	}

	return c.JSON(http.StatusOK, channel)
}

// Create godoc
// @Summary Create channel
// @Description Create a new channel
// @Tags channels
// @Accept json
// @Produce json
// @Param channel body models.Channel true "Channel data"
// @Success 201 {object} models.Channel
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /channels [post]
// @Security BearerAuth
func (h *ChannelHandler) Create(c echo.Context) error {
	tenantID, ok := c.Get("tenant_id").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "tenant_id not found in context"})
	}

	// Validate plan limits for channels
	if h.planLimitService != nil {
		if err := h.planLimitService.ValidateChannelLimit(tenantID, 1); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
		}
	}

	var channel models.Channel
	if err := c.Bind(&channel); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request"})
	}

	// Validate required fields
	if channel.Session == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Session is required"})
	}

	// Check if session already exists globally (across all tenants)
	sessionExists, err := h.channelRepo.SessionExists(channel.Session)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Error checking session uniqueness"})
	}

	if sessionExists {
		return c.JSON(http.StatusConflict, map[string]string{
			"error": "Session already exists. Please choose a different session name.",
		})
	}

	// Force tenant_id from JWT, ignore any value from frontend
	channel.TenantID = tenantID
	// Generate new ID to prevent ID injection
	channel.ID = uuid.New()

	if err := h.channelRepo.Create(&channel); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, channel)
}

// Update godoc
// @Summary Update channel
// @Description Update an existing channel
// @Tags channels
// @Accept json
// @Produce json
// @Param id path string true "Channel ID"
// @Param channel body models.Channel true "Channel data"
// @Success 200 {object} models.Channel
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /channels/{id} [put]
// @Security BearerAuth
func (h *ChannelHandler) Update(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid ID format"})
	}

	tenantID, ok := c.Get("tenant_id").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "tenant_id not found in context"})
	}

	existingChannel, err := h.channelRepo.GetByIDAndTenant(id, tenantID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Channel not found"})
	}

	var updateData models.Channel
	if err := c.Bind(&updateData); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request"})
	}

	// Merge only non-empty fields from the update data into existing channel
	if updateData.Name != "" {
		existingChannel.Name = updateData.Name
	}
	if updateData.Type != "" {
		existingChannel.Type = updateData.Type
	}
	if updateData.Session != "" {
		existingChannel.Session = updateData.Session
	}
	if updateData.Status != "" {
		existingChannel.Status = updateData.Status
	}
	if updateData.Config != "" {
		existingChannel.Config = updateData.Config
	}
	if updateData.WebhookURL != "" {
		existingChannel.WebhookURL = updateData.WebhookURL
	}
	if updateData.WebhookToken != "" {
		existingChannel.WebhookToken = updateData.WebhookToken
	}

	// Always update IsActive when it's provided in the request (boolean field)
	// We can't check if it's "empty" like strings, so we check if there's any update intent
	if c.Request().Body != nil {
		existingChannel.IsActive = updateData.IsActive
	}

	// Update timestamp
	existingChannel.UpdatedAt = time.Now()

	if err := h.channelRepo.Update(existingChannel); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, existingChannel)
}

// Delete deletes a channel
// @Summary Delete a channel
// @Description Delete a channel by ID (only if it has no conversations)
// @Tags channels
// @Accept json
// @Produce json
// @Param id path string true "Channel ID"
// @Success 204
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 409 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /channels/{id} [delete]
// @Security BearerAuth
func (h *ChannelHandler) Delete(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid ID format"})
	}

	tenantID, ok := c.Get("tenant_id").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "tenant_id not found in context"})
	}

	// Check if channel exists
	_, err = h.channelRepo.GetByIDAndTenant(id, tenantID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "Channel not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	// Check if channel has conversations
	conversationCount, err := h.channelRepo.CountConversationsByChannel(id, tenantID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	if conversationCount > 0 {
		return c.JSON(http.StatusConflict, map[string]string{
			"error": fmt.Sprintf("Cannot delete channel with %d conversations. Please migrate conversations to another channel first.", conversationCount),
		})
	}

	if err := h.channelRepo.Delete(id, tenantID); err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "Channel not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.NoContent(http.StatusNoContent)
}

// MigrateConversations migrates all conversations from one channel to another
// @Summary Migrate conversations between channels
// @Description Migrate all conversations from source channel to destination channel
// @Tags channels
// @Accept json
// @Produce json
// @Param id path string true "Source Channel ID"
// @Param request body object{destination_channel_id: string} true "Migration request"
// @Success 200 {object} MigrationResponse
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /channels/{id}/migrate-conversations [post]
// @Security BearerAuth
func (h *ChannelHandler) MigrateConversations(c echo.Context) error {
	sourceChannelID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid source channel ID format"})
	}

	tenantID, ok := c.Get("tenant_id").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "tenant_id not found in context"})
	}

	// Parse request body
	var request struct {
		DestinationChannelID string `json:"destination_channel_id" validate:"required"`
	}
	if err := c.Bind(&request); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request"})
	}

	destinationChannelID, err := uuid.Parse(request.DestinationChannelID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid destination channel ID format"})
	}

	// Verify both channels exist and belong to the tenant
	sourceChannel, err := h.channelRepo.GetByIDAndTenant(sourceChannelID, tenantID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "Source channel not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	destinationChannel, err := h.channelRepo.GetByIDAndTenant(destinationChannelID, tenantID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "Destination channel not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	// Prevent migration to the same channel
	if sourceChannelID == destinationChannelID {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Source and destination channels cannot be the same"})
	}

	// Only allow migration if source channel is inactive/disconnected
	if sourceChannel.IsActive && sourceChannel.Status == "connected" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Cannot migrate conversations from an active/connected channel. Please disconnect the channel first.",
		})
	}

	// Perform the migration
	migratedCount, err := h.channelRepo.MigrateConversations(sourceChannelID, destinationChannelID, tenantID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"migrated_count": migratedCount,
		"message":        fmt.Sprintf("Successfully migrated %d conversations from '%s' to '%s'", migratedCount, sourceChannel.Name, destinationChannel.Name),
	})
}

// MessageHandler handles message operations
type MessageHandler struct {
	messageRepo *repo.MessageRepository
	db          *gorm.DB
}

// NewMessageHandler creates a new message handler
func NewMessageHandler(messageRepo *repo.MessageRepository, db *gorm.DB) *MessageHandler {
	return &MessageHandler{
		messageRepo: messageRepo,
		db:          db,
	}
}

// ListByConversation godoc
// @Summary List messages by conversation
// @Description Get list of messages for a specific conversation with pagination
// @Tags messages
// @Accept json
// @Produce json
// @Param conversation_id query string true "Conversation ID"
// @Param limit query int false "Limit" default(50)
// @Param offset query int false "Offset" default(0)
// @Success 200 {array} models.Message
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /messages [get]
// @Security BearerAuth
func (h *MessageHandler) ListByConversation(c echo.Context) error {
	conversationIDStr := c.QueryParam("conversation_id")
	if conversationIDStr == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "conversation_id is required"})
	}

	conversationID, err := uuid.Parse(conversationIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid conversation ID format"})
	}

	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	offset, _ := strconv.Atoi(c.QueryParam("offset"))

	if limit <= 0 {
		limit = 50
	}

	messages, err := h.messageRepo.ListByConversation(conversationID, limit, offset)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, messages)
}

// GetByID godoc
// @Summary Get message by ID
// @Description Get a message by its ID
// @Tags messages
// @Accept json
// @Produce json
// @Param id path string true "Message ID"
// @Success 200 {object} models.Message
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /messages/{id} [get]
// @Security BearerAuth
func (h *MessageHandler) GetByID(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid ID format"})
	}

	message, err := h.messageRepo.GetByID(id)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Message not found"})
	}

	return c.JSON(http.StatusOK, message)
}

// Create godoc
// @Summary Create message
// @Description Create a new message
// @Tags messages
// @Accept json
// @Produce json
// @Param message body models.Message true "Message data"
// @Success 201 {object} models.Message
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /messages [post]
// @Security BearerAuth
func (h *MessageHandler) Create(c echo.Context) error {
	var message models.Message
	if err := c.Bind(&message); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request"})
	}

	// Get tenant ID from context
	tenantID, ok := c.Get("tenant_id").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "tenant_id not found in context"})
	}

	// Set tenant ID for the message
	message.TenantID = tenantID

	// If this is an outgoing message and has a user ID, get the user name
	if message.Direction == "out" && message.UserID != nil {
		var user models.User
		if err := h.db.Select("name").Where("id = ? AND tenant_id = ?", *message.UserID, tenantID).First(&user).Error; err == nil {
			message.UserName = user.Name
		}
	}

	// If no user name is set and it's an outgoing message, default to AI
	if message.Direction == "out" && message.UserName == "" {
		message.UserName = "Assistente IA"
	}

	if err := h.messageRepo.Create(&message); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, message)
}

// CreateNote godoc
// @Summary Create internal note
// @Description Create an internal note for a conversation
// @Tags messages
// @Accept json
// @Produce json
// @Param note body CreateNoteRequest true "Note data"
// @Success 201 {object} models.Message
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /messages/notes [post]
// @Security BearerAuth
func (h *MessageHandler) CreateNote(c echo.Context) error {
	var req CreateNoteRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request"})
	}

	if req.ConversationID == uuid.Nil || req.Content == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "conversation_id and content are required"})
	}

	// Get tenant ID and user ID from context
	tenantID, ok := c.Get("tenant_id").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "tenant_id not found in context"})
	}

	userID, ok := c.Get("user_id").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "user_id not found in context"})
	}

	// Get user name
	var user models.User
	if err := h.db.Select("name").Where("id = ? AND tenant_id = ?", userID, tenantID).First(&user).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to get user information"})
	}

	// Get customer ID from conversation
	var conversation models.Conversation
	if err := h.db.Select("customer_id").Where("id = ? AND tenant_id = ?", req.ConversationID, tenantID).First(&conversation).Error; err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Conversation not found"})
	}

	// Create the note as a special message
	note := models.Message{
		ConversationID: req.ConversationID,
		CustomerID:     conversation.CustomerID,
		UserID:         &userID,
		UserName:       user.Name,
		Type:           "text",
		Content:        req.Content,
		Direction:      "note", // Special direction for notes
		Status:         "sent",
		IsNote:         true, // Mark as internal note
	}
	note.TenantID = tenantID

	if err := h.messageRepo.Create(&note); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, note)
}

// CreateNoteRequest represents the request payload for creating a note
type CreateNoteRequest struct {
	ConversationID uuid.UUID `json:"conversation_id" validate:"required"`
	Content        string    `json:"content" validate:"required"`
}
