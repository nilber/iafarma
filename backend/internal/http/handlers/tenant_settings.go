package handlers

import (
	"fmt"
	"iafarma/internal/ai"
	"iafarma/pkg/models"
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

type TenantSettingsHandler struct {
	settingsService *ai.TenantSettingsService
}

func NewTenantSettingsHandler(db *gorm.DB) *TenantSettingsHandler {
	return &TenantSettingsHandler{
		settingsService: ai.NewTenantSettingsService(db),
	}
}

// GetSettings retrieves all settings for the current tenant
func (h *TenantSettingsHandler) GetSettings(c echo.Context) error {
	tenantID, ok := c.Get("tenant_id").(uuid.UUID)
	if !ok {
		return echo.NewHTTPError(http.StatusBadRequest, "tenant ID requerido")
	}

	settings, err := h.settingsService.GetAllSettings(c.Request().Context(), tenantID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "erro ao buscar configurações")
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success":  true,
		"settings": settings,
	})
}

// GetSetting retrieves a specific setting
func (h *TenantSettingsHandler) GetSetting(c echo.Context) error {
	tenantID, ok := c.Get("tenant_id").(uuid.UUID)
	if !ok {
		return echo.NewHTTPError(http.StatusBadRequest, "tenant ID requerido")
	}

	key := c.Param("key")
	if key == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "chave da configuração requerida")
	}

	setting, err := h.settingsService.GetSetting(c.Request().Context(), tenantID, key)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return echo.NewHTTPError(http.StatusNotFound, "configuração não encontrada")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "erro ao buscar configuração")
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"setting": setting,
	})
}

// UpdateSetting updates or creates a setting
func (h *TenantSettingsHandler) UpdateSetting(c echo.Context) error {
	tenantID, ok := c.Get("tenant_id").(uuid.UUID)
	if !ok {
		return echo.NewHTTPError(http.StatusBadRequest, "tenant ID requerido")
	}

	key := c.Param("key")
	if key == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "chave da configuração requerida")
	}

	var request struct {
		Value *string `json:"value"`
		Type  string  `json:"type,omitempty"`
	}

	if err := c.Bind(&request); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "dados inválidos")
	}

	settingType := request.Type
	if settingType == "" {
		settingType = "string"
	}

	err := h.settingsService.SetSetting(c.Request().Context(), tenantID, key, request.Value, settingType)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "erro ao atualizar configuração")
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Configuração atualizada com sucesso",
	})
}

// GenerateAIExamples generates AI product examples based on tenant's products
func (h *TenantSettingsHandler) GenerateAIExamples(c echo.Context) error {
	tenantID, ok := c.Get("tenant_id").(uuid.UUID)
	if !ok {
		return echo.NewHTTPError(http.StatusBadRequest, "tenant ID requerido")
	}

	examples, err := h.settingsService.GenerateAIProductExamples(c.Request().Context(), tenantID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "erro ao gerar exemplos")
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success":  true,
		"examples": examples,
	})
}

// GetWelcomeMessage retrieves the welcome message (custom or auto-generated)
func (h *TenantSettingsHandler) GetWelcomeMessage(c echo.Context) error {
	tenantID, ok := c.Get("tenant_id").(uuid.UUID)
	if !ok {
		return echo.NewHTTPError(http.StatusBadRequest, "tenant ID requerido")
	}

	message, err := h.settingsService.GetWelcomeMessage(c.Request().Context(), tenantID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "erro ao buscar mensagem de boas-vindas")
	}

	// Check if it's custom or auto-generated
	customWelcomeSetting, _ := h.settingsService.GetSetting(c.Request().Context(), tenantID, "ai_welcome_message")
	isCustom := customWelcomeSetting != nil && customWelcomeSetting.SettingValue != nil && *customWelcomeSetting.SettingValue != ""

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success":   true,
		"message":   message,
		"is_custom": isCustom,
	})
}

// GenerateWelcomeMessage generates a welcome message based on tenant's business
func (h *TenantSettingsHandler) GenerateWelcomeMessage(c echo.Context) error {
	tenantID, ok := c.Get("tenant_id").(uuid.UUID)
	if !ok {
		return echo.NewHTTPError(http.StatusBadRequest, "tenant ID requerido")
	}

	message, err := h.settingsService.GenerateWelcomeMessage(c.Request().Context(), tenantID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "erro ao gerar mensagem de boas-vindas")
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"message": message,
	})
}

