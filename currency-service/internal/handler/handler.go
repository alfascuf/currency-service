package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/alfascuf/currency-service/internal/models"
	"github.com/alfascuf/currency-service/internal/service"
)

type Handler struct {
	srv service.Service
}

func New(srv service.Service) *Handler {
	return &Handler{srv: srv}
}

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func (h *Handler) GetRate(w http.ResponseWriter, r *http.Request) {
	// Парсим query параметры
	req := &models.GetRateRequest{
		Base:   strings.TrimSpace(r.URL.Query().Get("base")),
		Target: strings.TrimSpace(r.URL.Query().Get("target")),
		Date:   strings.TrimSpace(r.URL.Query().Get("date")),
	}

	// Вызываем service слой
	resp, err := h.srv.GetRate(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *Handler) GetHistory(w http.ResponseWriter, r *http.Request) {
	// Парсим query параметры
	req := &models.GetHistoryRequest{
		Base:      strings.TrimSpace(r.URL.Query().Get("base")),
		Target:    strings.TrimSpace(r.URL.Query().Get("target")),
		StartDate: strings.TrimSpace(r.URL.Query().Get("start_date")),
		EndDate:   strings.TrimSpace(r.URL.Query().Get("end_date")),
	}

	// Вызываем service слой
	resp, err := h.srv.GetHistory(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
