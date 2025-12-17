package sse

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol"
)

// mockRequestContext creates a mock request context for testing
func mockRequestContext() *app.RequestContext {
	req := &protocol.Request{}
	resp := &protocol.Response{}
	c := app.NewContext(0)
	c.Request = *req
	c.Response = *resp
	return c
}

func TestWriter_WriteEvent(t *testing.T) {
	c := mockRequestContext()
	ctx := context.Background()

	writer := NewWriter(ctx, c)
	defer writer.Close()

	event := map[string]string{"type": "Test", "message": "Hello"}
	err := writer.WriteEvent(event)
	if err != nil {
		t.Fatalf("WriteEvent failed: %v", err)
	}

	body := c.Response.Body()
	if !bytes.HasPrefix(body, []byte("data: ")) {
		t.Errorf("Response should start with 'data: ', got: %s", string(body))
	}
	if !bytes.HasSuffix(body, []byte("\n\n")) {
		t.Errorf("Response should end with '\\n\\n', got: %s", string(body))
	}

	// Parse the JSON part
	jsonPart := body[6 : len(body)-2] // Remove "data: " prefix and "\n\n" suffix
	var parsed map[string]string
	if err := json.Unmarshal(jsonPart, &parsed); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	if parsed["type"] != "Test" {
		t.Errorf("type = %q, want %q", parsed["type"], "Test")
	}
	if parsed["message"] != "Hello" {
		t.Errorf("message = %q, want %q", parsed["message"], "Hello")
	}
}

func TestWriter_WriteRaw(t *testing.T) {
	c := mockRequestContext()
	ctx := context.Background()

	writer := NewWriter(ctx, c)
	defer writer.Close()

	rawData := []byte(`{"type":"Raw"}`)
	err := writer.WriteRaw(rawData)
	if err != nil {
		t.Fatalf("WriteRaw failed: %v", err)
	}

	body := c.Response.Body()
	expected := "data: {\"type\":\"Raw\"}\n\n"
	if string(body) != expected {
		t.Errorf("Body = %q, want %q", string(body), expected)
	}
}

func TestWriter_Headers(t *testing.T) {
	c := mockRequestContext()
	ctx := context.Background()

	writer := NewWriter(ctx, c)
	defer writer.Close()

	contentType := string(c.Response.Header.ContentType())
	if contentType != "text/event-stream" {
		t.Errorf("Content-Type = %q, want %q", contentType, "text/event-stream")
	}

	cacheControl := string(c.Response.Header.Peek("Cache-Control"))
	if cacheControl != "no-cache" {
		t.Errorf("Cache-Control = %q, want %q", cacheControl, "no-cache")
	}

	connection := string(c.Response.Header.Peek("Connection"))
	if connection != "keep-alive" {
		t.Errorf("Connection = %q, want %q", connection, "keep-alive")
	}
}

func TestWriter_Close(t *testing.T) {
	c := mockRequestContext()
	ctx := context.Background()

	writer := NewWriter(ctx, c)

	if writer.IsClosed() {
		t.Error("Writer should not be closed initially")
	}

	writer.Close()

	if !writer.IsClosed() {
		t.Error("Writer should be closed after Close()")
	}

	// Writing after close should fail
	err := writer.WriteEvent(map[string]string{"test": "value"})
	if err == nil {
		t.Error("WriteEvent should fail after Close()")
	}
}

func TestWriter_Done(t *testing.T) {
	c := mockRequestContext()
	ctx := context.Background()

	writer := NewWriter(ctx, c)
	done := writer.Done()

	// Check it's not closed initially
	select {
	case <-done:
		t.Error("Done channel should not be closed initially")
	default:
		// Good - channel is open
	}

	writer.Close()

	// Check it's closed after Close()
	select {
	case <-done:
		// Good - channel is closed
	case <-time.After(100 * time.Millisecond):
		t.Error("Done channel should be closed after Close()")
	}
}

func TestWriter_DoubleClose(t *testing.T) {
	c := mockRequestContext()
	ctx := context.Background()

	writer := NewWriter(ctx, c)

	// Should not panic on double close
	writer.Close()
	writer.Close()

	if !writer.IsClosed() {
		t.Error("Writer should be closed")
	}
}
