# Goose Server Go Replication Plan

## Document Information
- **Version**: 1.0.0
- **Date**: 2024-12-17
- **Status**: Draft - Awaiting Review

---

## Table of Contents
1. [Overview](#1-overview)
2. [REST API Endpoint Documentation](#2-rest-api-endpoint-documentation)
3. [SSE Event Format Documentation](#3-sse-event-format-documentation)
4. [Data Model Schemas](#4-data-model-schemas)
5. [Architecture Plan](#5-architecture-plan)
6. [Library Choices](#6-library-choices)
7. [Test Plan](#7-test-plan)
8. [Implementation Phases](#8-implementation-phases)

---

## 1. Overview

### 1.1 Source System
- **Name**: goose-server
- **Language**: Rust
- **Framework**: Axum
- **Version**: 1.16.0
- **License**: Apache-2.0

### 1.2 Target System
- **Language**: Go (1.21+)
- **Framework**: cloudwego/hertz (recommended)
- **Protocol**: HTTP/1.1, SSE

### 1.3 Authentication
All endpoints (except `/status` and `/mcp-ui-proxy`) require authentication via the `X-Secret-Key` header.

```
Header: X-Secret-Key: <secret_value>
```

The secret key is configured via the `GOOSE_SERVER__SECRET_KEY` environment variable.

---

## 2. REST API Endpoint Documentation

### 2.1 Status Endpoints

#### GET /status
Health check endpoint (no authentication required).

| Field | Value |
|-------|-------|
| Response | `text/plain`: "ok" |
| Status | 200 |

#### GET /diagnostics/{session_id}
Generate diagnostic information for a session.

| Field | Value |
|-------|-------|
| Path Parameter | `session_id`: string |
| Response | `application/zip`: binary |
| Status | 200, 500 |

---

### 2.2 Agent Endpoints

#### POST /agent/start
Start a new agent session.

**Request Body** (`StartAgentRequest`):
```json
{
  "working_dir": "string (required)",
  "recipe": "Recipe (optional)",
  "recipe_id": "string (optional)",
  "recipe_deeplink": "string (optional)"
}
```

**Response** (`Session`): See Session schema in Section 4.

**Status Codes**: 200, 400, 401, 500

---

#### POST /agent/resume
Resume an existing agent session.

**Request Body** (`ResumeAgentRequest`):
```json
{
  "session_id": "string (required)",
  "load_model_and_extensions": "boolean (required)"
}
```

**Response**: `Session`

**Status Codes**: 200, 400, 401, 500

---

#### POST /agent/add_extension
Add an extension to an active session.

**Request Body** (`AddExtensionRequest`):
```json
{
  "session_id": "string (required)",
  "config": "ExtensionConfig (required)"
}
```

**Response**: `text/plain`

**Status Codes**: 200, 401, 424, 500

---

#### POST /agent/remove_extension
Remove an extension from a session.

**Request Body** (`RemoveExtensionRequest`):
```json
{
  "name": "string (required)",
  "session_id": "string (required)"
}
```

**Response**: `text/plain`

**Status Codes**: 200, 401, 424, 500

---

#### GET /agent/tools
Get available tools for a session.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| session_id | query | Yes | Session ID |
| extension_name | query | No | Filter by extension |

**Response**: `ToolInfo[]`

**Status Codes**: 200, 401, 424, 500

---

#### POST /agent/call_tool
Execute a tool.

**Request Body** (`CallToolRequest`):
```json
{
  "session_id": "string (required)",
  "name": "string (required)",
  "arguments": "object (required)"
}
```

**Response** (`CallToolResponse`):
```json
{
  "content": "Content[] (required)",
  "is_error": "boolean (required)",
  "structured_content": "any (optional)"
}
```

**Status Codes**: 200, 401, 404, 424, 500

---

#### POST /agent/read_resource
Read a resource from an extension.

**Request Body** (`ReadResourceRequest`):
```json
{
  "session_id": "string (required)",
  "extension_name": "string (required)",
  "uri": "string (required)"
}
```

**Response** (`ReadResourceResponse`):
```json
{
  "html": "string (required)"
}
```

**Status Codes**: 200, 401, 404, 424, 500

---

#### POST /agent/update_provider
Update the LLM provider for a session.

**Request Body** (`UpdateProviderRequest`):
```json
{
  "provider": "string (required)",
  "session_id": "string (required)",
  "model": "string (optional)"
}
```

**Status Codes**: 200, 400, 401, 424, 500

---

#### POST /agent/update_from_session
Update agent from session data.

**Request Body** (`UpdateFromSessionRequest`):
```json
{
  "session_id": "string (required)"
}
```

**Status Codes**: 200, 401, 424

---

#### POST /agent/update_router_tool_selector
Update tool selection strategy.

**Request Body** (`UpdateRouterToolSelectorRequest`):
```json
{
  "session_id": "string (required)"
}
```

**Status Codes**: 200, 401, 424, 500

---

### 2.3 Reply Endpoint (SSE Streaming)

#### POST /reply
Main chat endpoint with SSE streaming response.

**Request Body** (`ChatRequest`):
```json
{
  "messages": "Message[] (required)",
  "session_id": "string (required)",
  "recipe_name": "string (optional)",
  "recipe_version": "string (optional)"
}
```

**Response**: `text/event-stream` (see Section 3 for SSE format)

**Status Codes**: 200, 424, 500

**Request Body Size Limit**: 50 MB

---

### 2.4 Session Management Endpoints

#### GET /sessions
List all sessions.

**Response** (`SessionListResponse`):
```json
{
  "sessions": "Session[]"
}
```

---

#### GET /sessions/{session_id}
Get a specific session.

**Response**: `Session`

**Status Codes**: 200, 401, 404, 500

---

#### DELETE /sessions/{session_id}
Delete a session.

**Status Codes**: 200, 401, 404, 500

---

#### GET /sessions/{session_id}/export
Export session as text.

**Response**: `text/plain`

**Status Codes**: 200, 401, 404, 500

---

#### PUT /sessions/{session_id}/name
Update session name.

**Request Body** (`UpdateSessionNameRequest`):
```json
{
  "name": "string (max 200 chars)"
}
```

**Status Codes**: 200, 400, 401, 404, 500

---

#### POST /sessions/{session_id}/edit_message
Edit a message in a session.

**Request Body** (`EditMessageRequest`):
```json
{
  "timestamp": "int64 (required)",
  "editType": "fork | edit (optional)"
}
```

**Response** (`EditMessageResponse`):
```json
{
  "sessionId": "string"
}
```

---

#### POST /sessions/import
Import a session from JSON.

**Request Body** (`ImportSessionRequest`):
```json
{
  "json": "string (required)"
}
```

**Response**: `Session`

---

#### GET /sessions/insights
Get session insights/statistics.

**Response** (`SessionInsights`):
```json
{
  "totalSessions": "integer",
  "totalTokens": "int64"
}
```

---

#### PUT /sessions/{session_id}/user_recipe_values
Update user recipe values.

**Request Body** (`UpdateSessionUserRecipeValuesRequest`):
```json
{
  "userRecipeValues": "map<string, string>"
}
```

**Response** (`UpdateSessionUserRecipeValuesResponse`):
```json
{
  "recipe": "Recipe"
}
```

---

### 2.5 Configuration Endpoints

#### GET /config
Read all configuration.

**Response** (`ConfigResponse`):
```json
{
  "config": "map<string, any>"
}
```

---

#### POST /config/read
Read a specific configuration value.

**Request Body** (`ConfigKeyQuery`):
```json
{
  "key": "string (required)",
  "is_secret": "boolean (required)"
}
```

---

#### POST /config/upsert
Create or update a configuration value.

**Request Body** (`UpsertConfigQuery`):
```json
{
  "key": "string (required)",
  "value": "any (required)",
  "is_secret": "boolean (required)"
}
```

---

#### POST /config/remove
Remove a configuration value.

**Request Body**: `ConfigKeyQuery`

---

#### POST /config/init
Initialize configuration.

---

#### GET /config/validate
Validate configuration.

---

#### POST /config/backup
Backup configuration.

---

#### POST /config/recover
Recover configuration from backup.

---

#### GET /config/providers
List all available providers.

**Response**: `ProviderDetails[]`

---

#### GET /config/providers/{name}/models
Get models for a provider.

**Response**: `string[]`

---

#### POST /config/set_provider
Set the active provider.

**Request Body** (`SetProviderRequest`):
```json
{
  "provider": "string (required)",
  "model": "string (required)"
}
```

---

#### POST /config/check_provider
Check if a provider is configured.

**Request Body** (`CheckProviderRequest`):
```json
{
  "provider": "string (required)"
}
```

---

#### POST /config/detect-provider
Auto-detect provider from API key.

**Request Body** (`DetectProviderRequest`):
```json
{
  "api_key": "string (required)"
}
```

**Response** (`DetectProviderResponse`):
```json
{
  "provider_name": "string",
  "models": "string[]"
}
```

---

#### GET /config/extensions
Get all extensions.

**Response** (`ExtensionResponse`):
```json
{
  "extensions": "ExtensionEntry[]"
}
```

---

#### POST /config/extensions
Add an extension.

**Request Body** (`ExtensionQuery`):
```json
{
  "name": "string (required)",
  "config": "ExtensionConfig (required)",
  "enabled": "boolean (required)"
}
```

---

#### DELETE /config/extensions/{name}
Remove an extension.

---

#### POST /config/permissions
Update tool permissions.

**Request Body** (`UpsertPermissionsQuery`):
```json
{
  "tool_permissions": "ToolPermission[]"
}
```

---

#### GET /config/slash_commands
Get available slash commands.

**Response** (`SlashCommandsResponse`):
```json
{
  "commands": "SlashCommand[]"
}
```

---

#### Custom Providers: POST/GET/PUT/DELETE /config/custom-providers/{id}

---

### 2.6 Recipe Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| /recipes/list | GET | List all recipes |
| /recipes/create | POST | Create recipe from session |
| /recipes/parse | POST | Parse recipe from content |
| /recipes/encode | POST | Encode recipe to deeplink |
| /recipes/decode | POST | Decode recipe from deeplink |
| /recipes/save | POST | Save recipe to file |
| /recipes/delete | POST | Delete a recipe |
| /recipes/scan | POST | Scan recipe for security issues |
| /recipes/schedule | POST | Schedule a recipe |
| /recipes/slash-command | POST | Set slash command for recipe |

---

### 2.7 Schedule Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| /schedule/list | GET | List scheduled jobs |
| /schedule/create | POST | Create scheduled job |
| /schedule/{id} | PUT | Update schedule |
| /schedule/delete/{id} | DELETE | Delete schedule |
| /schedule/{id}/pause | POST | Pause schedule |
| /schedule/{id}/unpause | POST | Unpause schedule |
| /schedule/{id}/run_now | POST | Run job immediately |
| /schedule/{id}/kill | POST | Kill running job |
| /schedule/{id}/inspect | GET | Inspect running job |
| /schedule/{id}/sessions | GET | Get sessions for schedule |

---

### 2.8 Action Required Endpoints

#### POST /action-required/tool-confirmation
Confirm or deny a tool action.

**Request Body** (`ConfirmToolActionRequest`):
```json
{
  "id": "string (required)",
  "action": "string (required)",
  "sessionId": "string (required)",
  "principalType": "Extension | Tool (optional)"
}
```

---

### 2.9 Tunnel Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| /tunnel/start | POST | Start tunnel |
| /tunnel/stop | POST | Stop tunnel |
| /tunnel/status | GET | Get tunnel status |

---

### 2.10 Setup Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| /handle_openrouter | POST | Start OpenRouter setup |
| /handle_tetrate | POST | Start Tetrate setup |

---

### 2.11 MCP UI Proxy

#### GET /mcp-ui-proxy
Proxy for MCP UI (authentication via query parameter).

| Parameter | Type | Required |
|-----------|------|----------|
| secret | query | Yes |

---

## 3. SSE Event Format Documentation

### 3.1 Overview

The `/reply` endpoint returns Server-Sent Events (SSE) with the following characteristics:

- **Content-Type**: `text/event-stream`
- **Cache-Control**: `no-cache`
- **Connection**: `keep-alive`
- **Heartbeat**: `Ping` events every 500ms

### 3.2 Event Format

Each event is sent as:
```
data: <JSON_PAYLOAD>\n\n
```

### 3.3 MessageEvent Types

The `MessageEvent` is a discriminated union with `type` as the discriminator:

#### 3.3.1 Message Event
Streaming message content from the agent.

```json
{
  "type": "Message",
  "message": {
    "role": "user | assistant",
    "created": 1702831234,
    "content": [...],
    "metadata": {
      "userVisible": true,
      "agentVisible": true
    },
    "id": "optional-string"
  },
  "token_state": {
    "inputTokens": 100,
    "outputTokens": 50,
    "totalTokens": 150,
    "accumulatedInputTokens": 1000,
    "accumulatedOutputTokens": 500,
    "accumulatedTotalTokens": 1500
  }
}
```

#### 3.3.2 Error Event
Indicates an error occurred.

```json
{
  "type": "Error",
  "error": "Error message string"
}
```

#### 3.3.3 Finish Event
Indicates the stream is complete.

```json
{
  "type": "Finish",
  "reason": "stop",
  "token_state": {
    "inputTokens": 100,
    "outputTokens": 50,
    "totalTokens": 150,
    "accumulatedInputTokens": 1000,
    "accumulatedOutputTokens": 500,
    "accumulatedTotalTokens": 1500
  }
}
```

#### 3.3.4 ModelChange Event
Indicates the model has changed.

```json
{
  "type": "ModelChange",
  "model": "claude-3-opus-20240229",
  "mode": "default"
}
```

#### 3.3.5 Notification Event
MCP server notification.

```json
{
  "type": "Notification",
  "request_id": "unique-id",
  "message": {
    // ServerNotification object from MCP
  }
}
```

#### 3.3.6 UpdateConversation Event
Full conversation replacement (e.g., after context compaction).

```json
{
  "type": "UpdateConversation",
  "conversation": [
    // Array of Message objects
  ]
}
```

#### 3.3.7 Ping Event
Heartbeat to keep connection alive.

```json
{
  "type": "Ping"
}
```

### 3.4 MessageContent Types

The `content` array in Message can contain:

| Type | Discriminator | Description |
|------|--------------|-------------|
| `text` | TextContent | Plain text |
| `image` | ImageContent | Base64 image data |
| `toolRequest` | ToolRequest | Tool invocation request |
| `toolResponse` | ToolResponse | Tool execution result |
| `toolConfirmationRequest` | ToolConfirmationRequest | Request user confirmation |
| `actionRequired` | ActionRequired | Action required from user |
| `frontendToolRequest` | FrontendToolRequest | Frontend tool call |
| `thinking` | ThinkingContent | LLM thinking content |
| `redactedThinking` | RedactedThinkingContent | Redacted thinking |
| `systemNotification` | SystemNotificationContent | System notification |

### 3.5 SSE Implementation Notes

1. **Cancellation**: Support client disconnection via CancellationToken
2. **Timeout**: 500ms poll timeout for stream events
3. **Heartbeat**: Send Ping every 500ms to detect dead connections
4. **Error Handling**: Send Error event and close stream on failures
5. **Serialization**: Use JSON for event payloads
6. **Line Format**: Each event is `data: <json>\n\n`

---

## 4. Data Model Schemas

### 4.1 Core Types

#### Message
```go
type Message struct {
    Role     Role            `json:"role"`
    Created  int64           `json:"created"`
    Content  []MessageContent `json:"content"`
    Metadata MessageMetadata  `json:"metadata"`
    ID       *string          `json:"id,omitempty"`
}
```

#### Role
```go
type Role string

const (
    RoleUser      Role = "user"
    RoleAssistant Role = "assistant"
)
```

#### MessageMetadata
```go
type MessageMetadata struct {
    UserVisible  bool `json:"userVisible"`
    AgentVisible bool `json:"agentVisible"`
}
```

#### TokenState
```go
type TokenState struct {
    InputTokens             int32 `json:"inputTokens"`
    OutputTokens            int32 `json:"outputTokens"`
    TotalTokens             int32 `json:"totalTokens"`
    AccumulatedInputTokens  int32 `json:"accumulatedInputTokens"`
    AccumulatedOutputTokens int32 `json:"accumulatedOutputTokens"`
    AccumulatedTotalTokens  int32 `json:"accumulatedTotalTokens"`
}
```

### 4.2 Session
```go
type Session struct {
    ID                       string                  `json:"id"`
    WorkingDir               string                  `json:"working_dir"`
    Name                     string                  `json:"name"`
    CreatedAt                time.Time               `json:"created_at"`
    UpdatedAt                time.Time               `json:"updated_at"`
    ExtensionData            map[string]interface{}  `json:"extension_data"`
    MessageCount             uint64                  `json:"message_count"`
    Conversation             []Message               `json:"conversation,omitempty"`
    InputTokens              *int32                  `json:"input_tokens,omitempty"`
    OutputTokens             *int32                  `json:"output_tokens,omitempty"`
    TotalTokens              *int32                  `json:"total_tokens,omitempty"`
    AccumulatedInputTokens   *int32                  `json:"accumulated_input_tokens,omitempty"`
    AccumulatedOutputTokens  *int32                  `json:"accumulated_output_tokens,omitempty"`
    AccumulatedTotalTokens   *int32                  `json:"accumulated_total_tokens,omitempty"`
    ProviderName             *string                 `json:"provider_name,omitempty"`
    ModelConfig              *ModelConfig            `json:"model_config,omitempty"`
    Recipe                   *Recipe                 `json:"recipe,omitempty"`
    ScheduleID               *string                 `json:"schedule_id,omitempty"`
    SessionType              *SessionType            `json:"session_type,omitempty"`
    UserRecipeValues         map[string]string       `json:"user_recipe_values,omitempty"`
    UserSetName              bool                    `json:"user_set_name,omitempty"`
}
```

### 4.3 Extension Types

#### ExtensionConfig (Discriminated Union)
```go
type ExtensionConfig struct {
    Type           string            `json:"type"` // sse, stdio, builtin, platform, streamable_http, frontend, inline_python
    Name           string            `json:"name"`
    Description    string            `json:"description"`
    AvailableTools []string          `json:"available_tools,omitempty"`
    Bundled        *bool             `json:"bundled,omitempty"`
    Timeout        *int64            `json:"timeout,omitempty"`

    // For sse, streamable_http
    URI            string            `json:"uri,omitempty"`
    EnvKeys        []string          `json:"env_keys,omitempty"`
    Envs           map[string]string `json:"envs,omitempty"`
    Headers        map[string]string `json:"headers,omitempty"`

    // For stdio
    Cmd            string            `json:"cmd,omitempty"`
    Args           []string          `json:"args,omitempty"`

    // For builtin
    DisplayName    *string           `json:"display_name,omitempty"`

    // For frontend
    Tools          []Tool            `json:"tools,omitempty"`
    Instructions   *string           `json:"instructions,omitempty"`

    // For inline_python
    Code           string            `json:"code,omitempty"`
    Dependencies   []string          `json:"dependencies,omitempty"`
}
```

### 4.4 Provider Types

#### ProviderDetails
```go
type ProviderDetails struct {
    Name         string           `json:"name"`
    Metadata     ProviderMetadata `json:"metadata"`
    IsConfigured bool             `json:"is_configured"`
    ProviderType ProviderType     `json:"provider_type"`
}
```

#### ProviderMetadata
```go
type ProviderMetadata struct {
    Name         string      `json:"name"`
    DisplayName  string      `json:"display_name"`
    Description  string      `json:"description"`
    DefaultModel string      `json:"default_model"`
    KnownModels  []ModelInfo `json:"known_models"`
    ModelDocLink string      `json:"model_doc_link"`
    ConfigKeys   []ConfigKey `json:"config_keys"`
}
```

### 4.5 Additional Type Definitions

See full OpenAPI spec for complete type definitions. Key discriminated unions:
- `MessageContent` (discriminator: `type`)
- `ExtensionConfig` (discriminator: `type`)
- `ActionRequiredData` (discriminator: `actionType`)
- `Content` (discriminator: based on structure)
- `SuccessCheck` (discriminator: `type`)

---

## 5. Architecture Plan

### 5.1 Project Structure

```
goose-server-go/
├── cmd/
│   └── goosed/
│       └── main.go              # Entry point
├── internal/
│   ├── config/
│   │   ├── config.go            # Configuration management
│   │   └── keychain.go          # Secret storage
│   ├── server/
│   │   ├── server.go            # HTTP server setup
│   │   ├── middleware/
│   │   │   └── auth.go          # X-Secret-Key auth
│   │   └── routes/
│   │       ├── agent.go         # Agent endpoints
│   │       ├── reply.go         # SSE streaming
│   │       ├── session.go       # Session management
│   │       ├── config_mgmt.go   # Configuration endpoints
│   │       ├── recipe.go        # Recipe endpoints
│   │       ├── schedule.go      # Scheduling endpoints
│   │       ├── action.go        # Action required endpoints
│   │       ├── tunnel.go        # Tunnel endpoints
│   │       └── status.go        # Health check
│   ├── agent/
│   │   ├── agent.go             # Agent implementation
│   │   ├── extension.go         # Extension management
│   │   └── provider.go          # Provider abstraction
│   ├── session/
│   │   ├── manager.go           # Session lifecycle
│   │   └── storage.go           # Persistence (SQLite)
│   ├── mcp/
│   │   ├── client.go            # MCP client
│   │   └── server.go            # Built-in MCP servers
│   └── models/
│       ├── message.go           # Message types
│       ├── session.go           # Session types
│       ├── extension.go         # Extension types
│       ├── provider.go          # Provider types
│       ├── recipe.go            # Recipe types
│       └── events.go            # SSE event types
├── pkg/
│   └── sse/
│       └── sse.go               # SSE streaming utilities
├── go.mod
├── go.sum
└── Makefile
```

### 5.2 Component Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                      HTTP Layer (Hertz)                      │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────────────────┐│
│  │   Routes    │ │ Middleware  │ │    SSE Handler          ││
│  └─────────────┘ └─────────────┘ └─────────────────────────┘│
└───────────────────────────┬─────────────────────────────────┘
                            │
┌───────────────────────────▼─────────────────────────────────┐
│                    Service Layer                             │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────────────────┐│
│  │   Agent     │ │   Session   │ │     Extension           ││
│  │   Service   │ │   Manager   │ │     Manager             ││
│  └─────────────┘ └─────────────┘ └─────────────────────────┘│
└───────────────────────────┬─────────────────────────────────┘
                            │
┌───────────────────────────▼─────────────────────────────────┐
│                     Data Layer                               │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────────────────┐│
│  │   SQLite    │ │   Config    │ │     Keychain            ││
│  │   Storage   │ │   Files     │ │     (OS-specific)       ││
│  └─────────────┘ └─────────────┘ └─────────────────────────┘│
└─────────────────────────────────────────────────────────────┘
```

### 5.3 State Management

1. **AppState**: Global application state (thread-safe)
   - Active agents per session
   - Configuration cache
   - Secret key

2. **Session State**: Per-session state
   - Conversation history
   - Extension instances
   - Token counts

3. **Persistence**:
   - Sessions: SQLite database
   - Config: YAML files
   - Secrets: OS keychain

---

## 6. Library Choices

### 6.1 HTTP Framework

**Primary: cloudwego/hertz**
- High performance HTTP framework
- Native SSE support needed (custom implementation)
- Middleware support

```go
import "github.com/cloudwego/hertz/pkg/app/server"
```

### 6.2 Dependencies

| Category | Library | Purpose |
|----------|---------|---------|
| HTTP | `github.com/cloudwego/hertz` | HTTP server |
| JSON | `encoding/json` (stdlib) | JSON encoding |
| SQLite | `github.com/mattn/go-sqlite3` | Session storage |
| YAML | `gopkg.in/yaml.v3` | Config files |
| UUID | `github.com/google/uuid` | ID generation |
| Keychain | `github.com/zalando/go-keyring` | Secret storage |
| Cron | `github.com/robfig/cron/v3` | Job scheduling |
| MCP | Custom implementation | MCP protocol |
| Logging | `github.com/rs/zerolog` | Structured logging |
| Tracing | `go.opentelemetry.io/otel` | Observability |

### 6.3 SSE Implementation

Custom SSE writer for Hertz:

```go
type SSEWriter struct {
    w       io.Writer
    flusher http.Flusher
}

func (s *SSEWriter) WriteEvent(event interface{}) error {
    data, err := json.Marshal(event)
    if err != nil {
        return err
    }
    _, err = fmt.Fprintf(s.w, "data: %s\n\n", data)
    if err != nil {
        return err
    }
    s.flusher.Flush()
    return nil
}
```

### 6.4 MCP Client Implementation

Options:
1. **Port existing Rust rmcp to Go** (recommended for compatibility)
2. **Use official MCP SDK for Go** (if available)
3. **Implement subset needed for goose**

Key MCP operations:
- Initialize client
- List tools
- Call tool
- Read resource
- Handle notifications

---

## 7. Test Plan

### 7.1 Testing Strategy

#### 7.1.1 Unit Tests
Test individual components in isolation:
- Message serialization/deserialization
- Route handlers with mocked dependencies
- Session manager operations
- Configuration parsing

#### 7.1.2 Integration Tests
Test component interactions:
- Full request/response cycles
- SSE streaming behavior
- Database operations
- Extension lifecycle

#### 7.1.3 E2E Tests (Using Goose Client)
Test against the real Goose desktop client:
- Start server, connect client
- Create sessions
- Send messages and verify SSE events
- Test all major workflows

### 7.2 Goose Client Integration Testing

#### 7.2.1 Prerequisites
1. Build Go server
2. Configure to match Goose client expectations
3. Set `GOOSE_SERVER__SECRET_KEY`

#### 7.2.2 Test Scenarios

**Scenario 1: Basic Session**
```bash
# 1. Start Go server
./goosed agent --port 3000

# 2. Start Goose client (configured to connect to Go server)
# 3. Create new session
# 4. Send message "Hello"
# 5. Verify SSE events received
# 6. Verify response displayed
```

**Scenario 2: Tool Execution**
```bash
# 1. Start session with developer extension
# 2. Ask to list files
# 3. Verify tool request SSE event
# 4. Verify tool response SSE event
# 5. Verify file list in response
```

**Scenario 3: Session Persistence**
```bash
# 1. Create session, send messages
# 2. Restart server
# 3. Resume session
# 4. Verify conversation restored
```

### 7.3 Compatibility Test Matrix

| Endpoint | Unit Test | Integration | E2E |
|----------|-----------|-------------|-----|
| /status | ✓ | ✓ | ✓ |
| /agent/start | ✓ | ✓ | ✓ |
| /agent/resume | ✓ | ✓ | ✓ |
| /reply (SSE) | ✓ | ✓ | ✓ |
| /sessions/* | ✓ | ✓ | ✓ |
| /config/* | ✓ | ✓ | ✓ |
| /recipes/* | ✓ | ✓ | - |
| /schedule/* | ✓ | ✓ | - |

### 7.4 SSE Compliance Tests

```go
func TestSSEFormat(t *testing.T) {
    // Test each event type format
    events := []MessageEvent{
        {Type: "Message", ...},
        {Type: "Error", ...},
        {Type: "Finish", ...},
        {Type: "Ping"},
    }

    for _, event := range events {
        data := encodeSSE(event)
        assert.HasPrefix(data, "data: ")
        assert.HasSuffix(data, "\n\n")
    }
}
```

### 7.5 Protocol Conformance

1. **Header Verification**
   - `Content-Type: text/event-stream`
   - `Cache-Control: no-cache`
   - `Connection: keep-alive`

2. **Authentication**
   - Valid X-Secret-Key accepted
   - Invalid/missing rejected with 401

3. **Event Timing**
   - Ping events at ~500ms intervals
   - Events delivered in order

---

## 8. Implementation Phases

### Phase 1: Foundation (Week 1-2) ✅ COMPLETED
- [x] Project setup with Hertz
- [x] Basic routing structure
- [x] Authentication middleware (X-Secret-Key)
- [x] Status endpoint (/status)
- [x] Configuration loading (env vars + YAML)
- [x] Unit tests for config, auth middleware, status endpoint

**Implementation Notes:**
- Using cloudwego/hertz v0.9.3
- Go 1.21+ required
- Test coverage: 10 tests passing
- All route stubs created for future phases

### Phase 2: Session Management (Week 3-4)
- [ ] Session storage (SQLite)
- [ ] Session CRUD endpoints
- [ ] Session state management

### Phase 3: SSE Streaming (Week 5-6)
- [ ] SSE writer implementation
- [ ] Reply endpoint with mock agent
- [ ] All event type serialization
- [ ] Heartbeat/ping mechanism

### Phase 4: Agent Integration (Week 7-8)
- [ ] Agent service abstraction
- [ ] Basic LLM provider support
- [ ] Tool execution framework

### Phase 5: Extensions (Week 9-10)
- [ ] MCP client implementation
- [ ] Extension configuration
- [ ] Built-in extensions

### Phase 6: Full API (Week 11-12)
- [ ] Recipe management
- [ ] Scheduling system
- [ ] Tunnel support
- [ ] All remaining endpoints

### Phase 7: Testing & Polish (Week 13-14)
- [ ] Comprehensive test suite
- [ ] E2E testing with Goose client
- [ ] Performance optimization
- [ ] Documentation

---

## Appendix A: Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `GOOSE_SERVER__SECRET_KEY` | Authentication secret | Required |
| `GOOSE_PORT` | Server port | 3000 |
| `GOOSE_PATH_ROOT` | Data directory root | OS-specific |
| `GOOSE_PROVIDER` | Default LLM provider | - |

---

## Appendix B: Error Response Format

All error responses follow:
```json
{
  "message": "Error description"
}
```

---

## Appendix C: OpenAPI Specification

The full OpenAPI 3.0.3 specification is available at:
`ui/desktop/openapi.json`

This specification is auto-generated from the Rust source code and should be considered the authoritative reference.

---

## Review Checklist

Before implementation, please confirm:

- [ ] Architecture aligns with team capabilities
- [ ] Library choices approved
- [ ] Test coverage requirements acceptable
- [ ] Timeline feasible
- [ ] Any endpoints that can be deprioritized
- [ ] MCP implementation approach decision
