package extension

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// DefaultTimeout is the default extension timeout in seconds
const DefaultTimeout = 300

// BaseClient provides common functionality for MCP clients
type BaseClient struct {
	info          *InitializeResult
	notifications chan ServerNotification
	closed        bool
	mu            sync.RWMutex
}

// NewBaseClient creates a new base client
func NewBaseClient(info *InitializeResult) *BaseClient {
	return &BaseClient{
		info:          info,
		notifications: make(chan ServerNotification, 100),
	}
}

// GetInfo returns the server initialization information
func (c *BaseClient) GetInfo() *InitializeResult {
	return c.info
}

// GetMoim returns nil (no MOIM by default)
func (c *BaseClient) GetMoim() *string {
	return nil
}

// Subscribe returns the notification channel
func (c *BaseClient) Subscribe() <-chan ServerNotification {
	return c.notifications
}

// Close closes the base client
func (c *BaseClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.closed {
		c.closed = true
		close(c.notifications)
	}
	return nil
}

// sendNotification sends a notification to subscribers
func (c *BaseClient) sendNotification(n ServerNotification) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.closed {
		select {
		case c.notifications <- n:
		default:
			// Drop if buffer full
		}
	}
}

// SSEClient implements McpClient for SSE-based extensions
type SSEClient struct {
	*BaseClient
	uri     string
	envs    map[string]string
	timeout time.Duration
	client  *http.Client
	cancel  context.CancelFunc
}

// NewSSEClient creates a new SSE client
func NewSSEClient(ctx context.Context, uri string, envs map[string]string, timeout *uint64) (*SSEClient, error) {
	timeoutSec := uint64(DefaultTimeout)
	if timeout != nil {
		timeoutSec = *timeout
	}

	ctx, cancel := context.WithCancel(ctx)

	client := &SSEClient{
		uri:     uri,
		envs:    envs,
		timeout: time.Duration(timeoutSec) * time.Second,
		client: &http.Client{
			Timeout: time.Duration(timeoutSec) * time.Second,
		},
		cancel: cancel,
	}

	// Initialize connection
	info, err := client.initialize(ctx)
	if err != nil {
		cancel()
		return nil, err
	}

	client.BaseClient = NewBaseClient(info)

	// Start SSE listener in background
	go client.listenSSE(ctx)

	return client, nil
}

// initialize performs the MCP initialization handshake
func (c *SSEClient) initialize(ctx context.Context) (*InitializeResult, error) {
	// For now, return a placeholder - real implementation would do JSON-RPC
	return &InitializeResult{
		ProtocolVersion: CurrentProtocolVersion,
		Capabilities: ServerCapabilities{
			Tools: &ToolsCapability{},
		},
		ServerInfo: Implementation{
			Name:    "sse-extension",
			Version: "1.0.0",
		},
	}, nil
}

// listenSSE listens for SSE events
func (c *SSEClient) listenSSE(ctx context.Context) {
	// Placeholder for SSE event handling
	<-ctx.Done()
}

// ListResources lists available resources
func (c *SSEClient) ListResources(ctx context.Context, cursor *string) (*ListResourcesResult, error) {
	return &ListResourcesResult{Resources: []Resource{}}, nil
}

// ReadResource reads a specific resource
func (c *SSEClient) ReadResource(ctx context.Context, uri string) (*ReadResourceResult, error) {
	return nil, fmt.Errorf("resource not found: %s", uri)
}

// ListTools lists available tools
func (c *SSEClient) ListTools(ctx context.Context, cursor *string) (*ListToolsResult, error) {
	return &ListToolsResult{Tools: []Tool{}}, nil
}

// CallTool executes a tool
func (c *SSEClient) CallTool(ctx context.Context, name string, arguments json.RawMessage) (*CallToolResult, error) {
	return nil, fmt.Errorf("tool not found: %s", name)
}

// ListPrompts lists available prompts
func (c *SSEClient) ListPrompts(ctx context.Context, cursor *string) (*ListPromptsResult, error) {
	return &ListPromptsResult{Prompts: []Prompt{}}, nil
}

// GetPrompt retrieves a specific prompt
func (c *SSEClient) GetPrompt(ctx context.Context, name string, arguments map[string]string) (*GetPromptResult, error) {
	return nil, fmt.Errorf("prompt not found: %s", name)
}

// Close closes the SSE client
func (c *SSEClient) Close() error {
	c.cancel()
	return c.BaseClient.Close()
}

