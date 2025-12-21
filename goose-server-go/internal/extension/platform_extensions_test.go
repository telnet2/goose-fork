package extension

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestTodoExtension_ListTools(t *testing.T) {
	ctx := PlatformExtensionContext{
		SessionID:  "test-session",
		WorkingDir: "/tmp",
	}

	ext, err := NewTodoExtension(ctx)
	if err != nil {
		t.Fatalf("NewTodoExtension failed: %v", err)
	}

	result, err := ext.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListTools failed: %v", err)
	}

	expectedTools := []string{"list", "add", "update", "remove"}
	if len(result.Tools) != len(expectedTools) {
		t.Errorf("Expected %d tools, got %d", len(expectedTools), len(result.Tools))
	}

	for i, tool := range result.Tools {
		if tool.Name != expectedTools[i] {
			t.Errorf("Tool[%d].Name = %v, want %v", i, tool.Name, expectedTools[i])
		}
		if tool.InputSchema == nil {
			t.Errorf("Tool[%d].InputSchema is nil", i)
		}
	}
}

func TestTodoExtension_AddAndList(t *testing.T) {
	ctx := PlatformExtensionContext{
		SessionID:  "test-session",
		WorkingDir: "/tmp",
	}

	ext, err := NewTodoExtension(ctx)
	if err != nil {
		t.Fatalf("NewTodoExtension failed: %v", err)
	}

	bgCtx := context.Background()

	// List should be empty initially
	result, err := ext.CallTool(bgCtx, "list", json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if result.IsError {
		t.Error("list should not return error")
	}
	if result.Content[0].Text != nil {
		text := *result.Content[0].Text
		if !strings.Contains(text, "No todos") {
			t.Errorf("Expected 'No todos' message, got %v", text)
		}
	}

	// Add a todo
	addArgs := json.RawMessage(`{"content": "Test todo", "activeForm": "Testing"}`)
	result, err = ext.CallTool(bgCtx, "add", addArgs)
	if err != nil {
		t.Fatalf("add failed: %v", err)
	}
	if result.IsError {
		t.Error("add should not return error")
	}

	// List should show the todo
	result, err = ext.CallTool(bgCtx, "list", json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if result.Content[0].Text != nil {
		text := *result.Content[0].Text
		if !strings.Contains(text, "Test todo") {
			t.Errorf("Expected todo content in list, got %v", text)
		}
	}
}

func TestTodoExtension_UpdateStatus(t *testing.T) {
	ctx := PlatformExtensionContext{
		SessionID:  "test-session",
		WorkingDir: "/tmp",
	}

	ext, err := NewTodoExtension(ctx)
	if err != nil {
		t.Fatalf("NewTodoExtension failed: %v", err)
	}

	todoExt := ext.(*TodoExtension)
	bgCtx := context.Background()

	// Add a todo
	addArgs := json.RawMessage(`{"content": "Update test"}`)
	ext.CallTool(bgCtx, "add", addArgs)

	// Get the todo ID
	todoExt.mu.RLock()
	if len(todoExt.items) == 0 {
		t.Fatal("No todos added")
	}
	todoID := todoExt.items[0].ID
	todoExt.mu.RUnlock()

	// Update status
	updateArgs, _ := json.Marshal(map[string]string{
		"id":     todoID,
		"status": "in_progress",
	})
	result, err := ext.CallTool(bgCtx, "update", updateArgs)
	if err != nil {
		t.Fatalf("update failed: %v", err)
	}
	if result.IsError {
		t.Error("update should not return error")
	}

	// Verify status changed
	todoExt.mu.RLock()
	if todoExt.items[0].Status != "in_progress" {
		t.Errorf("Status = %v, want 'in_progress'", todoExt.items[0].Status)
	}
	todoExt.mu.RUnlock()
}

func TestTodoExtension_Remove(t *testing.T) {
	ctx := PlatformExtensionContext{
		SessionID:  "test-session",
		WorkingDir: "/tmp",
	}

	ext, err := NewTodoExtension(ctx)
	if err != nil {
		t.Fatalf("NewTodoExtension failed: %v", err)
	}

	todoExt := ext.(*TodoExtension)
	bgCtx := context.Background()

	// Add a todo
	ext.CallTool(bgCtx, "add", json.RawMessage(`{"content": "Remove test"}`))

	// Get the todo ID
	todoExt.mu.RLock()
	todoID := todoExt.items[0].ID
	todoExt.mu.RUnlock()

	// Remove the todo
	removeArgs, _ := json.Marshal(map[string]string{"id": todoID})
	result, err := ext.CallTool(bgCtx, "remove", removeArgs)
	if err != nil {
		t.Fatalf("remove failed: %v", err)
	}
	if result.IsError {
		t.Error("remove should not return error")
	}

	// Verify removed
	todoExt.mu.RLock()
	if len(todoExt.items) != 0 {
		t.Errorf("Expected 0 todos, got %d", len(todoExt.items))
	}
	todoExt.mu.RUnlock()
}

func TestTodoExtension_UpdateNotFound(t *testing.T) {
	ctx := PlatformExtensionContext{
		SessionID:  "test-session",
		WorkingDir: "/tmp",
	}

	ext, err := NewTodoExtension(ctx)
	if err != nil {
		t.Fatalf("NewTodoExtension failed: %v", err)
	}

	bgCtx := context.Background()

	// Try to update nonexistent todo
	updateArgs := json.RawMessage(`{"id": "nonexistent", "status": "completed"}`)
	result, err := ext.CallTool(bgCtx, "update", updateArgs)
	if err != nil {
		t.Fatalf("update failed: %v", err)
	}
	if !result.IsError {
		t.Error("update should return error for nonexistent todo")
	}
}

func TestTodoExtension_GetInfo(t *testing.T) {
	ctx := PlatformExtensionContext{
		SessionID:  "test-session",
		WorkingDir: "/tmp",
	}

	ext, err := NewTodoExtension(ctx)
	if err != nil {
		t.Fatalf("NewTodoExtension failed: %v", err)
	}

	info := ext.GetInfo()
	if info == nil {
		t.Fatal("GetInfo returned nil")
	}

	if info.ServerInfo.Name != "todo" {
		t.Errorf("ServerInfo.Name = %v, want 'todo'", info.ServerInfo.Name)
	}

	if info.Instructions == nil || *info.Instructions == "" {
		t.Error("Instructions should not be empty")
	}

	if info.Capabilities.Tools == nil {
		t.Error("Tools capability should not be nil")
	}
}

func TestChatRecallExtension_ListTools(t *testing.T) {
	ctx := PlatformExtensionContext{
		SessionID:  "test-session",
		WorkingDir: "/tmp",
	}

	ext, err := NewChatRecallExtension(ctx)
	if err != nil {
		t.Fatalf("NewChatRecallExtension failed: %v", err)
	}

	result, err := ext.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListTools failed: %v", err)
	}

	if len(result.Tools) != 2 {
		t.Errorf("Expected 2 tools, got %d", len(result.Tools))
	}
}

func TestExtensionManagerExtension_ListTools(t *testing.T) {
	ctx := PlatformExtensionContext{
		SessionID:  "test-session",
		WorkingDir: "/tmp",
	}

	ext, err := NewExtensionManagerExtension(ctx)
	if err != nil {
		t.Fatalf("NewExtensionManagerExtension failed: %v", err)
	}

	result, err := ext.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListTools failed: %v", err)
	}

	if len(result.Tools) != 3 {
		t.Errorf("Expected 3 tools, got %d", len(result.Tools))
	}

	expectedTools := map[string]bool{
		"list_extensions":    true,
		"get_extension_info": true,
		"list_available":     true,
	}

	for _, tool := range result.Tools {
		if !expectedTools[tool.Name] {
			t.Errorf("Unexpected tool: %s", tool.Name)
		}
	}
}