// GenerateAutoPrompt generates a complete AI prompt automatically
func (h *TenantSettingsHandler) GenerateAutoPrompt(c echo.Context) error {
	tenantID, ok := c.Get("tenant_id").(uuid.UUID)
	if !ok {
		return echo.NewHTTPError(http.StatusBadRequest, "tenant ID requerido")
	}

	// Generate full system prompt with dynamic examples
	autoPrompt, err := h.settingsService.GenerateFullSystemPrompt(c.Request().Context(), tenantID)
	if err != nil {
		// Log the error for debugging
		fmt.Printf("Error generating prompt: %v\n", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "erro ao gerar prompt completo")
	}

	// Generate examples for response
	examples, err := h.settingsService.GenerateAIProductExamples(c.Request().Context(), tenantID)
	if err != nil {
		// Log the error for debugging
		fmt.Printf("Error generating examples: %v\n", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "erro ao gerar exemplos")
	}

	// Generate welcome message
	welcomeMessage, err := h.settingsService.GenerateWelcomeMessage(c.Request().Context(), tenantID)
	if err != nil {
		// Log the error for debugging
		fmt.Printf("Error generating welcome message: %v\n", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "erro ao gerar mensagem de boas-vindas")
	}

	// Save the auto-generated prompt
	err = h.settingsService.SetSetting(c.Request().Context(), tenantID, "ai_system_prompt_template", &autoPrompt, "text")
	if err != nil {
		// Log the error for debugging
		fmt.Printf("Error saving prompt: %v\n", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "erro ao salvar prompt gerado")
	}

	// Save the auto-generated welcome message
	err = h.settingsService.SetSetting(c.Request().Context(), tenantID, "ai_welcome_message", &welcomeMessage, "text")
	if err != nil {
		// Log the error for debugging
		fmt.Printf("Error saving welcome message: %v\n", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "erro ao salvar mensagem de boas-vindas")
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success":        true,
		"prompt":         autoPrompt,
		"examples":       examples,
		"welcomeMessage": welcomeMessage,
		"message":        "Prompt e mensagem de boas-vindas gerados e salvos automaticamente com base nos produtos do seu catálogo",
	})
}

// ResetToDefault resets AI prompt to use auto-generation
func (h *TenantSettingsHandler) ResetToDefault(c echo.Context) error {
	tenantID, ok := c.Get("tenant_id").(uuid.UUID)
	if !ok {
		return echo.NewHTTPError(http.StatusBadRequest, "tenant ID requerido")
	}

	// Set prompt to null to use auto-generation
	err := h.settingsService.SetSetting(c.Request().Context(), tenantID, "ai_system_prompt_template", nil, "text")
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "erro ao resetar configuração")
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Prompt resetado para geração automática baseada nos produtos",
	})
}

// SetWhatsAppGroupProxy sets the WhatsApp group proxy setting for the tenant
func (h *TenantSettingsHandler) SetWhatsAppGroupProxy(c echo.Context) error {
	tenantID, ok := c.Get("tenant_id").(uuid.UUID)
	if !ok {
		return echo.NewHTTPError(http.StatusBadRequest, "tenant ID requerido")
	}

	type WhatsAppGroupProxyRequest struct {
		Enabled bool `json:"enabled"`
	}

	var req WhatsAppGroupProxyRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "dados inválidos")
	}

	// Convert boolean to string for storage
	value := "false"
	if req.Enabled {
		value = "true"
	}

	// Save setting
	err := h.settingsService.SetSetting(c.Request().Context(), tenantID, "enable_whatsapp_group_proxy", &value, "boolean")
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "erro ao salvar configuração")
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"enabled": req.Enabled,
		"message": "Configuração de proxy de grupo WhatsApp atualizada",
	})
}

// GetWhatsAppGroupProxy gets the WhatsApp group proxy setting for the tenant
func (h *TenantSettingsHandler) GetWhatsAppGroupProxy(c echo.Context) error {
	tenantID, ok := c.Get("tenant_id").(uuid.UUID)
	if !ok {
		return echo.NewHTTPError(http.StatusBadRequest, "tenant ID requerido")
	}

	setting, err := h.settingsService.GetSetting(c.Request().Context(), tenantID, "enable_whatsapp_group_proxy")
	if err != nil {
		// Default to false if setting doesn't exist
		return c.JSON(http.StatusOK, map[string]interface{}{
			"enabled": false,
		})
	}

	enabled := false
	if setting.SettingValue != nil && *setting.SettingValue == "true" {
		enabled = true
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"enabled": enabled,
	})
}

