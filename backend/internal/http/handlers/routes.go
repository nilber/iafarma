package handlers

import (
	"log"

	"iafarma/internal/ai"
	"iafarma/internal/app"
	"iafarma/internal/http/middleware"
	"iafarma/internal/repo"
	servicesPackage "iafarma/internal/services"
	"iafarma/internal/webhook"

	"github.com/labstack/echo/v4"
)

// SetupRoutes sets up all API routes
func SetupRoutes(api *echo.Group, services *app.Services) {
	// Initialize WebSocket handler
	wsHandler := NewWebSocketHandler(services.DB, services.AuthService)

	// Auth routes (no authentication required) - using security repos that will be created later
	authHandler := NewAuthHandler(services.AuthService, services.EmailService)
	auth := api.Group("/auth")
	auth.POST("/login", authHandler.Login)
	auth.POST("/refresh", authHandler.RefreshToken)
	auth.POST("/forgot-password", authHandler.ForgotPassword)
	auth.POST("/reset-password", authHandler.ResetPassword)

	// Protected routes (require authentication)
	protected := api.Group("")
	protected.Use(middleware.JWTAuth(services.AuthService))
	protected.Use(middleware.TenantResolver(services.DB))

	// User profile routes (authenticated users)
	profileAuth := protected.Group("/auth")
	profileAuth.PUT("/profile", authHandler.UpdateProfile)
	profileAuth.PUT("/change-password", authHandler.ChangePassword)

	// System admin routes (only system_admin with no tenant)
	admin := protected.Group("/admin")
	admin.Use(middleware.RequireSystemRole())
	tenantHandler := NewTenantHandler(services.TenantRepo, services.DB)
	admin.GET("/tenants", tenantHandler.List)
	admin.GET("/tenants/stats", tenantHandler.GetStats)
	admin.POST("/tenants", tenantHandler.Create)
	admin.GET("/tenants/:id", tenantHandler.GetByID)
	admin.PUT("/tenants/:id", tenantHandler.Update)
	admin.DELETE("/tenants/:id", tenantHandler.Delete)
	admin.GET("/tenants/:id/stats", tenantHandler.GetTenantStats)
	admin.GET("/stats", tenantHandler.GetSystemStats)

	// Channel management for super admin
	adminChannelHandler := NewAdminChannelHandler(services.ChannelRepo, services.PlanLimitService)
	admin.GET("/tenants/:tenant_id/channels", adminChannelHandler.ListByTenant)
	admin.POST("/tenants/:tenant_id/channels", adminChannelHandler.CreateForTenant)
	admin.PUT("/tenants/:tenant_id/channels/:id", adminChannelHandler.UpdateForTenant)
	admin.DELETE("/tenants/:tenant_id/channels/:id", adminChannelHandler.DeleteForTenant)

	// Notification management for super admin
	notificationHandler := NewNotificationHandler(services.DB, services.EmailService)
	admin.GET("/notifications", notificationHandler.AdminListNotifications)
	admin.GET("/notifications/stats", notificationHandler.AdminGetNotificationStats)
	admin.POST("/notifications/resend", notificationHandler.AdminResendNotification)
	admin.POST("/notifications/trigger", notificationHandler.AdminTriggerNotification)

	// Billing and Plans management for super admin
	billingHandler := NewBillingHandler(services.PlanRepo)
	admin.GET("/plans", billingHandler.ListPlans)
	admin.POST("/plans", billingHandler.CreatePlan)
	admin.GET("/plans/:id", billingHandler.GetPlan)
	admin.PUT("/plans/:id", billingHandler.UpdatePlan)
	admin.DELETE("/plans/:id", billingHandler.DeletePlan)
	admin.PUT("/tenants/:tenant_id/plan/:plan_id", billingHandler.UpdateTenantPlan)
	admin.GET("/tenants/:tenant_id/usage", billingHandler.GetTenantUsageByID)
	admin.POST("/tenants/:tenant_id/billing/reset", billingHandler.ResetBillingCycle)

	// Export management for super admin
	adminExportHandler := NewAdminExportHandler(services.DB)
	adminExport := admin.Group("/export")
	adminExport.GET("/tenants", adminExportHandler.GetTenantsList)
	adminExport.GET("/tenants/:tenant_id/products", adminExportHandler.ExportTenantProducts)

	// WebSocket endpoint (handles authentication manually via query parameter)
	api.GET("/ws", wsHandler.HandleWebSocket)

	// Municipios endpoints (accessible to authenticated users)
	municipioHandler := NewMunicipioHandler(services.DB)
	municipios := protected.Group("/municipios")
	municipios.GET("/estados", municipioHandler.GetEstados)
	municipios.GET("/cidades", municipioHandler.GetCidades)

	// Tenant routes (require tenant context and tenant_admin role)
	tenant := protected.Group("")
	tenant.Use(middleware.RequireTenantRole())
	tenant.Use(middleware.RequireTenant())

	// Tenant profile management
	tenant.GET("/tenant/profile", tenantHandler.GetProfile)
	tenant.PUT("/tenant/profile", tenantHandler.UpdateProfile)

	// User management (only tenant_admin can manage users)
	userHandler := NewUserHandler(services.DB, services.AuthService)

	// System admin can create tenant admins
	admin.POST("/tenant-admin", userHandler.CreateTenantAdmin)
	admin.POST("/send-tenant-credentials", userHandler.SendTenantCredentials)
	admin.GET("/tenants/:tenant_id/users", userHandler.GetTenantUsersForAdmin)
	admin.PUT("/tenant-admin/:user_id", userHandler.UpdateTenantAdminForAdmin)

	users := tenant.Group("/users", middleware.RequireTenantAdminOnly())
	users.POST("", userHandler.CreateTenantUser)
	users.GET("", userHandler.GetTenantUsers)
	users.GET("/:id", userHandler.GetTenantUser)
	users.PUT("/:id", userHandler.UpdateTenantUser)
	users.PUT("/:id/password", userHandler.ChangeUserPassword)
	users.DELETE("/:id", userHandler.DeleteTenantUser)

	// Billing and usage management for tenants
	billing := tenant.Group("/billing")
	billing.GET("/usage", billingHandler.GetTenantUsage)
	billing.GET("/usage/check", billingHandler.CheckUsageLimit)
	billing.POST("/usage/:resource_type/increment", billingHandler.IncrementUsage)

	// WhatsApp Management (initialize early to use in other routes)
	// TODO: Initialize WhatsApp client with proper config
	whatsappHandler := NewWhatsAppHandler(services.DB, nil, services.StorageService) // Will need proper client
	whatsappHandler.SetWebSocketHandler(wsHandler)

	// Products
	productHandler := NewProductHandler(services.ProductRepo, services.CategoryRepo, services.EmbeddingService, services.PlanLimitService, services.StorageService, services.DB)
	if services.EmbeddingService == nil {
		log.Printf("⚠️  WARNING: EmbeddingService is nil during ProductHandler initialization")
	} else {
		log.Printf("✅ EmbeddingService is available during ProductHandler initialization")
	}
	products := tenant.Group("/products")
	products.GET("", productHandler.List)            // Only products with stock > 0
	products.GET("/admin", productHandler.ListAdmin) // ALL products for admin purposes
	products.GET("/stats", productHandler.GetStats)  // Product statistics
	products.POST("", productHandler.Create)
	products.GET("/:id", productHandler.GetByID)
	products.PUT("/:id", productHandler.Update)
	products.DELETE("/:id", productHandler.Delete)
	products.POST("/import", productHandler.ImportProducts)
	products.POST("/import-image", productHandler.ImportProductsFromImage)
	products.GET("/import/template", productHandler.GetImportTemplate)
	products.GET("/search", productHandler.Search) // Endpoint de busca semântica

	// Product Images
	products.POST("/:id/upload-image", productHandler.UploadProductImage)
	products.GET("/:id/images", productHandler.GetProductImages)
	products.DELETE("/:id/images/:image_id", productHandler.DeleteProductImage)

	// Product Characteristics
	characteristicRepo := repo.NewProductCharacteristicRepository(services.DB)
	itemRepo := repo.NewCharacteristicItemRepository(services.DB)
	characteristicHandler := NewProductCharacteristicHandler(characteristicRepo, itemRepo)
	products.GET("/:product_id/characteristics", characteristicHandler.GetCharacteristics)
	products.POST("/:product_id/characteristics", characteristicHandler.CreateCharacteristic)
	products.GET("/:product_id/characteristics/:id", characteristicHandler.GetCharacteristic)
	products.PUT("/:product_id/characteristics/:id", characteristicHandler.UpdateCharacteristic)
	products.DELETE("/:product_id/characteristics/:id", characteristicHandler.DeleteCharacteristic)

	// Characteristic Items
	characteristics := tenant.Group("/characteristics")
	characteristics.GET("/:characteristic_id/items", characteristicHandler.GetCharacteristicItems)
	characteristics.POST("/:characteristic_id/items", characteristicHandler.CreateCharacteristicItem)
	characteristics.PUT("/:characteristic_id/items/:id", characteristicHandler.UpdateCharacteristicItem)
	characteristics.DELETE("/:characteristic_id/items/:id", characteristicHandler.DeleteCharacteristicItem)

	// Categories
	categoryHandler := NewCategoryHandler(services.CategoryService)
	categoryHandler.RegisterRoutes(tenant)

	// Import Jobs (Async product import)
	importJobHandler := NewImportJobHandler(services.ImportJobService)
	importJobs := tenant.Group("/import")
	importJobs.POST("/products", importJobHandler.CreateProductImportJob)
	importJobs.GET("/jobs", importJobHandler.ListImportJobs)
	importJobs.GET("/jobs/:id/progress", importJobHandler.GetImportJobProgress)

	// Conversations (RAG)
	conversationHandler := NewConversationHandler(services.EmbeddingService)
	conversations := tenant.Group("/conversations")
	conversations.POST("", conversationHandler.StoreConversation)
	conversations.POST("/search", conversationHandler.SearchConversations)
	conversations.GET("/context/:customer_id", conversationHandler.GetConversationContext)

	// AI Credits
	aiCreditHandler := NewAICreditHandler(services.DB)
	aiCredits := tenant.Group("/ai-credits")
	aiCredits.GET("", aiCreditHandler.GetCredits)
	aiCredits.POST("/use", aiCreditHandler.UseCredits)
	aiCredits.GET("/transactions", aiCreditHandler.GetTransactions)

	// AI Credits Admin (system admin only)
	adminAICredits := admin.Group("/ai-credits")
	adminAICredits.POST("/add", aiCreditHandler.AddCredits)
	adminAICredits.GET("/tenant/:tenant_id", aiCreditHandler.GetCreditsByTenantID)
	adminAICredits.GET("/tenant/:tenant_id/transactions", aiCreditHandler.GetTransactionsByTenantID)

	// AI Product Generation
	productAIHandler := NewProductAIHandler(services.DB)
	aiProducts := tenant.Group("/ai/products")
	aiProducts.POST("/generate", productAIHandler.GenerateProductInfo)
	aiProducts.POST("/estimate", productAIHandler.GetCreditEstimate)

	// Onboarding (for Sales tenants)
	onboardingHandler := NewOnboardingHandler(services.DB)
	onboarding := tenant.Group("/onboarding")
	onboarding.GET("/status", onboardingHandler.GetOnboardingStatus)
	onboarding.POST("/complete/:item_id", onboardingHandler.CompleteOnboardingItem)
	onboarding.POST("/dismiss", onboardingHandler.DismissOnboarding)

	// Customers
	customerHandler := NewCustomerHandler(services.CustomerRepo)
	customers := tenant.Group("/customers")
	customers.GET("", customerHandler.List)
	customers.POST("", customerHandler.Create)
	customers.GET("/:id", customerHandler.GetByID)
	customers.PUT("/:id", customerHandler.Update)
	// customers.DELETE("/:id", customerHandler.Delete) // TODO: Implement Delete method

	// Addresses
	addressHandler := NewAddressHandler(services.AddressRepo, services.CustomerRepo)
	addresses := tenant.Group("/addresses")
	addresses.POST("", addressHandler.CreateAddress)
	addresses.GET("/customer/:customer_id", addressHandler.GetAddressesByCustomer)
	addresses.GET("/:id", addressHandler.GetAddress)
	addresses.PUT("/:id", addressHandler.UpdateAddress)
	addresses.DELETE("/:id", addressHandler.DeleteAddress)
	addresses.POST("/:id/set-default", addressHandler.SetDefaultAddress)

	// Orders
	orderHandler := NewOrderHandler(services.OrderRepo, services.CustomerRepo, services.ProductRepo, services.DB)
	orders := tenant.Group("/orders")
	orders.GET("", orderHandler.List)
	orders.POST("", orderHandler.Create)
	orders.GET("/:id", orderHandler.GetByID)
	orders.PUT("/:id", orderHandler.Update)
	orders.POST("/send-email", orderHandler.SendEmail)

	// Order Items
	orders.POST("/:id/items", orderHandler.AddItem)
	orders.PUT("/:id/items/:item_id", orderHandler.UpdateItem)
	orders.DELETE("/:id/items/:item_id", orderHandler.RemoveItem)

	// Payment Methods
	paymentMethodHandler := NewPaymentMethodHandler(services.DB)
	paymentMethods := tenant.Group("/payment-methods")
	paymentMethodHandler.SetupRoutes(paymentMethods)

	// Channels
	channelHandler := NewChannelHandler(services.ChannelRepo, services.PlanLimitService)
	channels := tenant.Group("/channels")
	channels.GET("", channelHandler.List)
	channels.POST("", channelHandler.Create)
	channels.GET("/:id", channelHandler.GetByID)
	channels.PUT("/:id", channelHandler.Update)
	channels.DELETE("/:id", channelHandler.Delete)
	channels.POST("/:id/migrate-conversations", channelHandler.MigrateConversations)

	// Alerts
	alertHandler := NewAlertHandler(services.DB)
	alerts := tenant.Group("/alerts")
	alerts.GET("", alertHandler.GetAlerts)
	alerts.POST("", alertHandler.CreateAlert)
	alerts.GET("/:id", alertHandler.GetAlert)
	alerts.PUT("/:id", alertHandler.UpdateAlert)
	alerts.DELETE("/:id", alertHandler.DeleteAlert)

	// Messages
	messageHandler := NewMessageHandler(services.MessageRepo, services.DB)
	messages := tenant.Group("/messages")
	messages.GET("", messageHandler.ListByConversation)
	messages.POST("", messageHandler.Create)
	messages.GET("/:id", messageHandler.GetByID)
	messages.POST("/notes", messageHandler.CreateNote)

	// Message Templates
	templateHandler := NewMessageTemplateHandler(services.MessageTemplateRepo, services.DB)
	templates := tenant.Group("/message-templates")
	templates.GET("", templateHandler.List)
	templates.POST("", templateHandler.Create)
	templates.GET("/categories", templateHandler.GetCategories)
	templates.POST("/process", templateHandler.ProcessTemplate)
	templates.GET("/:id", templateHandler.GetByID)
	templates.PUT("/:id", templateHandler.Update)
	templates.DELETE("/:id", templateHandler.Delete)

	// Dashboard
	dashboardHandler := NewDashboardHandler(services.MessageRepo, services.DB)
	dashboard := tenant.Group("/dashboard")
	dashboard.GET("/unread-messages", dashboardHandler.GetUnreadMessages)
	dashboard.GET("/stats", dashboardHandler.GetStats)
	dashboard.GET("/conversation-counts", dashboardHandler.GetConversationCounts)

	// Analytics and Reports
	analyticsHandler := NewAnalyticsHandler(services.DB, services.OrderRepo, services.ProductRepo, services.CustomerRepo)
	analytics := tenant.Group("/analytics")
	analytics.GET("/sales", analyticsHandler.GetSalesAnalytics)
	analytics.GET("/orders", analyticsHandler.GetOrderStats)

	reports := tenant.Group("/reports")
	reports.GET("", analyticsHandler.GetReportsData)
	reports.GET("/top-products", analyticsHandler.GetTopProducts)

	// Tenant Settings - AI Configuration
	settingsHandler := NewTenantSettingsHandler(services.DB)

	settings := tenant.Group("/settings")
	settings.GET("", settingsHandler.GetSettings)
	settings.GET("/:key", settingsHandler.GetSetting)
	settings.PUT("/:key", settingsHandler.UpdateSetting)
	settings.POST("/ai/generate-examples", settingsHandler.GenerateAIExamples)
	settings.GET("/ai/welcome", settingsHandler.GetWelcomeMessage)
	settings.POST("/ai/generate-welcome", settingsHandler.GenerateWelcomeMessage)
	settings.POST("/ai/generate-auto-prompt", settingsHandler.GenerateAutoPrompt)
	settings.POST("/ai/reset-to-default", settingsHandler.ResetToDefault)
	settings.GET("/ai/context-limitation", settingsHandler.GetContextLimitation)
	settings.POST("/ai/context-limitation", settingsHandler.SetContextLimitation)
	settings.POST("/ai/context-limitation/reset", settingsHandler.ResetContextLimitation)
	settings.GET("/whatsapp-group-proxy", settingsHandler.GetWhatsAppGroupProxy)
	settings.POST("/whatsapp-group-proxy", settingsHandler.SetWhatsAppGroupProxy)

	// Delivery Management - Store Location and Delivery Zones
	deliveryHandler := NewDeliveryHandler(services.DeliveryService)
	delivery := tenant.Group("/delivery")
	delivery.GET("/store-location", deliveryHandler.GetStoreLocation)
	delivery.PUT("/store-location", deliveryHandler.UpdateStoreLocation)
	delivery.GET("/zones", deliveryHandler.GetDeliveryZones)
	delivery.POST("/zones", deliveryHandler.ManageDeliveryZone)
	delivery.POST("/validate", deliveryHandler.ValidateDeliveryAddress)

	// Conversations (standalone route)
	conversations.GET("", whatsappHandler.ListConversations)
	conversations.POST("/customer/:customerId", whatsappHandler.CreateOrFindConversationByCustomer)
	conversations.POST("/:id/archive", whatsappHandler.ArchiveConversation)
	conversations.POST("/:id/pin", whatsappHandler.PinConversation)
	conversations.POST("/:id/toggle-ai", whatsappHandler.ToggleAIConversation)

	// WhatsApp endpoints
	whatsapp := tenant.Group("/whatsapp")
	whatsapp.GET("/status", whatsappHandler.GetStatus)
	whatsapp.GET("/session-status", whatsappHandler.GetSessionStatus)
	whatsapp.GET("/qr", whatsappHandler.GetQRCode)
	whatsapp.GET("/conversations", whatsappHandler.ListConversations)
	whatsapp.GET("/conversations/:id", whatsappHandler.GetConversation)
	whatsapp.PUT("/conversations/:id", whatsappHandler.UpdateConversationStatus)
	whatsapp.POST("/conversations/:id/read", whatsappHandler.MarkAsRead)
	whatsapp.POST("/conversations/:id/archive", whatsappHandler.ArchiveConversation)
	whatsapp.POST("/conversations/:id/pin", whatsappHandler.PinConversation)
	whatsapp.POST("/conversations/:id/toggle-ai", whatsappHandler.ToggleAIConversation)
	whatsapp.POST("/send", whatsappHandler.SendMessage)
	whatsapp.PUT("/messages/:id/status", whatsappHandler.UpdateMessageStatus)
	// whatsapp.POST("/test-websocket", whatsappHandler.TestWebSocket)

	// Media upload and sending
	whatsapp.POST("/upload/media", whatsappHandler.UploadMedia)
	whatsapp.POST("/send/image", whatsappHandler.SendImage)
	whatsapp.POST("/send/document", whatsappHandler.SendDocument)
	whatsapp.POST("/send/audio", whatsappHandler.SendAudio)

	// Admin Error Logs (requires auth)
	errorLogHandler := NewErrorLogHandler(services.DB)
	adminGroup := api.Group("/admin", middleware.JWTAuth(services.AuthService))
	adminGroup.GET("/error-logs", errorLogHandler.GetErrorLogs)
	adminGroup.GET("/error-logs/stats", errorLogHandler.GetErrorStats)
	adminGroup.PUT("/error-logs/:id/resolve", errorLogHandler.ResolveError)

	// Webhooks (public, no auth required)
	zapPlusWebhookHandler := webhook.NewZapPlusWebhookHandler(services.DB)
	zapPlusWebhookHandler.SetWebSocketNotifier(wsHandler)
	webhooks := api.Group("/webhook")
	webhooks.POST("/zapplus", zapPlusWebhookHandler.ProcessZapPlusWebhook)

	// Configure AI service with WebSocket and RAG support
	if services.EmbeddingService != nil {
		// Create adapter to bridge the interface differences
		embeddingAdapter := createEmbeddingAdapter(services.EmbeddingService)

		// Configure ZapPlusWebhookHandler with embedding service
		zapPlusWebhookHandler.SetAIServiceWithEmbedding(embeddingAdapter)
	}

	// Test endpoints
	// test := api.Group("/test")
	// test.POST("/websocket", whatsappHandler.TestWebSocket)
	// test.POST("/ai", func(c echo.Context) error {
	// 	return testAI(c, services.DB, wsHandler)
	// })
	// test.POST("/reload-embedding", func(c echo.Context) error {
	// 	return reloadEmbeddingService(c, services, wsHandler, zapPlusWebhookHandler)
	// })

	// Channel monitoring endpoints (admin only)
	monitoring := protected.Group("/monitoring")
	monitoring.Use(middleware.RequireRole("admin", "super_admin"))
	channelMonitorHandler := NewChannelMonitorHandler(services.ChannelMonitorService)
	monitoring.GET("/channels/status", channelMonitorHandler.GetMonitoringStatus)

	// Health check
	api.GET("/health", func(c echo.Context) error {
		return c.JSON(200, map[string]string{"status": "ok"})
	})

	// Simple WebSocket for testing (without tenant validation)
	// openaiAPIKey := os.Getenv("OPENAI_API_KEY")
	// aiServiceFactory := ai.NewAIServiceFactory(services.DB, openaiAPIKey)
	// webhookHandler := webhook.NewZapPlusWebhookHandler(services.DB)
	// simpleWSHandler := NewSimpleWebSocketHandler(services.DB, aiServiceFactory, webhookHandler)
	// api.GET("/test-ws", simpleWSHandler.HandleSimpleWebSocket)

}

