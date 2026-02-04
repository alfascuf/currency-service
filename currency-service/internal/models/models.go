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
	Base   string `json:"base" validate:"required,len=3,alpha"`
	Target string `json:"target" validate:"required,len=3,alpha"`
	Date   string `json:"date" validate:"required,datetime=2006-01-02"`
}
type GetRateResponse struct {
	Base   string  `json:"base"`
	Target string  `json:"target"`
	Rate   float64 `json:"rate"`
	Date   string  `json:"date"`
	Error  string  `json:"error,omitempty"`
}

type GetHistoryRequest struct {
	Base      string `json:"base" validate:"required,len=3,alpha"`
	Target    string `json:"target" validate:"required,len=3,alpha"`
	StartDate string `json:"start_date" validate:"required,datetime=2006-01-02"`
	EndDate   string `json:"end_date" validate:"required,datetime=2006-01-02"`
}

type GetHistoryResponse struct {
	Base   string         `json:"base"`
	Target string         `json:"target"`
	Data   []ExchangeRate `json:"data"`
	Error  string         `json:"error,omitempty"`
}
