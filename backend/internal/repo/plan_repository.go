package repo

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"iafarma/pkg/models"
)

type PlanRepository struct {
	db *gorm.DB
}

func NewPlanRepository(db *gorm.DB) *PlanRepository {
	return &PlanRepository{db: db}
}

// Plan CRUD operations
func (r *PlanRepository) GetAll() ([]models.Plan, error) {
	var plans []models.Plan
	err := r.db.Where("is_active = ?", true).Find(&plans).Error
	return plans, err
}

func (r *PlanRepository) GetByID(id uuid.UUID) (*models.Plan, error) {
	var plan models.Plan
	err := r.db.First(&plan, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &plan, nil
}

func (r *PlanRepository) GetByName(name string) (*models.Plan, error) {
	var plan models.Plan
	err := r.db.First(&plan, "name = ? AND is_active = ?", name, true).Error
	if err != nil {
		return nil, err
	}
	return &plan, nil
}

func (r *PlanRepository) Create(plan *models.Plan) error {
	return r.db.Create(plan).Error
}

func (r *PlanRepository) Update(plan *models.Plan) error {
	return r.db.Save(plan).Error
}

func (r *PlanRepository) Delete(id uuid.UUID) error {
	return r.db.Model(&models.Plan{}).Where("id = ?", id).Update("is_active", false).Error
}

// TenantUsage operations
func (r *PlanRepository) GetTenantUsage(tenantID uuid.UUID) (*models.TenantUsage, error) {
	var usage models.TenantUsage
	err := r.db.Preload("Plan").First(&usage, "tenant_id = ?", tenantID).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// Get tenant to find their plan
			var tenant models.Tenant
			if err := r.db.First(&tenant, "id = ?", tenantID).Error; err != nil {
				return nil, fmt.Errorf("tenant not found: %v", err)
			}

			// Get default plan if tenant doesn't have one
			var planID uuid.UUID
			if tenant.PlanID != nil {
				planID = *tenant.PlanID
			} else {
				// Get the default free plan
				var defaultPlan models.Plan
				if err := r.db.First(&defaultPlan, "name = ?", "free").Error; err != nil {
					return nil, fmt.Errorf("default plan not found: %v", err)
				}
				planID = defaultPlan.ID
			}

			// Create default usage record with tenant's plan
			now := time.Now()
			usage = models.TenantUsage{
				TenantID:           tenantID,
				PlanID:             planID,
				MessagesUsed:       0,
				CreditsUsed:        0,
				BillingCycleStart:  time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()),
				BillingCycleEnd:    time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, now.Location()).AddDate(0, 0, -1),
				ConversationsCount: 0,
				ProductsCount:      0,
				ChannelsCount:      0,
			}

			// Create the record in database
			if createErr := r.db.Create(&usage).Error; createErr != nil {
				return nil, fmt.Errorf("failed to create usage record: %v", createErr)
			}

			// Load the plan relationship
			if err := r.db.Preload("Plan").First(&usage, "tenant_id = ?", tenantID).Error; err != nil {
				return nil, fmt.Errorf("failed to load usage with plan: %v", err)
			}

			return &usage, nil
		}
		return nil, err
	}
	return &usage, nil
}

func (r *PlanRepository) CreateOrUpdateTenantUsage(usage *models.TenantUsage) error {
	var existingUsage models.TenantUsage
	err := r.db.First(&existingUsage, "tenant_id = ?", usage.TenantID).Error

	if err == gorm.ErrRecordNotFound {
		// Create new usage record
		return r.db.Create(usage).Error
	} else if err != nil {
		return err
	}

	// Update existing usage
	return r.db.Model(&existingUsage).Updates(usage).Error
}

func (r *PlanRepository) IncrementMessageUsage(tenantID uuid.UUID, count int) error {
	return r.db.Model(&models.TenantUsage{}).
		Where("tenant_id = ?", tenantID).
		UpdateColumn("messages_used", gorm.Expr("messages_used + ?", count)).Error
}

func (r *PlanRepository) IncrementCreditUsage(tenantID uuid.UUID, count int) error {
	return r.db.Model(&models.TenantUsage{}).
		Where("tenant_id = ?", tenantID).
		UpdateColumn("credits_used", gorm.Expr("credits_used + ?", count)).Error
}

