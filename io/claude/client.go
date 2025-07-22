package claude

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/sony/gobreaker"

	"github.com/igorsal/pr-documentator/internal/config"
	"github.com/igorsal/pr-documentator/internal/interfaces"
	"github.com/igorsal/pr-documentator/internal/models"
	pkgerrors "github.com/igorsal/pr-documentator/pkg/errors"
)

type Client struct {
	httpClient     *resty.Client
	config         config.ClaudeConfig
	logger         interfaces.Logger
	circuitBreaker interfaces.CircuitBreaker
	metrics        interfaces.MetricsCollector
}

// NewClient creates a new Claude API client with circuit breaker and metrics
func NewClient(cfg config.ClaudeConfig, logger interfaces.Logger, metrics interfaces.MetricsCollector) *Client {
	// Configure Resty client
	client := resty.New().
		SetTimeout(cfg.Timeout).
		SetRetryCount(3).
		SetRetryWaitTime(1*time.Second).
		SetRetryMaxWaitTime(5*time.Second).
		SetHeader("Content-Type", "application/json").
		SetHeader("x-api-key", cfg.APIKey).
		SetHeader("anthropic-version", "2023-06-01").
		SetBaseURL(cfg.BaseURL)

	// Configure circuit breaker
	cb := gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:        "claude-api",
		MaxRequests: 3,
		Interval:    30 * time.Second,
		Timeout:     60 * time.Second,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= 3
		},
		OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
			logger.Info("Claude API circuit breaker state changed",
				"name", name,
				"from", from.String(),
				"to", to.String(),
			)
		},
	})

	// Wrap circuit breaker in interface
	cbWrapper := &circuitBreakerWrapper{cb: cb}

	return &Client{
		httpClient:     client,
		config:         cfg,
		logger:         logger,
		circuitBreaker: cbWrapper,
		metrics:        metrics,
	}
}

// circuitBreakerWrapper implements interfaces.CircuitBreaker
type circuitBreakerWrapper struct {
	cb *gobreaker.CircuitBreaker
}

func (w *circuitBreakerWrapper) Execute(req func() (interface{}, error)) (interface{}, error) {
	return w.cb.Execute(req)
}

func (w *circuitBreakerWrapper) Name() string {
	return w.cb.Name()
}

func (w *circuitBreakerWrapper) State() string {
	return w.cb.State().String()
}

// AnalyzePR analyzes a pull request using Claude with function calling, circuit breaker, and metrics
func (c *Client) AnalyzePR(ctx context.Context, req models.AnalysisRequest) (*models.AnalysisResponse, error) {
	startTime := time.Now()
	labels := map[string]string{
		"service":    "claude",
		"operation":  "analyze_pr",
		"repository": req.Repository.FullName,
	}

	c.logger.Info("Starting PR analysis with Claude", 
		"pr_number", req.PullRequest.Number, 
		"repo", req.Repository.FullName,
		"circuit_breaker_state", c.circuitBreaker.State(),
	)

	// Execute with circuit breaker
	result, err := c.circuitBreaker.Execute(func() (interface{}, error) {
		return c.executeAnalysis(ctx, req)
	})

	// Record metrics
	duration := time.Since(startTime).Seconds()
	c.metrics.RecordDuration("claude_request_duration_seconds", duration, labels)

	if err != nil {
		labels["status"] = "error"
		c.metrics.IncrementCounter("claude_requests_total", labels)
		
		// Classify error type
		if gobreaker.StateOpen == c.circuitBreaker.(*circuitBreakerWrapper).cb.State() {
			c.logger.Error("Claude API circuit breaker open", err, 
				"pr_number", req.PullRequest.Number,
				"state", c.circuitBreaker.State(),
			)
			return nil, pkgerrors.NewUnavailableError("claude").WithCause(err)
		}

		c.logger.Error("Failed to analyze PR with Claude", err, "pr_number", req.PullRequest.Number)
		return nil, err
	}

	labels["status"] = "success"
	c.metrics.IncrementCounter("claude_requests_total", labels)

	analysisResp := result.(*models.AnalysisResponse)
	
	c.logger.Info("Successfully analyzed PR with Claude", 
		"pr_number", req.PullRequest.Number,
		"new_routes", len(analysisResp.NewRoutes),
		"modified_routes", len(analysisResp.ModifiedRoutes),
		"deleted_routes", len(analysisResp.DeletedRoutes),
		"confidence", analysisResp.Confidence,
		"duration_ms", duration*1000,
	)

	return analysisResp, nil
}

