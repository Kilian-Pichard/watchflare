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
	Port        string
	GRPCPort    string
	DatabaseURL string
	JWTSecret   string
	CORSOrigins  []string
	Environment  string
	CookieDomain string // Domain for JWT cookie (empty = localhost, set in production)

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
		Port:     getEnv("DEFAULT_PORT", "8080"),
		GRPCPort: getEnv("GRPC_PORT", "50051"),
		DatabaseURL: fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
			getEnv("POSTGRES_HOST", "localhost"),
			getEnv("POSTGRES_PORT", "5432"),
			getEnv("POSTGRES_USER", "watchflare"),
			getEnv("POSTGRES_PASSWORD", "watchflare_dev"),
			getEnv("POSTGRES_DB", "watchflare"),
			getEnv("POSTGRES_SSLMODE", "disable"),
		),
		JWTSecret: getEnv("JWT_SECRET", ""),
		CORSOrigins:  parseOrigins(getEnv("CORS_ORIGINS", "http://localhost:5173")),
		Environment:  getEnv("ENV", "development"),
		CookieDomain: getEnv("COOKIE_DOMAIN", ""), // Empty for dev (localhost), set in production

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

	// Warn if JWT secret looks weak (common patterns)
	weakSecrets := []string{"secret", "password", "admin", "test", "dev", "change", "please"}
	secretLower := strings.ToLower(AppConfig.JWTSecret)
	for _, weak := range weakSecrets {
		if strings.Contains(secretLower, weak) {
			slog.Warn("JWT_SECRET contains common word — use a cryptographically random string", "word", weak)
			break
		}
	}

	slog.Info("configuration loaded",
		"port", AppConfig.Port,
		"grpc_port", AppConfig.GRPCPort,
		"environment", AppConfig.Environment,
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
