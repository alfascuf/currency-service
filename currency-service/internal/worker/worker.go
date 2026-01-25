package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/alfascuf/currency-service/internal/models"
	"github.com/alfascuf/currency-service/internal/repository"
	"github.com/robfig/cron/v3"
)

type Worker struct {
	repo         repository.Repository // Используем интерфейс
	apiURL       string
	baseCurrency string
	cron         *cron.Cron
}

type FrankfurterResponse struct {
	Amount float64            `json:"amount"`
	Base   string             `json:"base"`
	Date   string             `json:"date"`
	Rates  map[string]float64 `json:"rates"`
}

func NewWorker(repo repository.Repository, apiURL, baseCurrency string) *Worker {
	return &Worker{
		repo:         repo,
		apiURL:       apiURL,
		baseCurrency: baseCurrency,
		cron:         cron.New(),
	}
}

func (w *Worker) Start(ctx context.Context) error {
	log.Println("Starting currency worker...")

	// Сразу обновляем курсы при старте
	if err := w.fetchAndUpdateRates(ctx); err != nil {
		log.Printf("Initial fetch failed: %v", err)
	}

	// Запускаем обновление каждый день в 00:00
	_, err := w.cron.AddFunc("0 0 * * *", func() {
		if err := w.fetchAndUpdateRates(ctx); err != nil {
			log.Printf("Failed to fetch rates: %v", err)
		}
	})
	if err != nil {
		return fmt.Errorf("failed to schedule cron job: %w", err)
	}

	w.cron.Start()
	log.Println("Currency worker started successfully")

	return nil
}

func (w *Worker) Stop() {
	log.Println("Stopping currency worker...")
	w.cron.Stop()
}

func (w *Worker) fetchAndUpdateRates(ctx context.Context) error {
	log.Println("Fetching currency rates...")

	url := fmt.Sprintf("%s/latest?from=%s", w.apiURL, w.baseCurrency)

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch rates: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var apiResp FrankfurterResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	// Сохраняем курсы в БД
	for currency, rate := range apiResp.Rates {
		exchangeRate := &models.ExchangeRate{
			Base:   apiResp.Base, // Используем Base вместо FromCurrency
			Target: currency,     // Используем Target вместо ToCurrency
			Rate:   rate,
			Date:   time.Now(),
		}

		// Используем SaveRate вместо SaveExchangeRate
		if err := w.repo.SaveRate(exchangeRate); err != nil {
			log.Printf("Failed to save rate %s/%s: %v", apiResp.Base, currency, err)
			continue
		}
	}

	log.Printf("Successfully updated %d currency rates", len(apiResp.Rates))
	return nil
}
