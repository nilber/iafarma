package handlers

import (
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"iafarma/internal/services"
	"iafarma/internal/whatsapp"
	"iafarma/internal/zapplus"
	"iafarma/pkg/models"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

// WhatsAppHandler handles WhatsApp-related operations
type WhatsAppHandler struct {
	db             *gorm.DB
	client         *whatsapp.Client
	wsHandler      *WebSocketHandler
	storageService *services.StorageService
}

// NewWhatsAppHandler creates a new WhatsApp handler
func NewWhatsAppHandler(db *gorm.DB, client *whatsapp.Client, storageService *services.StorageService) *WhatsAppHandler {
	return &WhatsAppHandler{
		db:             db,
		client:         client,
		storageService: storageService,
	}
}

// SetWebSocketHandler sets the WebSocket handler for real-time notifications
func (h *WhatsAppHandler) SetWebSocketHandler(wsHandler *WebSocketHandler) {
	h.wsHandler = wsHandler
}

// ExternalWhatsAppRequest represents a request to external WhatsApp API
type ExternalWhatsAppRequest struct {
	ChatID                 string `json:"chatId"`
	ReplyTo                string `json:"reply_to,omitempty"`
	Text                   string `json:"text"`
	LinkPreview            bool   `json:"linkPreview"`
	LinkPreviewHighQuality bool   `json:"linkPreviewHighQuality"`
	Session                string `json:"session"`
}

// ZapPlusResponse represents the response from ZapPlus API
type ZapPlusResponse struct {
	Data struct {
		ID struct {
			ID string `json:"id"`
		} `json:"id"`
	} `json:"_data"`
}

// formatPhoneForWhatsApp formats a phone number for WhatsApp API
func formatPhoneForWhatsApp(phone string) string {
	// Clean phone number - remove formatting
	cleanPhone := phone
	// Add @c.us if not present
	if !strings.Contains(cleanPhone, "@c.us") {
		cleanPhone = strings.ReplaceAll(cleanPhone, "(", "")
		cleanPhone = strings.ReplaceAll(cleanPhone, ")", "")
		cleanPhone = strings.ReplaceAll(cleanPhone, " ", "")
		cleanPhone = strings.ReplaceAll(cleanPhone, "-", "")
		cleanPhone = strings.ReplaceAll(cleanPhone, "+", "")
		cleanPhone = cleanPhone + "@c.us"
	}
	return cleanPhone
}

// sendViaExternalAPI sends a message via external WhatsApp API and returns the external ID
func (h *WhatsAppHandler) sendViaExternalAPI(session, phone, text string) (*string, error) {
	if session == "" {
		return nil, fmt.Errorf("channel session not configured")
	}

	// Format phone number for WhatsApp
	cleanPhone := formatPhoneForWhatsApp(phone)

	fmt.Printf("\n\n\nSending WhatsApp message to: %s via session: %s\n", cleanPhone, session)

	// Use centralized ZapPlus client
	client := zapplus.GetClient()
	err := client.SendTextMessage(session, cleanPhone, text)
	if err != nil {
		return nil, fmt.Errorf("failed to send message via ZapPlus: %w", err)
	}

	// Note: The centralized client doesn't return external ID yet
	// This is a simplified implementation - for full compatibility,
	// we might need to extend the client to return response data
	externalID := "sent_via_centralized_client"
	return &externalID, nil
}

// SendMessageRequest represents a message send request
type SendMessageRequest struct {
	ConversationID  string  `json:"conversation_id" validate:"required"`
	Type            string  `json:"type" validate:"required,oneof=text image audio video document"`
	Content         string  `json:"content" validate:"required"`
	ReplyToID       *string `json:"reply_to_id,omitempty"`
	ResendMessageID *string `json:"resend_message_id,omitempty"` // ID of message to resend instead of creating new
}

// GetStatus godoc
// @Summary Get WhatsApp status
// @Description Get the current status of the WhatsApp connection
// @Tags whatsapp
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]string
// @Router /whatsapp/status [get]
// @Security BearerAuth
func (h *WhatsAppHandler) GetStatus(c echo.Context) error {
	status := map[string]interface{}{
		"connected": false,
		"message":   "WhatsApp client not initialized",
	}

	if h.client != nil {
		// TODO: Implement actual client status check
		status["connected"] = true
		status["message"] = "WhatsApp client connected"
	}

	return c.JSON(http.StatusOK, status)
}

