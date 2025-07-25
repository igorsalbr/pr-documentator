package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/igorsal/pr-documentator/internal/interfaces"
	"github.com/igorsal/pr-documentator/internal/services"
	pkgerrors "github.com/igorsal/pr-documentator/pkg/errors"
)

type AuthHandler struct {
	tokenManager interfaces.TokenManager
	logger       interfaces.Logger
	metrics      interfaces.MetricsCollector
	validator    *validator.Validate
}

type AuthRequest struct {
	ClaudeAPIKey        string `json:"claude_api_key" validate:"required"`
	PostmanAPIKey       string `json:"postman_api_key" validate:"required"`
	PostmanWorkspaceID  string `json:"postman_workspace_id" validate:"required"`
	PostmanCollectionID string `json:"postman_collection_id" validate:"required"`
}

type AuthResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	Message   string    `json:"message"`
}

func NewAuthHandler(tokenManager interfaces.TokenManager, logger interfaces.Logger, metrics interfaces.MetricsCollector) *AuthHandler {
	return &AuthHandler{
		tokenManager: tokenManager,
		logger:       logger,
		metrics:      metrics,
		validator:    validator.New(),
	}
}

func (h *AuthHandler) Handle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.writeErrorResponse(w, pkgerrors.NewValidationError("method not allowed"), http.StatusMethodNotAllowed)
		return
	}

	var req AuthRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, MaxBodySize)).Decode(&req); err != nil {
		h.logger.Error("Failed to decode auth request", err)
		h.writeErrorResponse(w, pkgerrors.NewValidationError("invalid request body"), http.StatusBadRequest)
		return
	}

	if err := h.validator.Struct(req); err != nil {
		h.logger.Error("Request validation failed", err)
		h.writeErrorResponse(w, pkgerrors.NewValidationError("validation failed: "+err.Error()), http.StatusBadRequest)
		return
	}

	token, err := h.tokenManager.CreateSession(
		req.ClaudeAPIKey,
		req.PostmanAPIKey,
		req.PostmanWorkspaceID,
		req.PostmanCollectionID,
	)
	if err != nil {
		h.logger.Error("Failed to create session", err)
		h.writeErrorResponse(w, pkgerrors.NewInternalError("failed to create session"), http.StatusInternalServerError)
		return
	}

	expiresAt := time.Now().Add(services.TokenTTL)
	response := AuthResponse{
		Token:     token,
		ExpiresAt: expiresAt,
		Message:   "Session created successfully. Use this token for API requests.",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error("Failed to encode auth response", err)
	}

	h.logger.Info("User session created", "token", token[:8]+"...", "expires_at", expiresAt)
}

func (h *AuthHandler) writeErrorResponse(w http.ResponseWriter, err error, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	response := map[string]string{
		"error": err.Error(),
	}

	if encErr := json.NewEncoder(w).Encode(response); encErr != nil {
		h.logger.Error("Failed to encode error response", encErr)
	}
}