package service

import (
	"fmt"
	"time"

	"github.com/alfascuf/currency-service/internal/models"
	"github.com/alfascuf/currency-service/internal/repository"
)

const (
	baseCurrency = "RUB"        // Base currency for cross-conversion
	dateFormat   = "2006-01-02" // Standard date format for parsing
)

// Error messages
const (
	errRateNotFound = "rate not found for %s/%s on %s"
	errNoHistory    = "no history for %s/%s"
	errZeroRate     = "invalid rate: cannot be zero for %s/%s"
)

// Service provides business logic for currency operations
type Service interface {
	GetRate(req *models.GetRateRequest) (*models.GetRateResponse, error)
	GetHistory(req *models.GetHistoryRequest) (*models.GetHistoryResponse, error)
	SaveRate(rate *models.ExchangeRate) error
}

type service struct {
	repo repository.Repository
}

// New creates a new Service instance
func New(repo repository.Repository) Service {
	return &service{repo: repo}
}

// GetRate calculates exchange rate for given currency pair and date.
// It handles three cases:
//  1. Same currency (returns 1.0)
//  2. Direct rate from repository
//  3. Cross-rate calculation through base currency
func (s *service) GetRate(req *models.GetRateRequest) (*models.GetRateResponse, error) {
	// Special case: same currency
	if req.Base == req.Target {
		return &models.GetRateResponse{
			Base:   req.Base,
			Target: req.Target,
			Rate:   1.0,
			Date:   req.Date,
		}, nil
	}

	// Parse date (format already validated in handler)
	date, err := time.Parse(dateFormat, req.Date)
	if err != nil {
		return nil, fmt.Errorf("invalid date format: %w", err)
	}

	// Calculate rate
	rate, err := s.calculateRate(req.Base, req.Target, date)
	if err != nil {
		return &models.GetRateResponse{
			Error: err.Error(),
		}, nil
	}

	return &models.GetRateResponse{
		Base:   req.Base,
		Target: req.Target,
		Rate:   rate,
		Date:   req.Date,
	}, nil
}

// calculateRate calculates exchange rate for any currency pair through base currency.
// It supports three conversion scenarios:
//  1. Direct conversion (RUB → USD): uses direct rate from repository
//  2. Reverse conversion (USD → RUB): inverts the direct rate
//  3. Cross conversion (USD → EUR): calculates through base currency (USD → RUB → EUR)
func (s *service) calculateRate(base, target string, date time.Time) (float64, error) {
	dateStr := date.Format(dateFormat)

	// Case 1: Direct rate (RUB → USD)
	if base == baseCurrency {
		rate, err := s.repo.GetRate(base, target, date)
		if err != nil {
			return 0, fmt.Errorf(errRateNotFound, base, target, dateStr)
		}
		return rate.Rate, nil
	}

	// Case 2: Reverse rate (USD → RUB)
	if target == baseCurrency {
		rate, err := s.repo.GetRate(baseCurrency, base, date)
		if err != nil {
			return 0, fmt.Errorf(errRateNotFound, base, target, dateStr)
		}

		// Check for zero rate to prevent division by zero
		if rate.Rate == 0 {
			return 0, fmt.Errorf(errZeroRate, baseCurrency, base)
		}

		// Invert: if 1 RUB = 0.0131 USD, then 1 USD = 1/0.0131 RUB
		return 1.0 / rate.Rate, nil
	}

	// Case 3: Cross-conversion (USD → EUR through RUB)
	// Formula: USD → EUR = (USD → RUB) × (RUB → EUR)

	// Get RUB → base (e.g., RUB → USD)
	baseRate, err := s.repo.GetRate(baseCurrency, base, date)
	if err != nil {
		return 0, fmt.Errorf(errRateNotFound, baseCurrency, base, dateStr)
	}

	// Get RUB → target (e.g., RUB → EUR)
	targetRate, err := s.repo.GetRate(baseCurrency, target, date)
	if err != nil {
		return 0, fmt.Errorf(errRateNotFound, baseCurrency, target, dateStr)
	}

	// Check for zero rate to prevent division by zero
	if baseRate.Rate == 0 {
		return 0, fmt.Errorf(errZeroRate, baseCurrency, base)
	}

	// Calculate cross-rate:
	// USD → RUB = 1 / (RUB → USD)
	// USD → EUR = (USD → RUB) × (RUB → EUR) = (1 / baseRate) × targetRate
	crossRate := targetRate.Rate / baseRate.Rate
	return crossRate, nil
}

