package extension

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"
	"unicode"
)

// Extension represents a loaded extension with its client and configuration
type Extension struct {
	Config     ExtensionConfig
	Client     McpClient
	ServerInfo *InitializeResult
}

// PlatformExtensionContext provides context for platform extensions
type PlatformExtensionContext struct {
	SessionID        string
	ExtensionManager *Manager
	WorkingDir       string
}

// PlatformExtensionFactory creates a platform extension client
type PlatformExtensionFactory func(ctx PlatformExtensionContext) (McpClient, error)

// PlatformExtensionDef defines a built-in platform extension
type PlatformExtensionDef struct {
	Name           string
	Description    string
	DefaultEnabled bool
	Factory        PlatformExtensionFactory
}

// Manager manages extensions and their lifecycles
type Manager struct {
	extensions    map[string]*Extension // key -> Extension
	mu            sync.RWMutex
	sessionID     string
	workingDir    string
	platformDefs  []PlatformExtensionDef
	notifications chan ServerNotification
}

// NewManager creates a new extension manager
func NewManager(sessionID, workingDir string) *Manager {
	return &Manager{
		extensions:    make(map[string]*Extension),
		sessionID:     sessionID,
		workingDir:    workingDir,
		platformDefs:  DefaultPlatformExtensions(),
		notifications: make(chan ServerNotification, 100),
	}
}

// DefaultPlatformExtensions returns the default platform extension definitions
func DefaultPlatformExtensions() []PlatformExtensionDef {
	return []PlatformExtensionDef{
		{
			Name:           "todo",
			Description:    "Task list management for tracking and organizing todos",
			DefaultEnabled: true,
			Factory:        NewTodoExtension,
		},
		{
			Name:           "chatrecall",
			Description:    "Search conversations and session summaries",
			DefaultEnabled: false,
			Factory:        NewChatRecallExtension,
		},
		{
			Name:           "extensionmanager",
			Description:    "Discover and manage extensions",
			DefaultEnabled: true,
			Factory:        NewExtensionManagerExtension,
		},
		{
			Name:           "skills",
			Description:    "Load and execute skills from .goose/skills directory",
			DefaultEnabled: true,
			Factory:        NewSkillsExtension,
		},
	}
}

// AddExtension adds and initializes an extension
func (m *Manager) AddExtension(ctx context.Context, config ExtensionConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := config.Key()

	// Check if already exists
	if _, exists := m.extensions[key]; exists {
		return fmt.Errorf("extension %s already exists", key)
	}

	// Validate and merge environment variables
	envs := ValidateEnvs(config.Envs)

	// Resolve env keys from keychain (placeholder - in real impl would use keychain)
	for _, envKey := range config.EnvKeys {
		if !IsEnvKeyDisallowed(envKey) {
			if val := os.Getenv(envKey); val != "" {
				envs[envKey] = val
			}
		}
	}

	// Substitute environment variables in config
	config = substituteEnvVars(config, envs)

	// Create client based on type
	client, err := m.createClient(ctx, config, envs)
	if err != nil {
		return fmt.Errorf("failed to create client for %s: %w", key, err)
	}

	// Get server info
	serverInfo := client.GetInfo()

	m.extensions[key] = &Extension{
		Config:     config,
		Client:     client,
		ServerInfo: serverInfo,
	}

	// Start notification forwarding
	go m.forwardNotifications(key, client)

	return nil
}

// createClient creates the appropriate MCP client based on extension type
func (m *Manager) createClient(ctx context.Context, config ExtensionConfig, envs map[string]string) (McpClient, error) {
	switch config.Type {
	case ExtensionTypeSSE:
		return NewSSEClient(ctx, config.URI, envs, config.Timeout)

	case ExtensionTypeStdio:
		return NewStdioClient(ctx, config.Cmd, config.Args, envs, config.Timeout)

	case ExtensionTypeBuiltin:
		return NewBuiltinClient(ctx, config.Name, config.Timeout)

	case ExtensionTypePlatform:
		return m.createPlatformClient(config)

	case ExtensionTypeStreamableHTTP:
		return NewStreamableHTTPClient(ctx, config.URI, config.Headers, envs, config.Timeout)

	case ExtensionTypeFrontend:
		return NewFrontendClient(config.Name, config.Description, config.Tools, config.Instructions)

	case ExtensionTypeInlinePython:
		return NewInlinePythonClient(ctx, config.Code, config.Dependencies, config.Timeout)

	default:
		return nil, fmt.Errorf("unknown extension type: %s", config.Type)
	}
}

