package routes

import (
	"context"
	"strconv"
	"time"

	"github.com/block/goose-server-go/internal/scheduler"
	"github.com/block/goose-server-go/internal/session"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
)

// ScheduleRoutes handles scheduling-related endpoints
type ScheduleRoutes struct {
	scheduler      *scheduler.Scheduler
	sessionManager *session.Manager
}

// NewScheduleRoutes creates a new ScheduleRoutes instance
func NewScheduleRoutes(sched *scheduler.Scheduler, sessMgr *session.Manager) *ScheduleRoutes {
	return &ScheduleRoutes{
		scheduler:      sched,
		sessionManager: sessMgr,
	}
}

// ListSchedulesResponse is the response for listing schedules
type ListSchedulesResponse struct {
	Jobs []*scheduler.ScheduledJob `json:"jobs"`
}

// List handles GET /schedule/list
func (r *ScheduleRoutes) List(ctx context.Context, c *app.RequestContext) {
	if r.scheduler == nil {
		c.JSON(consts.StatusInternalServerError, map[string]string{
			"message": "Scheduler not available",
		})
		return
	}

	jobs := r.scheduler.ListJobs()
	c.JSON(consts.StatusOK, ListSchedulesResponse{
		Jobs: jobs,
	})
}

// CreateScheduleRequest is the request for creating a schedule
type CreateScheduleRequest struct {
	ID           string `json:"id"`
	RecipeSource string `json:"recipe_source"`
	Cron         string `json:"cron"`
}

// Create handles POST /schedule/create
func (r *ScheduleRoutes) Create(ctx context.Context, c *app.RequestContext) {
	if r.scheduler == nil {
		c.JSON(consts.StatusInternalServerError, map[string]string{
			"message": "Scheduler not available",
		})
		return
	}

	var req CreateScheduleRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(consts.StatusBadRequest, map[string]string{
			"message": "Invalid request body",
		})
		return
	}

	if req.ID == "" {
		c.JSON(consts.StatusBadRequest, map[string]string{
			"message": "Job ID is required",
		})
		return
	}

	if req.RecipeSource == "" {
		c.JSON(consts.StatusBadRequest, map[string]string{
			"message": "Recipe source is required",
		})
		return
	}

	if req.Cron == "" {
		c.JSON(consts.StatusBadRequest, map[string]string{
			"message": "Cron expression is required",
		})
		return
	}

	job := &scheduler.ScheduledJob{
		ID:     req.ID,
		Source: req.RecipeSource,
		Cron:   req.Cron,
	}

	if err := r.scheduler.AddJob(job, true); err != nil {
		c.JSON(consts.StatusInternalServerError, map[string]string{
			"message": err.Error(),
		})
		return
	}

	// Return the created job
	createdJob, _ := r.scheduler.GetJob(req.ID)
	c.JSON(consts.StatusCreated, createdJob)
}

// UpdateScheduleRequest is the request for updating a schedule
type UpdateScheduleRequest struct {
	Cron string `json:"cron"`
}

// Update handles PUT /schedule/:id
func (r *ScheduleRoutes) Update(ctx context.Context, c *app.RequestContext) {
	if r.scheduler == nil {
		c.JSON(consts.StatusInternalServerError, map[string]string{
			"message": "Scheduler not available",
		})
		return
	}

	id := c.Param("id")

	var req UpdateScheduleRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(consts.StatusBadRequest, map[string]string{
			"message": "Invalid request body",
		})
		return
	}

	if req.Cron == "" {
		c.JSON(consts.StatusBadRequest, map[string]string{
			"message": "Cron expression is required",
		})
		return
	}

	if err := r.scheduler.UpdateCron(id, req.Cron); err != nil {
		statusCode := consts.StatusInternalServerError
		if schedErr, ok := err.(*scheduler.SchedulerError); ok && schedErr.Code == scheduler.ErrJobNotFound {
			statusCode = consts.StatusNotFound
		}
		c.JSON(statusCode, map[string]string{
			"message": err.Error(),
		})
		return
	}

	// Return the updated job
	updatedJob, _ := r.scheduler.GetJob(id)
	c.JSON(consts.StatusOK, updatedJob)
}

