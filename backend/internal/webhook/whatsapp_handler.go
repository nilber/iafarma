package webhook

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime/debug"
	"strings"
	"time"

	"iafarma/internal/ai"
	"iafarma/internal/services"
	"iafarma/internal/zapplus"
	"iafarma/pkg/models"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

// ZapPlusWebhook represents the structure of incoming ZapPlus webhooks
type ZapPlusWebhook struct {
	ID        string                 `json:"id"`
	Timestamp int64                  `json:"timestamp"`
	Event     string                 `json:"event"`
	Session   string                 `json:"session"`
	Metadata  map[string]interface{} `json:"metadata"`
	Me        struct {
		ID       string `json:"id"`
		PushName string `json:"pushName"`
	} `json:"me"`
	Payload     ZapPlusPayload         `json:"payload"`
	Engine      string                 `json:"engine"`
	Environment map[string]interface{} `json:"environment"`
}

// ZapPlusPayload represents the message payload
type ZapPlusPayload struct {
	ID        string        `json:"id"`
	Timestamp int64         `json:"timestamp"`
	From      string        `json:"from"`
	FromMe    bool          `json:"fromMe"`
	Source    string        `json:"source"`
	To        string        `json:"to"`
	Body      string        `json:"body"`
	HasMedia  bool          `json:"hasMedia"`
	Media     *MediaInfo    `json:"media"`
	MediaURL  string        `json:"mediaUrl"`
	Ack       int           `json:"ack"`
	AckName   string        `json:"ackName"`
	VCards    []interface{} `json:"vCards"`
	Data      *struct {
		ID struct {
			FromMe     bool   `json:"fromMe"`
			Remote     string `json:"remote"`
			ID         string `json:"id"`
			Serialized string `json:"_serialized"`
		} `json:"id"`
		Viewed bool   `json:"viewed"`
		Body   string `json:"body"`
		Type   string `json:"type"`
		T      int64  `json:"t"`
		From   string `json:"from"`
		To     string `json:"to"`
		Ack    int    `json:"ack"`
	} `json:"_data"`
}

// MediaInfo represents media information in the webhook
type MediaInfo struct {
	URL      string `json:"url"`
	Filename string `json:"filename"`
	MimeType string `json:"mimetype"`
}

// ZapPlusWebhookHandler handles ZapPlus webhook processing
type ZapPlusWebhookHandler struct {
	db                    *gorm.DB
	wsNotifier            WebSocketNotifier
	aiService             *ai.AIService
	tenantSettingsService *ai.TenantSettingsService
}

// NewZapPlusWebhookHandler creates a new webhook handler
func NewZapPlusWebhookHandler(db *gorm.DB) *ZapPlusWebhookHandler {
	// Get OpenAI API key from environment
	openaiAPIKey := os.Getenv("OPENAI_API_KEY")

	// Create AI service if API key is available
	var aiService *ai.AIService
	if openaiAPIKey != "" {
		// Create delivery service for AI with Google Maps API key
		googleMapsAPIKey := os.Getenv("GOOGLE_MAPS_API_KEY")
		deliveryService := services.NewDeliveryService(db, googleMapsAPIKey)
		deliveryAdapter := ai.NewDeliveryServiceAdapter(deliveryService)

		aiService = ai.NewAIServiceFactoryWithDeliveryAndWebSocket(db, openaiAPIKey, deliveryAdapter, nil)
	}

	// Create tenant settings service
	tenantSettingsService := ai.NewTenantSettingsService(db)

	return &ZapPlusWebhookHandler{
		db:                    db,
		aiService:             aiService,
		tenantSettingsService: tenantSettingsService,
	}
}

// SetWebSocketNotifier sets the WebSocket notifier for sending real-time notifications
func (h *ZapPlusWebhookHandler) SetWebSocketNotifier(notifier WebSocketNotifier) {
	h.wsNotifier = notifier
}

