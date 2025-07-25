package middleware

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/igorsal/pr-documentator/internal/interfaces"
)

// GitHubWebhookAuth validates GitHub webhook signatures
func GitHubWebhookAuth(secret string, logger interfaces.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip validation if no secret is configured
			if secret == "" {
				logger.Warn("GitHub webhook secret not configured, skipping signature validation")
				next.ServeHTTP(w, r)
				return
			}

			// Get the signature from headers
			signature := r.Header.Get("X-Hub-Signature-256")
			if signature == "" {
				logger.Warn("Missing X-Hub-Signature-256 header")
				http.Error(w, "Missing signature", http.StatusUnauthorized)
				return
			}

			// Read the body
			body, err := io.ReadAll(r.Body)
			if err != nil {
				logger.Error("Failed to read request body", err)
				http.Error(w, "Failed to read body", http.StatusBadRequest)
				return
			}

			// Validate the signature
			if !validateGitHubSignature(signature, body, secret) {
				logger.Error("Invalid GitHub webhook signature", nil, "signature", signature)
				http.Error(w, "Invalid signature", http.StatusUnauthorized)
				return
			}

			// Create a new request with the body restored
			r.Body = io.NopCloser(strings.NewReader(string(body)))

			logger.Debug("GitHub webhook signature validated successfully")
			next.ServeHTTP(w, r)
		})
	}
}

func validateGitHubSignature(signature string, body []byte, secret string) bool {
	// Remove 'sha256=' prefix
	if !strings.HasPrefix(signature, "sha256=") {
		return false
	}
	signature = signature[7:]

	// Calculate expected signature
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expectedSignature := hex.EncodeToString(mac.Sum(nil))

	// Compare signatures
	return hmac.Equal([]byte(signature), []byte(expectedSignature))
}

// CORSMiddleware adds CORS headers
func CORSMiddleware(logger interfaces.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
			w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, X-GitHub-Event, X-Hub-Signature-256")

			// Handle preflight requests
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RecoveryMiddleware recovers from panics
func RecoveryMiddleware(logger interfaces.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					logger.Error("Panic recovered", fmt.Errorf("%v", err),
						"method", r.Method,
						"path", r.URL.Path,
						"remote_addr", r.RemoteAddr,
					)
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
