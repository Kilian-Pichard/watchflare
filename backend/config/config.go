package config

import (
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Port         string
	GRPCPort     string
	DBPath       string
	JWTSecret    string
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
		Port:         "8080",
		GRPCPort:     getEnv("GRPC_PORT", "50051"),
		DBPath:       getEnv("DB_PATH", "./watchflare.db"),
		JWTSecret:    getEnv("JWT_SECRET", ""),
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
		log.Fatal("JWT_SECRET is required in environment variables")
	}

	// Validate JWT secret strength (minimum 32 characters for 256-bit security)
	if len(AppConfig.JWTSecret) < 32 {
		log.Fatalf("JWT_SECRET must be at least 32 characters (current: %d chars)\n"+
			"Generate a secure secret: openssl rand -base64 32", len(AppConfig.JWTSecret))
	}

	// Warn if JWT secret looks weak (common patterns)
	weakSecrets := []string{"secret", "password", "admin", "test", "dev", "change", "please"}
	secretLower := strings.ToLower(AppConfig.JWTSecret)
	for _, weak := range weakSecrets {
		if strings.Contains(secretLower, weak) {
			log.Printf("⚠️  WARNING: JWT_SECRET contains common word '%s' - use cryptographically random string", weak)
			break
		}
	}

	log.Printf("Configuration loaded: Port=%s, GRPCPort=%s, DB=%s, Environment=%s",
		AppConfig.Port, AppConfig.GRPCPort, AppConfig.DBPath, AppConfig.Environment)
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
		boolVal, err := strconv.ParseBool(value)
		if err != nil {
			log.Printf("Warning: Invalid boolean value for %s, using default: %v", key, defaultValue)
			return defaultValue
		}
		return boolVal
	}
	return defaultValue
}

func getIntEnv(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		intVal, err := strconv.Atoi(value)
		if err != nil {
			log.Printf("Warning: Invalid integer value for %s, using default: %d", key, defaultValue)
			return defaultValue
		}
		return intVal
	}
	return defaultValue
}
