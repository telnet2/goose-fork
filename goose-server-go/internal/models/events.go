package models

import "encoding/json"

// MessageEventType represents the type of SSE event
type MessageEventType string

const (
	EventTypeMessage            MessageEventType = "Message"
	EventTypeError              MessageEventType = "Error"
	EventTypeFinish             MessageEventType = "Finish"
	EventTypeModelChange        MessageEventType = "ModelChange"
	EventTypeNotification       MessageEventType = "Notification"
	EventTypeUpdateConversation MessageEventType = "UpdateConversation"
	EventTypePing               MessageEventType = "Ping"
)

// MessageEvent is the base discriminated union for SSE events
// The "type" field determines which variant is used
type MessageEvent struct {
	Type MessageEventType `json:"type"`

	// For Message event
	Message    *Message    `json:"message,omitempty"`
	TokenState *TokenState `json:"token_state,omitempty"`

	// For Error event
	Error *string `json:"error,omitempty"`

	// For Finish event
	Reason *string `json:"reason,omitempty"`

	// For ModelChange event
	Model *string `json:"model,omitempty"`
	Mode  *string `json:"mode,omitempty"`

	// For Notification event
	RequestID    *string          `json:"request_id,omitempty"`
	Notification *json.RawMessage `json:"notification,omitempty"` // ServerNotification from MCP

	// For UpdateConversation event
	Conversation Conversation `json:"conversation,omitempty"`
}

// NewMessageEvent creates a Message SSE event
func NewMessageEvent(msg Message, tokenState *TokenState) MessageEvent {
	return MessageEvent{
		Type:       EventTypeMessage,
		Message:    &msg,
		TokenState: tokenState,
	}
}

// NewErrorEvent creates an Error SSE event
func NewErrorEvent(errMsg string) MessageEvent {
	return MessageEvent{
		Type:  EventTypeError,
		Error: &errMsg,
	}
}

// NewFinishEvent creates a Finish SSE event
func NewFinishEvent(reason string, tokenState *TokenState) MessageEvent {
	return MessageEvent{
		Type:       EventTypeFinish,
		Reason:     &reason,
		TokenState: tokenState,
	}
}

// NewModelChangeEvent creates a ModelChange SSE event
func NewModelChangeEvent(model, mode string) MessageEvent {
	return MessageEvent{
		Type:  EventTypeModelChange,
		Model: &model,
		Mode:  &mode,
	}
}

// NewNotificationEvent creates a Notification SSE event
func NewNotificationEvent(requestID string, notification json.RawMessage) MessageEvent {
	return MessageEvent{
		Type:         EventTypeNotification,
		RequestID:    &requestID,
		Notification: &notification,
	}
}

// NewUpdateConversationEvent creates an UpdateConversation SSE event
func NewUpdateConversationEvent(conversation Conversation) MessageEvent {
	return MessageEvent{
		Type:         EventTypeUpdateConversation,
		Conversation: conversation,
	}
}

// NewPingEvent creates a Ping SSE event
func NewPingEvent() MessageEvent {
	return MessageEvent{
		Type: EventTypePing,
	}
}

// ChatRequest represents a request to the /reply endpoint
type ChatRequest struct {
	Messages      []Message `json:"messages"`
	SessionID     string    `json:"session_id"`
	RecipeName    *string   `json:"recipe_name,omitempty"`
	RecipeVersion *string   `json:"recipe_version,omitempty"`
}

// FinishReason represents the reason for stream completion
type FinishReason string

const (
	FinishReasonStop          FinishReason = "stop"
	FinishReasonLength        FinishReason = "length"
	FinishReasonToolUse       FinishReason = "tool_use"
	FinishReasonContentFilter FinishReason = "content_filter"
	FinishReasonError         FinishReason = "error"
)
