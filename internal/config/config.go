package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Server   ServerConfig
	Claude   ClaudeConfig
	Postman  PostmanConfig
	GitHub   GitHubConfig
	Logging  LoggingConfig
}

type ServerConfig struct {
	Host         string
	Port         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	TLSCertFile  string
	TLSKeyFile   string
}

type ClaudeConfig struct {
	APIKey     string
	Model      string
	MaxTokens  int
	BaseURL    string
	Timeout    time.Duration
}

type PostmanConfig struct {
	APIKey       string
	WorkspaceID  string
	CollectionID string
	BaseURL      string
	Timeout      time.Duration
}

type GitHubConfig struct {
	WebhookSecret string
}

type LoggingConfig struct {
	Level  string
	Format string
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	// Try to load .env file (ignore error if file doesn't exist)
	_ = godotenv.Load()

	cfg := &Config{
		Server: ServerConfig{
			Host:         getEnvWithDefault("SERVER_HOST", "0.0.0.0"),
			Port:         getEnvWithDefault("SERVER_PORT", "8443"),
			ReadTimeout:  getDurationFromEnv("SERVER_READ_TIMEOUT", 15*time.Second),
			WriteTimeout: getDurationFromEnv("SERVER_WRITE_TIMEOUT", 15*time.Second),
			TLSCertFile:  getEnvWithDefault("TLS_CERT_FILE", "./certs/server.crt"),
			TLSKeyFile:   getEnvWithDefault("TLS_KEY_FILE", "./certs/server.key"),
		},
		Claude: ClaudeConfig{
			APIKey:    getRequiredEnv("CLAUDE_API_KEY"),
			Model:     getEnvWithDefault("CLAUDE_MODEL", "claude-3-sonnet-20240229"),
			MaxTokens: getIntFromEnv("CLAUDE_MAX_TOKENS", 4096),
			BaseURL:   getEnvWithDefault("CLAUDE_BASE_URL", "https://api.anthropic.com"),
			Timeout:   getDurationFromEnv("CLAUDE_TIMEOUT", 30*time.Second),
		},
		Postman: PostmanConfig{
			APIKey:       getRequiredEnv("POSTMAN_API_KEY"),
			WorkspaceID:  getRequiredEnv("POSTMAN_WORKSPACE_ID"),
			CollectionID: getRequiredEnv("POSTMAN_COLLECTION_ID"),
			BaseURL:      getEnvWithDefault("POSTMAN_BASE_URL", "https://api.postman.com"),
			Timeout:      getDurationFromEnv("POSTMAN_TIMEOUT", 30*time.Second),
		},
		GitHub: GitHubConfig{
			WebhookSecret: getEnvWithDefault("GITHUB_WEBHOOK_SECRET", ""),
		},
		Logging: LoggingConfig{
			Level:  getEnvWithDefault("LOG_LEVEL", "info"),
			Format: getEnvWithDefault("LOG_FORMAT", "json"),
		},
	}

	return cfg, nil
}

func getRequiredEnv(key string) string {
	value := os.Getenv(key)
	if value == "" {
		panic(fmt.Sprintf("Required environment variable %s is not set", key))
	}
	return value
}

func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getIntFromEnv(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getDurationFromEnv(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}