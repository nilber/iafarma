package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"iafarma/internal/webhook"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

// SimpleWebSocketHandler handles WebSocket connections without tenant validation for testing
type SimpleWebSocketHandler struct {
	db             *gorm.DB
	aiService      interface{}
	webhookHandler *webhook.ZapPlusWebhookHandler
	upgrader       websocket.Upgrader
}

// NewSimpleWebSocketHandler creates a new simple WebSocket handler for testing
func NewSimpleWebSocketHandler(db *gorm.DB, aiService interface{}, webhookHandler *webhook.ZapPlusWebhookHandler) *SimpleWebSocketHandler {
	return &SimpleWebSocketHandler{
		db:             db,
		aiService:      nil, // Will be nil for now
		webhookHandler: webhookHandler,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				// Allow all origins for testing
				return true
			},
		},
	}
}

// HandleSimpleWebSocket handles simple WebSocket connections for testing
func (h *SimpleWebSocketHandler) HandleSimpleWebSocket(c echo.Context) error {
	log.Println("🔌 Nova conexão WebSocket de teste chegando...")

	// Upgrade HTTP connection to WebSocket
	conn, err := h.upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		log.Printf("❌ Erro ao fazer upgrade WebSocket: %v", err)
		return err
	}
	defer conn.Close()

	log.Println("✅ WebSocket conectado com sucesso!")

	// Get token from query parameters
	token := c.QueryParam("token")
	if token == "" {
		log.Println("❌ Token não fornecido")
		conn.WriteMessage(websocket.TextMessage, []byte(`{"error": "Token não fornecido"}`))
		return nil
	}

	// Validate token and get customer info
	customerID, tenantID, err := h.validateTokenAndGetCustomerID(token)
	if err != nil {
		log.Printf("❌ Token inválido: %v", err)
		conn.WriteMessage(websocket.TextMessage, []byte(`{"error": "Token inválido"}`))
		return nil
	}

	log.Printf("✅ Cliente autenticado: %s, Tenant: %s", customerID, tenantID)

	// Send initial message
	initialMsg := map[string]interface{}{
		"type":      "system",
		"message":   "Conectado ao chat com IA! Digite sua mensagem.",
		"timestamp": "now",
	}
	initialJSON, _ := json.Marshal(initialMsg)
	conn.WriteMessage(websocket.TextMessage, initialJSON)

	// Main message loop
	for {
		// Read message from client
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Printf("❌ Erro ao ler mensagem: %v", err)
			break
		}

		log.Printf("📨 Mensagem recebida: %s", string(message))

		// Parse client message
		var clientMsg map[string]interface{}
		if err := json.Unmarshal(message, &clientMsg); err != nil {
			log.Printf("❌ Erro ao parsear mensagem: %v", err)
			continue
		}

		// Extract text from message
		text, ok := clientMsg["text"].(string)
		if !ok {
			log.Println("❌ Mensagem sem texto")
			continue
		}

		// Echo message back to confirm receipt
		echoMsg := map[string]interface{}{
			"type":      "message",
			"text":      text,
			"fromMe":    true,
			"timestamp": "now",
		}
		echoJSON, _ := json.Marshal(echoMsg)
		conn.WriteMessage(websocket.TextMessage, echoJSON)

		// Process with AI (simulated - we'll send a simple response)
		go h.processMessageWithAI(conn, text, customerID, tenantID)
	}

	log.Println("🔌 WebSocket desconectado")
	return nil
}

// validateTokenAndGetCustomerID validates JWT token and extracts customer ID
func (h *SimpleWebSocketHandler) validateTokenAndGetCustomerID(tokenString string) (string, string, error) {
	// Parse JWT token
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Make sure token's signing method is what we expect
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("invalid signing method")
		}
		// Return a default secret key for testing
		return []byte(os.Getenv("JWT_SECRET")), nil
	})

	if err != nil {
		return "", "", err
	}

	if !token.Valid {
		return "", "", errors.New("invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", "", errors.New("invalid claims")
	}

	// Extract user_id and tenant_id from token
	userID, ok := claims["user_id"].(string)
	if !ok {
		return "", "", errors.New("invalid user_id")
	}

	tenantID, ok := claims["tenant_id"].(string)
	if !ok {
		return "", "", errors.New("invalid tenant_id")
	}

	// Verify user role (should be customer for store access)
	role, ok := claims["role"].(string)
	if !ok {
		return "", "", errors.New("invalid role")
	}

	// For testing, allow tenant_admin as well
	if role != "customer" && role != "tenant_admin" {
		return "", "", errors.New("unauthorized role")
	}

	return userID, tenantID, nil
}

// processMessageWithAI processes message with AI and sends response
func (h *SimpleWebSocketHandler) processMessageWithAI(conn *websocket.Conn, text, customerID, tenantID string) {
	log.Printf("🤖 Processando mensagem com IA: %s", text)

	// Create a fake webhook payload to simulate WhatsApp message
	fakeWebhookData := map[string]interface{}{
		"id":        fmt.Sprintf("chat_msg_%d", time.Now().Unix()),
		"timestamp": time.Now().Unix(),
		"event":     "message",
		"session":   "chat",
		"metadata": map[string]interface{}{
			"tenantId": tenantID,
		},
		"me": map[string]interface{}{
			"id":       "chat_system",
			"pushName": "Chat IA",
		},
		"payload": map[string]interface{}{
			"id":        fmt.Sprintf("msg_%d", time.Now().Unix()),
			"timestamp": time.Now().Unix(),
			"from":      customerID + "@chat.us", // Simular formato WhatsApp mas com chat
			"fromMe":    false,
			"source":    "web",
			"to":        "chat_system@chat.us",
			"body":      text,
			"hasMedia":  false,
			"ack":       1,
			"ackName":   "sent",
		},
		"engine":      "chat", // Diferente de "whatsapp"
		"environment": map[string]interface{}{},
	}

	// Convert to JSON
	_, err := json.Marshal(fakeWebhookData)
	if err != nil {
		log.Printf("❌ Erro ao serializar webhook fake: %v", err)
		return
	}

	log.Printf("🔄 Processando via sistema de webhook simulado...")

	// Process through the webhook handler (this will handle AI, save to DB, etc.)
	// This simulates the normal WhatsApp flow but with source="chat"
	go func() {
		// Create a fake HTTP request context for the webhook
		// The webhook handler will process this and generate AI response
		// For now, we'll send a simple response, but this is where the real AI integration happens

		response := "🤖 Mensagem processada! Você disse: \"" + text + "\". Em breve implementaremos a IA completa aqui."

		// Send AI response via WebSocket
		aiMsg := map[string]interface{}{
			"type":      "message",
			"text":      response,
			"fromMe":    false,
			"timestamp": "now",
			"source":    "ai",
		}

		aiJSON, err := json.Marshal(aiMsg)
		if err != nil {
			log.Printf("❌ Erro ao serializar resposta da IA: %v", err)
			return
		}

		if err := conn.WriteMessage(websocket.TextMessage, aiJSON); err != nil {
			log.Printf("❌ Erro ao enviar resposta da IA: %v", err)
			return
		}

		log.Println("✅ Resposta da IA enviada com sucesso!")
	}()
}
