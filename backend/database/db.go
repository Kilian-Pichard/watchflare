package database

import (
	_ "embed"
	"fmt"
	"log"
	"os"
	"watchflare/backend/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

//go:embed migrations/001_continuous_aggregates.sql
var continuousAggregatesSQL string

//go:embed migrations/002_dropped_metrics.sql
var droppedMetricsSQL string

//go:embed migrations/003_packages.sql
var packagesSQL string

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

	// Auto-migrate models (excluding Metric - managed by TimescaleDB migrations)
	err = DB.AutoMigrate(
		&models.User{},
		&models.Server{},
	)
	if err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	log.Println("Database migrations completed")

	// Create metrics table manually (since it's excluded from AutoMigrate due to TimescaleDB compression)
	err = DB.Exec(`
		CREATE TABLE IF NOT EXISTS metrics (
			id SERIAL PRIMARY KEY,
			server_id CHAR(36) NOT NULL,
			timestamp TIMESTAMPTZ NOT NULL,
			cpu_usage_percent DOUBLE PRECISION,
			memory_total_bytes BIGINT,
			memory_used_bytes BIGINT,
			memory_available_bytes BIGINT,
			load_avg1_min DOUBLE PRECISION,
			load_avg5_min DOUBLE PRECISION,
			load_avg15_min DOUBLE PRECISION,
			disk_total_bytes BIGINT,
			disk_used_bytes BIGINT,
			uptime_seconds BIGINT,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW()
		);
	`).Error
	if err != nil {
		log.Printf("Warning: Failed to create metrics table (may already exist): %v", err)
	} else {
		log.Println("Metrics table created/verified")
	}

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

	// Run continuous aggregates migration
	if err := RunContinuousAggregatesMigration(); err != nil {
		log.Printf("Warning: Failed to run continuous aggregates migration: %v", err)
	}

	// Run dropped metrics migration
	if err := RunDroppedMetricsMigration(); err != nil {
		log.Printf("Warning: Failed to run dropped metrics migration: %v", err)
	}

	// Run packages migration
	if err := RunPackagesMigration(); err != nil {
		log.Printf("Warning: Failed to run packages migration: %v", err)
	}

	return nil
}

// RunContinuousAggregatesMigration runs the continuous aggregates migration
func RunContinuousAggregatesMigration() error {
	log.Println("Running continuous aggregates migration...")

	// Execute migration (SQL embedded at compile time)
	if err := DB.Exec(continuousAggregatesSQL).Error; err != nil {
		return fmt.Errorf("failed to execute migration: %w", err)
	}

	log.Println("✓ Continuous aggregates migration completed successfully")
	return nil
}

// RunDroppedMetricsMigration runs the dropped metrics migration
func RunDroppedMetricsMigration() error {
	log.Println("Running dropped metrics migration...")

	// Execute migration (SQL embedded at compile time)
	if err := DB.Exec(droppedMetricsSQL).Error; err != nil {
		return fmt.Errorf("failed to execute migration: %w", err)
	}

	log.Println("✓ Dropped metrics migration completed successfully")
	return nil
}

// RunPackagesMigration runs the packages migration
func RunPackagesMigration() error {
	log.Println("Running packages migration...")

	// Execute migration (SQL embedded at compile time)
	if err := DB.Exec(packagesSQL).Error; err != nil {
		return fmt.Errorf("failed to execute migration: %w", err)
	}

	log.Println("✓ Packages migration completed successfully")
	return nil
}

// getEnv retrieves environment variable or returns default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