// StdioClient implements McpClient for stdio-based extensions
type StdioClient struct {
	*BaseClient
	cmd     *exec.Cmd
	stdin   io.WriteCloser
	stdout  io.ReadCloser
	stderr  bytes.Buffer
	timeout time.Duration
	mu      sync.Mutex
	reqID   int
}

// NewStdioClient creates a new stdio client
func NewStdioClient(ctx context.Context, command string, args []string, envs map[string]string, timeout *uint64) (*StdioClient, error) {
	timeoutSec := uint64(DefaultTimeout)
	if timeout != nil {
		timeoutSec = *timeout
	}

	cmd := exec.CommandContext(ctx, command, args...)

	// Set environment
	cmd.Env = os.Environ()
	for k, v := range envs {
		cmd.Env = append(cmd.Env, k+"="+v)
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to get stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to get stdout pipe: %w", err)
	}

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start process: %w", err)
	}

	client := &StdioClient{
		cmd:     cmd,
		stdin:   stdin,
		stdout:  stdout,
		stderr:  stderr,
		timeout: time.Duration(timeoutSec) * time.Second,
	}

	// Initialize
	info, err := client.initialize(ctx)
	if err != nil {
		cmd.Process.Kill()
		return nil, err
	}

	client.BaseClient = NewBaseClient(info)

	// Start reader in background
	go client.readLoop()

	return client, nil
}

// initialize performs the MCP initialization handshake
func (c *StdioClient) initialize(ctx context.Context) (*InitializeResult, error) {
	// Send initialize request
	req := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]interface{}{
			"protocolVersion": CurrentProtocolVersion,
			"capabilities":    map[string]interface{}{},
			"clientInfo": map[string]interface{}{
				"name":    "goose-server-go",
				"version": "1.0.0",
			},
		},
	}

	resp, err := c.sendRequest(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("initialize failed: %w", err)
	}

	// Parse result
	result, ok := resp["result"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid initialize response")
	}

	// Convert to InitializeResult
	resultBytes, _ := json.Marshal(result)
	var info InitializeResult
	if err := json.Unmarshal(resultBytes, &info); err != nil {
		return nil, fmt.Errorf("failed to parse initialize result: %w", err)
	}

	// Send initialized notification
	notif := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "notifications/initialized",
	}
	c.sendNotification(notif)

	return &info, nil
}

// sendRequest sends a JSON-RPC request and waits for response
func (c *StdioClient) sendRequest(ctx context.Context, req map[string]interface{}) (map[string]interface{}, error) {
	c.mu.Lock()
	c.reqID++
	req["id"] = c.reqID
	c.mu.Unlock()

	data, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	// Write request
	if _, err := c.stdin.Write(append(data, '\n')); err != nil {
		return nil, err
	}

	// Read response (simplified - real impl would be async)
	scanner := bufio.NewScanner(c.stdout)
	if scanner.Scan() {
		var resp map[string]interface{}
		if err := json.Unmarshal(scanner.Bytes(), &resp); err != nil {
			return nil, err
		}
		return resp, nil
	}

	return nil, fmt.Errorf("no response received")
}

// sendNotification sends a JSON-RPC notification (no response expected)
func (c *StdioClient) sendNotification(notif map[string]interface{}) {
	data, _ := json.Marshal(notif)
	c.stdin.Write(append(data, '\n'))
}

// readLoop reads responses and notifications from stdout
func (c *StdioClient) readLoop() {
	scanner := bufio.NewScanner(c.stdout)
	for scanner.Scan() {
		var msg map[string]interface{}
		if err := json.Unmarshal(scanner.Bytes(), &msg); err != nil {
			continue
		}

		// Check if notification
		if method, ok := msg["method"].(string); ok {
			paramsBytes, _ := json.Marshal(msg["params"])
			c.BaseClient.sendNotification(ServerNotification{
				Method: method,
				Params: paramsBytes,
			})
		}
	}
}

// ListResources lists available resources
func (c *StdioClient) ListResources(ctx context.Context, cursor *string) (*ListResourcesResult, error) {
	params := map[string]interface{}{}
	if cursor != nil {
		params["cursor"] = *cursor
	}

	resp, err := c.sendRequest(ctx, map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "resources/list",
		"params":  params,
	})
	if err != nil {
		return nil, err
	}

	resultBytes, _ := json.Marshal(resp["result"])
	var result ListResourcesResult
	json.Unmarshal(resultBytes, &result)
	return &result, nil
}

