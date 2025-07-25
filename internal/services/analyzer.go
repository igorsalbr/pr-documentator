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

	// 	diff := `diff --git a/.gitignore b/.gitignore
	// index a95b6bc..c2968a5 100644
	// --- a/.gitignore
	// +++ b/.gitignore
	// @@ -69,4 +69,5 @@ coverage.out

	//  # Build artifacts
	//  *.tar.gz
	// -*.zip
	// \ No newline at end of file
	// +*.zip
	// +working_workspace.txt
	// \ No newline at end of file
	// diff --git a/Makefile b/Makefile
	// index a905dd1..ff374e6 100644
	// --- a/Makefile
	// +++ b/Makefile
	// @@ -35,7 +35,7 @@ clean: ## Clean build artifacts and temporary files
	//  dev: gen-certs ## Run the application with hot reload (requires air: go install github.com/cosmtrek/air@latest)
	//  	@if ! command -v air >/dev/null 2>&1; then \
	//  		echo "📦 Installing air for hot reload..."; \
	// -		go install github.com/cosmtrek/air@latest; \
	// +		go install github.com/air-verse/air@latest; \
	//  	fi
	//  	@echo "🚀 Starting development server with hot reload..."
	//  	@air -c .air.toml
	// @@ -105,7 +105,7 @@ docker-run: ## Run application in Docker
	//  # Installation commands
	//  install-tools: ## Install development tools
	//  	@echo "🛠️  Installing development tools..."
	// -	@go install github.com/cosmtrek/air@latest
	// +	@go install github.com/air-verse/air@latest
	//  	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	//  	@echo "✅ Development tools installed"

	// diff --git a/README.md b/README.md
	// index cb2b823..dd388c6 100644
	// --- a/README.md
	// +++ b/README.md
	// @@ -1067,7 +1067,7 @@ echo "🎉 Configuração válida!"
	//  - **⚡ [Circuit Breaker](https://github.com/sony/gobreaker)**: Proteção contra falhas em cascata
	//  - **📊 [Prometheus](https://github.com/prometheus/client_golang)**: Métricas e observabilidade
	//  - **🏗️ [Dependency Injection](https://github.com/igorsal/pr-documentator/tree/main/internal/interfaces)**: Interfaces para arquitetura limpa
	// -- **⚡ [Air](https://github.com/cosmtrek/air)**: Hot reload para desenvolvimento Go
	// +- **⚡ [Air](https://github.com/air-verse/air)**: Hot reload para desenvolvimento Go
	//  - **🧪 [Testify](https://github.com/stretchr/testify)**: Framework de testes

	//  ### Melhores Práticas
	// diff --git a/cmd/server/main.go b/cmd/server/main.go
	// index 3a5e6c6..c5e586e 100644
	// --- a/cmd/server/main.go
	// +++ b/cmd/server/main.go
	// @@ -41,7 +41,7 @@ func main() {
	//  		os.Exit(1)
	//  	}

	// -	app.logger.Info("Starting PR Documentator service",
	// +	app.logger.Info("Starting PR Documentator service",
	//  		"version", "2.0.0",
	//  		"environment", os.Getenv("ENVIRONMENT"),
	//  	)
	// @@ -93,6 +93,7 @@ func (app *Application) setupServer() {
	//  	// Initialize handlers
	//  	healthHandler := handlers.NewHealthHandler(app.logger, app.metrics)
	//  	prAnalyzerHandler := handlers.NewPRAnalyzerHandler(app.analyzerService, app.logger, app.metrics)
	// +	testChange := handlers.NewTestHandler(app.logger, app.metrics)

	//  	// Setup router
	//  	router := mux.NewRouter()
	// @@ -107,6 +108,7 @@ func (app *Application) setupServer() {
	//  	// Public endpoints
	//  	router.HandleFunc("/health", healthHandler.Handle).Methods("GET")
	//  	router.Handle("/metrics", promhttp.Handler()).Methods("GET")
	// +	router.HandleFunc("/test", testChange.Handle).Methods("GET")

	//  	// Protected endpoints
	//  	prRouter := router.PathPrefix("").Subrouter()
	// @@ -205,4 +207,4 @@ func (app *Application) gracefulShutdown() error {
	//  		}
	//  		return fmt.Errorf("shutdown timeout exceeded")
	//  	}
	// -}
	// \ No newline at end of file
	// +}
	// diff --git a/internal/handlers/test.go b/internal/handlers/test.go
	// new file mode 100644
	// index 0000000..4e01954
	// --- /dev/null
	// +++ b/internal/handlers/test.go
	// @@ -0,0 +1,54 @@
	// +package handlers
	// +
	// +import (
	// +	"encoding/json"
	// +	"net/http"
	// +	"time"
	// +
	// +	"github.com/igorsal/pr-documentator/internal/interfaces"
	// +)
	// +
	// +type TestHandler struct {
	// +	logger  interfaces.Logger
	// +	metrics interfaces.MetricsCollector
	// +}
	// +
	// +type TestResponse struct {
	// +	Status    string json:"status"
	// +	Timestamp string json:"timestamp"
	// +	Version   string json:"version"
	// +}
	// +
	// +// NewTestHandler creates a new Test handler
	// +func NewTestHandler(logger interfaces.Logger, metrics interfaces.MetricsCollector) *TestHandler {
	// +	return &TestHandler{
	// +		logger:  logger,
	// +		metrics: metrics,
	// +	}
	// +}
	// +
	// +// Handle processes Test check requests
	// +func (h *TestHandler) Handle(w http.ResponseWriter, r *http.Request) {
	// +	if r.Method != http.MethodGet {
	// +		h.logger.Warn("Invalid method for Test endpoint", "method", r.Method)
	// +		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	// +		return
	// +	}
	// +
	// +	response := TestResponse{
	// +		Status:    "Testy",
	// +		Timestamp: time.Now().UTC().Format(time.RFC3339),
	// +		Version:   "1.0.0",
	// +	}
	// +
	// +	w.Header().Set("Content-Type", "application/json")
	// +	w.WriteHeader(http.StatusOK)
	// +
	// +	if err := json.NewEncoder(w).Encode(response); err != nil {
	// +		h.logger.Error("Failed to encode Test response", err)
	// +		http.Error(w, "Internal server error", http.StatusInternalServerError)
	// +		return
	// +	}
	// +
	// +	h.logger.Debug("Test check completed successfully")
	// +}`

	// Create analysis request
	analysisReq := models.AnalysisRequest{
		PullRequest: payload.PullRequest,
		Repository:  payload.Repository,
		Diff:        diff,
	}

	// Get existing collection context for better analysis
	existingCollection, err := s.postmanClient.GetCollection(ctx)
	if err != nil {
		s.logger.Warn("Failed to get existing collection context", "error", err)
		// Continue without context - don't fail the entire operation
	}

	// Add collection context to analysis request
	if existingCollection != nil {
		analysisReq.ExistingRoutes = s.extractRoutesFromCollection(existingCollection)
		s.logger.Info("Added collection context", "existing_routes", len(analysisReq.ExistingRoutes))
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

// extractRoutesFromCollection extracts existing routes from Postman collection for context
func (s *AnalyzerService) extractRoutesFromCollection(collection *models.PostmanCollection) []models.ExistingRoute {
	var routes []models.ExistingRoute
	
	// Process items recursively to handle folders
	s.extractRoutesFromItems(collection.Items, []string{}, &routes)
	
	return routes
}

// extractRoutesFromItems recursively extracts routes from collection items
func (s *AnalyzerService) extractRoutesFromItems(items []models.PostmanItem, folderPath []string, routes *[]models.ExistingRoute) {
	for _, item := range items {
		if item.Request != nil {
			// This is a request item
			route := models.ExistingRoute{
				Method:      item.Request.Method,
				Path:        s.extractPathFromURL(item.Request.URL),
				Name:        item.Name,
				Description: item.Description,
				FolderPath:  append([]string{}, folderPath...), // Copy slice
			}
			*routes = append(*routes, route)
		} else if len(item.Items) > 0 {
			// This is a folder - recurse into it
			newFolderPath := append(folderPath, item.Name)
			s.extractRoutesFromItems(item.Items, newFolderPath, routes)
		}
	}
}

// extractPathFromURL extracts the clean path from Postman URL structure
func (s *AnalyzerService) extractPathFromURL(url models.PostmanURL) string {
	if url.Raw != "" {
		// Remove {{baseUrl}} and clean up the path
		path := url.Raw
		if len(path) > 0 && path[0:11] == "{{baseUrl}}" {
			path = path[11:]
		}
		return path
	}
	
	// Fallback to constructing from path segments
	if len(url.Path) > 1 {
		// Skip {{baseUrl}} if present
		pathSegments := url.Path
		if len(pathSegments) > 0 && pathSegments[0] == "{{baseUrl}}" {
			pathSegments = pathSegments[1:]
		}
		if len(pathSegments) > 0 {
			return "/" + pathSegments[0]
		}
	}
	
	return "/"
}
