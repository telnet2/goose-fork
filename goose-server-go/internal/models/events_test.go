package models

import (
	"encoding/json"
	"testing"
)

func TestNewMessageEvent(t *testing.T) {
	msg := NewUserMessage("Hello")
	tokenState := &TokenState{
		InputTokens:             100,
		OutputTokens:            50,
		TotalTokens:             150,
		AccumulatedInputTokens:  1000,
		AccumulatedOutputTokens: 500,
		AccumulatedTotalTokens:  1500,
	}

	event := NewMessageEvent(msg, tokenState)

	if event.Type != EventTypeMessage {
		t.Errorf("Type = %q, want %q", event.Type, EventTypeMessage)
	}

	if event.Message == nil {
		t.Fatal("Message should not be nil")
	}

	if event.Message.Role != RoleUser {
		t.Errorf("Message.Role = %q, want %q", event.Message.Role, RoleUser)
	}

	if event.TokenState == nil {
		t.Fatal("TokenState should not be nil")
	}

	if event.TokenState.InputTokens != 100 {
		t.Errorf("TokenState.InputTokens = %d, want %d", event.TokenState.InputTokens, 100)
	}
}

func TestNewErrorEvent(t *testing.T) {
	errMsg := "Something went wrong"
	event := NewErrorEvent(errMsg)

	if event.Type != EventTypeError {
		t.Errorf("Type = %q, want %q", event.Type, EventTypeError)
	}

	if event.Error == nil {
		t.Fatal("Error should not be nil")
	}

	if *event.Error != errMsg {
		t.Errorf("Error = %q, want %q", *event.Error, errMsg)
	}
}

func TestNewFinishEvent(t *testing.T) {
	tokenState := &TokenState{
		InputTokens:  100,
		OutputTokens: 50,
		TotalTokens:  150,
	}

	event := NewFinishEvent("stop", tokenState)

	if event.Type != EventTypeFinish {
		t.Errorf("Type = %q, want %q", event.Type, EventTypeFinish)
	}

	if event.Reason == nil {
		t.Fatal("Reason should not be nil")
	}

	if *event.Reason != "stop" {
		t.Errorf("Reason = %q, want %q", *event.Reason, "stop")
	}

	if event.TokenState == nil {
		t.Fatal("TokenState should not be nil")
	}
}

func TestNewModelChangeEvent(t *testing.T) {
	event := NewModelChangeEvent("claude-3-opus", "default")

	if event.Type != EventTypeModelChange {
		t.Errorf("Type = %q, want %q", event.Type, EventTypeModelChange)
	}

	if event.Model == nil {
		t.Fatal("Model should not be nil")
	}

	if *event.Model != "claude-3-opus" {
		t.Errorf("Model = %q, want %q", *event.Model, "claude-3-opus")
	}

	if event.Mode == nil {
		t.Fatal("Mode should not be nil")
	}

	if *event.Mode != "default" {
		t.Errorf("Mode = %q, want %q", *event.Mode, "default")
	}
}

func TestNewUpdateConversationEvent(t *testing.T) {
	conversation := Conversation{
		NewUserMessage("Hello"),
		NewAssistantMessage("Hi there!"),
	}

	event := NewUpdateConversationEvent(conversation)

	if event.Type != EventTypeUpdateConversation {
		t.Errorf("Type = %q, want %q", event.Type, EventTypeUpdateConversation)
	}

	if len(event.Conversation) != 2 {
		t.Errorf("len(Conversation) = %d, want 2", len(event.Conversation))
	}
}

func TestNewPingEvent(t *testing.T) {
	event := NewPingEvent()

	if event.Type != EventTypePing {
		t.Errorf("Type = %q, want %q", event.Type, EventTypePing)
	}
}

func TestMessageEventJSON_Message(t *testing.T) {
	msg := NewUserMessage("Hello")
	tokenState := &TokenState{InputTokens: 10}
	event := NewMessageEvent(msg, tokenState)

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if parsed["type"] != "Message" {
		t.Errorf("type = %v, want %q", parsed["type"], "Message")
	}

	// Check message field is present in JSON
	if _, ok := parsed["message"]; !ok {
		t.Errorf("message field should be present in JSON, got: %s", string(data))
	}

	// Check the token_state is present
	if _, ok := parsed["token_state"]; !ok {
		t.Error("token_state field should be present")
	}

	// Verify the message content
	if event.Message == nil {
		t.Error("event.Message should not be nil")
	} else if event.Message.Role != RoleUser {
		t.Errorf("event.Message.Role = %q, want %q", event.Message.Role, RoleUser)
	}
}

