package middleware

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/igorsal/pr-documentator/internal/interfaces"
)

func TokenAuthMiddleware(tokenManager interfaces.TokenManager, logger interfaces.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := extractToken(r)
			if token == "" {
				writeUnauthorizedResponse(w, "authorization token required", logger)
				return
			}

			_, exists := tokenManager.GetSession(token)
			if !exists {
				writeUnauthorizedResponse(w, "invalid or expired token", logger)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func extractToken(r *http.Request) string {
	// Check Authorization header
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" {
		// Remove "Bearer " prefix if present
		if strings.HasPrefix(authHeader, "Bearer ") {
			return authHeader[7:]
		}
		return authHeader
	}

	// Check query parameter as fallback
	return r.URL.Query().Get("token")
}

func writeUnauthorizedResponse(w http.ResponseWriter, message string, logger interfaces.Logger) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)

	response := map[string]string{
		"error": message,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		logger.Error("Failed to write unauthorized response", err)
	}
}