package routes

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
)

// Reply handles POST /reply - the main SSE streaming endpoint
func Reply(state interface{}) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		// TODO: Implement SSE streaming in Phase 3
		c.JSON(consts.StatusNotImplemented, map[string]string{
			"message": "Not implemented - Phase 3",
		})
	}
}
