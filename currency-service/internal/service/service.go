package service

import (
	"fmt"
	"time"

	"github.com/alfascuf/currency-service/internal/models"
	"github.com/alfascuf/currency-service/internal/repository"
)

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

	// Парсим дату
	date, err := time.Parse("2006-01-02", req.Date)
	if err != nil {
		return &models.GetRateResponse{
			Error: fmt.Sprintf("invalid date format: %v. Use YYYY-MM-DD", err),
		}, nil
	}

	// Получаем курс из БД
	rate, err := s.repo.GetRate(req.Base, req.Target, date)
	if err != nil {
		return &models.GetRateResponse{
			Error: fmt.Sprintf("rate not found: %v", err),
		}, nil
	}

	return &models.GetRateResponse{
		Base:   rate.Base,
		Target: rate.Target,
		Rate:   rate.Rate,
		Date:   rate.Date.Format("2006-01-02"),
	}, nil
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

	history, err := s.repo.GetHistory(req.Base, req.Target, startDate, endDate)
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

func (s *service) SaveRate(rate *models.ExchangeRate) error {
	return s.repo.SaveRate(rate)
}
