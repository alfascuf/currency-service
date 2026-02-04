package main

import (
	"context"
	"database/sql"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/alfascuf/currency-service/internal/config"
	"github.com/alfascuf/currency-service/internal/handler"
	"github.com/alfascuf/currency-service/internal/logger"
	"github.com/alfascuf/currency-service/internal/repository"
	"github.com/alfascuf/currency-service/internal/service"
	_ "github.com/lib/pq"
	"go.uber.org/zap"
)

const (
	// HTTP server timeouts
	serverReadTimeout     = 10 * time.Second
	serverWriteTimeout    = 10 * time.Second
	serverIdleTimeout     = 120 * time.Second
	serverShutdownTimeout = 30 * time.Second

	// Database connection pool settings
	dbMaxOpenConns    = 25
	dbMaxIdleConns    = 5
	dbConnMaxLifetime = 5 * time.Minute
)

func main() {
	// 1. Load configuration
	cfg := config.Load()

	// 2. Initialize logger
	if err := logger.Init(cfg.Environment); err != nil {
		panic("Failed to initialize logger: " + err.Error())
	}
	defer logger.Sync()

	logger.Log.Info("Starting Currency API Service",
		zap.String("port", cfg.Port),
		zap.String("environment", cfg.Environment),
		zap.String("base_currency", cfg.BaseCurrency),
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

	// 4. Initialize layers (repository, service, handler)
	repo := repository.New(db)
	logger.Log.Info("Database schema initialized")

	svc := service.New(repo)
	h := handler.New(svc)

	// 5. Setup HTTP server
	mux := http.NewServeMux()
	mux.HandleFunc("/health", h.Health)
	mux.HandleFunc("/api/v1/rates", h.GetRate)
	mux.HandleFunc("/api/v1/rates/history", h.GetHistory)

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      loggingMiddleware(mux),
		ReadTimeout:  serverReadTimeout,
		WriteTimeout: serverWriteTimeout,
		IdleTimeout:  serverIdleTimeout,
	}

	// 6. Start server in goroutine
	serverErrors := make(chan error, 1)
	go func() {
		logger.Log.Info("API server started",
			zap.String("address", srv.Addr),
			zap.Duration("read_timeout", serverReadTimeout),
			zap.Duration("write_timeout", serverWriteTimeout),
		)

		serverErrors <- srv.ListenAndServe()
	}()

	// 7. Wait for interrupt signal or server error
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		logger.Log.Fatal("Server startup error", zap.Error(err))
	case sig := <-quit:
		logger.Log.Info("Shutdown signal received", zap.String("signal", sig.String()))
	}

	// 8. Graceful shutdown
	logger.Log.Info("Shutting down server...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), serverShutdownTimeout)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Log.Error("Server forced to shutdown", zap.Error(err))
		_ = srv.Close()
		return
	}

	logger.Log.Info("Server stopped gracefully")
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Create ResponseWriter with status tracking
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(wrapped, r)

		logger.Log.Info("HTTP request",
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path),
			zap.String("query", r.URL.RawQuery),
			zap.String("remote_addr", r.RemoteAddr),
			zap.Int("status", wrapped.statusCode),
			zap.Duration("duration", time.Since(start)),
		)
	})
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

// WriteHeader captures the status code before writing it
func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