// createEmbeddingAdapter creates an adapter to bridge services.EmbeddingService to ai.EmbeddingServiceInterface
func createEmbeddingAdapter(embeddingService *servicesPackage.EmbeddingService) ai.EmbeddingServiceInterface {
	return ai.NewEmbeddingServiceAdapterWithFuncs(
		// SearchSimilarProducts adapter
		func(query, tenantID string, limit int) ([]ai.ProductSearchResult, error) {
			results, err := embeddingService.SearchSimilarProducts(query, tenantID, uint64(limit))
			if err != nil {
				return nil, err
			}

			// Convert []*servicesPackage.ProductSearchResult to []ai.ProductSearchResult
			aiResults := make([]ai.ProductSearchResult, len(results))
			for i, result := range results {
				if result != nil {
					aiResults[i] = ai.ProductSearchResult{
						ID:       result.ProductID, // servicesPackage uses ProductID, ai uses ID
						Text:     result.Text,
						Score:    result.Score,
						Metadata: result.Metadata,
					}
				}
			}
			return aiResults, nil
		},
		// SearchConversations adapter
		func(tenantID, customerID, query string, limit int) ([]ai.ConversationSearchResult, error) {
			results, err := embeddingService.SearchConversations(tenantID, customerID, query, limit)
			if err != nil {
				return nil, err
			}

			// Convert []servicesPackage.ConversationSearchResult to []ai.ConversationSearchResult
			aiResults := make([]ai.ConversationSearchResult, len(results))
			for i, result := range results {
				aiResults[i] = ai.ConversationSearchResult{
					ID:        result.ID,
					Score:     result.Score,
					Message:   result.Message,
					Response:  result.Response,
					Timestamp: result.Timestamp,
					Metadata:  result.Metadata,
				}
			}
			return aiResults, nil
		},
		// SearchConversationsWithMaxAge adapter
		func(tenantID, customerID, query string, limit int, maxAgeHours int) ([]ai.ConversationSearchResult, error) {
			results, err := embeddingService.SearchConversationsWithMaxAge(tenantID, customerID, query, limit, maxAgeHours)
			if err != nil {
				return nil, err
			}

			// Convert []servicesPackage.ConversationSearchResult to []ai.ConversationSearchResult
			aiResults := make([]ai.ConversationSearchResult, len(results))
			for i, result := range results {
				aiResults[i] = ai.ConversationSearchResult{
					ID:        result.ID,
					Score:     result.Score,
					Message:   result.Message,
					Response:  result.Response,
					Timestamp: result.Timestamp,
					Metadata:  result.Metadata,
				}
			}
			return aiResults, nil
		},
		// StoreConversation adapter
		func(tenantID, customerID string, entry ai.ConversationEntry) error {
			// Convert ai.ConversationEntry to servicesPackage.ConversationEntry
			servicesEntry := servicesPackage.ConversationEntry{
				ID:         entry.ID,
				TenantID:   entry.TenantID,
				CustomerID: entry.CustomerID,
				Message:    entry.Message,
				Response:   entry.Response,
				Timestamp:  entry.Timestamp,
				Metadata:   entry.Metadata,
			}
			return embeddingService.StoreConversation(tenantID, customerID, servicesEntry)
		},
		// CleanupOldConversations adapter
		func(tenantID, customerID string, maxAgeHours int) (int, error) {
			return embeddingService.CleanupOldConversations(tenantID, customerID, maxAgeHours)
		},
	)
}