func (r *PlanRepository) UpdateResourceCount(tenantID uuid.UUID, resourceType string, count int) error {
	field := fmt.Sprintf("%s_count", resourceType)
	return r.db.Model(&models.TenantUsage{}).
		Where("tenant_id = ?", tenantID).
		UpdateColumn(field, count).Error
}

func (r *PlanRepository) ResetMonthlyUsage(tenantID uuid.UUID) error {
	now := time.Now()
	cycleStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	cycleEnd := cycleStart.AddDate(0, 1, 0)

	return r.db.Model(&models.TenantUsage{}).
		Where("tenant_id = ?", tenantID).
		Updates(map[string]interface{}{
			"messages_used":       0,
			"credits_used":        0,
			"billing_cycle_start": cycleStart,
			"billing_cycle_end":   cycleEnd,
		}).Error
}

// Usage checking methods
func (r *PlanRepository) CheckUsageLimit(tenantID uuid.UUID, resourceType string, requestedAmount int) (*models.UsageLimitResult, error) {
	usage, err := r.GetTenantUsage(tenantID)
	if err != nil {
		return nil, err
	}

	result := &models.UsageLimitResult{
		ResourceType:    resourceType,
		RequiresUpgrade: false,
	}

	switch resourceType {
	case "messages":
		result.TotalLimit = usage.Plan.MaxMessagesPerMonth
		result.CurrentUsage = usage.MessagesUsed
		result.RemainingQuota = result.TotalLimit - result.CurrentUsage
		result.Allowed = result.CurrentUsage+requestedAmount <= result.TotalLimit

	case "credits":
		result.TotalLimit = usage.Plan.MaxCreditsPerMonth
		result.CurrentUsage = usage.CreditsUsed
		result.RemainingQuota = result.TotalLimit - result.CurrentUsage
		result.Allowed = result.CurrentUsage+requestedAmount <= result.TotalLimit

	case "conversations":
		result.TotalLimit = usage.Plan.MaxConversations
		result.CurrentUsage = usage.ConversationsCount
		result.RemainingQuota = result.TotalLimit - result.CurrentUsage
		result.Allowed = result.CurrentUsage+requestedAmount <= result.TotalLimit

	case "products":
		result.TotalLimit = usage.Plan.MaxProducts
		result.CurrentUsage = usage.ProductsCount
		result.RemainingQuota = result.TotalLimit - result.CurrentUsage
		result.Allowed = result.CurrentUsage+requestedAmount <= result.TotalLimit

	case "channels":
		result.TotalLimit = usage.Plan.MaxChannels
		result.CurrentUsage = usage.ChannelsCount
		result.RemainingQuota = result.TotalLimit - result.CurrentUsage
		result.Allowed = result.CurrentUsage+requestedAmount <= result.TotalLimit

	default:
		return nil, fmt.Errorf("tipo de recurso desconhecido: %s", resourceType)
	}

	// Calcular percentual de uso atual
	usagePercentage := (result.CurrentUsage * 100) / result.TotalLimit

	if !result.Allowed {
		result.Message = fmt.Sprintf("Limite de %s excedido. Use %d/%d", resourceType, result.CurrentUsage, result.TotalLimit)
		result.RequiresUpgrade = true
	} else if usagePercentage >= 90 {
		result.Message = fmt.Sprintf("Aviso: Uso de %s em %d%% (%d/%d)", resourceType, usagePercentage, result.CurrentUsage, result.TotalLimit)
	}

	return result, nil
} // Invoice operations

// Usage alerts
func (r *PlanRepository) CreateUsageAlert(alert *models.UsageAlert) error {
	return r.db.Create(alert).Error
}

func (r *PlanRepository) GetPendingAlerts() ([]models.UsageAlert, error) {
	var alerts []models.UsageAlert
	err := r.db.Where("alert_sent = ?", false).Find(&alerts).Error
	return alerts, err
}

func (r *PlanRepository) MarkAlertAsSent(id uuid.UUID) error {
	now := time.Now()
	return r.db.Model(&models.UsageAlert{}).Where("id = ?", id).
		Updates(map[string]interface{}{
			"alert_sent": true,
			"sent_at":    &now,
		}).Error
}

// Plan change history
func (r *PlanRepository) CreatePlanHistory(history *models.TenantPlanHistory) error {
	return r.db.Create(history).Error
}

