package routes

import (
	"context"

	"github.com/block/goose-server-go/internal/agent"
	"github.com/block/goose-server-go/internal/models"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
)

// StartAgentRequest represents the request body for starting an agent
type StartAgentRequest struct {
	WorkingDir     string          `json:"working_dir"`
	Recipe         *models.Recipe  `json:"recipe,omitempty"`
	RecipeID       *string         `json:"recipe_id,omitempty"`
	RecipeDeeplink *string         `json:"recipe_deeplink,omitempty"`
}

// ResumeAgentRequest represents the request body for resuming an agent
type ResumeAgentRequest struct {
	SessionID              string `json:"session_id"`
	LoadModelAndExtensions bool   `json:"load_model_and_extensions"`
}

// UpdateProviderRequest represents the request body for updating provider
type UpdateProviderRequest struct {
	SessionID string  `json:"session_id"`
	Provider  string  `json:"provider"`
	Model     *string `json:"model,omitempty"`
}

// UpdateFromSessionRequest represents the request body for updating from session
type UpdateFromSessionRequest struct {
	SessionID string `json:"session_id"`
}

// AgentRoutes handles agent-related endpoints
type AgentRoutes struct {
	agentManager *agent.Manager
}

// NewAgentRoutes creates a new AgentRoutes instance
func NewAgentRoutes(agentManager *agent.Manager) *AgentRoutes {
	return &AgentRoutes{agentManager: agentManager}
}

// Start handles POST /agent/start
func (r *AgentRoutes) Start(ctx context.Context, c *app.RequestContext) {
	var req StartAgentRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(consts.StatusBadRequest, map[string]string{
			"message": "Invalid request body: " + err.Error(),
		})
		return
	}

	if req.WorkingDir == "" {
		c.JSON(consts.StatusBadRequest, map[string]string{
			"message": "working_dir is required",
		})
		return
	}

	config := &agent.AgentConfig{
		WorkingDir:   req.WorkingDir,
		ProviderName: "mock", // Default to mock provider
		Recipe:       req.Recipe,
	}

	ag, err := r.agentManager.Start(ctx, config)
	if err != nil {
		c.JSON(consts.StatusInternalServerError, map[string]string{
			"message": "Failed to start agent: " + err.Error(),
		})
		return
	}

	// Return session info
	c.JSON(consts.StatusOK, map[string]interface{}{
		"id":          ag.SessionID,
		"working_dir": config.WorkingDir,
		"name":        "",
	})
}

// Resume handles POST /agent/resume
func (r *AgentRoutes) Resume(ctx context.Context, c *app.RequestContext) {
	var req ResumeAgentRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(consts.StatusBadRequest, map[string]string{
			"message": "Invalid request body: " + err.Error(),
		})
		return
	}

	if req.SessionID == "" {
		c.JSON(consts.StatusBadRequest, map[string]string{
			"message": "session_id is required",
		})
		return
	}

	ag, err := r.agentManager.Resume(ctx, req.SessionID, req.LoadModelAndExtensions)
	if err != nil {
		c.JSON(consts.StatusFailedDependency, map[string]string{
			"message": "Failed to resume agent: " + err.Error(),
		})
		return
	}

	c.JSON(consts.StatusOK, map[string]interface{}{
		"id":          ag.SessionID,
		"working_dir": ag.Config.WorkingDir,
	})
}

// AddExtension handles POST /agent/add_extension
func (r *AgentRoutes) AddExtension(ctx context.Context, c *app.RequestContext) {
	// TODO: Implement in Phase 5
	c.JSON(consts.StatusNotImplemented, map[string]string{
		"message": "Not implemented - Phase 5",
	})
}

// RemoveExtension handles POST /agent/remove_extension
func (r *AgentRoutes) RemoveExtension(ctx context.Context, c *app.RequestContext) {
	// TODO: Implement in Phase 5
	c.JSON(consts.StatusNotImplemented, map[string]string{
		"message": "Not implemented - Phase 5",
	})
}

