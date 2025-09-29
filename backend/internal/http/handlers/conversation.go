package handlers

import (
	"net/http"
	"strconv"
	"time"

	"iafarma/internal/services"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type ConversationHandler struct {
	embeddingService *services.EmbeddingService
}

func NewConversationHandler(embeddingService *services.EmbeddingService) *ConversationHandler {
	return &ConversationHandler{
		embeddingService: embeddingService,
	}
}

type StoreConversationRequest struct {
	CustomerID string                 `json:"customer_id" validate:"required"`
	Message    string                 `json:"message" validate:"required"`
	Response   string                 `json:"response" validate:"required"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

type SearchConversationsRequest struct {
	CustomerID string `json:"customer_id" validate:"required"`
	Query      string `json:"query" validate:"required"`
	Limit      int    `json:"limit,omitempty"`
}

type SearchConversationsResponse struct {
	Results []services.ConversationSearchResult `json:"results"`
	Total   int                                 `json:"total"`
}

// StoreConversation armazena uma conversa no Qdrant
func (h *ConversationHandler) StoreConversation(c echo.Context) error {
	var req StoreConversationRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}

	if err := c.Validate(req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	// Extrair informações do contexto (tenant_id do JWT)
	tenantID := c.Get("tenant_id").(string)

	// Criar entrada da conversa
	entry := services.ConversationEntry{
		ID:         uuid.New().String(),
		TenantID:   tenantID,
		CustomerID: req.CustomerID,
		Message:    req.Message,
		Response:   req.Response,
		Timestamp:  time.Now().Format(time.RFC3339),
		Metadata:   req.Metadata,
	}

	// Armazenar no Qdrant
	err := h.embeddingService.StoreConversation(tenantID, req.CustomerID, entry)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to store conversation: " + err.Error(),
		})
	}

	return c.JSON(http.StatusCreated, map[string]interface{}{
		"success": true,
		"id":      entry.ID,
		"message": "Conversation stored successfully",
	})
}

// SearchConversations busca conversas similares usando RAG
func (h *ConversationHandler) SearchConversations(c echo.Context) error {
	var req SearchConversationsRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}

	if err := c.Validate(req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	// Extrair informações do contexto (tenant_id do JWT)
	tenantID := c.Get("tenant_id").(string)

	// Definir limite padrão
	limit := req.Limit
	if limit <= 0 || limit > 50 {
		limit = 10
	}

	// Buscar conversas similares
	results, err := h.embeddingService.SearchConversations(tenantID, req.CustomerID, req.Query, limit)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to search conversations: " + err.Error(),
		})
	}

	response := SearchConversationsResponse{
		Results: results,
		Total:   len(results),
	}

	return c.JSON(http.StatusOK, response)
}

// GetConversationContext retorna contexto de conversas para um customer (últimas conversas + busca semântica)
func (h *ConversationHandler) GetConversationContext(c echo.Context) error {
	customerID := c.Param("customer_id")
	query := c.QueryParam("query")
	limitStr := c.QueryParam("limit")

	if customerID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "customer_id is required"})
	}

	if query == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "query parameter is required"})
	}

	// Extrair informações do contexto (tenant_id do JWT)
	tenantID := c.Get("tenant_id").(string)

	// Definir limite
	limit := 5
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 50 {
			limit = l
		}
	}

	// Buscar conversas similares
	results, err := h.embeddingService.SearchConversations(tenantID, customerID, query, limit)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to get conversation context: " + err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"customer_id": customerID,
		"query":       query,
		"context":     results,
		"total":       len(results),
	})
}
