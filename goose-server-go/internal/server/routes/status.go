package routes

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
)

// Status handles the GET /status endpoint
// This is a health check endpoint that does not require authentication
func Status(ctx context.Context, c *app.RequestContext) {
	c.String(consts.StatusOK, "ok")
}
