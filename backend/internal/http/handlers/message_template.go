package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"iafarma/internal/repo"
	"iafarma/pkg/models"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

// MessageTemplateHandler handles message template operations
type MessageTemplateHandler struct {
	templateRepo *repo.MessageTemplateRepository
	db           *gorm.DB
}

// NewMessageTemplateHandler creates a new message template handler
func NewMessageTemplateHandler(templateRepo *repo.MessageTemplateRepository, db *gorm.DB) *MessageTemplateHandler {
	return &MessageTemplateHandler{
		templateRepo: templateRepo,
		db:           db,
	}
}

// CreateTemplateRequest represents the request to create a message template
type CreateTemplateRequest struct {
	Title       string   `json:"title" validate:"required"`
	Content     string   `json:"content" validate:"required"`
	Variables   []string `json:"variables"`
	Category    string   `json:"category"`
	Description string   `json:"description"`
}

// UpdateTemplateRequest represents the request to update a message template
type UpdateTemplateRequest struct {
	Title       string   `json:"title" validate:"required"`
	Content     string   `json:"content" validate:"required"`
	Variables   []string `json:"variables"`
	Category    string   `json:"category"`
	Description string   `json:"description"`
}

// ProcessTemplateRequest represents the request to process a template with variables
type ProcessTemplateRequest struct {
	TemplateID uuid.UUID         `json:"template_id" validate:"required"`
	Variables  map[string]string `json:"variables" validate:"required"`
}

// ProcessTemplateResponse represents the response with processed template
type ProcessTemplateResponse struct {
	ProcessedContent string `json:"processed_content"`
	OriginalContent  string `json:"original_content"`
	TemplateTitle    string `json:"template_title"`
}

// List godoc
// @Summary List user's message templates
// @Description Get list of message templates for the authenticated user
// @Tags message-templates
// @Accept json
// @Produce json
// @Param limit query int false "Limit" default(50)
// @Param offset query int false "Offset" default(0)
// @Param category query string false "Filter by category"
// @Param search query string false "Search in title and content"
// @Success 200 {array} models.MessageTemplate
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /message-templates [get]
// @Security BearerAuth
func (h *MessageTemplateHandler) List(c echo.Context) error {
	// Get user ID from context
	userID, ok := c.Get("user_id").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "user_id not found in context"})
	}

	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	offset, _ := strconv.Atoi(c.QueryParam("offset"))
	category := c.QueryParam("category")
	search := c.QueryParam("search")

	if limit <= 0 {
		limit = 50
	}

	var templates []*models.MessageTemplate
	var err error

	if search != "" {
		templates, err = h.templateRepo.Search(userID, search, limit, offset)
	} else if category != "" {
		templates, err = h.templateRepo.ListByUserAndCategory(userID, category, limit, offset)
	} else {
		templates, err = h.templateRepo.ListByUser(userID, limit, offset)
	}

	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, templates)
}

// GetByID godoc
// @Summary Get message template by ID
// @Description Get a message template by its ID
// @Tags message-templates
// @Accept json
// @Produce json
// @Param id path string true "Template ID"
// @Success 200 {object} models.MessageTemplate
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /message-templates/{id} [get]
// @Security BearerAuth
func (h *MessageTemplateHandler) GetByID(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid ID format"})
	}

	userID, ok := c.Get("user_id").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "user_id not found in context"})
	}

	template, err := h.templateRepo.GetByIDAndUser(id, userID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Template not found"})
	}

	return c.JSON(http.StatusOK, template)
}

