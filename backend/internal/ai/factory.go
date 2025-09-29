package ai

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"iafarma/internal/repo"
	"iafarma/internal/zapplus"
	"iafarma/pkg/models"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/sashabaranov/go-openai"
	"gorm.io/gorm"
)

// WebSocketBroadcaster interface for WebSocket broadcasting
type WebSocketBroadcaster interface {
	BroadcastToTenant(tenantID string, messageType string, data interface{})
}

// categoryServiceImpl implements CategoryServiceInterface for AI service
type categoryServiceImpl struct {
	repo *repo.CategoryRepository
}

func (c *categoryServiceImpl) ListCategories(tenantID uuid.UUID) ([]models.Category, error) {
	return c.repo.List(tenantID)
}

func (c *categoryServiceImpl) GetCategoryByID(tenantID uuid.UUID, categoryID uuid.UUID) (*models.Category, error) {
	return c.repo.GetByID(tenantID, categoryID)
}

// Global memory manager singleton to persist across service instances
var globalMemoryManager *MemoryManager
var memoryManagerOnce sync.Once

// GetGlobalMemoryManager returns the singleton memory manager
func GetGlobalMemoryManager() *MemoryManager {
	memoryManagerOnce.Do(func() {
		globalMemoryManager = NewMemoryManager()
		log.Info().Msg("🧠 Created global memory manager singleton")
	})
	return globalMemoryManager
}

// GetGlobalMemoryManagerWithDB returns the singleton memory manager with database persistence
func GetGlobalMemoryManagerWithDB(db *gorm.DB) *MemoryManager {
	memoryManagerOnce.Do(func() {
		globalMemoryManager = NewMemoryManagerWithDB(db)
		log.Info().Msg("🧠 Created global memory manager singleton with database persistence")
	})
	return globalMemoryManager
}

// AIServiceFactory creates and configures an AI service with all dependencies
func NewAIServiceFactory(db *gorm.DB, openaiAPIKey string) *AIService {
	return NewAIServiceFactoryWithDelivery(db, openaiAPIKey, nil)
}

// NewAIServiceFactoryWithDelivery creates and configures an AI service with delivery service
func NewAIServiceFactoryWithDelivery(db *gorm.DB, openaiAPIKey string, deliveryService DeliveryServiceInterface) *AIService {
	return NewAIServiceFactoryWithDeliveryAndWebSocket(db, openaiAPIKey, deliveryService, nil)
}

// NewAIServiceFactoryWithDeliveryAndWebSocket creates and configures an AI service with delivery service and WebSocket
func NewAIServiceFactoryWithDeliveryAndWebSocket(db *gorm.DB, openaiAPIKey string, deliveryService DeliveryServiceInterface, wsHandler WebSocketBroadcaster) *AIService {
	return NewAIServiceFactoryComplete(db, openaiAPIKey, deliveryService, wsHandler, nil)
}

// NewAIServiceFactoryComplete creates and configures an AI service with all dependencies
func NewAIServiceFactoryComplete(db *gorm.DB, openaiAPIKey string, deliveryService DeliveryServiceInterface, wsHandler WebSocketBroadcaster, embeddingService EmbeddingServiceInterface) *AIService {
	// Create custom HTTP client with TLS configuration for macOS compatibility
	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: false, // Keep security but use system certs
			},
		},
	}

	// Create OpenAI client with custom HTTP client
	config := openai.DefaultConfig(openaiAPIKey)
	config.HTTPClient = httpClient
	client := openai.NewClientWithConfig(config)

	// Create service implementations
	productService := NewProductService(db)
	cartService := NewCartService(db)
	orderService := NewOrderService(db)
	customerService := NewCustomerService(db)
	addressService := NewAddressService(db)
	settingsService := NewTenantSettingsService(db)
	errorHandler := NewErrorHandler(db)

	// Create category service
	categoryRepo := repo.NewCategoryRepository(db)
	categoryService := &categoryServiceImpl{repo: categoryRepo}

	// Create alert service wrapper to avoid circular dependency
	alertService := NewAlertServiceWrapper(func(tenantID uuid.UUID, order *models.Order, customerPhone string) error {
		// Simple inline implementation to send alerts
		return sendOrderAlert(db, tenantID, order, customerPhone)
	})

	// Set the human support alert function
	alertService.SetSendHumanSupportAlertFunc(func(tenantID uuid.UUID, customerID uuid.UUID, customerPhone string, reason string) error {
		return sendHumanSupportAlertWithWebSocket(db, wsHandler, tenantID, customerID, customerPhone, reason)
	})

	// If no delivery service provided, create a default one
	if deliveryService == nil {
		// Create a default implementation that returns "not configured"
		deliveryService = &defaultDeliveryService{}
	}

	// Initialize S3 client for storage
	var s3Client *s3.S3
	var s3Bucket, s3BaseURL string

	endpoint := os.Getenv("S3_ENDPOINT")
	accessKey := os.Getenv("S3_ACCESS_KEY")
	secretKey := os.Getenv("S3_SECRET_KEY")
	bucket := os.Getenv("S3_BUCKET")

	if accessKey != "" && secretKey != "" && bucket != "" {
		// Create AWS session with proper endpoint configuration
		config := &aws.Config{
			Region: aws.String("us-east-1"),
			Credentials: credentials.NewStaticCredentials(
				accessKey,
				secretKey,
				"",
			),
		}

		// Only set custom endpoint if provided
		if endpoint != "" {
			config.Endpoint = aws.String(endpoint)
			config.S3ForcePathStyle = aws.Bool(true)
		}

		sess, err := session.NewSession(config)
		if err == nil {
			s3Client = s3.New(sess)
			s3Bucket = bucket
			// Use AWS S3 standard URL format instead of custom domain
			s3BaseURL = fmt.Sprintf("https://s3.us-east-1.amazonaws.com/%s", bucket)
			log.Info().Str("bucket", bucket).Msg("S3 storage initialized successfully in factory")
		} else {
			log.Error().Err(err).Msg("Failed to initialize S3 storage in factory")
		}
	} else {
		log.Warn().Msg("S3 configuration missing in factory, storage features disabled")
	}

	// Create and return AI service
	aiService := &AIService{
		client:           client,
		productService:   productService,
		cartService:      cartService,
		orderService:     orderService,
		customerService:  customerService,
		messageService:   NewMessageService(db),
		addressService:   addressService,
		settingsService:  settingsService,
		municipioService: nil, // Será implementado posteriormente
		categoryService:  categoryService,
		memoryManager:    GetGlobalMemoryManagerWithDB(db), // Use singleton with DB persistence
		errorHandler:     errorHandler,
		alertService:     alertService,
		deliveryService:  deliveryService,
		embeddingService: embeddingService,
		s3Client:         s3Client,
		s3Bucket:         s3Bucket,
		s3BaseURL:        s3BaseURL,
	}

	return aiService
}

