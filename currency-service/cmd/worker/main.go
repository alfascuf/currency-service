package main

import (
	"context"
	"database/sql"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/alfascuf/currency-service/internal/config"
	"github.com/alfascuf/currency-service/internal/logger"
	"github.com/alfascuf/currency-service/internal/repository"
	"github.com/alfascuf/currency-service/internal/worker"
	_ "github.com/lib/pq"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

const (
	// Cron schedule: every day at 00:00 UTC
	dailySchedule = "0 0 * * *"

	// Database connection pool settings
	dbMaxOpenConns    = 10
	dbMaxIdleConns    = 5
	dbConnMaxLifetime = 5 * time.Minute

	// Shutdown timeout
	shutdownTimeout = 30 * time.Second
)

func main() {
	// 1. Load configuration
	cfg := config.Load()

	// 2. Initialize logger
	if err := logger.Init(cfg.Environment); err != nil {
		panic("Failed to initialize logger: " + err.Error())
	}
	defer logger.Sync()

	logger.Log.Info("Starting Currency Worker",
		zap.String("environment", cfg.Environment),
		zap.String("base_currency", cfg.BaseCurrency),
		zap.String("api_url", cfg.CurrencyAPIURL),
		zap.String("schedule", dailySchedule),
	)

	// 3. Connect to database with pool settings
	db, err := sql.Open(cfg.DatabaseDriver, cfg.DatabaseURL)
	if err != nil {
		logger.Log.Fatal("Failed to connect to database",
			zap.Error(err),
			zap.String("driver", cfg.DatabaseDriver),
		)
	}
	defer db.Close()

	// Configure connection pool
	db.SetMaxOpenConns(dbMaxOpenConns)
	db.SetMaxIdleConns(dbMaxIdleConns)
	db.SetConnMaxLifetime(dbConnMaxLifetime)

	// Verify database connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		logger.Log.Fatal("Failed to ping database", zap.Error(err))
	}

	logger.Log.Info("Connected to database successfully")

	// 4. Initialize repository and worker
	repo := repository.New(db)
	w := worker.New(cfg, repo)

	// 5. Setup cron scheduler
	c := cron.New()
	_, err = c.AddFunc(dailySchedule, func() {
		logger.Log.Info("Running scheduled currency update")
		if err := w.FetchAndSaveRates(); err != nil {
			logger.Log.Error("Scheduled update failed", zap.Error(err))
		} else {
			logger.Log.Info("Scheduled update completed successfully")
		}
	})
	if err != nil {
		logger.Log.Fatal("Failed to schedule cron job", zap.Error(err))
	}

	c.Start()
	logger.Log.Info("Cron scheduler started", zap.String("next_run", "00:00 UTC"))

	// 6. Run initial currency fetch
	logger.Log.Info("Running initial currency fetch")
	if err := w.FetchAndSaveRates(); err != nil {
		logger.Log.Error("Initial fetch failed", zap.Error(err))
	} else {
		logger.Log.Info("Initial fetch completed successfully")
	}

	// 7. Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	sig := <-quit
	logger.Log.Info("Shutdown signal received", zap.String("signal", sig.String()))

	// 8. Graceful shutdown
	logger.Log.Info("Shutting down worker...")

	// Stop accepting new cron jobs
	cronCtx := c.Stop()

	// Wait for running jobs with timeout
	select {
	case <-cronCtx.Done():
		logger.Log.Info("All cron jobs completed")
	case <-time.After(shutdownTimeout):
		logger.Log.Warn("Shutdown timeout reached, forcing stop")
	}

	logger.Log.Info("Worker stopped gracefully")
}
