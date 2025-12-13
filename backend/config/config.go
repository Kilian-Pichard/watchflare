package config

import (
	"log"
	"os"
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
