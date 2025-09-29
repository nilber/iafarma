package ai

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"iafarma/pkg/models"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/sashabaranov/go-openai"
	"gorm.io/gorm"
)

// ConversationMemory stores temporary data for a conversation (in-memory)
type ConversationMemory struct {
	TenantID            uuid.UUID
	CustomerPhone       string
	ProductList         []ProductReference
	ConversationHistory []openai.ChatCompletionMessage
	LastUpdateTime      time.Time
	SequentialNumber    int
	TempData            map[string]interface{} // For storing temporary data like order lists
}

// ProductReference stores product info with sequential number
type ProductReference struct {
	SequentialID int
	ProductID    uuid.UUID
	Name         string
	Price        string
	SalePrice    string
	Description  string
}

// MemoryManager manages conversation memories with PostgreSQL persistence
type MemoryManager struct {
	memories      map[string]*ConversationMemory
	mutex         sync.RWMutex
	db            *gorm.DB
	cleanupTicker *time.Ticker
	stopCleanup   chan struct{}
	isRunning     bool
}

// NewMemoryManager creates a new memory manager with database persistence
func NewMemoryManager() *MemoryManager {
	return &MemoryManager{
		memories:    make(map[string]*ConversationMemory),
		stopCleanup: make(chan struct{}),
		isRunning:   false,
	}
}

// NewMemoryManagerWithDB creates a new memory manager with database persistence
func NewMemoryManagerWithDB(db *gorm.DB) *MemoryManager {
	mm := &MemoryManager{
		memories:    make(map[string]*ConversationMemory),
		db:          db,
		stopCleanup: make(chan struct{}),
		isRunning:   false,
	}

	// Load existing memories from database on startup
	if db != nil {
		go mm.loadMemoriesFromDB()
	}

	// Start automatic cleanup ticker
	mm.StartCleanupScheduler()

	return mm
}

// getMemoryKey creates a unique key for tenant+customer
func (m *MemoryManager) getMemoryKey(tenantID uuid.UUID, customerPhone string) string {
	return fmt.Sprintf("%s:%s", tenantID.String(), customerPhone)
}

// GetOrCreateMemory gets existing memory or creates new one with default timeout (1 hour)
func (m *MemoryManager) GetOrCreateMemory(tenantID uuid.UUID, customerPhone string) *ConversationMemory {
	return m.GetOrCreateMemoryWithTimeout(tenantID, customerPhone, 1*time.Hour)
}

// GetOrCreateMemoryWithTimeout gets existing memory or creates new one with custom timeout
func (m *MemoryManager) GetOrCreateMemoryWithTimeout(tenantID uuid.UUID, customerPhone string, timeout time.Duration) *ConversationMemory {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	key := m.getMemoryKey(tenantID, customerPhone)

	memory, exists := m.memories[key]
	if !exists || time.Since(memory.LastUpdateTime) > timeout {
		log.Info().
			Str("tenant_id", tenantID.String()).
			Str("customer_phone", customerPhone).
			Str("memory_key", key).
			Bool("memory_exists", exists).
			Dur("timeout", timeout).
			Msg("🧠 Creating new memory (not found or expired)")

		// Create new memory or refresh expired one
		memory = &ConversationMemory{
			TenantID:            tenantID,
			CustomerPhone:       customerPhone,
			ProductList:         []ProductReference{},
			ConversationHistory: []openai.ChatCompletionMessage{},
			LastUpdateTime:      time.Now(),
			SequentialNumber:    0,
			TempData:            make(map[string]interface{}),
		}
		m.memories[key] = memory
	} else {
		log.Info().
			Str("tenant_id", tenantID.String()).
			Str("customer_phone", customerPhone).
			Str("memory_key", key).
			Int("history_count", len(memory.ConversationHistory)).
			Time("last_update", memory.LastUpdateTime).
			Msg("🧠 Found existing memory")

		// Initialize TempData if it's nil (for existing memories)
		if memory.TempData == nil {
			memory.TempData = make(map[string]interface{})
		}
	}

	return memory
}

// GetConversationHistory gets the conversation history for a customer
func (m *MemoryManager) GetConversationHistory(tenantID uuid.UUID, customerPhone string) []openai.ChatCompletionMessage {
	memory := m.GetOrCreateMemory(tenantID, customerPhone)

	m.mutex.RLock()
	defer m.mutex.RUnlock()

	log.Info().
		Str("tenant_id", tenantID.String()).
		Str("customer_phone", customerPhone).
		Int("history_count", len(memory.ConversationHistory)).
		Time("last_update", memory.LastUpdateTime).
		Msg("📚 Getting conversation history")

	return memory.ConversationHistory
}