// Delete handles DELETE /schedule/delete/:id
func (r *ScheduleRoutes) Delete(ctx context.Context, c *app.RequestContext) {
	if r.scheduler == nil {
		c.JSON(consts.StatusInternalServerError, map[string]string{
			"message": "Scheduler not available",
		})
		return
	}

	id := c.Param("id")

	if err := r.scheduler.RemoveJob(id, true); err != nil {
		statusCode := consts.StatusInternalServerError
		if schedErr, ok := err.(*scheduler.SchedulerError); ok && schedErr.Code == scheduler.ErrJobNotFound {
			statusCode = consts.StatusNotFound
		}
		c.JSON(statusCode, map[string]string{
			"message": err.Error(),
		})
		return
	}

	c.Status(consts.StatusNoContent)
}

// Pause handles POST /schedule/:id/pause
func (r *ScheduleRoutes) Pause(ctx context.Context, c *app.RequestContext) {
	if r.scheduler == nil {
		c.JSON(consts.StatusInternalServerError, map[string]string{
			"message": "Scheduler not available",
		})
		return
	}

	id := c.Param("id")

	if err := r.scheduler.PauseJob(id); err != nil {
		statusCode := consts.StatusInternalServerError
		if schedErr, ok := err.(*scheduler.SchedulerError); ok && schedErr.Code == scheduler.ErrJobNotFound {
			statusCode = consts.StatusNotFound
		}
		c.JSON(statusCode, map[string]string{
			"message": err.Error(),
		})
		return
	}

	c.Status(consts.StatusNoContent)
}

// Unpause handles POST /schedule/:id/unpause
func (r *ScheduleRoutes) Unpause(ctx context.Context, c *app.RequestContext) {
	if r.scheduler == nil {
		c.JSON(consts.StatusInternalServerError, map[string]string{
			"message": "Scheduler not available",
		})
		return
	}

	id := c.Param("id")

	if err := r.scheduler.UnpauseJob(id); err != nil {
		statusCode := consts.StatusInternalServerError
		if schedErr, ok := err.(*scheduler.SchedulerError); ok && schedErr.Code == scheduler.ErrJobNotFound {
			statusCode = consts.StatusNotFound
		}
		c.JSON(statusCode, map[string]string{
			"message": err.Error(),
		})
		return
	}

	c.Status(consts.StatusNoContent)
}

// RunNowResponse is the response for running a job immediately
type RunNowResponse struct {
	SessionID string `json:"session_id"`
}

// RunNow handles POST /schedule/:id/run_now
func (r *ScheduleRoutes) RunNow(ctx context.Context, c *app.RequestContext) {
	if r.scheduler == nil {
		c.JSON(consts.StatusInternalServerError, map[string]string{
			"message": "Scheduler not available",
		})
		return
	}

	id := c.Param("id")

	sessionID, err := r.scheduler.RunNow(id)
	if err != nil {
		statusCode := consts.StatusInternalServerError
		if schedErr, ok := err.(*scheduler.SchedulerError); ok && schedErr.Code == scheduler.ErrJobNotFound {
			statusCode = consts.StatusNotFound
		}
		c.JSON(statusCode, map[string]string{
			"message": err.Error(),
		})
		return
	}

	c.JSON(consts.StatusOK, RunNowResponse{
		SessionID: sessionID,
	})
}

// KillResponse is the response for killing a running job
type KillResponse struct {
	Message string `json:"message"`
}

// Kill handles POST /schedule/:id/kill
func (r *ScheduleRoutes) Kill(ctx context.Context, c *app.RequestContext) {
	if r.scheduler == nil {
		c.JSON(consts.StatusInternalServerError, map[string]string{
			"message": "Scheduler not available",
		})
		return
	}

	id := c.Param("id")

	if err := r.scheduler.KillRunningJob(id); err != nil {
		statusCode := consts.StatusInternalServerError
		if schedErr, ok := err.(*scheduler.SchedulerError); ok && schedErr.Code == scheduler.ErrJobNotFound {
			statusCode = consts.StatusNotFound
		}
		c.JSON(statusCode, map[string]string{
			"message": err.Error(),
		})
		return
	}

	c.JSON(consts.StatusOK, KillResponse{
		Message: "Job killed successfully",
	})
}

