package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/igorsal/pr-documentator/internal/interfaces"
	"github.com/igorsal/pr-documentator/internal/models"
)

type PRAnalyzerHandler struct {
	analyzerService interfaces.AnalyzerService
	logger          interfaces.Logger
	metrics         interfaces.MetricsCollector
}

// NewPRAnalyzerHandler creates a new PR analyzer handler
func NewPRAnalyzerHandler(analyzerService interfaces.AnalyzerService, logger interfaces.Logger, metrics interfaces.MetricsCollector) *PRAnalyzerHandler {
	return &PRAnalyzerHandler{
		analyzerService: analyzerService,
		logger:          logger,
		metrics:         metrics,
	}
}

// Handle processes PR analysis requests
func (h *PRAnalyzerHandler) Handle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.logger.Warn("Invalid method for PR analyzer endpoint", "method", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Validate GitHub event header
	eventType := r.Header.Get("X-GitHub-Event")
	if eventType != "pull_request" {
		h.logger.Warn("Invalid GitHub event type", "event_type", eventType)
		http.Error(w, "Invalid event type", http.StatusBadRequest)
		return
	}

	// Parse the GitHub PR payload
	var payload models.GitHubPRPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		h.logger.Error("Failed to decode GitHub payload", err)
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	h.logger.Info("Received GitHub PR webhook",
		"pr_number", payload.PullRequest.Number,
		"repo", payload.Repository.FullName,
		"action", payload.Action,
		"sender", payload.Sender.Login,
	)

	// Analyze the PR
	analysisResp, err := h.analyzerService.AnalyzePR(r.Context(), payload)
	if err != nil {
		h.logger.Error("Failed to analyze PR", err,
			"pr_number", payload.PullRequest.Number,
			"repo", payload.Repository.FullName,
		)
		http.Error(w, "Analysis failed", http.StatusInternalServerError)
		return
	}

	// Return the analysis response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "success",
		"analysis":  analysisResp,
		"timestamp": payload.PullRequest.UpdatedAt,
	}); err != nil {
		h.logger.Error("Failed to encode analysis response", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}

	h.logger.Info("PR analysis completed successfully",
		"pr_number", payload.PullRequest.Number,
		"new_routes", len(analysisResp.NewRoutes),
		"modified_routes", len(analysisResp.ModifiedRoutes),
		"deleted_routes", len(analysisResp.DeletedRoutes),
		"postman_status", analysisResp.PostmanUpdate.Status,
	)
}
