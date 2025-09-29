package db

import (
	"fmt"
	"iafarma/internal/services"
	"iafarma/pkg/models"
	"log"
	"os"
	"path/filepath"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// NewDatabase creates a new database connection
func NewDatabase() (*gorm.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=%s",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_SSLMODE"),
		os.Getenv("DB_TIMEZONE"),
	)

	var gormLogger logger.Interface
	// if os.Getenv("ENV") == "development" {
	// 	gormLogger = logger.Default.LogMode(logger.Info)
	// } else {
	// 	gormLogger = logger.Default.LogMode(logger.Silent)
	// }

	gormLogger = logger.Default.LogMode(logger.Error)

	config := &gorm.Config{
		Logger: gormLogger,
		// Habilitar criação automática de foreign key constraints
		DisableForeignKeyConstraintWhenMigrating: false,
	}

	db, err := gorm.Open(postgres.Open(dsn), config)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	return db, nil
}

// AutoMigrate runs database migrations using GORM
func AutoMigrate(db *gorm.DB) error {
	log.Println("Running GORM AutoMigrate...")

	// Create required extensions first
	if err := db.Exec(`CREATE EXTENSION IF NOT EXISTS "uuid-ossp"`).Error; err != nil {
		log.Printf("Warning: Could not create uuid-ossp extension: %v", err)
	}

	// Run GORM AutoMigrate with all models
	if err := db.AutoMigrate(models.GetAllModels()...); err != nil {
		return fmt.Errorf("failed to run GORM AutoMigrate: %w", err)
	}

	// Create any custom indexes that GORM might not handle automatically
	if err := createCustomIndexes(db); err != nil {
		log.Printf("Warning: Failed to create some custom indexes: %v", err)
	}

	// Import Brazilian municipalities if needed
	if err := ImportarMunicipiosBrasileiros(db); err != nil {
		log.Printf("Warning: Failed to import Brazilian municipalities: %v", err)
	}

	log.Println("GORM AutoMigrate completed successfully")
	return nil
}

// createCustomIndexes creates any custom indexes that GORM might not handle
func createCustomIndexes(db *gorm.DB) error {
	indexes := []string{
		// Unique constraint for tenant + session in channels (excluding empty sessions)
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_channels_tenant_session ON channels(tenant_id, session) WHERE session != ''`,

		// Composite unique constraint for notification controls
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_notification_controls_unique ON notification_controls(tenant_id, notification_type, reference_date)`,

		// Full text search index for products
		`CREATE INDEX IF NOT EXISTS idx_products_search ON products USING gin(to_tsvector('portuguese', coalesce(name, '') || ' ' || coalesce(description, '') || ' ' || coalesce(brand, '') || ' ' || coalesce(tags, '')))`,

		// Index for address default flag per customer
		`CREATE INDEX IF NOT EXISTS idx_addresses_customer_default ON addresses (customer_id, is_default) WHERE is_default = true`,

		// Index for conversation memory unique constraint
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_conversation_memory_tenant_phone ON conversation_memories(tenant_id, customer_phone)`,

		// Função para normalização de texto (necessária para busca de municípios)
		`CREATE OR REPLACE FUNCTION normalize_text(input_text TEXT) RETURNS TEXT AS $$
		BEGIN
			RETURN lower(
				translate(
					input_text,
					'áàâãäåéèêëíìîïóòôõöúùûüýÿñç',
					'aaaaaaeeeeiiiioooooouuuuyync'
				)
			);
		END;
		$$ LANGUAGE plpgsql IMMUTABLE;`,

		// Index para busca otimizada de municípios
		`CREATE INDEX IF NOT EXISTS idx_municipios_nome_uf ON municipios_brasileiros(normalize_text(nome_cidade), uf)`,
		`CREATE INDEX IF NOT EXISTS idx_municipios_uf ON municipios_brasileiros(uf)`,
	}

	for _, idx := range indexes {
		if err := db.Exec(idx).Error; err != nil {
			log.Printf("Warning: Failed to create index: %s - %v", idx, err)
		}
	}

	return nil
}

// SeedInitialData creates initial system data
func SeedInitialData(db *gorm.DB) error {
	log.Println("Seeding initial data...")

	// Check if admin user already exists
	var userCount int64
	if err := db.Model(&models.User{}).Where("role = ?", "system_admin").Count(&userCount).Error; err != nil {
		return fmt.Errorf("failed to check existing users: %w", err)
	}

	if userCount == 0 {
		// Create system admin user
		adminUser := models.User{
			Email:    "nilber+zapvendas@vittorazzi.com",
			Password: "$2a$10$ihq36CvkxLkl2FlsN1xI7.iRADfxaBLWHbNzdOCGzJYY/sqsCP1I2", // admin123
			Name:     "System Administrator",
			Role:     "system_admin",
			IsActive: true,
		}

		if err := db.Create(&adminUser).Error; err != nil {
			return fmt.Errorf("failed to create admin user: %w", err)
		}

		log.Println("Admin user created successfully")
	}

	return nil
}

// ImportarMunicipiosBrasileiros importa os municípios do CSV se a tabela estiver vazia
func ImportarMunicipiosBrasileiros(db *gorm.DB) error {
	// Check if municipalities already exist
	var count int64
	if err := db.Model(&models.MunicipioBrasileiro{}).Count(&count).Error; err != nil {
		return fmt.Errorf("failed to check municipalities count: %w", err)
	}

	if count > 0 {
		log.Printf("Brazilian municipalities already imported (%d records), skipping...", count)
		return nil
	}

	log.Println("Importing Brazilian municipalities from CSV...")

	// Get the current working directory and construct CSV path
	csvPath := filepath.Join(".", "municipios_com_uf.csv")

	// Check if file exists
	if _, err := os.Stat(csvPath); os.IsNotExist(err) {
		return fmt.Errorf("CSV file not found: %s", csvPath)
	}

	// Create municipality service and import
	municipioService := services.NewMunicipioService(db)
	if err := municipioService.ImportarMunicipiosFromCSV(csvPath); err != nil {
		return fmt.Errorf("failed to import municipalities: %w", err)
	}

	// Check how many were imported
	if err := db.Model(&models.MunicipioBrasileiro{}).Count(&count).Error; err == nil {
		log.Printf("Successfully imported %d Brazilian municipalities", count)
	}

	return nil
}

// RunMigrations is the main migration function called from main.go
func RunMigrations(db *gorm.DB) error {
	log.Println("Starting database migrations...")

	// Run GORM AutoMigrate
	if err := AutoMigrate(db); err != nil {
		return fmt.Errorf("AutoMigrate failed: %w", err)
	}

	// Seed initial data
	if err := SeedInitialData(db); err != nil {
		return fmt.Errorf("initial data seeding failed: %w", err)
	}

	log.Println("All migrations completed successfully")
	return nil
}