// executeAnalysis performs the actual Claude API call
func (c *Client) executeAnalysis(ctx context.Context, req models.AnalysisRequest) (*models.AnalysisResponse, error) {
	prompt := buildAnalysisPrompt(req)
	analysisToolSchema := buildAnalysisToolSchema()
	
	claudeReq := ClaudeRequest{
		Model:     c.config.Model,
		MaxTokens: c.config.MaxTokens,
		Messages: []Message{
			{
				Role:    "user",
				Content: prompt,
			},
		},
		System: systemPrompt,
		Tools: []Tool{analysisToolSchema},
		ToolChoice: map[string]interface{}{
			"type": "tool",
			"name": "analyze_api_changes",
		},
	}

	var claudeResp ClaudeResponse
	resp, err := c.httpClient.R().
		SetContext(ctx).
		SetBody(claudeReq).
		SetResult(&claudeResp).
		Post("/v1/messages")

	if err != nil {
		return nil, pkgerrors.NewExternalError("claude", err.Error()).WithCause(err)
	}

	if resp.IsError() {
		errorMsg := fmt.Sprintf("HTTP %d: %s", resp.StatusCode(), string(resp.Body()))
		
		// Handle specific error cases
		switch resp.StatusCode() {
		case 401:
			return nil, pkgerrors.NewUnauthorizedError("Invalid Claude API key")
		case 429:
			return nil, pkgerrors.NewRateLimitError("claude")
		case 500, 502, 503, 504:
			return nil, pkgerrors.NewUnavailableError("claude").WithContext("status_code", resp.StatusCode())
		default:
			return nil, pkgerrors.NewExternalError("claude", errorMsg)
		}
	}

	if len(claudeResp.Content) == 0 {
		return nil, pkgerrors.NewExternalError("claude", "empty response content")
	}

	// Find the tool use in the response
	var toolUse *Content
	for _, content := range claudeResp.Content {
		if content.Type == "tool_use" && content.Name == "analyze_api_changes" {
			toolUse = &content
			break
		}
	}

	if toolUse == nil {
		return nil, pkgerrors.NewExternalError("claude", "no tool use found in response")
	}

	// Convert the tool input to our analysis response
	analysisResp, err := c.convertToolInputToAnalysis(toolUse.Input)
	if err != nil {
		return nil, pkgerrors.WrapError(err, "failed to convert Claude response to analysis")
	}

	return analysisResp, nil
}

// Remove obsolete function - now using Resty in executeAnalysis

func buildAnalysisPrompt(req models.AnalysisRequest) string {
	return fmt.Sprintf(`
Analyze the following GitHub Pull Request for API changes and provide a structured response.

**Pull Request Details:**
- Title: %s
- Description: %s
- Repository: %s
- Number: %d
- Diff URL: %s

**Instructions:**
1. Analyze the diff/changes for new API routes, modifications to existing routes, or deleted routes
2. Look for changes in HTTP methods, endpoints, request/response payloads, headers, and parameters
3. Identify route descriptions, parameter types, and example values where possible
4. Provide confidence score (0-1) for your analysis

**Required JSON Response Format:**
{
  "new_routes": [
    {
      "method": "POST",
      "path": "/api/v1/users",
      "description": "Create a new user",
      "parameters": [
        {
          "name": "name",
          "in": "body",
          "type": "string",
          "required": true,
          "description": "User's full name"
        }
      ],
      "request_body": {
        "name": "string",
        "email": "string"
      },
      "response": {
        "id": "string",
        "name": "string",
        "email": "string"
      },
      "headers": [
        {
          "name": "Content-Type",
          "required": true,
          "description": "Must be application/json"
        }
      ]
    }
  ],
  "modified_routes": [],
  "deleted_routes": [],
  "summary": "Brief summary of changes",
  "confidence": 0.95
}

**Analysis Context:**
%s
`, req.PullRequest.Title, req.PullRequest.Body, req.Repository.FullName, req.PullRequest.Number, req.PullRequest.DiffURL, req.Diff)
}

