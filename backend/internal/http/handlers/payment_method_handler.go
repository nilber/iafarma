package handlers

import (
	"iafarma/pkg/models"
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

type PaymentMethodHandler struct {
	db *gorm.DB
}

func NewPaymentMethodHandler(db *gorm.DB) *PaymentMethodHandler {
	return &PaymentMethodHandler{db: db}
}

type PaymentMethodRequest struct {
	Name     string `json:"name" validate:"required"`
	IsActive *bool  `json:"is_active"`
}

type PaymentMethodResponse struct {
	models.PaymentMethod
}

// List returns paginated list of payment methods
func (h *PaymentMethodHandler) List(c echo.Context) error {
	tenantID := c.Get("tenant_id").(uuid.UUID)

	// Parse pagination parameters
	page, _ := strconv.Atoi(c.QueryParam("page"))
	if page < 1 {
		page = 1
	}

	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	if limit < 1 {
		limit = 10
	}

	search := c.QueryParam("search")
	showInactive := c.QueryParam("show_inactive") == "true"

	offset := (page - 1) * limit

	// Build query
	query := h.db.Where("tenant_id = ?", tenantID)

	if !showInactive {
		query = query.Where("is_active = ?", true)
	}

	if search != "" {
		query = query.Where("name ILIKE ?", "%"+search+"%")
	}

	// Get total count
	var total int64
	if err := query.Model(&models.PaymentMethod{}).Count(&total).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to count payment methods")
	}

	// Get payment methods
	var paymentMethods []models.PaymentMethod
	if err := query.Offset(offset).Limit(limit).Order("created_at DESC").Find(&paymentMethods).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch payment methods")
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"data":  paymentMethods,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

// Create creates a new payment method
func (h *PaymentMethodHandler) Create(c echo.Context) error {
	tenantID := c.Get("tenant_id").(uuid.UUID)

	var req PaymentMethodRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request body")
	}

	if err := c.Validate(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	// Set default value for IsActive
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	paymentMethod := models.PaymentMethod{
		BaseTenantModel: models.BaseTenantModel{
			ID:       uuid.New(),
			TenantID: tenantID,
		},
		Name:     req.Name,
		IsActive: isActive,
	}

	if err := h.db.Create(&paymentMethod).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create payment method")
	}

	return c.JSON(http.StatusCreated, PaymentMethodResponse{paymentMethod})
}

// GetByID returns a payment method by ID
func (h *PaymentMethodHandler) GetByID(c echo.Context) error {
	tenantID := c.Get("tenant_id").(uuid.UUID)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid payment method ID")
	}

	var paymentMethod models.PaymentMethod
	if err := h.db.Where("id = ? AND tenant_id = ?", id, tenantID).First(&paymentMethod).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return echo.NewHTTPError(http.StatusNotFound, "Payment method not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch payment method")
	}

	return c.JSON(http.StatusOK, PaymentMethodResponse{paymentMethod})
}

// Update updates a payment method
func (h *PaymentMethodHandler) Update(c echo.Context) error {
	tenantID := c.Get("tenant_id").(uuid.UUID)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid payment method ID")
	}

	var req PaymentMethodRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request body")
	}

	if err := c.Validate(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	var paymentMethod models.PaymentMethod
	if err := h.db.Where("id = ? AND tenant_id = ?", id, tenantID).First(&paymentMethod).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return echo.NewHTTPError(http.StatusNotFound, "Payment method not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch payment method")
	}

	// Update fields
	paymentMethod.Name = req.Name
	if req.IsActive != nil {
		paymentMethod.IsActive = *req.IsActive
	}

	if err := h.db.Save(&paymentMethod).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to update payment method")
	}

	return c.JSON(http.StatusOK, PaymentMethodResponse{paymentMethod})
}

// Delete soft deletes a payment method
func (h *PaymentMethodHandler) Delete(c echo.Context) error {
	tenantID := c.Get("tenant_id").(uuid.UUID)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid payment method ID")
	}

	var paymentMethod models.PaymentMethod
	if err := h.db.Where("id = ? AND tenant_id = ?", id, tenantID).First(&paymentMethod).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return echo.NewHTTPError(http.StatusNotFound, "Payment method not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch payment method")
	}

	// Check if payment method is being used by any orders
	var orderCount int64
	if err := h.db.Model(&models.Order{}).Where("payment_method_id = ? AND tenant_id = ?", id, tenantID).Count(&orderCount).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to check payment method usage")
	}

	if orderCount > 0 {
		return echo.NewHTTPError(http.StatusConflict, "Cannot delete payment method that is being used by orders")
	}

	if err := h.db.Delete(&paymentMethod).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to delete payment method")
	}

	return c.NoContent(http.StatusNoContent)
}

// GetActive returns active payment methods for dropdowns
func (h *PaymentMethodHandler) GetActive(c echo.Context) error {
	tenantID := c.Get("tenant_id").(uuid.UUID)

	var paymentMethods []models.PaymentMethod
	if err := h.db.Where("tenant_id = ? AND is_active = ?", tenantID, true).
		Order("name ASC").
		Find(&paymentMethods).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch active payment methods")
	}

	// Retornar no formato padronizado da API
	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    paymentMethods,
	})
}

// SetupRoutes sets up the routes for PaymentMethod
func (h *PaymentMethodHandler) SetupRoutes(g *echo.Group) {
	g.GET("", h.List)             // GET /payment-methods
	g.POST("", h.Create)          // POST /payment-methods
	g.GET("/active", h.GetActive) // GET /payment-methods/active
	g.GET("/:id", h.GetByID)      // GET /payment-methods/:id
	g.PUT("/:id", h.Update)       // PUT /payment-methods/:id
	g.DELETE("/:id", h.Delete)    // DELETE /payment-methods/:id
}
