package extension

import (
	"encoding/json"
	"strings"
)

// ExtensionType represents the type of extension
type ExtensionType string

const (
	ExtensionTypeSSE            ExtensionType = "sse"
	ExtensionTypeStdio          ExtensionType = "stdio"
	ExtensionTypeBuiltin        ExtensionType = "builtin"
	ExtensionTypePlatform       ExtensionType = "platform"
	ExtensionTypeStreamableHTTP ExtensionType = "streamable_http"
	ExtensionTypeFrontend       ExtensionType = "frontend"
	ExtensionTypeInlinePython   ExtensionType = "inline_python"
)

// ExtensionConfig represents the configuration for an extension
// This is a discriminated union based on the Type field
type ExtensionConfig struct {
	Type ExtensionType `json:"type"`

	// Common fields
	Name           string            `json:"name"`
	Description    string            `json:"description"`
	Timeout        *uint64           `json:"timeout,omitempty"`
	Bundled        *bool             `json:"bundled,omitempty"`
	AvailableTools []string          `json:"available_tools,omitempty"`
	Envs           map[string]string `json:"envs,omitempty"`
	EnvKeys        []string          `json:"env_keys,omitempty"`

	// For SSE and StreamableHTTP
	URI string `json:"uri,omitempty"`

	// For StreamableHTTP
	Headers map[string]string `json:"headers,omitempty"`

	// For Stdio
	Cmd  string   `json:"cmd,omitempty"`
	Args []string `json:"args,omitempty"`

	// For Builtin
	DisplayName *string `json:"display_name,omitempty"`

	// For Frontend
	Tools        []json.RawMessage `json:"tools,omitempty"`
	Instructions *string           `json:"instructions,omitempty"`

	// For InlinePython
	Code         string   `json:"code,omitempty"`
	Dependencies []string `json:"dependencies,omitempty"`
}

// Key returns a normalized key for the extension name
func (c *ExtensionConfig) Key() string {
	return NameToKey(c.Name)
}

// IsToolAvailable checks if a tool should be available to the LLM
func (c *ExtensionConfig) IsToolAvailable(toolName string) bool {
	// If no tools are specified, all tools are available
	if len(c.AvailableTools) == 0 {
		return true
	}
	// If tools are specified, only those tools are available
	for _, t := range c.AvailableTools {
		if t == toolName {
			return true
		}
	}
	return false
}

// NewSSEConfig creates a new SSE extension config
func NewSSEConfig(name, uri, description string, timeout uint64) ExtensionConfig {
	return ExtensionConfig{
		Type:        ExtensionTypeSSE,
		Name:        name,
		URI:         uri,
		Description: description,
		Timeout:     &timeout,
	}
}

// NewStdioConfig creates a new Stdio extension config
func NewStdioConfig(name, cmd, description string, timeout uint64) ExtensionConfig {
	return ExtensionConfig{
		Type:        ExtensionTypeStdio,
		Name:        name,
		Cmd:         cmd,
		Description: description,
		Timeout:     &timeout,
		Args:        []string{},
	}
}

// NewBuiltinConfig creates a new Builtin extension config
func NewBuiltinConfig(name, description string, timeout uint64) ExtensionConfig {
	bundled := true
	return ExtensionConfig{
		Type:        ExtensionTypeBuiltin,
		Name:        name,
		Description: description,
		Timeout:     &timeout,
		Bundled:     &bundled,
	}
}

// NewPlatformConfig creates a new Platform extension config
func NewPlatformConfig(name, description string) ExtensionConfig {
	bundled := true
	return ExtensionConfig{
		Type:        ExtensionTypePlatform,
		Name:        name,
		Description: description,
		Bundled:     &bundled,
	}
}

// ExtensionEntry represents an extension with enabled state
type ExtensionEntry struct {
	Enabled bool            `json:"enabled"`
	Config  ExtensionConfig `json:"config"`
}

// ExtensionInfo contains information about an extension for prompts
type ExtensionInfo struct {
	Name         string `json:"name"`
	Instructions string `json:"instructions"`
	HasResources bool   `json:"hasResources"`
}

// NameToKey normalizes an extension name to a key
func NameToKey(name string) string {
	// Remove whitespace and convert to lowercase
	var result strings.Builder
	for _, r := range name {
		if r != ' ' && r != '\t' && r != '\n' && r != '\r' {
			result.WriteRune(r)
		}
	}
	return strings.ToLower(result.String())
}

// Default extension constants
const (
	DefaultExtension            = "developer"
	DefaultExtensionTimeout     = 300
	DefaultExtensionDescription = ""
	DefaultDisplayName          = "Developer"
)

// DisallowedEnvKeys is a list of environment variables that should not be overridden
var DisallowedEnvKeys = []string{
	// Binary path manipulation
	"PATH", "PATHEXT", "SystemRoot", "windir",
	// Dynamic linker hijacking (Linux/macOS)
	"LD_LIBRARY_PATH", "LD_PRELOAD", "LD_AUDIT", "LD_DEBUG", "LD_BIND_NOW", "LD_ASSUME_KERNEL",
	// macOS dynamic linker variables
	"DYLD_LIBRARY_PATH", "DYLD_INSERT_LIBRARIES", "DYLD_FRAMEWORK_PATH",
	// Python / Node / Ruby / Java / Golang hijacking
	"PYTHONPATH", "PYTHONHOME", "NODE_OPTIONS", "RUBYOPT", "GEM_PATH", "GEM_HOME",
	"CLASSPATH", "GO111MODULE", "GOROOT",
	// Windows-specific process & DLL hijacking
	"APPINIT_DLLS", "SESSIONNAME", "ComSpec", "TEMP", "TMP", "LOCALAPPDATA",
	"USERPROFILE", "HOMEDRIVE", "HOMEPATH",
}

// IsEnvKeyDisallowed checks if an environment key is in the disallowed list
func IsEnvKeyDisallowed(key string) bool {
	keyLower := strings.ToLower(key)
	for _, disallowed := range DisallowedEnvKeys {
		if strings.ToLower(disallowed) == keyLower {
			return true
		}
	}
	return false
}

// ValidateEnvs validates environment variables and removes disallowed ones
func ValidateEnvs(envs map[string]string) map[string]string {
	validated := make(map[string]string)
	for key, value := range envs {
		if !IsEnvKeyDisallowed(key) {
			validated[key] = value
		}
	}
	return validated
}
