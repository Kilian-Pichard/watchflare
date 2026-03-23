package database

import (
	"database/sql"
	_ "embed"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"watchflare/backend/models"

	applogger "watchflare/backend/logger"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

//go:embed migrations/001_continuous_aggregates.sql
var continuousAggregatesSQL string

//go:embed migrations/002_dropped_metrics.sql
var droppedMetricsSQL string

//go:embed migrations/003_packages.sql
var packagesSQL string

//go:embed migrations/004_environment_detection.sql
var environmentDetectionSQL string

//go:embed migrations/006_new_metrics.sql
var newMetricsSQL string

//go:embed migrations/007_container_metrics.sql
var containerMetricsSQL string

//go:embed migrations/008_container_continuous_aggregates.sql
var containerContinuousAggregatesSQL string

//go:embed migrations/009_username.sql
var usernameSQL string

//go:embed migrations/010_agent_version.sql
var agentVersionSQL string

//go:embed migrations/011_sensor_readings.sql
var sensorReadingsSQL string

//go:embed migrations/013_sensor_metrics.sql
var sensorMetricsSQL string

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
		Logger: applogger.NewGORMLogger(),
	})
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	slog.Info("database connected")

	// Enable TimescaleDB extension
	if err := DB.Exec("CREATE EXTENSION IF NOT EXISTS timescaledb CASCADE").Error; err != nil {
		slog.Warn("failed to enable TimescaleDB extension", "error", err)
	} else {
		slog.Info("TimescaleDB extension enabled")
	}

	// Auto-migrate models (excluding Metric - managed by TimescaleDB migrations)
	err = DB.AutoMigrate(
		&models.User{},
		&models.Server{},
	)
	if err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	slog.Info("database migrations completed")

	// Create metrics table manually (since it's excluded from AutoMigrate due to TimescaleDB compression)
	err = DB.Exec(`
		CREATE TABLE IF NOT EXISTS metrics (
			id CHAR(36) NOT NULL,
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
			disk_read_bytes_per_sec BIGINT DEFAULT 0,
			disk_write_bytes_per_sec BIGINT DEFAULT 0,
			network_rx_bytes_per_sec BIGINT DEFAULT 0,
			network_tx_bytes_per_sec BIGINT DEFAULT 0,
			cpu_temperature_celsius DOUBLE PRECISION DEFAULT 0,
			uptime_seconds BIGINT,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW(),
			PRIMARY KEY (id, timestamp)
		);
	`).Error
	if err != nil {
		slog.Warn("failed to create metrics table (may already exist)", "error", err)
	} else {
		slog.Info("metrics table created/verified")
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
		slog.Warn("failed to create hypertable (may already exist)", "error", err)
	} else {
		slog.Info("TimescaleDB hypertable created/verified", "table", "metrics")
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
		slog.Warn("failed to add retention policy", "error", err)
	} else {
		slog.Info("TimescaleDB retention policy set", "interval", "30d")
	}

	// Run SQL migrations (idempotent — warnings for already-existing objects are expected)
	slog.Info("running SQL migrations")
	RunContinuousAggregatesMigration()
	RunDroppedMetricsMigration()
	RunPackagesMigration()
	RunEnvironmentDetectionMigration()
	RunNewMetricsMigration()
	RunContainerMetricsMigration()
	RunContainerContinuousAggregatesMigration()
	RunUsernameMigration()
	RunAgentVersionMigration()
	RunSensorReadingsMigration()
	RunSensorMetricsMigration()
	slog.Info("SQL migrations complete")

	return nil
}

// RunContinuousAggregatesMigration runs the continuous aggregates migration
// Statements are executed individually outside transactions because
// refresh_continuous_aggregate() cannot run inside a transaction block.
func RunContinuousAggregatesMigration() error {
	sqlDB, err := DB.DB()
	if err != nil {
		return fmt.Errorf("failed to get raw DB connection: %w", err)
	}
	return execStatementsOutsideTx(sqlDB, continuousAggregatesSQL)
}

