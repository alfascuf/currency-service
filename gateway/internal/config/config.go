package config

import (
	"log"
	"os"
)

// Config has configuration of Gateway Service
type Config struct {
	Port           string // Port gateway
	CurrencyAPIURL string // URL of Currency
	AuthAPIURL     string // URL Auth service
	JWTSecret      string // secret key for JWT check
	Environment    string // prod or dev
}

// Load start configuration from .env
func Load() *Config {
	cfg := &Config{
		Port:           getEnv("PORT", "8080"),
		CurrencyAPIURL: getEnv("CURRENCY_API_URL", "https://localhost:8082"),
		AuthAPIURL:     getEnv("AUTH_API_URL", "https://localhost:8081"),
		JWTSecret:      getEnv("JWT_SECRET", "secret-key-change-in-production"),
		Environment:    getEnv("ENVIRONMENT", "development"),
	}

	if cfg.JWTSecret == "secret-key-change-in-production" && cfg.Environment == "production" {
		log.Fatal("JWT_SECRET must be set to production")
	}
	return cfg
}

// getEnv get .env or def
func getEnv(key string, defaultVal string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultVal
	}
	return value
}
