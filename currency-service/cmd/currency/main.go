package main

import (
	"log"
	"net/http"
	"time"

	"github.com/alfascuf/currency-service/internal/config"
	"github.com/alfascuf/currency-service/internal/handler"
)

func main() {
	cfg := config.Load()
	log.Printf("Currency service started %s\n", cfg.Port)
	mux := http.NewServeMux()
	mux.HandleFunc("/health", handler.Health)
	mux.HandleFunc("/api/v1/rates", handler.GetRate)

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	log.Fatal(srv.ListenAndServe())

}
