package handlers

import (
	"context"
	"iafarma/internal/ai"
	"iafarma/internal/repo"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

// ProductAIHandler handles AI-related product operations
type ProductAIHandler struct {
	aiService    *ai.ProductAIService
	aiCreditRepo *repo.AICreditRepository
}

// NewProductAIHandler creates a new product AI handler
func NewProductAIHandler(db *gorm.DB) *ProductAIHandler {
	openaiKey := os.Getenv("OPENAI_API_KEY")
	if openaiKey == "" {
		// Log warning but don't fail - allow the handler to be created
		// The actual API calls will fail gracefully
	}

	return &ProductAIHandler{
		aiService:    ai.NewProductAIService(openaiKey),
		aiCreditRepo: repo.NewAICreditRepository(db),
	}
}

// GenerateProductInfoRequest represents the request to generate product info
type GenerateProductInfoRequest struct {
	ProductName string `json:"product_name" validate:"required"`
}

// GenerateProductInfo godoc
// @Summary Generate product information with AI
// @Description Generate product description and tags using AI based on product name
// @Tags ai-products
// @Accept json
// @Produce json
// @Param request body GenerateProductInfoRequest true "Generate product info request"
// @Success 200 {object} ai.ProductGenerationResponse
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 402 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /ai/products/generate [post]
// @Security BearerAuth
func (h *ProductAIHandler) GenerateProductInfo(c echo.Context) error {
	// Get tenant ID from context
	tenantID, ok := c.Get("tenant_id").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid tenant"})
	}

	// Get user ID from context
	userID := c.Get("user_id").(uuid.UUID)

	var req GenerateProductInfoRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request format"})
	}

	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	// Check if OpenAI key is configured
	if os.Getenv("OPENAI_API_KEY") == "" {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "AI service not configured"})
	}

	// Get usage estimate
	creditCost := h.aiService.GetUsageEstimate(req.ProductName)

	// Check if tenant has enough credits
	credits, err := h.aiCreditRepo.GetByTenantID(tenantID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to check credits"})
	}

	if credits.RemainingCredits < creditCost {
		return c.JSON(http.StatusPaymentRequired, map[string]interface{}{
			"error":     "Insufficient AI credits",
			"required":  creditCost,
			"available": credits.RemainingCredits,
		})
	}

	// Use credits first (before making the API call)
	err = h.aiCreditRepo.UseCredits(
		tenantID,
		&userID,
		creditCost,
		"AI product generation for: "+req.ProductName,
		"product",
		nil,
	)
	if err != nil {
		return c.JSON(http.StatusPaymentRequired, map[string]string{"error": "Failed to reserve credits"})
	}

	// Generate product info with AI
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := h.aiService.GenerateProductInfo(ctx, req.ProductName)
	if err != nil {
		// If AI generation fails, we should ideally refund the credits
		// For now, we'll just log the error and return it
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to generate product information: " + err.Error(),
		})
	}

	// Get updated credits to return to the client
	updatedCredits, err := h.aiCreditRepo.GetByTenantID(tenantID)
	if err != nil {
		// Don't fail the request if we can't get updated credits
		// Just use the calculated value
		return c.JSON(http.StatusOK, map[string]interface{}{
			"product_info":      result,
			"credits_used":      creditCost,
			"remaining_credits": credits.RemainingCredits - creditCost,
			"message":           "Product information generated successfully",
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"product_info":      result,
		"credits_used":      creditCost,
		"remaining_credits": updatedCredits.RemainingCredits,
		"message":           "Product information generated successfully",
	})
}

// GetCreditEstimate godoc
// @Summary Get credit estimate for product generation
// @Description Get estimated credit cost for generating product information
// @Tags ai-products
// @Accept json
// @Produce json
// @Param request body GenerateProductInfoRequest true "Product name for estimate"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /ai/products/estimate [post]
// @Security BearerAuth
func (h *ProductAIHandler) GetCreditEstimate(c echo.Context) error {
	// Get tenant ID from context
	_, ok := c.Get("tenant_id").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid tenant"})
	}

	var req GenerateProductInfoRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request format"})
	}

	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	estimate := h.aiService.GetUsageEstimate(req.ProductName)

	return c.JSON(http.StatusOK, map[string]interface{}{
		"estimated_credits": estimate,
		"product_name":      req.ProductName,
	})
}
