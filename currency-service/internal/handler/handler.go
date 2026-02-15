package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/alfascuf/PROD1/currency-service/internal/logger"
	"github.com/alfascuf/PROD1/currency-service/internal/models"
	"github.com/alfascuf/PROD1/currency-service/internal/service"
	customValidator "github.com/alfascuf/PROD1/currency-service/internal/validator"
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

// Health godoc
// @Summary      Health check
// @Description  Проверка состояния API
// @Tags         health
// @Produce      json
// @Success      200  {object}  map[string]string
// @Router       /health [get]
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// GetRate godoc
// @Summary      Получить курс валюты
// @Description  Возвращает курс обмена между двумя валютами на указанную дату
// @Tags         rates
// @Accept       json
// @Produce      json
// @Param        base    query     string  true  "Базовая валюта (например: USD, RUB, EUR)"
// @Param        target  query     string  true  "Целевая валюта (например: USD, RUB, EUR)"
// @Param        date    query     string  true  "Дата в формате YYYY-MM-DD (например: 2026-02-15)"
// @Success      200  {object}  models.GetRateResponse
// @Failure      400  {object}  models.GetRateResponse
// @Router       /api/v1/rates [get]
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

// GetHistory godoc
// @Summary      Получить историю курсов
// @Description  Возвращает исторические данные курсов обмена между двумя валютами за указанный период
// @Tags         rates
// @Accept       json
// @Produce      json
// @Param        base         query     string  true  "Базовая валюта (например: USD, RUB, EUR)"
// @Param        target       query     string  true  "Целевая валюта (например: USD, RUB, EUR)"
// @Param        start_date   query     string  true  "Начальная дата в формате YYYY-MM-DD"
// @Param        end_date     query     string  true  "Конечная дата в формате YYYY-MM-DD"
// @Success      200  {object}  models.GetHistoryResponse
// @Failure      400  {object}  models.GetHistoryResponse
// @Router       /api/v1/rates/history [get]
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
