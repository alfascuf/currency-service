package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/alfascuf/PROD1/gateway/internal/service"
)

// contextKey - type for key context
type contextKey string

const (
	// UserIDKey - key for storage user_id
	UserIDKey contextKey = "user_id"
)

// AuthMiddleware create middleware for check JWT token
func AuthMiddleware(authService service.AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 1. Get token from header Authorization
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				respondWithError(w, http.StatusUnauthorized, "Authorization header required")
				return
			}

			// 2. Check form: "Bearer <token>"
			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				respondWithError(w, http.StatusUnauthorized, "Invalid authorization header format. Use: Bearer <token>")
				return
			}

			token := parts[1]

			// 3. Validate token and get user_id
			userID, err := authService.ValidateToken(token)
			if err != nil {
				respondWithError(w, http.StatusUnauthorized, "Invalid or expired token")
				return
			}

			// 4. Save user_id in context request
			ctx := context.WithValue(r.Context(), UserIDKey, userID)
			r = r.WithContext(ctx)

			// 5. Give management to the next handler
			next.ServeHTTP(w, r)
		})
	}
}

// GetUserID get user_id from request's context
func GetUserID(ctx context.Context) (int, bool) {
	userID, ok := ctx.Value(UserIDKey).(int)
	return userID, ok
}

// respondWithError send JSON err
func respondWithError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_, _ = w.Write([]byte(`{"error":"` + message + `"}`))

}
