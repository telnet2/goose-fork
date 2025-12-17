package routes

import (
	"context"
	"time"

	"github.com/block/goose-server-go/internal/models"
	"github.com/block/goose-server-go/internal/session"
	"github.com/block/goose-server-go/pkg/sse"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
)

const (
	// SSEHeartbeatInterval is the interval for sending ping events
	SSEHeartbeatInterval = 500 * time.Millisecond
	// MaxRequestBodySize is the maximum size for /reply request body (50MB)
	MaxRequestBodySize = 50 * 1024 * 1024
)

// ReplyRoutes handles the /reply endpoint
type ReplyRoutes struct {
	sessionManager *session.Manager
}

// NewReplyRoutes creates a new ReplyRoutes instance
func NewReplyRoutes(manager *session.Manager) *ReplyRoutes {
	return &ReplyRoutes{sessionManager: manager}
}

// Reply handles POST /reply - the main SSE streaming endpoint
func (r *ReplyRoutes) Reply(ctx context.Context, c *app.RequestContext) {
	// Parse request body
	var req models.ChatRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(consts.StatusBadRequest, map[string]string{
			"message": "Invalid request body: " + err.Error(),
		})
		return
	}

	// Validate required fields
	if req.SessionID == "" {
		c.JSON(consts.StatusBadRequest, map[string]string{
			"message": "session_id is required",
		})
		return
	}

	// Get session (with conversation since we need to append messages)
	sess, err := r.sessionManager.Get(req.SessionID, true)
	if err != nil {
		c.JSON(consts.StatusInternalServerError, map[string]string{
			"message": "Failed to get session: " + err.Error(),
		})
		return
	}

	if sess == nil {
		c.JSON(consts.StatusFailedDependency, map[string]string{
			"message": "Session not found or not active",
		})
		return
	}

	// Create SSE writer
	writer := sse.NewWriter(ctx, c)
	defer writer.Close()

	// Start heartbeat
	writer.StartHeartbeat(SSEHeartbeatInterval)

	// Process messages and stream response
	r.streamResponse(ctx, writer, sess, req.Messages)
}

// streamResponse processes the chat request and streams SSE events
func (r *ReplyRoutes) streamResponse(ctx context.Context, writer *sse.Writer, sess *models.Session, messages []models.Message) {
	// Add incoming messages to session conversation
	for _, msg := range messages {
		sess.Conversation = append(sess.Conversation, msg)
	}

	// Initialize token tracking
	tokenState := &models.TokenState{
		InputTokens:             0,
		OutputTokens:            0,
		TotalTokens:             0,
		AccumulatedInputTokens:  0,
		AccumulatedOutputTokens: 0,
		AccumulatedTotalTokens:  0,
	}

	// Copy accumulated tokens from session if available
	if sess.AccumulatedInputTokens != nil {
		tokenState.AccumulatedInputTokens = *sess.AccumulatedInputTokens
	}
	if sess.AccumulatedOutputTokens != nil {
		tokenState.AccumulatedOutputTokens = *sess.AccumulatedOutputTokens
	}
	if sess.AccumulatedTotalTokens != nil {
		tokenState.AccumulatedTotalTokens = *sess.AccumulatedTotalTokens
	}

	// TODO: In Phase 4, this will call the actual agent/LLM provider
	// For now, implement a mock response that demonstrates SSE protocol

	// Simulate processing time
	select {
	case <-ctx.Done():
		writer.WriteEvent(models.NewErrorEvent("Request cancelled"))
		return
	case <-time.After(100 * time.Millisecond):
	}

	// Generate mock assistant response
	// In production, this would come from the LLM provider
	responseText := "I received your message. This is a mock response from the Go server. " +
		"In Phase 4, I will be replaced with actual LLM integration."

	// Create assistant message
	assistantMsg := models.NewAssistantMessage(responseText)

	// Simulate token usage
	tokenState.InputTokens = 50
	tokenState.OutputTokens = 30
	tokenState.TotalTokens = 80
	tokenState.AccumulatedInputTokens += tokenState.InputTokens
	tokenState.AccumulatedOutputTokens += tokenState.OutputTokens
	tokenState.AccumulatedTotalTokens += tokenState.TotalTokens

	// Send Message event
	if err := writer.WriteEvent(models.NewMessageEvent(assistantMsg, tokenState)); err != nil {
		// Client disconnected
		return
	}

	// Add assistant message to conversation
	sess.Conversation = append(sess.Conversation, assistantMsg)
	sess.MessageCount = uint64(len(sess.Conversation))

	// Update session token counts
	sess.InputTokens = &tokenState.InputTokens
	sess.OutputTokens = &tokenState.OutputTokens
	sess.TotalTokens = &tokenState.TotalTokens
	sess.AccumulatedInputTokens = &tokenState.AccumulatedInputTokens
	sess.AccumulatedOutputTokens = &tokenState.AccumulatedOutputTokens
	sess.AccumulatedTotalTokens = &tokenState.AccumulatedTotalTokens

	// Save session
	if err := r.sessionManager.Update(sess); err != nil {
		writer.WriteEvent(models.NewErrorEvent("Failed to save session: " + err.Error()))
		return
	}

	// Send Finish event
	reason := string(models.FinishReasonStop)
	writer.WriteEvent(models.NewFinishEvent(reason, tokenState))
}

// Reply is the legacy function for backward compatibility
func Reply(state interface{}) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		// Check if state is a ReplyRoutes instance
		if r, ok := state.(*ReplyRoutes); ok {
			r.Reply(ctx, c)
			return
		}

		// Fallback: not implemented
		c.JSON(consts.StatusNotImplemented, map[string]string{
			"message": "Reply endpoint not properly configured",
		})
	}
}
