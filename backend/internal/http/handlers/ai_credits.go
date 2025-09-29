package handlers

import (
	"iafarma/internal/repo"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

// AICreditHandler handles AI credit requests
type AICreditHandler struct {
	aiCreditRepo *repo.AICreditRepository
}

// NewAICreditHandler creates a new AI credit handler
func NewAICreditHandler(db *gorm.DB) *AICreditHandler {
	return &AICreditHandler{
		aiCreditRepo: repo.NewAICreditRepository(db),
	}
}

// AICreditResponse represents the AI credits response for Swagger
type AICreditResponse struct {
	ID               uuid.UUID  `json:"id"`
	TenantID         uuid.UUID  `json:"tenant_id"`
	TotalCredits     int        `json:"total_credits"`
	UsedCredits      int        `json:"used_credits"`
	RemainingCredits int        `json:"remaining_credits"`
	LastUpdatedBy    *uuid.UUID `json:"last_updated_by,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

// GetCredits godoc
// @Summary Get AI credits for tenant
// @Description Get AI credits balance for the current tenant
// @Tags ai-credits
// @Accept json
// @Produce json
// @Success 200 {object} AICreditResponse
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /ai-credits [get]
// @Security BearerAuth
func (h *AICreditHandler) GetCredits(c echo.Context) error {
	// Get tenant ID from context
	tenantID, ok := c.Get("tenant_id").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid tenant"})
	}

	credits, err := h.aiCreditRepo.GetByTenantID(tenantID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to get credits"})
	}

	return c.JSON(http.StatusOK, credits)
}

// AddCreditsRequest represents the request to add credits
type AddCreditsRequest struct {
	TenantID    string `json:"tenant_id" validate:"required"`
	Amount      int    `json:"amount" validate:"required,min=1"`
	Description string `json:"description" validate:"required"`
}

// AddCredits godoc
// @Summary Add AI credits to tenant (Admin only)
// @Description Add AI credits to a specific tenant - only system admin can do this
// @Tags ai-credits
// @Accept json
// @Produce json
// @Param request body AddCreditsRequest true "Add credits request"
// @Success 200 {object} AICreditResponse
// @Failure 400 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /admin/ai-credits/add [post]
// @Security BearerAuth
func (h *AICreditHandler) AddCredits(c echo.Context) error {
	// Check if user is system admin
	userRole := c.Get("user_role").(string)
	if userRole != "system_admin" {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "Only system admin can add credits"})
	}

	var req AddCreditsRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request format"})
	}

	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	// Parse tenant ID
	tenantUUID, err := uuid.Parse(req.TenantID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid tenant ID"})
	}

	// Get user ID from context
	userID := c.Get("user_id").(uuid.UUID)

	// Add credits
	err = h.aiCreditRepo.AddCredits(tenantUUID, &userID, req.Amount, req.Description)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to add credits"})
	}

	// Get updated credits
	credits, err := h.aiCreditRepo.GetByTenantID(tenantUUID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to get updated credits"})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "Credits added successfully",
		"credits": credits,
	})
}

// UseCreditsRequest represents the request to use credits
type UseCreditsRequest struct {
	Amount        int    `json:"amount" validate:"required,min=1"`
	Description   string `json:"description" validate:"required"`
	RelatedEntity string `json:"related_entity,omitempty"`
	RelatedID     string `json:"related_id,omitempty"`
}

// UseCredits godoc
// @Summary Use AI credits
// @Description Use AI credits for AI operations (like product description generation)
// @Tags ai-credits
// @Accept json
// @Produce json
// @Param request body UseCreditsRequest true "Use credits request"
// @Success 200 {object} AICreditResponse
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 402 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /ai-credits/use [post]
// @Security BearerAuth
func (h *AICreditHandler) UseCredits(c echo.Context) error {
	// Get tenant ID from context
	tenantID, ok := c.Get("tenant_id").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid tenant"})
	}

	// Get user ID from context
	userID := c.Get("user_id").(uuid.UUID)

	var req UseCreditsRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request format"})
	}

	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	// Parse related ID if provided
	var relatedID *uuid.UUID
	if req.RelatedID != "" {
		parsedID, err := uuid.Parse(req.RelatedID)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid related ID"})
		}
		relatedID = &parsedID
	}

	// Use credits
	err := h.aiCreditRepo.UseCredits(tenantID, &userID, req.Amount, req.Description, req.RelatedEntity, relatedID)
	if err != nil {
		if err == gorm.ErrCheckConstraintViolated {
			return c.JSON(http.StatusPaymentRequired, map[string]string{"error": "Insufficient credits"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to use credits"})
	}

	// Get updated credits
	credits, err := h.aiCreditRepo.GetByTenantID(tenantID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to get updated credits"})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "Credits used successfully",
		"credits": credits,
	})
}

// GetTransactions godoc
// @Summary Get AI credit transactions
// @Description Get AI credit transaction history for the current tenant
// @Tags ai-credits
// @Accept json
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20)
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /ai-credits/transactions [get]
// @Security BearerAuth
func (h *AICreditHandler) GetTransactions(c echo.Context) error {
	// Get tenant ID from context
	tenantID, ok := c.Get("tenant_id").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid tenant"})
	}

	// Parse pagination parameters
	page, _ := strconv.Atoi(c.QueryParam("page"))
	if page < 1 {
		page = 1
	}

	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	if limit < 1 || limit > 100 {
		limit = 20
	}

	offset := (page - 1) * limit

	transactions, err := h.aiCreditRepo.GetTransactionsByTenantID(tenantID, limit, offset)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to get transactions"})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"transactions": transactions,
		"page":         page,
		"limit":        limit,
	})
}

// GetCreditsByTenantID godoc
// @Summary Get AI credits for specific tenant (Admin only)
// @Description Get AI credits balance for a specific tenant
// @Tags admin-ai-credits
// @Accept json
// @Produce json
// @Param tenant_id path string true "Tenant ID"
// @Success 200 {object} AICreditResponse
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /admin/ai-credits/tenant/{tenant_id} [get]
// @Security BearerAuth
func (h *AICreditHandler) GetCreditsByTenantID(c echo.Context) error {
	// Parse tenant ID from URL
	tenantIDStr := c.Param("tenant_id")
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid tenant ID"})
	}

	credits, err := h.aiCreditRepo.GetByTenantID(tenantID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to get credits"})
	}

	return c.JSON(http.StatusOK, credits)
}

// GetTransactionsByTenantID godoc
// @Summary Get AI credit transactions for specific tenant (Admin only)
// @Description Get AI credit transaction history for a specific tenant
// @Tags admin-ai-credits
// @Accept json
// @Produce json
// @Param tenant_id path string true "Tenant ID"
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20)
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /admin/ai-credits/tenant/{tenant_id}/transactions [get]
// @Security BearerAuth
func (h *AICreditHandler) GetTransactionsByTenantID(c echo.Context) error {
	// Parse tenant ID from URL
	tenantIDStr := c.Param("tenant_id")
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid tenant ID"})
	}

	// Parse pagination parameters
	page, _ := strconv.Atoi(c.QueryParam("page"))
	if page < 1 {
		page = 1
	}

	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	if limit < 1 || limit > 100 {
		limit = 20
	}

	offset := (page - 1) * limit

	transactions, err := h.aiCreditRepo.GetTransactionsByTenantID(tenantID, limit, offset)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to get transactions"})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"transactions": transactions,
		"page":         page,
		"limit":        limit,
	})
}
