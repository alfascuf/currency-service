package service

import (
	"fmt"
	"time"

	"github.com/alfascuf/currency-service/internal/models"
	"github.com/alfascuf/currency-service/internal/repository"
)

const baseCurrency = "RUB" // Базовая валюта для кросс-конвертации

type Service interface {
	GetRate(req *models.GetRateRequest) (*models.GetRateResponse, error)
	GetHistory(req *models.GetHistoryRequest) (*models.GetHistoryResponse, error)
	SaveRate(rate *models.ExchangeRate) error
}

type service struct {
	repo repository.Repository
}

func New(repo repository.Repository) Service {
	return &service{repo: repo}
}

func (s *service) GetRate(req *models.GetRateRequest) (*models.GetRateResponse, error) {
	if req.Base == "" || req.Target == "" {
		return &models.GetRateResponse{
			Error: "base and target currencies are required",
		}, nil
	}

	// Если base == target, курс = 1
	if req.Base == req.Target {
		return &models.GetRateResponse{
			Base:   req.Base,
			Target: req.Target,
			Rate:   1.0,
			Date:   req.Date,
		}, nil
	}

	// Парсим дату
	date, err := time.Parse("2006-01-02", req.Date)
	if err != nil {
		return &models.GetRateResponse{
			Error: fmt.Sprintf("invalid date format: %v. Use YYYY-MM-DD", err),
		}, nil
	}

	// Вычисляем курс
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

// calculateRate - универсальная конвертация любой пары через RUB
func (s *service) calculateRate(base, target string, date time.Time) (float64, error) {
	// Случай 1: Прямой курс (RUB → USD)
	if base == baseCurrency {
		rate, err := s.repo.GetRate(base, target, date)
		if err != nil {
			return 0, fmt.Errorf("rate not found for %s/%s on %s", base, target, date.Format("2006-01-02"))
		}
		return rate.Rate, nil
	}

	// Случай 2: Обратный курс (USD → RUB)
	if target == baseCurrency {
		rate, err := s.repo.GetRate(baseCurrency, base, date)
		if err != nil {
			return 0, fmt.Errorf("rate not found for %s/%s on %s", base, target, date.Format("2006-01-02"))
		}
		// Инверсия: если 1 RUB = 0.0131 USD, то 1 USD = 1/0.0131 RUB
		return 1.0 / rate.Rate, nil
	}

	// Случай 3: Кросс-конвертация (USD → EUR через RUB)
	// USD → EUR = (USD → RUB) × (RUB → EUR)

	// Получаем RUB → base (например, RUB → USD)
	baseRate, err := s.repo.GetRate(baseCurrency, base, date)
	if err != nil {
		return 0, fmt.Errorf("rate not found for %s/%s on %s", baseCurrency, base, date.Format("2006-01-02"))
	}

	// Получаем RUB → target (например, RUB → EUR)
	targetRate, err := s.repo.GetRate(baseCurrency, target, date)
	if err != nil {
		return 0, fmt.Errorf("rate not found for %s/%s on %s", baseCurrency, target, date.Format("2006-01-02"))
	}

	// Вычисляем кросс-курс:
	// USD → RUB = 1 / (RUB → USD)
	// USD → EUR = (USD → RUB) × (RUB → EUR) = (1 / baseRate) × targetRate
	crossRate := targetRate.Rate / baseRate.Rate

	return crossRate, nil
}

func (s *service) GetHistory(req *models.GetHistoryRequest) (*models.GetHistoryResponse, error) {
	if req.Base == "" || req.Target == "" {
		return &models.GetHistoryResponse{
			Error: "base and target currencies are required",
		}, nil
	}

	startDate, err := time.Parse("2006-01-02", req.StartDate)
	if err != nil {
		return &models.GetHistoryResponse{
			Error: fmt.Sprintf("invalid start_date format: %v", err),
		}, nil
	}

	endDate, err := time.Parse("2006-01-02", req.EndDate)
	if err != nil {
		return &models.GetHistoryResponse{
			Error: fmt.Sprintf("invalid end_date format: %v", err),
		}, nil
	}

	// Универсальная история через RUB
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

// НОВЫЙ метод для кросс-истории
func (s *service) calculateHistory(base, target string, startDate, endDate time.Time) ([]models.ExchangeRate, error) {
	const rub = "RUB"

	// Случай 1: Прямой курс (RUB → USD)
	if base == rub {
		return s.repo.GetHistory(base, target, startDate, endDate)
	}

	// Случай 2: Обратный курс (USD → RUB)
	if target == rub {
		history, err := s.repo.GetHistory(rub, base, startDate, endDate)
		if err != nil {
			return nil, err
		}
		// Инвертируем каждый курс
		for i := range history {
			history[i].Base = base
			history[i].Target = target
			history[i].Rate = 1.0 / history[i].Rate
		}
		return history, nil
	}

	// Случай 3: Кросс-конвертация (USD → EUR через RUB)
	rubToBase, err := s.repo.GetHistory(rub, base, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("no history for %s/%s", rub, base)
	}

	rubToTarget, err := s.repo.GetHistory(rub, target, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("no history for %s/%s", rub, target)
	}

	// Создаём map для быстрого поиска по дате
	targetMap := make(map[string]float64)
	for _, rate := range rubToTarget {
		dateKey := rate.Date.Format("2006-01-02")
		targetMap[dateKey] = rate.Rate
	}

	// Вычисляем кросс-курс для каждой даты
	var result []models.ExchangeRate
	for _, baseRate := range rubToBase {
		dateKey := baseRate.Date.Format("2006-01-02")
		targetRate, exists := targetMap[dateKey]
		if !exists {
			continue
		}

		result = append(result, models.ExchangeRate{
			ID:        0, // ID не нужен для истории
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

func (s *service) SaveRate(rate *models.ExchangeRate) error {
	return s.repo.SaveRate(rate)
}
