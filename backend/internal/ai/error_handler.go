package ai

import (
	"encoding/json"
	"strings"
	"time"

	"iafarma/pkg/models"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

type ErrorHandler struct {
	db *gorm.DB
}

func NewErrorHandler(db *gorm.DB) *ErrorHandler {
	return &ErrorHandler{db: db}
}

// LogAIError registra um erro da IA e retorna uma mensagem amigável para o usuário
func (eh *ErrorHandler) LogAIError(tenantID, customerID uuid.UUID, customerPhone, userMessage, toolName string, toolArgs map[string]interface{}, err error) string {
	// Determinar tipo de erro
	errorType := eh.categorizeError(err)

	// Converter argumentos para JSON
	argsJSON, _ := json.Marshal(toolArgs)

	// Gerar mensagem amigável para o usuário
	userResponse := eh.generateUserFriendlyMessage(toolName, errorType, err)

	// Determinar severidade
	severity := eh.determineSeverity(errorType, err)

	// Criar log de erro
	errorLog := models.AIErrorLog{
		BaseTenantModel: models.BaseTenantModel{
			ID:       uuid.New(),
			TenantID: tenantID,
		},
		CustomerPhone: customerPhone,
		CustomerID:    customerID,
		UserMessage:   userMessage,
		ToolName:      toolName,
		ToolArgs:      string(argsJSON),
		ErrorMessage:  err.Error(),
		ErrorType:     errorType,
		UserResponse:  userResponse,
		Severity:      severity,
		Resolved:      false,
	}

	// Salvar no banco de dados
	if dbErr := eh.db.Create(&errorLog).Error; dbErr != nil {
		log.Error().
			Err(dbErr).
			Str("tenant_id", tenantID.String()).
			Str("customer_phone", customerPhone).
			Msg("Failed to save AI error log")
	} else {
		log.Info().
			Str("error_log_id", errorLog.ID.String()).
			Str("error_type", errorType).
			Str("severity", severity).
			Msg("AI error logged successfully")
	}

	return userResponse
}

// categorizeError determina o tipo de erro baseado na mensagem
func (eh *ErrorHandler) categorizeError(err error) string {
	errorMsg := strings.ToLower(err.Error())

	switch {
	case strings.Contains(errorMsg, "relation") && strings.Contains(errorMsg, "does not exist"):
		return "database_schema"
	case strings.Contains(errorMsg, "connection") || strings.Contains(errorMsg, "timeout"):
		return "database_connection"
	case strings.Contains(errorMsg, "invalid") || strings.Contains(errorMsg, "validation"):
		return "validation"
	case strings.Contains(errorMsg, "not found") || strings.Contains(errorMsg, "record not found"):
		return "not_found"
	case strings.Contains(errorMsg, "permission") || strings.Contains(errorMsg, "unauthorized"):
		return "permission"
	case strings.Contains(errorMsg, "openai") || strings.Contains(errorMsg, "api"):
		return "external_api"
	case strings.Contains(errorMsg, "json") || strings.Contains(errorMsg, "unmarshal"):
		return "parsing"
	default:
		return "unknown"
	}
}

// determineSeverity determina a severidade do erro
func (eh *ErrorHandler) determineSeverity(errorType string, err error) string {
	switch errorType {
	case "database_schema", "database_connection":
		return "critical"
	case "external_api", "not_found":
		return "warning"
	case "validation", "parsing":
		return "error"
	default:
		return "error"
	}
}

// generateUserFriendlyMessage gera uma mensagem amigável baseada no tipo de erro
func (eh *ErrorHandler) generateUserFriendlyMessage(toolName, errorType string, err error) string {
	switch errorType {
	case "database_schema", "database_connection":
		return "😔 Ops! Estamos com um problema técnico temporário. Nossa equipe já foi notificada e está trabalhando para resolver. Tente novamente em alguns minutos."

	case "not_found":
		switch toolName {
		case "adicionarProdutoPorNome", "consultarItens":
			return "😔 Não encontrei esse produto no momento. Que tal ver nossa lista completa de produtos disponíveis? Digite 'produtos' para ver o catálogo."
		case "verCarrinho":
			return "🛒 Seu carrinho está vazio no momento. Que tal adicionar alguns produtos? Digite 'produtos' para ver o que temos disponível."
		default:
			return "😔 Não encontrei o que você está procurando. Posso ajudar de outra forma?"
		}

	case "validation":
		switch toolName {
		case "adicionarProdutoPorNome", "adicionarAoCarrinho":
			return "🤔 Preciso de mais informações para adicionar o produto. Tente especificar o nome do produto e a quantidade, por exemplo: 'adicione 2 sabonetes'."
		default:
			return "🤔 Verifique se as informações estão corretas e tente novamente."
		}

	case "external_api":
		return "😔 Estamos com dificuldades para processar sua solicitação no momento. Nossa equipe técnica já foi notificada. Tente novamente em alguns minutos."

	case "permission":
		return "🔒 Você não tem permissão para realizar esta ação. Entre em contato com nosso suporte se precisar de ajuda."

	default:
		return "😔 Algo não saiu como esperado. Nossa equipe foi notificada e está investigando. Posso ajudar de outra forma?"
	}
}

// GetErrorLogs retorna logs de erro com paginação (para o super admin)
func (eh *ErrorHandler) GetErrorLogs(tenantID *uuid.UUID, page, limit int, severity string, resolved *bool) ([]models.AIErrorLog, int64, error) {
	var logs []models.AIErrorLog
	var total int64

	query := eh.db.Model(&models.AIErrorLog{})

	// Filtrar por tenant se especificado (super admin pode ver todos)
	if tenantID != nil {
		query = query.Where("tenant_id = ?", *tenantID)
	}

	// Filtrar por severidade
	if severity != "" {
		query = query.Where("severity = ?", severity)
	}

	// Filtrar por status de resolução
	if resolved != nil {
		query = query.Where("resolved = ?", *resolved)
	}

	// Contar total
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Buscar com paginação
	offset := (page - 1) * limit
	if err := query.Order("created_at DESC").Offset(offset).Limit(limit).Find(&logs).Error; err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}

// MarkErrorAsResolved marca um erro como resolvido
func (eh *ErrorHandler) MarkErrorAsResolved(errorID, resolvedBy uuid.UUID) error {
	now := time.Now()
	return eh.db.Model(&models.AIErrorLog{}).
		Where("id = ?", errorID).
		Updates(map[string]interface{}{
			"resolved":    true,
			"resolved_at": &now,
			"resolved_by": resolvedBy,
		}).Error
}

// GetErrorStats retorna estatísticas de erros
func (eh *ErrorHandler) GetErrorStats(tenantID *uuid.UUID, days int) (map[string]interface{}, error) {
	var stats map[string]interface{} = make(map[string]interface{})

	// Data limite
	since := time.Now().AddDate(0, 0, -days)

	query := eh.db.Model(&models.AIErrorLog{}).Where("created_at >= ?", since)
	if tenantID != nil {
		query = query.Where("tenant_id = ?", *tenantID)
	}

	// Total de erros
	var totalErrors int64
	if err := query.Count(&totalErrors).Error; err != nil {
		return nil, err
	}
	stats["total_errors"] = totalErrors

	// Erros por severidade
	var severityStats []struct {
		Severity string `json:"severity"`
		Count    int64  `json:"count"`
	}
	if err := query.Select("severity, COUNT(*) as count").Group("severity").Find(&severityStats).Error; err != nil {
		return nil, err
	}
	stats["by_severity"] = severityStats

	// Erros por tipo
	var typeStats []struct {
		ErrorType string `json:"error_type"`
		Count     int64  `json:"count"`
	}
	if err := query.Select("error_type, COUNT(*) as count").Group("error_type").Find(&typeStats).Error; err != nil {
		return nil, err
	}
	stats["by_type"] = typeStats

	// Erros resolvidos vs não resolvidos
	var resolvedCount, unresolvedCount int64
	query.Where("resolved = true").Count(&resolvedCount)
	query.Where("resolved = false").Count(&unresolvedCount)
	stats["resolved"] = resolvedCount
	stats["unresolved"] = unresolvedCount

	return stats, nil
}
