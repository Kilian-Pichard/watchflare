package config

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	GRPCPort    string
	DatabaseURL string
	JWTSecret          string
	SMTPEncryptionKey  string
	CORSOrigins        []string
	Environment  string
	CookieDomain string // Domain for JWT cookie (empty = localhost, set in production)
	CookieSecure bool   // Secure flag on JWT cookie — set false when serving over HTTP without TLS

	// TLS Configuration
	TLSMode   string // "auto" or "custom"
	TLSPKIDir string // For auto mode

	// Custom TLS (user-provided certificates)
	TLSCertFile string
	TLSKeyFile  string
	TLSCAFile   string

	// gRPC Security
	GRPCTimestampWindow int // HMAC is always required
}

var AppConfig *Config

// Load reads configuration from environment variables
func Load() {
	// Load .env file if it exists (ignore error in production)
	_ = godotenv.Load()

	AppConfig = &Config{
		GRPCPort: getEnv("GRPC_PORT", "50051"),
		DatabaseURL: fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
			getEnv("POSTGRES_HOST", "localhost"),
			getEnv("POSTGRES_PORT", "5432"),
			getEnv("POSTGRES_USER", "watchflare"),
			getEnv("POSTGRES_PASSWORD", "watchflare_dev"),
			getEnv("POSTGRES_DB", "watchflare"),
			getEnv("POSTGRES_SSLMODE", "disable"),
		),
		JWTSecret:         getEnv("JWT_SECRET", ""),
		SMTPEncryptionKey: getEnv("SMTP_ENCRYPTION_KEY", ""),
		CORSOrigins:       parseOrigins(getEnv("CORS_ORIGINS", "http://localhost:5173")),
		Environment:  getEnv("ENV", "development"),
		CookieDomain: getEnv("COOKIE_DOMAIN", ""), // Empty for dev (localhost), set in production
		CookieSecure: getBoolEnv("COOKIE_SECURE", false),

		// TLS Configuration
		TLSMode:   getEnv("TLS_MODE", "auto"),
		TLSPKIDir: getEnv("TLS_PKI_DIR", "/var/lib/watchflare/pki"),

		// Custom TLS
		TLSCertFile: getEnv("TLS_CERT_FILE", ""),
		TLSKeyFile:  getEnv("TLS_KEY_FILE", ""),
		TLSCAFile:   getEnv("TLS_CA_FILE", ""),

		// gRPC Security (HMAC always required)
		GRPCTimestampWindow: getIntEnv("GRPC_TIMESTAMP_WINDOW", 300),
	}

	// Validate required fields
	if AppConfig.JWTSecret == "" {
		slog.Error("JWT_SECRET is required in environment variables")
		os.Exit(1)
	}

	// Validate JWT secret strength (minimum 32 characters for 256-bit security)
	if len(AppConfig.JWTSecret) < 32 {
		slog.Error("JWT_SECRET too short",
			"current_length", len(AppConfig.JWTSecret),
			"required", 32,
			"hint", "Generate a secure secret: openssl rand -base64 32",
		)
		os.Exit(1)
	}

	// Validate SMTP encryption key — required if SMTP is configured, warn otherwise
	if AppConfig.SMTPEncryptionKey == "" {
		slog.Warn("SMTP_ENCRYPTION_KEY is not set — SMTP password storage will be unavailable",
			"hint", "Generate a secure key: openssl rand -base64 32",
		)
	} else if len(AppConfig.SMTPEncryptionKey) < 32 {
		slog.Error("SMTP_ENCRYPTION_KEY too short",
			"current_length", len(AppConfig.SMTPEncryptionKey),
			"required", 32,
			"hint", "Generate a secure key: openssl rand -base64 32",
		)
		os.Exit(1)
	}

	// Warn if JWT secret looks weak (common patterns)
	weakSecrets := []string{"secret", "password", "admin", "test", "dev", "change", "please"}
	secretLower := strings.ToLower(AppConfig.JWTSecret)
	for _, weak := range weakSecrets {
		if strings.Contains(secretLower, weak) {
			slog.Warn("JWT_SECRET contains common word — use a cryptographically random string", "word", weak)
			break
		}
	}

	// Warn about cookie security
	if !AppConfig.CookieSecure {
		slog.Warn("COOKIE_SECURE is false — JWT cookies are not marked Secure, set to true when serving over HTTPS")
	} else if AppConfig.Environment != "production" {
		slog.Warn("COOKIE_SECURE is true in a non-production environment — ensure you are serving over HTTPS or cookies will not be sent by browsers")
	}

	slog.Info("configuration loaded",
		"grpc_port", AppConfig.GRPCPort,
		"environment", AppConfig.Environment,
		"cookie_secure", AppConfig.CookieSecure,
	)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func parseOrigins(originsStr string) []string {
	if originsStr == "" {
		return []string{}
	}
	origins := strings.Split(originsStr, ",")
	for i, origin := range origins {
		origins[i] = strings.TrimSpace(origin)
	}
	return origins
}

func getBoolEnv(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		b, err := strconv.ParseBool(value)
		if err != nil {
			slog.Warn("invalid boolean env var, using default", "key", key, "default", defaultValue)
			return defaultValue
		}
		return b
	}
	return defaultValue
}

func getIntEnv(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		intVal, err := strconv.Atoi(value)
		if err != nil {
			slog.Warn("invalid integer env var, using default", "key", key, "default", defaultValue)
			return defaultValue
		}
		return intVal
	}
	return defaultValue
}