func TestMessageEventJSON_Error(t *testing.T) {
	event := NewErrorEvent("Test error")

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if parsed["type"] != "Error" {
		t.Errorf("type = %v, want %q", parsed["type"], "Error")
	}

	if parsed["error"] != "Test error" {
		t.Errorf("error = %v, want %q", parsed["error"], "Test error")
	}
}

func TestMessageEventJSON_Ping(t *testing.T) {
	event := NewPingEvent()

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if parsed["type"] != "Ping" {
		t.Errorf("type = %v, want %q", parsed["type"], "Ping")
	}

	// Ping event should be minimal
	if len(parsed) != 1 {
		t.Errorf("Ping event should only have type field, got %d fields", len(parsed))
	}
}

func TestMessageEventJSON_Notification(t *testing.T) {
	// Test that Notification event correctly uses "message" field (not "notification")
	// to match Rust protocol
	notification := json.RawMessage(`{"method":"test","params":{}}`)
	event := NewNotificationEvent("req-123", notification)

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if parsed["type"] != "Notification" {
		t.Errorf("type = %v, want %q", parsed["type"], "Notification")
	}

	if parsed["request_id"] != "req-123" {
		t.Errorf("request_id = %v, want %q", parsed["request_id"], "req-123")
	}

	// CRITICAL: Must be "message" not "notification" to match Rust protocol
	if _, ok := parsed["message"]; !ok {
		t.Errorf("Notification event should have 'message' field (not 'notification'), got: %s", string(data))
	}

	// Should NOT have "notification" field
	if _, ok := parsed["notification"]; ok {
		t.Errorf("Notification event should NOT have 'notification' field, got: %s", string(data))
	}
}

func TestMessageEventJSON_Finish(t *testing.T) {
	tokenState := &TokenState{
		InputTokens:  100,
		OutputTokens: 50,
		TotalTokens:  150,
	}
	event := NewFinishEvent("stop", tokenState)

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if parsed["type"] != "Finish" {
		t.Errorf("type = %v, want %q", parsed["type"], "Finish")
	}

	if parsed["reason"] != "stop" {
		t.Errorf("reason = %v, want %q", parsed["reason"], "stop")
	}

	if _, ok := parsed["token_state"]; !ok {
		t.Error("token_state field should be present")
	}
}

func TestMessageEventJSON_ModelChange(t *testing.T) {
	event := NewModelChangeEvent("claude-3-opus", "default")

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if parsed["type"] != "ModelChange" {
		t.Errorf("type = %v, want %q", parsed["type"], "ModelChange")
	}

	if parsed["model"] != "claude-3-opus" {
		t.Errorf("model = %v, want %q", parsed["model"], "claude-3-opus")
	}

	if parsed["mode"] != "default" {
		t.Errorf("mode = %v, want %q", parsed["mode"], "default")
	}
}

func TestMessageEventJSON_UpdateConversation(t *testing.T) {
	conversation := Conversation{
		NewUserMessage("Hello"),
		NewAssistantMessage("Hi there!"),
	}
	event := NewUpdateConversationEvent(conversation)

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if parsed["type"] != "UpdateConversation" {
		t.Errorf("type = %v, want %q", parsed["type"], "UpdateConversation")
	}

	if _, ok := parsed["conversation"]; !ok {
		t.Error("conversation field should be present")
	}
}

func TestFinishReason_Values(t *testing.T) {
	reasons := []FinishReason{
		FinishReasonStop,
		FinishReasonLength,
		FinishReasonToolUse,
		FinishReasonContentFilter,
		FinishReasonError,
	}

	expected := []string{"stop", "length", "tool_use", "content_filter", "error"}

	for i, reason := range reasons {
		if string(reason) != expected[i] {
			t.Errorf("FinishReason[%d] = %q, want %q", i, reason, expected[i])
		}
	}
}
