package main

import (
	"context"
	"database/sql"
	"fmt"
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
	dailySchedule     = "0 0 * * *"
	dbMaxOpenConns    = 10
	dbMaxIdleConns    = 5
	dbConnMaxLifetime = 5 * time.Minute
	dbPingTimeout     = 5 * time.Second
	shutdownTimeout   = 30 * time.Second
	maxRetries        = 10
	retryDelay        = 2 * time.Second
)

func main() {
	// 1. Load configuration
	cfg := config.Load()

	// 2. Initialize logger
	if err := logger.Init(cfg.Environment); err != nil {
		panic(fmt.Sprintf("Failed to initialize logger: %v", err))
	}
	defer logger.Sync()

	logger.Log.Info("Starting Currency Worker",
		zap.String("environment", cfg.Environment),
		zap.String("base_currency", cfg.BaseCurrency),
		zap.String("api_url", cfg.CurrencyAPIURL),
		zap.String("schedule", dailySchedule),
	)

	// 3. Validate configuration
	if err := validateConfig(cfg); err != nil {
		logger.Log.Fatal("Invalid configuration", zap.Error(err))
	}

	// 4. Connect to database with retry logic
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	db, err := connectWithRetry(ctx, cfg.DatabaseDriver, cfg.DatabaseURL)
	if err != nil {
		logger.Log.Fatal("Failed to connect to database",
			zap.Error(err),
			zap.Int("max_retries", maxRetries),
		)
	}
	defer db.Close()

	logger.Log.Info("Connected to database successfully")

	// 5. Initialize repository and worker
	repo := repository.New(db)
	w := worker.New(cfg, repo)

	// 6. Setup cron scheduler
	c := cron.New()
	_, err = c.AddFunc(dailySchedule, func() {
		start := time.Now()
		logger.Log.Info("Running scheduled currency update")

		if err := w.FetchAndSaveRates(); err != nil {
			logger.Log.Error("Scheduled update failed",
				zap.Error(err),
				zap.Duration("duration", time.Since(start)),
			)
		} else {
			logger.Log.Info("Scheduled update completed successfully",
				zap.Duration("duration", time.Since(start)),
			)
		}
	})
	if err != nil {
		logger.Log.Fatal("Failed to schedule cron job", zap.Error(err))
	}

	c.Start()
	logger.Log.Info("Cron scheduler started",
		zap.String("schedule", dailySchedule),
		zap.String("next_run", "00:00 UTC daily"),
	)

	// 7. Run initial currency fetch
	logger.Log.Info("Running initial currency fetch")
	if err := w.FetchAndSaveRates(); err != nil {
		logger.Log.Error("Initial fetch failed", zap.Error(err))
	} else {
		logger.Log.Info("Initial fetch completed successfully")
	}

	// 8. Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	sig := <-quit
	logger.Log.Info("Shutdown signal received", zap.String("signal", sig.String()))

	// 9. Graceful shutdown
	logger.Log.Info("Shutting down worker...")

	cronCtx := c.Stop()

	select {
	case <-cronCtx.Done():
		logger.Log.Info("All cron jobs completed")
	case <-time.After(shutdownTimeout):
		logger.Log.Warn("Shutdown timeout reached, forcing stop")
	}

	logger.Log.Info("Worker stopped gracefully")
}

func validateConfig(cfg *config.Config) error {
	if cfg.ExchangeApiKey == "" { // ← с маленькой "a" (как в вашем config.go)
		return fmt.Errorf("missing EXCHANGE_API_KEY")
	}
	if cfg.CurrencyAPIURL == "" {
		return fmt.Errorf("missing CURRENCY_API_URL")
	}
	if cfg.BaseCurrency == "" {
		return fmt.Errorf("missing BASE_CURRENCY")
	}
	if cfg.DatabaseURL == "" {
		return fmt.Errorf("missing DATABASE_URL")
	}
	return nil
}

func connectWithRetry(ctx context.Context, driver, dsn string) (*sql.DB, error) {
	var db *sql.DB
	var err error

	for i := 1; i <= maxRetries; i++ {
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

		db.SetMaxOpenConns(dbMaxOpenConns)
		db.SetMaxIdleConns(dbMaxIdleConns)
		db.SetConnMaxLifetime(dbConnMaxLifetime)

		pingCtx, pingCancel := context.WithTimeout(ctx, dbPingTimeout)
		err = db.PingContext(pingCtx)
		pingCancel()

		if err != nil {
			db.Close()
			if i < maxRetries {
				logger.Log.Warn("Failed to ping database, retrying...",
					zap.Error(err),
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