func TestExtensionManagerExtension_ListAvailable(t *testing.T) {
	ctx := PlatformExtensionContext{
		SessionID:  "test-session",
		WorkingDir: "/tmp",
	}

	ext, err := NewExtensionManagerExtension(ctx)
	if err != nil {
		t.Fatalf("NewExtensionManagerExtension failed: %v", err)
	}

	result, err := ext.CallTool(context.Background(), "list_available", json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("list_available failed: %v", err)
	}

	if result.IsError {
		t.Error("list_available should not return error")
	}

	if result.Content[0].Text != nil {
		text := *result.Content[0].Text
		if !strings.Contains(text, "todo") {
			t.Error("Expected 'todo' in available extensions")
		}
	}
}

func TestSkillsExtension_ListTools(t *testing.T) {
	ctx := PlatformExtensionContext{
		SessionID:  "test-session",
		WorkingDir: "/tmp",
	}

	ext, err := NewSkillsExtension(ctx)
	if err != nil {
		t.Fatalf("NewSkillsExtension failed: %v", err)
	}

	result, err := ext.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListTools failed: %v", err)
	}

	if len(result.Tools) != 3 {
		t.Errorf("Expected 3 tools, got %d", len(result.Tools))
	}
}

func TestSkillsExtension_ListSkillsEmpty(t *testing.T) {
	ctx := PlatformExtensionContext{
		SessionID:  "test-session",
		WorkingDir: "/tmp",
	}

	ext, err := NewSkillsExtension(ctx)
	if err != nil {
		t.Fatalf("NewSkillsExtension failed: %v", err)
	}

	result, err := ext.CallTool(context.Background(), "list_skills", json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("list_skills failed: %v", err)
	}

	if result.IsError {
		t.Error("list_skills should not return error")
	}

	// Should indicate no skills found since we haven't added any
	if result.Content[0].Text != nil {
		text := *result.Content[0].Text
		if !strings.Contains(text, "No skills found") {
			t.Errorf("Expected 'No skills found' message, got %v", text)
		}
	}
}

