package errors

import (
	"fmt"
	"net/http"
)

// ErrorType represents different types of errors in the system
type ErrorType string

const (
	ErrorTypeValidation   ErrorType = "validation"
	ErrorTypeNotFound     ErrorType = "not_found"
	ErrorTypeUnauthorized ErrorType = "unauthorized"
	ErrorTypeExternal     ErrorType = "external"
	ErrorTypeInternal     ErrorType = "internal"
	ErrorTypeRateLimit    ErrorType = "rate_limit"
	ErrorTypeTimeout      ErrorType = "timeout"
	ErrorTypeUnavailable  ErrorType = "unavailable"
)

// AppError represents a structured application error
type AppError struct {
	Type       ErrorType              `json:"type"`
	Message    string                 `json:"message"`
	Code       string                 `json:"code,omitempty"`
	StatusCode int                    `json:"status_code"`
	Cause      error                  `json:"-"`
	Context    map[string]interface{} `json:"context,omitempty"`
}

func (e *AppError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

func (e *AppError) Unwrap() error {
	return e.Cause
}

func (e *AppError) WithContext(key string, value interface{}) *AppError {
	if e.Context == nil {
		e.Context = make(map[string]interface{})
	}
	e.Context[key] = value
	return e
}

func (e *AppError) WithCause(cause error) *AppError {
	e.Cause = cause
	return e
}

// Error constructors
func NewValidationError(message string) *AppError {
	return &AppError{
		Type:       ErrorTypeValidation,
		Message:    message,
		StatusCode: http.StatusBadRequest,
	}
}

func NewNotFoundError(message string) *AppError {
	return &AppError{
		Type:       ErrorTypeNotFound,
		Message:    message,
		StatusCode: http.StatusNotFound,
	}
}

func NewUnauthorizedError(message string) *AppError {
	return &AppError{
		Type:       ErrorTypeUnauthorized,
		Message:    message,
		StatusCode: http.StatusUnauthorized,
	}
}

func NewExternalError(service, message string) *AppError {
	return &AppError{
		Type:       ErrorTypeExternal,
		Message:    fmt.Sprintf("%s service error: %s", service, message),
		StatusCode: http.StatusBadGateway,
		Context:    map[string]interface{}{"service": service},
	}
}

func NewInternalError(message string) *AppError {
	return &AppError{
		Type:       ErrorTypeInternal,
		Message:    message,
		StatusCode: http.StatusInternalServerError,
	}
}

func NewRateLimitError(service string) *AppError {
	return &AppError{
		Type:       ErrorTypeRateLimit,
		Message:    fmt.Sprintf("Rate limit exceeded for %s", service),
		StatusCode: http.StatusTooManyRequests,
		Context:    map[string]interface{}{"service": service},
	}
}

func NewTimeoutError(service string, timeout string) *AppError {
	return &AppError{
		Type:       ErrorTypeTimeout,
		Message:    fmt.Sprintf("Timeout calling %s after %s", service, timeout),
		StatusCode: http.StatusGatewayTimeout,
		Context:    map[string]interface{}{"service": service, "timeout": timeout},
	}
}

func NewUnavailableError(service string) *AppError {
	return &AppError{
		Type:       ErrorTypeUnavailable,
		Message:    fmt.Sprintf("Service %s is unavailable", service),
		StatusCode: http.StatusServiceUnavailable,
		Context:    map[string]interface{}{"service": service},
	}
}

// IsAppError checks if an error is an AppError
func IsAppError(err error) bool {
	_, ok := err.(*AppError)
	return ok
}

// AsAppError attempts to cast an error to AppError
func AsAppError(err error) (*AppError, bool) {
	appErr, ok := err.(*AppError)
	return appErr, ok
}

// WrapError wraps an error as an internal AppError
func WrapError(err error, message string) *AppError {
	return &AppError{
		Type:       ErrorTypeInternal,
		Message:    message,
		StatusCode: http.StatusInternalServerError,
		Cause:      err,
	}
}
