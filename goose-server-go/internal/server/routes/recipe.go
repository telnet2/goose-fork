package routes

import (
	"context"

	"github.com/block/goose-server-go/internal/recipe"
	"github.com/block/goose-server-go/internal/scheduler"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
)

// RecipeRoutes handles recipe-related endpoints
type RecipeRoutes struct {
	storage   *recipe.Storage
	scheduler *scheduler.Scheduler
}

// NewRecipeRoutes creates a new RecipeRoutes instance
func NewRecipeRoutes(storage *recipe.Storage, sched *scheduler.Scheduler) *RecipeRoutes {
	return &RecipeRoutes{
		storage:   storage,
		scheduler: sched,
	}
}

// ListRecipesResponse is the response for listing recipes
type ListRecipesResponse struct {
	Manifests []recipe.RecipeManifest `json:"manifests"`
}

// List handles GET /recipes/list
func (r *RecipeRoutes) List(ctx context.Context, c *app.RequestContext) {
	manifests, err := r.storage.List()
	if err != nil {
		c.JSON(consts.StatusInternalServerError, map[string]string{
			"message": err.Error(),
		})
		return
	}

	// Add schedule info from scheduler
	if r.scheduler != nil {
		jobs := r.scheduler.ListJobs()
		jobMap := make(map[string]*scheduler.ScheduledJob)
		for _, job := range jobs {
			jobMap[job.Source] = job
		}

		for i := range manifests {
			if job, ok := jobMap[manifests[i].FilePath]; ok {
				manifests[i].ScheduleCron = &job.Cron
			}
		}
	}

	c.JSON(consts.StatusOK, ListRecipesResponse{
		Manifests: manifests,
	})
}

// CreateRecipeRequest is the request for creating a recipe
type CreateRecipeRequest struct {
	SessionID string `json:"session_id"`
}

// CreateRecipeResponse is the response for creating a recipe
type CreateRecipeResponse struct {
	Recipe recipe.Recipe `json:"recipe"`
}

// Create handles POST /recipes/create
func (r *RecipeRoutes) Create(ctx context.Context, c *app.RequestContext) {
	// TODO: Create recipe from session - requires session manager integration
	c.JSON(consts.StatusNotImplemented, map[string]string{
		"message": "Creating recipes from sessions not yet implemented",
	})
}

// ParseRecipeRequest is the request for parsing a recipe
type ParseRecipeRequest struct {
	Content string `json:"content"`
}

// ParseRecipeResponse is the response for parsing a recipe
type ParseRecipeResponse struct {
	Recipe recipe.Recipe `json:"recipe"`
}

// Parse handles POST /recipes/parse
func (r *RecipeRoutes) Parse(ctx context.Context, c *app.RequestContext) {
	var req ParseRecipeRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(consts.StatusBadRequest, map[string]string{
			"message": "Invalid request body",
		})
		return
	}

	parsed, err := recipe.FromContent(req.Content)
	if err != nil {
		c.JSON(consts.StatusBadRequest, map[string]string{
			"message": err.Error(),
		})
		return
	}

	c.JSON(consts.StatusOK, ParseRecipeResponse{
		Recipe: *parsed,
	})
}

// EncodeRecipeRequest is the request for encoding a recipe
type EncodeRecipeRequest struct {
	Recipe recipe.Recipe `json:"recipe"`
}

// EncodeRecipeResponse is the response for encoding a recipe
type EncodeRecipeResponse struct {
	Deeplink string `json:"deeplink"`
}

// Encode handles POST /recipes/encode
func (r *RecipeRoutes) Encode(ctx context.Context, c *app.RequestContext) {
	var req EncodeRecipeRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(consts.StatusBadRequest, map[string]string{
			"message": "Invalid request body",
		})
		return
	}

	deeplink, err := recipe.Encode(&req.Recipe)
	if err != nil {
		c.JSON(consts.StatusInternalServerError, map[string]string{
			"message": err.Error(),
		})
		return
	}

	c.JSON(consts.StatusOK, EncodeRecipeResponse{
		Deeplink: deeplink,
	})
}

// DecodeRecipeRequest is the request for decoding a recipe
type DecodeRecipeRequest struct {
	Deeplink string `json:"deeplink"`
}

// DecodeRecipeResponse is the response for decoding a recipe
type DecodeRecipeResponse struct {
	Recipe recipe.Recipe `json:"recipe"`
}

// Decode handles POST /recipes/decode
func (r *RecipeRoutes) Decode(ctx context.Context, c *app.RequestContext) {
	var req DecodeRecipeRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(consts.StatusBadRequest, map[string]string{
			"message": "Invalid request body",
		})
		return
	}

	decoded, err := recipe.Decode(req.Deeplink)
	if err != nil {
		c.JSON(consts.StatusBadRequest, map[string]string{
			"message": err.Error(),
		})
		return
	}

	c.JSON(consts.StatusOK, DecodeRecipeResponse{
		Recipe: *decoded,
	})
}