// ReadResource reads a specific resource
func (c *StdioClient) ReadResource(ctx context.Context, uri string) (*ReadResourceResult, error) {
	resp, err := c.sendRequest(ctx, map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "resources/read",
		"params": map[string]interface{}{
			"uri": uri,
		},
	})
	if err != nil {
		return nil, err
	}

	resultBytes, _ := json.Marshal(resp["result"])
	var result ReadResourceResult
	json.Unmarshal(resultBytes, &result)
	return &result, nil
}

// ListTools lists available tools
func (c *StdioClient) ListTools(ctx context.Context, cursor *string) (*ListToolsResult, error) {
	params := map[string]interface{}{}
	if cursor != nil {
		params["cursor"] = *cursor
	}

	resp, err := c.sendRequest(ctx, map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "tools/list",
		"params":  params,
	})
	if err != nil {
		return nil, err
	}

	resultBytes, _ := json.Marshal(resp["result"])
	var result ListToolsResult
	json.Unmarshal(resultBytes, &result)
	return &result, nil
}

// CallTool executes a tool
func (c *StdioClient) CallTool(ctx context.Context, name string, arguments json.RawMessage) (*CallToolResult, error) {
	var args interface{}
	json.Unmarshal(arguments, &args)

	resp, err := c.sendRequest(ctx, map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "tools/call",
		"params": map[string]interface{}{
			"name":      name,
			"arguments": args,
		},
	})
	if err != nil {
		return nil, err
	}

	if errObj, ok := resp["error"].(map[string]interface{}); ok {
		errMsg := fmt.Sprintf("%v", errObj["message"])
		return &CallToolResult{
			Content: []ToolContent{NewTextToolContent(errMsg)},
			IsError: true,
		}, nil
	}

	resultBytes, _ := json.Marshal(resp["result"])
	var result CallToolResult
	json.Unmarshal(resultBytes, &result)
	return &result, nil
}

// ListPrompts lists available prompts
func (c *StdioClient) ListPrompts(ctx context.Context, cursor *string) (*ListPromptsResult, error) {
	params := map[string]interface{}{}
	if cursor != nil {
		params["cursor"] = *cursor
	}

	resp, err := c.sendRequest(ctx, map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "prompts/list",
		"params":  params,
	})
	if err != nil {
		return nil, err
	}

	resultBytes, _ := json.Marshal(resp["result"])
	var result ListPromptsResult
	json.Unmarshal(resultBytes, &result)
	return &result, nil
}

// GetPrompt retrieves a specific prompt
func (c *StdioClient) GetPrompt(ctx context.Context, name string, arguments map[string]string) (*GetPromptResult, error) {
	resp, err := c.sendRequest(ctx, map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "prompts/get",
		"params": map[string]interface{}{
			"name":      name,
			"arguments": arguments,
		},
	})
	if err != nil {
		return nil, err
	}

	resultBytes, _ := json.Marshal(resp["result"])
	var result GetPromptResult
	json.Unmarshal(resultBytes, &result)
	return &result, nil
}

// Close closes the stdio client
func (c *StdioClient) Close() error {
	c.stdin.Close()
	c.cmd.Process.Kill()
	c.cmd.Wait()
	return c.BaseClient.Close()
}

// BuiltinClient wraps a stdio client for builtin extensions
type BuiltinClient struct {
	*StdioClient
}

// NewBuiltinClient creates a client for a builtin extension
func NewBuiltinClient(ctx context.Context, name string, timeout *uint64) (*BuiltinClient, error) {
	// Get current executable
	executable, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("failed to get executable path: %w", err)
	}

	// Create stdio client with mcp command
	stdioClient, err := NewStdioClient(ctx, executable, []string{"mcp", name}, nil, timeout)
	if err != nil {
		return nil, err
	}

	return &BuiltinClient{StdioClient: stdioClient}, nil
}

// StreamableHTTPClient implements McpClient for HTTP-based extensions
type StreamableHTTPClient struct {
	*BaseClient
	uri     string
	headers map[string]string
	envs    map[string]string
	timeout time.Duration
	client  *http.Client
}

