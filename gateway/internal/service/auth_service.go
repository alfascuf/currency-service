package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/alfascuf/PROD1/gateway/internal/config"
	"github.com/alfascuf/PROD1/gateway/internal/models"
	"github.com/alfascuf/PROD1/gateway/internal/repository"
	"github.com/golang-jwt/jwt/v5"
)

const (
	httpTimeout = 10 * time.Second
)

// AuthService interface for auth
type AuthService interface {
	Login(req *models.LoginRequest) (*models.LoginResponse, error)
	ValidateToken(tokenString string) (int, error) // returns user_id
}

type authService struct {
	userRepo   repository.UserRepository
	authAPIURL string
	jwtSecret  string
	httpClient *http.Client
}

// NewAuthService create new auth service
func NewAuthService(cfg *config.Config, userRepo repository.UserRepository) AuthService {
	return &authService{
		userRepo:   userRepo,
		authAPIURL: cfg.AuthAPIURL,
		jwtSecret:  cfg.JWTSecret,
		httpClient: &http.Client{
			Timeout: httpTimeout,
		},
	}
}

// Login makes auth of user
func (s *authService) Login(req *models.LoginRequest) (*models.LoginResponse, error) {
	// 1. Find user by Login
	user, err := s.userRepo.GetByLogin(req.Login)
	if err != nil {
		return &models.LoginResponse{
			Error: "Invalid credentials",
		}, nil
	}
	// 2. Check login
	// in prod use bcrypt.CompareHashAndPassword
	if user.Password != req.Password {
		return &models.LoginResponse{
			Error: "Invalid credentials",
		}, nil
	}

	// 3. Get JWT token by Auth Service
	token, err := s.generateToken(user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	return &models.LoginResponse{
		Token: token,
	}, nil
}

// generateToken response token to Auth Service API
func (s *authService) generateToken(userID int) (string, error) {
	// create to AuthService
	authReq := models.AuthServiceRequest{
		UserID: userID,
	}

	reqBody, err := json.Marshal(authReq)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// URL Auth Service for token gen
	url := fmt.Sprintf("%s/generate", s.authAPIURL)

	// Create HTTP request
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(reqBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to call auth service: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("auth service returned %d: %s", resp.StatusCode, string(body))
	}

	// Read token from response
	var authResp models.AuthServiceResponse
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return authResp.Token, nil
}

// ValidateToken check JWT token and return user_id
func (s *authService) ValidateToken(tokenString string) (int, error) {
	// Parse token
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Check sign-algo
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.jwtSecret), nil
	})

	if err != nil {
		return 0, fmt.Errorf("invalid token: %w", err)
	}

	// Get claims (token's data)
	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		// Get user_id from token
		userIDFloat, ok := claims["user_id"].(float64)
		if !ok {
			return 0, fmt.Errorf("user_id not found in token")
		}

		userID := int(userIDFloat)
		return userID, nil
	}

	return 0, fmt.Errorf("invalid token claims")
}