// buildAnalysisToolSchema creates the JSON schema for the analysis tool
func buildAnalysisToolSchema() Tool {
	return Tool{
		Name:        "analyze_api_changes",
		Description: "Analyze GitHub Pull Request diffs to identify API route changes and return structured data about new, modified, or deleted endpoints",
		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]Property{
				"new_routes": {
					Type:        "array",
					Description: "Array of new API routes found in the PR",
					Items: &Property{
						Type: "object",
						Properties: map[string]Property{
							"method":      {Type: "string", Description: "HTTP method (GET, POST, PUT, DELETE, etc.)"},
							"path":        {Type: "string", Description: "API endpoint path (e.g., /api/v1/users)"},
							"description": {Type: "string", Description: "Description of what this endpoint does"},
							"parameters": {
								Type: "array",
								Items: &Property{
									Type: "object",
									Properties: map[string]Property{
										"name":        {Type: "string", Description: "Parameter name"},
										"in":          {Type: "string", Description: "Parameter location (query, path, header, body)"},
										"type":        {Type: "string", Description: "Parameter type (string, number, boolean, etc.)"},
										"required":    {Type: "boolean", Description: "Whether parameter is required"},
										"description": {Type: "string", Description: "Parameter description"},
									},
								},
							},
							"request_body": {Type: "object", Description: "Request body schema"},
							"response":     {Type: "object", Description: "Response body schema"},
						},
					},
				},
				"modified_routes": {
					Type:        "array",
					Description: "Array of modified API routes",
					Items: &Property{
						Type: "object",
						Properties: map[string]Property{
							"method":      {Type: "string", Description: "HTTP method"},
							"path":        {Type: "string", Description: "API endpoint path"},
							"description": {Type: "string", Description: "Description of changes made"},
							"request_body": {Type: "object", Description: "Updated request body schema"},
							"response":     {Type: "object", Description: "Updated response body schema"},
						},
					},
				},
				"deleted_routes": {
					Type:        "array",
					Description: "Array of deleted or deprecated API routes",
					Items: &Property{
						Type: "object",
						Properties: map[string]Property{
							"method": {Type: "string", Description: "HTTP method"},
							"path":   {Type: "string", Description: "API endpoint path"},
							"reason": {Type: "string", Description: "Reason for deletion/deprecation"},
						},
					},
				},
				"summary": {
					Type:        "string",
					Description: "Brief summary of all API changes found in this PR",
				},
				"confidence": {
					Type:        "number",
					Description: "Confidence score between 0 and 1 for the analysis accuracy",
				},
			},
			Required: []string{"new_routes", "modified_routes", "deleted_routes", "summary", "confidence"},
		},
	}
}

// convertToolInputToAnalysis converts Claude's tool input to our AnalysisResponse
func (c *Client) convertToolInputToAnalysis(input map[string]interface{}) (*models.AnalysisResponse, error) {
	// Marshal and unmarshal to convert to our struct
	jsonData, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal tool input: %w", err)
	}

	var analysisResp models.AnalysisResponse
	if err := json.Unmarshal(jsonData, &analysisResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal to AnalysisResponse: %w", err)
	}

	return &analysisResp, nil
}

const systemPrompt = `You are an expert API documentation analyst. Your role is to analyze GitHub Pull Request diffs and identify changes to REST API endpoints.

Key responsibilities:
1. Identify new API routes being added
2. Detect modifications to existing routes (changes in parameters, request/response format, etc.)
3. Find deleted or deprecated routes
4. Extract detailed information about each route including methods, paths, parameters, request/response schemas
5. Provide confidence scores for your analysis

You must use the analyze_api_changes tool to return structured data. Be thorough but precise in your analysis.

Guidelines:
- Look for HTTP route definitions (app.get, router.post, @RequestMapping, etc.)
- Identify request/response payload structures
- Note parameter changes (query params, path params, headers)
- Detect middleware changes that affect API behavior
- Consider both code and documentation changes`