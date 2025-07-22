package services

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/igorsal/pr-documentator/internal/interfaces"
	"github.com/igorsal/pr-documentator/internal/models"
)

type AnalyzerService struct {
	claudeClient  interfaces.ClaudeClient
	postmanClient interfaces.PostmanClient
	logger        interfaces.Logger
	metrics       interfaces.MetricsCollector
}

// NewAnalyzerService creates a new analyzer service
func NewAnalyzerService(claudeClient interfaces.ClaudeClient, postmanClient interfaces.PostmanClient, logger interfaces.Logger, metrics interfaces.MetricsCollector) *AnalyzerService {
	return &AnalyzerService{
		claudeClient:  claudeClient,
		postmanClient: postmanClient,
		logger:        logger,
		metrics:       metrics,
	}
}

// AnalyzePR analyzes a pull request and updates Postman documentation
func (s *AnalyzerService) AnalyzePR(ctx context.Context, payload models.GitHubPRPayload) (*models.AnalysisResponse, error) {
	s.logger.Info("Starting PR analysis", 
		"pr_number", payload.PullRequest.Number,
		"repo", payload.Repository.FullName,
		"action", payload.Action,
	)

	// Only process opened, synchronize, or reopened PRs
	if !s.shouldProcessAction(payload.Action) {
		s.logger.Info("Skipping PR action", "action", payload.Action)
		return &models.AnalysisResponse{
			Summary: fmt.Sprintf("Skipped action: %s", payload.Action),
		}, nil
	}

	// Fetch the PR diff
	diff, err := s.fetchPRDiff(ctx, payload.PullRequest.DiffURL)
	if err != nil {
		s.logger.Error("Failed to fetch PR diff", err, "diff_url", payload.PullRequest.DiffURL)
		return nil, fmt.Errorf("failed to fetch PR diff: %w", err)
	}

	// Create analysis request
	analysisReq := models.AnalysisRequest{
		PullRequest: payload.PullRequest,
		Repository:  payload.Repository,
		Diff:        diff,
	}

	// Analyze with Claude
	analysisResp, err := s.claudeClient.AnalyzePR(ctx, analysisReq)
	if err != nil {
		s.logger.Error("Failed to analyze PR with Claude", err, "pr_number", payload.PullRequest.Number)
		return nil, fmt.Errorf("claude analysis failed: %w", err)
	}

	// Only update Postman if there are changes
	if s.hasAPIChanges(analysisResp) {
		s.logger.Info("API changes detected, updating Postman collection", 
			"new_routes", len(analysisResp.NewRoutes),
			"modified_routes", len(analysisResp.ModifiedRoutes),
			"deleted_routes", len(analysisResp.DeletedRoutes),
		)

		postmanUpdate, err := s.postmanClient.UpdateCollection(ctx, analysisResp)
		if err != nil {
			s.logger.Error("Failed to update Postman collection", err, "pr_number", payload.PullRequest.Number)
			// Don't fail the entire operation if Postman update fails
			analysisResp.PostmanUpdate = models.PostmanUpdate{
				Status:       "error",
				ErrorMessage: err.Error(),
				UpdatedAt:    time.Now().Format(time.RFC3339),
			}
		} else {
			analysisResp.PostmanUpdate = *postmanUpdate
		}
	} else {
		s.logger.Info("No API changes detected, skipping Postman update")
		analysisResp.PostmanUpdate = models.PostmanUpdate{
			Status:    "skipped",
			UpdatedAt: time.Now().Format(time.RFC3339),
		}
	}

	s.logger.Info("PR analysis completed successfully", 
		"pr_number", payload.PullRequest.Number,
		"confidence", analysisResp.Confidence,
		"postman_status", analysisResp.PostmanUpdate.Status,
	)

	return analysisResp, nil
}

func (s *AnalyzerService) shouldProcessAction(action string) bool {
	processableActions := []string{"opened", "synchronize", "reopened"}
	for _, a := range processableActions {
		if a == action {
			return true
		}
	}
	return false
}

func (s *AnalyzerService) fetchPRDiff(ctx context.Context, diffURL string) (string, error) {
	if diffURL == "" {
		return "", fmt.Errorf("diff URL is empty")
	}

	s.logger.Debug("Fetching PR diff", "diff_url", diffURL)

	req, err := http.NewRequestWithContext(ctx, "GET", diffURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// GitHub returns plain text diff
	req.Header.Set("Accept", "text/plain")

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to fetch diff, status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	diff := string(body)
	s.logger.Debug("Successfully fetched PR diff", 
		"diff_size_bytes", len(body),
		"diff_size_chars", len(diff),
	)

	return diff, nil
}

func (s *AnalyzerService) hasAPIChanges(resp *models.AnalysisResponse) bool {
	return len(resp.NewRoutes) > 0 || len(resp.ModifiedRoutes) > 0 || len(resp.DeletedRoutes) > 0
}