// AddToConversationHistory adds a message to the conversation history
func (m *MemoryManager) AddToConversationHistory(tenantID uuid.UUID, customerPhone string, message openai.ChatCompletionMessage) {
	memory := m.GetOrCreateMemory(tenantID, customerPhone)

	m.mutex.Lock()
	defer m.mutex.Unlock()

	log.Info().
		Str("tenant_id", tenantID.String()).
		Str("customer_phone", customerPhone).
		Str("message_role", string(message.Role)).
		Str("message_content", message.Content[:min(100, len(message.Content))]).
		Int("history_before", len(memory.ConversationHistory)).
		Msg("🔄 Adding message to conversation history")

	// Add message to history
	memory.ConversationHistory = append(memory.ConversationHistory, message)

	// Keep only last 10 messages (5 pairs) to avoid token limits
	if len(memory.ConversationHistory) > 10 {
		memory.ConversationHistory = memory.ConversationHistory[len(memory.ConversationHistory)-10:]
	}

	log.Info().
		Int("history_after", len(memory.ConversationHistory)).
		Time("last_update", memory.LastUpdateTime).
		Msg("🔄 Message added to conversation history")

	memory.LastUpdateTime = time.Now()

	// Save to database asynchronously (but don't save too frequently)
	// Only save every 3 messages to reduce DB load
	if len(memory.ConversationHistory)%3 == 0 {
		// Create a deep copy of the memory to avoid race conditions
		memoryCopy := &ConversationMemory{
			TenantID:            memory.TenantID,
			CustomerPhone:       memory.CustomerPhone,
			LastUpdateTime:      memory.LastUpdateTime,
			SequentialNumber:    memory.SequentialNumber,
			ConversationHistory: make([]openai.ChatCompletionMessage, len(memory.ConversationHistory)),
			ProductList:         make([]ProductReference, len(memory.ProductList)),
		}

		// Copy conversation history
		copy(memoryCopy.ConversationHistory, memory.ConversationHistory)

		// Copy product list
		copy(memoryCopy.ProductList, memory.ProductList)

		go m.saveMemoryToDB(memoryCopy)
	}
}

// StoreProductList stores a list of products with sequential numbering
func (m *MemoryManager) StoreProductList(tenantID uuid.UUID, customerPhone string, products []models.Product) []ProductReference {
	memory := m.GetOrCreateMemory(tenantID, customerPhone)

	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Clear previous product list and reset sequential number
	memory.ProductList = []ProductReference{}
	memory.SequentialNumber = 0

	// Add products with sequential IDs
	refs := make([]ProductReference, len(products))
	for i, product := range products {
		memory.SequentialNumber++
		ref := ProductReference{
			SequentialID: memory.SequentialNumber,
			ProductID:    product.ID,
			Name:         product.Name,
			Price:        product.Price,
			SalePrice:    product.SalePrice,
			Description:  product.Description,
		}
		memory.ProductList = append(memory.ProductList, ref)
		refs[i] = ref
	}

	memory.LastUpdateTime = time.Now()

	// Save to database asynchronously
	go m.saveMemoryToDB(memory)

	return refs
}

// AppendProductList adds more products to existing list with continued sequential numbering
func (m *MemoryManager) AppendProductList(tenantID uuid.UUID, customerPhone string, products []models.Product) []ProductReference {
	memory := m.GetOrCreateMemory(tenantID, customerPhone)

	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Add products with sequential IDs continuing from last number
	newRefs := make([]ProductReference, len(products))
	for i, product := range products {
		memory.SequentialNumber++
		ref := ProductReference{
			SequentialID: memory.SequentialNumber,
			ProductID:    product.ID,
			Name:         product.Name,
			Price:        product.Price,
			SalePrice:    product.SalePrice,
			Description:  product.Description,
		}
		memory.ProductList = append(memory.ProductList, ref)
		newRefs[i] = ref
	}

	memory.LastUpdateTime = time.Now()

	// Save to database asynchronously
	go m.saveMemoryToDB(memory)

	return newRefs
}

