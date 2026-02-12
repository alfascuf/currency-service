package config

import (
	"os"

	"github.com/joho/godotenv"
)

// Config holds all application configuration
type Config struct {
	Port           string
	ExchangeApiKey string
	BaseCurrency   string
	DatabaseURL    string
	DatabaseDriver string
	CurrencyAPIURL string
	Environment    string
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
