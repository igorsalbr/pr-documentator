package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/igorsal/pr-documentator/internal/interfaces"
)

type TestHandler struct {
	logger  interfaces.Logger
	metrics interfaces.MetricsCollector
}

type TestResponse struct {
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
	Version   string `json:"version"`
}

// NewTestHandler creates a new Test handler
func NewTestHandler(logger interfaces.Logger, metrics interfaces.MetricsCollector) *TestHandler {
	return &TestHandler{
		logger:  logger,
		metrics: metrics,
	}
}

// Handle processes Test check requests
func (h *TestHandler) Handle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.logger.Warn("Invalid method for Test endpoint", "method", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	response := TestResponse{
		Status:    "Testy",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Version:   "1.0.0",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error("Failed to encode Test response", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	h.logger.Debug("Test check completed successfully")
}