// createPlatformClient creates a platform extension client
func (m *Manager) createPlatformClient(config ExtensionConfig) (McpClient, error) {
	key := config.Key()

	for _, def := range m.platformDefs {
		if NameToKey(def.Name) == key {
			platformCtx := PlatformExtensionContext{
				SessionID:        m.sessionID,
				ExtensionManager: m,
				WorkingDir:       m.workingDir,
			}
			return def.Factory(platformCtx)
		}
	}

	return nil, fmt.Errorf("unknown platform extension: %s", key)
}

// forwardNotifications forwards notifications from an extension to the manager
func (m *Manager) forwardNotifications(key string, client McpClient) {
	ch := client.Subscribe()
	if ch == nil {
		return
	}

	for notification := range ch {
		select {
		case m.notifications <- notification:
		default:
			// Drop notification if buffer is full
		}
	}
}

// RemoveExtension removes an extension
func (m *Manager) RemoveExtension(key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	ext, exists := m.extensions[key]
	if !exists {
		return fmt.Errorf("extension %s not found", key)
	}

	if err := ext.Client.Close(); err != nil {
		return fmt.Errorf("failed to close extension %s: %w", key, err)
	}

	delete(m.extensions, key)
	return nil
}

// GetExtension returns an extension by key
func (m *Manager) GetExtension(key string) (*Extension, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	ext, ok := m.extensions[key]
	return ext, ok
}

// ListExtensions returns all loaded extensions
func (m *Manager) ListExtensions() []*Extension {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*Extension, 0, len(m.extensions))
	for _, ext := range m.extensions {
		result = append(result, ext)
	}
	return result
}

// GetPrefixedTools returns all tools with extension name prefix
func (m *Manager) GetPrefixedTools(ctx context.Context, extensionName *string) ([]Tool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var allTools []Tool
	var wg sync.WaitGroup
	var mu sync.Mutex
	errChan := make(chan error, len(m.extensions))

	for key, ext := range m.extensions {
		// Filter by extension name if specified
		if extensionName != nil && key != NameToKey(*extensionName) {
			continue
		}

		wg.Add(1)
		go func(k string, e *Extension) {
			defer wg.Done()

			tools, err := m.getExtensionTools(ctx, k, e)
			if err != nil {
				errChan <- fmt.Errorf("extension %s: %w", k, err)
				return
			}

			mu.Lock()
			allTools = append(allTools, tools...)
			mu.Unlock()
		}(key, ext)
	}

	wg.Wait()
	close(errChan)

	// Return first error if any
	for err := range errChan {
		return nil, err
	}

	return allTools, nil
}

// getExtensionTools gets tools from an extension with pagination
func (m *Manager) getExtensionTools(ctx context.Context, key string, ext *Extension) ([]Tool, error) {
	var allTools []Tool
	var cursor *string

	for {
		result, err := ext.Client.ListTools(ctx, cursor)
		if err != nil {
			return nil, err
		}

		// Filter and prefix tools
		for _, tool := range result.Tools {
			if ext.Config.IsToolAvailable(tool.Name) {
				prefixedTool := Tool{
					Name:        PrefixToolName(key, tool.Name),
					Description: tool.Description,
					InputSchema: tool.InputSchema,
				}
				allTools = append(allTools, prefixedTool)
			}
		}

		cursor = result.NextCursor
		if cursor == nil {
			break
		}
	}

	return allTools, nil
}

// CallTool dispatches a tool call to the appropriate extension
func (m *Manager) CallTool(ctx context.Context, prefixedName string, arguments json.RawMessage) (*CallToolResult, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Parse the prefixed tool name
	extKey, toolName, err := ParsePrefixedToolName(prefixedName)
	if err != nil {
		return nil, err
	}

	ext, exists := m.extensions[extKey]
	if !exists {
		return nil, fmt.Errorf("extension %s not found", extKey)
	}

	// Check if tool is available
	if !ext.Config.IsToolAvailable(toolName) {
		return nil, fmt.Errorf("tool %s is not available in extension %s", toolName, extKey)
	}

	return ext.Client.CallTool(ctx, toolName, arguments)
}

// GetResources lists resources from all extensions
func (m *Manager) GetResources(ctx context.Context, schemeFilter *string) ([]Resource, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var allResources []Resource
	var wg sync.WaitGroup
	var mu sync.Mutex
	errChan := make(chan error, len(m.extensions))

	for key, ext := range m.extensions {
		wg.Add(1)
		go func(k string, e *Extension) {
			defer wg.Done()

			resources, err := m.getExtensionResources(ctx, e, schemeFilter)
			if err != nil {
				errChan <- fmt.Errorf("extension %s: %w", k, err)
				return
			}

			mu.Lock()
			allResources = append(allResources, resources...)
			mu.Unlock()
		}(key, ext)
	}

	wg.Wait()
	close(errChan)

	for err := range errChan {
		return nil, err
	}

	return allResources, nil
}

