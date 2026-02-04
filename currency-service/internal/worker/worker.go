package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/alfascuf/currency-service/internal/config"
	"github.com/alfascuf/currency-service/internal/logger"
	"github.com/alfascuf/currency-service/internal/models"
	"github.com/alfascuf/currency-service/internal/repository"
	"go.uber.org/zap"
)

const (
	// HTTP client timeouts
	httpTimeout        = 30 * time.Second
	httpRetries        = 3
	retryDelay         = 2 * time.Second
	maxIdleConnections = 10
)

type Worker struct {
	config     *config.Config
	repo       repository.Repository
	httpClient *http.Client
}

func New(cfg *config.Config, repo repository.Repository) *Worker {
	// Configure HTTP client with timeouts and keep-alive
	client := &http.Client{
		Timeout: httpTimeout,
		Transport: &http.Transport{
			MaxIdleConns:        maxIdleConnections,
			MaxIdleConnsPerHost: maxIdleConnections,
			IdleConnTimeout:     90 * time.Second,
			DisableKeepAlives:   false,
		},
	}

	return &Worker{
		config:     cfg,
		repo:       repo,
		httpClient: client,
	}
}

// FetchAndSaveRates fetches currency rates with retry mechanism
func (w *Worker) FetchAndSaveRates() error {
	return w.FetchAndSaveRatesWithContext(context.Background())
}

// FetchAndSaveRatesWithContext fetches currency rates with context support
func (w *Worker) FetchAndSaveRatesWithContext(ctx context.Context) error {
	var lastErr error

	// Retry mechanism
	for attempt := 1; attempt <= httpRetries; attempt++ {
		// Check if context is cancelled
		select {
		case <-ctx.Done():
			return fmt.Errorf("operation cancelled: %w", ctx.Err())
		default:
		}

		logger.Log.Info("Fetching currency rates",
			zap.Int("attempt", attempt),
			zap.Int("max_attempts", httpRetries),
		)

		err := w.fetchAndSave(ctx)
		if err == nil {
			return nil // Success!
		}

		lastErr = err
		logger.Log.Warn("Failed to fetch rates",
			zap.Error(err),
			zap.Int("attempt", attempt),
			zap.Int("max_attempts", httpRetries),
		)

		// Don't wait after last attempt
		if attempt < httpRetries {
			logger.Log.Info("Retrying after delay",
				zap.Duration("delay", retryDelay),
			)

			select {
			case <-time.After(retryDelay):
			case <-ctx.Done():
				return fmt.Errorf("operation cancelled during retry: %w", ctx.Err())
			}
		}
	}

	return fmt.Errorf("failed after %d attempts: %w", httpRetries, lastErr)
}

// fetchAndSave performs the actual API call and database save
func (w *Worker) fetchAndSave(ctx context.Context) error {
	// Create request context with timeout
	reqCtx, cancel := context.WithTimeout(ctx, httpTimeout)
	defer cancel()

	url := fmt.Sprintf("%s/latest/%s", w.config.CurrencyAPIURL, w.config.BaseCurrency)

	logger.Log.Info("Requesting currency API",
		zap.String("url", url),
		zap.String("base_currency", w.config.BaseCurrency),
		zap.Duration("timeout", httpTimeout),
	)

	// Create HTTP request with context
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Add User-Agent header
	req.Header.Set("User-Agent", "Currency-Service/1.0")

	// Execute request
	resp, err := w.httpClient.Do(req)
	if err != nil {
		logger.Log.Error("Failed to fetch rates from API",
			zap.Error(err),
			zap.String("url", url),
		)
		return fmt.Errorf("failed to fetch rates: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		logger.Log.Error("API returned non-200 status",
			zap.Int("status_code", resp.StatusCode),
			zap.String("url", url),
		)
		return fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	// Read response with size limit
	body, err := io.ReadAll(io.LimitReader(resp.Body, 10*1024*1024)) // 10MB limit
	if err != nil {
		logger.Log.Error("Failed to read response body", zap.Error(err))
		return fmt.Errorf("failed to read response: %w", err)
	}

	// Parse JSON
	var result struct {
		Rates map[string]float64 `json:"rates"`
		Date  string             `json:"date"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		logger.Log.Error("Failed to parse JSON response",
			zap.Error(err),
			zap.Int("body_size", len(body)),
		)
		return fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Validate response
	if len(result.Rates) == 0 {
		return fmt.Errorf("no rates returned from API")
	}

	// Parse date
	date, err := time.Parse("2006-01-02", result.Date)
	if err != nil {
		date = time.Now().UTC()
	}

	// Save rates to database
	count := 0
	errorCount := 0

	for target, rate := range result.Rates {
		// Check context before each save
		select {
		case <-ctx.Done():
			return fmt.Errorf("operation cancelled during save: %w", ctx.Err())
		default:
		}

		err := w.repo.SaveRate(&models.ExchangeRate{
			Base:   w.config.BaseCurrency,
			Target: target,
			Rate:   rate,
			Date:   date,
		})

		if err != nil {
			logger.Log.Warn("Failed to save rate",
				zap.Error(err),
				zap.String("base", w.config.BaseCurrency),
				zap.String("target", target),
				zap.Float64("rate", rate),
			)
			errorCount++
			continue
		}

		count++
	}

	logger.Log.Info("Currency rates update completed",
		zap.Int("saved", count),
		zap.Int("errors", errorCount),
		zap.String("date", result.Date),
		zap.String("base_currency", w.config.BaseCurrency),
	)

	if errorCount > 0 && count == 0 {
		return fmt.Errorf("failed to save any rates: %d errors", errorCount)
	}

	return nil
}

// Close closes HTTP client connections
func (w *Worker) Close() {
	if w.httpClient != nil {
		w.httpClient.CloseIdleConnections()
	}
}
