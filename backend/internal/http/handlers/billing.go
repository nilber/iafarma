package handlers

import (
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"iafarma/internal/repo"
	"iafarma/pkg/models"
)

type BillingHandler struct {
	planRepo *repo.PlanRepository
}

func NewBillingHandler(planRepo *repo.PlanRepository) *BillingHandler {
	return &BillingHandler{
		planRepo: planRepo,
	}
}

// @Summary List all plans
// @Description Get all available plans for SuperAdmin
// @Tags billing
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {array} models.Plan
// @Router /admin/plans [get]
func (h *BillingHandler) ListPlans(c echo.Context) error {
	// O middleware RequireSystemRole já validou que é system_admin
	plans, err := h.planRepo.GetAll()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Erro ao buscar planos")
	}

	return c.JSON(http.StatusOK, plans)
}

// @Summary Get plan by ID
// @Description Get detailed information about a specific plan
// @Tags billing
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param id path string true "Plan ID"
// @Success 200 {object} models.Plan
// @Router /admin/plans/{id} [get]
func (h *BillingHandler) GetPlan(c echo.Context) error {
	// O middleware RequireSystemRole já validou que é system_admin
	planID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "ID do plano inválido")
	}

	plan, err := h.planRepo.GetByID(planID)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Plano não encontrado")
	}

	return c.JSON(http.StatusOK, plan)
}

// @Summary Create new plan
// @Description Create a new billing plan (SuperAdmin only)
// @Tags billing
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param plan body models.Plan true "Plan data"
// @Success 201 {object} models.Plan
// @Router /admin/plans [post]
func (h *BillingHandler) CreatePlan(c echo.Context) error {
	// O middleware RequireSystemRole já validou que é system_admin
	var plan models.Plan
	if err := c.Bind(&plan); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, map[string]string{
			"message": "Dados do plano inválidos",
			"error":   err.Error(),
		})
	}

	// Validação detalhada dos campos obrigatórios
	validationErrors := make(map[string]string)

	if plan.Name == "" {
		validationErrors["name"] = "Nome é obrigatório"
	}
	if plan.Price < 0 {
		validationErrors["price"] = "Preço deve ser maior ou igual a zero"
	}
	if plan.MaxConversations <= 0 {
		validationErrors["max_conversations"] = "Número máximo de conversas deve ser maior que zero"
	}
	if plan.MaxMessagesPerMonth <= 0 {
		validationErrors["max_messages_per_month"] = "Número máximo de mensagens por mês deve ser maior que zero"
	}
	if plan.MaxProducts <= 0 {
		validationErrors["max_products"] = "Número máximo de produtos deve ser maior que zero"
	}
	if plan.MaxChannels <= 0 {
		validationErrors["max_channels"] = "Número máximo de canais deve ser maior que zero"
	}
	if plan.MaxCreditsPerMonth <= 0 {
		validationErrors["max_credits_per_month"] = "Número máximo de créditos por mês deve ser maior que zero"
	}
	if plan.Currency == "" {
		plan.Currency = "BRL" // Valor padrão
	}
	if plan.BillingPeriod == "" {
		plan.BillingPeriod = "monthly" // Valor padrão
	}

	// Se há erros de validação, retornar detalhes
	if len(validationErrors) > 0 {
		return echo.NewHTTPError(http.StatusBadRequest, map[string]interface{}{
			"message":           "Dados do plano inválidos",
			"validation_errors": validationErrors,
		})
	}

	if err := h.planRepo.Create(&plan); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Erro ao criar plano")
	}

	return c.JSON(http.StatusCreated, plan)
}

