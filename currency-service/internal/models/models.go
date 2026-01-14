package models

import "time"

type ExchangeRate struct {
	ID        int64     `json:"id"`
	Base      string    `json:"base"`
	Target    string    `json:"target"`
	Rate      float64   `json:"rate"`
	Date      time.Time `json:"date"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type GetRateRequest struct {
	Base   string `json:"base"`
	Target string `json:"target"`
	Date   string `json:"date"`
}
type GetRateResponse struct {
	Base   string  `json:"base"`
	Target string  `json:"target"`
	Rate   float64 `json:"rate"`
	Date   string  `json:"date"`
	Error  string  `json:"error,omitempty"`
}

type GetHistoryRequest struct {
	Base      string `json:"base"`
	Target    string `json:"target"`
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
}

type GetHistoryResponse struct {
	Base   string         `json:"base"`
	Target string         `json:"target"`
	Data   []ExchangeRate `json:"data"`
	Error  string         `json:"error,omitempty"`
}
