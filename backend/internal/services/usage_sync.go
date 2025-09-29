package services

import (
	"context"
	"iafarma/internal/repo"
	"iafarma/pkg/models"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// UsageSyncService handles automatic synchronization of tenant usage counters
type UsageSyncService struct {
	db            *gorm.DB
	planRepo      *repo.PlanRepository
	checkInterval time.Duration
	mutex         sync.RWMutex
	isRunning     bool
	stopChan      chan struct{}
}

// NewUsageSyncService creates a new usage sync service
func NewUsageSyncService(db *gorm.DB) *UsageSyncService {
	planRepo := repo.NewPlanRepository(db)

	return &UsageSyncService{
		db:            db,
		planRepo:      planRepo,
		checkInterval: 1 * time.Minute, // Sync every minute
		stopChan:      make(chan struct{}),
	}
}

// Start begins the usage sync process
func (uss *UsageSyncService) Start(ctx context.Context) {
	uss.mutex.Lock()
	if uss.isRunning {
		uss.mutex.Unlock()
		return
	}
	uss.isRunning = true
	uss.mutex.Unlock()

	log.Println("📊 Iniciando sincronização automática de contadores de uso...")

	go func() {
		ticker := time.NewTicker(uss.checkInterval)
		defer ticker.Stop()

		// Execute primeira sincronização imediatamente
		uss.syncAllTenantUsageCounters(ctx)

		for {
			select {
			case <-ticker.C:
				uss.syncAllTenantUsageCounters(ctx)
			case <-uss.stopChan:
				log.Println("📊 Parando sincronização de contadores de uso...")
				return
			case <-ctx.Done():
				log.Println("📊 Contexto cancelado, parando sincronização...")
				return
			}
		}
	}()
}

// Stop stops the usage sync process
func (uss *UsageSyncService) Stop() {
	uss.mutex.Lock()
	defer uss.mutex.Unlock()

	if !uss.isRunning {
		return
	}

	uss.isRunning = false
	close(uss.stopChan)
}

// syncAllTenantUsageCounters synchronizes usage counters for all tenants
func (uss *UsageSyncService) syncAllTenantUsageCounters(ctx context.Context) {
	// Get all tenants
	var tenants []models.Tenant
	if err := uss.db.Find(&tenants).Error; err != nil {
		log.Printf("❌ Erro ao buscar tenants para sincronização: %v", err)
		return
	}

	// log.Printf("📊 Sincronizando contadores para %d tenants...", len(tenants))

	syncedCount := 0
	for _, tenant := range tenants {
		select {
		case <-ctx.Done():
			log.Println("📊 Sincronização interrompida pelo contexto")
			return
		default:
			if err := uss.syncTenantUsageCounters(tenant.ID); err != nil {
				log.Printf("❌ Erro ao sincronizar contadores para tenant %s: %v", tenant.Name, err)
				continue
			}
			syncedCount++
		}
	}

	// log.Printf("✅ Sincronização concluída: %d/%d tenants processados", syncedCount, len(tenants))
}

// syncTenantUsageCounters synchronizes usage counters for a specific tenant
func (uss *UsageSyncService) syncTenantUsageCounters(tenantID uuid.UUID) error {
	// log.Printf("🔍 Sincronizando tenant: %s", tenantID)

	// Count products
	var productCount int64
	if err := uss.db.Model(&models.Product{}).Where("tenant_id = ?", tenantID).Count(&productCount).Error; err != nil {
		log.Printf("❌ Erro ao contar produtos: %v", err)
		return err
	}

	// Count conversations (using customers table)
	var conversationCount int64
	if err := uss.db.Model(&models.Customer{}).Where("tenant_id = ?", tenantID).Count(&conversationCount).Error; err != nil {
		log.Printf("❌ Erro ao contar conversas: %v", err)
		return err
	}

	// Count channels
	var channelCount int64
	if err := uss.db.Model(&models.Channel{}).Where("tenant_id = ?", tenantID).Count(&channelCount).Error; err != nil {
		log.Printf("❌ Erro ao contar canais: %v", err)
		return err
	}

	// Count messages sent this month
	var messageCount int64
	now := time.Now()
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	if err := uss.db.Model(&models.Message{}).
		Where("tenant_id = ? AND created_at >= ?", tenantID, startOfMonth).
		Count(&messageCount).Error; err != nil {
		log.Printf("❌ Erro ao contar mensagens: %v", err)
		return err
	}

	// log.Printf("📊 Contadores - Produtos: %d, Conversas: %d, Canais: %d, Mensagens: %d",
	// 	productCount, conversationCount, channelCount, messageCount)

	// Check if tenant usage record exists
	var existingUsage models.TenantUsage
	err := uss.db.Where("tenant_id = ?", tenantID).First(&existingUsage).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			log.Printf("📝 Criando novo registro de usage para tenant %s", tenantID)

			// Record doesn't exist, create it
			// First get the tenant's plan
			var tenant models.Tenant
			if err := uss.db.First(&tenant, "id = ?", tenantID).Error; err != nil {
				log.Printf("❌ Erro ao buscar tenant: %v", err)
				return err
			}

			// Get plan ID
			var planID uuid.UUID
			if tenant.PlanID != nil {
				planID = *tenant.PlanID
				log.Printf("✅ Usando plano do tenant: %s", planID)
			} else {
				// Get default plan
				var defaultPlan models.Plan
				if err := uss.db.Where("is_default = true OR name = 'Gratuito'").First(&defaultPlan).Error; err != nil {
					log.Printf("❌ Erro ao buscar plano padrão: %v", err)
					return err
				}
				planID = defaultPlan.ID
				log.Printf("✅ Usando plano padrão: %s", planID)
			}

			// Create new usage record
			newUsage := models.TenantUsage{
				TenantID:           tenantID,
				PlanID:             planID,
				MessagesUsed:       int(messageCount),
				CreditsUsed:        0, // Will be updated by other services
				BillingCycleStart:  startOfMonth,
				BillingCycleEnd:    startOfMonth.AddDate(0, 1, 0).Add(-time.Second),
				ConversationsCount: int(conversationCount),
				ProductsCount:      int(productCount),
				ChannelsCount:      int(channelCount),
			}

			if err := uss.db.Create(&newUsage).Error; err != nil {
				log.Printf("❌ Erro ao criar registro de usage: %v", err)
				return err
			}
			log.Printf("✅ Registro de usage criado com sucesso")
		} else {
			log.Printf("❌ Erro ao verificar registro existente: %v", err)
			return err
		}
	} else {
		// log.Printf("📝 Atualizando registro existente de usage para tenant %s", tenantID)

		// Record exists, update only the countable fields
		updateMap := map[string]interface{}{
			"products_count":      int(productCount),
			"conversations_count": int(conversationCount),
			"channels_count":      int(channelCount),
			"messages_used":       int(messageCount),
		}

		if err := uss.db.Model(&models.TenantUsage{}).
			Where("tenant_id = ?", tenantID).
			Updates(updateMap).Error; err != nil {
			log.Printf("❌ Erro ao atualizar registro de usage: %v", err)
			return err
		}
		// log.Printf("✅ Registro de usage atualizado com sucesso")
	}

	return nil
}

// GetSyncStatus returns the current status of usage sync service
func (uss *UsageSyncService) GetSyncStatus() map[string]interface{} {
	uss.mutex.RLock()
	defer uss.mutex.RUnlock()

	return map[string]interface{}{
		"is_running":     uss.isRunning,
		"check_interval": uss.checkInterval.String(),
		"last_sync":      time.Now().Format("2006-01-02 15:04:05"),
	}
}