// getExtensionResources gets resources from an extension with optional scheme filter
func (m *Manager) getExtensionResources(ctx context.Context, ext *Extension, schemeFilter *string) ([]Resource, error) {
	var allResources []Resource
	var cursor *string

	for {
		result, err := ext.Client.ListResources(ctx, cursor)
		if err != nil {
			return nil, err
		}

		for _, resource := range result.Resources {
			// Apply scheme filter if specified
			if schemeFilter != nil {
				if !strings.HasPrefix(resource.URI, *schemeFilter+"://") {
					continue
				}
			}
			allResources = append(allResources, resource)
		}

		cursor = result.NextCursor
		if cursor == nil {
			break
		}
	}

	return allResources, nil
}

// ReadResource reads a resource by URI
func (m *Manager) ReadResource(ctx context.Context, uri string) (*ReadResourceResult, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Try each extension until one succeeds
	for _, ext := range m.extensions {
		result, err := ext.Client.ReadResource(ctx, uri)
		if err == nil && len(result.Contents) > 0 {
			return result, nil
		}
	}

	return nil, fmt.Errorf("resource not found: %s", uri)
}

// GetExtensionInfo returns information about extensions for prompts
func (m *Manager) GetExtensionInfo() []ExtensionInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var infos []ExtensionInfo
	for _, ext := range m.extensions {
		info := ExtensionInfo{
			Name: ext.Config.Name,
		}

		if ext.ServerInfo != nil && ext.ServerInfo.Instructions != nil {
			info.Instructions = *ext.ServerInfo.Instructions
		}

		// Check if extension has resources
		if ext.ServerInfo != nil && ext.ServerInfo.Capabilities.Resources != nil {
			info.HasResources = true
		}

		infos = append(infos, info)
	}

	return infos
}

// Subscribe returns the notification channel
func (m *Manager) Subscribe() <-chan ServerNotification {
	return m.notifications
}

// Close closes all extensions
func (m *Manager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var lastErr error
	for key, ext := range m.extensions {
		if err := ext.Client.Close(); err != nil {
			lastErr = fmt.Errorf("failed to close extension %s: %w", key, err)
		}
	}

	m.extensions = make(map[string]*Extension)
	close(m.notifications)

	return lastErr
}

// PrefixToolName creates a prefixed tool name
func PrefixToolName(extensionKey, toolName string) string {
	return extensionKey + "__" + toolName
}

// ParsePrefixedToolName parses a prefixed tool name into extension key and tool name
func ParsePrefixedToolName(prefixedName string) (extensionKey, toolName string, err error) {
	parts := strings.SplitN(prefixedName, "__", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid prefixed tool name: %s", prefixedName)
	}
	return parts[0], parts[1], nil
}

// NormalizeExtensionKey normalizes a string for use as an extension key
// This handles special characters, unicode, and whitespace
func NormalizeExtensionKey(name string) string {
	var result strings.Builder

	for _, r := range name {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
			result.WriteRune(unicode.ToLower(r))
		} else if unicode.IsSpace(r) {
			// Skip whitespace
		} else {
			// Replace other characters with underscore
			result.WriteRune('_')
		}
	}

	key := result.String()

	// Remove consecutive underscores
	re := regexp.MustCompile(`_+`)
	key = re.ReplaceAllString(key, "_")

	// Trim leading/trailing underscores
	key = strings.Trim(key, "_")

	return key
}

// substituteEnvVars substitutes environment variables in config strings
func substituteEnvVars(config ExtensionConfig, envs map[string]string) ExtensionConfig {
	// Create regex for ${VAR} and $VAR patterns
	re := regexp.MustCompile(`\$\{([^}]+)\}|\$([A-Za-z_][A-Za-z0-9_]*)`)

	substitute := func(s string) string {
		return re.ReplaceAllStringFunc(s, func(match string) string {
			var varName string
			if strings.HasPrefix(match, "${") {
				varName = match[2 : len(match)-1]
			} else {
				varName = match[1:]
			}

			// First check provided envs
			if val, ok := envs[varName]; ok {
				return val
			}
			// Then check system env
			if val := os.Getenv(varName); val != "" {
				return val
			}
			return match // Keep original if not found
		})
	}

	// Substitute in relevant fields
	config.URI = substitute(config.URI)
	config.Cmd = substitute(config.Cmd)

	// Substitute in args
	for i, arg := range config.Args {
		config.Args[i] = substitute(arg)
	}

	// Substitute in headers
	if config.Headers != nil {
		newHeaders := make(map[string]string)
		for k, v := range config.Headers {
			newHeaders[k] = substitute(v)
		}
		config.Headers = newHeaders
	}

	return config
}
