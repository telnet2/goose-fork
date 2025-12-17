package routes

import (
	"context"

	"github.com/block/goose-server-go/internal/tunnel"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
)

// TunnelRoutes handles tunnel-related endpoints
type TunnelRoutes struct {
	manager *tunnel.Manager
}

// NewTunnelRoutes creates a new TunnelRoutes instance
func NewTunnelRoutes(manager *tunnel.Manager) *TunnelRoutes {
	return &TunnelRoutes{manager: manager}
}

// Start handles POST /tunnel/start
func (r *TunnelRoutes) Start(ctx context.Context, c *app.RequestContext) {
	if r.manager == nil {
		c.JSON(consts.StatusInternalServerError, map[string]string{
			"message": "Tunnel manager not available",
		})
		return
	}

	info, err := r.manager.Start()
	if err != nil {
		c.JSON(consts.StatusInternalServerError, map[string]string{
			"message": err.Error(),
		})
		return
	}

	c.JSON(consts.StatusOK, info)
}

// Stop handles POST /tunnel/stop
func (r *TunnelRoutes) Stop(ctx context.Context, c *app.RequestContext) {
	if r.manager == nil {
		c.JSON(consts.StatusInternalServerError, map[string]string{
			"message": "Tunnel manager not available",
		})
		return
	}

	if err := r.manager.Stop(true); err != nil {
		c.JSON(consts.StatusInternalServerError, map[string]string{
			"message": err.Error(),
		})
		return
	}

	c.Status(consts.StatusOK)
}

// Status handles GET /tunnel/status
func (r *TunnelRoutes) Status(ctx context.Context, c *app.RequestContext) {
	if r.manager == nil {
		c.JSON(consts.StatusInternalServerError, map[string]string{
			"message": "Tunnel manager not available",
		})
		return
	}

	info := r.manager.GetInfo()
	c.JSON(consts.StatusOK, info)
}
