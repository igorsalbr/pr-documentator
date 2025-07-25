package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/igorsal/pr-documentator/internal/config"
	"github.com/igorsal/pr-documentator/internal/interfaces"
	"github.com/igorsal/pr-documentator/internal/models"
	"github.com/igorsal/pr-documentator/internal/services"
	"github.com/igorsal/pr-documentator/io/claude"
	"github.com/igorsal/pr-documentator/io/postman"
	pkgerrors "github.com/igorsal/pr-documentator/pkg/errors"
)

type WebAnalyzeHandler struct {
	tokenManager interfaces.TokenManager
	logger       interfaces.Logger
	metrics      interfaces.MetricsCollector
	validator    *validator.Validate //nolint
}

type WebAnalyzeRequest struct {
	Diff string `json:"diff" validate:"required"`
}

func NewWebAnalyzeHandler(tokenManager interfaces.TokenManager, logger interfaces.Logger, metrics interfaces.MetricsCollector) *WebAnalyzeHandler {
	return &WebAnalyzeHandler{
		tokenManager: tokenManager,
		logger:       logger,
		metrics:      metrics,
		validator:    validator.New(), //nolint
	}
}

func (h *WebAnalyzeHandler) Handle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.writeErrorResponse(w, pkgerrors.NewValidationError("method not allowed"), http.StatusMethodNotAllowed)
		return
	}

	token := r.Header.Get("Authorization")
	if token == "" {
		h.writeErrorResponse(w, pkgerrors.NewUnauthorizedError("authorization token required"), http.StatusUnauthorized)
		return
	}

	// Remove "Bearer " prefix if present
	if len(token) > 7 && token[:7] == "Bearer " {
		token = token[7:]
	}

	session, exists := h.tokenManager.GetSession(token)
	if !exists {
		h.writeErrorResponse(w, pkgerrors.NewUnauthorizedError("invalid or expired token"), http.StatusUnauthorized)
		return
	}

	var req WebAnalyzeRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, MaxBodySize)).Decode(&req); err != nil {
		h.logger.Error("Failed to decode web analyze request", err)
		h.writeErrorResponse(w, pkgerrors.NewValidationError("invalid request body"), http.StatusBadRequest)
		return
	}

	if err := h.validator.Struct(req); err != nil {
		h.logger.Error("Request validation failed", err)
		h.writeErrorResponse(w, pkgerrors.NewValidationError("validation failed: "+err.Error()), http.StatusBadRequest)
		return
	}

	// Create clients with session credentials
	claudeConfig := config.ClaudeConfig{
		APIKey:    session.ClaudeAPIKey,
		Model:     "claude-3-sonnet-20240229",
		MaxTokens: 4096,
		BaseURL:   "https://api.anthropic.com",
		Timeout:   30 * time.Second,
	}
	claudeClient := claude.NewClient(claudeConfig, h.logger, h.metrics)

	postmanConfig := config.PostmanConfig{
		APIKey:       session.PostmanAPIKey,
		WorkspaceID:  session.PostmanWorkspaceID,
		CollectionID: session.PostmanCollectionID,
		BaseURL:      "https://api.postman.com",
		Timeout:      30 * time.Second,
	}
	postmanClient := postman.NewClient(postmanConfig, h.logger, h.metrics)

	// Create analyzer with user-specific clients
	analyzer := services.NewAnalyzerService(claudeClient, postmanClient, h.logger, h.metrics)

	// Create mock payload for analysis
	payload := models.GitHubPRPayload{
		Action: "opened",
		Repository: models.Repository{
			FullName: "web/analysis",
		},
		PullRequest: models.PullRequest{
			Number:  1,
			Title:   "Web Analysis",
			Body:    "Analysis triggered via web interface",
			DiffURL: "web",
		},
		Diff: req.Diff,
	}

	// Analyze the diff
	result, err := analyzer.AnalyzePR(r.Context(), payload)
	if err != nil {
		h.logger.Error("Failed to analyze web diff", err)

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

	h.logger.Info("Web analysis completed successfully",
		"token", token[:8]+"...",
		"new_routes", len(result.NewRoutes),
		"modified_routes", len(result.ModifiedRoutes),
		"confidence", result.Confidence,
	)
}

func (h *WebAnalyzeHandler) writeErrorResponse(w http.ResponseWriter, err error, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	response := map[string]string{
		"error": err.Error(),
	}

	if encErr := json.NewEncoder(w).Encode(response); encErr != nil {
		h.logger.Error("Failed to encode error response", encErr)
	}
}
