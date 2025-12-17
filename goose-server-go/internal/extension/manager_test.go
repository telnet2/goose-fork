package extension

import (
	"context"
	"encoding/json"
	"testing"
)

func TestPrefixToolName(t *testing.T) {
	tests := []struct {
		extensionKey string
		toolName     string
		expected     string
	}{
		{"developer", "shell", "developer__shell"},
		{"my_ext", "my_tool", "my_ext__my_tool"},
		{"ext", "tool_with_underscores", "ext__tool_with_underscores"},
	}

	for _, tt := range tests {
		result := PrefixToolName(tt.extensionKey, tt.toolName)
		if result != tt.expected {
			t.Errorf("PrefixToolName(%q, %q) = %q, want %q",
				tt.extensionKey, tt.toolName, result, tt.expected)
		}
	}
}

func TestParsePrefixedToolName(t *testing.T) {
	tests := []struct {
		input       string
		expectKey   string
		expectTool  string
		expectError bool
	}{
		{"developer__shell", "developer", "shell", false},
		{"ext__tool", "ext", "tool", false},
		{"invalid_no_separator", "", "", true},
		{"ext__tool__extra", "ext", "tool__extra", false}, // Only split on first __
	}

	for _, tt := range tests {
		key, tool, err := ParsePrefixedToolName(tt.input)

		if tt.expectError {
			if err == nil {
				t.Errorf("ParsePrefixedToolName(%q) expected error, got nil", tt.input)
			}
			continue
		}

		if err != nil {
			t.Errorf("ParsePrefixedToolName(%q) unexpected error: %v", tt.input, err)
			continue
		}

		if key != tt.expectKey {
			t.Errorf("ParsePrefixedToolName(%q) key = %q, want %q", tt.input, key, tt.expectKey)
		}
		if tool != tt.expectTool {
			t.Errorf("ParsePrefixedToolName(%q) tool = %q, want %q", tt.input, tool, tt.expectTool)
		}
	}
}

