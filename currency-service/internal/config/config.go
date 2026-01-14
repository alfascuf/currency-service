package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port           string
	ExchangeApiKey string
	BaseCurrency   string
	DatabaseURL    string
	DatabaseDriver string
}

func Load() *Config {
	err := godotenv.Load(".env")
	if err != nil {
		log.Println("Error loading .env file")
	}
	cfg := &Config{
		Port:           getEnv("PORT", "8080"),
		ExchangeApiKey: getEnv("EXCHANGE_API_KEY", ""),
		BaseCurrency:   getEnv("BASE_CURRENCY", "USD"),
		DatabaseURL:    getEnv("DATABASE_URL", "currency.db"),
		DatabaseDriver: getEnv("DATABASE_DRIVER", "postgres"),
	}
	return cfg
}
func getEnv(key string, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value

	}
	return defaultValue
}
