package service

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/alfascuf/PROD1/currency-service/internal/models"
	"github.com/alfascuf/PROD1/currency-service/internal/repository/mocks"
)

func TestService_GetRate(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockRepository(ctrl)
	svc := New(mockRepo, nil)

	testDate := time.Date(2026, 2, 10, 0, 0, 0, 0, time.UTC)
	dateStr := "2026-02-10"

	t.Run("одинаковые валюты - возвращает 1.0", func(t *testing.T) {
		req := &models.GetRateRequest{
			Base:   "USD",
			Target: "USD",
			Date:   dateStr,
		}

		resp, err := svc.GetRate(req)

		require.NoError(t, err)
		assert.Equal(t, "USD", resp.Base)
		assert.Equal(t, "USD", resp.Target)
		assert.Equal(t, 1.0, resp.Rate)
		assert.Empty(t, resp.Error)
	})

	t.Run("прямая конвертация RUB → USD", func(t *testing.T) {
		req := &models.GetRateRequest{
			Base:   "RUB",
			Target: "USD",
			Date:   dateStr,
		}

		expectedRate := &models.ExchangeRate{
			Base:   "RUB",
			Target: "USD",
			Rate:   0.0131,
			Date:   testDate,
		}

		mockRepo.EXPECT().
			GetRate("RUB", "USD", testDate).
			Return(expectedRate, nil).
			Times(1)

		resp, err := svc.GetRate(req)

		require.NoError(t, err)
		assert.Equal(t, "RUB", resp.Base)
		assert.Equal(t, "USD", resp.Target)
		assert.Equal(t, 0.0131, resp.Rate)
		assert.Empty(t, resp.Error)
	})

	t.Run("обратная конвертация USD → RUB", func(t *testing.T) {
		req := &models.GetRateRequest{
			Base:   "USD",
			Target: "RUB",
			Date:   dateStr,
		}

		expectedRate := &models.ExchangeRate{
			Base:   "RUB",
			Target: "USD",
			Rate:   0.0131,
			Date:   testDate,
		}

		mockRepo.EXPECT().
			GetRate("RUB", "USD", testDate).
			Return(expectedRate, nil).
			Times(1)

		resp, err := svc.GetRate(req)

		require.NoError(t, err)
		assert.Equal(t, "USD", resp.Base)
		assert.Equal(t, "RUB", resp.Target)
		assert.InDelta(t, 76.335, resp.Rate, 0.001) // 1 / 0.0131 ≈ 76.335
		assert.Empty(t, resp.Error)
	})

	t.Run("кросс-конвертация USD → EUR", func(t *testing.T) {
		req := &models.GetRateRequest{
			Base:   "USD",
			Target: "EUR",
			Date:   dateStr,
		}

		rubToUSD := &models.ExchangeRate{
			Base:   "RUB",
			Target: "USD",
			Rate:   0.0131,
			Date:   testDate,
		}

		rubToEUR := &models.ExchangeRate{
			Base:   "RUB",
			Target: "EUR",
			Rate:   0.0118,
			Date:   testDate,
		}

		mockRepo.EXPECT().
			GetRate("RUB", "USD", testDate).
			Return(rubToUSD, nil).
			Times(1)

		mockRepo.EXPECT().
			GetRate("RUB", "EUR", testDate).
			Return(rubToEUR, nil).
			Times(1)

		resp, err := svc.GetRate(req)

		require.NoError(t, err)
		assert.Equal(t, "USD", resp.Base)
		assert.Equal(t, "EUR", resp.Target)
		assert.InDelta(t, 0.900, resp.Rate, 0.001) // 0.0118 / 0.0131 ≈ 0.9
		assert.Empty(t, resp.Error)
	})

	t.Run("курс не найден - ошибка в response.Error", func(t *testing.T) {
		req := &models.GetRateRequest{
			Base:   "RUB",
			Target: "JPY",
			Date:   dateStr,
		}

		mockRepo.EXPECT().
			GetRate("RUB", "JPY", testDate).
			Return(nil, fmt.Errorf("not found")).
			Times(1)

		resp, err := svc.GetRate(req)

		require.NoError(t, err) // Ошибка возвращается в resp.Error
		assert.NotEmpty(t, resp.Error)
		assert.Contains(t, resp.Error, "rate not found")
	})

	t.Run("нулевой курс - защита от деления на ноль", func(t *testing.T) {
		req := &models.GetRateRequest{
			Base:   "USD",
			Target: "RUB",
			Date:   dateStr,
		}

		zeroRate := &models.ExchangeRate{
			Base:   "RUB",
			Target: "USD",
			Rate:   0.0, // Нулевой курс
			Date:   testDate,
		}

		mockRepo.EXPECT().
			GetRate("RUB", "USD", testDate).
			Return(zeroRate, nil).
			Times(1)

		resp, err := svc.GetRate(req)

		require.NoError(t, err)
		assert.NotEmpty(t, resp.Error)
		assert.Contains(t, resp.Error, "invalid rate: cannot be zero")
	})

	t.Run("невалидный формат даты", func(t *testing.T) {
		req := &models.GetRateRequest{
			Base:   "USD",
			Target: "EUR",
			Date:   "invalid-date",
		}

		resp, err := svc.GetRate(req)

		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "invalid date format")
	})
}

