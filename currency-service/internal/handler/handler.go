package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/alfascuf/currency-service/internal/logger"
	"github.com/alfascuf/currency-service/internal/models"
	"github.com/alfascuf/currency-service/internal/service"
	customValidator "github.com/alfascuf/currency-service/internal/validator"
	"go.uber.org/zap"
)

// Handler handles HTTP requests for currency service
type Handler struct {
	srv service.Service
}

// New creates a new Handler instance
func New(srv service.Service) *Handler {
	return &Handler{srv: srv}
}

// Health checks if the service is running
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK"))
}

// GetRate returns exchange rate for given currency pair and date.
// Query parameters:
//   - base: source currency code (3 letters, e.g. RUB)
//   - target: target currency code (3 letters, e.g. USD)
//   - date: date in format YYYY-MM-DD (e.g. 2026-02-02)
//
// Returns JSON with rate or error message
func (h *Handler) GetRate(w http.ResponseWriter, r *http.Request) {
	// Парсим query параметры
	req := &models.GetRateRequest{
		Base:   strings.TrimSpace(strings.ToUpper(r.URL.Query().Get("base"))),
		Target: strings.TrimSpace(strings.ToUpper(r.URL.Query().Get("target"))),
		Date:   strings.TrimSpace(r.URL.Query().Get("date")),
	}

	// Валидация
	if err := customValidator.Validate(req); err != nil {
		logger.Log.Warn("Validation error in GetRate",
			zap.Error(err),
			zap.String("base", req.Base),
			zap.String("target", req.Target),
			zap.String("date", req.Date),
		)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(models.GetRateResponse{
			Error: err.Error(),
		})
		return
	}

	// Вызываем service слой
	resp, err := h.srv.GetRate(req)
	if err != nil {
		logger.Log.Error("Service error in GetRate",
			zap.Error(err),
			zap.String("base", req.Base),
			zap.String("target", req.Target),
			zap.String("date", req.Date),
		)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(models.GetRateResponse{
			Error: err.Error(),
		})
		return
	}

	// Если есть ошибка в ответе от service
	if resp.Error != "" {
		logger.Log.Info("Business logic error in GetRate",
			zap.String("error", resp.Error),
			zap.String("base", req.Base),
			zap.String("target", req.Target),
			zap.String("date", req.Date),
		)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(resp)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

// GetHistory returns historical exchange rates for given currency pair and date range.
// Query parameters:
//   - base: source currency code (3 letters, e.g. RUB)
//   - target: target currency code (3 letters, e.g. USD)
//   - start_date: start date in format YYYY-MM-DD
//   - end_date: end date in format YYYY-MM-DD
//
// Returns JSON array with rates or error message
func (h *Handler) GetHistory(w http.ResponseWriter, r *http.Request) {
	// Парсим query параметры
	req := &models.GetHistoryRequest{
		Base:      strings.TrimSpace(strings.ToUpper(r.URL.Query().Get("base"))),
		Target:    strings.TrimSpace(strings.ToUpper(r.URL.Query().Get("target"))),
		StartDate: strings.TrimSpace(r.URL.Query().Get("start_date")),
		EndDate:   strings.TrimSpace(r.URL.Query().Get("end_date")),
	}

	// Валидация базовых полей
	if err := customValidator.Validate(req); err != nil {
		logger.Log.Warn("Validation error in GetHistory",
			zap.Error(err),
			zap.String("base", req.Base),
			zap.String("target", req.Target),
			zap.String("start_date", req.StartDate),
			zap.String("end_date", req.EndDate),
		)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(models.GetHistoryResponse{
			Error: err.Error(),
		})
		return
	}

	// Дополнительная валидация диапазона дат
	if err := customValidator.ValidateHistoryDates(req.StartDate, req.EndDate); err != nil {
		logger.Log.Warn("Date range validation error in GetHistory",
			zap.Error(err),
			zap.String("start_date", req.StartDate),
			zap.String("end_date", req.EndDate),
		)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(models.GetHistoryResponse{
			Error: err.Error(),
		})
		return
	}

	// Вызываем service слой
	resp, err := h.srv.GetHistory(req)
	if err != nil {
		logger.Log.Error("Service error in GetHistory",
			zap.Error(err),
			zap.String("base", req.Base),
			zap.String("target", req.Target),
			zap.String("start_date", req.StartDate),
			zap.String("end_date", req.EndDate),
		)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(models.GetHistoryResponse{
			Error: err.Error(),
		})
		return
	}

	// Если есть ошибка в ответе от service
	if resp.Error != "" {
		logger.Log.Info("Business logic error in GetHistory",
			zap.String("error", resp.Error),
			zap.String("base", req.Base),
			zap.String("target", req.Target),
		)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(resp)
		return
	}

	logger.Log.Info("GetHistory success",
		zap.String("base", req.Base),
		zap.String("target", req.Target),
		zap.Int("records_count", len(resp.Data)),
	)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}