// SendMessage godoc
// @Summary Send WhatsApp message
// @Description Send a message through WhatsApp
// @Tags whatsapp
// @Accept json
// @Produce json
// @Param request body SendMessageRequest true "Send message request"
// @Success 200 {object} models.Message
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /whatsapp/send [post]
// @Security BearerAuth
func (h *WhatsAppHandler) SendMessage(c echo.Context) error {
	var req SendMessageRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request payload",
		})
	}

	// Parse conversation ID
	conversationID, err := uuid.Parse(req.ConversationID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid conversation ID",
		})
	}

	// Get user ID from context (from JWT middleware)
	userIDInterface := c.Get("user_id")
	if userIDInterface == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "User ID not found in context",
		})
	}
	userID, ok := userIDInterface.(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Invalid user ID format",
		})
	}

	// Get conversation details
	var conversation models.Conversation
	if err := h.db.First(&conversation, conversationID).Error; err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Conversation not found",
		})
	}

	// Get user name for the message
	var user models.User
	userName := "Assistente IA" // default name
	if err := h.db.Select("name").Where("id = ? AND tenant_id = ?", userID, conversation.TenantID).First(&user).Error; err == nil {
		userName = user.Name
	}

	var message models.Message

	// Check if this is a resend operation
	if req.ResendMessageID != nil {
		// Find the existing message to resend
		resendMessageID, err := uuid.Parse(*req.ResendMessageID)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Invalid resend message ID",
			})
		}

		// Get the existing message
		if err := h.db.Where("id = ? AND tenant_id = ?", resendMessageID, conversation.TenantID).First(&message).Error; err != nil {
			return c.JSON(http.StatusNotFound, map[string]string{
				"error": "Message to resend not found",
			})
		}

		// Update the message content and status
		message.Content = req.Content
		message.Status = "sending"
		message.UpdatedAt = time.Now()
	} else {
		// Create new message record
		message = models.Message{
			BaseTenantModel: models.BaseTenantModel{
				ID:        uuid.New(),
				TenantID:  conversation.TenantID,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			ConversationID: conversationID,
			CustomerID:     conversation.CustomerID,
			UserID:         &userID,
			UserName:       userName,
			Type:           req.Type,
			Content:        req.Content,
			Direction:      "out",
			Status:         "sent",
			ExternalID:     fmt.Sprintf("local_%d", time.Now().Unix()),
		}

		if req.ReplyToID != nil {
			// Find the message being replied to
			var replyTo models.Message
			if err := h.db.Where("external_id = ? OR id = ?", *req.ReplyToID, *req.ReplyToID).First(&replyTo).Error; err == nil {
				message.ReplyToID = &replyTo.ID
			}
		}
	}

	// Save or update message in database
	if req.ResendMessageID != nil {
		// Update existing message
		if err := h.db.Save(&message).Error; err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Failed to update message: " + err.Error(),
			})
		}
	} else {
		// Create new message
		if err := h.db.Create(&message).Error; err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Failed to save message: " + err.Error(),
			})
		}
	}

	// Update conversation
	now := time.Now()
	conversation.LastMessageAt = &now
	if err := h.db.Save(&conversation).Error; err != nil {
		// Log but don't fail the request
		fmt.Printf("Failed to update conversation: %v\n", err)
	}

	// Get channel info to get session for external API
	var channel models.Channel
	if err := h.db.First(&channel, conversation.ChannelID).Error; err != nil {
		fmt.Printf("Failed to get channel info: %v\n", err)
	} else if channel.Session != "" {
		// Get customer info for phone number
		var customer models.Customer
		if err := h.db.First(&customer, conversation.CustomerID).Error; err == nil {
			// Try to send via external WhatsApp API
			externalID, err := h.sendViaExternalAPI(channel.Session, customer.Phone, req.Content)
			if err != nil {
				fmt.Printf("Failed to send via external WhatsApp API: %v\n", err)
				// Update message status to indicate it failed to send
				message.Status = "failed"
				h.db.Save(&message)
			} else {
				// Update message with external ID and status
				if externalID != nil {
					message.ExternalID = *externalID
				}
				message.Status = "sent"
				h.db.Save(&message)
			}
		} else {
			// Customer not found, mark as failed
			message.Status = "failed"
			h.db.Save(&message)
			fmt.Printf("Customer not found for conversation %s\n", conversationID)
		}
	} else {
		// No session configured, mark as failed
		message.Status = "failed"
		h.db.Save(&message)
		fmt.Printf("Channel session not configured, message failed\n")
	}

	// Send WebSocket notification for real-time update
	if h.wsHandler != nil {
		messageData := map[string]interface{}{
			"type":            "new_message",
			"conversation_id": conversationID.String(),
			"message":         message,
		}
		h.wsHandler.BroadcastToTenant(conversation.TenantID.String(), "message_sent", messageData)
		log.Printf("WebSocket notification sent for new message in conversation %s", conversationID.String())
	}

	return c.JSON(http.StatusOK, message)
}

