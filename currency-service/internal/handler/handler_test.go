package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alfascuf/PROD1/currency-service/internal/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"

	"github.com/alfascuf/PROD1/currency-service/internal/models"
	"github.com/alfascuf/PROD1/currency-service/internal/service/mocks"
)

// init инициализирует logger для тестов
func init() {
	logger.Log = zap.NewNop()
}

func TestHandler_Health(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockService(ctrl)
	handler := New(mockService)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	handler.Health(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "OK", w.Body.String())
}

func TestHandler_GetRate(t *testing.T) {
	t.Run("успешное получение курса", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockService := mocks.NewMockService(ctrl)
		handler := New(mockService)

		expectedResp := &models.GetRateResponse{
			Base:   "USD",
			Target: "RUB",
			Rate:   76.35,
			Date:   "2026-02-10",
		}

		mockService.EXPECT().
			GetRate(gomock.Any()).
			DoAndReturn(func(req *models.GetRateRequest) (*models.GetRateResponse, error) {
				assert.Equal(t, "USD", req.Base)
				assert.Equal(t, "RUB", req.Target)
				assert.Equal(t, "2026-02-10", req.Date)
				return expectedResp, nil
			}).
			Times(1)

		req := httptest.NewRequest(http.MethodGet, "/api/rate?base=USD&target=RUB&date=2026-02-10", nil)
		w := httptest.NewRecorder()

		handler.GetRate(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

		var resp models.GetRateResponse
		err := json.NewDecoder(w.Body).Decode(&resp)
		require.NoError(t, err)
		assert.Equal(t, "USD", resp.Base)
		assert.Equal(t, "RUB", resp.Target)
		assert.Equal(t, 76.35, resp.Rate)
		assert.Empty(t, resp.Error)
	})

	t.Run("валидация - отсутствует параметр base", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockService := mocks.NewMockService(ctrl)
		handler := New(mockService)

		req := httptest.NewRequest(http.MethodGet, "/api/rate?target=RUB&date=2026-02-10", nil)
		w := httptest.NewRecorder()

		handler.GetRate(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var resp models.GetRateResponse
		err := json.NewDecoder(w.Body).Decode(&resp)
		require.NoError(t, err)
		assert.NotEmpty(t, resp.Error)
	})

	t.Run("валидация - отсутствует параметр target", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockService := mocks.NewMockService(ctrl)
		handler := New(mockService)

		req := httptest.NewRequest(http.MethodGet, "/api/rate?base=USD&date=2026-02-10", nil)
		w := httptest.NewRecorder()

		handler.GetRate(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var resp models.GetRateResponse
		err := json.NewDecoder(w.Body).Decode(&resp)
		require.NoError(t, err)
		assert.NotEmpty(t, resp.Error)
	})

	t.Run("валидация - отсутствует параметр date", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockService := mocks.NewMockService(ctrl)
		handler := New(mockService)

		req := httptest.NewRequest(http.MethodGet, "/api/rate?base=USD&target=RUB", nil)
		w := httptest.NewRecorder()

		handler.GetRate(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var resp models.GetRateResponse
		err := json.NewDecoder(w.Body).Decode(&resp)
		require.NoError(t, err)
		assert.NotEmpty(t, resp.Error)
	})

	t.Run("валидация - невалидная длина валюты", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockService := mocks.NewMockService(ctrl)
		handler := New(mockService)

		req := httptest.NewRequest(http.MethodGet, "/api/rate?base=US&target=RUB&date=2026-02-10", nil)
		w := httptest.NewRecorder()

		handler.GetRate(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var resp models.GetRateResponse
		err := json.NewDecoder(w.Body).Decode(&resp)
		require.NoError(t, err)
		assert.NotEmpty(t, resp.Error)
	})

	t.Run("service вернул ошибку", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockService := mocks.NewMockService(ctrl)
		handler := New(mockService)

		mockService.EXPECT().
			GetRate(gomock.Any()).
			Return(nil, fmt.Errorf("database connection error")).
			Times(1)

		req := httptest.NewRequest(http.MethodGet, "/api/rate?base=USD&target=RUB&date=2026-02-10", nil)
		w := httptest.NewRecorder()

		handler.GetRate(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)

		var resp models.GetRateResponse
		err := json.NewDecoder(w.Body).Decode(&resp)
		require.NoError(t, err)
		assert.Contains(t, resp.Error, "database connection error")
	})

	t.Run("бизнес-логика вернула ошибку (курс не найден)", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockService := mocks.NewMockService(ctrl)
		handler := New(mockService)

		mockService.EXPECT().
			GetRate(gomock.Any()).
			Return(&models.GetRateResponse{
				Error: "rate not found for USD/EUR on 2026-02-10",
			}, nil).
			Times(1)

		req := httptest.NewRequest(http.MethodGet, "/api/rate?base=USD&target=EUR&date=2026-02-10", nil)
		w := httptest.NewRecorder()

		handler.GetRate(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var resp models.GetRateResponse
		err := json.NewDecoder(w.Body).Decode(&resp)
		require.NoError(t, err)
		assert.Contains(t, resp.Error, "rate not found")
	})

	t.Run("trim и uppercase параметров", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockService := mocks.NewMockService(ctrl)
		handler := New(mockService)

		mockService.EXPECT().
			GetRate(gomock.Any()).
			DoAndReturn(func(req *models.GetRateRequest) (*models.GetRateResponse, error) {
				// Проверяем что handler обработал параметры
				assert.Equal(t, "USD", req.Base)   // было " usd "
				assert.Equal(t, "RUB", req.Target) // было " rub "
				assert.Equal(t, "2026-02-10", req.Date)
				return &models.GetRateResponse{
					Base:   req.Base,
					Target: req.Target,
					Rate:   76.35,
					Date:   req.Date,
				}, nil
			}).
			Times(1)

		req := httptest.NewRequest(http.MethodGet, "/api/rate?base=%20usd%20&target=%20rub%20&date=2026-02-10", nil)
		w := httptest.NewRecorder()

		handler.GetRate(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

}

func TestHandler_GetHistory(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockService(ctrl)
	handler := New(mockService)

	t.Run("успешное получение истории", func(t *testing.T) {
		expectedResp := &models.GetHistoryResponse{
			Base:   "USD",
			Target: "RUB",
			Data: []models.ExchangeRate{
				{Base: "USD", Target: "RUB", Rate: 76.00, Date: parseDate("2026-02-01")},
				{Base: "USD", Target: "RUB", Rate: 76.35, Date: parseDate("2026-02-10")},
			},
		}

		mockService.EXPECT().
			GetHistory(gomock.Any()).
			DoAndReturn(func(req *models.GetHistoryRequest) (*models.GetHistoryResponse, error) {
				assert.Equal(t, "USD", req.Base)
				assert.Equal(t, "RUB", req.Target)
				assert.Equal(t, "2026-02-01", req.StartDate)
				assert.Equal(t, "2026-02-10", req.EndDate)
				return expectedResp, nil
			}).
			Times(1)

		req := httptest.NewRequest(http.MethodGet, "/api/history?base=USD&target=RUB&start_date=2026-02-01&end_date=2026-02-10", nil)
		w := httptest.NewRecorder()

		handler.GetHistory(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

		var resp models.GetHistoryResponse
		err := json.NewDecoder(w.Body).Decode(&resp)
		require.NoError(t, err)
		assert.Equal(t, "USD", resp.Base)
		assert.Equal(t, "RUB", resp.Target)
		assert.Len(t, resp.Data, 2)
		assert.Empty(t, resp.Error)
	})

	t.Run("валидация - отсутствует start_date", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/history?base=USD&target=RUB&end_date=2026-02-10", nil)
		w := httptest.NewRecorder()

		handler.GetHistory(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var resp models.GetHistoryResponse
		err := json.NewDecoder(w.Body).Decode(&resp)
		require.NoError(t, err)
		assert.NotEmpty(t, resp.Error)
	})

	t.Run("валидация - отсутствует end_date", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/history?base=USD&target=RUB&start_date=2026-02-01", nil)
		w := httptest.NewRecorder()

		handler.GetHistory(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var resp models.GetHistoryResponse
		err := json.NewDecoder(w.Body).Decode(&resp)
		require.NoError(t, err)
		assert.NotEmpty(t, resp.Error)
	})

	t.Run("валидация - start_date > end_date", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/history?base=USD&target=RUB&start_date=2026-02-10&end_date=2026-02-01", nil)
		w := httptest.NewRecorder()

		handler.GetHistory(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var resp models.GetHistoryResponse
		err := json.NewDecoder(w.Body).Decode(&resp)
		require.NoError(t, err)
		assert.Contains(t, resp.Error, "start_date")
	})

	t.Run("service вернул ошибку", func(t *testing.T) {
		mockService.EXPECT().
			GetHistory(gomock.Any()).
			Return(nil, fmt.Errorf("database error")).
			Times(1)

		req := httptest.NewRequest(http.MethodGet, "/api/history?base=USD&target=RUB&start_date=2026-02-01&end_date=2026-02-10", nil)
		w := httptest.NewRecorder()

		handler.GetHistory(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)

		var resp models.GetHistoryResponse
		err := json.NewDecoder(w.Body).Decode(&resp)
		require.NoError(t, err)
		assert.Contains(t, resp.Error, "database error")
	})

	t.Run("бизнес-логика вернула ошибку", func(t *testing.T) {
		mockService.EXPECT().
			GetHistory(gomock.Any()).
			Return(&models.GetHistoryResponse{
				Error: "failed to get history: no history for RUB/JPY",
			}, nil).
			Times(1)

		req := httptest.NewRequest(http.MethodGet, "/api/history?base=USD&target=JPY&start_date=2026-02-01&end_date=2026-02-10", nil)
		w := httptest.NewRecorder()

		handler.GetHistory(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var resp models.GetHistoryResponse
		err := json.NewDecoder(w.Body).Decode(&resp)
		require.NoError(t, err)
		assert.Contains(t, resp.Error, "failed to get history")
	})
}

// Helper function для парсинга дат в тестах
func parseDate(dateStr string) time.Time {
	t, _ := time.Parse("2006-01-02", dateStr)
	return t
}
