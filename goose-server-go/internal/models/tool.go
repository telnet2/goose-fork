package models

import "encoding/json"

// ToolInfo represents information about an available tool
type ToolInfo struct {
	Name           string           `json:"name"`
	Description    string           `json:"description"`
	ExtensionName  string           `json:"extensionName"`
	InputSchema    *json.RawMessage `json:"inputSchema,omitempty"`
	RequiresAction bool             `json:"requiresAction"`
}

// ToolRequest represents a request to execute a tool
type ToolRequest struct {
	ID        string           `json:"id"`
	Name      string           `json:"name"`
	Arguments *json.RawMessage `json:"arguments,omitempty"`
}

// ToolResponse represents the result of tool execution
type ToolResponse struct {
	ID               string           `json:"id"`
	IsError          bool             `json:"is_error"`
	Content          []Content        `json:"content"`
	StructuredOutput *json.RawMessage `json:"structured_content,omitempty"`
}

// Content represents content in a tool response (discriminated union)
type Content struct {
	Type string `json:"type"` // text, image, resource

	// For text content
	Text *string `json:"text,omitempty"`

	// For image content
	Data     *string `json:"data,omitempty"`
	MimeType *string `json:"mimeType,omitempty"`

	// For resource content
	Resource *ResourceContent `json:"resource,omitempty"`
}

// ResourceContent represents embedded resource content
type ResourceContent struct {
	URI      string `json:"uri"`
	MimeType string `json:"mimeType"`
	Text     string `json:"text,omitempty"`
	Blob     string `json:"blob,omitempty"`
}

// NewTextContent creates a text content
func NewTextContent(text string) Content {
	return Content{
		Type: "text",
		Text: &text,
	}
}

// NewImageContent creates an image content
func NewImageContent(data, mimeType string) Content {
	return Content{
		Type:     "image",
		Data:     &data,
		MimeType: &mimeType,
	}
}

// CallToolRequest represents a request to call a tool
type CallToolRequest struct {
	SessionID string           `json:"session_id"`
	Name      string           `json:"name"`
	Arguments *json.RawMessage `json:"arguments"`
}

// CallToolResponse represents the response from a tool call
type CallToolResponse struct {
	Content          []Content        `json:"content"`
	IsError          bool             `json:"is_error"`
	StructuredOutput *json.RawMessage `json:"structured_content,omitempty"`
}

// ToolPermission represents permission settings for a tool
type ToolPermission struct {
	ToolName string `json:"tool_name"`
	Mode     string `json:"mode"` // always_allow, always_deny, ask
}

// ToolConfirmationRequest represents a request to confirm tool execution
type ToolConfirmationRequest struct {
	ID        string           `json:"id"`
	ToolName  string           `json:"toolName"`
	Arguments *json.RawMessage `json:"arguments,omitempty"`
}

// ToolConfirmationAction represents the action to take for a tool confirmation
type ToolConfirmationAction struct {
	ID            string  `json:"id"`
	Action        string  `json:"action"` // confirm, deny
	SessionID     string  `json:"sessionId"`
	PrincipalType *string `json:"principalType,omitempty"` // Extension, Tool
}
