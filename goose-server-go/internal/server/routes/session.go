package routes

import (
	"context"
	"database/sql"

	"github.com/block/goose-server-go/internal/models"
	"github.com/block/goose-server-go/internal/session"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
)

// SessionRoutes handles session-related endpoints
type SessionRoutes struct {
	manager *session.Manager
}

// NewSessionRoutes creates a new SessionRoutes instance
func NewSessionRoutes(manager *session.Manager) *SessionRoutes {
	return &SessionRoutes{manager: manager}
}

// List handles GET /sessions
func (r *SessionRoutes) List(ctx context.Context, c *app.RequestContext) {
	sessions, err := r.manager.List()
	if err != nil {
		c.JSON(consts.StatusInternalServerError, map[string]string{
			"message": err.Error(),
		})
		return
	}

	c.JSON(consts.StatusOK, models.SessionListResponse{
		Sessions: sessions,
	})
}

// Get handles GET /sessions/:session_id
func (r *SessionRoutes) Get(ctx context.Context, c *app.RequestContext) {
	sessionID := c.Param("session_id")

	sess, err := r.manager.Get(sessionID, true)
	if err != nil {
		c.JSON(consts.StatusInternalServerError, map[string]string{
			"message": err.Error(),
		})
		return
	}

	if sess == nil {
		c.JSON(consts.StatusNotFound, map[string]string{
			"message": "Session not found",
		})
		return
	}

	c.JSON(consts.StatusOK, sess)
}

// Delete handles DELETE /sessions/:session_id
func (r *SessionRoutes) Delete(ctx context.Context, c *app.RequestContext) {
	sessionID := c.Param("session_id")

	err := r.manager.Delete(sessionID)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(consts.StatusNotFound, map[string]string{
				"message": "Session not found",
			})
			return
		}
		c.JSON(consts.StatusInternalServerError, map[string]string{
			"message": err.Error(),
		})
		return
	}

	c.JSON(consts.StatusOK, map[string]string{
		"message": "Session deleted",
	})
}

// Export handles GET /sessions/:session_id/export
func (r *SessionRoutes) Export(ctx context.Context, c *app.RequestContext) {
	sessionID := c.Param("session_id")

	text, err := r.manager.Export(sessionID)
	if err != nil {
		c.JSON(consts.StatusInternalServerError, map[string]string{
			"message": err.Error(),
		})
		return
	}

	c.Header("Content-Type", "text/plain")
	c.String(consts.StatusOK, text)
}

// UpdateNameRequest is the request body for updating session name
type UpdateNameRequest struct {
	Name string `json:"name"`
}

// UpdateName handles PUT /sessions/:session_id/name
func (r *SessionRoutes) UpdateName(ctx context.Context, c *app.RequestContext) {
	sessionID := c.Param("session_id")

	var req UpdateNameRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(consts.StatusBadRequest, map[string]string{
			"message": "Invalid request body",
		})
		return
	}

	if len(req.Name) > 200 {
		c.JSON(consts.StatusBadRequest, map[string]string{
			"message": "Name too long (max 200 characters)",
		})
		return
	}

	err := r.manager.UpdateName(sessionID, req.Name)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(consts.StatusNotFound, map[string]string{
				"message": "Session not found",
			})
			return
		}
		c.JSON(consts.StatusInternalServerError, map[string]string{
			"message": err.Error(),
		})
		return
	}

	c.JSON(consts.StatusOK, map[string]string{
		"message": "Name updated",
	})
}

// EditMessageRequest is the request body for editing a message
type EditMessageRequest struct {
	Timestamp int64   `json:"timestamp"`
	EditType  *string `json:"editType,omitempty"`
}

// EditMessageResponse is the response for editing a message
type EditMessageResponse struct {
	SessionID string `json:"sessionId"`
}

