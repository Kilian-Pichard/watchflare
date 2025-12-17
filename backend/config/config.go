package config

import (
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Port        string
	GRPCPort    string
	DBPath      string
	JWTSecret   string
	CORSOrigins []string
	Environment string

	// gRPC Security
	GRPCEnableTLS       bool
	GRPCCertFile        string
	GRPCKeyFile         string
	GRPCRequireHMAC     bool
	GRPCTimestampWindow int
}

var AppConfig *Config

// Load reads configuration from environment variables
func Load() {
	// Load .env file if it exists (ignore error in production)
	_ = godotenv.Load()

	AppConfig = &Config{
		Port:        getEnv("PORT", "8080"),
		GRPCPort:    getEnv("GRPC_PORT", "50051"),
		DBPath:      getEnv("DB_PATH", "./watchflare.db"),
		JWTSecret:   getEnv("JWT_SECRET", ""),
		CORSOrigins: parseOrigins(getEnv("CORS_ORIGINS", "http://localhost:5173")),
		Environment: getEnv("ENV", "development"),

		// gRPC Security
		GRPCEnableTLS:       getBoolEnv("GRPC_ENABLE_TLS", false),
		GRPCCertFile:        getEnv("GRPC_CERT_FILE", "/etc/watchflare/certs/server-cert.pem"),
		GRPCKeyFile:         getEnv("GRPC_KEY_FILE", "/etc/watchflare/certs/server-key.pem"),
		GRPCRequireHMAC:     getBoolEnv("GRPC_REQUIRE_HMAC", false),
		GRPCTimestampWindow: getIntEnv("GRPC_TIMESTAMP_WINDOW", 300),
	}

	// Validate required fields
	if AppConfig.JWTSecret == "" {
		log.Fatal("JWT_SECRET is required in environment variables")
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
