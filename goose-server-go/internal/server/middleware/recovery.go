package middleware

import (
	"context"
	"fmt"
	"runtime/debug"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/rs/zerolog/log"
)

// Recovery creates a panic recovery middleware
func Recovery() app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		defer func() {
			if r := recover(); r != nil {
				stack := debug.Stack()
				log.Error().
					Interface("panic", r).
					Str("stack", string(stack)).
					Msg("Panic recovered")

				c.AbortWithStatusJSON(consts.StatusInternalServerError, map[string]string{
					"message": fmt.Sprintf("Internal server error: %v", r),
				})
			}
		}()

		c.Next(ctx)
	}
}
