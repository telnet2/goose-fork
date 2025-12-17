package server

import (
	"context"
	"fmt"
	"sync"

	"github.com/block/goose-server-go/internal/config"
	"github.com/block/goose-server-go/internal/server/middleware"
	"github.com/block/goose-server-go/internal/server/routes"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/rs/zerolog/log"
)

// Server represents the goose HTTP server
type Server struct {
	config *config.Config
	hertz  *server.Hertz
	state  *AppState
	mu     sync.Mutex
}

// AppState holds the global application state
type AppState struct {
	Config *config.Config

	// Active agents per session
	agents sync.Map // map[string]*Agent

	// Recipe run tracking
	recipeRuns sync.Map // map[string]bool
}

// NewAppState creates a new application state
func NewAppState(cfg *config.Config) *AppState {
	return &AppState{
		Config: cfg,
	}
}

// MarkRecipeRunIfAbsent marks a recipe as run for a session, returns true if it was absent
func (s *AppState) MarkRecipeRunIfAbsent(sessionID string) bool {
	_, loaded := s.recipeRuns.LoadOrStore(sessionID, true)
	return !loaded
}

// New creates a new server instance
func New(cfg *config.Config) *Server {
	state := NewAppState(cfg)

	// Create Hertz server
	h := server.Default(
		server.WithHostPorts(fmt.Sprintf(":%d", cfg.Port)),
		server.WithMaxRequestBodySize(50*1024*1024), // 50MB for /reply endpoint
	)

	srv := &Server{
		config: cfg,
		hertz:  h,
		state:  state,
	}

	// Setup routes
	srv.setupRoutes()

	return srv
}

// setupRoutes configures all HTTP routes
func (s *Server) setupRoutes() {
	// Global middleware
	s.hertz.Use(middleware.Logger())
	s.hertz.Use(middleware.Recovery())
	s.hertz.Use(middleware.CORS())

	// Public routes (no auth required)
	s.hertz.GET("/status", routes.Status)

	// Protected routes (require auth)
	protected := s.hertz.Group("/")
	protected.Use(middleware.Auth(s.config.SecretKey))

	// Agent routes
	agentRoutes := routes.NewAgentRoutes(s.state)
	protected.POST("/agent/start", agentRoutes.Start)
	protected.POST("/agent/resume", agentRoutes.Resume)
	protected.POST("/agent/add_extension", agentRoutes.AddExtension)
	protected.POST("/agent/remove_extension", agentRoutes.RemoveExtension)
	protected.GET("/agent/tools", agentRoutes.GetTools)
	protected.POST("/agent/call_tool", agentRoutes.CallTool)
	protected.POST("/agent/read_resource", agentRoutes.ReadResource)
	protected.POST("/agent/update_provider", agentRoutes.UpdateProvider)
	protected.POST("/agent/update_from_session", agentRoutes.UpdateFromSession)
	protected.POST("/agent/update_router_tool_selector", agentRoutes.UpdateRouterToolSelector)

	// Reply route (SSE streaming)
	protected.POST("/reply", routes.Reply(s.state))

	// Session routes
	sessionRoutes := routes.NewSessionRoutes(s.state)
	protected.GET("/sessions", sessionRoutes.List)
	protected.GET("/sessions/insights", sessionRoutes.GetInsights)
	protected.POST("/sessions/import", sessionRoutes.Import)
	protected.GET("/sessions/:session_id", sessionRoutes.Get)
	protected.DELETE("/sessions/:session_id", sessionRoutes.Delete)
	protected.GET("/sessions/:session_id/export", sessionRoutes.Export)
	protected.PUT("/sessions/:session_id/name", sessionRoutes.UpdateName)
	protected.POST("/sessions/:session_id/edit_message", sessionRoutes.EditMessage)
	protected.PUT("/sessions/:session_id/user_recipe_values", sessionRoutes.UpdateUserRecipeValues)

	// Config routes
	configRoutes := routes.NewConfigRoutes(s.state)
	protected.GET("/config", configRoutes.Get)
	protected.POST("/config/read", configRoutes.Read)
	protected.POST("/config/upsert", configRoutes.Upsert)
	protected.POST("/config/remove", configRoutes.Remove)
	protected.POST("/config/init", configRoutes.Init)
	protected.GET("/config/validate", configRoutes.Validate)
	protected.POST("/config/backup", configRoutes.Backup)
	protected.POST("/config/recover", configRoutes.Recover)
	protected.GET("/config/providers", configRoutes.ListProviders)
	protected.GET("/config/providers/:name/models", configRoutes.GetProviderModels)
	protected.POST("/config/set_provider", configRoutes.SetProvider)
	protected.POST("/config/check_provider", configRoutes.CheckProvider)
	protected.POST("/config/detect-provider", configRoutes.DetectProvider)
	protected.GET("/config/extensions", configRoutes.GetExtensions)
	protected.POST("/config/extensions", configRoutes.AddExtension)
	protected.DELETE("/config/extensions/:name", configRoutes.RemoveExtension)
	protected.POST("/config/permissions", configRoutes.UpdatePermissions)
	protected.GET("/config/slash_commands", configRoutes.GetSlashCommands)

	// Recipe routes
	recipeRoutes := routes.NewRecipeRoutes(s.state)
	protected.GET("/recipes/list", recipeRoutes.List)
	protected.POST("/recipes/create", recipeRoutes.Create)
	protected.POST("/recipes/parse", recipeRoutes.Parse)
	protected.POST("/recipes/encode", recipeRoutes.Encode)
	protected.POST("/recipes/decode", recipeRoutes.Decode)
	protected.POST("/recipes/save", recipeRoutes.Save)
	protected.POST("/recipes/delete", recipeRoutes.Delete)
	protected.POST("/recipes/scan", recipeRoutes.Scan)
	protected.POST("/recipes/schedule", recipeRoutes.Schedule)
	protected.POST("/recipes/slash-command", recipeRoutes.SetSlashCommand)

	// Schedule routes
	scheduleRoutes := routes.NewScheduleRoutes(s.state)
	protected.GET("/schedule/list", scheduleRoutes.List)
	protected.POST("/schedule/create", scheduleRoutes.Create)
	protected.PUT("/schedule/:id", scheduleRoutes.Update)
	protected.DELETE("/schedule/delete/:id", scheduleRoutes.Delete)
	protected.POST("/schedule/:id/pause", scheduleRoutes.Pause)
	protected.POST("/schedule/:id/unpause", scheduleRoutes.Unpause)
	protected.POST("/schedule/:id/run_now", scheduleRoutes.RunNow)
	protected.POST("/schedule/:id/kill", scheduleRoutes.Kill)
	protected.GET("/schedule/:id/inspect", scheduleRoutes.Inspect)
	protected.GET("/schedule/:id/sessions", scheduleRoutes.GetSessions)

	// Action required routes
	protected.POST("/action-required/tool-confirmation", routes.ToolConfirmation(s.state))

	// Tunnel routes
	tunnelRoutes := routes.NewTunnelRoutes(s.state)
	protected.POST("/tunnel/start", tunnelRoutes.Start)
	protected.POST("/tunnel/stop", tunnelRoutes.Stop)
	protected.GET("/tunnel/status", tunnelRoutes.Status)

	// Diagnostics
	protected.GET("/diagnostics/:session_id", routes.Diagnostics(s.state))

	log.Info().Msg("Routes configured")
}

// Start starts the HTTP server
func (s *Server) Start() error {
	log.Info().Int("port", s.config.Port).Msg("Server starting")
	s.hertz.Spin()
	return nil
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	ctx := context.Background()
	return s.hertz.Shutdown(ctx)
}
