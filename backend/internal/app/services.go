package app

import (
	"fmt"
	"iafarma/internal/auth"
	"iafarma/internal/repo"
	"iafarma/internal/services"
	"os"

	"gorm.io/gorm"
)

// Services holds all application services
type Services struct {
	DB                           *gorm.DB
	AuthService                  *auth.Service
	UserRepo                     *repo.UserRepository
	TenantRepo                   *repo.TenantRepository
	ProductRepo                  *repo.ProductRepository
	CustomerRepo                 *repo.CustomerRepository
	CategoryRepo                 *repo.CategoryRepository
	AddressRepo                  *repo.AddressRepository
	OrderRepo                    *repo.OrderRepository
	ChannelRepo                  *repo.ChannelRepository
	MessageRepo                  *repo.MessageRepository
	MessageTemplateRepo          *repo.MessageTemplateRepository
	PlanRepo                     *repo.PlanRepository
	AlertService                 *services.AlertService
	DeliveryService              *services.DeliveryService
	EmailService                 *services.EmailService
	ChannelMonitorService        *services.ChannelMonitorService
	ChannelReconnectionService   *services.ChannelReconnectionService
	EmbeddingService             *services.EmbeddingService
	ImportJobService             *services.ImportJobService
	StorageService               *services.StorageService
	PlanLimitService             *services.PlanLimitService
	CategoryService              *services.CategoryService
	UsageSyncService             *services.UsageSyncService
	InfrastructureMonitorService *services.InfrastructureMonitorService
}

// NewServices creates a new services container
func NewServices(db *gorm.DB) *Services {
	// Initialize repositories
	userRepo := repo.NewUserRepository(db)
	tenantRepo := repo.NewTenantRepository(db)
	productRepo := repo.NewProductRepository(db)
	customerRepo := repo.NewCustomerRepository(db)
	categoryRepo := repo.NewCategoryRepository(db)
	addressRepo := repo.NewAddressRepository(db)
	orderRepo := repo.NewOrderRepository(db)
	channelRepo := repo.NewChannelRepository(db)
	messageRepo := repo.NewMessageRepository(db)
	messageTemplateRepo := repo.NewMessageTemplateRepository(db)
	planRepo := repo.NewPlanRepository(db)

	// Initialize services
	authService := auth.NewService(userRepo)
	alertService := services.NewAlertService(db)

	// Get Google Maps API key from environment
	googleMapsAPIKey := os.Getenv("GOOGLE_MAPS_API_KEY")
	deliveryService := services.NewDeliveryService(db, googleMapsAPIKey)

	// Initialize email service
	emailService, err := services.NewEmailService(db)
	if err != nil {
		// Log warning but continue - email service is optional
		// Note: In production, you might want to fail here
	}

	// Initialize channel monitor service
	channelMonitorService := services.NewChannelMonitorService(db, emailService)

	// Initialize channel reconnection service
	channelReconnectionService := services.NewChannelReconnectionService(db, emailService, channelMonitorService)

	// Initialize embedding service for RAG first
	var embeddingService *services.EmbeddingService
	openaiAPIKey := os.Getenv("OPENAI_API_KEY")
	qdrantURL := os.Getenv("QDRANT_URL")
	qdrantPassword := os.Getenv("QDRANT_PASSWORD")
	if qdrantURL == "" {
		qdrantURL = "localhost:6334" // default gRPC port
	}

	if openaiAPIKey != "" {
		embeddingService, err = services.NewEmbeddingService(openaiAPIKey, qdrantURL, qdrantPassword)
		if err != nil {
			// Log warning but continue - embedding service is optional
			// Note: In production, you might want to log this properly
			fmt.Printf("Warning: Failed to initialize embedding service: %v\n", err)
		} else {
			if qdrantPassword != "" {
				fmt.Printf("✅ Embedding service initialized successfully with authentication (Qdrant: %s)\n", qdrantURL)
			} else {
				fmt.Printf("✅ Embedding service initialized successfully (Qdrant: %s)\n", qdrantURL)
			}

			// Check Qdrant availability on startup
			if err := embeddingService.CheckQdrantHealth(); err != nil {
				fmt.Printf("\n🛑  RAG Service Status: Unavailable - Qdrant connection failed (%v)\n\n", err)
			} else {
				fmt.Printf("✅ RAG Service Status: Available - Qdrant connection healthy\n")
			}

			// 🔄 Check if product sync is enabled on startup
			syncEnabled := os.Getenv("RAG_SYNC_PRODUCTS_ON_STARTUP")
			if syncEnabled == "true" {
				fmt.Printf("🔄 RAG_SYNC_PRODUCTS_ON_STARTUP enabled - syncing products...\n")
				go func() {
					if err := embeddingService.SyncAllProductsFromDB(db); err != nil {
						fmt.Printf("❌ Failed to sync products to RAG: %v\n", err)
					} else {
						fmt.Printf("✅ Products successfully synced to RAG on startup\n")
					}
				}()
			} else {
				fmt.Printf("ℹ️ RAG product sync disabled (set RAG_SYNC_PRODUCTS_ON_STARTUP=true to enable)\n")
			}
		}
	} else {
		fmt.Println("Warning: OPENAI_API_KEY not set, embedding service disabled")
	}

	// Initialize import job service with embedding service
	importJobService := services.NewImportJobService(db, productRepo, categoryRepo, embeddingService)

	// Initialize storage service
	storageService, err := services.NewStorageService()
	if err != nil {
		// Log warning but continue - storage service is optional for basic functionality
		fmt.Printf("Warning: Failed to initialize storage service: %v\n", err)
	}

	// Initialize plan limit service
	planLimitService := services.NewPlanLimitService(db)

	// Initialize category service
	categoryService := services.NewCategoryService(categoryRepo)

	// Initialize usage sync service
	usageSyncService := services.NewUsageSyncService(db)

	// Initialize Infrastructure Monitor service
	infrastructureMonitorService, err := services.NewInfrastructureMonitorService(db, embeddingService)
	if err != nil {
		fmt.Printf("Warning: Failed to initialize infrastructure monitor service: %v\n", err)
	}

	return &Services{
		DB:                           db,
		AuthService:                  authService,
		UserRepo:                     userRepo,
		TenantRepo:                   tenantRepo,
		ProductRepo:                  productRepo,
		CustomerRepo:                 customerRepo,
		CategoryRepo:                 categoryRepo,
		AddressRepo:                  addressRepo,
		OrderRepo:                    orderRepo,
		ChannelRepo:                  channelRepo,
		MessageRepo:                  messageRepo,
		MessageTemplateRepo:          messageTemplateRepo,
		PlanRepo:                     planRepo,
		AlertService:                 alertService,
		DeliveryService:              deliveryService,
		EmailService:                 emailService,
		ChannelMonitorService:        channelMonitorService,
		ChannelReconnectionService:   channelReconnectionService,
		EmbeddingService:             embeddingService,
		ImportJobService:             importJobService,
		StorageService:               storageService,
		PlanLimitService:             planLimitService,
		CategoryService:              categoryService,
		UsageSyncService:             usageSyncService,
		InfrastructureMonitorService: infrastructureMonitorService,
	}
}