// NewStreamableHTTPClient creates a new streamable HTTP client
func NewStreamableHTTPClient(ctx context.Context, uri string, headers, envs map[string]string, timeout *uint64) (*StreamableHTTPClient, error) {
	timeoutSec := uint64(DefaultTimeout)
	if timeout != nil {
		timeoutSec = *timeout
	}

	client := &StreamableHTTPClient{
		uri:     uri,
		headers: headers,
		envs:    envs,
		timeout: time.Duration(timeoutSec) * time.Second,
		client: &http.Client{
			Timeout: time.Duration(timeoutSec) * time.Second,
		},
	}

	// Initialize
	info, err := client.initialize(ctx)
	if err != nil {
		return nil, err
	}

	client.BaseClient = NewBaseClient(info)

	return client, nil
}

// initialize performs initialization
func (c *StreamableHTTPClient) initialize(ctx context.Context) (*InitializeResult, error) {
	return &InitializeResult{
		ProtocolVersion: CurrentProtocolVersion,
		Capabilities: ServerCapabilities{
			Tools: &ToolsCapability{},
		},
		ServerInfo: Implementation{
			Name:    "http-extension",
			Version: "1.0.0",
		},
	}, nil
}

// sendRequest sends an HTTP request
func (c *StreamableHTTPClient) sendRequest(ctx context.Context, method string, params interface{}) (map[string]interface{}, error) {
	body := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  method,
		"params":  params,
	}

	data, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.uri, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	for k, v := range c.headers {
		req.Header.Set(k, v)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result, nil
}

// ListResources lists available resources
func (c *StreamableHTTPClient) ListResources(ctx context.Context, cursor *string) (*ListResourcesResult, error) {
	params := map[string]interface{}{}
	if cursor != nil {
		params["cursor"] = *cursor
	}

	resp, err := c.sendRequest(ctx, "resources/list", params)
	if err != nil {
		return nil, err
	}

	resultBytes, _ := json.Marshal(resp["result"])
	var result ListResourcesResult
	json.Unmarshal(resultBytes, &result)
	return &result, nil
}

// ReadResource reads a specific resource
func (c *StreamableHTTPClient) ReadResource(ctx context.Context, uri string) (*ReadResourceResult, error) {
	resp, err := c.sendRequest(ctx, "resources/read", map[string]interface{}{"uri": uri})
	if err != nil {
		return nil, err
	}

	resultBytes, _ := json.Marshal(resp["result"])
	var result ReadResourceResult
	json.Unmarshal(resultBytes, &result)
	return &result, nil
}

// ListTools lists available tools
func (c *StreamableHTTPClient) ListTools(ctx context.Context, cursor *string) (*ListToolsResult, error) {
	params := map[string]interface{}{}
	if cursor != nil {
		params["cursor"] = *cursor
	}

	resp, err := c.sendRequest(ctx, "tools/list", params)
	if err != nil {
		return nil, err
	}

	resultBytes, _ := json.Marshal(resp["result"])
	var result ListToolsResult
	json.Unmarshal(resultBytes, &result)
	return &result, nil
}

// CallTool executes a tool
func (c *StreamableHTTPClient) CallTool(ctx context.Context, name string, arguments json.RawMessage) (*CallToolResult, error) {
	var args interface{}
	json.Unmarshal(arguments, &args)

	resp, err := c.sendRequest(ctx, "tools/call", map[string]interface{}{
		"name":      name,
		"arguments": args,
	})
	if err != nil {
		return nil, err
	}

	resultBytes, _ := json.Marshal(resp["result"])
	var result CallToolResult
	json.Unmarshal(resultBytes, &result)
	return &result, nil
}

// ListPrompts lists available prompts
func (c *StreamableHTTPClient) ListPrompts(ctx context.Context, cursor *string) (*ListPromptsResult, error) {
	params := map[string]interface{}{}
	if cursor != nil {
		params["cursor"] = *cursor
	}

	resp, err := c.sendRequest(ctx, "prompts/list", params)
	if err != nil {
		return nil, err
	}

	resultBytes, _ := json.Marshal(resp["result"])
	var result ListPromptsResult
	json.Unmarshal(resultBytes, &result)
	return &result, nil
}

// GetPrompt retrieves a specific prompt
func (c *StreamableHTTPClient) GetPrompt(ctx context.Context, name string, arguments map[string]string) (*GetPromptResult, error) {
	resp, err := c.sendRequest(ctx, "prompts/get", map[string]interface{}{
		"name":      name,
		"arguments": arguments,
	})
	if err != nil {
		return nil, err
	}

	resultBytes, _ := json.Marshal(resp["result"])
	var result GetPromptResult
	json.Unmarshal(resultBytes, &result)
	return &result, nil
}

