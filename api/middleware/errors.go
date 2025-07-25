package middleware

import (
	"encoding/json"
	"net/http"

	"github.com/igorsal/pr-documentator/internal/interfaces"
	pkgerrors "github.com/igorsal/pr-documentator/pkg/errors"
)

// ErrorResponse represents a structured error response
type ErrorResponse struct {
	Error   ErrorDetail `json:"error"`
	TraceID string      `json:"trace_id,omitempty"`
}

type ErrorDetail struct {
	Type    string         `json:"type"`
	Message string         `json:"message"`
	Code    string         `json:"code,omitempty"`
	Context map[string]any `json:"context,omitempty"`
}

// ErrorHandlerMiddleware provides centralized error handling
func ErrorHandlerMiddleware(logger interfaces.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Create a custom ResponseWriter that can capture errors
			errorWriter := &errorResponseWriter{
				ResponseWriter: w,
				logger:         logger,
				request:        r,
			}

			next.ServeHTTP(errorWriter, r)
		})
	}
}

type errorResponseWriter struct {
	http.ResponseWriter
	logger  interfaces.Logger
	request *http.Request
}

// WriteError writes a structured error response
func (erw *errorResponseWriter) WriteError(err error) {
	var appErr *pkgerrors.AppError
	var statusCode int
	var errorResp ErrorResponse

	// Check if it's our custom error type
	if pkgerrors.IsAppError(err) {
		appErr, _ = pkgerrors.AsAppError(err)
		statusCode = appErr.StatusCode
		errorResp = ErrorResponse{
			Error: ErrorDetail{
				Type:    string(appErr.Type),
				Message: appErr.Message,
				Code:    appErr.Code,
				Context: appErr.Context,
			},
		}
	} else {
		// Generic error handling
		statusCode = http.StatusInternalServerError
		errorResp = ErrorResponse{
			Error: ErrorDetail{
				Type:    string(pkgerrors.ErrorTypeInternal),
				Message: "Internal server error",
			},
		}
	}

	// Log the error with context
	erw.logger.Error("Request error",
		err,
		"method", erw.request.Method,
		"path", erw.request.URL.Path,
		"remote_addr", erw.request.RemoteAddr,
		"status_code", statusCode,
		"error_type", errorResp.Error.Type,
	)

	// Write response
	erw.Header().Set("Content-Type", "application/json") // nolint
	erw.WriteHeader(statusCode)                          // nolint

	if err := json.NewEncoder(erw).Encode(errorResp); err != nil {
		erw.logger.Error("Failed to encode error response", err)
		// Fallback to plain text
		http.Error(erw, "Internal server error", http.StatusInternalServerError)
	}
}

// PanicRecoveryMiddleware recovers from panics and converts them to errors
func PanicRecoveryMiddleware(logger interfaces.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if recovery := recover(); recovery != nil {
					logger.Error("Panic recovered",
						pkgerrors.NewInternalError("panic recovered"),
						"method", r.Method,
						"path", r.URL.Path,
						"remote_addr", r.RemoteAddr,
						"panic", recovery,
					)

					errorWriter := &errorResponseWriter{
						ResponseWriter: w,
						logger:         logger,
						request:        r,
					}

					errorWriter.WriteError(pkgerrors.NewInternalError("Internal server error"))
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}
