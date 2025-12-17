package routes

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
)

// AgentRoutes handles agent-related endpoints
type AgentRoutes struct {
	state interface{} // Will be *server.AppState
}

// NewAgentRoutes creates a new AgentRoutes instance
func NewAgentRoutes(state interface{}) *AgentRoutes {
	return &AgentRoutes{state: state}
}

// Start handles POST /agent/start
func (r *AgentRoutes) Start(ctx context.Context, c *app.RequestContext) {
	// TODO: Implement in Phase 4
	c.JSON(consts.StatusNotImplemented, map[string]string{
		"message": "Not implemented - Phase 4",
	})
}

// Resume handles POST /agent/resume
func (r *AgentRoutes) Resume(ctx context.Context, c *app.RequestContext) {
	// TODO: Implement in Phase 4
	c.JSON(consts.StatusNotImplemented, map[string]string{
		"message": "Not implemented - Phase 4",
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
	// TODO: Implement in Phase 5
	c.JSON(consts.StatusNotImplemented, map[string]string{
		"message": "Not implemented - Phase 5",
	})
}

// CallTool handles POST /agent/call_tool
func (r *AgentRoutes) CallTool(ctx context.Context, c *app.RequestContext) {
	// TODO: Implement in Phase 5
	c.JSON(consts.StatusNotImplemented, map[string]string{
		"message": "Not implemented - Phase 5",
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
	// TODO: Implement in Phase 4
	c.JSON(consts.StatusNotImplemented, map[string]string{
		"message": "Not implemented - Phase 4",
	})
}

// UpdateFromSession handles POST /agent/update_from_session
func (r *AgentRoutes) UpdateFromSession(ctx context.Context, c *app.RequestContext) {
	// TODO: Implement in Phase 4
	c.JSON(consts.StatusNotImplemented, map[string]string{
		"message": "Not implemented - Phase 4",
	})
}

// UpdateRouterToolSelector handles POST /agent/update_router_tool_selector
func (r *AgentRoutes) UpdateRouterToolSelector(ctx context.Context, c *app.RequestContext) {
	// TODO: Implement in Phase 5
	c.JSON(consts.StatusNotImplemented, map[string]string{
		"message": "Not implemented - Phase 5",
	})
}
