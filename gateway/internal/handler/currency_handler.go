package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/alfascuf/gateway/internal/middleware"
	"github.com/alfascuf/gateway/internal/models"
	"github.com/alfascuf/gateway/internal/service"
	"go.uber.org/zap"
)

// CurrencyHandler обработчик для курсов валют
type CurrencyHandler struct {
	currencyService service.CurrencyService
	logger          *zap.Logger
}

// NewCurrencyHandler создаёт новый currency handler
func NewCurrencyHandler(currencyService service.CurrencyService, logger *zap.Logger) *CurrencyHandler {
	return &CurrencyHandler{
		currencyService: currencyService,
		logger:          logger,
	}
}

// GetRate обрабатывает GET /api/rates
func (h *CurrencyHandler) GetRate(w http.ResponseWriter, r *http.Request) {
	// Получаем user_id из контекста
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	// Parse query параметры
	req := &models.CurrencyRateRequest{
		Base:   strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("base"))),
		Target: strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("target"))),
		Date:   strings.TrimSpace(r.URL.Query().Get("date")),
	}

	// Validation
	if req.Base == "" || req.Target == "" || req.Date == "" {
		respondWithError(w, http.StatusBadRequest, "base, target, and date parameters are required")
		return
	}

	if len(req.Base) != 3 || len(req.Target) != 3 {
		respondWithError(w, http.StatusBadRequest, "Currency codes must be 3 characters")
		return
	}

	// Call Currency Service
	resp, err := h.currencyService.GetRate(req)
	if err != nil {
		h.logger.Error("Failed to get rate",
			zap.Error(err),
			zap.Int("user_id", userID),
			zap.String("base", req.Base),
			zap.String("target", req.Target),
		)
		respondWithError(w, http.StatusInternalServerError, "Failed to get exchange rate")
		return
	}

	// Если Currency Service вернул err
	if resp.Error != "" {
		h.logger.Info("Currency service returned error",
			zap.String("error", resp.Error),
			zap.Int("user_id", userID),
		)
		respondWithJSON(w, http.StatusBadRequest, resp)
		return
	}

	// Success
	h.logger.Info("Rate retrieved successfully",
		zap.Int("user_id", userID),
		zap.String("base", req.Base),
		zap.String("target", req.Target),
		zap.Float64("rate", resp.Rate),
	)
	respondWithJSON(w, http.StatusOK, resp)
}

// GetHistory обрабатывает GET /api/rates/history
func (h *CurrencyHandler) GetHistory(w http.ResponseWriter, r *http.Request) {
	// Get user_id из контекста
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	// Parse query параметры
	req := &models.HistoryRequest{
		Base:      strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("base"))),
		Target:    strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("target"))),
		StartDate: strings.TrimSpace(r.URL.Query().Get("start_date")),
		EndDate:   strings.TrimSpace(r.URL.Query().Get("end_date")),
	}

	// Validation
	if req.Base == "" || req.Target == "" || req.StartDate == "" || req.EndDate == "" {
		respondWithError(w, http.StatusBadRequest, "base, target, start_date, and end_date parameters are required")
		return
	}

	if len(req.Base) != 3 || len(req.Target) != 3 {
		respondWithError(w, http.StatusBadRequest, "Currency codes must be 3 characters")
		return
	}

	// call Currency Service
	resp, err := h.currencyService.GetHistory(req)
	if err != nil {
		h.logger.Error("Failed to get history",
			zap.Error(err),
			zap.Int("user_id", userID),
			zap.String("base", req.Base),
			zap.String("target", req.Target),
		)
		respondWithError(w, http.StatusInternalServerError, "Failed to get exchange rate history")
		return
	}

	// Если Currency Service error
	if resp.Error != "" {
		h.logger.Info("Currency service returned error",
			zap.String("error", resp.Error),
			zap.Int("user_id", userID),
		)
		respondWithJSON(w, http.StatusBadRequest, resp)
		return
	}

	// success
	h.logger.Info("History retrieved successfully",
		zap.Int("user_id", userID),
		zap.String("base", req.Base),
		zap.String("target", req.Target),
		zap.Int("records", len(resp.Data)),
	)
	respondWithJSON(w, http.StatusOK, resp)
}

// respondWithJSON отправляет JSON ответ
func respondWithJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(data)
}

// respondWithError отправляет JSON ошибку
func respondWithError(w http.ResponseWriter, statusCode int, message string) {
	respondWithJSON(w, statusCode, map[string]string{"error": message})
}
