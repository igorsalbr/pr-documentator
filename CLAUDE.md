# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**PR Documentator** is a production-ready Go microservice that automatically analyzes GitHub Pull Requests using Claude AI to detect API changes and update Postman documentation. The service exposes an HTTPS endpoint that receives GitHub webhooks and intelligently processes code diffs to maintain up-to-date API documentation.

## Essential Commands

### Quick Start
```bash
# Setup and run (first time)
cp .env.example .env  # Configure your API keys
make gen-certs        # Generate HTTPS certificates
make dev              # Run with hot reload

# Build and deploy
make build            # Production build -> bin/pr-documentator
make docker-build     # Docker image build
```

### Development Workflow
```bash
make dev              # Hot reload development server
make run              # Direct execution without hot reload
make test             # Run all tests with coverage
make lint             # Code linting and formatting
make clean            # Clean build artifacts
```

### Essential Testing
```bash
make test             # All tests
make test-unit        # Unit tests only (fast)
make test-coverage    # Generate coverage report
go test -v ./internal/handlers  # Test specific package
```

### Deployment
```bash
make build            # Production binary -> bin/pr-documentator
docker-compose up -d  # Run with Docker
make gen-certs        # Generate SSL certificates for HTTPS
```

## Architecture & Code Structure

### Layered Architecture
```
┌─────────────────┐
│   HTTP Layer    │ -> cmd/server/ (main.go, routing)
├─────────────────┤
│  Handler Layer  │ -> internal/handlers/ (HTTP handlers)
├─────────────────┤
│  Service Layer  │ -> internal/services/ (business logic)
├─────────────────┤
│Integration Layer│ -> io/ (external API clients)
├─────────────────┤
│  Utility Layer  │ -> pkg/ (shared utilities)
└─────────────────┘
```

### Key Directories
- **`cmd/server/`**: Application entry point and server setup
- **`internal/config/`**: Environment variable management
- **`internal/handlers/`**: HTTP request handlers (`/health`, `/analyze-pr`)
- **`internal/models/`**: Data structures (GitHub, Analysis, Postman models)
- **`internal/services/`**: Core business logic (PR analysis orchestration)
- **`internal/middleware/`**: HTTP middleware (auth, logging, recovery, CORS)
- **`io/claude/`**: Claude API client integration
- **`io/postman/`**: Postman API client integration
- **`pkg/logger/`**: Structured logging with zerolog

### Critical Components

#### 1. PR Analysis Handler (`internal/handlers/pr_analyzer.go`)
- Validates GitHub webhook signatures using HMAC-SHA256
- Parses GitHub PR payloads
- Orchestrates analysis through service layer
- Returns structured JSON responses

#### 2. Analysis Service (`internal/services/analyzer.go`)
- Fetches PR diffs from GitHub
- Calls Claude API with structured prompts
- Updates Postman collections
- Handles errors gracefully

#### 3. Claude Integration (`io/claude/`)
- HTTP client with configurable timeouts
- Structured prompt building for API analysis
- JSON response parsing and validation
- Error handling and rate limiting

#### 4. Postman Integration (`io/postman/`)
- Collection retrieval and updating
- Route-to-Postman-item conversion
- Handles new, modified, and deprecated endpoints
- Maintains collection structure and metadata

## Environment Configuration

### Required Variables
```env
# Claude API (required)
CLAUDE_API_KEY=sk-ant-api03-your-key-here
CLAUDE_MODEL=claude-3-sonnet-20240229

# Postman API (required)
POSTMAN_API_KEY=PMAK-your-key-here
POSTMAN_WORKSPACE_ID=workspace-id
POSTMAN_COLLECTION_ID=collection-id

# Server Configuration
SERVER_PORT=8443
TLS_CERT_FILE=./certs/server.crt
TLS_KEY_FILE=./certs/server.key

# GitHub Webhook (optional but recommended)
GITHUB_WEBHOOK_SECRET=your-secret-here
```

### Optional Configuration
```env
# Logging
LOG_LEVEL=info          # debug, info, warn, error
LOG_FORMAT=json         # json, console

# Timeouts
CLAUDE_TIMEOUT=30s
POSTMAN_TIMEOUT=30s
SERVER_READ_TIMEOUT=15s
```