// EditMessage handles POST /sessions/:session_id/edit_message
func (r *SessionRoutes) EditMessage(ctx context.Context, c *app.RequestContext) {
	sessionID := c.Param("session_id")

	var req EditMessageRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(consts.StatusBadRequest, map[string]string{
			"message": "Invalid request body",
		})
		return
	}

	// Get the session
	sess, err := r.manager.Get(sessionID, true)
	if err != nil {
		c.JSON(consts.StatusInternalServerError, map[string]string{
			"message": err.Error(),
		})
		return
	}

	if sess == nil {
		c.JSON(consts.StatusNotFound, map[string]string{
			"message": "Session not found",
		})
		return
	}

	// Find the message index by timestamp
	messageIndex := -1
	for i, msg := range sess.Conversation {
		if msg.Created == req.Timestamp {
			messageIndex = i
			break
		}
	}

	if messageIndex == -1 {
		c.JSON(consts.StatusNotFound, map[string]string{
			"message": "Message not found",
		})
		return
	}

	// Default to "fork" to match Rust's default_edit_type() -> EditType::Fork
	editType := "fork"
	if req.EditType != nil {
		editType = *req.EditType
	}

	var resultSessionID string

	if editType == "fork" {
		// Create a new session with messages up to (but not including) the selected message
		newSession := models.NewSession(sess.WorkingDir)
		newSession.Name = sess.Name + " (fork)"
		newSession.Conversation = make(models.Conversation, messageIndex)
		copy(newSession.Conversation, sess.Conversation[:messageIndex])
		newSession.MessageCount = uint64(messageIndex)

		if err := r.manager.Update(newSession); err != nil {
			c.JSON(consts.StatusInternalServerError, map[string]string{
				"message": err.Error(),
			})
			return
		}
		resultSessionID = newSession.ID
	} else {
		// Truncate the conversation at the selected message
		sess.Conversation = sess.Conversation[:messageIndex]
		sess.MessageCount = uint64(messageIndex)

		if err := r.manager.Update(sess); err != nil {
			c.JSON(consts.StatusInternalServerError, map[string]string{
				"message": err.Error(),
			})
			return
		}
		resultSessionID = sess.ID
	}

	c.JSON(consts.StatusOK, EditMessageResponse{
		SessionID: resultSessionID,
	})
}

// ImportSessionRequest is the request body for importing a session
type ImportSessionRequest struct {
	JSON string `json:"json"`
}

// Import handles POST /sessions/import
func (r *SessionRoutes) Import(ctx context.Context, c *app.RequestContext) {
	var req ImportSessionRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(consts.StatusBadRequest, map[string]string{
			"message": "Invalid request body",
		})
		return
	}

	sess, err := r.manager.Import(req.JSON)
	if err != nil {
		c.JSON(consts.StatusBadRequest, map[string]string{
			"message": err.Error(),
		})
		return
	}

	c.JSON(consts.StatusOK, sess)
}

// GetInsights handles GET /sessions/insights
func (r *SessionRoutes) GetInsights(ctx context.Context, c *app.RequestContext) {
	insights, err := r.manager.GetInsights()
	if err != nil {
		c.JSON(consts.StatusInternalServerError, map[string]string{
			"message": err.Error(),
		})
		return
	}

	c.JSON(consts.StatusOK, insights)
}

// UpdateUserRecipeValuesRequest is the request body for updating user recipe values
type UpdateUserRecipeValuesRequest struct {
	UserRecipeValues map[string]string `json:"userRecipeValues"`
}

// UpdateUserRecipeValuesResponse is the response for updating user recipe values
type UpdateUserRecipeValuesResponse struct {
	Recipe *models.Recipe `json:"recipe"`
}

// UpdateUserRecipeValues handles PUT /sessions/:session_id/user_recipe_values
func (r *SessionRoutes) UpdateUserRecipeValues(ctx context.Context, c *app.RequestContext) {
	sessionID := c.Param("session_id")

	var req UpdateUserRecipeValuesRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(consts.StatusBadRequest, map[string]string{
			"message": "Invalid request body",
		})
		return
	}

	sess, err := r.manager.Get(sessionID, false)
	if err != nil {
		c.JSON(consts.StatusInternalServerError, map[string]string{
			"message": err.Error(),
		})
		return
	}

	if sess == nil {
		c.JSON(consts.StatusNotFound, map[string]string{
			"message": "Session not found",
		})
		return
	}

	sess.UserRecipeValues = req.UserRecipeValues

	if err := r.manager.Update(sess); err != nil {
		c.JSON(consts.StatusInternalServerError, map[string]string{
			"message": err.Error(),
		})
		return
	}

	c.JSON(consts.StatusOK, UpdateUserRecipeValuesResponse{
		Recipe: sess.Recipe,
	})
}