// ListConversations lists conversations with pagination
func (h *WhatsAppHandler) ListConversations(c echo.Context) error {
	// Get pagination parameters
	page, _ := strconv.Atoi(c.QueryParam("page"))
	if page <= 0 {
		page = 1
	}

	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	offset := (page - 1) * limit

	// Get tenant ID from context (should come from middleware)
	tenantID, ok := c.Get("tenant_id").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Tenant ID not found in context",
		})
	}

	// Query conversations
	var conversations []models.Conversation
	query := h.db.Where("conversations.tenant_id = ?", tenantID)

	// Search functionality
	if search := c.QueryParam("search"); search != "" {
		// Search in customer name, phone, and message content
		searchPattern := "%" + strings.ToLower(search) + "%"
		query = query.Joins("LEFT JOIN customers ON conversations.customer_id = customers.id").
			Joins("LEFT JOIN messages ON conversations.id = messages.conversation_id").
			Where("LOWER(customers.name) LIKE ? OR LOWER(customers.phone) LIKE ? OR LOWER(messages.content) LIKE ?",
				searchPattern, searchPattern, searchPattern).
			Group("conversations.id")
	}

	// Filter by archived status
	archivedParam := c.QueryParam("archived")
	if archivedParam != "" {
		archived := archivedParam == "true"
		query = query.Where("conversations.is_archived = ?", archived)
	} else {
		// Default to non-archived conversations if no parameter is specified
		query = query.Where("conversations.is_archived = ?", false)
	}

	// Apply other filters
	if status := c.QueryParam("status"); status != "" {
		query = query.Where("conversations.status = ?", status)
	}

	if channelID := c.QueryParam("channel_id"); channelID != "" {
		if chID, err := uuid.Parse(channelID); err == nil {
			query = query.Where("conversations.channel_id = ?", chID)
		}
	}

	// Filter by assigned agent
	if assignedAgentID := c.QueryParam("assigned_agent_id"); assignedAgentID != "" {
		if agentID, err := uuid.Parse(assignedAgentID); err == nil {
			query = query.Where("conversations.assigned_agent_id = ?", agentID)
		}
	}

	// Filter by has_agent (conversations with or without assigned agent)
	if hasAgentParam := c.QueryParam("has_agent"); hasAgentParam != "" {
		hasAgent := hasAgentParam == "true"
		if hasAgent {
			query = query.Where("conversations.assigned_agent_id IS NOT NULL")
		} else {
			query = query.Where("conversations.assigned_agent_id IS NULL")
		}
	}

	// Order: pinned first, then by last message time
	query = query.Order("conversations.is_pinned DESC, conversations.last_message_at DESC NULLS LAST, conversations.created_at DESC")

	// Get total count
	var total int64
	if err := query.Model(&models.Conversation{}).Count(&total).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to count conversations",
		})
	}

	// Get conversations with pagination and preload related data
	if err := query.Preload("Customer").Preload("Channel").Preload("AssignedAgent").Offset(offset).Limit(limit).Find(&conversations).Error; err != nil {
		log.Printf("Database error fetching conversations: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to fetch conversations",
		})
	}

	response := map[string]interface{}{
		"conversations": conversations,
		"pagination": map[string]interface{}{
			"page":        page,
			"limit":       limit,
			"total":       total,
			"total_pages": (total + int64(limit) - 1) / int64(limit),
		},
	}

	return c.JSON(http.StatusOK, response)
}

// GetConversation gets a specific conversation with recent messages
func (h *WhatsAppHandler) GetConversation(c echo.Context) error {
	conversationID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid conversation ID",
		})
	}

	// Get conversation
	var conversation models.Conversation
	if err := h.db.Where("id = ?", conversationID).
		Preload("Customer").
		Preload("AssignedAgent").
		First(&conversation).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.JSON(http.StatusNotFound, map[string]string{
				"error": "Conversation not found",
			})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to fetch conversation",
		})
	}

	// Get recent messages with user information
	var messages []models.Message
	messagesLimit, _ := strconv.Atoi(c.QueryParam("messages_limit"))
	if messagesLimit <= 0 || messagesLimit > 100 {
		messagesLimit = 50
	}

	if err := h.db.Where("conversation_id = ?", conversationID).
		Order("created_at DESC").
		Limit(messagesLimit).
		Find(&messages).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to fetch messages",
		})
	}

	// Update user_name for messages that have user_id but no user_name
	for i := range messages {
		if messages[i].UserID != nil && messages[i].UserName == "" {
			var user models.User
			if err := h.db.Select("name").Where("id = ?", *messages[i].UserID).First(&user).Error; err == nil {
				messages[i].UserName = user.Name
			}
		}
		// Set default for outgoing messages without user_name
		if messages[i].Direction == "out" && messages[i].UserName == "" {
			messages[i].UserName = "Assistente IA"
		}
	}

	// Reverse messages to show oldest first
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"conversation": conversation,
		"messages":     messages,
	})
}

// UpdateConversationStatus updates conversation status
func (h *WhatsAppHandler) UpdateConversationStatus(c echo.Context) error {
	conversationID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid conversation ID",
		})
	}

	// Get tenant ID from context
	tenantID, ok := c.Get("tenant_id").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Tenant ID not found in context",
		})
	}

	var req struct {
		Status          string     `json:"status" validate:"omitempty,oneof=open closed"`
		AssignedTo      *uuid.UUID `json:"assigned_to"`
		AssignedAgentID *uuid.UUID `json:"assigned_agent_id"`
		Priority        string     `json:"priority" validate:"omitempty,oneof=low normal high urgent"`
		IsArchived      *bool      `json:"is_archived"`
		IsPinned        *bool      `json:"is_pinned"`
	}

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request payload",
		})
	}

	// Get current conversation to check for assignment changes
	// Using a temporary struct that matches the actual database structure
	var currentConversation struct {
		ID         uuid.UUID  `gorm:"column:id"`
		TenantID   uuid.UUID  `gorm:"column:tenant_id"`
		CustomerID uuid.UUID  `gorm:"column:customer_id"`
		AssignedTo *uuid.UUID `gorm:"column:assigned_to"`
		Status     string     `gorm:"column:status"`
	}
	if err := h.db.Table("conversations").Where("id = ? AND tenant_id = ?", conversationID, tenantID).First(&currentConversation).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.JSON(http.StatusNotFound, map[string]string{
				"error": "Conversation not found",
			})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to fetch conversation",
		})
	}

	// Check if we're assigning a new agent
	var newAgentID *uuid.UUID
	if req.AssignedTo != nil {
		newAgentID = req.AssignedTo
	} else if req.AssignedAgentID != nil {
		newAgentID = req.AssignedAgentID
	}

	// If assigning to a new agent (different from current), handle WhatsApp group proxy
	if newAgentID != nil && (currentConversation.AssignedTo == nil || *currentConversation.AssignedTo != *newAgentID) {
		// Get conversation with channel to get session
		var fullConversation models.Conversation
		if err := h.db.Preload("Channel").Where("id = ? AND tenant_id = ?", conversationID, tenantID).First(&fullConversation).Error; err != nil {
			log.Printf("Failed to load conversation with channel: %v", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Failed to load conversation details",
			})
		}

		if fullConversation.Channel.Session == "" {
			log.Printf("Channel session not configured for conversation %s", conversationID)
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Canal WhatsApp não está configurado com sessão",
			})
		}
	}

	// Update conversation
	updates := make(map[string]interface{})
	if req.Status != "" {
		updates["status"] = req.Status
	}
	if req.AssignedTo != nil {
		updates["assigned_to"] = req.AssignedTo
	}
	if req.AssignedAgentID != nil {
		updates["assigned_agent_id"] = req.AssignedAgentID
	}
	if req.Priority != "" {
		updates["priority"] = req.Priority
	}
	if req.IsArchived != nil {
		updates["is_archived"] = *req.IsArchived
	}
	if req.IsPinned != nil {
		updates["is_pinned"] = *req.IsPinned
	}

	if err := h.db.Model(&models.Conversation{}).
		Where("id = ? AND tenant_id = ?", conversationID, tenantID).
		Updates(updates).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to update conversation",
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Conversation updated successfully",
	})
}