// FrontendClient implements McpClient for frontend-defined tools
type FrontendClient struct {
	*BaseClient
	name         string
	description  string
	tools        []json.RawMessage
	instructions *string
}

// NewFrontendClient creates a new frontend client
func NewFrontendClient(name, description string, tools []json.RawMessage, instructions *string) (*FrontendClient, error) {
	instr := ""
	if instructions != nil {
		instr = *instructions
	}

	client := &FrontendClient{
		name:         name,
		description:  description,
		tools:        tools,
		instructions: instructions,
		BaseClient: NewBaseClient(&InitializeResult{
			ProtocolVersion: CurrentProtocolVersion,
			Capabilities: ServerCapabilities{
				Tools: &ToolsCapability{},
			},
			ServerInfo: Implementation{
				Name:    name,
				Version: "1.0.0",
			},
			Instructions: &instr,
		}),
	}

	return client, nil
}

// ListResources returns empty for frontend extensions
func (c *FrontendClient) ListResources(ctx context.Context, cursor *string) (*ListResourcesResult, error) {
	return &ListResourcesResult{Resources: []Resource{}}, nil
}

// ReadResource returns error for frontend extensions
func (c *FrontendClient) ReadResource(ctx context.Context, uri string) (*ReadResourceResult, error) {
	return nil, fmt.Errorf("frontend extensions don't support resources")
}

// ListTools returns the frontend-defined tools
func (c *FrontendClient) ListTools(ctx context.Context, cursor *string) (*ListToolsResult, error) {
	var tools []Tool
	for _, toolData := range c.tools {
		var tool Tool
		if err := json.Unmarshal(toolData, &tool); err != nil {
			continue
		}
		tools = append(tools, tool)
	}
	return &ListToolsResult{Tools: tools}, nil
}

// CallTool returns action required for frontend tools
func (c *FrontendClient) CallTool(ctx context.Context, name string, arguments json.RawMessage) (*CallToolResult, error) {
	// Frontend tools need to be handled by the frontend
	return &CallToolResult{
		Content: []ToolContent{
			NewTextToolContent(fmt.Sprintf("Action required: tool %s needs frontend handling", name)),
		},
	}, nil
}

// ListPrompts returns empty for frontend extensions
func (c *FrontendClient) ListPrompts(ctx context.Context, cursor *string) (*ListPromptsResult, error) {
	return &ListPromptsResult{Prompts: []Prompt{}}, nil
}

// GetPrompt returns error for frontend extensions
func (c *FrontendClient) GetPrompt(ctx context.Context, name string, arguments map[string]string) (*GetPromptResult, error) {
	return nil, fmt.Errorf("frontend extensions don't support prompts")
}

// InlinePythonClient implements McpClient for inline Python code
type InlinePythonClient struct {
	*StdioClient
	tempDir string
}

// NewInlinePythonClient creates a new inline Python client
func NewInlinePythonClient(ctx context.Context, code string, dependencies []string, timeout *uint64) (*InlinePythonClient, error) {
	// Create temp directory for Python code
	tempDir, err := os.MkdirTemp("", "goose-python-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %w", err)
	}

	// Write code to file
	codePath := tempDir + "/extension.py"
	if err := os.WriteFile(codePath, []byte(code), 0644); err != nil {
		os.RemoveAll(tempDir)
		return nil, fmt.Errorf("failed to write code: %w", err)
	}

	// Build uvx command with dependencies
	args := []string{"--with", "mcp"}
	for _, dep := range dependencies {
		args = append(args, "--with", dep)
	}
	args = append(args, "mcp", "run", codePath)

	// Create stdio client with uvx
	stdioClient, err := NewStdioClient(ctx, "uvx", args, nil, timeout)
	if err != nil {
		os.RemoveAll(tempDir)
		return nil, err
	}

	return &InlinePythonClient{
		StdioClient: stdioClient,
		tempDir:     tempDir,
	}, nil
}

// Close closes the inline Python client and cleans up temp files
func (c *InlinePythonClient) Close() error {
	err := c.StdioClient.Close()
	os.RemoveAll(c.tempDir)
	return err
}

// isToolNameValid checks if a tool name matches conventions
func isToolNameValid(name string) bool {
	// Tool names should be alphanumeric with underscores
	for _, r := range name {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_') {
			return false
		}
	}
	return len(name) > 0 && !strings.HasPrefix(name, "_") && !strings.HasSuffix(name, "_")
}