// @Summary Update plan
// @Description Update an existing plan (SuperAdmin only)
// @Tags billing
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param id path string true "Plan ID"
// @Param plan body models.Plan true "Updated plan data"
// @Success 200 {object} models.Plan
// @Router /admin/plans/{id} [put]
func (h *BillingHandler) UpdatePlan(c echo.Context) error {
	// O middleware RequireSystemRole já validou que é system_admin
	planID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "ID do plano inválido")
	}

	var plan models.Plan
	if err := c.Bind(&plan); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, map[string]interface{}{
			"message": "Dados do plano inválidos",
			"error":   err.Error(),
		})
	}

	// Validação detalhada dos campos obrigatórios
	validationErrors := make(map[string]string)

	if plan.Name == "" {
		validationErrors["name"] = "Nome é obrigatório"
	}

	if plan.Price < 0 {
		validationErrors["price"] = "Preço deve ser maior ou igual a zero"
	}

	if plan.MaxConversations <= 0 {
		validationErrors["max_conversations"] = "Número máximo de conversas deve ser maior que zero"
	}

	if plan.MaxMessagesPerMonth <= 0 {
		validationErrors["max_messages_per_month"] = "Número máximo de mensagens por mês deve ser maior que zero"
	}

	if plan.MaxProducts <= 0 {
		validationErrors["max_products"] = "Número máximo de produtos deve ser maior que zero"
	}

	if plan.MaxChannels <= 0 {
		validationErrors["max_channels"] = "Número máximo de canais deve ser maior que zero"
	}

	if plan.MaxCreditsPerMonth <= 0 {
		validationErrors["max_credits_per_month"] = "Número máximo de créditos por mês deve ser maior que zero"
	}

	// Se houver erros de validação, retornar com detalhes
	if len(validationErrors) > 0 {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"message":           "Dados do plano inválidos",
			"validation_errors": validationErrors,
		})
	}

	// Definir valores padrão se não fornecidos
	if plan.Currency == "" {
		plan.Currency = "BRL"
	}
	if plan.BillingPeriod == "" {
		plan.BillingPeriod = "monthly"
	}

	plan.ID = planID
	if err := h.planRepo.Update(&plan); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Erro ao atualizar plano")
	}

	return c.JSON(http.StatusOK, plan)
}

// @Summary Delete plan
// @Description Delete a plan (SuperAdmin only)
// @Tags billing
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param id path string true "Plan ID"
// @Success 204
// @Router /admin/plans/{id} [delete]
func (h *BillingHandler) DeletePlan(c echo.Context) error {
	// O middleware RequireSystemRole já validou que é system_admin
	planID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "ID do plano inválido")
	}

	if err := h.planRepo.Delete(planID); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Erro ao excluir plano")
	}

	return c.NoContent(http.StatusNoContent)
}

// @Summary Update tenant plan
// @Description Change a tenant's billing plan (SuperAdmin only)
// @Tags billing
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param tenant_id path string true "Tenant ID"
// @Param plan_id path string true "Plan ID"
// @Success 200 {object} map[string]string
// @Router /admin/tenants/{tenant_id}/plan/{plan_id} [put]
func (h *BillingHandler) UpdateTenantPlan(c echo.Context) error {
	// O middleware RequireSystemRole já validou que é system_admin
	tenantID, err := uuid.Parse(c.Param("tenant_id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "ID do tenant inválido")
	}

	planID, err := uuid.Parse(c.Param("plan_id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "ID do plano inválido")
	}

	if err := h.planRepo.UpdateTenantPlan(tenantID, planID); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Erro ao atualizar plano do tenant")
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Plano do tenant atualizado com sucesso",
	})
}

// @Summary Get tenant usage
// @Description Get current usage statistics for a tenant
// @Tags billing
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} models.TenantUsage
// @Router /billing/usage [get]
func (h *BillingHandler) GetTenantUsage(c echo.Context) error {
	// Verificar o role do usuário
	userRole := c.Get("user_role")
	if userRole == nil {
		return echo.NewHTTPError(http.StatusForbidden, "User role not found")
	}

	roleStr := userRole.(string)

	// Para system_admin, este endpoint não é aplicável - deve usar /admin/tenants/{id}/usage
	if roleStr == "system_admin" {
		return echo.NewHTTPError(http.StatusBadRequest, "System admin deve usar /admin/tenants/{tenant_id}/usage")
	}

	// Para tenant users, usar o TenantID do contexto
	tenantID, ok := c.Get("tenant_id").(uuid.UUID)
	if !ok {
		return echo.NewHTTPError(http.StatusBadRequest, "Contexto de tenant inválido")
	}

	usage, err := h.planRepo.GetTenantUsage(tenantID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Erro ao buscar uso do tenant")
	}

	return c.JSON(http.StatusOK, usage)
}