func (r *PlanRepository) GetPlanHistory(tenantID uuid.UUID) ([]models.TenantPlanHistory, error) {
	var history []models.TenantPlanHistory
	err := r.db.Preload("OldPlan").Preload("NewPlan").Preload("ChangedByUser").
		Where("tenant_id = ?", tenantID).
		Order("effective_date DESC").Find(&history).Error
	return history, err
}

// Tenant plan assignment
func (r *PlanRepository) AssignPlanToTenant(tenantID, planID uuid.UUID, changedByUserID uuid.UUID) error {
	tx := r.db.Begin()

	// Get current plan
	var tenant models.Tenant
	if err := tx.First(&tenant, "id = ?", tenantID).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Update tenant plan
	if err := tx.Model(&tenant).Update("plan_id", planID).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Create history record
	history := &models.TenantPlanHistory{
		TenantID:        tenantID,
		OldPlanID:       tenant.PlanID,
		NewPlanID:       planID,
		ChangedByUserID: changedByUserID,
		EffectiveDate:   time.Now(),
		ChangeReason:    "Plan assignment",
	}

	if err := tx.Create(history).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Initialize or update tenant usage
	now := time.Now()
	cycleStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	cycleEnd := cycleStart.AddDate(0, 1, 0)

	usage := &models.TenantUsage{
		TenantID:          tenantID,
		PlanID:            planID,
		BillingCycleStart: cycleStart,
		BillingCycleEnd:   cycleEnd,
	}

	if err := r.CreateOrUpdateTenantUsage(usage); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

// UpdateTenantPlan atualiza o plano de um tenant
func (r *PlanRepository) UpdateTenantPlan(tenantID, planID uuid.UUID) error {
	// Verificar se o plano existe
	var plan models.Plan
	if err := r.db.First(&plan, planID).Error; err != nil {
		return err
	}

	// Atualizar o tenant
	if err := r.db.Model(&models.Tenant{}).Where("id = ?", tenantID).Update("plan_id", planID).Error; err != nil {
		return err
	}

	// Criar ou atualizar o usage do tenant com o novo plano
	return r.createTenantUsageForPlan(tenantID, planID)
}

// IncrementUsage incrementa o uso de um recurso para um tenant
func (r *PlanRepository) IncrementUsage(tenantID uuid.UUID, resourceType string, amount int) error {
	usage, err := r.GetTenantUsage(tenantID)
	if err != nil {
		return err
	}

	updateMap := make(map[string]interface{})

	switch resourceType {
	case "messages":
		updateMap["messages_used"] = usage.MessagesUsed + amount
	case "credits":
		updateMap["credits_used"] = usage.CreditsUsed + amount
	case "conversations":
		updateMap["conversations_count"] = usage.ConversationsCount + amount
	case "products":
		updateMap["products_count"] = usage.ProductsCount + amount
	case "channels":
		updateMap["channels_count"] = usage.ChannelsCount + amount
	default:
		return fmt.Errorf("tipo de recurso desconhecido: %s", resourceType)
	}

	return r.db.Model(&models.TenantUsage{}).
		Where("tenant_id = ? AND billing_cycle_start <= ? AND billing_cycle_end > ?",
			tenantID, time.Now(), time.Now()).
		Updates(updateMap).Error
}

// ResetBillingCycle reseta o ciclo de billing de um tenant
func (r *PlanRepository) ResetBillingCycle(tenantID uuid.UUID) error {
	now := time.Now()
	cycleStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	cycleEnd := cycleStart.AddDate(0, 1, 0)

	// Resetar os contadores mensais
	resetMap := map[string]interface{}{
		"messages_used":       0,
		"credits_used":        0,
		"billing_cycle_start": cycleStart,
		"billing_cycle_end":   cycleEnd,
	}

	return r.db.Model(&models.TenantUsage{}).
		Where("tenant_id = ?", tenantID).
		Updates(resetMap).Error
}

// createTenantUsageForPlan cria ou atualiza o usage para um tenant com um plano específico
func (r *PlanRepository) createTenantUsageForPlan(tenantID, planID uuid.UUID) error {
	now := time.Now()
	cycleStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	cycleEnd := cycleStart.AddDate(0, 1, 0)

	usage := &models.TenantUsage{
		TenantID:          tenantID,
		PlanID:            planID,
		BillingCycleStart: cycleStart,
		BillingCycleEnd:   cycleEnd,
	}

	return r.CreateOrUpdateTenantUsage(usage)
}
