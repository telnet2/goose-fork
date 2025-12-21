package extension

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"sync"
	"time"
)

// TodoItem represents a single todo item
type TodoItem struct {
	ID          string    `json:"id"`
	Content     string    `json:"content"`
	Status      string    `json:"status"` // pending, in_progress, completed
	ActiveForm  string    `json:"activeForm,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
	CompletedAt *time.Time `json:"completedAt,omitempty"`
}

// TodoExtension implements the todo list platform extension
type TodoExtension struct {
	*BaseClient
	ctx     PlatformExtensionContext
	items   []TodoItem
	mu      sync.RWMutex
}

// NewTodoExtension creates a new todo extension
func NewTodoExtension(ctx PlatformExtensionContext) (McpClient, error) {
	instructions := `# Todo Extension

This extension provides task list management for tracking and organizing todos.

## Available Tools

- **todo_list**: List all todos with their current status
- **todo_add**: Add a new todo item
- **todo_update**: Update an existing todo's status or content
- **todo_remove**: Remove a todo item

## Usage Notes

- Use todo_add when starting a new task or set of tasks
- Update status to "in_progress" when actively working on a task
- Mark as "completed" when finished
- Keep only one task in_progress at a time for clarity`

	ext := &TodoExtension{
		ctx:   ctx,
		items: make([]TodoItem, 0),
		BaseClient: NewBaseClient(&InitializeResult{
			ProtocolVersion: CurrentProtocolVersion,
			Capabilities: ServerCapabilities{
				Tools: &ToolsCapability{ListChanged: false},
			},
			ServerInfo: Implementation{
				Name:    "todo",
				Version: "1.0.0",
			},
			Instructions: &instructions,
		}),
	}

	return ext, nil
}

// ListResources returns empty for todo extension
func (t *TodoExtension) ListResources(ctx context.Context, cursor *string) (*ListResourcesResult, error) {
	return &ListResourcesResult{Resources: []Resource{}}, nil
}

// ReadResource returns error for todo extension
func (t *TodoExtension) ReadResource(ctx context.Context, uri string) (*ReadResourceResult, error) {
	return nil, fmt.Errorf("todo extension does not support resources")
}

// ListTools returns available todo tools
func (t *TodoExtension) ListTools(ctx context.Context, cursor *string) (*ListToolsResult, error) {
	listSchema := json.RawMessage(`{
		"type": "object",
		"properties": {},
		"additionalProperties": false
	}`)

	addSchema := json.RawMessage(`{
		"type": "object",
		"properties": {
			"content": {"type": "string", "description": "The todo item content"},
			"activeForm": {"type": "string", "description": "Active form description (e.g., 'Running tests')"}
		},
		"required": ["content"],
		"additionalProperties": false
	}`)

	updateSchema := json.RawMessage(`{
		"type": "object",
		"properties": {
			"id": {"type": "string", "description": "The todo ID to update"},
			"status": {"type": "string", "enum": ["pending", "in_progress", "completed"], "description": "New status"},
			"content": {"type": "string", "description": "New content (optional)"}
		},
		"required": ["id"],
		"additionalProperties": false
	}`)

	removeSchema := json.RawMessage(`{
		"type": "object",
		"properties": {
			"id": {"type": "string", "description": "The todo ID to remove"}
		},
		"required": ["id"],
		"additionalProperties": false
	}`)

	tools := []Tool{
		{
			Name:        "list",
			Description: "List all todo items with their current status",
			InputSchema: &listSchema,
		},
		{
			Name:        "add",
			Description: "Add a new todo item",
			InputSchema: &addSchema,
		},
		{
			Name:        "update",
			Description: "Update an existing todo item's status or content",
			InputSchema: &updateSchema,
		},
		{
			Name:        "remove",
			Description: "Remove a todo item",
			InputSchema: &removeSchema,
		},
	}

	return &ListToolsResult{Tools: tools}, nil
}

// CallTool executes a todo tool
func (t *TodoExtension) CallTool(ctx context.Context, name string, arguments json.RawMessage) (*CallToolResult, error) {
	switch name {
	case "list":
		return t.listTodos()
	case "add":
		var args struct {
			Content    string `json:"content"`
			ActiveForm string `json:"activeForm"`
		}
		if err := json.Unmarshal(arguments, &args); err != nil {
			return errorResult(fmt.Sprintf("invalid arguments: %v", err))
		}
		return t.addTodo(args.Content, args.ActiveForm)
	case "update":
		var args struct {
			ID      string  `json:"id"`
			Status  *string `json:"status"`
			Content *string `json:"content"`
		}
		if err := json.Unmarshal(arguments, &args); err != nil {
			return errorResult(fmt.Sprintf("invalid arguments: %v", err))
		}
		return t.updateTodo(args.ID, args.Status, args.Content)
	case "remove":
		var args struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(arguments, &args); err != nil {
			return errorResult(fmt.Sprintf("invalid arguments: %v", err))
		}
		return t.removeTodo(args.ID)
	default:
		return errorResult(fmt.Sprintf("unknown tool: %s", name))
	}
}

func (t *TodoExtension) listTodos() (*CallToolResult, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if len(t.items) == 0 {
		return successResult("No todos found.")
	}

	// Format todos as text
	var result string
	for i, item := range t.items {
		statusIcon := "○"
		if item.Status == "in_progress" {
			statusIcon = "●"
		} else if item.Status == "completed" {
			statusIcon = "✓"
		}
		result += fmt.Sprintf("%d. [%s] %s (%s)\n", i+1, statusIcon, item.Content, item.Status)
		if item.ID != "" {
			result += fmt.Sprintf("   ID: %s\n", item.ID)
		}
	}

	return successResult(result)
}

func (t *TodoExtension) addTodo(content, activeForm string) (*CallToolResult, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	id := fmt.Sprintf("todo-%d", time.Now().UnixNano())
	item := TodoItem{
		ID:         id,
		Content:    content,
		Status:     "pending",
		ActiveForm: activeForm,
		CreatedAt:  time.Now(),
	}

	t.items = append(t.items, item)

	return successResult(fmt.Sprintf("Added todo: %s (ID: %s)", content, id))
}

func (t *TodoExtension) updateTodo(id string, status, content *string) (*CallToolResult, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	for i, item := range t.items {
		if item.ID == id {
			if status != nil {
				t.items[i].Status = *status
				if *status == "completed" {
					now := time.Now()
					t.items[i].CompletedAt = &now
				}
			}
			if content != nil {
				t.items[i].Content = *content
			}
			return successResult(fmt.Sprintf("Updated todo: %s", id))
		}
	}

	return errorResult(fmt.Sprintf("Todo not found: %s", id))
}

func (t *TodoExtension) removeTodo(id string) (*CallToolResult, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	for i, item := range t.items {
		if item.ID == id {
			t.items = append(t.items[:i], t.items[i+1:]...)
			return successResult(fmt.Sprintf("Removed todo: %s", id))
		}
	}

	return errorResult(fmt.Sprintf("Todo not found: %s", id))
}

// ListPrompts returns empty for todo extension
func (t *TodoExtension) ListPrompts(ctx context.Context, cursor *string) (*ListPromptsResult, error) {
	return &ListPromptsResult{Prompts: []Prompt{}}, nil
}

// GetPrompt returns error for todo extension
func (t *TodoExtension) GetPrompt(ctx context.Context, name string, arguments map[string]string) (*GetPromptResult, error) {
	return nil, fmt.Errorf("todo extension does not support prompts")
}

// ChatRecallExtension implements conversation search functionality
type ChatRecallExtension struct {
	*BaseClient
	ctx PlatformExtensionContext
}

// NewChatRecallExtension creates a new chat recall extension
func NewChatRecallExtension(ctx PlatformExtensionContext) (McpClient, error) {
	instructions := `# Chat Recall Extension

This extension allows searching through conversation history and session summaries.

## Available Tools

- **search**: Search conversations for specific terms
- **get_summary**: Get a summary of a specific session`

	ext := &ChatRecallExtension{
		ctx: ctx,
		BaseClient: NewBaseClient(&InitializeResult{
			ProtocolVersion: CurrentProtocolVersion,
			Capabilities: ServerCapabilities{
				Tools: &ToolsCapability{},
			},
			ServerInfo: Implementation{
				Name:    "chatrecall",
				Version: "1.0.0",
			},
			Instructions: &instructions,
		}),
	}

	return ext, nil
}

// ListResources returns empty
func (c *ChatRecallExtension) ListResources(ctx context.Context, cursor *string) (*ListResourcesResult, error) {
	return &ListResourcesResult{Resources: []Resource{}}, nil
}

// ReadResource returns error
func (c *ChatRecallExtension) ReadResource(ctx context.Context, uri string) (*ReadResourceResult, error) {
	return nil, fmt.Errorf("chatrecall extension does not support resources")
}

// ListTools returns available tools
func (c *ChatRecallExtension) ListTools(ctx context.Context, cursor *string) (*ListToolsResult, error) {
	searchSchema := json.RawMessage(`{
		"type": "object",
		"properties": {
			"query": {"type": "string", "description": "Search query"},
			"limit": {"type": "integer", "description": "Maximum results to return", "default": 10}
		},
		"required": ["query"],
		"additionalProperties": false
	}`)

	summarySchema := json.RawMessage(`{
		"type": "object",
		"properties": {
			"session_id": {"type": "string", "description": "Session ID to summarize"}
		},
		"required": ["session_id"],
		"additionalProperties": false
	}`)

	tools := []Tool{
		{
			Name:        "search",
			Description: "Search conversations for specific terms",
			InputSchema: &searchSchema,
		},
		{
			Name:        "get_summary",
			Description: "Get a summary of a specific session",
			InputSchema: &summarySchema,
		},
	}

	return &ListToolsResult{Tools: tools}, nil
}

// CallTool executes a chat recall tool
func (c *ChatRecallExtension) CallTool(ctx context.Context, name string, arguments json.RawMessage) (*CallToolResult, error) {
	// Placeholder implementation
	return successResult("Chat recall functionality not yet implemented")
}

// ListPrompts returns empty
func (c *ChatRecallExtension) ListPrompts(ctx context.Context, cursor *string) (*ListPromptsResult, error) {
	return &ListPromptsResult{Prompts: []Prompt{}}, nil
}

// GetPrompt returns error
func (c *ChatRecallExtension) GetPrompt(ctx context.Context, name string, arguments map[string]string) (*GetPromptResult, error) {
	return nil, fmt.Errorf("chatrecall extension does not support prompts")
}

// ExtensionManagerExtension allows discovering and managing extensions
type ExtensionManagerExtension struct {
	*BaseClient
	ctx PlatformExtensionContext
}

// NewExtensionManagerExtension creates a new extension manager extension
func NewExtensionManagerExtension(ctx PlatformExtensionContext) (McpClient, error) {
	instructions := `# Extension Manager

This extension allows discovering and managing other extensions.

## Available Tools

- **list_extensions**: List all loaded extensions
- **get_extension_info**: Get detailed information about an extension
- **list_available**: List extensions available for installation`

	ext := &ExtensionManagerExtension{
		ctx: ctx,
		BaseClient: NewBaseClient(&InitializeResult{
			ProtocolVersion: CurrentProtocolVersion,
			Capabilities: ServerCapabilities{
				Tools: &ToolsCapability{},
			},
			ServerInfo: Implementation{
				Name:    "extensionmanager",
				Version: "1.0.0",
			},
			Instructions: &instructions,
		}),
	}

	return ext, nil
}

// ListResources returns empty
func (e *ExtensionManagerExtension) ListResources(ctx context.Context, cursor *string) (*ListResourcesResult, error) {
	return &ListResourcesResult{Resources: []Resource{}}, nil
}

// ReadResource returns error
func (e *ExtensionManagerExtension) ReadResource(ctx context.Context, uri string) (*ReadResourceResult, error) {
	return nil, fmt.Errorf("extension manager does not support resources")
}

// ListTools returns available tools
func (e *ExtensionManagerExtension) ListTools(ctx context.Context, cursor *string) (*ListToolsResult, error) {
	listSchema := json.RawMessage(`{
		"type": "object",
		"properties": {},
		"additionalProperties": false
	}`)

	infoSchema := json.RawMessage(`{
		"type": "object",
		"properties": {
			"name": {"type": "string", "description": "Extension name"}
		},
		"required": ["name"],
		"additionalProperties": false
	}`)

	tools := []Tool{
		{
			Name:        "list_extensions",
			Description: "List all loaded extensions",
			InputSchema: &listSchema,
		},
		{
			Name:        "get_extension_info",
			Description: "Get detailed information about an extension",
			InputSchema: &infoSchema,
		},
		{
			Name:        "list_available",
			Description: "List extensions available for installation",
			InputSchema: &listSchema,
		},
	}

	return &ListToolsResult{Tools: tools}, nil
}

// CallTool executes an extension manager tool
func (e *ExtensionManagerExtension) CallTool(ctx context.Context, name string, arguments json.RawMessage) (*CallToolResult, error) {
	switch name {
	case "list_extensions":
		return e.listExtensions()
	case "get_extension_info":
		var args struct {
			Name string `json:"name"`
		}
		if err := json.Unmarshal(arguments, &args); err != nil {
			return errorResult(fmt.Sprintf("invalid arguments: %v", err))
		}
		return e.getExtensionInfo(args.Name)
	case "list_available":
		return e.listAvailable()
	default:
		return errorResult(fmt.Sprintf("unknown tool: %s", name))
	}
}

func (e *ExtensionManagerExtension) listExtensions() (*CallToolResult, error) {
	if e.ctx.ExtensionManager == nil {
		return successResult("No extensions loaded")
	}

	extensions := e.ctx.ExtensionManager.ListExtensions()
	if len(extensions) == 0 {
		return successResult("No extensions loaded")
	}

	var result string
	for _, ext := range extensions {
		result += fmt.Sprintf("- %s (%s): %s\n", ext.Config.Name, ext.Config.Type, ext.Config.Description)
	}

	return successResult(result)
}

func (e *ExtensionManagerExtension) getExtensionInfo(name string) (*CallToolResult, error) {
	if e.ctx.ExtensionManager == nil {
		return errorResult("Extension manager not available")
	}

	ext, ok := e.ctx.ExtensionManager.GetExtension(NameToKey(name))
	if !ok {
		return errorResult(fmt.Sprintf("Extension not found: %s", name))
	}

	info := fmt.Sprintf("Name: %s\nType: %s\nDescription: %s\n",
		ext.Config.Name, ext.Config.Type, ext.Config.Description)

	if ext.ServerInfo != nil {
		info += fmt.Sprintf("Protocol Version: %s\n", ext.ServerInfo.ProtocolVersion)
		if ext.ServerInfo.Instructions != nil {
			info += fmt.Sprintf("\nInstructions:\n%s\n", *ext.ServerInfo.Instructions)
		}
	}

	return successResult(info)
}

func (e *ExtensionManagerExtension) listAvailable() (*CallToolResult, error) {
	// List platform extensions
	defs := DefaultPlatformExtensions()
	var result string
	for _, def := range defs {
		enabled := "disabled"
		if def.DefaultEnabled {
			enabled = "enabled"
		}
		result += fmt.Sprintf("- %s: %s (default: %s)\n", def.Name, def.Description, enabled)
	}

	return successResult(result)
}

// ListPrompts returns empty
func (e *ExtensionManagerExtension) ListPrompts(ctx context.Context, cursor *string) (*ListPromptsResult, error) {
	return &ListPromptsResult{Prompts: []Prompt{}}, nil
}

// GetPrompt returns error
func (e *ExtensionManagerExtension) GetPrompt(ctx context.Context, name string, arguments map[string]string) (*GetPromptResult, error) {
	return nil, fmt.Errorf("extension manager does not support prompts")
}

// SkillsExtension loads and executes skills from .goose/skills directory
type SkillsExtension struct {
	*BaseClient
	ctx    PlatformExtensionContext
	skills map[string]Skill
	mu     sync.RWMutex
}

// Skill represents a loaded skill
type Skill struct {
	Name         string            `json:"name"`
	Description  string            `json:"description"`
	Instructions string            `json:"instructions"`
	Parameters   map[string]string `json:"parameters,omitempty"`
}

// NewSkillsExtension creates a new skills extension
func NewSkillsExtension(ctx PlatformExtensionContext) (McpClient, error) {
	instructions := `# Skills Extension

This extension loads and manages skills from the .goose/skills directory.

## Available Tools

- **list_skills**: List all available skills
- **get_skill**: Get details about a specific skill
- **run_skill**: Execute a skill with parameters`

	ext := &SkillsExtension{
		ctx:    ctx,
		skills: make(map[string]Skill),
		BaseClient: NewBaseClient(&InitializeResult{
			ProtocolVersion: CurrentProtocolVersion,
			Capabilities: ServerCapabilities{
				Tools:     &ToolsCapability{},
				Resources: &ResourcesCapability{},
			},
			ServerInfo: Implementation{
				Name:    "skills",
				Version: "1.0.0",
			},
			Instructions: &instructions,
		}),
	}

	// Load skills from working directory
	ext.loadSkills()

	return ext, nil
}

// loadSkills loads skills from the .goose/skills directory
func (s *SkillsExtension) loadSkills() {
	// This is a placeholder - real implementation would scan directories
	// and load skill definitions from markdown files
}

// ListResources returns skill resources
func (s *SkillsExtension) ListResources(ctx context.Context, cursor *string) (*ListResourcesResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var resources []Resource
	for name, skill := range s.skills {
		desc := skill.Description
		resources = append(resources, Resource{
			URI:         fmt.Sprintf("skill://%s", name),
			Name:        skill.Name,
			Description: &desc,
		})
	}

	// Sort for consistent ordering
	sort.Slice(resources, func(i, j int) bool {
		return resources[i].Name < resources[j].Name
	})

	return &ListResourcesResult{Resources: resources}, nil
}

// ReadResource reads a skill's content
func (s *SkillsExtension) ReadResource(ctx context.Context, uri string) (*ReadResourceResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Parse skill:// URI
	if len(uri) < 9 || uri[:8] != "skill://" {
		return nil, fmt.Errorf("invalid skill URI: %s", uri)
	}

	name := uri[8:]
	skill, ok := s.skills[name]
	if !ok {
		return nil, fmt.Errorf("skill not found: %s", name)
	}

	content := skill.Instructions
	mimeType := "text/markdown"

	return &ReadResourceResult{
		Contents: []ResourceContent{
			{
				URI:      uri,
				MimeType: mimeType,
				Text:     &content,
			},
		},
	}, nil
}

// ListTools returns available skill tools
func (s *SkillsExtension) ListTools(ctx context.Context, cursor *string) (*ListToolsResult, error) {
	listSchema := json.RawMessage(`{
		"type": "object",
		"properties": {},
		"additionalProperties": false
	}`)

	getSchema := json.RawMessage(`{
		"type": "object",
		"properties": {
			"name": {"type": "string", "description": "Skill name"}
		},
		"required": ["name"],
		"additionalProperties": false
	}`)

	runSchema := json.RawMessage(`{
		"type": "object",
		"properties": {
			"name": {"type": "string", "description": "Skill name"},
			"parameters": {"type": "object", "description": "Skill parameters"}
		},
		"required": ["name"],
		"additionalProperties": false
	}`)

	tools := []Tool{
		{
			Name:        "list_skills",
			Description: "List all available skills",
			InputSchema: &listSchema,
		},
		{
			Name:        "get_skill",
			Description: "Get details about a specific skill",
			InputSchema: &getSchema,
		},
		{
			Name:        "run_skill",
			Description: "Execute a skill with parameters",
			InputSchema: &runSchema,
		},
	}

	return &ListToolsResult{Tools: tools}, nil
}

// CallTool executes a skills tool
func (s *SkillsExtension) CallTool(ctx context.Context, name string, arguments json.RawMessage) (*CallToolResult, error) {
	switch name {
	case "list_skills":
		return s.listSkills()
	case "get_skill":
		var args struct {
			Name string `json:"name"`
		}
		if err := json.Unmarshal(arguments, &args); err != nil {
			return errorResult(fmt.Sprintf("invalid arguments: %v", err))
		}
		return s.getSkill(args.Name)
	case "run_skill":
		var args struct {
			Name       string            `json:"name"`
			Parameters map[string]string `json:"parameters"`
		}
		if err := json.Unmarshal(arguments, &args); err != nil {
			return errorResult(fmt.Sprintf("invalid arguments: %v", err))
		}
		return s.runSkill(args.Name, args.Parameters)
	default:
		return errorResult(fmt.Sprintf("unknown tool: %s", name))
	}
}

func (s *SkillsExtension) listSkills() (*CallToolResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.skills) == 0 {
		return successResult("No skills found. Add skills to .goose/skills directory.")
	}

	var result string
	for name, skill := range s.skills {
		result += fmt.Sprintf("- %s: %s\n", name, skill.Description)
	}

	return successResult(result)
}

func (s *SkillsExtension) getSkill(name string) (*CallToolResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	skill, ok := s.skills[name]
	if !ok {
		return errorResult(fmt.Sprintf("Skill not found: %s", name))
	}

	info := fmt.Sprintf("Name: %s\nDescription: %s\n\nInstructions:\n%s",
		skill.Name, skill.Description, skill.Instructions)

	return successResult(info)
}

func (s *SkillsExtension) runSkill(name string, parameters map[string]string) (*CallToolResult, error) {
	s.mu.RLock()
	skill, ok := s.skills[name]
	s.mu.RUnlock()

	if !ok {
		return errorResult(fmt.Sprintf("Skill not found: %s", name))
	}

	// Return skill instructions for LLM to follow
	return successResult(fmt.Sprintf("Executing skill: %s\n\n%s", skill.Name, skill.Instructions))
}

// ListPrompts returns empty
func (s *SkillsExtension) ListPrompts(ctx context.Context, cursor *string) (*ListPromptsResult, error) {
	return &ListPromptsResult{Prompts: []Prompt{}}, nil
}

// GetPrompt returns error
func (s *SkillsExtension) GetPrompt(ctx context.Context, name string, arguments map[string]string) (*GetPromptResult, error) {
	return nil, fmt.Errorf("skills extension does not support prompts")
}

// Helper functions for creating tool results

func successResult(text string) (*CallToolResult, error) {
	return &CallToolResult{
		Content: []ToolContent{NewTextToolContent(text)},
		IsError: false,
	}, nil
}

func errorResult(text string) (*CallToolResult, error) {
	return &CallToolResult{
		Content: []ToolContent{NewTextToolContent(text)},
		IsError: true,
	}, nil
}
