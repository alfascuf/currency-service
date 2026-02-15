package config

import "os"

type Config struct {
	Port                string
	CurrencyAPIURL      string // ← Оставьте для обратной совместимости
	CurrencyGRPCAddress string // ← НОВОЕ для gRPC
	AuthAPIURL          string
	JWTSecret           string
	Environment         string
}

func Load() *Config {
	return &Config{
		Port:                getEnv("PORT", "8080"),
		CurrencyAPIURL:      getEnv("CURRENCY_API_URL", "http://localhost:8082"), // Старое
		CurrencyGRPCAddress: getEnv("CURRENCY_GRPC_ADDRESS", "localhost:50051"),  // Новое
		AuthAPIURL:          getEnv("AUTH_API_URL", "http://localhost:8081"),
		JWTSecret:           getEnv("JWT_SECRET", "secret"),
		Environment:         getEnv("ENVIRONMENT", "development"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
