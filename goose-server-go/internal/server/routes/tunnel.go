package routes

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
)

// TunnelRoutes handles tunnel-related endpoints
type TunnelRoutes struct {
	state interface{} // Will be *server.AppState
}

// NewTunnelRoutes creates a new TunnelRoutes instance
func NewTunnelRoutes(state interface{}) *TunnelRoutes {
	return &TunnelRoutes{state: state}
}

// Start handles POST /tunnel/start
func (r *TunnelRoutes) Start(ctx context.Context, c *app.RequestContext) {
	// TODO: Implement in Phase 6
	c.JSON(consts.StatusNotImplemented, map[string]string{
		"message": "Not implemented - Phase 6",
	})
}

// Stop handles POST /tunnel/stop
func (r *TunnelRoutes) Stop(ctx context.Context, c *app.RequestContext) {
	// TODO: Implement in Phase 6
	c.JSON(consts.StatusNotImplemented, map[string]string{
		"message": "Not implemented - Phase 6",
	})
}

// Status handles GET /tunnel/status
func (r *TunnelRoutes) Status(ctx context.Context, c *app.RequestContext) {
	// TODO: Implement in Phase 6
	c.JSON(consts.StatusNotImplemented, map[string]string{
		"message": "Not implemented - Phase 6",
	})
}
