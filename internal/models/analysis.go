package models

import "time"

// AnalysisRequest represents the request to analyze a PR
type AnalysisRequest struct {
	PullRequest    PullRequest     `json:"pull_request"`
	Repository     Repository      `json:"repository"`
	Diff           string          `json:"diff,omitempty"`
	ExistingRoutes []ExistingRoute `json:"existing_routes,omitempty"`
}

// ExistingRoute represents a route already documented in the collection
type ExistingRoute struct {
	Method      string   `json:"method"`
	Path        string   `json:"path"`
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	FolderPath  []string `json:"folder_path,omitempty"` // For nested folders
}

// AnalysisResponse represents the structured response from Claude
type AnalysisResponse struct {
	NewRoutes      []APIRoute    `json:"new_routes"`
	ModifiedRoutes []APIRoute    `json:"modified_routes"`
	DeletedRoutes  []APIRoute    `json:"deleted_routes"`
	Summary        string        `json:"summary"`
	Confidence     float64       `json:"confidence"`
	PostmanUpdate  PostmanUpdate `json:"postman_update"`
}

// APIRoute represents an API route with its details
type APIRoute struct {
	Method      string         `json:"method"`
	Path        string         `json:"path"`
	Description string         `json:"description"`
	Parameters  []Parameter    `json:"parameters,omitempty"`
	RequestBody map[string]any `json:"request_body,omitempty"`
	Response    map[string]any `json:"response,omitempty"`
	Headers     []Header       `json:"headers,omitempty"`
	Tags        []string       `json:"tags,omitempty"`
	Deprecated  bool           `json:"deprecated,omitempty"`
}

// Parameter represents an API parameter
type Parameter struct {
	Name        string `json:"name"`
	In          string `json:"in"` // query, path, header, body
	Type        string `json:"type"`
	Required    bool   `json:"required"`
	Description string `json:"description"`
	Default     any    `json:"default,omitempty"`
	Example     any    `json:"example,omitempty"`
}

// Header represents an HTTP header
type Header struct {
	Name        string `json:"name"`
	Required    bool   `json:"required"`
	Description string `json:"description"`
	Example     any    `json:"example,omitempty"`
}

// PostmanUpdate represents the result of updating Postman
type PostmanUpdate struct {
	CollectionID  string `json:"collection_id"`
	Status        string `json:"status"` // success, error, partial
	ItemsAdded    int    `json:"items_added"`
	ItemsModified int    `json:"items_modified"`
	ItemsDeleted  int    `json:"items_deleted"`
	ErrorMessage  string `json:"error_message,omitempty"`
	UpdatedAt     string `json:"updated_at"`
}

// UserSession represents a user session with their credentials
type UserSession struct {
	ClaudeAPIKey        string    `json:"claude_api_key"`
	PostmanAPIKey       string    `json:"postman_api_key"`
	PostmanWorkspaceID  string    `json:"postman_workspace_id"`
	PostmanCollectionID string    `json:"postman_collection_id"`
	CreatedAt           time.Time `json:"created_at"`
	ExpiresAt           time.Time `json:"expires_at"`
}