// sendOrderAlert is an inline implementation to avoid circular dependency
func sendOrderAlert(db *gorm.DB, tenantID uuid.UUID, order *models.Order, customerPhone string) error {
	notificationService := zapplus.NewNotificationService(db)
	return notificationService.SendOrderAlert(tenantID, order, customerPhone)
}

// sendGroupMessage sends a message to the WhatsApp group
func sendGroupMessage(groupID, message, session string) error {
	client := zapplus.GetClient()
	return client.SendGroupMessage(session, groupID, message)
}

// defaultDeliveryService is a simple implementation for when no delivery service is provided
type defaultDeliveryService struct{}

func (d *defaultDeliveryService) ValidateDeliveryAddress(tenantID uuid.UUID, street, number, neighborhood, city, state string) (*DeliveryValidationResult, error) {
	return &DeliveryValidationResult{
		CanDeliver: false,
		Reason:     "no_store_location",
	}, nil
}

func (d *defaultDeliveryService) GetStoreLocation(tenantID uuid.UUID) (*StoreLocationInfo, error) {
	return &StoreLocationInfo{
		Address:     "",
		City:        "",
		State:       "",
		Coordinates: false,
		Latitude:    0,
		Longitude:   0,
		RadiusKm:    0,
	}, nil
}

// sendHumanSupportAlertWithWebSocket sends alerts when customer requests human support (with WebSocket support)
func sendHumanSupportAlertWithWebSocket(db *gorm.DB, wsHandler WebSocketBroadcaster, tenantID uuid.UUID, customerID uuid.UUID, customerPhone string, reason string) error {
	notificationService := zapplus.NewNotificationService(db)

	// Send ZapPlus notification
	err := notificationService.SendHumanSupportAlert(tenantID, customerID, customerPhone, reason)
	if err != nil {
		log.Error().Err(err).Msg("Failed to send human support alert via ZapPlus")
	}

	// Send WebSocket notification if handler available
	if wsHandler != nil {
		var customer models.Customer
		if err := db.Where("id = ? AND tenant_id = ?", customerID, tenantID).First(&customer).Error; err == nil {
			alertData := map[string]interface{}{
				"customer_id":    customerID.String(),
				"customer_name":  customer.Name,
				"customer_phone": customerPhone,
				"reason":         reason,
				"timestamp":      time.Now().Format("02/01/2006 15:04"),
			}
			wsHandler.BroadcastToTenant(tenantID.String(), "human_support_alert", alertData)
		}
	}

	return err
}

// MockLocationService implements LocationServiceInterface for scheduling
type MockLocationService struct {
	db *gorm.DB
}

// NewMockLocationService creates a new mock location service
func NewMockLocationService(db *gorm.DB) *MockLocationService {
	return &MockLocationService{db: db}
}

// GetStoreLocation implements LocationServiceInterface by querying tenant directly
func (m *MockLocationService) GetStoreLocation(tenantID uuid.UUID) (*StoreLocationInfo, error) {
	var tenant models.Tenant
	err := m.db.Where("id = ?", tenantID).First(&tenant).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant: %w", err)
	}

	// Build address string
	addressParts := []string{}
	if tenant.StoreStreet != "" {
		addressParts = append(addressParts, tenant.StoreStreet)
	}
	if tenant.StoreNumber != "" {
		addressParts = append(addressParts, tenant.StoreNumber)
	}
	if tenant.StoreNeighborhood != "" {
		addressParts = append(addressParts, tenant.StoreNeighborhood)
	}
	if tenant.StoreCity != "" {
		addressParts = append(addressParts, tenant.StoreCity)
	}
	if tenant.StoreState != "" {
		addressParts = append(addressParts, tenant.StoreState)
	}

	address := ""
	if len(addressParts) > 0 {
		address = addressParts[0]
		for _, part := range addressParts[1:] {
			address += ", " + part
		}
	}

	latitude := 0.0
	longitude := 0.0
	if tenant.StoreLatitude != nil {
		latitude = *tenant.StoreLatitude
	}
	if tenant.StoreLongitude != nil {
		longitude = *tenant.StoreLongitude
	}

	return &StoreLocationInfo{
		Address:     address,
		City:        tenant.StoreCity,
		State:       tenant.StoreState,
		Coordinates: tenant.StoreLatitude != nil && tenant.StoreLongitude != nil && *tenant.StoreLatitude != 0 && *tenant.StoreLongitude != 0,
		Latitude:    latitude,
		Longitude:   longitude,
		RadiusKm:    float64(tenant.DeliveryRadiusKm),
	}, nil
}
