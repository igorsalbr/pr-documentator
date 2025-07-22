package claude

// ClaudeRequest represents a request to the Claude API
type ClaudeRequest struct {
	Model     string    `json:"model"`
	MaxTokens int       `json:"max_tokens"`
	Messages  []Message `json:"messages"`
	System    string    `json:"system,omitempty"`
	Tools     []Tool    `json:"tools,omitempty"`
	ToolChoice interface{} `json:"tool_choice,omitempty"`
}

// Message represents a message in the Claude conversation
type Message struct {
	Role    string `json:"role"`    // "user" or "assistant"
	Content string `json:"content"`
}

// Tool represents a function tool that Claude can call
type Tool struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema InputSchema `json:"input_schema"`
}

// InputSchema defines the JSON schema for tool inputs
type InputSchema struct {
	Type       string                 `json:"type"`
	Properties map[string]Property    `json:"properties"`
	Required   []string               `json:"required"`
}

// Property represents a property in the JSON schema
type Property struct {
	Type        string                 `json:"type"`
	Description string                 `json:"description,omitempty"`
	Items       *Property             `json:"items,omitempty"`
	Properties  map[string]Property   `json:"properties,omitempty"`
	Required    []string              `json:"required,omitempty"`
}

// ClaudeResponse represents the response from Claude API
type ClaudeResponse struct {
	ID           string        `json:"id"`
	Type         string        `json:"type"`
	Role         string        `json:"role"`
	Content      []Content     `json:"content"`
	Model        string        `json:"model"`
	StopReason   string        `json:"stop_reason"`
	StopSequence string        `json:"stop_sequence"`
	Usage        Usage         `json:"usage"`
}

// Content represents the content in Claude's response
type Content struct {
	Type  string   `json:"type"`
	Text  string   `json:"text,omitempty"`
	Name  string   `json:"name,omitempty"`
	Input map[string]interface{} `json:"input,omitempty"`
}

// Usage represents token usage information
type Usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// ClaudeError represents an error response from Claude API
type ClaudeError struct {
	Type    string      `json:"type"`
	Error   ErrorDetail `json:"error"`
}

// ErrorDetail contains error details
type ErrorDetail struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}