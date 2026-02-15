package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/alfascuf/PROD1/gateway/internal/clients/currency"
	"github.com/alfascuf/PROD1/gateway/internal/config"
	"github.com/alfascuf/PROD1/gateway/internal/handler"
	"github.com/alfascuf/PROD1/gateway/internal/middleware"
	"github.com/alfascuf/PROD1/gateway/internal/repository"
	"github.com/alfascuf/PROD1/gateway/internal/service"
	"go.uber.org/zap"
)

const (
	// HTTP timeouts
	serverReadTimeout     = 10 * time.Second
	serverWriteTimeout    = 10 * time.Second
	serverIdleTimeout     = 120 * time.Second
	serverShutdownTimeout = 30 * time.Second
)

func main() {
	// 1. Load conf
	cfg := config.Load()

	// 2. Init logger
	var logger *zap.Logger
	var err error

	if cfg.Environment == "production" {
		logger, err = zap.NewProduction()
	} else {
		logger, err = zap.NewDevelopment()
	}

	if err != nil {
		panic("Failed to initialize logger: " + err.Error())
	}
	defer logger.Sync()

	logger.Info("Starting Gateway Service",
		zap.String("port", cfg.Port),
		zap.String("environment", cfg.Environment),
		zap.String("currency_grpc_address", cfg.CurrencyGRPCAddress), // ← Изменено
		zap.String("auth_api_url", cfg.AuthAPIURL),
	)

	// 3. Connect to Currency Service via gRPC
	currencyClient, err := currency.NewClient(cfg.CurrencyGRPCAddress)
	if err != nil {
		logger.Fatal("Failed to connect to currency service",
			zap.Error(err),
			zap.String("address", cfg.CurrencyGRPCAddress),
		)
	}
	defer currencyClient.Close()

	logger.Info("✓ Connected to Currency Service (gRPC)",
		zap.String("address", cfg.CurrencyGRPCAddress),
	)

	// 4. Init layers (repository, service, handler)
	userRepo := repository.NewUserRepository()
	logger.Info("User repository initialized with test users")

	authService := service.NewAuthService(cfg, userRepo)
	currencyService := service.NewCurrencyService(currencyClient) // ← Передаем gRPC клиент
	logger.Info("Services initialized")

	authHandler := handler.NewAuthHandler(authService, logger)
	currencyHandler := handler.NewCurrencyHandler(currencyService, logger)
	logger.Info("Handlers initialized")

	// 5. Set HTTP routes
	mux := http.NewServeMux()

	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("OK")); err != nil {
			logger.Error("Failed to write health response", zap.Error(err))
		}
	})

	// Auth endpoints (без авторизации)
	mux.HandleFunc("/auth/login", authHandler.Login)

	// Currency endpoints (с авторизацией через middleware)
	authMiddleware := middleware.AuthMiddleware(authService)
	mux.Handle("/api/rates", authMiddleware(http.HandlerFunc(currencyHandler.GetRate)))
	mux.Handle("/api/rates/history", authMiddleware(http.HandlerFunc(currencyHandler.GetHistory)))

	// 6. Set HTTP server
	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      loggingMiddleware(logger)(mux),
		ReadTimeout:  serverReadTimeout,
		WriteTimeout: serverWriteTimeout,
		IdleTimeout:  serverIdleTimeout,
	}

	// 7. Start server in go()
	serverErrors := make(chan error, 1)
	go func() {
		logger.Info("Gateway server started",
			zap.String("address", srv.Addr),
			zap.Duration("read_timeout", serverReadTimeout),
			zap.Duration("write_timeout", serverWriteTimeout),
		)

		serverErrors <- srv.ListenAndServe()
	}()

	// 8. Wait for signal stop or server err
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		logger.Fatal("Server startup error", zap.Error(err))
	case sig := <-quit:
		logger.Info("Shutdown signal received", zap.String("signal", sig.String()))
	}

	// 9. Graceful shutdown
	logger.Info("Shutting down server...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), serverShutdownTimeout)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("Server forced to shutdown", zap.Error(err))
		_ = srv.Close()
		return
	}

	logger.Info("Server stopped gracefully")
}

// loggingMiddleware логирует все HTTP запросы
func loggingMiddleware(logger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Оборачиваем ResponseWriter для захвата status code
			wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			next.ServeHTTP(wrapped, r)

			logger.Info("HTTP request",
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
				zap.String("query", r.URL.RawQuery),
				zap.String("remote_addr", r.RemoteAddr),
				zap.Int("status", wrapped.statusCode),
				zap.Duration("duration", time.Since(start)),
			)
		})
	}
}

// responseWriter обёртка для захвата status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

// WriteHeader перехватывает status code
func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
