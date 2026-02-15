package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/alfascuf/PROD1/currency-service/internal/config"
	"github.com/alfascuf/PROD1/currency-service/internal/logger"
	"github.com/alfascuf/PROD1/currency-service/internal/migrations"
	_ "github.com/lib/pq"
	"go.uber.org/zap"
)

const (
	maxRetries    = 10
	retryDelay    = 2 * time.Second
	dbPingTimeout = 5 * time.Second
)

func main() {
	// Parse command-line flags
	command := flag.String("cmd", "up", "Migration command: up")
	flag.Parse()

	// Load configuration
	cfg := config.Load()

	// Initialize logger
	if err := logger.Init(cfg.Environment); err != nil {
		panic(fmt.Sprintf("Failed to initialize logger: %v", err))
	}
	defer logger.Sync()

	logger.Log.Info("Starting database migration",
		zap.String("command", *command),
		zap.String("database_driver", cfg.DatabaseDriver),
	)

	// Validate command
	if *command != "up" {
		logger.Log.Fatal("Unsupported migration command",
			zap.String("command", *command),
			zap.String("supported", "up"),
		)
	}

	// Setup context for cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-quit
		logger.Log.Warn("Shutdown signal received, canceling migration...",
			zap.String("signal", sig.String()),
		)
		cancel()
	}()

	// Connect to database with retry logic
	db, err := connectWithRetry(ctx, cfg.DatabaseDriver, cfg.DatabaseURL)
	if err != nil {
		logger.Log.Fatal("Failed to connect to database",
			zap.Error(err),
			zap.Int("max_retries", maxRetries),
		)
	}
	defer db.Close()

	logger.Log.Info("Connected to database successfully")

	// Execute migration
	if err := migrations.Run(db); err != nil {
		logger.Log.Fatal("Failed to run migrations", zap.Error(err))
	}

	logger.Log.Info("Migrations applied successfully")
}

// connectWithRetry attempts to connect to the database with retry logic
func connectWithRetry(ctx context.Context, driver, dsn string) (*sql.DB, error) {
	var db *sql.DB
	var err error

	for i := 1; i <= maxRetries; i++ {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("connection canceled: %w", ctx.Err())
		default:
		}

		logger.Log.Info("Attempting to connect to database",
			zap.Int("attempt", i),
			zap.Int("max_retries", maxRetries),
		)

		db, err = sql.Open(driver, dsn)
		if err != nil {
			if db != nil {
				db.Close()
			}
			if i < maxRetries {
				logger.Log.Warn("Failed to open database, retrying...",
					zap.Error(err),
					zap.Duration("retry_in", retryDelay),
				)
				time.Sleep(retryDelay)
				continue
			}
			return nil, fmt.Errorf("failed to open database after %d attempts: %w", maxRetries, err)
		}

		// Test connection with timeout
		pingCtx, pingCancel := context.WithTimeout(ctx, dbPingTimeout)
		err = db.PingContext(pingCtx)
		pingCancel()

		if err != nil {
			db.Close()
			if i < maxRetries {
				logger.Log.Warn("Failed to ping database, retrying...",
					zap.Error(err),
					zap.Duration("retry_in", retryDelay),
				)
				time.Sleep(retryDelay)
				continue
			}
			return nil, fmt.Errorf("failed to ping database after %d attempts: %w", maxRetries, err)
		}

		logger.Log.Info("Database connection verified")
		return db, nil
	}

	return nil, fmt.Errorf("exhausted all %d retry attempts", maxRetries)
}