## Development Guidelines

### Adding New Features

#### 1. New HTTP Endpoint
```go
// 1. Create handler in internal/handlers/
type NewHandler struct {
    service *services.SomeService
    logger  *logger.Logger
}

// 2. Register route in cmd/server/main.go
router.HandleFunc("/new-endpoint", newHandler.Handle).Methods("POST")
```

#### 2. New External Integration
```go
// 1. Create client in io/service_name/
type Client struct {
    httpClient *http.Client
    config     config.ServiceConfig
    logger     *logger.Logger
}

// 2. Add config in internal/config/config.go
type ServiceConfig struct {
    APIKey  string
    BaseURL string
    Timeout time.Duration
}
```

#### 3. New Models
```go
// Add to internal/models/
type APIEndpoint struct {
    Method      string                 `json:"method"`
    Path        string                 `json:"path"`
    Description string                 `json:"description"`
    RequestBody map[string]interface{} `json:"request_body,omitempty"`
}
```

### Security & Best Practices

#### HTTPS & Certificates
- Service **must** run on HTTPS (port 8443)
- Use `make gen-certs` for development certificates
- Production should use valid SSL certificates
- Never commit certificate files

#### Webhook Security
- GitHub webhook signatures are validated using HMAC-SHA256
- Secret should be cryptographically secure (32+ characters)
- Validation happens in middleware before handler execution

#### Logging & Monitoring
```go
// Structured logging examples
logger.Info("PR analysis started", "pr_number", prNumber, "repo", repoName)
logger.Error("API request failed", err, "url", requestURL, "status_code", statusCode)
logger.Debug("Processing routes", "new_count", len(newRoutes), "modified_count", len(modifiedRoutes))
```

### Testing Approach

#### Unit Tests
- Test handlers with `httptest`
- Mock external dependencies
- Focus on business logic validation
- Use testify for assertions

#### Integration Tests
- Test full request flow
- Use real but sandboxed environments
- Validate external API interactions
- Test webhook signature validation

#### Example Test Structure
```go
func TestPRAnalyzerHandler(t *testing.T) {
    // Arrange
    mockService := &mocks.MockAnalyzerService{}
    handler := handlers.NewPRAnalyzerHandler(mockService, logger)
    
    // Act
    req := httptest.NewRequest("POST", "/analyze-pr", payload)
    rec := httptest.NewRecorder()
    handler.Handle(rec, req)
    
    // Assert
    assert.Equal(t, http.StatusOK, rec.Code)
    mockService.AssertExpectations(t)
}
```

## Troubleshooting

### Common Issues

#### Compilation Errors
```bash
go mod tidy           # Fix dependency issues
make fmt             # Format code
make lint            # Check for issues
```

#### HTTPS Certificate Issues
```bash
make gen-certs       # Regenerate certificates
ls -la certs/        # Check permissions (key=600, cert=644)
```

#### API Integration Issues
```bash
LOG_LEVEL=debug make dev    # Enable debug logging
curl -k https://localhost:8443/health  # Test connectivity
```

### Debugging Tips

#### Enable Verbose Logging
```bash
LOG_LEVEL=debug LOG_FORMAT=console make dev
```

#### Test Webhook Locally
```bash
# Use ngrok to expose local server
ngrok http 8443

# Configure GitHub webhook to ngrok URL
# https://abc123.ngrok.io/analyze-pr
```

#### Manual API Testing
```bash
# Health check
curl -k https://localhost:8443/health

# Simulate GitHub webhook
curl -X POST https://localhost:8443/analyze-pr \
  -H "Content-Type: application/json" \
  -H "X-GitHub-Event: pull_request" \
  -d @test/fixtures/sample_pr.json
```

## Production Considerations

- Use environment-specific configuration files
- Implement proper secret management (not .env files)
- Configure reverse proxy (nginx) for SSL termination
- Set up monitoring and alerting
- Use container orchestration (Kubernetes/Docker Swarm)
- Implement circuit breakers for external APIs
- Configure log aggregation (ELK stack, etc.)