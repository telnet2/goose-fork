package models

import (
	"encoding/json"
	"time"
)

// Role represents the message sender
type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
)

// MessageMetadata contains visibility settings for a message
type MessageMetadata struct {
	UserVisible  bool `json:"userVisible"`
	AgentVisible bool `json:"agentVisible"`
}

// Message represents a message to or from an LLM
type Message struct {
	Role     Role              `json:"role"`
	Created  int64             `json:"created"`
	Content  []MessageContent  `json:"content"`
	Metadata MessageMetadata   `json:"metadata"`
	ID       *string           `json:"id,omitempty"`
}

// NewUserMessage creates a new user message
func NewUserMessage(text string) Message {
	return Message{
		Role:    RoleUser,
		Created: time.Now().Unix(),
		Content: []MessageContent{
			{Type: "text", Text: &text},
		},
		Metadata: MessageMetadata{
			UserVisible:  true,
			AgentVisible: true,
		},
	}
}

// NewAssistantMessage creates a new assistant message
func NewAssistantMessage(text string) Message {
	return Message{
		Role:    RoleAssistant,
		Created: time.Now().Unix(),
		Content: []MessageContent{
			{Type: "text", Text: &text},
		},
		Metadata: MessageMetadata{
			UserVisible:  true,
			AgentVisible: true,
		},
	}
}

// MessageContent represents content within a message
// This is a discriminated union based on the "type" field
type MessageContent struct {
	Type string `json:"type"`

	// For text content
	Text *string `json:"text,omitempty"`

	// For image content
	Data     *string `json:"data,omitempty"`
	MimeType *string `json:"mimeType,omitempty"`

	// For tool request/response
	ID               *string          `json:"id,omitempty"`
	ToolCall         *json.RawMessage `json:"toolCall,omitempty"`
	ToolResult       *json.RawMessage `json:"toolResult,omitempty"`
	ToolName         *string          `json:"toolName,omitempty"`
	Arguments        *json.RawMessage `json:"arguments,omitempty"`
	ThoughtSignature *string          `json:"thoughtSignature,omitempty"` // For toolRequest type

	// For thinking content
	Thinking  *string `json:"thinking,omitempty"`
	Signature *string `json:"signature,omitempty"` // For thinking type (different from thoughtSignature)

	// For system notification
	NotificationType *string `json:"notificationType,omitempty"`
	Msg              *string `json:"msg,omitempty"`

	// For action required
	ActionData *ActionRequiredData `json:"data,omitempty"`
}

// ActionRequiredData represents action required from user
type ActionRequiredData struct {
	ActionType     string           `json:"actionType"`
	ID             string           `json:"id"`
	ToolName       *string          `json:"toolName,omitempty"`
	Arguments      *json.RawMessage `json:"arguments,omitempty"`
	Prompt         *string          `json:"prompt,omitempty"`
	Message        *string          `json:"message,omitempty"`
	RequestedSchema *json.RawMessage `json:"requested_schema,omitempty"`
	UserData       *json.RawMessage `json:"user_data,omitempty"`
}

// Conversation is a list of messages
type Conversation []Message

// TokenState represents token usage information
type TokenState struct {
	InputTokens             int32 `json:"inputTokens"`
	OutputTokens            int32 `json:"outputTokens"`
	TotalTokens             int32 `json:"totalTokens"`
	AccumulatedInputTokens  int32 `json:"accumulatedInputTokens"`
	AccumulatedOutputTokens int32 `json:"accumulatedOutputTokens"`
	AccumulatedTotalTokens  int32 `json:"accumulatedTotalTokens"`
}
