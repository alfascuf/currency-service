package config

import (
	"os"

	"github.com/joho/godotenv"
)

// Config holds all application configuration
type Config struct {
	Port           string // HTTP server port
	ExchangeApiKey string // External API key for currency rates
	BaseCurrency   string // Base currency for conversions (e.g., RUB)
	DatabaseURL    string // PostgreSQL connection string
	DatabaseDriver string // Database driver name (postgres)
	CurrencyAPIURL string // External currency API URL
	Environment    string // Environment: development or production
}

// Load reads configuration from environment variables and .env file
func Load() *Config {
	// Try to load .env file (Ignore in docker)
	_ = godotenv.Load(".env")

	cfg := &Config{
		Port:           getEnv("PORT", "8080"),
		ExchangeApiKey: getEnv("EXCHANGE_API_KEY", ""),
		BaseCurrency:   getEnv("BASE_CURRENCY", "USD"),
		DatabaseURL:    getEnv("DATABASE_URL", "currency.db"),
		DatabaseDriver: getEnv("DATABASE_DRIVER", "postgres"),
		CurrencyAPIURL: getEnv("CURRENCY_API_URL", "https://api.frankfurter.dev"),
		Environment:    getEnv("ENVIRONMENT", "development"),
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
