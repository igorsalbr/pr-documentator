package models

import "time"

// PostmanCollection represents a Postman collection
type PostmanCollection struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Schema      string            `json:"schema"`
	Items       []PostmanItem     `json:"item"`
	Variables   []PostmanVariable `json:"variable,omitempty"`
	Auth        *PostmanAuth      `json:"auth,omitempty"`
	Info        PostmanInfo       `json:"info"`
}

// PostmanInfo contains collection metadata
type PostmanInfo struct {
	PostmanID   string `json:"_postman_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Schema      string `json:"schema"`
}

// PostmanItem represents a request item in Postman
type PostmanItem struct {
	ID          string            `json:"id,omitempty"`
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Request     *PostmanRequest   `json:"request,omitempty"`
	Response    []PostmanResponse `json:"response,omitempty"`
	Items       []PostmanItem     `json:"item,omitempty"` // For folders
	Event       []PostmanEvent    `json:"event,omitempty"`
}

// PostmanRequest represents a request in Postman
type PostmanRequest struct {
	Method      string          `json:"method"`
	Header      []PostmanHeader `json:"header,omitempty"`
	Body        *PostmanBody    `json:"body,omitempty"`
	URL         PostmanURL      `json:"url"`
	Auth        *PostmanAuth    `json:"auth,omitempty"`
	Description string          `json:"description,omitempty"`
}

// PostmanURL represents a URL in Postman
type PostmanURL struct {
	Raw      string              `json:"raw"`
	Protocol string              `json:"protocol,omitempty"`
	Host     []string            `json:"host,omitempty"`
	Path     []string            `json:"path,omitempty"`
	Query    []PostmanQueryParam `json:"query,omitempty"`
	Variable []PostmanVariable   `json:"variable,omitempty"`
}

// PostmanHeader represents a header in Postman
type PostmanHeader struct {
	Key         string `json:"key"`
	Value       string `json:"value"`
	Type        string `json:"type,omitempty"`
	Disabled    bool   `json:"disabled,omitempty"`
	Description string `json:"description,omitempty"`
}

// PostmanBody represents a request body in Postman
type PostmanBody struct {
	Mode    string         `json:"mode"` // raw, formdata, urlencoded, etc.
	Raw     string         `json:"raw,omitempty"`
	Options map[string]any `json:"options,omitempty"`
}

// PostmanQueryParam represents a query parameter
type PostmanQueryParam struct {
	Key         string `json:"key"`
	Value       string `json:"value"`
	Disabled    bool   `json:"disabled,omitempty"`
	Description string `json:"description,omitempty"`
}

// PostmanVariable represents a variable
type PostmanVariable struct {
	Key         string `json:"key"`
	Value       any    `json:"value"`
	Type        string `json:"type,omitempty"`
	Description string `json:"description,omitempty"`
}

// PostmanAuth represents authentication
type PostmanAuth struct {
	Type   string         `json:"type"`
	Config map[string]any `json:"config,omitempty"`
}

// PostmanResponse represents a response example
type PostmanResponse struct {
	Name            string          `json:"name"`
	OriginalRequest PostmanRequest  `json:"originalRequest"`
	Status          string          `json:"status"`
	Code            int             `json:"code"`
	Header          []PostmanHeader `json:"header"`
	Body            string          `json:"body"`
}

// PostmanEvent represents an event (pre-request, test)
type PostmanEvent struct {
	Listen string             `json:"listen"`
	Script PostmanEventScript `json:"script"`
}

// PostmanEventScript represents event script
type PostmanEventScript struct {
	Type string   `json:"type"`
	Exec []string `json:"exec"`
}

// PostmanCollectionResponse represents the API response when getting a collection
type PostmanCollectionResponse struct {
	Collection PostmanCollection `json:"collection"`
}

// PostmanUpdateRequest represents a request to update a collection
type PostmanUpdateRequest struct {
	Collection PostmanCollection `json:"collection"`
}

// PostmanUpdateResponse represents the response from updating a collection
type PostmanUpdateResponse struct {
	Collection PostmanCollectionMeta `json:"collection"`
}

// PostmanCollectionMeta represents collection metadata
type PostmanCollectionMeta struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	UID       string    `json:"uid"`
	UpdatedAt time.Time `json:"updatedAt"`
}
