package models

// User represents user in system
type User struct {
	ID       int    `json:"id"`
	Login    string `json:"login"`
	Password string `json:"password"` // hash
}

// LoginRequest - auth request
type LoginRequest struct {
	Login    string `json:"login" validate:"required"`
	Password string `json:"password" validate:"required"`
}

// LoginResponse - auth response
type LoginResponse struct {
	Token string `json:"token"`
	Error string `json:"error,omitempty"`
}

// AuthServiceRequest - request to Auth Service for token gen
type AuthServiceRequest struct {
	UserID int `json:"user_id"`
}

// AuthServiceResponse - request from Auth Service
type AuthServiceResponse struct {
	Token string `json:"token"`
}

// CurrencyRateRequest - request to
type CurrencyRateRequest struct {
	Base   string `json:"base" validate:"required,len=3"`
	Target string `json:"target" validate:"required,len=3"`
	Date   string `json:"date" validate:"required"`
}

// CurrencyRateResponse - request with currency
type CurrencyRateResponse struct {
	Base   string  `json:"base"`
	Target string  `json:"target"`
	Rate   float64 `json:"rate"`
	Date   string  `json:"date"`
	Error  string  `json:"error,omitempty"`
}

// HistoryRequest - history of currency
type HistoryRequest struct {
	Base      string `json:"base" validate:"required,len=3"`
	Target    string `json:"target" validate:"required,len=3"`
	StartDate string `json:"start_date" validate:"required"`
	EndDate   string `json:"end_date" validate:"required"`
}

// HistoryResponse - history response
type HistoryResponse struct {
	Base   string         `json:"base"`
	Target string         `json:"target"`
	Data   []ExchangeRate `json:"data"`
	Error  string         `json:"error,omitempty"`
}

// ExchangeRate - rate currency to date
type ExchangeRate struct {
	Base   string  `json:"base"`
	Target string  `json:"target"`
	Rate   float64 `json:"rate"`
	Date   string  `json:"date"`
}