func TestNormalizeExtensionKey(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"simple", "simple"},
		{"with spaces", "withspaces"},
		{"With_Underscores", "with_underscores"},
		{"special!@#chars", "special_chars"},
		{"emoji🚀test", "emoji_test"},
		{"___leading_trailing___", "leading_trailing"},
		{"UPPERCASE", "uppercase"},
		{"mixed CASE with Spaces", "mixedcasewithspaces"},
	}

	for _, tt := range tests {
		result := NormalizeExtensionKey(tt.input)
		if result != tt.expected {
			t.Errorf("NormalizeExtensionKey(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestNewManager(t *testing.T) {
	manager := NewManager("session-123", "/tmp/work")

	if manager.sessionID != "session-123" {
		t.Errorf("sessionID = %v, want session-123", manager.sessionID)
	}
	if manager.workingDir != "/tmp/work" {
		t.Errorf("workingDir = %v, want /tmp/work", manager.workingDir)
	}
	if len(manager.extensions) != 0 {
		t.Errorf("extensions should be empty, got %d", len(manager.extensions))
	}
	if len(manager.platformDefs) == 0 {
		t.Error("platformDefs should not be empty")
	}
}

func TestDefaultPlatformExtensions(t *testing.T) {
	defs := DefaultPlatformExtensions()

	expectedNames := []string{"todo", "chatrecall", "extensionmanager", "skills"}

	if len(defs) != len(expectedNames) {
		t.Errorf("got %d platform extensions, want %d", len(defs), len(expectedNames))
	}

	for i, def := range defs {
		if def.Name != expectedNames[i] {
			t.Errorf("def[%d].Name = %v, want %v", i, def.Name, expectedNames[i])
		}
		if def.Factory == nil {
			t.Errorf("def[%d].Factory is nil", i)
		}
	}
}

func TestManager_AddPlatformExtension(t *testing.T) {
	manager := NewManager("session-123", "/tmp/work")
	ctx := context.Background()

	// Add todo platform extension
	config := NewPlatformConfig("todo", "Todo management")
	err := manager.AddExtension(ctx, config)
	if err != nil {
		t.Fatalf("AddExtension failed: %v", err)
	}

	// Verify extension was added
	ext, ok := manager.GetExtension("todo")
	if !ok {
		t.Fatal("GetExtension returned false for 'todo'")
	}
	if ext.Config.Name != "todo" {
		t.Errorf("Config.Name = %v, want 'todo'", ext.Config.Name)
	}

	// Cleanup
	manager.Close()
}

func TestManager_RemoveExtension(t *testing.T) {
	manager := NewManager("session-123", "/tmp/work")
	ctx := context.Background()

	// Add extension
	config := NewPlatformConfig("todo", "Todo management")
	if err := manager.AddExtension(ctx, config); err != nil {
		t.Fatalf("AddExtension failed: %v", err)
	}

	// Remove extension
	if err := manager.RemoveExtension("todo"); err != nil {
		t.Fatalf("RemoveExtension failed: %v", err)
	}

	// Verify removed
	_, ok := manager.GetExtension("todo")
	if ok {
		t.Error("Extension should be removed")
	}
}

func TestManager_ListExtensions(t *testing.T) {
	manager := NewManager("session-123", "/tmp/work")
	ctx := context.Background()

	// Add multiple extensions
	configs := []ExtensionConfig{
		NewPlatformConfig("todo", "Todo management"),
		NewPlatformConfig("skills", "Skills management"),
	}

	for _, config := range configs {
		if err := manager.AddExtension(ctx, config); err != nil {
			t.Fatalf("AddExtension failed: %v", err)
		}
	}

	// List extensions
	extensions := manager.ListExtensions()
	if len(extensions) != 2 {
		t.Errorf("ListExtensions returned %d, want 2", len(extensions))
	}

	// Cleanup
	manager.Close()
}

func TestManager_GetPrefixedTools(t *testing.T) {
	manager := NewManager("session-123", "/tmp/work")
	ctx := context.Background()

	// Add todo extension
	config := NewPlatformConfig("todo", "Todo management")
	if err := manager.AddExtension(ctx, config); err != nil {
		t.Fatalf("AddExtension failed: %v", err)
	}

	// Get prefixed tools
	tools, err := manager.GetPrefixedTools(ctx, nil)
	if err != nil {
		t.Fatalf("GetPrefixedTools failed: %v", err)
	}

	if len(tools) == 0 {
		t.Error("Expected tools from todo extension")
	}

	// Verify tools are prefixed
	for _, tool := range tools {
		if len(tool.Name) < 6 || tool.Name[:6] != "todo__" {
			t.Errorf("Tool %q should be prefixed with 'todo__'", tool.Name)
		}
	}

	// Cleanup
	manager.Close()
}

func TestManager_CallTool(t *testing.T) {
	manager := NewManager("session-123", "/tmp/work")
	ctx := context.Background()

	// Add todo extension
	config := NewPlatformConfig("todo", "Todo management")
	if err := manager.AddExtension(ctx, config); err != nil {
		t.Fatalf("AddExtension failed: %v", err)
	}

	// Call list tool
	result, err := manager.CallTool(ctx, "todo__list", json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("CallTool failed: %v", err)
	}

	if result == nil {
		t.Error("Result should not be nil")
	}

	// Cleanup
	manager.Close()
}

func TestManager_GetExtensionInfo(t *testing.T) {
	manager := NewManager("session-123", "/tmp/work")
	ctx := context.Background()

	// Add todo extension
	config := NewPlatformConfig("todo", "Todo management")
	if err := manager.AddExtension(ctx, config); err != nil {
		t.Fatalf("AddExtension failed: %v", err)
	}

	// Get extension info
	infos := manager.GetExtensionInfo()

	if len(infos) != 1 {
		t.Errorf("Expected 1 extension info, got %d", len(infos))
	}

	if infos[0].Name != "todo" {
		t.Errorf("Name = %v, want 'todo'", infos[0].Name)
	}

	// Cleanup
	manager.Close()
}

func TestManager_DuplicateExtension(t *testing.T) {
	manager := NewManager("session-123", "/tmp/work")
	ctx := context.Background()

	config := NewPlatformConfig("todo", "Todo management")

	// Add first time
	if err := manager.AddExtension(ctx, config); err != nil {
		t.Fatalf("First AddExtension failed: %v", err)
	}

	// Add second time should fail
	err := manager.AddExtension(ctx, config)
	if err == nil {
		t.Error("Second AddExtension should fail")
	}

	// Cleanup
	manager.Close()
}

func TestManager_CallToolNotFound(t *testing.T) {
	manager := NewManager("session-123", "/tmp/work")
	ctx := context.Background()

	// Call tool without any extensions
	_, err := manager.CallTool(ctx, "nonexistent__tool", json.RawMessage(`{}`))
	if err == nil {
		t.Error("CallTool should fail for nonexistent extension")
	}
}

func TestManager_RemoveNonexistent(t *testing.T) {
	manager := NewManager("session-123", "/tmp/work")

	err := manager.RemoveExtension("nonexistent")
	if err == nil {
		t.Error("RemoveExtension should fail for nonexistent")
	}
}

func TestSubstituteEnvVars(t *testing.T) {
	// Set test env var
	envs := map[string]string{
		"TEST_HOST": "localhost",
		"TEST_PORT": "8080",
	}

	config := ExtensionConfig{
		URI: "http://${TEST_HOST}:$TEST_PORT/api",
		Cmd: "${TEST_HOST}",
		Args: []string{
			"--host=$TEST_HOST",
			"--port=${TEST_PORT}",
		},
		Headers: map[string]string{
			"X-Host": "${TEST_HOST}",
		},
	}

	result := substituteEnvVars(config, envs)

	if result.URI != "http://localhost:8080/api" {
		t.Errorf("URI = %v, want 'http://localhost:8080/api'", result.URI)
	}

	if result.Cmd != "localhost" {
		t.Errorf("Cmd = %v, want 'localhost'", result.Cmd)
	}

	if result.Args[0] != "--host=localhost" {
		t.Errorf("Args[0] = %v, want '--host=localhost'", result.Args[0])
	}

	if result.Headers["X-Host"] != "localhost" {
		t.Errorf("Headers[X-Host] = %v, want 'localhost'", result.Headers["X-Host"])
	}
}