// SetAIServiceWithEmbedding creates and sets the AI service with embedding support
func (h *ZapPlusWebhookHandler) SetAIServiceWithEmbedding(embeddingService ai.EmbeddingServiceInterface) {
	openaiAPIKey := os.Getenv("OPENAI_API_KEY")
	if openaiAPIKey != "" {
		// Create delivery service for AI with Google Maps API key
		googleMapsAPIKey := os.Getenv("GOOGLE_MAPS_API_KEY")
		deliveryService := services.NewDeliveryService(h.db, googleMapsAPIKey)
		deliveryAdapter := ai.NewDeliveryServiceAdapter(deliveryService)

		// Create AI service with embedding service
		h.aiService = ai.NewAIServiceFactoryComplete(h.db, openaiAPIKey, deliveryAdapter, nil, embeddingService)

		log.Printf("🤖 ZapPlus AI Service updated with embedding support")
		if embeddingService != nil {
			log.Printf("🧠 ZapPlus RAG functionality enabled")
		}
	}

}

// checkCreditsAndDeduct verifica se o tenant tem créditos suficientes e desconta se necessário
func (h *ZapPlusWebhookHandler) checkCreditsAndDeduct(ctx context.Context, tenant *models.Tenant) bool {
	// Se custo por mensagem é 0, não precisa descontar créditos
	if tenant.CostPerMessage == 0 {
		return true
	}

	// Verificar se há créditos suficientes
	var aiCredits models.AICredits
	err := h.db.WithContext(ctx).Where("tenant_id = ?", tenant.ID).First(&aiCredits).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			log.Printf("💳 No credits record found for tenant %s", tenant.ID)
			return false
		}
		log.Printf("❌ Error checking credits for tenant %s: %v", tenant.ID, err)
		return false
	}

	if aiCredits.RemainingCredits < tenant.CostPerMessage {
		log.Printf("💳 Insufficient credits for tenant %s: has %d, needs %d", tenant.ID, aiCredits.RemainingCredits, tenant.CostPerMessage)
		return false
	}

	// Descontar créditos através de transação (UserID nulo para transações automáticas)
	transaction := &models.AICreditTransaction{
		TenantID:    tenant.ID,
		UserID:      nil, // Transação automática do sistema (NULL)
		Type:        "use",
		Amount:      tenant.CostPerMessage,
		Description: "Desconto automático por mensagem processada pela IA",
	}

	// Atualizar créditos usados
	err = h.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Criar transação
		if err := tx.Create(transaction).Error; err != nil {
			return err
		}

		// Atualizar créditos
		return tx.Model(&aiCredits).Updates(map[string]interface{}{
			"used_credits":      gorm.Expr("used_credits + ?", tenant.CostPerMessage),
			"remaining_credits": gorm.Expr("remaining_credits - ?", tenant.CostPerMessage),
		}).Error
	})

	if err != nil {
		log.Printf("❌ Error deducting credits for tenant %s: %v", tenant.ID, err)
		return false
	}

	log.Printf("💳 Deducted %d credits from tenant %s (IA message processing)", tenant.CostPerMessage, tenant.ID)
	return true
}

// getUnavailableMessage retorna mensagem padrão quando IA está indisponível
func (h *ZapPlusWebhookHandler) getUnavailableMessage(tenant *models.Tenant) string {
	phone := tenant.StorePhone
	if phone == "" {
		phone = "não informado"
	} else {
		// Formatar telefone se possível (exemplo: (11) 99999-9999)
		if len(phone) >= 10 {
			if len(phone) == 11 {
				phone = fmt.Sprintf("(%s) %s-%s", phone[0:2], phone[2:7], phone[7:11])
			} else if len(phone) == 10 {
				phone = fmt.Sprintf("(%s) %s-%s", phone[0:2], phone[2:6], phone[6:10])
			}
		}
	}

	return fmt.Sprintf("Nosso atendimento automático está indisponível no momento, você pode tentar ligar para nossa loja no telefone %s, ou aguardar nosso atendimento humano, obrigado pela compreensão.", phone)
}

