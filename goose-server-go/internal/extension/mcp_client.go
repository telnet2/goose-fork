package extension

import (
	"context"
	"encoding/json"
)

// McpClient defines the interface for MCP (Model Context Protocol) clients
// This matches the Rust McpClientTrait interface
type McpClient interface {
	// ListResources lists available resources with optional pagination
	ListResources(ctx context.Context, cursor *string) (*ListResourcesResult, error)

	// ReadResource reads a specific resource by URI
	ReadResource(ctx context.Context, uri string) (*ReadResourceResult, error)

	// ListTools lists available tools with optional pagination
	ListTools(ctx context.Context, cursor *string) (*ListToolsResult, error)

	// CallTool executes a tool with the given arguments
	CallTool(ctx context.Context, name string, arguments json.RawMessage) (*CallToolResult, error)

	// ListPrompts lists available prompts with optional pagination
	ListPrompts(ctx context.Context, cursor *string) (*ListPromptsResult, error)

	// GetPrompt retrieves a specific prompt by name
	GetPrompt(ctx context.Context, name string, arguments map[string]string) (*GetPromptResult, error)

	// Subscribe returns a channel for receiving server notifications
	Subscribe() <-chan ServerNotification

	// GetInfo returns the server initialization information
	GetInfo() *InitializeResult

	// GetMoim returns the model information markdown (optional)
	GetMoim() *string

	// Close closes the client connection
	Close() error
}

// InitializeResult contains the result of MCP server initialization
type InitializeResult struct {
	ProtocolVersion string             `json:"protocolVersion"`
	Capabilities    ServerCapabilities `json:"capabilities"`
	ServerInfo      Implementation     `json:"serverInfo"`
	Instructions    *string            `json:"instructions,omitempty"`
}

// ServerCapabilities describes what the MCP server supports
type ServerCapabilities struct {
	Tools     *ToolsCapability     `json:"tools,omitempty"`
	Resources *ResourcesCapability `json:"resources,omitempty"`
	Prompts   *PromptsCapability   `json:"prompts,omitempty"`
	Logging   *LoggingCapability   `json:"logging,omitempty"`
}

// ToolsCapability indicates tool support
type ToolsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

// ResourcesCapability indicates resource support
type ResourcesCapability struct {
	Subscribe   bool `json:"subscribe,omitempty"`
	ListChanged bool `json:"listChanged,omitempty"`
}

// PromptsCapability indicates prompt support
type PromptsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

// LoggingCapability indicates logging support
type LoggingCapability struct{}

// Implementation identifies the MCP server
type Implementation struct {
	Name    string  `json:"name"`
	Version string  `json:"version"`
	Title   *string `json:"title,omitempty"`
}

// Tool represents an MCP tool definition
type Tool struct {
	Name        string           `json:"name"`
	Description string           `json:"description,omitempty"`
	InputSchema *json.RawMessage `json:"inputSchema,omitempty"`
}

// ListToolsResult contains the result of listing tools
type ListToolsResult struct {
	Tools      []Tool  `json:"tools"`
	NextCursor *string `json:"nextCursor,omitempty"`
}

// CallToolResult contains the result of a tool call
type CallToolResult struct {
	Content          []ToolContent    `json:"content"`
	IsError          bool             `json:"isError,omitempty"`
	StructuredOutput *json.RawMessage `json:"_meta,omitempty"`
}

// ToolContent represents content in a tool result
type ToolContent struct {
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
	URI      string  `json:"uri"`
	MimeType string  `json:"mimeType"`
	Text     *string `json:"text,omitempty"`
	Blob     *string `json:"blob,omitempty"`
}

// NewTextToolContent creates a text tool content
func NewTextToolContent(text string) ToolContent {
	return ToolContent{
		Type: "text",
		Text: &text,
	}
}

// NewImageToolContent creates an image tool content
func NewImageToolContent(data, mimeType string) ToolContent {
	return ToolContent{
		Type:     "image",
		Data:     &data,
		MimeType: &mimeType,
	}
}

// Resource represents an MCP resource
type Resource struct {
	URI         string  `json:"uri"`
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
	MimeType    *string `json:"mimeType,omitempty"`
}

// ListResourcesResult contains the result of listing resources
type ListResourcesResult struct {
	Resources  []Resource `json:"resources"`
	NextCursor *string    `json:"nextCursor,omitempty"`
}

// ReadResourceResult contains the result of reading a resource
type ReadResourceResult struct {
	Contents []ResourceContent `json:"contents"`
}

// Prompt represents an MCP prompt
type Prompt struct {
	Name        string            `json:"name"`
	Description *string           `json:"description,omitempty"`
	Arguments   []PromptArgument  `json:"arguments,omitempty"`
}

// PromptArgument represents an argument for a prompt
type PromptArgument struct {
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
	Required    bool    `json:"required,omitempty"`
}

// ListPromptsResult contains the result of listing prompts
type ListPromptsResult struct {
	Prompts    []Prompt `json:"prompts"`
	NextCursor *string  `json:"nextCursor,omitempty"`
}

// GetPromptResult contains the result of getting a prompt
type GetPromptResult struct {
	Description *string         `json:"description,omitempty"`
	Messages    []PromptMessage `json:"messages"`
}

// PromptMessage represents a message in a prompt
type PromptMessage struct {
	Role    string          `json:"role"` // user, assistant
	Content json.RawMessage `json:"content"`
}

// ServerNotification represents a notification from the MCP server
type ServerNotification struct {
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// Common notification methods
const (
	NotificationToolsListChanged     = "notifications/tools/list_changed"
	NotificationResourcesListChanged = "notifications/resources/list_changed"
	NotificationPromptsListChanged   = "notifications/prompts/list_changed"
	NotificationProgress             = "notifications/progress"
	NotificationMessage              = "notifications/message"
)

// ProgressNotification represents progress updates
type ProgressNotification struct {
	ProgressToken string  `json:"progressToken"`
	Progress      float64 `json:"progress"`
	Total         *int64  `json:"total,omitempty"`
}

// LoggingMessage represents a logging message from the server
type LoggingMessage struct {
	Level   string          `json:"level"` // debug, info, warning, error
	Logger  *string         `json:"logger,omitempty"`
	Data    json.RawMessage `json:"data,omitempty"`
}

// ProtocolVersion constants
const (
	ProtocolVersion2024_11_05 = "2024-11-05"
	ProtocolVersion2025_03_26 = "2025-03-26"
	CurrentProtocolVersion    = ProtocolVersion2025_03_26
)
