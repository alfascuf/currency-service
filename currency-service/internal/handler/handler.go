package handler

import (
	"encoding/json"
	"net/http"
)

func Health(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func GetRate(w http.ResponseWriter, r *http.Request) {
	type Response struct {
		Base   string `json:"base"`
		Target string `json:"target"`
		Date   string `json:"date"`
		Rate   string `json:"rate"`
		Note   string `json:"note"`
	}

	base := r.URL.Query().Get("base")
	target := r.URL.Query().Get("target")
	date := r.URL.Query().Get("date")

	if base == "" || target == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "base or target required"})
		return
	}

	resp := Response{
		Base:   base,
		Target: target,
		Date:   date,
		Note:   "stub: data not loaded yet",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
