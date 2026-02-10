package repository

import (
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/alfascuf/currency-service/internal/models"
)

func TestRepository_SaveRate(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := New(db)

	testDate := time.Date(2026, 2, 10, 0, 0, 0, 0, time.UTC)
	rate := &models.ExchangeRate{
		Base:   "USD",
		Target: "RUB",
		Rate:   95.50,
		Date:   testDate,
	}

	t.Run("успешное сохранение нового курса", func(t *testing.T) {
		mock.ExpectQuery(`INSERT INTO exchange_rates`).
			WithArgs(
				rate.Base,
				rate.Target,
				rate.Rate,
				rate.Date,
				sqlmock.AnyArg(), // created_at
				sqlmock.AnyArg(), // updated_at
			).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))

		err := repo.SaveRate(rate)

		require.NoError(t, err)
		assert.Equal(t, int64(1), rate.ID)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("обновление существующего курса (conflict)", func(t *testing.T) {
		mock.ExpectQuery(`INSERT INTO exchange_rates`).
			WithArgs(
				rate.Base,
				rate.Target,
				rate.Rate,
				rate.Date,
				sqlmock.AnyArg(),
				sqlmock.AnyArg(),
			).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(2))

		err := repo.SaveRate(rate)

		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("ошибка базы данных", func(t *testing.T) {
		mock.ExpectQuery(`INSERT INTO exchange_rates`).
			WithArgs(
				rate.Base,
				rate.Target,
				rate.Rate,
				rate.Date,
				sqlmock.AnyArg(),
				sqlmock.AnyArg(),
			).
			WillReturnError(sql.ErrConnDone)

		err := repo.SaveRate(rate)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to save rate")
	})
}

func TestRepository_GetRate(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := New(db)

	testDate := time.Date(2026, 2, 10, 0, 0, 0, 0, time.UTC)
	createdAt := time.Now()
	updatedAt := time.Now()

	t.Run("успешное получение курса", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{
			"id", "base", "target", "rate", "date", "created_at", "updated_at",
		}).AddRow(1, "USD", "RUB", 95.50, testDate, createdAt, updatedAt)

		mock.ExpectQuery(`SELECT (.+) FROM exchange_rates`).
			WithArgs("USD", "RUB", testDate).
			WillReturnRows(rows)

		rate, err := repo.GetRate("USD", "RUB", testDate)

		require.NoError(t, err)
		assert.NotNil(t, rate)
		assert.Equal(t, int64(1), rate.ID)
		assert.Equal(t, "USD", rate.Base)
		assert.Equal(t, "RUB", rate.Target)
		assert.Equal(t, 95.50, rate.Rate)
		assert.Equal(t, testDate, rate.Date)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("курс не найден", func(t *testing.T) {
		mock.ExpectQuery(`SELECT (.+) FROM exchange_rates`).
			WithArgs("EUR", "RUB", testDate).
			WillReturnError(sql.ErrNoRows)

		rate, err := repo.GetRate("EUR", "RUB", testDate)

		assert.Error(t, err)
		assert.Nil(t, rate)
		assert.Contains(t, err.Error(), "rate not found")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("ошибка базы данных", func(t *testing.T) {
		mock.ExpectQuery(`SELECT (.+) FROM exchange_rates`).
			WithArgs("USD", "RUB", testDate).
			WillReturnError(sql.ErrConnDone)

		rate, err := repo.GetRate("USD", "RUB", testDate)

		assert.Error(t, err)
		assert.Nil(t, rate)
		assert.Contains(t, err.Error(), "failed to get rate")
	})
}

func TestRepository_GetHistory(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := New(db)

	startDate := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2026, 2, 10, 0, 0, 0, 0, time.UTC)
	createdAt := time.Now()
	updatedAt := time.Now()

	t.Run("успешное получение истории", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{
			"id", "base", "target", "rate", "date", "created_at", "updated_at",
		}).
			AddRow(1, "USD", "RUB", 95.00, startDate, createdAt, updatedAt).
			AddRow(2, "USD", "RUB", 95.50, endDate, createdAt, updatedAt)

		mock.ExpectQuery(`SELECT (.+) FROM exchange_rates`).
			WithArgs("USD", "RUB", startDate, endDate).
			WillReturnRows(rows)

		rates, err := repo.GetHistory("USD", "RUB", startDate, endDate)

		require.NoError(t, err)
		assert.Len(t, rates, 2)
		assert.Equal(t, 95.00, rates[0].Rate)
		assert.Equal(t, 95.50, rates[1].Rate)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("пустая история (нет данных)", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{
			"id", "base", "target", "rate", "date", "created_at", "updated_at",
		})

		mock.ExpectQuery(`SELECT (.+) FROM exchange_rates`).
			WithArgs("EUR", "RUB", startDate, endDate).
			WillReturnRows(rows)

		rates, err := repo.GetHistory("EUR", "RUB", startDate, endDate)

		require.NoError(t, err)
		assert.Empty(t, rates)
	})

	t.Run("ошибка при запросе", func(t *testing.T) {
		mock.ExpectQuery(`SELECT (.+) FROM exchange_rates`).
			WithArgs("USD", "RUB", startDate, endDate).
			WillReturnError(sql.ErrConnDone)

		rates, err := repo.GetHistory("USD", "RUB", startDate, endDate)

		assert.Error(t, err)
		assert.Nil(t, rates)
		assert.Contains(t, err.Error(), "failed to get history")
	})

	t.Run("ошибка при сканировании строки", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{
			"id", "base", "target", "rate", "date", "created_at", "updated_at",
		}).
			AddRow(1, "USD", "RUB", "invalid_float", startDate, createdAt, updatedAt)

		mock.ExpectQuery(`SELECT (.+) FROM exchange_rates`).
			WithArgs("USD", "RUB", startDate, endDate).
			WillReturnRows(rows)

		rates, err := repo.GetHistory("USD", "RUB", startDate, endDate)

		assert.Error(t, err)
		assert.Nil(t, rates)
		assert.Contains(t, err.Error(), "failed to scan row")
	})
}

func TestRepository_Close(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)

	repo := New(db)

	mock.ExpectClose()

	err = repo.Close()

	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}
