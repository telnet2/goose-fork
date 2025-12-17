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

// MessageEvent is a discriminated union for SSE events
// Uses custom JSON marshaling to match Rust's #[serde(tag = "type")] behavior
type MessageEvent struct {
	Type MessageEventType

	// For Message event
	Message    *Message
	TokenState *TokenState

	// For Error event
	Error *string

	// For Finish event
	Reason *string

	// For ModelChange event
	Model *string
	Mode  *string

	// For Notification event
	RequestID           *string
	NotificationMessage *json.RawMessage // Named differently to avoid Go conflict

	// For UpdateConversation event
	Conversation Conversation
}

// MarshalJSON implements custom JSON marshaling to match Rust's internally-tagged enum
func (e MessageEvent) MarshalJSON() ([]byte, error) {
	switch e.Type {
	case EventTypeMessage:
		return json.Marshal(struct {
			Type       MessageEventType `json:"type"`
			Message    *Message         `json:"message"`
			TokenState *TokenState      `json:"token_state"`
		}{
			Type:       e.Type,
			Message:    e.Message,
			TokenState: e.TokenState,
		})

	case EventTypeError:
		return json.Marshal(struct {
			Type  MessageEventType `json:"type"`
			Error *string          `json:"error"`
		}{
			Type:  e.Type,
			Error: e.Error,
		})

	case EventTypeFinish:
		return json.Marshal(struct {
			Type       MessageEventType `json:"type"`
			Reason     *string          `json:"reason"`
			TokenState *TokenState      `json:"token_state"`
		}{
			Type:       e.Type,
			Reason:     e.Reason,
			TokenState: e.TokenState,
		})

	case EventTypeModelChange:
		return json.Marshal(struct {
			Type  MessageEventType `json:"type"`
			Model *string          `json:"model"`
			Mode  *string          `json:"mode"`
		}{
			Type:  e.Type,
			Model: e.Model,
			Mode:  e.Mode,
		})

	case EventTypeNotification:
		// Note: The field is "message" to match Rust's Notification { message: ServerNotification }
		return json.Marshal(struct {
			Type      MessageEventType `json:"type"`
			RequestID *string          `json:"request_id"`
			Message   *json.RawMessage `json:"message"`
		}{
			Type:      e.Type,
			RequestID: e.RequestID,
			Message:   e.NotificationMessage,
		})

	case EventTypeUpdateConversation:
		return json.Marshal(struct {
			Type         MessageEventType `json:"type"`
			Conversation Conversation     `json:"conversation"`
		}{
			Type:         e.Type,
			Conversation: e.Conversation,
		})

	case EventTypePing:
		return json.Marshal(struct {
			Type MessageEventType `json:"type"`
		}{
			Type: e.Type,
		})

	default:
		return json.Marshal(struct {
			Type MessageEventType `json:"type"`
		}{
			Type: e.Type,
		})
	}
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
// Note: notification parameter will be serialized as "message" to match Rust protocol
func NewNotificationEvent(requestID string, notification json.RawMessage) MessageEvent {
	return MessageEvent{
		Type:                EventTypeNotification,
		RequestID:           &requestID,
		NotificationMessage: &notification,
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
