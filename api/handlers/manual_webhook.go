package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/igorsal/pr-documentator/internal/interfaces"
	"github.com/igorsal/pr-documentator/internal/models"
	pkgerrors "github.com/igorsal/pr-documentator/pkg/errors"
)

const (
	MaxBodySize = 10 * 1024 * 1024 // 10MB max
)

type ManualWebhookHandler struct {
	analyzer interfaces.AnalyzerService
	logger   interfaces.Logger
	metrics  interfaces.MetricsCollector
}

type ManualWebhookRequest struct {
	Diff string `json:"diff" validate:"required"`
}

func NewManualWebhookHandler(analyzer interfaces.AnalyzerService, logger interfaces.Logger, metrics interfaces.MetricsCollector) *ManualWebhookHandler {
	return &ManualWebhookHandler{
		analyzer: analyzer,
		logger:   logger,
		metrics:  metrics,
	}
}

func (h *ManualWebhookHandler) Handle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.writeErrorResponse(w, pkgerrors.NewValidationError("method not allowed"), http.StatusMethodNotAllowed)
		return
	}

	// Parse request body
	var req ManualWebhookRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, MaxBodySize)).Decode(&req); err != nil {
		h.logger.Error("Failed to decode manual webhook request", err)
		h.writeErrorResponse(w, pkgerrors.NewValidationError("invalid request body"), http.StatusBadRequest)
		return
	}

	if req.Diff == "" {
		h.writeErrorResponse(w, pkgerrors.NewValidationError("diff field is required"), http.StatusBadRequest)
		return
	}

	// Create a mock payload for manual analysis
	payload := models.GitHubPRPayload{
		Action: "opened",
		Repository: models.Repository{
			FullName: "manual/analysis",
		},
		PullRequest: models.PullRequest{
			Number:  1,
			Title:   "Manual Analysis",
			Body:    "Manual analysis triggered via webhook",
			DiffURL: "manual",
		},
		Diff: req.Diff,
	}

	// Analyze the diff
	result, err := h.analyzer.AnalyzePR(r.Context(), payload)
	if err != nil {
		h.logger.Error("Failed to analyze manual diff", err)

		var statusCode int
		if appErr, ok := pkgerrors.AsAppError(err); ok {
			switch appErr.Type {
			case pkgerrors.ErrorTypeValidation:
				statusCode = http.StatusBadRequest
			case pkgerrors.ErrorTypeUnauthorized:
				statusCode = http.StatusUnauthorized
			case pkgerrors.ErrorTypeRateLimit:
				statusCode = http.StatusTooManyRequests
			case pkgerrors.ErrorTypeUnavailable:
				statusCode = http.StatusServiceUnavailable
			default:
				statusCode = http.StatusInternalServerError
			}
		} else {
			statusCode = http.StatusInternalServerError
		}

		h.writeErrorResponse(w, err, statusCode)
		return
	}

	// Return analysis result
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(result); err != nil {
		h.logger.Error("Failed to encode response", err)
	}

	h.logger.Info("Manual webhook analysis completed successfully",
		"new_routes", len(result.NewRoutes),
		"modified_routes", len(result.ModifiedRoutes),
		"confidence", result.Confidence,
	)
}

func (h *ManualWebhookHandler) writeErrorResponse(w http.ResponseWriter, err error, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	response := map[string]string{
		"error": err.Error(),
	}

	if encErr := json.NewEncoder(w).Encode(response); encErr != nil {
		h.logger.Error("Failed to encode error response", encErr)
	}
}
