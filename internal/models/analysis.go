package models

// AnalysisRequest represents the request to analyze a PR
type AnalysisRequest struct {
	PullRequest PullRequest `json:"pull_request"`
	Repository  Repository  `json:"repository"`
	Diff        string      `json:"diff,omitempty"`
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
	Method      string                 `json:"method"`
	Path        string                 `json:"path"`
	Description string                 `json:"description"`
	Parameters  []Parameter            `json:"parameters,omitempty"`
	RequestBody map[string]interface{} `json:"request_body,omitempty"`
	Response    map[string]interface{} `json:"response,omitempty"`
	Headers     []Header               `json:"headers,omitempty"`
	Tags        []string               `json:"tags,omitempty"`
	Deprecated  bool                   `json:"deprecated,omitempty"`
}

// Parameter represents an API parameter
type Parameter struct {
	Name        string      `json:"name"`
	In          string      `json:"in"` // query, path, header, body
	Type        string      `json:"type"`
	Required    bool        `json:"required"`
	Description string      `json:"description"`
	Default     interface{} `json:"default,omitempty"`
	Example     interface{} `json:"example,omitempty"`
}

// Header represents an HTTP header
type Header struct {
	Name        string      `json:"name"`
	Required    bool        `json:"required"`
	Description string      `json:"description"`
	Example     interface{} `json:"example,omitempty"`
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
