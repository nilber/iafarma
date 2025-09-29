package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "iafarma/docs" // Import swagger docsx
	"iafarma/internal/app"
	"iafarma/internal/db"
	"iafarma/internal/http/handlers"
	"iafarma/internal/http/middleware"
	"iafarma/internal/telemetry"

	"github.com/go-playground/validator/v10"
	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	echomiddleware "github.com/labstack/echo/v4/middleware"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	echoSwagger "github.com/swaggo/echo-swagger"
)

// @title IAFarma API
// @version 1.0
// @description Multi-tenant SaaS para vendas e atendimento via WhatsApp
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:8080
// @BasePath /api/v1

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.

// CustomValidator wraps the validator
type CustomValidator struct {
	validator *validator.Validate
}

// Validate implements echo.Validator interface
func (cv *CustomValidator) Validate(i interface{}) error {
	if err := cv.validator.Struct(i); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	return nil
}

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Info().Msg("No .env file found, using environment variables")
	}

	// Setup logger
	zerolog.TimeFieldFormat = time.RFC3339
	if os.Getenv("ENV") == "development" {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	}

	// Initialize telemetry (optional service)
	shutdown, enabled, err := telemetry.InitTelemetry()
	if err != nil {
		log.Warn().Err(err).Msg("Failed to initialize telemetry, continuing without it")
		shutdown = func() {} // noop shutdown function
	} else if enabled {
		log.Info().Msg("Telemetry initialized successfully")
	} else {
		log.Info().Msg("Telemetry disabled")
	}
	defer shutdown()

	// Initialize database
	database, err := db.NewDatabase()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to database")
	}

	// Run migrations
	if err := db.RunMigrations(database); err != nil {
		log.Fatal().Err(err).Msg("Failed to run migrations")
	}

	// Initialize services
	services := app.NewServices(database)

	// Start channel monitor service
	if services.ChannelMonitorService != nil {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		go services.ChannelMonitorService.Start(ctx)
		log.Info().Msg("Channel monitor service started")

		// Start channel reconnection service
		if services.ChannelReconnectionService != nil {
			go services.ChannelReconnectionService.Start(ctx)
			log.Info().Msg("Channel reconnection service started")
		}

		// Start usage sync service
		if services.UsageSyncService != nil {
			go services.UsageSyncService.Start(ctx)
			log.Info().Msg("Usage sync service started")
		}
	} else {
		log.Warn().Msg("Channel monitor service not available")
	}

	// Start infrastructure monitoring service
	if services.InfrastructureMonitorService != nil {
		go services.InfrastructureMonitorService.Start()
		log.Info().Msg("Infrastructure monitoring service started")
	} else {
		log.Warn().Msg("Infrastructure monitoring service not available")
	}

	// Setup Echo
	e := echo.New()
	e.HideBanner = true

	// Set custom validator
	e.Validator = &CustomValidator{validator: validator.New()}

	// Middleware
	// e.Use(echomiddleware.Logger())
	e.Use(echomiddleware.Recover())
	e.Use(echomiddleware.CORS())
	e.Use(middleware.RequestID())
	e.Use(middleware.Telemetry())

	// Health check
	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	// Swagger - only enabled in development environment
	env := os.Getenv("ENV")
	if env == "development" {
		e.GET("/docs/*", echoSwagger.WrapHandler)
		e.GET("/swagger/*", echoSwagger.WrapHandler)
	}

	// Setup routes
	api := e.Group("/api/v1")

	handlers.SetupRoutes(api, services)

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	go func() {
		if err := e.Start(":" + port); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("Failed to start server")
		}
	}()

	log.Info().Str("port", port).Msg("Server started")

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info().Msg("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := e.Shutdown(ctx); err != nil {
		log.Error().Err(err).Msg("Server forced to shutdown")
	}

	log.Info().Msg("Server exited")
}
