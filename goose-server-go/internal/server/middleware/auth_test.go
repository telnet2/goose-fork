package middleware

import (
	"context"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/config"
	"github.com/cloudwego/hertz/pkg/common/ut"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/cloudwego/hertz/pkg/route"
)

func newTestRouter() *route.Engine {
	opt := config.NewOptions([]config.Option{})
	return route.NewEngine(opt)
}

func TestAuth_ValidKey(t *testing.T) {
	secretKey := "test-secret-key"

	router := newTestRouter()
	router.Use(Auth(secretKey))
	router.GET("/protected", func(ctx context.Context, c *app.RequestContext) {
		c.String(consts.StatusOK, "success")
	})

	w := ut.PerformRequest(router, "GET", "/protected", nil,
		ut.Header{Key: "X-Secret-Key", Value: secretKey})

	resp := w.Result()
	if resp.StatusCode() != consts.StatusOK {
		t.Errorf("Status = %d, want %d", resp.StatusCode(), consts.StatusOK)
	}
}

func TestAuth_InvalidKey(t *testing.T) {
	secretKey := "test-secret-key"

	router := newTestRouter()
	router.Use(Auth(secretKey))
	router.GET("/protected", func(ctx context.Context, c *app.RequestContext) {
		c.String(consts.StatusOK, "success")
	})

	w := ut.PerformRequest(router, "GET", "/protected", nil,
		ut.Header{Key: "X-Secret-Key", Value: "wrong-key"})

	resp := w.Result()
	if resp.StatusCode() != consts.StatusUnauthorized {
		t.Errorf("Status = %d, want %d", resp.StatusCode(), consts.StatusUnauthorized)
	}
}

func TestAuth_MissingKey(t *testing.T) {
	secretKey := "test-secret-key"

	router := newTestRouter()
	router.Use(Auth(secretKey))
	router.GET("/protected", func(ctx context.Context, c *app.RequestContext) {
		c.String(consts.StatusOK, "success")
	})

	w := ut.PerformRequest(router, "GET", "/protected", nil)

	resp := w.Result()
	if resp.StatusCode() != consts.StatusUnauthorized {
		t.Errorf("Status = %d, want %d", resp.StatusCode(), consts.StatusUnauthorized)
	}
}