// InspectJobResponse is the response for inspecting a job
type InspectJobResponse struct {
	SessionID              *string `json:"session_id,omitempty"`
	ProcessStartTime       *string `json:"process_start_time,omitempty"`
	RunningDurationSeconds *int64  `json:"running_duration_seconds,omitempty"`
}

// Inspect handles GET /schedule/:id/inspect
func (r *ScheduleRoutes) Inspect(ctx context.Context, c *app.RequestContext) {
	if r.scheduler == nil {
		c.JSON(consts.StatusInternalServerError, map[string]string{
			"message": "Scheduler not available",
		})
		return
	}

	id := c.Param("id")

	sessionID, startTime, err := r.scheduler.GetRunningJobInfo(id)
	if err != nil {
		statusCode := consts.StatusInternalServerError
		if schedErr, ok := err.(*scheduler.SchedulerError); ok && schedErr.Code == scheduler.ErrJobNotFound {
			statusCode = consts.StatusNotFound
		}
		c.JSON(statusCode, map[string]string{
			"message": err.Error(),
		})
		return
	}

	response := InspectJobResponse{
		SessionID: sessionID,
	}

	if startTime != nil {
		timeStr := startTime.Format(time.RFC3339)
		response.ProcessStartTime = &timeStr

		duration := int64(time.Since(*startTime).Seconds())
		response.RunningDurationSeconds = &duration
	}

	c.JSON(consts.StatusOK, response)
}

// SessionDisplayInfo is a simplified session info for display
type SessionDisplayInfo struct {
	ID                       string  `json:"id"`
	Name                     string  `json:"name"`
	CreatedAt                string  `json:"created_at"`
	WorkingDir               string  `json:"working_dir"`
	ScheduleID               *string `json:"schedule_id,omitempty"`
	MessageCount             uint64  `json:"message_count"`
	TotalTokens              *int32  `json:"total_tokens,omitempty"`
	InputTokens              *int32  `json:"input_tokens,omitempty"`
	OutputTokens             *int32  `json:"output_tokens,omitempty"`
	AccumulatedTotalTokens   *int32  `json:"accumulated_total_tokens,omitempty"`
	AccumulatedInputTokens   *int32  `json:"accumulated_input_tokens,omitempty"`
	AccumulatedOutputTokens  *int32  `json:"accumulated_output_tokens,omitempty"`
}

// GetSessions handles GET /schedule/:id/sessions
func (r *ScheduleRoutes) GetSessions(ctx context.Context, c *app.RequestContext) {
	if r.scheduler == nil || r.sessionManager == nil {
		c.JSON(consts.StatusInternalServerError, map[string]string{
			"message": "Scheduler or session manager not available",
		})
		return
	}

	id := c.Param("id")

	// Get limit from query parameter (default 10)
	limit := 10
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	// Get all sessions and filter by schedule ID
	allSessions, err := r.sessionManager.List()
	if err != nil {
		c.JSON(consts.StatusInternalServerError, map[string]string{
			"message": err.Error(),
		})
		return
	}

	var matchingSessions []SessionDisplayInfo
	for _, sess := range allSessions {
		if sess.ScheduleID != nil && *sess.ScheduleID == id {
			info := SessionDisplayInfo{
				ID:                       sess.ID,
				Name:                     sess.Name,
				CreatedAt:                sess.CreatedAt.Format(time.RFC3339),
				WorkingDir:               sess.WorkingDir,
				ScheduleID:               sess.ScheduleID,
				MessageCount:             sess.MessageCount,
				TotalTokens:              sess.TotalTokens,
				InputTokens:              sess.InputTokens,
				OutputTokens:             sess.OutputTokens,
				AccumulatedTotalTokens:   sess.AccumulatedTotalTokens,
				AccumulatedInputTokens:   sess.AccumulatedInputTokens,
				AccumulatedOutputTokens:  sess.AccumulatedOutputTokens,
			}
			matchingSessions = append(matchingSessions, info)

			if len(matchingSessions) >= limit {
				break
			}
		}
	}

	c.JSON(consts.StatusOK, matchingSessions)
}