// MarkAsRead marks conversation messages as read
func (h *WhatsAppHandler) MarkAsRead(c echo.Context) error {
	conversationID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid conversation ID",
		})
	}

	// Reset unread count
	if err := h.db.Model(&models.Conversation{}).
		Where("id = ?", conversationID).
		Update("unread_count", 0).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to mark as read",
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Conversation marked as read",
	})
}

// WhatsAppSessionStatus represents the session status response
type WhatsAppSessionStatus struct {
	Name   string      `json:"name"`
	Status string      `json:"status"`
	Config interface{} `json:"config,omitempty"`
	Me     interface{} `json:"me,omitempty"`
	Engine interface{} `json:"engine,omitempty"`
}

// GetSessionStatus gets session status from external WhatsApp API using channel session
func (h *WhatsAppHandler) GetSessionStatus(c echo.Context) error {
	// Get tenant ID from context (set by middleware)
	tenantID := c.Get("tenant_id")
	if tenantID == nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Tenant ID not found",
		})
	}

	// Get channel ID from query parameter
	channelIDStr := c.QueryParam("channel_id")
	if channelIDStr == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Channel ID is required",
		})
	}

	channelID, err := uuid.Parse(channelIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid channel ID format",
		})
	}

	// Get channel from database to retrieve session
	var channel models.Channel
	if err := h.db.Where("id = ? AND tenant_id = ?", channelID, tenantID).First(&channel).Error; err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Channel not found",
		})
	}

	if channel.Session == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Channel session not configured",
		})
	}

	// Use centralized ZapPlus client to check session status
	client := zapplus.GetClient()
	sessionResponse, err := client.GetSessionStatus(channel.Session)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to fetch session status from ZapPlus API",
		})
	}

	// Convert to the expected format (backward compatibility)
	sessionStatus := WhatsAppSessionStatus{
		Status: sessionResponse.Status,
	}

	// Update channel status based on session status
	var newStatus string
	switch sessionStatus.Status {
	case "WORKING":
		newStatus = "connected"
	case "STARTING":
		newStatus = "connecting"
	default:
		newStatus = "disconnected"
	}

	// Update channel status in database if it has changed
	if channel.Status != newStatus {
		channel.Status = newStatus
		if err := h.db.Save(&channel).Error; err != nil {
			// Log error but don't fail the request
			log.Printf("Failed to update channel status: %v", err)
		}
	}

	return c.JSON(http.StatusOK, sessionStatus)
}

// GetQRCode gets QR code from external WhatsApp API using channel session
func (h *WhatsAppHandler) GetQRCode(c echo.Context) error {
	// Get user role from context
	userRole, _ := c.Get("user_role").(string)

	// Get tenant ID from context (for normal admins) or query parameter (for system_admin)
	var tenantID uuid.UUID
	var ok bool

	if userRole == "system_admin" {
		// For system_admin, tenant_id must be provided as query parameter
		tenantIDParam := c.QueryParam("tenant_id")
		if tenantIDParam == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "tenant_id query parameter required for system_admin",
			})
		}

		var err error
		tenantID, err = uuid.Parse(tenantIDParam)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Invalid tenant_id format",
			})
		}
	} else {
		// For regular admins, get tenant from context
		tenantIDInterface := c.Get("tenant_id")
		if tenantIDInterface == nil {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Tenant ID not found",
			})
		}

		tenantID, ok = tenantIDInterface.(uuid.UUID)
		if !ok {
			return c.JSON(http.StatusUnauthorized, map[string]string{
				"error": "Invalid tenant",
			})
		}
	}

	// Get channel ID from query parameter
	channelIDStr := c.QueryParam("channel_id")
	if channelIDStr == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Channel ID is required",
		})
	}

	channelID, err := uuid.Parse(channelIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid channel ID format",
		})
	}

	// Get channel from database to retrieve session
	var channel models.Channel
	if err := h.db.Where("id = ? AND tenant_id = ?", channelID, tenantID).First(&channel).Error; err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Channel not found",
		})
	}

	if channel.Session == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Channel session not configured",
		})
	}

	// First, check session status using centralized client
	client := zapplus.GetClient()
	sessionStatus, err := client.GetSessionStatus(channel.Session)
	if err == nil && sessionStatus != nil {
		// If already connected (WORKING status), update channel and return error
		if sessionStatus.Status == "WORKING" {
			// Update channel status to connected
			if channel.Status != "connected" {
				channel.Status = "connected"
				if err := h.db.Save(&channel).Error; err != nil {
					log.Printf("Failed to update channel status: %v", err)
				}
			}

			return c.JSON(http.StatusConflict, map[string]interface{}{
				"error":   "Session already connected",
				"status":  sessionStatus.Status,
				"session": sessionStatus,
			})
		}
	}

	// Get QR code image using centralized client
	resp, err := client.GetQRCodeImage(channel.Session)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to fetch QR code from external API",
		})
	}
	defer resp.Body.Close()

	// Set appropriate headers and return image
	c.Response().Header().Set("Content-Type", "image/png")
	c.Response().Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	c.Response().Header().Set("Pragma", "no-cache")
	c.Response().Header().Set("Expires", "0")

	return c.Stream(http.StatusOK, "image/png", resp.Body)
}

