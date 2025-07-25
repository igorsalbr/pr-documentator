package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/igorsal/pr-documentator/api/handlers"
	"github.com/igorsal/pr-documentator/api/middleware"
	"github.com/igorsal/pr-documentator/internal/config"
	"github.com/igorsal/pr-documentator/internal/interfaces"
	"github.com/igorsal/pr-documentator/internal/services"
	"github.com/igorsal/pr-documentator/io/claude"
	"github.com/igorsal/pr-documentator/io/postman"
	"github.com/igorsal/pr-documentator/pkg/logger"
	"github.com/igorsal/pr-documentator/pkg/metrics"
)

const (
	DefaultVersion  = "2.0.0"
	ShutdownTimeout = 30 * time.Second
	IdleTimeout     = 120 * time.Second
)

// Application holds all dependencies
type Application struct {
	config          *config.Config
	logger          interfaces.Logger
	metrics         interfaces.MetricsCollector
	claudeClient    interfaces.ClaudeClient
	postmanClient   interfaces.PostmanClient
	analyzerService interfaces.AnalyzerService
	server          *http.Server
}

func main() {
	app, err := initializeApplication()
	if err != nil {
		fmt.Printf("Failed to initialize application: %v\n", err)
		os.Exit(1)
	}

	app.logger.Info("Starting PR Documentator service",
		"version", DefaultVersion,
		"environment", os.Getenv("ENVIRONMENT"),
	)

	if err := app.run(); err != nil {
		app.logger.Fatal("Application failed to run", err)
	}
}

// initializeApplication sets up all dependencies using dependency injection pattern
func initializeApplication() (*Application, error) {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	// Initialize logger
	logger := logger.NewAdapter(cfg.Logging.Level, cfg.Logging.Format)

	// Initialize metrics collector
	metrics := metrics.NewPrometheusCollector()

	// Initialize clients with dependencies
	claudeClient := claude.NewClient(cfg.Claude, logger, metrics)
	postmanClient := postman.NewClient(cfg.Postman, logger, metrics)

	// Initialize services
	analyzerService := services.NewAnalyzerService(claudeClient, postmanClient, logger, metrics)

	// Create application
	app := &Application{
		config:          cfg,
		logger:          logger,
		metrics:         metrics,
		claudeClient:    claudeClient,
		postmanClient:   postmanClient,
		analyzerService: analyzerService,
	}

	// Setup HTTP server
	app.setupServer()

	return app, nil
}

// setupServer configures the HTTP server with all routes and middleware
func (app *Application) setupServer() {
	// Initialize handlers
	healthHandler := handlers.NewHealthHandler(app.logger, app.metrics)
	prAnalyzerHandler := handlers.NewPRAnalyzerHandler(app.analyzerService, app.logger, app.metrics)
	manualWebhookHandler := handlers.NewManualWebhookHandler(app.analyzerService, app.logger, app.metrics)

	// Setup router
	router := mux.NewRouter()

	// Apply global middleware in order
	router.Use(middleware.PanicRecoveryMiddleware(app.logger))
	router.Use(middleware.MetricsMiddleware(app.metrics))
	router.Use(middleware.LoggingMiddleware(app.logger))
	router.Use(middleware.ErrorHandlerMiddleware(app.logger))
	router.Use(middleware.CORSMiddleware(app.logger))

	// Public endpoints
	router.HandleFunc("/health", healthHandler.Handle).Methods("GET")
	router.Handle("/metrics", promhttp.Handler()).Methods("GET")
	router.HandleFunc("/manual-analyze", manualWebhookHandler.Handle).Methods("POST")

	// Protected endpoints
	prRouter := router.PathPrefix("").Subrouter()
	prRouter.Use(middleware.GitHubWebhookAuth(app.config.GitHub.WebhookSecret, app.logger))
	prRouter.HandleFunc("/analyze-pr", prAnalyzerHandler.Handle).Methods("POST")

	// Setup server with robust configuration
	app.server = &http.Server{
		Addr:         fmt.Sprintf("%s:%s", app.config.Server.Host, app.config.Server.Port),
		Handler:      router,
		ReadTimeout:  app.config.Server.ReadTimeout,
		WriteTimeout: app.config.Server.WriteTimeout,
		IdleTimeout:  IdleTimeout,
		// Add security headers
		ErrorLog: nil, // Use our custom logger
	}
}

// run starts the application and handles graceful shutdown
func (app *Application) run() error {
	// Channel to capture server errors
	serverErrors := make(chan error, 1)

	// Start HTTPS server in goroutine
	go func() {
		app.logger.Info("Starting HTTPS server",
			"host", app.config.Server.Host,
			"port", app.config.Server.Port,
			"cert_file", app.config.Server.TLSCertFile,
			"key_file", app.config.Server.TLSKeyFile,
		)

		if err := app.server.ListenAndServeTLS(
			app.config.Server.TLSCertFile,
			app.config.Server.TLSKeyFile,
		); err != nil && err != http.ErrServerClosed {
			serverErrors <- err
		}
	}()

	// Wait for interrupt signal or server error
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	select {
	case err := <-serverErrors:
		return fmt.Errorf("server failed to start: %w", err)

	case <-ctx.Done():
		app.logger.Info("Shutdown signal received")
		return app.gracefulShutdown()
	}
}

// gracefulShutdown performs graceful shutdown with timeout
func (app *Application) gracefulShutdown() error {
	app.logger.Info("Starting graceful shutdown")

	// Create shutdown context with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), ShutdownTimeout)
	defer cancel()

	// Track shutdown progress
	shutdownComplete := make(chan error, 1)

	go func() {
		// Shutdown HTTP server
		if err := app.server.Shutdown(shutdownCtx); err != nil {
			shutdownComplete <- fmt.Errorf("server shutdown failed: %w", err)
			return
		}

		// Close other resources if needed (database connections, etc.)
		app.logger.Info("All services shutdown successfully")
		shutdownComplete <- nil
	}()

	// Wait for shutdown to complete or timeout
	select {
	case err := <-shutdownComplete:
		if err != nil {
			app.logger.Error("Graceful shutdown failed", err)
			// Force close as last resort
			if closeErr := app.server.Close(); closeErr != nil {
				app.logger.Error("Force shutdown also failed", closeErr)
			}
			return err
		}
		app.logger.Info("Graceful shutdown completed successfully")
		return nil

	case <-shutdownCtx.Done():
		app.logger.Error("Shutdown timeout exceeded, forcing close", nil)
		if err := app.server.Close(); err != nil {
			app.logger.Error("Force shutdown failed", err)
		}
		return fmt.Errorf("shutdown timeout exceeded")
	}
}
