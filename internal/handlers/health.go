package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/igorsal/pr-documentator/internal/interfaces"
)

type HealthHandler struct {
	logger  interfaces.Logger
	metrics interfaces.MetricsCollector
}

type HealthResponse struct {
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
	Version   string `json:"version"`
}

// NewHealthHandler creates a new health handler
func NewHealthHandler(logger interfaces.Logger, metrics interfaces.MetricsCollector) *HealthHandler {
	return &HealthHandler{
		logger:  logger,
		metrics: metrics,
	}
}

// Handle processes health check requests
func (h *HealthHandler) Handle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.logger.Warn("Invalid method for health endpoint", "method", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	response := HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Version:   "1.0.0",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error("Failed to encode health response", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	h.logger.Debug("Health check completed successfully")
}