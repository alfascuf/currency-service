package main

import (
	"database/sql"
	"log"
	"net/http"
	"time"

	_ "github.com/lib/pq"

	"github.com/alfascuf/currency-service/internal/config"
	"github.com/alfascuf/currency-service/internal/handler"
	"github.com/alfascuf/currency-service/internal/repository"
	"github.com/alfascuf/currency-service/internal/service"
)

func main() {
	// Загрузка конфигурации
	cfg := config.Load()
	log.Printf("Starting Currency Service with config: Port=%s, BaseCurrency=%s, DB=%s\n",
		cfg.Port, cfg.BaseCurrency, cfg.DatabaseDriver)

	// Подключение к PostgreSQL
	db, err := sql.Open(cfg.DatabaseDriver, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Проверка соединения
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}
	log.Println("Database connection established successfully")

	// Создание repository
	repo := repository.New(db)
	if err := repo.InitDB(); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	log.Println("Database tables created successfully")

	svc := service.New(repo)
	h := handler.New(svc)

	// Настройка роутов
	mux := http.NewServeMux()
	mux.HandleFunc("/health", h.Health)
	mux.HandleFunc("/api/v1/rates", h.GetRate)
	mux.HandleFunc("/api/v1/rates/history", h.GetHistory)

	// Создание HTTP сервера
	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	log.Printf("Currency service started on port %s\n", cfg.Port)
	log.Printf("📋 Available endpoints:")
	log.Printf("   GET  /health")
	log.Printf("   GET  /api/v1/rates?base=RUB&target=USD&date=2026-01-20")
	log.Printf("   GET  /api/v1/rates/history?base=RUB&target=USD&start_date=2026-01-01&end_date=2026-01-31")
	log.Fatal(srv.ListenAndServe())
}