// CreateOrFindConversationByCustomer creates a new conversation or finds existing one for a customer
func (h *WhatsAppHandler) CreateOrFindConversationByCustomer(c echo.Context) error {
	// Get tenant ID from context
	tenantID, ok := c.Get("tenant_id").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Tenant ID not found in context",
		})
	}

	// Parse customer ID from URL parameter
	customerIDStr := c.Param("customerId")
	customerID, err := uuid.Parse(customerIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid customer ID format",
		})
	}

	// Check if customer exists and belongs to the tenant
	var customer models.Customer
	if err := h.db.Where("id = ? AND tenant_id = ?", customerID, tenantID).First(&customer).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.JSON(http.StatusNotFound, map[string]string{
				"error": "Customer not found",
			})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to fetch customer",
		})
	}

	// Look for existing conversation with this customer
	var conversation models.Conversation
	err = h.db.Where("tenant_id = ? AND customer_id = ?", tenantID, customerID).
		Preload("Customer").
		First(&conversation).Error

	if err == nil {
		// Conversation exists, update the last_message_at to bring it to the top
		now := time.Now()
		conversation.LastMessageAt = &now
		if err := h.db.Save(&conversation).Error; err != nil {
			log.Printf("Error updating conversation timestamp: %v", err)
			// Continue anyway as we can still return the existing conversation
		}

		return c.JSON(http.StatusOK, conversation)
	}

	if err != gorm.ErrRecordNotFound {
		// Unexpected database error
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to query existing conversations",
		})
	}

	// No existing conversation found, create a new one
	// First, get the default channel for this tenant
	var channel models.Channel
	if err := h.db.Where("tenant_id = ? AND is_active = true", tenantID).First(&channel).Error; err != nil {
		log.Printf("Error finding active channel for tenant %s: %v", tenantID, err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "No active channel found for this tenant",
		})
	}

	now := time.Now()
	newConversation := models.Conversation{
		BaseTenantModel: models.BaseTenantModel{
			ID:       uuid.New(),
			TenantID: tenantID,
		},
		CustomerID:    customerID,
		ChannelID:     channel.ID, // Use the existing channel ID
		Status:        "open",
		LastMessageAt: &now,
	}

	if err := h.db.Create(&newConversation).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to create conversation",
		})
	}

	// Load the customer data for the response
	if err := h.db.Preload("Customer").Where("id = ?", newConversation.ID).First(&newConversation).Error; err != nil {
		log.Printf("Error loading conversation with customer: %v", err)
		// Return conversation without customer data as fallback
	}

	return c.JSON(http.StatusCreated, newConversation)
}

// ArchiveConversation archives or unarchives a conversation
func (h *WhatsAppHandler) ArchiveConversation(c echo.Context) error {
	conversationID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid conversation ID",
		})
	}

	tenantID, ok := c.Get("tenant_id").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Tenant ID not found in context",
		})
	}

	// Get current conversation state
	var conversation models.Conversation
	if err := h.db.Where("id = ? AND tenant_id = ?", conversationID, tenantID).First(&conversation).Error; err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Conversation not found",
		})
	}

	// Toggle archive state
	newArchiveState := !conversation.IsArchived

	// Update conversation
	result := h.db.Model(&models.Conversation{}).
		Where("id = ? AND tenant_id = ?", conversationID, tenantID).
		Update("is_archived", newArchiveState)

	if result.Error != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to update conversation",
		})
	}

	if result.RowsAffected == 0 {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Conversation not found",
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success":     true,
		"is_archived": newArchiveState,
	})
}

// PinConversation pins or unpins a conversation
func (h *WhatsAppHandler) PinConversation(c echo.Context) error {
	conversationID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid conversation ID",
		})
	}

	tenantID, ok := c.Get("tenant_id").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Tenant ID not found in context",
		})
	}

	// Get current conversation state
	var conversation models.Conversation
	if err := h.db.Where("id = ? AND tenant_id = ?", conversationID, tenantID).First(&conversation).Error; err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Conversation not found",
		})
	}

	// Toggle pin state
	newPinState := !conversation.IsPinned

	// Update conversation
	result := h.db.Model(&models.Conversation{}).
		Where("id = ? AND tenant_id = ?", conversationID, tenantID).
		Update("is_pinned", newPinState)

	if result.Error != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to update conversation",
		})
	}

	if result.RowsAffected == 0 {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Conversation not found",
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success":   true,
		"is_pinned": newPinState,
	})
}

