package extension

import (
	"encoding/json"
	"testing"
)

func TestNameToKey(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple", "developer", "developer"},
		{"uppercase", "Developer", "developer"},
		{"with spaces", "My Extension", "myextension"},
		{"with tabs", "My\tExtension", "myextension"},
		{"mixed whitespace", " My \n Extension ", "myextension"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NameToKey(tt.input)
			if result != tt.expected {
				t.Errorf("NameToKey(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestExtensionConfig_Key(t *testing.T) {
	config := ExtensionConfig{
		Name: "My Test Extension",
	}

	expected := "mytestextension"
	if config.Key() != expected {
		t.Errorf("Key() = %q, want %q", config.Key(), expected)
	}
}

func TestExtensionConfig_IsToolAvailable(t *testing.T) {
	tests := []struct {
		name           string
		availableTools []string
		toolName       string
		expected       bool
	}{
		{"empty list allows all", nil, "any_tool", true},
		{"empty slice allows all", []string{}, "any_tool", true},
		{"specified tool allowed", []string{"tool1", "tool2"}, "tool1", true},
		{"unspecified tool denied", []string{"tool1", "tool2"}, "tool3", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := ExtensionConfig{
				AvailableTools: tt.availableTools,
			}
			result := config.IsToolAvailable(tt.toolName)
			if result != tt.expected {
				t.Errorf("IsToolAvailable(%q) = %v, want %v", tt.toolName, result, tt.expected)
			}
		})
	}
}

func TestNewSSEConfig(t *testing.T) {
	config := NewSSEConfig("test", "http://localhost:8080", "Test SSE", 60)

	if config.Type != ExtensionTypeSSE {
		t.Errorf("Type = %v, want %v", config.Type, ExtensionTypeSSE)
	}
	if config.Name != "test" {
		t.Errorf("Name = %v, want %v", config.Name, "test")
	}
	if config.URI != "http://localhost:8080" {
		t.Errorf("URI = %v, want %v", config.URI, "http://localhost:8080")
	}
	if *config.Timeout != 60 {
		t.Errorf("Timeout = %v, want %v", *config.Timeout, 60)
	}
}

func TestNewStdioConfig(t *testing.T) {
	config := NewStdioConfig("test", "/usr/bin/test", "Test Stdio", 120)

	if config.Type != ExtensionTypeStdio {
		t.Errorf("Type = %v, want %v", config.Type, ExtensionTypeStdio)
	}
	if config.Cmd != "/usr/bin/test" {
		t.Errorf("Cmd = %v, want %v", config.Cmd, "/usr/bin/test")
	}
	if len(config.Args) != 0 {
		t.Errorf("Args should be empty, got %v", config.Args)
	}
}

func TestNewBuiltinConfig(t *testing.T) {
	config := NewBuiltinConfig("developer", "Developer tools", 300)

	if config.Type != ExtensionTypeBuiltin {
		t.Errorf("Type = %v, want %v", config.Type, ExtensionTypeBuiltin)
	}
	if config.Bundled == nil || !*config.Bundled {
		t.Error("Bundled should be true for builtin")
	}
}

func TestNewPlatformConfig(t *testing.T) {
	config := NewPlatformConfig("todo", "Todo management")

	if config.Type != ExtensionTypePlatform {
		t.Errorf("Type = %v, want %v", config.Type, ExtensionTypePlatform)
	}
	if config.Bundled == nil || !*config.Bundled {
		t.Error("Bundled should be true for platform")
	}
}

func TestIsEnvKeyDisallowed(t *testing.T) {
	tests := []struct {
		key      string
		expected bool
	}{
		{"PATH", true},
		{"path", true}, // Case insensitive
		{"PATH_INFO", false},
		{"LD_PRELOAD", true},
		{"PYTHONPATH", true},
		{"MY_CUSTOM_VAR", false},
		{"NODE_OPTIONS", true},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			result := IsEnvKeyDisallowed(tt.key)
			if result != tt.expected {
				t.Errorf("IsEnvKeyDisallowed(%q) = %v, want %v", tt.key, result, tt.expected)
			}
		})
	}
}

func TestValidateEnvs(t *testing.T) {
	input := map[string]string{
		"MY_VAR":     "value1",
		"PATH":       "should be removed",
		"LD_PRELOAD": "should be removed",
		"SAFE_VAR":   "value2",
	}

	result := ValidateEnvs(input)

	if _, ok := result["PATH"]; ok {
		t.Error("PATH should be removed")
	}
	if _, ok := result["LD_PRELOAD"]; ok {
		t.Error("LD_PRELOAD should be removed")
	}
	if result["MY_VAR"] != "value1" {
		t.Errorf("MY_VAR = %v, want %v", result["MY_VAR"], "value1")
	}
	if result["SAFE_VAR"] != "value2" {
		t.Errorf("SAFE_VAR = %v, want %v", result["SAFE_VAR"], "value2")
	}
}

func TestExtensionConfigJSON(t *testing.T) {
	// Test SSE config serialization
	sseConfig := NewSSEConfig("test-sse", "http://example.com/sse", "Test SSE extension", 60)

	data, err := json.Marshal(sseConfig)
	if err != nil {
		t.Fatalf("Failed to marshal SSE config: %v", err)
	}

	var decoded ExtensionConfig
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal SSE config: %v", err)
	}

	if decoded.Type != ExtensionTypeSSE {
		t.Errorf("Type = %v, want %v", decoded.Type, ExtensionTypeSSE)
	}
	if decoded.URI != "http://example.com/sse" {
		t.Errorf("URI = %v, want %v", decoded.URI, "http://example.com/sse")
	}
}

func TestExtensionEntryJSON(t *testing.T) {
	config := NewPlatformConfig("todo", "Todo management")
	entry := ExtensionEntry{
		Enabled: true,
		Config:  config,
	}

	data, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("Failed to marshal entry: %v", err)
	}

	var decoded ExtensionEntry
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal entry: %v", err)
	}

	if !decoded.Enabled {
		t.Error("Enabled should be true")
	}
	if decoded.Config.Type != ExtensionTypePlatform {
		t.Errorf("Config.Type = %v, want %v", decoded.Config.Type, ExtensionTypePlatform)
	}
}