// @Summary Get tenant usage (Admin)
// @Description Get current usage statistics for a specific tenant (SuperAdmin only)
// @Tags billing
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param tenant_id path string true "Tenant ID"
// @Success 200 {object} models.TenantUsage
// @Router /admin/tenants/{tenant_id}/usage [get]
func (h *BillingHandler) GetTenantUsageByID(c echo.Context) error {
	// O middleware RequireSystemRole já validou que é system_admin
	tenantID, err := uuid.Parse(c.Param("tenant_id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "ID do tenant inválido")
	}

	usage, err := h.planRepo.GetTenantUsage(tenantID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Erro ao buscar uso do tenant")
	}

	return c.JSON(http.StatusOK, usage)
}

// @Summary Check usage limit
// @Description Check if tenant can perform an action based on usage limits
// @Tags billing
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param resource_type query string true "Resource type (messages, credits, conversations, products, channels)"
// @Param amount query int false "Amount to check (default: 1)"
// @Success 200 {object} models.UsageLimitResult
// @Router /billing/usage/check [get]
func (h *BillingHandler) CheckUsageLimit(c echo.Context) error {
	// Verificar o role do usuário
	userRole := c.Get("user_role")
	if userRole == nil {
		return echo.NewHTTPError(http.StatusForbidden, "User role not found")
	}

	roleStr := userRole.(string)

	// Para system_admin, este endpoint não é aplicável
	if roleStr == "system_admin" {
		return echo.NewHTTPError(http.StatusBadRequest, "System admin não precisa verificar limites de uso")
	}

	// Para tenant users, usar o TenantID do contexto
	tenantID, ok := c.Get("tenant_id").(uuid.UUID)
	if !ok {
		return echo.NewHTTPError(http.StatusBadRequest, "Contexto de tenant inválido")
	}

	resourceType := c.QueryParam("resource_type")
	if resourceType == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Tipo de recurso é obrigatório")
	}

	amount := 1
	if amountStr := c.QueryParam("amount"); amountStr != "" {
		if parsedAmount, err := strconv.Atoi(amountStr); err == nil {
			amount = parsedAmount
		}
	}

	result, err := h.planRepo.CheckUsageLimit(tenantID, resourceType, amount)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Erro ao verificar limite de uso")
	}

	return c.JSON(http.StatusOK, result)
}

// @Summary Increment tenant usage
// @Description Increment usage counter for a tenant (for internal API calls)
// @Tags billing
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param resource_type path string true "Resource type"
// @Param amount body map[string]int true "Amount to increment"
// @Success 200 {object} map[string]string
// @Router /billing/usage/{resource_type}/increment [post]
func (h *BillingHandler) IncrementUsage(c echo.Context) error {
	// Verificar o role do usuário
	userRole := c.Get("user_role")
	if userRole == nil {
		return echo.NewHTTPError(http.StatusForbidden, "User role not found")
	}

	roleStr := userRole.(string)

	// Para system_admin, este endpoint não é aplicável
	if roleStr == "system_admin" {
		return echo.NewHTTPError(http.StatusBadRequest, "System admin não incrementa uso de tenant")
	}

	// Para tenant users, usar o TenantID do contexto
	tenantID, ok := c.Get("tenant_id").(uuid.UUID)
	if !ok {
		return echo.NewHTTPError(http.StatusBadRequest, "Contexto de tenant inválido")
	}

	resourceType := c.Param("resource_type")

	var body map[string]int
	if err := c.Bind(&body); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Dados inválidos")
	}

	amount, exists := body["amount"]
	if !exists || amount <= 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "Quantidade deve ser maior que 0")
	}

	if err := h.planRepo.IncrementUsage(tenantID, resourceType, amount); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Erro ao incrementar uso")
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Uso incrementado com sucesso",
	})
}

// @Summary Reset tenant billing cycle
// @Description Reset billing cycle for a tenant (SuperAdmin only)
// @Tags billing
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param tenant_id path string true "Tenant ID"
// @Success 200 {object} map[string]string
// @Router /admin/tenants/{tenant_id}/billing/reset [post]
func (h *BillingHandler) ResetBillingCycle(c echo.Context) error {
	// O middleware RequireSystemRole já validou que é system_admin
	tenantID, err := uuid.Parse(c.Param("tenant_id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "ID do tenant inválido")
	}

	if err := h.planRepo.ResetBillingCycle(tenantID); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Erro ao resetar ciclo de billing")
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Ciclo de billing resetado com sucesso",
	})
}
