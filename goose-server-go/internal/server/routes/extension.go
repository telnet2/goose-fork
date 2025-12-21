package routes

import (
	"context"
	"encoding/json"

	"github.com/block/goose-server-go/internal/extension"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
)

// ExtensionRoutes handles extension-related endpoints
type ExtensionRoutes struct {
	manager *extension.Manager
}

// NewExtensionRoutes creates a new ExtensionRoutes instance
func NewExtensionRoutes(manager *extension.Manager) *ExtensionRoutes {
	return &ExtensionRoutes{manager: manager}
}

// ExtensionListResponse is the response for listing extensions
type ExtensionListResponse struct {
	Extensions []ExtensionInfo `json:"extensions"`
}

// ExtensionInfo contains information about an extension
type ExtensionInfo struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Enabled     bool   `json:"enabled"`
}

// List handles GET /extensions
func (r *ExtensionRoutes) List(ctx context.Context, c *app.RequestContext) {
	extensions := r.manager.ListExtensions()

	var infos []ExtensionInfo
	for _, ext := range extensions {
		infos = append(infos, ExtensionInfo{
			Name:        ext.Config.Name,
			Type:        string(ext.Config.Type),
			Description: ext.Config.Description,
			Enabled:     true, // All loaded extensions are enabled
		})
	}

	c.JSON(consts.StatusOK, ExtensionListResponse{
		Extensions: infos,
	})
}

