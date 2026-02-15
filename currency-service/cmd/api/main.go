package main

// @title           Currency Exchange API
// @version         1.0
// @description     API для получения курсов валют с поддержкой gRPC и HTTP
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.email  support@currency-api.com

// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html

// @host      localhost:8082
// @BasePath  /api/v1

// @schemes http https

import (
	"context"
	"database/sql"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/alfascuf/PROD1/currency-service/internal/cache"
	"github.com/alfascuf/PROD1/currency-service/internal/config"
	"github.com/alfascuf/PROD1/currency-service/internal/handler"
	"github.com/alfascuf/PROD1/currency-service/internal/logger"
	"github.com/alfascuf/PROD1/currency-service/internal/repository"
	"github.com/alfascuf/PROD1/currency-service/internal/service"
	pb "github.com/alfascuf/PROD1/pkg/grpc/pb"
	_ "github.com/lib/pq"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	_ "github.com/alfascuf/PROD1/currency-service/docs"
	httpSwagger "github.com/swaggo/http-swagger"
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
	dbPingTimeout     = 5 * time.Second
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
		zap.String("http_port", cfg.Port),
		zap.String("grpc_port", cfg.GRPCPort), // Добавьте в config
		zap.String("environment", cfg.Environment),
		zap.String("base_currency", cfg.BaseCurrency),
	)

	// 3. Connect to database
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
	ctx, cancel := context.WithTimeout(context.Background(), dbPingTimeout)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		logger.Log.Fatal("Failed to ping database", zap.Error(err))
	}

	logger.Log.Info("Connected to database successfully")

	// 4. Initialize Redis cache
	var redisCache cache.Cache
	redisCache, err = cache.New(cfg)
	if err != nil {
		logger.Log.Warn("Failed to connect to Redis, continuing without cache",
			zap.Error(err),
			zap.String("redis_host", cfg.RedisHost),
			zap.String("redis_port", cfg.RedisPort),
		)
		redisCache = nil
	} else {
		logger.Log.Info("Redis cache initialized successfully",
			zap.String("redis_host", cfg.RedisHost),
			zap.String("redis_port", cfg.RedisPort),
			zap.Int("cache_ttl_seconds", cfg.CacheTTL),
		)
		defer func() {
			if err := redisCache.Close(); err != nil {
				logger.Log.Error("Failed to close Redis connection", zap.Error(err))
			}
		}()
	}

	// 5. Initialize layers
	repo := repository.New(db)
	logger.Log.Info("Repository initialized")

	svc := service.New(repo, redisCache)
	logger.Log.Info("Service initialized", zap.Bool("cache_enabled", redisCache != nil))

	// 6. Setup HTTP server
	httpHandler := handler.New(svc)
	mux := http.NewServeMux()
	mux.HandleFunc("/health", httpHandler.Health)
	mux.HandleFunc("/api/v1/rates", httpHandler.GetRate)
	mux.HandleFunc("/api/v1/rates/history", httpHandler.GetHistory)
	mux.HandleFunc("/swagger/", httpSwagger.Handler(
		httpSwagger.URL("http://localhost:"+cfg.Port+"/swagger/doc.json"),
	))

	httpServer := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      loggingMiddleware(mux),
		ReadTimeout:  serverReadTimeout,
		WriteTimeout: serverWriteTimeout,
		IdleTimeout:  serverIdleTimeout,
	}

	// 7. Setup gRPC server
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(grpcLoggingInterceptor),
	)
	grpcHandler := handler.NewGrpcHandler(svc)
	pb.RegisterCurrencyServiceServer(grpcServer, grpcHandler)

	// Enable gRPC reflection for debugging (удалите в production)
	reflection.Register(grpcServer)

	grpcListener, err := net.Listen("tcp", ":"+cfg.GRPCPort)
	if err != nil {
		logger.Log.Fatal("Failed to create gRPC listener",
			zap.Error(err),
			zap.String("port", cfg.GRPCPort),
		)
	}

	// 8. Start servers in goroutines
	serverErrors := make(chan error, 2)

	// Start HTTP server
	go func() {
		logger.Log.Info("HTTP server started",
			zap.String("address", httpServer.Addr),
			zap.Duration("read_timeout", serverReadTimeout),
			zap.Duration("write_timeout", serverWriteTimeout),
		)
		serverErrors <- httpServer.ListenAndServe()
	}()

	// Start gRPC server
	go func() {
		logger.Log.Info("gRPC server started",
			zap.String("address", grpcListener.Addr().String()),
		)
		serverErrors <- grpcServer.Serve(grpcListener)
	}()

	// 9. Wait for interrupt signal or server error
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		logger.Log.Fatal("Server startup error", zap.Error(err))
	case sig := <-quit:
		logger.Log.Info("Shutdown signal received", zap.String("signal", sig.String()))
	}

	// 10. Graceful shutdown
	logger.Log.Info("Shutting down servers...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), serverShutdownTimeout)
	defer shutdownCancel()

	// Shutdown HTTP server
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Log.Error("HTTP server forced shutdown", zap.Error(err))
		_ = httpServer.Close()
	} else {
		logger.Log.Info("HTTP server stopped gracefully")
	}

	// Shutdown gRPC server
	stopped := make(chan struct{})
	go func() {
		grpcServer.GracefulStop()
		close(stopped)
	}()

	select {
	case <-shutdownCtx.Done():
		logger.Log.Warn("gRPC graceful shutdown timeout, forcing stop")
		grpcServer.Stop()
	case <-stopped:
		logger.Log.Info("gRPC server stopped gracefully")
	}

	logger.Log.Info("All servers stopped")
}

// HTTP logging middleware
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
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

// gRPC logging interceptor
func grpcLoggingInterceptor(
	ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (interface{}, error) {
	start := time.Now()

	resp, err := handler(ctx, req)

	logger.Log.Info("gRPC request",
		zap.String("method", info.FullMethod),
		zap.Duration("duration", time.Since(start)),
		zap.Bool("error", err != nil),
	)

	return resp, err
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
