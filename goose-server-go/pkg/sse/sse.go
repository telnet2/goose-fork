package sse

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
)

// Writer provides SSE streaming capabilities
type Writer struct {
	ctx      context.Context
	c        *app.RequestContext
	done     chan struct{}
	closed   bool
	pingTick *time.Ticker
}

// NewWriter creates a new SSE writer for the request
func NewWriter(ctx context.Context, c *app.RequestContext) *Writer {
	// Set SSE headers
	c.SetContentType("text/event-stream")
	c.Response.Header.Set("Cache-Control", "no-cache")
	c.Response.Header.Set("Connection", "keep-alive")
	c.Response.Header.Set("X-Accel-Buffering", "no")

	return &Writer{
		ctx:  ctx,
		c:    c,
		done: make(chan struct{}),
	}
}

// WriteEvent writes an event to the SSE stream
func (w *Writer) WriteEvent(event interface{}) error {
	if w.closed {
		return fmt.Errorf("writer is closed")
	}

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// Write in SSE format: "data: <json>\n\n"
	_, err = w.c.Write([]byte(fmt.Sprintf("data: %s\n\n", data)))
	if err != nil {
		return fmt.Errorf("failed to write event: %w", err)
	}

	// Flush to ensure immediate delivery
	w.c.Flush()
	return nil
}

// WriteRaw writes raw bytes to the SSE stream
func (w *Writer) WriteRaw(data []byte) error {
	if w.closed {
		return fmt.Errorf("writer is closed")
	}

	_, err := w.c.Write([]byte(fmt.Sprintf("data: %s\n\n", data)))
	if err != nil {
		return fmt.Errorf("failed to write raw data: %w", err)
	}

	w.c.Flush()
	return nil
}

// StartHeartbeat starts sending ping events at the specified interval
func (w *Writer) StartHeartbeat(interval time.Duration) {
	w.pingTick = time.NewTicker(interval)

	go func() {
		for {
			select {
			case <-w.done:
				return
			case <-w.ctx.Done():
				return
			case <-w.pingTick.C:
				// Send ping event
				if err := w.WriteEvent(map[string]string{"type": "Ping"}); err != nil {
					return
				}
			}
		}
	}()
}

// Close stops the heartbeat and marks the writer as closed
func (w *Writer) Close() {
	if w.closed {
		return
	}
	w.closed = true

	if w.pingTick != nil {
		w.pingTick.Stop()
	}

	close(w.done)
}

// IsClosed returns whether the writer has been closed
func (w *Writer) IsClosed() bool {
	return w.closed
}

// Done returns a channel that's closed when the writer is done
func (w *Writer) Done() <-chan struct{} {
	return w.done
}
