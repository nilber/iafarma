package whatsapp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"

	"iafarma/pkg/models"
)

// Client represents a WhatsApp Business API client
type Client struct {
	baseURL     string
	accessToken string
	phoneID     string
	db          *gorm.DB
}

// NewClient creates a new WhatsApp Business API client
func NewClient(baseURL, accessToken, phoneID string, db *gorm.DB) *Client {
	return &Client{
		baseURL:     baseURL,
		accessToken: accessToken,
		phoneID:     phoneID,
		db:          db,
	}
}

// SendMessageRequest represents a message send request
type SendMessageRequest struct {
	MessagingProduct string    `json:"messaging_product"`
	RecipientType    string    `json:"recipient_type"`
	To               string    `json:"to"`
	Type             string    `json:"type"`
	Text             *TextBody `json:"text,omitempty"`
	Image            *Media    `json:"image,omitempty"`
	Audio            *Media    `json:"audio,omitempty"`
	Video            *Media    `json:"video,omitempty"`
	Document         *Document `json:"document,omitempty"`
	Context          *Context  `json:"context,omitempty"`
}

// TextBody represents text message content
type TextBody struct {
	PreviewURL bool   `json:"preview_url"`
	Body       string `json:"body"`
}

// Media represents media content
type Media struct {
	ID      string `json:"id,omitempty"`
	Link    string `json:"link,omitempty"`
	Caption string `json:"caption,omitempty"`
}

// Document represents document content
type Document struct {
	ID       string `json:"id,omitempty"`
	Link     string `json:"link,omitempty"`
	Caption  string `json:"caption,omitempty"`
	Filename string `json:"filename,omitempty"`
}

// Context represents message context (for replies)
type Context struct {
	MessageID string `json:"message_id"`
}

// SendMessageResponse represents the API response
type SendMessageResponse struct {
	MessagingProduct string `json:"messaging_product"`
	Contacts         []struct {
		Input string `json:"input"`
		WaID  string `json:"wa_id"`
	} `json:"contacts"`
	Messages []struct {
		ID string `json:"id"`
	} `json:"messages"`
}

// SendTextMessage sends a text message
func (c *Client) SendTextMessage(to, content string, replyToID *string) (*string, error) {
	request := SendMessageRequest{
		MessagingProduct: "whatsapp",
		RecipientType:    "individual",
		To:               to,
		Type:             "text",
		Text: &TextBody{
			PreviewURL: true,
			Body:       content,
		},
	}

	if replyToID != nil {
		request.Context = &Context{
			MessageID: *replyToID,
		}
	}

	return c.sendMessage(request)
}

// SendImageMessage sends an image message
func (c *Client) SendImageMessage(to, imageURL, caption string, replyToID *string) (*string, error) {
	request := SendMessageRequest{
		MessagingProduct: "whatsapp",
		RecipientType:    "individual",
		To:               to,
		Type:             "image",
		Image: &Media{
			Link:    imageURL,
			Caption: caption,
		},
	}

	if replyToID != nil {
		request.Context = &Context{
			MessageID: *replyToID,
		}
	}

	return c.sendMessage(request)
}

// SendDocumentMessage sends a document message
func (c *Client) SendDocumentMessage(to, documentURL, filename, caption string, replyToID *string) (*string, error) {
	request := SendMessageRequest{
		MessagingProduct: "whatsapp",
		RecipientType:    "individual",
		To:               to,
		Type:             "document",
		Document: &Document{
			Link:     documentURL,
			Filename: filename,
			Caption:  caption,
		},
	}

	if replyToID != nil {
		request.Context = &Context{
			MessageID: *replyToID,
		}
	}

	return c.sendMessage(request)
}

// sendMessage sends a message to the WhatsApp API
func (c *Client) sendMessage(request SendMessageRequest) (*string, error) {
	url := fmt.Sprintf("%s/%s/messages", c.baseURL, c.phoneID)

	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.accessToken)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var response SendMessageResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(response.Messages) == 0 {
		return nil, fmt.Errorf("no message ID in response")
	}

	messageID := response.Messages[0].ID
	return &messageID, nil
}

// SendMessage sends a message and stores it in the database
func (c *Client) SendMessage(conversationID uuid.UUID, userID uuid.UUID, messageType, content string, replyToID *string) (*models.Message, error) {
	// Get conversation details
	var conversation models.Conversation
	if err := c.db.First(&conversation, conversationID).Error; err != nil {
		return nil, fmt.Errorf("conversation not found: %w", err)
	}

	// Get customer details
	var customer models.Customer
	if err := c.db.First(&customer, conversation.CustomerID).Error; err != nil {
		return nil, fmt.Errorf("customer not found: %w", err)
	}

	// Send message via WhatsApp API
	var externalID *string
	var err error

	switch messageType {
	case "text":
		externalID, err = c.SendTextMessage(customer.Phone, content, replyToID)
	case "image":
		// For images, content should contain the URL and optionally caption separated by |
		externalID, err = c.SendImageMessage(customer.Phone, content, "", replyToID)
	case "document":
		// For documents, implement similar logic
		externalID, err = c.SendDocumentMessage(customer.Phone, content, "", "", replyToID)
	default:
		return nil, fmt.Errorf("unsupported message type: %s", messageType)
	}

	if err != nil {
		log.Error().Err(err).
			Str("conversation_id", conversationID.String()).
			Str("type", messageType).
			Msg("Failed to send WhatsApp message")
		return nil, fmt.Errorf("failed to send message: %w", err)
	}

	// Create message record
	message := models.Message{
		BaseTenantModel: models.BaseTenantModel{
			ID:        uuid.New(),
			TenantID:  conversation.TenantID,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		ConversationID: conversationID,
		CustomerID:     conversation.CustomerID,
		UserID:         &userID,
		Type:           messageType,
		Content:        content,
		Direction:      "out",
		Status:         "sent",
		ExternalID:     *externalID,
	}

	if replyToID != nil {
		// Find the message being replied to
		var replyTo models.Message
		if err := c.db.Where("external_id = ?", *replyToID).First(&replyTo).Error; err == nil {
			message.ReplyToID = &replyTo.ID
		}
	}

	// Save message to database
	if err := c.db.Create(&message).Error; err != nil {
		log.Error().Err(err).
			Str("external_id", *externalID).
			Msg("Failed to save outbound message to database")
		return nil, fmt.Errorf("failed to save message: %w", err)
	}

	// Update conversation
	now := time.Now()
	conversation.LastMessageAt = &now
	if err := c.db.Save(&conversation).Error; err != nil {
		log.Error().Err(err).Msg("Failed to update conversation")
	}

	log.Info().
		Str("message_id", message.ID.String()).
		Str("external_id", *externalID).
		Str("conversation_id", conversationID.String()).
		Str("type", messageType).
		Msg("Message sent successfully")

	return &message, nil
}

// MarkAsRead marks a message as read
func (c *Client) MarkAsRead(messageID string) error {
	url := fmt.Sprintf("%s/%s/messages", c.baseURL, c.phoneID)

	request := map[string]interface{}{
		"messaging_product": "whatsapp",
		"status":            "read",
		"message_id":        messageID,
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.accessToken)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	return nil
}
