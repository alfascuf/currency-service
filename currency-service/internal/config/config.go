package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config holds all application configuration
type Config struct {
	// Server ports
	Port     string
	GRPCPort string // ← Добавлено для gRPC

	// External API
	ExchangeApiKey string
	BaseCurrency   string
	CurrencyAPIURL string
	Environment    string

	// Database
	DatabaseURL    string
	DatabaseDriver string

	// Redis configuration
	RedisHost     string
	RedisPort     string
	RedisPassword string
	RedisDB       int
	CacheTTL      int
}

// Load reads configuration from environment variables and .env file
func Load() *Config {
	// Приоритет: .env.local (для локальной разработки)
	if err := godotenv.Load(".env.local"); err != nil {
		// Попробовать ../.env.local
		if err := godotenv.Load("../.env.local"); err != nil {
			// Fallback на обычный .env
			if err := godotenv.Load(".env"); err != nil {
				godotenv.Load("../.env")
			}
		}
	}

	cfg := &Config{
		// Server
		Port:     getEnv("PORT", "8082"),
		GRPCPort: getEnv("GRPC_PORT", "50051"), // ← Добавлено

		// External API
		ExchangeApiKey: getEnv("EXCHANGE_API_KEY", ""),
		BaseCurrency:   getEnv("BASE_CURRENCY", "USD"),
		CurrencyAPIURL: getEnv("CURRENCY_API_URL", "https://api.frankfurter.dev"),
		Environment:    getEnv("ENVIRONMENT", "development"),

		// Database
		DatabaseURL:    getEnv("DATABASE_URL", "currency.db"),
		DatabaseDriver: getEnv("DATABASE_DRIVER", "postgres"),

		// Redis
		RedisHost:     getEnv("REDIS_HOST", "localhost"),
		RedisPort:     getEnv("REDIS_PORT", "6379"),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		RedisDB:       getEnvAsInt("REDIS_DB", 0),
		CacheTTL:      getEnvAsInt("CACHE_TTL", 3600),
	}
	return cfg
}

// getEnv retrieves environment variable value or returns default
func getEnv(key string, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

// getEnvAsInt helper for int
func getEnvAsInt(key string, defaultValue int) int {
	valueStr := getEnv(key, "")
	if valueStr == "" {
		return defaultValue
	}
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}
	return value
}