// Create godoc
// @Summary Create message template
// @Description Create a new message template for the authenticated user
// @Tags message-templates
// @Accept json
// @Produce json
// @Param template body CreateTemplateRequest true "Template data"
// @Success 201 {object} models.MessageTemplate
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /message-templates [post]
// @Security BearerAuth
func (h *MessageTemplateHandler) Create(c echo.Context) error {
	var req CreateTemplateRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request"})
	}

	// Get user ID and tenant ID from context
	userID, ok := c.Get("user_id").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "user_id not found in context"})
	}

	tenantID, ok := c.Get("tenant_id").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "tenant_id not found in context"})
	}

	// Convert variables to JSON
	variablesJSON, _ := json.Marshal(req.Variables)

	template := &models.MessageTemplate{
		UserID:      userID,
		Title:       req.Title,
		Content:     req.Content,
		Variables:   string(variablesJSON),
		Category:    req.Category,
		Description: req.Description,
		IsActive:    true,
		UsageCount:  0,
	}
	template.TenantID = tenantID

	if err := h.templateRepo.Create(template); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, template)
}

// Update godoc
// @Summary Update message template
// @Description Update an existing message template
// @Tags message-templates
// @Accept json
// @Produce json
// @Param id path string true "Template ID"
// @Param template body UpdateTemplateRequest true "Template data"
// @Success 200 {object} models.MessageTemplate
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /message-templates/{id} [put]
// @Security BearerAuth
func (h *MessageTemplateHandler) Update(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid ID format"})
	}

	userID, ok := c.Get("user_id").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "user_id not found in context"})
	}

	var req UpdateTemplateRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request"})
	}

	// Get existing template
	template, err := h.templateRepo.GetByIDAndUser(id, userID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Template not found"})
	}

	// Convert variables to JSON
	variablesJSON, _ := json.Marshal(req.Variables)

	// Update fields
	template.Title = req.Title
	template.Content = req.Content
	template.Variables = string(variablesJSON)
	template.Category = req.Category
	template.Description = req.Description

	if err := h.templateRepo.Update(template); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, template)
}

// Delete godoc
// @Summary Delete message template
// @Description Delete a message template (soft delete)
// @Tags message-templates
// @Accept json
// @Produce json
// @Param id path string true "Template ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /message-templates/{id} [delete]
// @Security BearerAuth
func (h *MessageTemplateHandler) Delete(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid ID format"})
	}

	userID, ok := c.Get("user_id").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "user_id not found in context"})
	}

	if err := h.templateRepo.Delete(id, userID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "Template deleted successfully"})
}

// GetCategories godoc
// @Summary Get template categories
// @Description Get all categories for user's templates
// @Tags message-templates
// @Accept json
// @Produce json
// @Success 200 {array} string
// @Failure 500 {object} map[string]string
// @Router /message-templates/categories [get]
// @Security BearerAuth
func (h *MessageTemplateHandler) GetCategories(c echo.Context) error {
	userID, ok := c.Get("user_id").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "user_id not found in context"})
	}

	categories, err := h.templateRepo.GetCategories(userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, categories)
}

// ProcessTemplate godoc
// @Summary Process template with variables
// @Description Process a template by replacing variables with provided values
// @Tags message-templates
// @Accept json
// @Produce json
// @Param request body ProcessTemplateRequest true "Process request"
// @Success 200 {object} ProcessTemplateResponse
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /message-templates/process [post]
// @Security BearerAuth
func (h *MessageTemplateHandler) ProcessTemplate(c echo.Context) error {
	var req ProcessTemplateRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request"})
	}

	userID, ok := c.Get("user_id").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "user_id not found in context"})
	}

	// Get template
	template, err := h.templateRepo.GetByIDAndUser(req.TemplateID, userID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Template not found"})
	}

	// Process template content
	processedContent := template.Content
	for variable, value := range req.Variables {
		placeholder := "{{" + variable + "}}"
		processedContent = strings.ReplaceAll(processedContent, placeholder, value)
	}

	// Increment usage count
	go h.templateRepo.IncrementUsageCount(template.ID)

	response := ProcessTemplateResponse{
		ProcessedContent: processedContent,
		OriginalContent:  template.Content,
		TemplateTitle:    template.Title,
	}

	return c.JSON(http.StatusOK, response)
}
