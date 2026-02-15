package service

import (
	"context"
	"fmt"
	"time"

	"github.com/alfascuf/PROD1/gateway/internal/clients/currency"
	"github.com/alfascuf/PROD1/gateway/internal/models"
)

// CurrencyService interface for currency working
type CurrencyService interface {
	GetRate(req *models.CurrencyRateRequest) (*models.CurrencyRateResponse, error)
	GetHistory(req *models.HistoryRequest) (*models.HistoryResponse, error)
}

type currencyService struct {
	grpcClient *currency.Client
}

// NewCurrencyService создаёт create service to work with Currency API via gRPC
func NewCurrencyService(grpcClient *currency.Client) CurrencyService {
	return &currencyService{
		grpcClient: grpcClient,
	}
}

// GetRate get currency from Currency Service via gRPC
func (s *currencyService) GetRate(req *models.CurrencyRateRequest) (*models.CurrencyRateResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Вызов gRPC метода
	resp, err := s.grpcClient.GetRate(ctx, req.Base, req.Target, req.Date)
	if err != nil {
		return nil, fmt.Errorf("grpc call failed: %w", err)
	}

	// Если сервис вернул бизнес-ошибку
	if resp.Error != "" {
		return &models.CurrencyRateResponse{
			Error: resp.Error,
		}, nil
	}

	// Успешный ответ
	return &models.CurrencyRateResponse{
		Base:   resp.Base,
		Target: resp.Target,
		Rate:   resp.Rate,
		Date:   resp.Date,
	}, nil
}

// GetHistory get history of currencies from Currency Service via gRPC
func (s *currencyService) GetHistory(req *models.HistoryRequest) (*models.HistoryResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Вызов gRPC метода
	resp, err := s.grpcClient.GetHistory(ctx, req.Base, req.Target, req.StartDate, req.EndDate)
	if err != nil {
		return nil, fmt.Errorf("grpc call failed: %w", err)
	}

	// Если сервис вернул бизнес-ошибку
	if resp.Error != "" {
		return &models.HistoryResponse{
			Error: resp.Error,
		}, nil
	}

	// Конвертируем protobuf в модели Gateway (только 4 поля)
	rates := make([]models.ExchangeRate, len(resp.Data))
	for i, rate := range resp.Data {
		rates[i] = models.ExchangeRate{
			Base:   rate.Base,
			Target: rate.Target,
			Rate:   rate.Rate,
			Date:   rate.Date.AsTime().Format("2006-01-02"),
		}
	}

	return &models.HistoryResponse{
		Base:   resp.Base,
		Target: resp.Target,
		Data:   rates,
	}, nil
}