// SaveRecipeRequest is the request for saving a recipe
type SaveRecipeRequest struct {
	Recipe recipe.Recipe `json:"recipe"`
	ID     *string       `json:"id,omitempty"`
}

// SaveRecipeResponse is the response for saving a recipe
type SaveRecipeResponse struct {
	ID string `json:"id"`
}

// Save handles POST /recipes/save
func (r *RecipeRoutes) Save(ctx context.Context, c *app.RequestContext) {
	var req SaveRecipeRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(consts.StatusBadRequest, map[string]string{
			"message": "Invalid request body",
		})
		return
	}

	// Validate recipe
	if err := req.Recipe.Validate(); err != nil {
		c.JSON(consts.StatusBadRequest, map[string]string{
			"message": err.Error(),
		})
		return
	}

	// If ID provided, find existing file path
	var filePath *string
	if req.ID != nil {
		manifest, err := r.storage.Get(*req.ID)
		if err == nil {
			filePath = &manifest.FilePath
		}
	}

	// Save recipe
	savedPath, err := r.storage.Save(&req.Recipe, filePath)
	if err != nil {
		c.JSON(consts.StatusInternalServerError, map[string]string{
			"message": err.Error(),
		})
		return
	}

	// Get the new ID
	manifest, _ := r.storage.GetByFilePath(savedPath)
	id := req.Recipe.GenerateID()
	if manifest != nil {
		id = manifest.ID
	}

	c.JSON(consts.StatusOK, SaveRecipeResponse{
		ID: id,
	})
}

// DeleteRecipeRequest is the request for deleting a recipe
type DeleteRecipeRequest struct {
	ID string `json:"id"`
}

// Delete handles POST /recipes/delete
func (r *RecipeRoutes) Delete(ctx context.Context, c *app.RequestContext) {
	var req DeleteRecipeRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(consts.StatusBadRequest, map[string]string{
			"message": "Invalid request body",
		})
		return
	}

	if err := r.storage.Delete(req.ID); err != nil {
		c.JSON(consts.StatusNotFound, map[string]string{
			"message": err.Error(),
		})
		return
	}

	c.Status(consts.StatusNoContent)
}

// ScanRecipeRequest is the request for scanning a recipe
type ScanRecipeRequest struct {
	Recipe recipe.Recipe `json:"recipe"`
}

// ScanRecipeResponse is the response for scanning a recipe
type ScanRecipeResponse struct {
	HasSecurityWarnings bool     `json:"has_security_warnings"`
	Warnings            []string `json:"warnings,omitempty"`
}

// Scan handles POST /recipes/scan
func (r *RecipeRoutes) Scan(ctx context.Context, c *app.RequestContext) {
	var req ScanRecipeRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(consts.StatusBadRequest, map[string]string{
			"message": "Invalid request body",
		})
		return
	}

	warnings := req.Recipe.CheckSecurityWarnings()

	c.JSON(consts.StatusOK, ScanRecipeResponse{
		HasSecurityWarnings: len(warnings) > 0,
		Warnings:            warnings,
	})
}

// ScheduleRecipeRequest is the request for scheduling a recipe
type ScheduleRecipeRequest struct {
	ID           string  `json:"id"`
	CronSchedule *string `json:"cron_schedule,omitempty"`
}

// Schedule handles POST /recipes/schedule
func (r *RecipeRoutes) Schedule(ctx context.Context, c *app.RequestContext) {
	var req ScheduleRecipeRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(consts.StatusBadRequest, map[string]string{
			"message": "Invalid request body",
		})
		return
	}

	if r.scheduler == nil {
		c.JSON(consts.StatusInternalServerError, map[string]string{
			"message": "Scheduler not available",
		})
		return
	}

	// Get recipe file path
	manifest, err := r.storage.Get(req.ID)
	if err != nil {
		c.JSON(consts.StatusNotFound, map[string]string{
			"message": err.Error(),
		})
		return
	}

	// Schedule or unschedule
	if err := r.scheduler.ScheduleRecipe(manifest.FilePath, req.CronSchedule); err != nil {
		c.JSON(consts.StatusInternalServerError, map[string]string{
			"message": err.Error(),
		})
		return
	}

	c.Status(consts.StatusOK)
}

// SlashCommandRequest is the request for setting a slash command
type SlashCommandRequest struct {
	ID           string  `json:"id"`
	SlashCommand *string `json:"slash_command,omitempty"`
}

// SetSlashCommand handles POST /recipes/slash-command
func (r *RecipeRoutes) SetSlashCommand(ctx context.Context, c *app.RequestContext) {
	var req SlashCommandRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(consts.StatusBadRequest, map[string]string{
			"message": "Invalid request body",
		})
		return
	}

	// Update slash command in storage
	if err := r.storage.UpdateSlashCommand(req.ID, req.SlashCommand); err != nil {
		c.JSON(consts.StatusInternalServerError, map[string]string{
			"message": err.Error(),
		})
		return
	}

	c.Status(consts.StatusOK)
}