// loadMemoriesFromDB loads recent memories from database on startup
func (m *MemoryManager) loadMemoriesFromDB() {
	if m.db == nil {
		return
	}

	log.Info().Msg("🧠 Loading conversation memories from database...")

	var dbMemories []models.ConversationMemory
	// Load memories updated in the last 7 days (for scheduling workflows)
	err := m.db.Where("updated_at > ?", time.Now().Add(-7*24*time.Hour)).Find(&dbMemories).Error
	if err != nil {
		log.Error().Err(err).Msg("Failed to load memories from database")
		return
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	loaded := 0
	for _, dbMem := range dbMemories {
		key := m.getMemoryKey(dbMem.TenantID, dbMem.CustomerPhone)

		// Convert database model to in-memory model
		productList := make([]ProductReference, len(dbMem.ProductList))
		for i, dbProd := range dbMem.ProductList {
			productID, _ := uuid.Parse(dbProd.ProductID)
			productList[i] = ProductReference{
				SequentialID: dbProd.SequentialID,
				ProductID:    productID,
				Name:         dbProd.Name,
				Price:        dbProd.Price,
				SalePrice:    dbProd.SalePrice,
				Description:  dbProd.Description,
			}
		}

		// Convert conversation history
		history := make([]openai.ChatCompletionMessage, len(dbMem.ConversationHistory))
		for i, dbHist := range dbMem.ConversationHistory {
			history[i] = openai.ChatCompletionMessage{
				Role:    dbHist.Role,
				Content: dbHist.Content,
			}
		}

		memory := &ConversationMemory{
			TenantID:            dbMem.TenantID,
			CustomerPhone:       dbMem.CustomerPhone,
			ProductList:         productList,
			ConversationHistory: history,
			LastUpdateTime:      dbMem.UpdatedAt,
			SequentialNumber:    dbMem.SequentialNumber,
		}

		m.memories[key] = memory
		loaded++
	}

	log.Info().Int("loaded_count", loaded).Msg("🧠 Conversation memories loaded from database")
}

// saveMemoryToDB saves a memory to the database
func (m *MemoryManager) saveMemoryToDB(memory *ConversationMemory) {
	if m.db == nil {
		return
	}

	// Validate memory data
	if memory == nil {
		log.Error().Msg("Cannot save nil memory to database")
		return
	}

	// Convert in-memory model to database model
	productList := make(models.ProductReferenceList, len(memory.ProductList))
	for i, prod := range memory.ProductList {
		if i >= len(productList) {
			log.Error().Int("index", i).Int("length", len(productList)).Msg("Index out of range while saving products")
			break
		}
		productList[i] = models.ProductReference{
			SequentialID: prod.SequentialID,
			ProductID:    prod.ProductID.String(),
			Name:         prod.Name,
			Price:        prod.Price,
			SalePrice:    prod.SalePrice,
			Description:  prod.Description,
		}
	}

	// Convert conversation history
	history := make(models.ConversationHistoryList, len(memory.ConversationHistory))
	for i, hist := range memory.ConversationHistory {
		if i >= len(history) {
			log.Error().Int("index", i).Int("length", len(history)).Msg("Index out of range while saving conversation history")
			break
		}
		history[i] = models.ConversationHistoryItem{
			Role:    hist.Role,
			Content: hist.Content,
		}
	}

	dbMemory := models.ConversationMemory{
		TenantID:            memory.TenantID,
		CustomerPhone:       memory.CustomerPhone,
		ProductList:         productList,
		ConversationHistory: history,
		SequentialNumber:    memory.SequentialNumber,
	}

	// Use UPSERT - primeiro tenta atualizar, se não existe, cria
	var existingMemory models.ConversationMemory
	err := m.db.Where("tenant_id = ? AND customer_phone = ?",
		memory.TenantID, memory.CustomerPhone).First(&existingMemory).Error

	if err == gorm.ErrRecordNotFound {
		// Criar novo
		err = m.db.Create(&dbMemory).Error
	} else if err == nil {
		// Atualizar existente
		err = m.db.Model(&existingMemory).Updates(map[string]interface{}{
			"product_list":         dbMemory.ProductList,
			"conversation_history": dbMemory.ConversationHistory,
			"sequential_number":    dbMemory.SequentialNumber,
			"updated_at":           time.Now(),
		}).Error
	}

	if err != nil {
		log.Error().Err(err).
			Str("tenant_id", memory.TenantID.String()).
			Str("customer_phone", memory.CustomerPhone).
			Msg("Failed to save memory to database")
	}
}

// GetCurrentProductList returns the current product list for a user
func (m *MemoryManager) GetCurrentProductList(tenantID uuid.UUID, customerPhone string) []ProductReference {
	memory := m.GetOrCreateMemory(tenantID, customerPhone)

	m.mutex.RLock()
	defer m.mutex.RUnlock()

	return memory.ProductList
}

// GetProductBySequentialID gets product by sequential number
func (m *MemoryManager) GetProductBySequentialID(tenantID uuid.UUID, customerPhone string, sequentialID int) *ProductReference {
	memory := m.GetOrCreateMemory(tenantID, customerPhone)

	m.mutex.RLock()
	defer m.mutex.RUnlock()

	for _, product := range memory.ProductList {
		if product.SequentialID == sequentialID {
			return &product
		}
	}

	return nil
}

// GetProductByName gets product by name (case insensitive)
func (m *MemoryManager) GetProductByName(tenantID uuid.UUID, customerPhone string, name string) *ProductReference {
	memory := m.GetOrCreateMemory(tenantID, customerPhone)

	m.mutex.RLock()
	defer m.mutex.RUnlock()

	nameLower := strings.ToLower(name)
	for _, product := range memory.ProductList {
		if strings.Contains(strings.ToLower(product.Name), nameLower) {
			return &product
		}
	}

	return nil
}

// ClearMemory clears memory for a specific customer (used after order completion)
func (m *MemoryManager) ClearMemory(tenantID uuid.UUID, customerPhone string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	key := m.getMemoryKey(tenantID, customerPhone)
	delete(m.memories, key)

	log.Info().
		Str("tenant_id", tenantID.String()).
		Str("customer_phone", customerPhone).
		Str("memory_key", key).
		Msg("🧹 Memory cleared for customer")
}

// CleanupExpiredMemories removes old memories (call periodically)
// Uses maximum timeout of 7 days to ensure scheduling memories are preserved
func (m *MemoryManager) CleanupExpiredMemories() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Use 7 days as maximum cutoff to handle scheduling workflows
	cutoff := time.Now().Add(-7 * 24 * time.Hour)
	cleaned := 0
	for key, memory := range m.memories {
		if memory.LastUpdateTime.Before(cutoff) {
			delete(m.memories, key)
			cleaned++
		}
	}

	if cleaned > 0 {
		log.Info().Int("cleaned_count", cleaned).Msg("🧹 Cleaned up expired memories (7 days cutoff)")
	}
}

