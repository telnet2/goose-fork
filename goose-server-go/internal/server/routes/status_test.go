package routes

import (
	"testing"

	"github.com/cloudwego/hertz/pkg/common/config"
	"github.com/cloudwego/hertz/pkg/common/ut"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/cloudwego/hertz/pkg/route"
)

func newTestRouter() *route.Engine {
	opt := config.NewOptions([]config.Option{})
	return route.NewEngine(opt)
}

func TestStatus(t *testing.T) {
	router := newTestRouter()
	router.GET("/status", Status)

	w := ut.PerformRequest(router, "GET", "/status", nil)

	resp := w.Result()
	if resp.StatusCode() != consts.StatusOK {
		t.Errorf("Status = %d, want %d", resp.StatusCode(), consts.StatusOK)
	}

	body := string(resp.Body())
	if body != "ok" {
		t.Errorf("Body = %q, want %q", body, "ok")
	}
}