// execStatementsOutsideTx splits SQL into individual statements and executes
// each one outside a transaction. Skips comments and empty statements.
func execStatementsOutsideTx(db *sql.DB, sqlContent string) error {
	statements := strings.Split(sqlContent, ";")
	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		// Skip comment-only blocks
		lines := strings.Split(stmt, "\n")
		hasCode := false
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if trimmed != "" && !strings.HasPrefix(trimmed, "--") {
				hasCode = true
				break
			}
		}
		if !hasCode {
			continue
		}

		if _, err := db.Exec(stmt); err != nil {
			slog.Warn("migration statement failed (may be idempotent)", "error", err)
		}
	}
	return nil
}

// RunDroppedMetricsMigration runs the dropped metrics migration
func RunDroppedMetricsMigration() error {
	sqlDB, err := DB.DB()
	if err != nil {
		return fmt.Errorf("failed to get raw DB connection: %w", err)
	}
	return execStatementsOutsideTx(sqlDB, droppedMetricsSQL)
}

// RunPackagesMigration runs the packages migration
func RunPackagesMigration() error {
	sqlDB, err := DB.DB()
	if err != nil {
		return fmt.Errorf("failed to get raw DB connection: %w", err)
	}
	return execStatementsOutsideTx(sqlDB, packagesSQL)
}

// RunEnvironmentDetectionMigration runs the environment detection migration
func RunEnvironmentDetectionMigration() error {
	sqlDB, err := DB.DB()
	if err != nil {
		return fmt.Errorf("failed to get raw DB connection: %w", err)
	}
	return execStatementsOutsideTx(sqlDB, environmentDetectionSQL)
}

// RunNewMetricsMigration runs the new metrics migration (disk I/O, network, temperature)
func RunNewMetricsMigration() error {
	sqlDB, err := DB.DB()
	if err != nil {
		return fmt.Errorf("failed to get raw DB connection: %w", err)
	}
	return execStatementsOutsideTx(sqlDB, newMetricsSQL)
}

// RunContainerMetricsMigration runs the container metrics migration
func RunContainerMetricsMigration() error {
	sqlDB, err := DB.DB()
	if err != nil {
		return fmt.Errorf("failed to get raw DB connection: %w", err)
	}
	return execStatementsOutsideTx(sqlDB, containerMetricsSQL)
}

// RunContainerContinuousAggregatesMigration runs the container continuous aggregates migration
func RunContainerContinuousAggregatesMigration() error {
	sqlDB, err := DB.DB()
	if err != nil {
		return fmt.Errorf("failed to get raw DB connection: %w", err)
	}
	return execStatementsOutsideTx(sqlDB, containerContinuousAggregatesSQL)
}

// RunAgentVersionMigration runs the agent version migration
func RunAgentVersionMigration() error {
	sqlDB, err := DB.DB()
	if err != nil {
		return fmt.Errorf("failed to get raw DB connection: %w", err)
	}
	return execStatementsOutsideTx(sqlDB, agentVersionSQL)
}

// RunUsernameMigration runs the username migration
func RunUsernameMigration() error {
	sqlDB, err := DB.DB()
	if err != nil {
		return fmt.Errorf("failed to get raw DB connection: %w", err)
	}
	return execStatementsOutsideTx(sqlDB, usernameSQL)
}

// RunSensorReadingsMigration runs the sensor readings migration
func RunSensorReadingsMigration() error {
	sqlDB, err := DB.DB()
	if err != nil {
		return fmt.Errorf("failed to get raw DB connection: %w", err)
	}
	return execStatementsOutsideTx(sqlDB, sensorReadingsSQL)
}

// RunSensorMetricsMigration runs the sensor metrics hypertable migration
func RunSensorMetricsMigration() error {
	sqlDB, err := DB.DB()
	if err != nil {
		return fmt.Errorf("failed to get raw DB connection: %w", err)
	}
	return execStatementsOutsideTx(sqlDB, sensorMetricsSQL)
}

// getEnv retrieves environment variable or returns default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
