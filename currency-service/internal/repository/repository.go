package repository

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/alfascuf/currency-service/internal/models"
	_ "github.com/lib/pq"
)

type Repository interface {
	InitDB() error
	SaveRate(rate *models.ExchangeRate) error
	GetRate(base, target string, date time.Time) (*models.ExchangeRate, error)
	GetHistory(base, target string, startDate, endDate time.Time) ([]models.ExchangeRate, error)
	Close() error
}

type repository struct {
	db *sql.DB
}

func New(db *sql.DB) Repository {
	return &repository{db: db}
}

// InitDB создает таблицы при старте сервиса
func (r *repository) InitDB() error {
	query := `
	CREATE TABLE IF NOT EXISTS exchange_rates (
		id SERIAL PRIMARY KEY,
		base VARCHAR(10) NOT NULL,
		target VARCHAR(10) NOT NULL,
		rate DECIMAL(20, 6) NOT NULL,
		date DATE NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(base, target, date)
	);

	CREATE INDEX IF NOT EXISTS idx_base_target_date 
	ON exchange_rates(base, target, date);
	`

	_, err := r.db.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to create tables: %w", err)
	}

	return nil
}

// SaveRate сохраняет курс в БД (или обновляет, если уже есть)
func (r *repository) SaveRate(rate *models.ExchangeRate) error {
	query := `
	INSERT INTO exchange_rates (base, target, rate, date, created_at, updated_at)
	VALUES ($1, $2, $3, $4, $5, $6)
	ON CONFLICT (base, target, date)
	DO UPDATE SET 
		rate = EXCLUDED.rate,
		updated_at = EXCLUDED.updated_at
	RETURNING id
	`

	err := r.db.QueryRow(
		query,
		rate.Base,
		rate.Target,
		rate.Rate,
		rate.Date,
		time.Now(),
		time.Now(),
	).Scan(&rate.ID)

	if err != nil {
		return fmt.Errorf("failed to save rate: %w", err)
	}

	return nil
}

// GetRate получает курс на конкретную дату
func (r *repository) GetRate(base, target string, date time.Time) (*models.ExchangeRate, error) {
	query := `
	SELECT id, base, target, rate, date, created_at, updated_at
	FROM exchange_rates
	WHERE base = $1 AND target = $2 AND date = $3
	LIMIT 1
	`

	rate := &models.ExchangeRate{}
	err := r.db.QueryRow(query, base, target, date).Scan(
		&rate.ID,
		&rate.Base,
		&rate.Target,
		&rate.Rate,
		&rate.Date,
		&rate.CreatedAt,
		&rate.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("rate not found for %s/%s on %s", base, target, date.Format("2006-01-02"))
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get rate: %w", err)
	}

	return rate, nil
}

// GetHistory получает историю курсов за период
func (r *repository) GetHistory(base, target string, startDate, endDate time.Time) ([]models.ExchangeRate, error) {
	query := `
	SELECT id, base, target, rate, date, created_at, updated_at
	FROM exchange_rates
	WHERE base = $1 AND target = $2 AND date BETWEEN $3 AND $4
	ORDER BY date ASC
	`

	rows, err := r.db.Query(query, base, target, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get history: %w", err)
	}
	defer rows.Close()

	var rates []models.ExchangeRate
	for rows.Next() {
		var rate models.ExchangeRate
		err := rows.Scan(
			&rate.ID,
			&rate.Base,
			&rate.Target,
			&rate.Rate,
			&rate.Date,
			&rate.CreatedAt,
			&rate.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		rates = append(rates, rate)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return rates, nil
}

// Close закрывает соединение с БД
func (r *repository) Close() error {
	return r.db.Close()
}
