package handler

import (
	"encoding/json"
	"net/http"

	"github.com/alfascuf/gateway/internal/models"
	"github.com/alfascuf/gateway/internal/service"
	"go.uber.org/zap"
)

// AuthHandler обработчик для аутентификации
type AuthHandler struct {
	authService service.AuthService
	logger      *zap.Logger
}

// NewAuthHandler создаёт новый auth handler
func NewAuthHandler(authService service.AuthService, logger *zap.Logger) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		logger:      logger,
	}
}

// Login обрабатывает POST login auth
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	// Только POST метод
	if r.Method != http.MethodPost {
		respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	// Parse Json
	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Warn("Failed to decode login request",
			zap.Error(err),
		)
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validation
	if req.Login == "" || req.Password == "" {
		respondWithError(w, http.StatusBadRequest, "Login and password are required")
		return
	}

	// Authentifiaction
	resp, err := h.authService.Login(&req)
	if err != nil {
		h.logger.Error("Login failed",
			zap.Error(err),
			zap.String("login", req.Login),
		)
		respondWithError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	// Check error
	if resp.Error != "" {
		h.logger.Info("Invalid credentials",
			zap.String("login", req.Login),
		)
		respondWithJSON(w, http.StatusUnauthorized, resp)
		return
	}

	// Return token
	h.logger.Info("User logged in successfully",
		zap.String("login", req.Login),
	)
	respondWithJSON(w, http.StatusOK, resp)
}