// GetHistory returns historical exchange rates for given currency pair and date range.
// It supports the same three conversion scenarios as GetRate.
func (s *service) GetHistory(req *models.GetHistoryRequest) (*models.GetHistoryResponse, error) {
	// Parse dates (format already validated in handler)
	startDate, err := time.Parse(dateFormat, req.StartDate)
	if err != nil {
		return nil, fmt.Errorf("invalid start_date format: %w", err)
	}

	endDate, err := time.Parse(dateFormat, req.EndDate)
	if err != nil {
		return nil, fmt.Errorf("invalid end_date format: %w", err)
	}

	// Calculate history through base currency
	history, err := s.calculateHistory(req.Base, req.Target, startDate, endDate)
	if err != nil {
		return &models.GetHistoryResponse{
			Error: fmt.Sprintf("failed to get history: %v", err),
		}, nil
	}

	return &models.GetHistoryResponse{
		Base:   req.Base,
		Target: req.Target,
		Data:   history,
	}, nil
}

// calculateHistory calculates historical exchange rates for any currency pair.
// It handles the same three conversion scenarios as calculateRate:
//  1. Direct history (RUB → USD)
//  2. Reverse history (USD → RUB) with inverted rates
//  3. Cross history (USD → EUR) calculated through base currency
func (s *service) calculateHistory(base, target string, startDate, endDate time.Time) ([]models.ExchangeRate, error) {
	const rub = "RUB"

	// Case 1: Direct history (RUB → USD)
	if base == rub {
		return s.repo.GetHistory(base, target, startDate, endDate)
	}

	// Case 2: Reverse history (USD → RUB)
	if target == rub {
		history, err := s.repo.GetHistory(rub, base, startDate, endDate)
		if err != nil {
			return nil, err
		}

		// Invert each rate and swap base/target
		for i := range history {
			// Check for zero rate
			if history[i].Rate == 0 {
				continue // Skip zero rates to prevent division by zero
			}

			history[i].Base = base
			history[i].Target = target
			history[i].Rate = 1.0 / history[i].Rate
		}

		return history, nil
	}

	// Case 3: Cross-conversion history (USD → EUR through RUB)
	rubToBase, err := s.repo.GetHistory(rub, base, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf(errNoHistory, rub, base)
	}

	rubToTarget, err := s.repo.GetHistory(rub, target, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf(errNoHistory, rub, target)
	}

	// Create map for fast lookup by date
	targetMap := make(map[string]float64)
	for _, rate := range rubToTarget {
		dateKey := rate.Date.Format(dateFormat)
		targetMap[dateKey] = rate.Rate
	}

	// Calculate cross-rate for each date
	var result []models.ExchangeRate
	for _, baseRate := range rubToBase {
		dateKey := baseRate.Date.Format(dateFormat)
		targetRate, exists := targetMap[dateKey]
		if !exists {
			continue // Skip dates without target rate
		}

		// Check for zero rate to prevent division by zero
		if baseRate.Rate == 0 {
			continue
		}

		result = append(result, models.ExchangeRate{
			ID:        0, // ID not needed for history response
			Base:      base,
			Target:    target,
			Rate:      targetRate / baseRate.Rate,
			Date:      baseRate.Date,
			CreatedAt: baseRate.CreatedAt,
			UpdatedAt: baseRate.UpdatedAt,
		})
	}

	return result, nil
}

// SaveRate saves exchange rate to database.
// It delegates the actual saving to the repository layer.
func (s *service) SaveRate(rate *models.ExchangeRate) error {
	return s.repo.SaveRate(rate)
}
