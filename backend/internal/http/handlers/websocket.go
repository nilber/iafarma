package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"iafarma/internal/auth"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

// WebSocketMessage represents a message sent through WebSocket
type WebSocketMessage struct {
	Type      string      `json:"type"`
	Data      interface{} `json:"data"`
	Timestamp time.Time   `json:"timestamp"`
	TenantID  string      `json:"tenant_id,omitempty"`
}

// WebSocketClient represents a connected WebSocket client
type WebSocketClient struct {
	conn     *websocket.Conn
	tenantID string
	send     chan WebSocketMessage
	hub      *WebSocketHub
}

// WebSocketHub manages all WebSocket connections
type WebSocketHub struct {
	clients    map[*WebSocketClient]bool
	broadcast  chan WebSocketMessage
	register   chan *WebSocketClient
	unregister chan *WebSocketClient
	db         *gorm.DB
	mu         sync.RWMutex
}

// WebSocketHandler handles WebSocket connections
type WebSocketHandler struct {
	hub         *WebSocketHub
	authService *auth.Service
}

// NewWebSocketHandler creates a new WebSocket handler
func NewWebSocketHandler(db *gorm.DB, authService *auth.Service) *WebSocketHandler {
	hub := &WebSocketHub{
		clients:    make(map[*WebSocketClient]bool),
		broadcast:  make(chan WebSocketMessage),
		register:   make(chan *WebSocketClient),
		unregister: make(chan *WebSocketClient),
		db:         db,
	}

	go hub.run()
	return &WebSocketHandler{
		hub:         hub,
		authService: authService,
	}
}

// Upgrader configures the websocket upgrader
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// Allow connections from any origin in development
		// In production, check against allowed origins
		return true
	},
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// HandleWebSocket handles WebSocket connection upgrades
func (h *WebSocketHandler) HandleWebSocket(c echo.Context) error {
	var tenantID string

	// Try to get tenant from context first (if JWT middleware was applied)
	if tid, ok := c.Get("tenant_id").(string); ok {
		tenantID = tid
	} else {
		// If not in context, try to get token from query parameter and validate manually
		token := c.QueryParam("token")
		if token == "" {
			return echo.NewHTTPError(http.StatusUnauthorized, "Missing authorization token")
		}

		// Validate JWT token manually
		claims, err := h.authService.ValidateToken(token)
		if err != nil {
			return echo.NewHTTPError(http.StatusUnauthorized, "Invalid token: "+err.Error())
		}

		tenantID = claims.TenantID.String()
	}

	// Upgrade HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return err
	}

	// Create new client
	client := &WebSocketClient{
		conn:     conn,
		tenantID: tenantID,
		send:     make(chan WebSocketMessage, 256),
		hub:      h.hub,
	}

	// Register client with hub
	h.hub.register <- client

	// Start goroutines for reading and writing
	go client.writePump()
	go client.readPump()

	return nil
}

// BroadcastToTenant broadcasts a message to all clients of a specific tenant
func (h *WebSocketHandler) BroadcastToTenant(tenantID string, messageType string, data interface{}) {
	message := WebSocketMessage{
		Type:      messageType,
		Data:      data,
		Timestamp: time.Now(),
		TenantID:  tenantID,
	}

	h.hub.broadcast <- message
}

// run manages the WebSocket hub
func (hub *WebSocketHub) run() {
	for {
		select {
		case client := <-hub.register:
			hub.mu.Lock()
			hub.clients[client] = true
			hub.mu.Unlock()
			log.Printf("WebSocket client connected for tenant: %s", client.tenantID)

			// Send welcome message
			welcome := WebSocketMessage{
				Type:      "connection",
				Data:      map[string]string{"status": "connected"},
				Timestamp: time.Now(),
			}
			select {
			case client.send <- welcome:
			default:
				close(client.send)
				delete(hub.clients, client)
			}

		case client := <-hub.unregister:
			hub.mu.Lock()
			if _, ok := hub.clients[client]; ok {
				delete(hub.clients, client)
				close(client.send)
				log.Printf("WebSocket client disconnected for tenant: %s", client.tenantID)
			}
			hub.mu.Unlock()

		case message := <-hub.broadcast:
			hub.mu.RLock()
			for client := range hub.clients {
				// Only send to clients of the same tenant (if tenantID is specified)
				if message.TenantID == "" || client.tenantID == message.TenantID {
					select {
					case client.send <- message:
					default:
						close(client.send)
						delete(hub.clients, client)
					}
				}
			}
			hub.mu.RUnlock()
		}
	}
}

// readPump handles reading messages from the WebSocket
func (c *WebSocketClient) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	// Set read deadline and pong handler - 30s timeout since we ping every 20s
	c.conn.SetReadDeadline(time.Now().Add(30 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(30 * time.Second))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		// Handle incoming messages (ping, etc.)
		var msg WebSocketMessage
		if err := json.Unmarshal(message, &msg); err == nil {
			switch msg.Type {
			case "ping":
				pong := WebSocketMessage{
					Type:      "pong",
					Data:      map[string]string{"status": "ok"},
					Timestamp: time.Now(),
				}
				select {
				case c.send <- pong:
				default:
					return
				}
			}
		}
	}
}

// writePump handles writing messages to the WebSocket
func (c *WebSocketClient) writePump() {
	ticker := time.NewTicker(20 * time.Second) // Send ping every 20 seconds
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			// Send the message
			if err := c.conn.WriteJSON(message); err != nil {
				log.Printf("WebSocket write error: %v", err)
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Printf("WebSocket ping failed: %v", err)
				return
			}
		}
	}
}

// GetConnectedClients returns the number of connected clients
func (h *WebSocketHandler) GetConnectedClients() int {
	h.hub.mu.RLock()
	defer h.hub.mu.RUnlock()
	return len(h.hub.clients)
}

// BroadcastWebhookNotification sends a webhook notification to connected clients
func (h *WebSocketHandler) BroadcastWebhookNotification(tenantID string, webhookType string, data interface{}) {
	notification := map[string]interface{}{
		"webhook_type": webhookType,
		"data":         data,
		"timestamp":    time.Now(),
	}

	h.BroadcastToTenant(tenantID, "webhook_notification", notification)
}
