package routes

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
)

// ToolConfirmation handles POST /action-required/tool-confirmation
func ToolConfirmation(state interface{}) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		// TODO: Implement in Phase 5
		c.JSON(consts.StatusNotImplemented, map[string]string{
			"message": "Not implemented - Phase 5",
		})
	}
}

// Diagnostics handles GET /diagnostics/:session_id
func Diagnostics(state interface{}) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		// TODO: Implement in Phase 6
		c.JSON(consts.StatusNotImplemented, map[string]string{
			"message": "Not implemented - Phase 6",
		})
	}
}
