package interfaces

import (
	"context"

	"github.com/igorsal/pr-documentator/internal/models"
)

// ClaudeClient defines the interface for Claude AI integration
type ClaudeClient interface {
	AnalyzePR(ctx context.Context, req models.AnalysisRequest) (*models.AnalysisResponse, error)
}

// PostmanClient defines the interface for Postman integration
type PostmanClient interface {
	UpdateCollection(ctx context.Context, analysisResp *models.AnalysisResponse) (*models.PostmanUpdate, error)
	GetCollection(ctx context.Context) (*models.PostmanCollection, error)
}

// AnalyzerService defines the interface for PR analysis orchestration
type AnalyzerService interface {
	AnalyzePR(ctx context.Context, payload models.GitHubPRPayload) (*models.AnalysisResponse, error)
}

// Logger defines the logging interface
type Logger interface {
	Debug(msg string, fields ...interface{})
	Info(msg string, fields ...interface{})
	Warn(msg string, fields ...interface{})
	Error(msg string, err error, fields ...interface{})
	Fatal(msg string, err error, fields ...interface{})
}

// MetricsCollector defines the interface for collecting metrics
type MetricsCollector interface {
	IncrementCounter(name string, labels map[string]string)
	RecordDuration(name string, duration float64, labels map[string]string)
	SetGauge(name string, value float64, labels map[string]string)
}

// CircuitBreaker defines the interface for circuit breaker pattern
type CircuitBreaker interface {
	Execute(req func() (interface{}, error)) (interface{}, error)
	Name() string
	State() string
}

// HTTPClient defines the interface for HTTP operations
type HTTPClient interface {
	Get(ctx context.Context, url string) (*HTTPResponse, error)
	Post(ctx context.Context, url string, body interface{}) (*HTTPResponse, error)
	Put(ctx context.Context, url string, body interface{}) (*HTTPResponse, error)
	Delete(ctx context.Context, url string) (*HTTPResponse, error)
}

// HTTPResponse represents an HTTP response
type HTTPResponse struct {
	StatusCode int
	Body       []byte
	Headers    map[string]string
}

// ConfigProvider defines the interface for configuration management
type ConfigProvider interface {
	GetString(key string) string
	GetInt(key string) int
	GetBool(key string) bool
	GetDuration(key string) string
	Validate() error
}