func TestService_GetHistory(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockRepository(ctrl)
	svc := New(mockRepo, nil)

	startDate := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2026, 2, 3, 0, 0, 0, 0, time.UTC)
	startDateStr := "2026-02-01"
	endDateStr := "2026-02-03"

	t.Run("прямая история RUB → USD", func(t *testing.T) {
		req := &models.GetHistoryRequest{
			Base:      "RUB",
			Target:    "USD",
			StartDate: startDateStr,
			EndDate:   endDateStr,
		}

		expectedHistory := []models.ExchangeRate{
			{Base: "RUB", Target: "USD", Rate: 0.0130, Date: startDate},
			{Base: "RUB", Target: "USD", Rate: 0.0131, Date: startDate.AddDate(0, 0, 1)},
			{Base: "RUB", Target: "USD", Rate: 0.0132, Date: endDate},
		}

		mockRepo.EXPECT().
			GetHistory("RUB", "USD", startDate, endDate).
			Return(expectedHistory, nil).
			Times(1)

		resp, err := svc.GetHistory(req)

		require.NoError(t, err)
		assert.Equal(t, "RUB", resp.Base)
		assert.Equal(t, "USD", resp.Target)
		assert.Len(t, resp.Data, 3)
		assert.Equal(t, 0.0130, resp.Data[0].Rate)
		assert.Empty(t, resp.Error)
	})

	t.Run("обратная история USD → RUB с инверсией курсов", func(t *testing.T) {
		req := &models.GetHistoryRequest{
			Base:      "USD",
			Target:    "RUB",
			StartDate: startDateStr,
			EndDate:   endDateStr,
		}

		rubToUSD := []models.ExchangeRate{
			{Base: "RUB", Target: "USD", Rate: 0.0131, Date: startDate},
			{Base: "RUB", Target: "USD", Rate: 0.0132, Date: endDate},
		}

		mockRepo.EXPECT().
			GetHistory("RUB", "USD", startDate, endDate).
			Return(rubToUSD, nil).
			Times(1)

		resp, err := svc.GetHistory(req)

		require.NoError(t, err)
		assert.Equal(t, "USD", resp.Base)
		assert.Equal(t, "RUB", resp.Target)
		assert.Len(t, resp.Data, 2)
		// Проверяем инвертированные курсы
		assert.InDelta(t, 76.335, resp.Data[0].Rate, 0.001) // 1/0.0131
		assert.InDelta(t, 75.757, resp.Data[1].Rate, 0.001) // 1/0.0132
	})

	t.Run("кросс-история USD → EUR", func(t *testing.T) {
		req := &models.GetHistoryRequest{
			Base:      "USD",
			Target:    "EUR",
			StartDate: startDateStr,
			EndDate:   endDateStr,
		}

		rubToUSD := []models.ExchangeRate{
			{Base: "RUB", Target: "USD", Rate: 0.0131, Date: startDate},
			{Base: "RUB", Target: "USD", Rate: 0.0132, Date: endDate},
		}

		rubToEUR := []models.ExchangeRate{
			{Base: "RUB", Target: "EUR", Rate: 0.0118, Date: startDate},
			{Base: "RUB", Target: "EUR", Rate: 0.0119, Date: endDate},
		}

		mockRepo.EXPECT().
			GetHistory("RUB", "USD", startDate, endDate).
			Return(rubToUSD, nil).
			Times(1)

		mockRepo.EXPECT().
			GetHistory("RUB", "EUR", startDate, endDate).
			Return(rubToEUR, nil).
			Times(1)

		resp, err := svc.GetHistory(req)

		require.NoError(t, err)
		assert.Equal(t, "USD", resp.Base)
		assert.Equal(t, "EUR", resp.Target)
		assert.Len(t, resp.Data, 2)
		// Проверяем кросс-курсы
		assert.InDelta(t, 0.900, resp.Data[0].Rate, 0.001) // 0.0118/0.0131
		assert.InDelta(t, 0.901, resp.Data[1].Rate, 0.001) // 0.0119/0.0132
	})

	t.Run("история не найдена для базовой валюты", func(t *testing.T) {
		req := &models.GetHistoryRequest{
			Base:      "USD",
			Target:    "EUR",
			StartDate: startDateStr,
			EndDate:   endDateStr,
		}

		mockRepo.EXPECT().
			GetHistory("RUB", "USD", startDate, endDate).
			Return(nil, fmt.Errorf("not found")).
			Times(1)

		resp, err := svc.GetHistory(req)

		require.NoError(t, err)
		assert.NotEmpty(t, resp.Error)
		assert.Contains(t, resp.Error, "failed to get history")
	})

	t.Run("пропуск нулевых курсов при инверсии", func(t *testing.T) {
		req := &models.GetHistoryRequest{
			Base:      "USD",
			Target:    "RUB",
			StartDate: startDateStr,
			EndDate:   endDateStr,
		}

		rubToUSD := []models.ExchangeRate{
			{Base: "RUB", Target: "USD", Rate: 0.0131, Date: startDate},
			{Base: "RUB", Target: "USD", Rate: 0.0, Date: startDate.AddDate(0, 0, 1)}, // Нулевой
			{Base: "RUB", Target: "USD", Rate: 0.0132, Date: endDate},
		}

		mockRepo.EXPECT().
			GetHistory("RUB", "USD", startDate, endDate).
			Return(rubToUSD, nil).
			Times(1)

		resp, err := svc.GetHistory(req)

		require.NoError(t, err)
		assert.Len(t, resp.Data, 2) // Нулевой курс должен быть пропущен
	})

	t.Run("невалидный формат start_date", func(t *testing.T) {
		req := &models.GetHistoryRequest{
			Base:      "USD",
			Target:    "EUR",
			StartDate: "invalid",
			EndDate:   endDateStr,
		}

		resp, err := svc.GetHistory(req)

		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "invalid start_date format")
	})

	t.Run("невалидный формат end_date", func(t *testing.T) {
		req := &models.GetHistoryRequest{
			Base:      "USD",
			Target:    "EUR",
			StartDate: startDateStr,
			EndDate:   "invalid",
		}

		resp, err := svc.GetHistory(req)

		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "invalid end_date format")
	})
}

func TestService_SaveRate(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockRepository(ctrl)
	svc := New(mockRepo, nil)

	t.Run("успешное сохранение курса", func(t *testing.T) {
		rate := &models.ExchangeRate{
			Base:   "RUB",
			Target: "USD",
			Rate:   0.0131,
			Date:   time.Date(2026, 2, 10, 0, 0, 0, 0, time.UTC),
		}

		mockRepo.EXPECT().
			SaveRate(rate).
			Return(nil).
			Times(1)

		err := svc.SaveRate(rate)

		assert.NoError(t, err)
	})

	t.Run("ошибка при сохранении", func(t *testing.T) {
		rate := &models.ExchangeRate{
			Base:   "RUB",
			Target: "USD",
			Rate:   0.0131,
			Date:   time.Date(2026, 2, 10, 0, 0, 0, 0, time.UTC),
		}

		mockRepo.EXPECT().
			SaveRate(rate).
			Return(fmt.Errorf("database error")).
			Times(1)

		err := svc.SaveRate(rate)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "database error")
	})
}
