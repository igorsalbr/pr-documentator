package postman

// PostmanErrorResponse represents an error response from Postman API
type PostmanErrorResponse struct {
	Error PostmanError `json:"error"`
}

// PostmanError contains error details
type PostmanError struct {
	Name    string `json:"name"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// WorkspaceResponse represents the response when getting workspace info
type WorkspaceResponse struct {
	Workspace Workspace `json:"workspace"`
}

// Workspace represents a Postman workspace
type Workspace struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
}

// CollectionsResponse represents the response when listing collections
type CollectionsResponse struct {
	Collections []CollectionSummary `json:"collections"`
}

// CollectionSummary represents a collection summary
type CollectionSummary struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	UID  string `json:"uid"`
}
