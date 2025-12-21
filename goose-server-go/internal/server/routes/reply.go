package routes

import (
	"context"
	"time"

	"github.com/block/goose-server-go/internal/agent"
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
	agentManager   *agent.Manager
}

// NewReplyRoutes creates a new ReplyRoutes instance
func NewReplyRoutes(sessionMgr *session.Manager, agentMgr *agent.Manager) *ReplyRoutes {
	return &ReplyRoutes{
		sessionManager: sessionMgr,
		agentManager:   agentMgr,
	}
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
			"message": "Session not found",
		})
		return
	}

	// Get or resume agent
	ag, ok := r.agentManager.Get(req.SessionID)
	if !ok {
		// Try to resume agent
		var resumeErr error
		ag, resumeErr = r.agentManager.Resume(ctx, req.SessionID, true)
		if resumeErr != nil {
			c.JSON(consts.StatusFailedDependency, map[string]string{
				"message": "Agent not active. Please start or resume the agent first: " + resumeErr.Error(),
			})
			return
		}
	}

	// Create SSE writer
	writer := sse.NewWriter(ctx, c)
	defer writer.Close()

	// Start heartbeat
	writer.StartHeartbeat(SSEHeartbeatInterval)

	// Process messages and stream response
	r.streamResponse(ctx, writer, sess, ag, req.Messages)
}

// streamResponse processes the chat request and streams SSE events
func (r *ReplyRoutes) streamResponse(ctx context.Context, writer *sse.Writer, sess *models.Session, ag *agent.Agent, messages []models.Message) {
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

	// Send messages to agent for processing
	eventChan, err := ag.Chat(ctx, sess.Conversation)
	if err != nil {
		writer.WriteEvent(models.NewErrorEvent("Failed to process chat: " + err.Error()))
		return
	}

	// Stream events from agent
	for event := range eventChan {
		select {
		case <-ctx.Done():
			writer.WriteEvent(models.NewErrorEvent("Request cancelled"))
			return
		default:
		}

		// Update token state from event
		if event.TokenState != nil {
			tokenState = event.TokenState
		}

		// Add message to conversation if it's a Message event
		if event.Type == models.EventTypeMessage && event.Message != nil {
			sess.Conversation = append(sess.Conversation, *event.Message)
		}

		// Write event to SSE stream
		if err := writer.WriteEvent(event); err != nil {
			// Client disconnected
			return
		}
	}

	// Update session with final state
	sess.MessageCount = uint64(len(sess.Conversation))
	sess.InputTokens = &tokenState.InputTokens
	sess.OutputTokens = &tokenState.OutputTokens
	sess.TotalTokens = &tokenState.TotalTokens
	sess.AccumulatedInputTokens = &tokenState.AccumulatedInputTokens
	sess.AccumulatedOutputTokens = &tokenState.AccumulatedOutputTokens
	sess.AccumulatedTotalTokens = &tokenState.AccumulatedTotalTokens

	// Save session
	if err := r.sessionManager.Update(sess); err != nil {
		writer.WriteEvent(models.NewErrorEvent("Failed to save session: " + err.Error()))
	}
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