// ProcessZapPlusWebhook processes incoming ZapPlus webhook
// @Summary Process ZapPlus webhook
// @Description Receive and process ZapPlus webhook messages
// @Tags webhook
// @Accept json
// @Produce json
// @Param webhook body ZapPlusWebhook true "ZapPlus webhook data"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /webhook/zapplus [post]
func (h *ZapPlusWebhookHandler) ProcessZapPlusWebhook(c echo.Context) error {
	var webhook ZapPlusWebhook
	if err := c.Bind(&webhook); err != nil {
		log.Printf("Failed to parse webhook: %v", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid webhook data"})
	}

	// Log incoming webhook for debugging
	log.Printf("Received webhook: Event=%s, Session=%s, From=%s",
		webhook.Event, webhook.Session, webhook.Payload.From)

	// Handle message.ack events for status updates
	if webhook.Event == "message.ack" {
		if err := h.processMessageAck(webhook); err != nil {
			log.Printf("Error processing message.ack: %v", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to process ACK"})
		}
		return c.JSON(http.StatusOK, map[string]string{"status": "ack_processed"})
	}

	// Only process message events for regular messages
	if webhook.Event != "message" {
		log.Printf("Ignoring non-message event: %s", webhook.Event)
		return c.JSON(http.StatusOK, map[string]string{"status": "ignored"})
	}

	// Skip outgoing messages (fromMe = true)
	if webhook.Payload.FromMe {
		log.Printf("Ignoring outgoing message")
		return c.JSON(http.StatusOK, map[string]string{"status": "ignored"})
	}

	// Find channel by session, then get tenant
	var channel models.Channel
	if err := h.db.Where("session = ?", webhook.Session).First(&channel).Error; err != nil {
		// If channel not found, try to get tenant from metadata or create default channel
		var tenantID uuid.UUID

		// Try to get tenant ID from metadata first
		if metaTenantID, exists := webhook.Metadata["tenantId"]; exists {
			if tenantIDStr, ok := metaTenantID.(string); ok {
				if parsedID, parseErr := uuid.Parse(tenantIDStr); parseErr == nil {
					tenantID = parsedID
				}
			}
		}

		// If no tenant ID in metadata, get from session (phone number should map to existing tenant)
		if tenantID == uuid.Nil {
			log.Printf("Channel not found for session %s and no tenant ID in metadata: %v", webhook.Session, err)
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Channel not found and no tenant ID provided"})
		}

		// Create channel for this session
		channel = models.Channel{
			BaseTenantModel: models.BaseTenantModel{
				ID:        uuid.New(),
				TenantID:  tenantID,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			Name:     fmt.Sprintf("ZapPlus %s", webhook.Session),
			Type:     "zapplus",
			Session:  webhook.Session,
			Status:   "connected",
			Config:   "{}",
			IsActive: true,
		}

		if err := h.db.Create(&channel).Error; err != nil {
			log.Printf("Failed to create channel for session %s: %v", webhook.Session, err)
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Failed to create channel"})
		}

		log.Printf("Created new channel for session %s", webhook.Session)
	}

	// Get tenant from channel
	var tenant models.Tenant
	if err := h.db.Where("id = ?", channel.TenantID).First(&tenant).Error; err != nil {
		log.Printf("Tenant not found for channel %s: %v", channel.ID, err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Tenant not found"})
	}

	// Extract and clean phone number from webhook
	phone := h.extractPhoneNumber(webhook.Payload.From)
	if phone == "" {
		log.Printf("Failed to extract phone number from: %s", webhook.Payload.From)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid phone number"})
	}

	// Find or create customer
	customer, err := h.findOrCreateCustomer(tenant.ID, phone)
	if err != nil {
		log.Printf("Failed to find/create customer: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Customer handling failed"})
	}

	// Find or create conversation
	conversation, err := h.findOrCreateConversation(tenant.ID, customer.ID)
	if err != nil {
		log.Printf("Failed to find/create conversation: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Conversation handling failed"})
	}

	// Create message
	// Get media URL - try mediaUrl first, then media.url as fallback
	mediaURL := webhook.Payload.MediaURL
	if mediaURL == "" && webhook.Payload.Media != nil {
		mediaURL = webhook.Payload.Media.URL
	}

	log.Printf("Debug - MediaURL: '%s', HasMedia: %t, Media: %+v", mediaURL, webhook.Payload.HasMedia, webhook.Payload.Media)

	// Determine message source (chat or whatsapp)
	messageSource := "whatsapp" // default
	if source, ok := webhook.Metadata["source"].(string); ok && source == "chat" {
		messageSource = "chat"
	}

	message := models.Message{
		BaseTenantModel: models.BaseTenantModel{
			ID:       uuid.New(),
			TenantID: tenant.ID,
		},
		ConversationID: conversation.ID,
		CustomerID:     customer.ID,
		UserID:         nil, // Incoming message
		Type:           h.determineMessageType(&webhook.Payload),
		Content:        webhook.Payload.Body,
		Direction:      "in",
		Status:         "received",
		Source:         messageSource, // Set source field
		ExternalID:     webhook.Payload.ID,
		MediaURL:       mediaURL,
		MediaType:      h.getMediaType(&webhook.Payload),
		Filename:       h.getFilename(&webhook.Payload),
		IsRead:         false,
	}

	if err := h.db.Create(&message).Error; err != nil {
		log.Printf("Failed to create message: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Message creation failed"})
	}

	// Update conversation last message time and unread count
	err = h.db.Model(&conversation).Updates(map[string]interface{}{
		"last_message_at": time.Now(),
		"unread_count":    gorm.Expr("unread_count + 1"),
	}).Error
	if err != nil {
		log.Printf("Failed to update conversation: %v", err)
	}

	log.Printf("Successfully processed webhook message: %s", message.ID)

	// Send WebSocket notification if notifier is available
	if h.wsNotifier != nil {
		notificationData := map[string]interface{}{
			"type":            "new_message",
			"message_id":      message.ID.String(),
			"conversation_id": conversation.ID.String(),
			"customer_phone":  phone,
			"content":         webhook.Payload.Body,
			"from_me":         webhook.Payload.FromMe,
		}
		log.Printf("Sending WebSocket notification for ZapPlus message: %s", message.ID)
		h.wsNotifier.BroadcastWebhookNotification(tenant.ID.String(), "message", notificationData)
		log.Printf("WebSocket notification sent successfully for ZapPlus message")
	} else {
		log.Printf("WebSocket notifier is nil - no notification sent")
	}

	// Process with AI if message is not from us, and AI is enabled for this conversation
	if !webhook.Payload.FromMe && (message.Type == "text" || message.Type == "image" || message.Type == "audio") && (message.Content != "" || message.MediaURL != "") {
		// Check if customer is active (not blocked)
		if !customer.IsActive {
			log.Printf("Ignoring message from blocked customer: %s", phone)
			return c.JSON(http.StatusOK, map[string]string{"status": "message_processed"})
		}

		// Reload conversation to get the latest AI enabled status
		var currentConversation models.Conversation
		if err := h.db.First(&currentConversation, conversation.ID).Error; err != nil {
			log.Printf("Failed to reload conversation for AI check: %v", err)
			return c.JSON(http.StatusOK, map[string]string{"status": "message_processed"})
		}

		log.Printf("AI processing check - conversation_id: %s, ai_enabled: %t, customer_active: %t, message_type: %s", currentConversation.ID, currentConversation.AIEnabled, customer.IsActive, message.Type)

		// Check global AI setting first
		aiGlobalSetting, err := h.tenantSettingsService.GetSetting(context.Background(), tenant.ID, "ai_global_enabled")
		aiGlobalEnabled := true // default to true if setting not found
		if err == nil && aiGlobalSetting != nil && aiGlobalSetting.SettingValue != nil {
			aiGlobalEnabled = *aiGlobalSetting.SettingValue == "true"
		}

		log.Printf("AI Global Setting - enabled: %t, conversation_enabled: %t", aiGlobalEnabled, currentConversation.AIEnabled)

		// Only process with AI if both global and conversation AI are enabled
		if aiGlobalEnabled && currentConversation.AIEnabled {
			// Verificar se o tenant está ativo
			if tenant.Status != "active" {
				log.Printf("AI processing skipped - tenant is not active (status: %s): %s", tenant.Status, tenant.ID)
				return c.JSON(http.StatusOK, map[string]string{"status": "message_processed"})
			}

			// Verificar créditos e descontar se necessário
			if !h.checkCreditsAndDeduct(c.Request().Context(), &tenant) {
				log.Printf("AI processing skipped - insufficient credits or credit check failed for tenant: %s", tenant.ID)

				// Enviar mensagem de indisponibilidade se o tenant configurou uma
				unavailableMsg := h.getUnavailableMessage(&tenant)
				if unavailableMsg != "" {
					go func() {
						_, err := h.sendViaExternalAPI(webhook.Session, webhook.Payload.From, unavailableMsg)
						if err != nil {
							log.Printf("Failed to send unavailable message: %v", err)
						}
					}()
				}

				return c.JSON(http.StatusOK, map[string]string{"status": "message_processed"})
			}

			log.Printf("Processing message with AI for customer: %s", phone)

			// Process message with AI in a goroutine to avoid blocking the webhook response
			go func() {
				defer func() {
					if r := recover(); r != nil {
						log.Printf("Panic in AI processing goroutine: %v", r)
						log.Printf("Stack trace: %s", debug.Stack())
					}
				}()

				var aiResponse string
				var err error

				log.Printf("Starting AI processing - MessageType: %s, MediaURL: %s, TenantBusinessType: %s", message.Type, message.MediaURL, tenant.BusinessType)

				log.Printf("Using standard sales AI for tenant: %s", tenant.ID)
				// Use standard sales AI service
				if message.Type == "image" && message.MediaURL != "" {
					log.Printf("Processing image message for medication analysis: %s", message.MediaURL)
					aiResponse, err = h.aiService.ProcessImageMessage(context.Background(), tenant.ID, phone, message.MediaURL, message.ID.String())
				} else if message.Type == "audio" && message.MediaURL != "" {
					log.Printf("Processing audio message for transcription and analysis: %s", message.MediaURL)
					aiResponse, err = h.aiService.ProcessAudioMessage(context.Background(), tenant.ID, phone, message.MediaURL, message.ID.String())
				} else if message.Type == "text" && message.Content != "" {
					log.Printf("Processing text message: %s", message.Content)
					aiResponse, err = h.aiService.ProcessMessageWithConversation(context.Background(), tenant.ID, phone, message.Content, conversation.ID)
				} else {
					log.Printf("Skipping AI processing - no content or unsupported type: %s", message.Type)
					return
				}

				if err != nil {
					log.Printf("Error processing message with AI: %v", err)
					return
				}

				log.Printf("AI processing completed successfully. Response length: %d", len(aiResponse))

				if aiResponse != "" {
					log.Printf("AI generated response: %s", aiResponse)

					// Create outgoing message with source
					responseMessage := models.Message{
						BaseTenantModel: models.BaseTenantModel{
							ID:       uuid.New(),
							TenantID: tenant.ID,
						},
						ConversationID: conversation.ID,
						CustomerID:     customer.ID,
						UserID:         nil,             // AI response
						UserName:       "Assistente IA", // AI name
						Type:           "text",
						Content:        aiResponse,
						Direction:      "out",
						Status:         "sent",
						Source:         messageSource, // Same source as incoming message
						IsRead:         true,          // AI responses are considered read
					}

					if err := h.db.Create(&responseMessage).Error; err != nil {
						log.Printf("Failed to create AI response message: %v", err)
						return
					}

					log.Printf("AI response message saved: %s", responseMessage.ID)

					// Send WebSocket notification for AI response
					if h.wsNotifier != nil {
						notificationData := map[string]interface{}{
							"type":            "new_message",
							"message_id":      responseMessage.ID.String(),
							"conversation_id": conversation.ID.String(),
							"customer_phone":  phone,
							"content":         aiResponse,
							"from_me":         true,
						}
						h.wsNotifier.BroadcastWebhookNotification(tenant.ID.String(), "message", notificationData)
						log.Printf("WebSocket notification sent for AI response")
					}

					// Send response back to WhatsApp via ZapPlus API (skip if source is chat)
					if messageSource != "chat" {
						externalID, err := h.sendViaExternalAPI(webhook.Session, webhook.Payload.From, aiResponse)
						if err != nil {
							log.Printf("Failed to send AI response via ZapPlus API: %v", err)
							log.Printf("AI to: %s, tosession: %s", webhook.Payload.From, webhook.Session)
							// Update message status to failed
							responseMessage.Status = "failed"
							h.db.Save(&responseMessage)
						} else {
							log.Printf("AI response sent successfully via ZapPlus API")
							// Update message with external ID
							if externalID != nil {
								responseMessage.ExternalID = *externalID
								responseMessage.Status = "sent"
								h.db.Save(&responseMessage)
								log.Printf("AI response message updated with external_id: %s", *externalID)
							}
						}
					} else {
						log.Printf("Skipping ZapPlus API send - message source is chat")
						// For chat messages, we already saved the response message above
						// The WebSocket handler will send the response directly to the client
					}
				}
			}()
		} else {
			if !aiGlobalEnabled {
				log.Printf("AI processing skipped - AI globally disabled for tenant: %s", tenant.ID)
			} else {
				log.Printf("AI processing skipped - AI disabled for conversation: %s", currentConversation.ID)
			}
		}
	} else {
		log.Printf("AI processing skipped - message conditions not met (fromMe: %t, type: %s, hasContent: %t, hasMediaURL: %t)",
			webhook.Payload.FromMe, message.Type, message.Content != "", message.MediaURL != "")
	}

	return c.JSON(http.StatusOK, map[string]string{
		"status":     "processed",
		"message_id": message.ID.String(),
	})
}

// extractPhoneNumber extracts clean phone number from WhatsApp format
func (h *ZapPlusWebhookHandler) extractPhoneNumber(from string) string {
	// Remove everything after @ symbol
	parts := strings.Split(from, "@")
	if len(parts) == 0 {
		return ""
	}

	phone := parts[0]

	// Remove any non-numeric characters
	var cleaned strings.Builder
	for _, char := range phone {
		if char >= '0' && char <= '9' {
			cleaned.WriteRune(char)
		}
	}

	return cleaned.String()
}

// findOrCreateCustomer finds existing customer or creates new one
func (h *ZapPlusWebhookHandler) findOrCreateCustomer(tenantID uuid.UUID, phone string) (*models.Customer, error) {
	var customer models.Customer

	// Try to find existing customer
	err := h.db.Where("tenant_id = ? AND phone = ?", tenantID, phone).First(&customer).Error
	if err == nil {
		return &customer, nil
	}

	if err != gorm.ErrRecordNotFound {
		return nil, err
	}

	// Create new customer
	customer = models.Customer{
		BaseTenantModel: models.BaseTenantModel{
			ID:       uuid.New(),
			TenantID: tenantID,
		},
		Phone:    phone,
		Name:     "", // Nome será solicitado durante o checkout
		IsActive: true,
	}

	if err := h.db.Create(&customer).Error; err != nil {
		return nil, err
	}

	return &customer, nil
}

// findOrCreateConversation finds existing conversation or creates new one
func (h *ZapPlusWebhookHandler) findOrCreateConversation(tenantID, customerID uuid.UUID) (*models.Conversation, error) {
	var conversation models.Conversation

	log.Printf("🔍 Looking for conversation - TenantID: %s, CustomerID: %s", tenantID, customerID)

	// Try to find existing open AND non-archived conversation first
	err := h.db.Where("tenant_id = ? AND customer_id = ? AND status = ? AND is_archived = ?",
		tenantID, customerID, "open", false).First(&conversation).Error
	if err == nil {
		log.Printf("✅ Found existing open non-archived conversation: %s", conversation.ID)
		return &conversation, nil
	}

	if err != gorm.ErrRecordNotFound {
		log.Printf("❌ Error finding open non-archived conversation: %v", err)
		return nil, err
	}

	log.Printf("🔍 No open conversation found, looking for archived conversations...")

	// If no open conversation found, look for archived conversations
	err = h.db.Where("tenant_id = ? AND customer_id = ? AND is_archived = ?",
		tenantID, customerID, true).First(&conversation).Error
	if err == nil {
		// Found archived conversation - unarchive it and reset unread count
		log.Printf("📂 Found archived conversation %s, unarchiving due to new message", conversation.ID)
		err = h.db.Model(&conversation).Updates(map[string]interface{}{
			"is_archived":  false,
			"status":       "open",
			"unread_count": 0, // Reset to 0, will be incremented when message is processed
		}).Error
		if err != nil {
			log.Printf("❌ Failed to unarchive conversation: %v", err)
			return nil, err
		}
		log.Printf("✅ Successfully unarchived conversation %s", conversation.ID)
		return &conversation, nil
	}

	if err != gorm.ErrRecordNotFound {
		log.Printf("❌ Error finding archived conversation: %v", err)
		return nil, err
	}

	log.Printf("🔍 No archived conversation found, creating new conversation...")

	// Create default channel if not exists
	var channel models.Channel
	err = h.db.Where("tenant_id = ? AND type = ?", tenantID, "whatsapp").First(&channel).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// Create default WhatsApp channel
			channel = models.Channel{
				BaseTenantModel: models.BaseTenantModel{
					ID:       uuid.New(),
					TenantID: tenantID,
				},
				Name:     "WhatsApp",
				Type:     "whatsapp",
				Status:   "connected",
				IsActive: true,
			}
			if err := h.db.Create(&channel).Error; err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	// Create new conversation
	conversation = models.Conversation{
		BaseTenantModel: models.BaseTenantModel{
			ID:       uuid.New(),
			TenantID: tenantID,
		},
		CustomerID:  customerID,
		ChannelID:   channel.ID,
		Status:      "open",
		Priority:    "normal",
		IsArchived:  false,
		IsPinned:    false,
		UnreadCount: 0,
	}

	if err := h.db.Create(&conversation).Error; err != nil {
		log.Printf("❌ Failed to create new conversation: %v", err)
		return nil, err
	}

	log.Printf("✅ Created new conversation: %s", conversation.ID)
	return &conversation, nil
}

// determineMessageType determines the message type based on payload
func (h *ZapPlusWebhookHandler) determineMessageType(payload *ZapPlusPayload) string {
	if !payload.HasMedia {
		return "text"
	}

	if payload.Media == nil {
		return "text"
	}

	mimeType := payload.Media.MimeType
	switch {
	case strings.HasPrefix(mimeType, "image/"):
		return "image"
	case strings.HasPrefix(mimeType, "audio/"):
		return "audio"
	case strings.HasPrefix(mimeType, "video/"):
		return "video"
	case strings.Contains(mimeType, "document") || strings.Contains(mimeType, "application/"):
		return "document"
	default:
		return "file"
	}
}

// getMediaType returns the media MIME type
func (h *ZapPlusWebhookHandler) getMediaType(payload *ZapPlusPayload) string {
	if payload.Media != nil {
		return payload.Media.MimeType
	}
	return ""
}

// getFilename returns the filename from payload
func (h *ZapPlusWebhookHandler) getFilename(payload *ZapPlusPayload) string {
	if payload.Media != nil && payload.Media.Filename != "" {
		return payload.Media.Filename
	}

	// For documents, filename might be in body
	if payload.HasMedia && payload.Body != "" {
		return payload.Body
	}

	return ""
}

// sendViaExternalAPI sends a message via external WhatsApp API and returns the external ID
func (h *ZapPlusWebhookHandler) sendViaExternalAPI(session, phone, text string) (*string, error) {
	if session == "" {
		return nil, fmt.Errorf("channel session not configured")
	}

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

	text = strings.ReplaceAll(text, "**", "*")

	// Use centralized ZapPlus client
	client := zapplus.GetClient()
	response, err := client.SendTextMessageWithResponse(session, cleanPhone, text)
	if err != nil {
		return nil, fmt.Errorf("failed to send message: %w", err)
	}

	externalID := response.Data.ID.ID
	if externalID == "" {
		return nil, fmt.Errorf("external ID not found in response")
	}

	return &externalID, nil
}

// processMessageAck processes message acknowledgment events from ZapPlus
func (h *ZapPlusWebhookHandler) processMessageAck(webhook ZapPlusWebhook) error {
	log.Printf("Processing message.ack event for external_id: %s, ackName: %s",
		webhook.Payload.Data.ID.ID, webhook.Payload.AckName)

	// Extract external ID from payload._data.id.id
	if webhook.Payload.Data == nil || webhook.Payload.Data.ID.ID == "" {
		log.Printf("No external ID found in message.ack event")
		return nil
	}

	externalID := webhook.Payload.Data.ID.ID
	ackName := webhook.Payload.AckName

	// Map ACK status to our internal status
	var newStatus string
	switch ackName {
	case "SERVER":
		newStatus = "sent"
	case "DEVICE":
		newStatus = "delivered"
	case "READ":
		newStatus = "read"
	default:
		log.Printf("Unknown ackName: %s", ackName)
		return nil
	}

	// Find the message by external_id
	var message models.Message
	if err := h.db.Where("external_id = ?", externalID).First(&message).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			log.Printf("Message not found for external_id: %s", externalID)
			return nil
		}
		log.Printf("Database error finding message: %v", err)
		return err
	}

	// Security rule: Don't downgrade status
	// Status hierarchy: sent < delivered < read
	statusHierarchy := map[string]int{
		"sent":      1,
		"delivered": 2,
		"read":      3,
	}

	currentLevel, currentExists := statusHierarchy[message.Status]
	newLevel, newExists := statusHierarchy[newStatus]

	if !currentExists || !newExists {
		// If current or new status is not in hierarchy, allow update
		log.Printf("Status not in hierarchy, allowing update from %s to %s", message.Status, newStatus)
	} else if newLevel <= currentLevel {
		// Don't downgrade status
		log.Printf("Ignoring status downgrade from %s (level %d) to %s (level %d) for message %s",
			message.Status, currentLevel, newStatus, newLevel, message.ID)
		return nil
	}

	// Update message status
	oldStatus := message.Status
	message.Status = newStatus
	message.UpdatedAt = time.Now()

	if err := h.db.Save(&message).Error; err != nil {
		log.Printf("Failed to update message status: %v", err)
		return err
	}

	log.Printf("Message status updated: %s -> %s for external_id: %s, message_id: %s",
		oldStatus, newStatus, externalID, message.ID)

	// Send WebSocket notification about status change
	if h.wsNotifier != nil {
		notificationData := map[string]interface{}{
			"type":       "message_status_update",
			"message_id": message.ID.String(),
			"old_status": oldStatus,
			"new_status": newStatus,
		}
		h.wsNotifier.BroadcastWebhookNotification(message.TenantID.String(), "message_status", notificationData)
	}

	return nil
}