// AddExtensionRequest is the request body for adding an extension
type AddExtensionRequest struct {
	Type        string            `json:"type"`
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	URI         string            `json:"uri,omitempty"`
	Cmd         string            `json:"cmd,omitempty"`
	Args        []string          `json:"args,omitempty"`
	Envs        map[string]string `json:"envs,omitempty"`
	EnvKeys     []string          `json:"env_keys,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
	Timeout     *uint64           `json:"timeout,omitempty"`
}

// Add handles POST /extensions
func (r *ExtensionRoutes) Add(ctx context.Context, c *app.RequestContext) {
	var req AddExtensionRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(consts.StatusBadRequest, map[string]string{
			"message": "Invalid request body",
		})
		return
	}

	if req.Name == "" {
		c.JSON(consts.StatusBadRequest, map[string]string{
			"message": "Name is required",
		})
		return
	}

	// Build extension config based on type
	config := extension.ExtensionConfig{
		Type:        extension.ExtensionType(req.Type),
		Name:        req.Name,
		Description: req.Description,
		URI:         req.URI,
		Cmd:         req.Cmd,
		Args:        req.Args,
		Envs:        req.Envs,
		EnvKeys:     req.EnvKeys,
		Headers:     req.Headers,
		Timeout:     req.Timeout,
	}

	if err := r.manager.AddExtension(ctx, config); err != nil {
		c.JSON(consts.StatusInternalServerError, map[string]string{
			"message": err.Error(),
		})
		return
	}

	c.JSON(consts.StatusCreated, map[string]string{
		"message": "Extension added successfully",
		"name":    req.Name,
	})
}

// Remove handles DELETE /extensions/:name
func (r *ExtensionRoutes) Remove(ctx context.Context, c *app.RequestContext) {
	name := c.Param("name")
	key := extension.NameToKey(name)

	if err := r.manager.RemoveExtension(key); err != nil {
		c.JSON(consts.StatusNotFound, map[string]string{
			"message": err.Error(),
		})
		return
	}

	c.JSON(consts.StatusOK, map[string]string{
		"message": "Extension removed",
	})
}

// ListToolsResponse is the response for listing tools
type ListToolsResponse struct {
	Tools []ToolInfo `json:"tools"`
}

// ToolInfo contains information about a tool
type ToolInfo struct {
	Name        string           `json:"name"`
	Description string           `json:"description"`
	InputSchema *json.RawMessage `json:"inputSchema,omitempty"`
}

// ListTools handles GET /extensions/tools
func (r *ExtensionRoutes) ListTools(ctx context.Context, c *app.RequestContext) {
	// Get optional extension filter
	var extFilter *string
	if name := c.Query("extension"); name != "" {
		extFilter = &name
	}

	tools, err := r.manager.GetPrefixedTools(ctx, extFilter)
	if err != nil {
		c.JSON(consts.StatusInternalServerError, map[string]string{
			"message": err.Error(),
		})
		return
	}

	var infos []ToolInfo
	for _, tool := range tools {
		infos = append(infos, ToolInfo{
			Name:        tool.Name,
			Description: tool.Description,
			InputSchema: tool.InputSchema,
		})
	}

	c.JSON(consts.StatusOK, ListToolsResponse{
		Tools: infos,
	})
}

// CallToolRequest is the request body for calling a tool
type CallToolRequest struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

// CallToolResponse is the response for calling a tool
type CallToolResponse struct {
	Content []extension.ToolContent `json:"content"`
	IsError bool                    `json:"isError"`
}

// CallTool handles POST /extensions/tools/call
func (r *ExtensionRoutes) CallTool(ctx context.Context, c *app.RequestContext) {
	var req CallToolRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(consts.StatusBadRequest, map[string]string{
			"message": "Invalid request body",
		})
		return
	}

	if req.Name == "" {
		c.JSON(consts.StatusBadRequest, map[string]string{
			"message": "Tool name is required",
		})
		return
	}

	result, err := r.manager.CallTool(ctx, req.Name, req.Arguments)
	if err != nil {
		c.JSON(consts.StatusInternalServerError, map[string]string{
			"message": err.Error(),
		})
		return
	}

	c.JSON(consts.StatusOK, CallToolResponse{
		Content: result.Content,
		IsError: result.IsError,
	})
}

// ListResourcesResponse is the response for listing resources
type ListResourcesResponse struct {
	Resources []extension.Resource `json:"resources"`
}

// ListResources handles GET /extensions/resources
func (r *ExtensionRoutes) ListResources(ctx context.Context, c *app.RequestContext) {
	// Get optional scheme filter
	var schemeFilter *string
	if scheme := c.Query("scheme"); scheme != "" {
		schemeFilter = &scheme
	}

	resources, err := r.manager.GetResources(ctx, schemeFilter)
	if err != nil {
		c.JSON(consts.StatusInternalServerError, map[string]string{
			"message": err.Error(),
		})
		return
	}

	c.JSON(consts.StatusOK, ListResourcesResponse{
		Resources: resources,
	})
}

// ReadResourceRequest is the request body for reading a resource
type ReadResourceRequest struct {
	URI string `json:"uri"`
}

// ReadResource handles POST /extensions/resources/read
func (r *ExtensionRoutes) ReadResource(ctx context.Context, c *app.RequestContext) {
	var req ReadResourceRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(consts.StatusBadRequest, map[string]string{
			"message": "Invalid request body",
		})
		return
	}

	if req.URI == "" {
		c.JSON(consts.StatusBadRequest, map[string]string{
			"message": "URI is required",
		})
		return
	}

	result, err := r.manager.ReadResource(ctx, req.URI)
	if err != nil {
		c.JSON(consts.StatusNotFound, map[string]string{
			"message": err.Error(),
		})
		return
	}

	c.JSON(consts.StatusOK, result)
}

// GetExtensionInfo handles GET /extensions/:name
func (r *ExtensionRoutes) GetExtensionInfo(ctx context.Context, c *app.RequestContext) {
	name := c.Param("name")
	key := extension.NameToKey(name)

	ext, ok := r.manager.GetExtension(key)
	if !ok {
		c.JSON(consts.StatusNotFound, map[string]string{
			"message": "Extension not found",
		})
		return
	}

	response := map[string]interface{}{
		"name":        ext.Config.Name,
		"type":        ext.Config.Type,
		"description": ext.Config.Description,
		"enabled":     true,
	}

	if ext.ServerInfo != nil {
		response["serverInfo"] = ext.ServerInfo
	}

	c.JSON(consts.StatusOK, response)
}
