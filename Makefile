.PHONY: help build clean dev test test-unit test-int lint fmt deps gen-certs run

# Variables
BINARY_NAME=pr-documentator
BUILD_DIR=bin
MAIN_PATH=cmd/server/main.go
COVERAGE_DIR=coverage

# Default target
help: ## Show this help message
	@echo "Available commands:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'

# Build commands
build: ## Build the application for production
	@echo "🔨 Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	@go build -ldflags="-w -s" -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)
	@echo "✅ Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

build-dev: ## Build the application for development (with debug symbols)
	@echo "🔨 Building $(BINARY_NAME) (development)..."
	@mkdir -p $(BUILD_DIR)
	@go build -o $(BUILD_DIR)/$(BINARY_NAME)-dev $(MAIN_PATH)
	@echo "✅ Development build complete: $(BUILD_DIR)/$(BINARY_NAME)-dev"

clean: ## Clean build artifacts and temporary files
	@echo "🧹 Cleaning..."
	@rm -rf $(BUILD_DIR)
	@rm -rf $(COVERAGE_DIR)
	@go clean
	@echo "✅ Clean complete"

# Development commands
dev: gen-certs ## Run the application with hot reload (requires air: go install github.com/cosmtrek/air@latest)
	@if ! command -v air >/dev/null 2>&1; then \
		echo "📦 Installing air for hot reload..."; \
		go install github.com/cosmtrek/air@latest; \
	fi
	@echo "🚀 Starting development server with hot reload..."
	@air -c .air.toml

run: gen-certs ## Run the application directly
	@echo "🚀 Starting $(BINARY_NAME)..."
	@go run $(MAIN_PATH)

# Testing commands
test: ## Run all tests
	@echo "🧪 Running all tests..."
	@go test -v ./...

test-unit: ## Run unit tests only
	@echo "🧪 Running unit tests..."
	@go test -v -short ./...

test-int: ## Run integration tests only
	@echo "🧪 Running integration tests..."
	@go test -v -run Integration ./...

test-coverage: ## Run tests with coverage report
	@echo "🧪 Running tests with coverage..."
	@mkdir -p $(COVERAGE_DIR)
	@go test -coverprofile=$(COVERAGE_DIR)/coverage.out ./...
	@go tool cover -html=$(COVERAGE_DIR)/coverage.out -o $(COVERAGE_DIR)/coverage.html
	@echo "📊 Coverage report generated: $(COVERAGE_DIR)/coverage.html"

# Code quality commands
lint: ## Run linter (requires golangci-lint)
	@if ! command -v golangci-lint >/dev/null 2>&1; then \
		echo "📦 Installing golangci-lint..."; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
	fi
	@echo "🔍 Running linter..."
	@golangci-lint run

fmt: ## Format code
	@echo "✨ Formatting code..."
	@go fmt ./...
	@go mod tidy

# Dependency commands
deps: ## Download and verify dependencies
	@echo "📦 Downloading dependencies..."
	@go mod download
	@go mod verify
	@go mod tidy

deps-upgrade: ## Upgrade all dependencies
	@echo "📦 Upgrading dependencies..."
	@go get -u ./...
	@go mod tidy

# Utility commands
gen-certs: ## Generate self-signed certificates for HTTPS
	@./scripts/generate_certs.sh

docker-build: ## Build Docker image
	@echo "🐳 Building Docker image..."
	@docker build -t $(BINARY_NAME):latest .

docker-run: ## Run application in Docker
	@echo "🐳 Running Docker container..."
	@docker run --rm -p 8443:8443 --env-file .env $(BINARY_NAME):latest

# Installation commands
install-tools: ## Install development tools
	@echo "🛠️  Installing development tools..."
	@go install github.com/cosmtrek/air@latest
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@echo "✅ Development tools installed"

# Testing commands
test-webhook: gen-certs ## Test webhook locally with sample payload
	@echo "🧪 Testing webhook locally..."
	@./scripts/test_webhook.sh

test-full: ## Run complete local development test
	@echo "🚀 Running full local test suite..."
	@./scripts/test_local_development.sh

test-health: ## Test health endpoint
	@echo "🩺 Testing health endpoint..."
	@curl -k https://localhost:8443/health || echo "Server not running. Start with 'make dev' first."

test-metrics: ## Test metrics endpoint
	@echo "📊 Testing metrics endpoint..."
	@curl -k https://localhost:8443/metrics || echo "Server not running. Start with 'make dev' first."

# Development helpers
dev-logs: ## Show development logs with formatting
	@echo "📋 Showing formatted logs..."
	@tail -f logs/app.log | jq . 2>/dev/null || tail -f logs/app.log

ngrok-expose: ## Expose local server with ngrok (requires ngrok installation)
	@echo "🌐 Exposing server via ngrok..."
	@ngrok http 8443

# Git hooks
setup-hooks: ## Setup git hooks
	@echo "🎣 Setting up git hooks..."
	@echo '#!/bin/bash\nmake fmt lint' > .git/hooks/pre-commit
	@chmod +x .git/hooks/pre-commit
	@echo "✅ Git hooks setup complete"