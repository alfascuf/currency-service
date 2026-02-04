package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/alfascuf/currency-service/internal/config"
	"github.com/alfascuf/currency-service/internal/migrations"
	_ "github.com/lib/pq"
)

const (
	maxRetries = 10
	retryDelay = 2 * time.Second
)

func main() {
	// Parse command-line flags
	command := flag.String("cmd", "up", "Migration command: up, down, or status")
	flag.Parse()

	fmt.Printf("Running migrations: command=%s\n", *command)

	// Load configuration
	cfg := config.Load()
	fmt.Printf("Database: %s\n", cfg.DatabaseURL)

	// Connect to database with retry logic
	db, err := connectWithRetry(cfg.DatabaseDriver, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database after %d attempts: %v", maxRetries, err)
	}
	defer db.Close()

	fmt.Println("Connected to database successfully")

	// Execute migration command
	switch *command {
	case "up":
		if err := migrations.Run(db); err != nil {
			log.Fatalf("Failed to run migrations: %v", err)
		}
		fmt.Println("Migrations applied successfully")
	default:
		log.Fatalf("Unknown command: %s (supported: up)", *command)
	}
}

func connectWithRetry(driver, dsn string) (*sql.DB, error) {
	var db *sql.DB
	var err error

	for i := 1; i <= maxRetries; i++ {
		fmt.Printf("Attempting to connect to database (attempt %d/%d)...\n", i, maxRetries)

		db, err = sql.Open(driver, dsn)
		if err != nil {
			if i < maxRetries {
				fmt.Printf("Failed to open database: %v\nRetrying in %v...\n", err, retryDelay)
				time.Sleep(retryDelay)
				continue
			}
			return nil, fmt.Errorf("failed after %d attempts: %w", maxRetries, err)
		}

		// Test connection
		if err = db.Ping(); err != nil {
			if i < maxRetries {
				fmt.Printf("Failed to ping database: %v\nRetrying in %v...\n", err, retryDelay)
				time.Sleep(retryDelay)
				continue
			}
			return nil, fmt.Errorf("failed to ping database: %w", err)
		}

		fmt.Println("Database connection verified")
		return db, nil
	}

	return nil, fmt.Errorf("failed after %d attempts", maxRetries)
}