// StartCleanupScheduler starts the automatic cleanup scheduler
func (m *MemoryManager) StartCleanupScheduler() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.isRunning {
		return
	}

	m.isRunning = true
	m.cleanupTicker = time.NewTicker(60 * time.Minute) // Check every 10 minutes

	go func() {
		log.Info().Msg("🧠 Started automatic memory cleanup scheduler (every 10 minutes)")

		for {
			select {
			case <-m.cleanupTicker.C:
				m.CleanupExpiredMemories()
			case <-m.stopCleanup:
				log.Info().Msg("🧠 Memory cleanup scheduler stopped")
				return
			}
		}
	}()
}

// StopCleanupScheduler stops the automatic cleanup scheduler
func (m *MemoryManager) StopCleanupScheduler() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if !m.isRunning {
		return
	}

	m.isRunning = false
	if m.cleanupTicker != nil {
		m.cleanupTicker.Stop()
	}

	close(m.stopCleanup)
}

// StoreTempData stores temporary data in memory
func (m *MemoryManager) StoreTempData(tenantID uuid.UUID, customerPhone string, data map[string]interface{}) {
	memory := m.GetOrCreateMemory(tenantID, customerPhone)

	m.mutex.Lock()
	defer m.mutex.Unlock()

	for key, value := range data {
		memory.TempData[key] = value
	}
	memory.LastUpdateTime = time.Now()

	log.Info().
		Str("tenant_id", tenantID.String()).
		Str("customer_phone", customerPhone).
		Int("temp_data_keys", len(memory.TempData)).
		Msg("📦 Stored temporary data in memory")
}

// GetTempData retrieves temporary data from memory
func (m *MemoryManager) GetTempData(tenantID uuid.UUID, customerPhone string, key string) (interface{}, bool) {
	memory := m.GetOrCreateMemory(tenantID, customerPhone)

	m.mutex.RLock()
	defer m.mutex.RUnlock()

	value, exists := memory.TempData[key]
	return value, exists
}
