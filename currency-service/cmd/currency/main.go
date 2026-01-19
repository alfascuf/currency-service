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

	// Настройка роутов
	mux := http.NewServeMux()
	mux.HandleFunc("/health", handler.Health)
	mux.HandleFunc("/api/v1/rates", handler.GetRate)

	// Создание HTTP сервера
	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	log.Printf("Currency service started on port %s\n", cfg.Port)
	log.Fatal(srv.ListenAndServe())
}