// GetContextLimitation retrieves the AI context limitation setting for sales tenants
func (h *TenantSettingsHandler) GetContextLimitation(c echo.Context) error {
	tenantID, ok := c.Get("tenant_id").(uuid.UUID)
	if !ok {
		return echo.NewHTTPError(http.StatusBadRequest, "tenant ID requerido")
	}

	// Buscar a configuração personalizada
	customSetting, err := h.settingsService.GetSetting(c.Request().Context(), tenantID, "ai_context_limitation_custom")
	if err != nil {
		// Se não encontrou, retornar o padrão
		return c.JSON(http.StatusOK, map[string]interface{}{
			"success":    true,
			"limitation": getDefaultContextLimitation(),
			"isCustom":   false,
			"hasCustom":  false,
		})
	}

	var limitation string
	var hasCustom bool
	if customSetting.SettingValue != nil && *customSetting.SettingValue != "" {
		limitation = *customSetting.SettingValue
		hasCustom = true
	} else {
		limitation = getDefaultContextLimitation()
		hasCustom = false
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success":    true,
		"limitation": limitation,
		"isCustom":   hasCustom,
		"hasCustom":  hasCustom,
	})
}

// SetContextLimitation updates the AI context limitation setting for sales tenants
func (h *TenantSettingsHandler) SetContextLimitation(c echo.Context) error {
	tenantID, ok := c.Get("tenant_id").(uuid.UUID)
	if !ok {
		return echo.NewHTTPError(http.StatusBadRequest, "tenant ID requerido")
	}

	type ContextLimitationRequest struct {
		Limitation string `json:"limitation"`
	}

	var req ContextLimitationRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "dados inválidos")
	}

	// Validar que a limitação não está vazia
	if req.Limitation == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "limitação de contexto não pode estar vazia")
	}

	// Salvar a configuração
	err := h.settingsService.SetSetting(c.Request().Context(), tenantID, "ai_context_limitation_custom", &req.Limitation, "text")
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "erro ao salvar configuração")
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Limitação de contexto personalizada salva com sucesso",
	})
}

// ResetContextLimitation resets the context limitation to default
func (h *TenantSettingsHandler) ResetContextLimitation(c echo.Context) error {
	tenantID, ok := c.Get("tenant_id").(uuid.UUID)
	if !ok {
		return echo.NewHTTPError(http.StatusBadRequest, "tenant ID requerido")
	}

	// Remover a configuração personalizada (usar NULL para voltar ao padrão)
	err := h.settingsService.SetSetting(c.Request().Context(), tenantID, "ai_context_limitation_custom", nil, "text")
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "erro ao resetar configuração")
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success":    true,
		"message":    "Limitação de contexto resetada para o padrão",
		"limitation": getDefaultContextLimitation(),
	})
}

// getDefaultContextLimitation retorna o texto padrão da limitação de contexto
func getDefaultContextLimitation() string {
	return `🚨 LIMITAÇÃO DE CONTEXTO - SUPER IMPORTANTE:
- Você é um ASSISTENTE DE VENDAS, não um assistente geral
- NUNCA responda perguntas sobre: política, notícias, medicina, direito, aposentadoria, educação, tecnologia geral, ou qualquer assunto não relacionado à nossa loja
- Para perguntas fora do contexto, responda: "Sou um assistente focado em vendas da nossa loja. Como posso ajudá-lo com nossos produtos ou serviços?"
- SEMPRE redirecione conversas para produtos, pedidos, entregas ou informações da loja
- Sua função é EXCLUSIVAMENTE ajudar com vendas e atendimento comercial`
}

