package service

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/alfascuf/gateway/internal/config"
	"github.com/alfascuf/gateway/internal/models"
)

// CurrencyService interface for currency working
type CurrencyService interface {
	GetRate(req *models.CurrencyRateRequest) (*models.CurrencyRateResponse, error)
	GetHistory(req *models.HistoryRequest) (*models.HistoryResponse, error)
}

type currencyService struct {
	currencyAPIURL string
	httpClient     *http.Client
}

// NewCurrencyService создаёт create service to work with Currency API
func NewCurrencyService(cfg *config.Config) CurrencyService {
	return &currencyService{
		currencyAPIURL: cfg.CurrencyAPIURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GetRate get currency from Currency Service
func (s *currencyService) GetRate(req *models.CurrencyRateRequest) (*models.CurrencyRateResponse, error) {
	// Form URL with query param
	baseURL := fmt.Sprintf("%s/api/v1/rates", s.currencyAPIURL)

	params := url.Values{}
	params.Add("base", req.Base)
	params.Add("target", req.Target)
	params.Add("date", req.Date)

	fullURL := fmt.Sprintf("%s?%s", baseURL, params.Encode())

	// Create HTTP request
	httpReq, err := http.NewRequest(http.MethodGet, fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Send request
	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to call currency service: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Parse JSON
	var currencyResp models.CurrencyRateResponse
	if err := json.Unmarshal(body, &currencyResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// If Currency Service returned err in response
	if currencyResp.Error != "" {
		return &currencyResp, nil
	}

	return &currencyResp, nil
}

// GetHistory get history of currencies from Currency Service
func (s *currencyService) GetHistory(req *models.HistoryRequest) (*models.HistoryResponse, error) {
	// Form URL with query params
	baseURL := fmt.Sprintf("%s/api/v1/rates/history", s.currencyAPIURL)

	params := url.Values{}
	params.Add("base", req.Base)
	params.Add("target", req.Target)
	params.Add("start_date", req.StartDate)
	params.Add("end_date", req.EndDate)

	fullURL := fmt.Sprintf("%s?%s", baseURL, params.Encode())

	// Create HTTP request
	httpReq, err := http.NewRequest(http.MethodGet, fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// send req
	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to call currency service: %w", err)
	}
	defer resp.Body.Close()

	// Read resp
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Parse JSON
	var historyResp models.HistoryResponse
	if err := json.Unmarshal(body, &historyResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// If Currency Service return err in response
	if historyResp.Error != "" {
		return &historyResp, nil
	}

	return &historyResp, nil
}