// ToggleAIConversation toggles AI response for a conversation
func (h *WhatsAppHandler) ToggleAIConversation(c echo.Context) error {
	conversationID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid conversation ID",
		})
	}

	tenantID, ok := c.Get("tenant_id").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Tenant ID not found in context",
		})
	}

	// Get current conversation state
	var conversation models.Conversation
	if err := h.db.Where("id = ? AND tenant_id = ?", conversationID, tenantID).First(&conversation).Error; err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Conversation not found",
		})
	}

	// Toggle AI state
	newAIState := !conversation.AIEnabled

	// Update conversation
	result := h.db.Model(&models.Conversation{}).
		Where("id = ? AND tenant_id = ?", conversationID, tenantID).
		Update("ai_enabled", newAIState)

	if result.Error != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to update conversation",
		})
	}

	if result.RowsAffected == 0 {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Conversation not found",
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success":    true,
		"ai_enabled": newAIState,
	})
}

// UpdateMessageStatusRequest represents a message status update request
type UpdateMessageStatusRequest struct {
	Status string `json:"status" validate:"required"`
}

// UpdateMessageStatus updates the status of a message
// @Summary Update message status
// @Description Update the status of a message (e.g., from failed to sent)
// @Tags WhatsApp
// @Accept json
// @Produce json
// @Param id path string true "Message ID"
// @Param request body UpdateMessageStatusRequest true "Update message status request"
// @Success 200 {object} models.Message
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
func (h *WhatsAppHandler) UpdateMessageStatus(c echo.Context) error {
	messageID := c.Param("id")
	parsedMessageID, err := uuid.Parse(messageID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid message ID",
		})
	}

	var req UpdateMessageStatusRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request payload",
		})
	}

	tenantID, ok := c.Get("tenant_id").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Tenant ID not found in context",
		})
	}

	// Update message status
	var message models.Message
	result := h.db.Model(&message).
		Where("id = ? AND tenant_id = ?", parsedMessageID, tenantID).
		Updates(map[string]interface{}{
			"status":     req.Status,
			"updated_at": time.Now(),
		})

	if result.Error != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to update message status",
		})
	}

	if result.RowsAffected == 0 {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Message not found",
		})
	}

	// Get updated message
	if err := h.db.Where("id = ? AND tenant_id = ?", parsedMessageID, tenantID).First(&message).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to fetch updated message",
		})
	}

	return c.JSON(http.StatusOK, message)
}

// UploadMedia handles file uploads to S3
func (h *WhatsAppHandler) UploadMedia(c echo.Context) error {
	// Get file from request
	file, err := c.FormFile("file")
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "No file provided",
		})
	}

	// Get tenant from context
	tenantID, ok := c.Get("tenant_id").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusForbidden, map[string]string{
			"error": "Tenant context required",
		})
	}

	// Get media type from query parameter or detect from file
	mediaType := c.QueryParam("type")
	if mediaType == "" {
		// Detect media type from file extension or content
		ext := strings.ToLower(filepath.Ext(file.Filename))
		switch ext {
		case ".jpg", ".jpeg", ".png", ".gif", ".webp":
			mediaType = "image"
		case ".mp3", ".wav", ".ogg", ".m4a", ".aac":
			mediaType = "audio"
		case ".pdf", ".doc", ".docx", ".txt", ".xlsx", ".pptx":
			mediaType = "document"
		default:
			mediaType = "document"
		}
	}

	// Generate message ID to use as filename
	messageID := uuid.New().String()

	// Check if storage service is available
	if h.storageService == nil {
		// Fallback to simple URL generation for development
		ext := filepath.Ext(file.Filename)
		s3Key := fmt.Sprintf("%s/media/%s%s", tenantID.String(), messageID, ext)
		publicURL := fmt.Sprintf("https://your-s3-bucket.com/%s", s3Key)
		return c.JSON(http.StatusOK, map[string]string{
			"url":       publicURL,
			"messageId": messageID,
		})
	}

	// Upload to S3 using StorageService with message ID as filename
	publicURL, err := h.storageService.UploadMediaFile(file, tenantID.String(), messageID, mediaType)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("Failed to upload file: %v", err),
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"url":       publicURL,
		"messageId": messageID,
	})
}

