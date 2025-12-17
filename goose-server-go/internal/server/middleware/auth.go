package middleware

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
)

// Auth creates an authentication middleware that checks the X-Secret-Key header
func Auth(secretKey string) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		// Get the secret key from header
		key := string(c.GetHeader("X-Secret-Key"))

		// Validate
		if key != secretKey {
			c.AbortWithStatusJSON(consts.StatusUnauthorized, map[string]string{
				"message": "Unauthorized - Invalid or missing API key",
			})
			return
		}

		c.Next(ctx)
	}
}
