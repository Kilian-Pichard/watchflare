package database

import (
	"fmt"
	"log"
	"os"
	"watchflare/backend/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

// Connect establishes database connection and runs migrations
func Connect() error {
	var err error

	// Build PostgreSQL DSN from environment variables
	host := getEnv("POSTGRES_HOST", "localhost")
	port := getEnv("POSTGRES_PORT", "5432")
	user := getEnv("POSTGRES_USER", "watchflare")
	password := getEnv("POSTGRES_PASSWORD", "watchflare_dev")
	dbname := getEnv("POSTGRES_DB", "watchflare")
	sslmode := getEnv("POSTGRES_SSLMODE", "disable")

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		host, port, user, password, dbname, sslmode)

	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	log.Println("Database connected successfully")

	// Enable TimescaleDB extension
	if err := DB.Exec("CREATE EXTENSION IF NOT EXISTS timescaledb CASCADE").Error; err != nil {
		log.Printf("Warning: Failed to enable TimescaleDB extension: %v", err)
	} else {
		log.Println("TimescaleDB extension enabled")
	}

	// Auto-migrate models
	err = DB.AutoMigrate(
		&models.User{},
		&models.Server{},
		&models.Metric{},
	)
	if err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	log.Println("Database migrations completed")

	// Convert metrics table to TimescaleDB hypertable
	// This should only be done once, TimescaleDB will handle it gracefully if already a hypertable
	err = DB.Exec(`
		SELECT create_hypertable(
			'metrics',
			'timestamp',
			if_not_exists => TRUE,
			migrate_data => TRUE
		);
	`).Error
	if err != nil {
		log.Printf("Warning: Failed to create hypertable (may already exist): %v", err)
	} else {
		log.Println("TimescaleDB hypertable 'metrics' created/verified")
	}

	// Add retention policy: keep metrics for 30 days
	err = DB.Exec(`
		SELECT add_retention_policy(
			'metrics',
			INTERVAL '30 days',
			if_not_exists => TRUE
		);
	`).Error
	if err != nil {
		log.Printf("Warning: Failed to add retention policy: %v", err)
	} else {
		log.Println("TimescaleDB retention policy added (30 days)")
	}

	return nil
}

// getEnv retrieves environment variable or returns default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