// GetTools handles GET /agent/tools
func (r *AgentRoutes) GetTools(ctx context.Context, c *app.RequestContext) {
	sessionID := c.Query("session_id")
	if sessionID == "" {
		c.JSON(consts.StatusBadRequest, map[string]string{
			"message": "session_id is required",
		})
		return
	}

	ag, ok := r.agentManager.Get(sessionID)
	if !ok {
		c.JSON(consts.StatusFailedDependency, map[string]string{
			"message": "Agent not found or not active",
		})
		return
	}

	tools := ag.GetTools()
	c.JSON(consts.StatusOK, tools)
}

// CallTool handles POST /agent/call_tool
func (r *AgentRoutes) CallTool(ctx context.Context, c *app.RequestContext) {
	var req models.CallToolRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(consts.StatusBadRequest, map[string]string{
			"message": "Invalid request body: " + err.Error(),
		})
		return
	}

	ag, ok := r.agentManager.Get(req.SessionID)
	if !ok {
		c.JSON(consts.StatusFailedDependency, map[string]string{
			"message": "Agent not found or not active",
		})
		return
	}

	// Find the tool
	var found bool
	for _, t := range ag.GetTools() {
		if t.Name == req.Name {
			found = true
			break
		}
	}

	if !found {
		c.JSON(consts.StatusNotFound, map[string]string{
			"message": "Tool not found: " + req.Name,
		})
		return
	}

	// TODO: Implement actual tool execution in Phase 5
	// For now, return a mock response
	c.JSON(consts.StatusOK, models.CallToolResponse{
		Content: []models.Content{
			models.NewTextContent("Tool execution not yet implemented"),
		},
		IsError: false,
	})
}

// ReadResource handles POST /agent/read_resource
func (r *AgentRoutes) ReadResource(ctx context.Context, c *app.RequestContext) {
	// TODO: Implement in Phase 5
	c.JSON(consts.StatusNotImplemented, map[string]string{
		"message": "Not implemented - Phase 5",
	})
}

// UpdateProvider handles POST /agent/update_provider
func (r *AgentRoutes) UpdateProvider(ctx context.Context, c *app.RequestContext) {
	var req UpdateProviderRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(consts.StatusBadRequest, map[string]string{
			"message": "Invalid request body: " + err.Error(),
		})
		return
	}

	if req.SessionID == "" || req.Provider == "" {
		c.JSON(consts.StatusBadRequest, map[string]string{
			"message": "session_id and provider are required",
		})
		return
	}

	ag, ok := r.agentManager.Get(req.SessionID)
	if !ok {
		c.JSON(consts.StatusFailedDependency, map[string]string{
			"message": "Agent not found or not active",
		})
		return
	}

	// Get the provider
	provider, ok := r.agentManager.GetProvider(req.Provider)
	if !ok {
		// Fall back to mock provider with the requested name
		provider = agent.NewMockProvider()
	}

	// Update agent's provider
	ag.Provider = provider
	ag.Config.ProviderName = req.Provider
	if req.Model != nil {
		ag.Config.ModelName = *req.Model
	}

	c.JSON(consts.StatusOK, map[string]string{
		"message": "Provider updated",
	})
}

// UpdateFromSession handles POST /agent/update_from_session
func (r *AgentRoutes) UpdateFromSession(ctx context.Context, c *app.RequestContext) {
	var req UpdateFromSessionRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(consts.StatusBadRequest, map[string]string{
			"message": "Invalid request body: " + err.Error(),
		})
		return
	}

	if req.SessionID == "" {
		c.JSON(consts.StatusBadRequest, map[string]string{
			"message": "session_id is required",
		})
		return
	}

	ag, ok := r.agentManager.Get(req.SessionID)
	if !ok {
		c.JSON(consts.StatusFailedDependency, map[string]string{
			"message": "Agent not found or not active",
		})
		return
	}

	// The agent is already synced with session state in Resume
	// This endpoint is for explicit refresh
	_ = ag // Agent already has latest state

	c.JSON(consts.StatusOK, map[string]string{
		"message": "Agent updated from session",
	})
}

// UpdateRouterToolSelector handles POST /agent/update_router_tool_selector
func (r *AgentRoutes) UpdateRouterToolSelector(ctx context.Context, c *app.RequestContext) {
	// TODO: Implement in Phase 5
	c.JSON(consts.StatusNotImplemented, map[string]string{
		"message": "Not implemented - Phase 5",
	})
}
