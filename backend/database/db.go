package database

import (
	"embed"
	"fmt"
	"log/slog"
	"os"
	"watchflare/backend/models"

	applogger "watchflare/backend/logger"

	"github.com/pressly/goose/v3"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

//go:embed migrations/*.sql
var embedMigrations embed.FS

var DB *gorm.DB

// Connect establishes the database connection and runs all migrations.
func Connect(dsn string) error {
	var err error

	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: applogger.NewGORMLogger(),
	})
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	slog.Info("database connected")

	// Enable TimescaleDB extension (non-fatal: app can still run without it,
	// but time-series features will be unavailable)
	if err := DB.Exec("CREATE EXTENSION IF NOT EXISTS timescaledb CASCADE").Error; err != nil {
		slog.Warn("timescaledb extension unavailable", "error", err)
	} else {
		slog.Info("timescaledb extension enabled")
	}

	// AutoMigrate core tables — hosts and users must exist before goose runs
	// since several migrations reference hosts(id) and users(id) via FK.
	if err := DB.AutoMigrate(&models.User{}, &models.Host{}); err != nil {
		return fmt.Errorf("failed to migrate core tables: %w", err)
	}
	slog.Info("core tables ready")

	// Create the metrics hypertable before goose migrations run:
	// migration 001 (continuous aggregates) depends on it existing.
	if err := initMetricsTable(); err != nil {
		return fmt.Errorf("failed to initialize metrics table: %w", err)
	}

	// Run SQL migrations
	sqlDB, err := DB.DB()
	if err != nil {
		return fmt.Errorf("failed to get raw DB connection: %w", err)
	}

	goose.SetBaseFS(embedMigrations)
	goose.SetLogger(&slogGooseLogger{})
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("failed to set goose dialect: %w", err)
	}
	if err := goose.Up(sqlDB, "migrations"); err != nil {
		return fmt.Errorf("failed to run SQL migrations: %w", err)
	}
	slog.Info("SQL migrations complete")

	return nil
}

// initMetricsTable creates the metrics hypertable if it doesn't exist.
// This must run before goose migrations since migration 001 depends on it.
func initMetricsTable() error {
	if err := DB.Exec(`
		CREATE TABLE IF NOT EXISTS metrics (
			id                       CHAR(36) NOT NULL,
			host_id                  CHAR(36) NOT NULL,
			timestamp                TIMESTAMPTZ NOT NULL,
			cpu_usage_percent        DOUBLE PRECISION,
			memory_total_bytes       BIGINT,
			memory_used_bytes        BIGINT,
			memory_available_bytes   BIGINT,
			load_avg1_min            DOUBLE PRECISION,
			load_avg5_min            DOUBLE PRECISION,
			load_avg15_min           DOUBLE PRECISION,
			disk_total_bytes         BIGINT,
			disk_used_bytes          BIGINT,
			disk_read_bytes_per_sec  BIGINT DEFAULT 0,
			disk_write_bytes_per_sec BIGINT DEFAULT 0,
			network_rx_bytes_per_sec BIGINT DEFAULT 0,
			network_tx_bytes_per_sec BIGINT DEFAULT 0,
			cpu_temperature_celsius  DOUBLE PRECISION DEFAULT 0,
			sensor_readings          JSONB,
			uptime_seconds           BIGINT,
			created_at               TIMESTAMPTZ DEFAULT NOW(),
			PRIMARY KEY (id, timestamp)
		)
	`).Error; err != nil {
		return fmt.Errorf("failed to create metrics table: %w", err)
	}

	if err := DB.Exec(
		`SELECT create_hypertable('metrics', 'timestamp', if_not_exists => TRUE, migrate_data => TRUE)`,
	).Error; err != nil {
		return fmt.Errorf("failed to create metrics hypertable: %w", err)
	}

	return nil
}

// slogGooseLogger adapts goose's Logger interface to log/slog.
type slogGooseLogger struct{}

func (l *slogGooseLogger) Fatalf(format string, v ...any) {
	slog.Error(fmt.Sprintf(format, v...))
	os.Exit(1)
}

func (l *slogGooseLogger) Printf(format string, v ...any) {
	slog.Info(fmt.Sprintf(format, v...))
}