func TestSkillsExtension_GetSkillNotFound(t *testing.T) {
	ctx := PlatformExtensionContext{
		SessionID:  "test-session",
		WorkingDir: "/tmp",
	}

	ext, err := NewSkillsExtension(ctx)
	if err != nil {
		t.Fatalf("NewSkillsExtension failed: %v", err)
	}

	result, err := ext.CallTool(context.Background(), "get_skill", json.RawMessage(`{"name": "nonexistent"}`))
	if err != nil {
		t.Fatalf("get_skill failed: %v", err)
	}

	if !result.IsError {
		t.Error("get_skill should return error for nonexistent skill")
	}
}

func TestSuccessResult(t *testing.T) {
	result, err := successResult("Test message")
	if err != nil {
		t.Fatalf("successResult failed: %v", err)
	}

	if result.IsError {
		t.Error("IsError should be false")
	}

	if len(result.Content) != 1 {
		t.Errorf("Expected 1 content item, got %d", len(result.Content))
	}

	if result.Content[0].Type != "text" {
		t.Errorf("Content type = %v, want 'text'", result.Content[0].Type)
	}

	if result.Content[0].Text == nil || *result.Content[0].Text != "Test message" {
		t.Errorf("Content text = %v, want 'Test message'", result.Content[0].Text)
	}
}

func TestErrorResult(t *testing.T) {
	result, err := errorResult("Error message")
	if err != nil {
		t.Fatalf("errorResult failed: %v", err)
	}

	if !result.IsError {
		t.Error("IsError should be true")
	}

	if result.Content[0].Text == nil || *result.Content[0].Text != "Error message" {
		t.Errorf("Content text = %v, want 'Error message'", result.Content[0].Text)
	}
}

func TestBaseClient(t *testing.T) {
	info := &InitializeResult{
		ProtocolVersion: CurrentProtocolVersion,
		ServerInfo: Implementation{
			Name:    "test",
			Version: "1.0.0",
		},
	}

	client := NewBaseClient(info)

	// Test GetInfo
	if client.GetInfo() != info {
		t.Error("GetInfo should return the provided info")
	}

	// Test GetMoim
	if client.GetMoim() != nil {
		t.Error("GetMoim should return nil by default")
	}

	// Test Subscribe
	ch := client.Subscribe()
	if ch == nil {
		t.Error("Subscribe should return a channel")
	}

	// Test Close
	if err := client.Close(); err != nil {
		t.Errorf("Close failed: %v", err)
	}

	// Test double close is safe
	if err := client.Close(); err != nil {
		t.Errorf("Double close failed: %v", err)
	}
}