// SendImage sends an image via WhatsApp
func (h *WhatsAppHandler) SendImage(c echo.Context) error {
	var req struct {
		File struct {
			Mimetype string `json:"mimetype"`
			Filename string `json:"filename"`
			URL      string `json:"url"`
		} `json:"file"`
		Caption        string `json:"caption,omitempty"`
		ConversationID string `json:"conversation_id"`
	}

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request format",
		})
	}

	// Get tenant ID from context
	tenantID, ok := c.Get("tenant_id").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Tenant ID not found in context",
		})
	}

	// Get session from conversation -> channel
	conversationID, err := uuid.Parse(req.ConversationID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid conversation ID",
		})
	}

	var conversation models.Conversation
	if err := h.db.Where("id = ? AND tenant_id = ?", conversationID, tenantID).First(&conversation).Error; err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Conversation not found",
		})
	}

	var channel models.Channel
	if err := h.db.Where("id = ? AND tenant_id = ?", conversation.ChannelID, tenantID).First(&channel).Error; err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Channel not found",
		})
	}

	if channel.Session == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Channel session not configured",
		})
	}

	// Get customer phone number and format for WhatsApp
	var customer models.Customer
	if err := h.db.Where("id = ? AND tenant_id = ?", conversation.CustomerID, tenantID).First(&customer).Error; err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Customer not found",
		})
	}

	// Format phone number for WhatsApp
	chatID := formatPhoneForWhatsApp(customer.Phone)

	// Send via ZapPlus API
	response, err := h.sendImageViaZapPlus(chatID, req.File, req.Caption, channel.Session)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("Failed to send image: %v", err),
		})
	}

	// Extract external_id from response and create message record
	if responseData, ok := response["_data"].(map[string]interface{}); ok {
		if idData, ok := responseData["id"].(map[string]interface{}); ok {
			if externalID, ok := idData["id"].(string); ok {
				// Create message record
				message := models.Message{
					BaseTenantModel: models.BaseTenantModel{
						ID:        uuid.New(),
						TenantID:  tenantID,
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					},
					ConversationID: conversationID,
					CustomerID:     conversation.CustomerID,
					Type:           "image",
					Content:        req.Caption,
					Direction:      "out",
					Status:         "sent",
					ExternalID:     externalID,
					MediaURL:       req.File.URL,
					MediaType:      req.File.Mimetype,
				}

				// Get user ID from context for user_name
				if userIDInterface := c.Get("user_id"); userIDInterface != nil {
					if userID, ok := userIDInterface.(uuid.UUID); ok {
						message.UserID = &userID
						// Get user name
						var user models.User
						if err := h.db.Select("name").Where("id = ? AND tenant_id = ?", userID, tenantID).First(&user).Error; err == nil {
							message.UserName = user.Name
						} else {
							message.UserName = "Assistente IA"
						}
					}
				} else {
					message.UserName = "Assistente IA"
				}

				// Save message to database
				if err := h.db.Create(&message).Error; err != nil {
					fmt.Printf("Failed to save image message: %v\n", err)
				} else {
					// Update conversation timestamp
					now := time.Now()
					conversation.LastMessageAt = &now
					h.db.Save(&conversation)

					// Send WebSocket notification
					if h.wsHandler != nil {
						messageData := map[string]interface{}{
							"type":            "new_message",
							"conversation_id": conversationID.String(),
							"message":         message,
						}
						h.wsHandler.BroadcastToTenant(tenantID.String(), "message_sent", messageData)
					}
				}
			}
		}
	}

	return c.JSON(http.StatusOK, response)
}

// SendDocument sends a document via WhatsApp
func (h *WhatsAppHandler) SendDocument(c echo.Context) error {
	var req struct {
		File struct {
			Mimetype string `json:"mimetype"`
			Filename string `json:"filename"`
			URL      string `json:"url"`
		} `json:"file"`
		Caption        string `json:"caption,omitempty"`
		ConversationID string `json:"conversation_id"`
	}

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request format",
		})
	}

	// Get tenant ID from context
	tenantID, ok := c.Get("tenant_id").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Tenant ID not found in context",
		})
	}

	// Get session from conversation -> channel
	conversationID, err := uuid.Parse(req.ConversationID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid conversation ID",
		})
	}

	var conversation models.Conversation
	if err := h.db.Where("id = ? AND tenant_id = ?", conversationID, tenantID).First(&conversation).Error; err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Conversation not found",
		})
	}

	var channel models.Channel
	if err := h.db.Where("id = ? AND tenant_id = ?", conversation.ChannelID, tenantID).First(&channel).Error; err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Channel not found",
		})
	}

	if channel.Session == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Channel session not configured",
		})
	}

	// Get customer phone number and format for WhatsApp
	var customer models.Customer
	if err := h.db.Where("id = ? AND tenant_id = ?", conversation.CustomerID, tenantID).First(&customer).Error; err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Customer not found",
		})
	}

	// Format phone number for WhatsApp
	chatID := formatPhoneForWhatsApp(customer.Phone)

	// Send via ZapPlus API
	response, err := h.sendDocumentViaZapPlus(chatID, req.File, req.Caption, channel.Session)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("Failed to send document: %v", err),
		})
	}

	// Extract external_id from response and create message record
	if responseData, ok := response["_data"].(map[string]interface{}); ok {
		if idData, ok := responseData["id"].(map[string]interface{}); ok {
			if externalID, ok := idData["id"].(string); ok {
				// Create message record
				message := models.Message{
					BaseTenantModel: models.BaseTenantModel{
						ID:        uuid.New(),
						TenantID:  tenantID,
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					},
					ConversationID: conversationID,
					CustomerID:     conversation.CustomerID,
					Type:           "document",
					Content:        req.Caption,
					Direction:      "out",
					Status:         "sent",
					ExternalID:     externalID,
					MediaURL:       req.File.URL,
					MediaType:      req.File.Mimetype,
				}

				// Get user ID from context for user_name
				if userIDInterface := c.Get("user_id"); userIDInterface != nil {
					if userID, ok := userIDInterface.(uuid.UUID); ok {
						message.UserID = &userID
						// Get user name
						var user models.User
						if err := h.db.Select("name").Where("id = ? AND tenant_id = ?", userID, tenantID).First(&user).Error; err == nil {
							message.UserName = user.Name
						} else {
							message.UserName = "Assistente IA"
						}
					}
				} else {
					message.UserName = "Assistente IA"
				}

				// Save message to database
				if err := h.db.Create(&message).Error; err != nil {
					fmt.Printf("Failed to save document message: %v\n", err)
				} else {
					// Update conversation timestamp
					now := time.Now()
					conversation.LastMessageAt = &now
					h.db.Save(&conversation)

					// Send WebSocket notification
					if h.wsHandler != nil {
						messageData := map[string]interface{}{
							"type":            "new_message",
							"conversation_id": conversationID.String(),
							"message":         message,
						}
						h.wsHandler.BroadcastToTenant(tenantID.String(), "message_sent", messageData)
					}
				}
			}
		}
	}

	return c.JSON(http.StatusOK, response)
}