// GetChatCustomization retrieves chat customization settings for the current tenant
func (h *TenantSettingsHandler) GetChatCustomization(c echo.Context) error {
	tenantID, ok := c.Get("tenant_id").(uuid.UUID)
	if !ok {
		return echo.NewHTTPError(http.StatusBadRequest, "tenant ID requerido")
	}

	// Buscar as configurações de chat existentes
	primaryColor, err := h.settingsService.GetSetting(c.Request().Context(), tenantID, "chat_primary_color")
	if err != nil && err != gorm.ErrRecordNotFound {
		return echo.NewHTTPError(http.StatusInternalServerError, "Erro ao buscar cor primária")
	}

	secondaryColor, err := h.settingsService.GetSetting(c.Request().Context(), tenantID, "chat_secondary_color")
	if err != nil && err != gorm.ErrRecordNotFound {
		return echo.NewHTTPError(http.StatusInternalServerError, "Erro ao buscar cor secundária")
	}

	accentColor, err := h.settingsService.GetSetting(c.Request().Context(), tenantID, "chat_accent_color")
	if err != nil && err != gorm.ErrRecordNotFound {
		return echo.NewHTTPError(http.StatusInternalServerError, "Erro ao buscar cor de destaque")
	}

	backgroundColor, err := h.settingsService.GetSetting(c.Request().Context(), tenantID, "chat_background_color")
	if err != nil && err != gorm.ErrRecordNotFound {
		return echo.NewHTTPError(http.StatusInternalServerError, "Erro ao buscar cor de fundo")
	}

	textColor, err := h.settingsService.GetSetting(c.Request().Context(), tenantID, "chat_text_color")
	if err != nil && err != gorm.ErrRecordNotFound {
		return echo.NewHTTPError(http.StatusInternalServerError, "Erro ao buscar cor do texto")
	}

	botTextColor, err := h.settingsService.GetSetting(c.Request().Context(), tenantID, "chat_bot_text_color")
	if err != nil && err != gorm.ErrRecordNotFound {
		return echo.NewHTTPError(http.StatusInternalServerError, "Erro ao buscar cor do texto do bot")
	}

	// Valores padrão se não encontrados
	defaultPrimary := "#3B82F6"
	defaultSecondary := "#1E40AF"
	defaultAccent := "#60A5FA"
	defaultBackground := "#F8FAFC"
	defaultTextColor := "#333333"    // Texto escuro para fundo claro
	defaultBotTextColor := "#FFFFFF" // Texto claro para fundo escuro

	result := map[string]interface{}{
		"primary_color":    getStringValue(primaryColor, defaultPrimary),
		"secondary_color":  getStringValue(secondaryColor, defaultSecondary),
		"accent_color":     getStringValue(accentColor, defaultAccent),
		"background_color": getStringValue(backgroundColor, defaultBackground),
		"text_color":       getStringValue(textColor, defaultTextColor),
		"bot_text_color":   getStringValue(botTextColor, defaultBotTextColor),
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    result,
	})
}

// UpdateChatCustomization updates chat customization settings
func (h *TenantSettingsHandler) UpdateChatCustomization(c echo.Context) error {
	tenantID, ok := c.Get("tenant_id").(uuid.UUID)
	if !ok {
		return echo.NewHTTPError(http.StatusBadRequest, "tenant ID requerido")
	}

	var req struct {
		PrimaryColor    string `json:"primary_color"`
		SecondaryColor  string `json:"secondary_color"`
		AccentColor     string `json:"accent_color"`
		BackgroundColor string `json:"background_color"`
		TextColor       string `json:"text_color"`
		BotTextColor    string `json:"bot_text_color"`
	}

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Dados inválidos")
	}

	// Validar que as cores são válidas
	colorSettings := map[string]string{
		"chat_primary_color":    req.PrimaryColor,
		"chat_secondary_color":  req.SecondaryColor,
		"chat_accent_color":     req.AccentColor,
		"chat_background_color": req.BackgroundColor,
		"chat_text_color":       req.TextColor,
		"chat_bot_text_color":   req.BotTextColor,
	}

	// Salvar cada configuração
	for key, value := range colorSettings {
		if value != "" {
			err := h.settingsService.SetSetting(c.Request().Context(), tenantID, key, &value, "string")
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Erro ao salvar %s: %v", key, err))
			}
		}
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Configurações de chat atualizadas com sucesso",
	})
}

// ResetChatCustomization resets chat customization to default values
func (h *TenantSettingsHandler) ResetChatCustomization(c echo.Context) error {
	tenantID, ok := c.Get("tenant_id").(uuid.UUID)
	if !ok {
		return echo.NewHTTPError(http.StatusBadRequest, "tenant ID requerido")
	}

	// Cores padrão
	defaultSettings := map[string]string{
		"chat_primary_color":    "#3B82F6",
		"chat_secondary_color":  "#1E40AF",
		"chat_accent_color":     "#60A5FA",
		"chat_background_color": "#F8FAFC",
		"chat_text_color":       "#333333",
		"chat_bot_text_color":   "#FFFFFF",
	}

	// Salvar configurações padrão
	for key, value := range defaultSettings {
		err := h.settingsService.SetSetting(c.Request().Context(), tenantID, key, &value, "string")
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Erro ao resetar %s: %v", key, err))
		}
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Configurações de chat resetadas para os valores padrão",
	})
}

// Helper function to get string value from TenantSetting
func getStringValue(setting *models.TenantSetting, defaultValue string) string {
	if setting == nil {
		return defaultValue
	}
	if setting.SettingValue != nil {
		return *setting.SettingValue
	}
	return defaultValue
}