// SendAudio sends audio via WhatsApp
func (h *WhatsAppHandler) SendAudio(c echo.Context) error {
	var req struct {
		File struct {
			Mimetype string `json:"mimetype"`
			URL      string `json:"url"`
		} `json:"file"`
		Convert        bool   `json:"convert,omitempty"`
		ConversationID string `json:"conversation_id"`
	}

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request format",
		})
	}

	// Get tenant ID from context
	tenantID, ok := c.Get("tenant_id").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Tenant ID not found in context",
		})
	}

	// Get session from conversation -> channel
	conversationID, err := uuid.Parse(req.ConversationID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid conversation ID",
		})
	}

	var conversation models.Conversation
	if err := h.db.Where("id = ? AND tenant_id = ?", conversationID, tenantID).First(&conversation).Error; err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Conversation not found",
		})
	}

	var channel models.Channel
	if err := h.db.Where("id = ? AND tenant_id = ?", conversation.ChannelID, tenantID).First(&channel).Error; err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Channel not found",
		})
	}

	if channel.Session == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Channel session not configured",
		})
	}

	// Get customer phone number and format for WhatsApp
	var customer models.Customer
	if err := h.db.Where("id = ? AND tenant_id = ?", conversation.CustomerID, tenantID).First(&customer).Error; err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Customer not found",
		})
	}

	// Format phone number for WhatsApp
	chatID := formatPhoneForWhatsApp(customer.Phone)

	// Send via ZapPlus API - always convert audio to ensure proper format
	response, err := h.sendAudioViaZapPlus(chatID, req.File, true, channel.Session)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("Failed to send audio: %v", err),
		})
	}

	// Extract external_id from response and create message record
	if responseData, ok := response["_data"].(map[string]interface{}); ok {
		if idData, ok := responseData["id"].(map[string]interface{}); ok {
			if externalID, ok := idData["id"].(string); ok {
				// Create message record
				message := models.Message{
					BaseTenantModel: models.BaseTenantModel{
						ID:        uuid.New(),
						TenantID:  tenantID,
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					},
					ConversationID: conversationID,
					CustomerID:     conversation.CustomerID,
					Type:           "audio",
					Content:        "", // Audio doesn't have text content
					Direction:      "out",
					Status:         "sent",
					ExternalID:     externalID,
					MediaURL:       req.File.URL,
					MediaType:      "audio/ogg; codecs=opus", // Force OGG Opus for audio
				}

				// Get user ID from context for user_name
				if userIDInterface := c.Get("user_id"); userIDInterface != nil {
					if userID, ok := userIDInterface.(uuid.UUID); ok {
						message.UserID = &userID
						// Get user name
						var user models.User
						if err := h.db.Select("name").Where("id = ? AND tenant_id = ?", userID, tenantID).First(&user).Error; err == nil {
							message.UserName = user.Name
						} else {
							message.UserName = "Assistente IA"
						}
					}
				} else {
					message.UserName = "Assistente IA"
				}

				// Save message to database
				if err := h.db.Create(&message).Error; err != nil {
					fmt.Printf("Failed to save audio message: %v\n", err)
				} else {
					// Update conversation timestamp
					now := time.Now()
					conversation.LastMessageAt = &now
					h.db.Save(&conversation)

					// Send WebSocket notification
					if h.wsHandler != nil {
						messageData := map[string]interface{}{
							"type":            "new_message",
							"conversation_id": conversationID.String(),
							"message":         message,
						}
						h.wsHandler.BroadcastToTenant(tenantID.String(), "message_sent", messageData)
					}
				}
			}
		}
	}

	return c.JSON(http.StatusOK, response)
}

// Helper functions for ZapPlus API calls

func (h *WhatsAppHandler) sendImageViaZapPlus(chatID string, file struct {
	Mimetype string `json:"mimetype"`
	Filename string `json:"filename"`
	URL      string `json:"url"`
}, caption, session string) (map[string]interface{}, error) {
	fmt.Printf("Sending image request via ZapPlus: %s to %s\n", file.URL, chatID)

	// Use centralized ZapPlus client
	client := zapplus.GetClient()
	response, err := client.SendImageWithResponse(session, chatID, file.URL, caption)
	if err != nil {
		return nil, fmt.Errorf("failed to send image via ZapPlus: %w", err)
	}

	return response, nil
}

func (h *WhatsAppHandler) sendDocumentViaZapPlus(chatID string, file struct {
	Mimetype string `json:"mimetype"`
	Filename string `json:"filename"`
	URL      string `json:"url"`
}, caption, session string) (map[string]interface{}, error) {
	fmt.Printf("Sending document request via ZapPlus: %s to %s\n", file.URL, chatID)

	// Use centralized ZapPlus client
	client := zapplus.GetClient()
	response, err := client.SendFileWithResponse(session, chatID, file.URL, caption)
	if err != nil {
		return nil, fmt.Errorf("failed to send file via ZapPlus: %w", err)
	}

	return response, nil
}

func (h *WhatsAppHandler) sendAudioViaZapPlus(chatID string, file struct {
	Mimetype string `json:"mimetype"`
	URL      string `json:"url"`
}, convert bool, session string) (map[string]interface{}, error) {
	fmt.Printf("Sending audio request via ZapPlus: %s to %s\n", file.URL, chatID)

	// Use centralized ZapPlus client
	client := zapplus.GetClient()
	response, err := client.SendVoiceWithResponse(session, chatID, file.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to send voice via ZapPlus: %w", err)
	}

	return response, nil
}
