package routes

import (
	"context"

	"github.com/block/goose-server-go/internal/agent"
	"github.com/block/goose-server-go/internal/provider"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
)

// ConfigRoutes handles configuration-related endpoints
type ConfigRoutes struct {
	agentManager *agent.Manager
}

// NewConfigRoutes creates a new ConfigRoutes instance
func NewConfigRoutes(agentManager *agent.Manager) *ConfigRoutes {
	return &ConfigRoutes{agentManager: agentManager}
}

// Get handles GET /config
func (r *ConfigRoutes) Get(ctx context.Context, c *app.RequestContext) {
	// TODO: Implement full config retrieval in Phase 7
	c.JSON(consts.StatusOK, map[string]interface{}{
		"version": "1.0.0",
		"providers": r.agentManager.GetProviderMetadata(),
	})
}

// Read handles POST /config/read
func (r *ConfigRoutes) Read(ctx context.Context, c *app.RequestContext) {
	// TODO: Implement in Phase 7
	c.JSON(consts.StatusNotImplemented, map[string]string{
		"message": "Not implemented - Phase 7",
	})
}

// Upsert handles POST /config/upsert
func (r *ConfigRoutes) Upsert(ctx context.Context, c *app.RequestContext) {
	// TODO: Implement in Phase 7
	c.JSON(consts.StatusNotImplemented, map[string]string{
		"message": "Not implemented - Phase 7",
	})
}

// Remove handles POST /config/remove
func (r *ConfigRoutes) Remove(ctx context.Context, c *app.RequestContext) {
	// TODO: Implement in Phase 7
	c.JSON(consts.StatusNotImplemented, map[string]string{
		"message": "Not implemented - Phase 7",
	})
}

// Init handles POST /config/init
func (r *ConfigRoutes) Init(ctx context.Context, c *app.RequestContext) {
	// TODO: Implement in Phase 7
	c.JSON(consts.StatusNotImplemented, map[string]string{
		"message": "Not implemented - Phase 7",
	})
}

// Validate handles GET /config/validate
func (r *ConfigRoutes) Validate(ctx context.Context, c *app.RequestContext) {
	// TODO: Implement in Phase 7
	c.JSON(consts.StatusNotImplemented, map[string]string{
		"message": "Not implemented - Phase 7",
	})
}

// Backup handles POST /config/backup
func (r *ConfigRoutes) Backup(ctx context.Context, c *app.RequestContext) {
	// TODO: Implement in Phase 7
	c.JSON(consts.StatusNotImplemented, map[string]string{
		"message": "Not implemented - Phase 7",
	})
}

// Recover handles POST /config/recover
func (r *ConfigRoutes) Recover(ctx context.Context, c *app.RequestContext) {
	// TODO: Implement in Phase 7
	c.JSON(consts.StatusNotImplemented, map[string]string{
		"message": "Not implemented - Phase 7",
	})
}

// ProviderResponse represents a provider in the response
type ProviderResponse struct {
	Name         string                `json:"name"`
	DisplayName  string                `json:"display_name"`
	Description  string                `json:"description,omitempty"`
	IsConfigured bool                  `json:"is_configured"`
	DefaultModel string                `json:"default_model"`
	ConfigKeys   []provider.ConfigKey  `json:"config_keys,omitempty"`
	ModelDocLink string                `json:"model_doc_link,omitempty"`
}

// ListProviders handles GET /config/providers
func (r *ConfigRoutes) ListProviders(ctx context.Context, c *app.RequestContext) {
	metadata := r.agentManager.GetProviderMetadata()

	// Also include providers that aren't registered but have metadata
	allMetadata := provider.ListProviderMetadata()

	// Merge: prefer registered providers, add static metadata for unregistered
	metadataMap := make(map[string]provider.ProviderMetadata)
	for _, m := range allMetadata {
		metadataMap[m.Name] = m
	}
	for _, m := range metadata {
		metadataMap[m.Name] = m
	}

	var response []ProviderResponse
	for _, m := range metadataMap {
		response = append(response, ProviderResponse{
			Name:         m.Name,
			DisplayName:  m.DisplayName,
			Description:  m.Description,
			IsConfigured: m.IsConfigured,
			DefaultModel: m.DefaultModel,
			ConfigKeys:   m.ConfigKeys,
			ModelDocLink: m.ModelDocLink,
		})
	}

	c.JSON(consts.StatusOK, response)
}

// ModelResponse represents a model in the response
type ModelResponse struct {
	Name            string  `json:"name"`
	DisplayName     string  `json:"display_name,omitempty"`
	ContextLength   int     `json:"context_length,omitempty"`
	InputCostPer1M  float64 `json:"input_cost_per_1m,omitempty"`
	OutputCostPer1M float64 `json:"output_cost_per_1m,omitempty"`
	SupportsTools   bool    `json:"supports_tools"`
	SupportsVision  bool    `json:"supports_vision"`
}

// GetProviderModels handles GET /config/providers/:name/models
func (r *ConfigRoutes) GetProviderModels(ctx context.Context, c *app.RequestContext) {
	providerName := c.Param("name")

	prov, ok := r.agentManager.GetProvider(providerName)
	if !ok {
		// Try to get static metadata
		meta := provider.GetProviderMetadata(providerName)
		if meta == nil {
			c.JSON(consts.StatusNotFound, map[string]string{
				"message": "Provider not found: " + providerName,
			})
			return
		}

		var models []ModelResponse
		for _, m := range meta.KnownModels {
			models = append(models, ModelResponse{
				Name:            m.Name,
				DisplayName:     m.DisplayName,
				ContextLength:   m.ContextLength,
				InputCostPer1M:  m.InputCostPer1M,
				OutputCostPer1M: m.OutputCostPer1M,
				SupportsTools:   m.SupportsTools,
				SupportsVision:  m.SupportsVision,
			})
		}
		c.JSON(consts.StatusOK, models)
		return
	}

	var models []ModelResponse
	for _, m := range prov.GetModels() {
		models = append(models, ModelResponse{
			Name:            m.Name,
			DisplayName:     m.DisplayName,
			ContextLength:   m.ContextLength,
			InputCostPer1M:  m.InputCostPer1M,
			OutputCostPer1M: m.OutputCostPer1M,
			SupportsTools:   m.SupportsTools,
			SupportsVision:  m.SupportsVision,
		})
	}

	c.JSON(consts.StatusOK, models)
}

// SetProviderRequest is the request body for setting a provider
type SetProviderRequest struct {
	Provider string  `json:"provider"`
	Model    *string `json:"model,omitempty"`
}

// SetProvider handles POST /config/set_provider
func (r *ConfigRoutes) SetProvider(ctx context.Context, c *app.RequestContext) {
	var req SetProviderRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(consts.StatusBadRequest, map[string]string{
			"message": "Invalid request body: " + err.Error(),
		})
		return
	}

	if req.Provider == "" {
		c.JSON(consts.StatusBadRequest, map[string]string{
			"message": "provider is required",
		})
		return
	}

	prov, ok := r.agentManager.GetProvider(req.Provider)
	if !ok {
		c.JSON(consts.StatusNotFound, map[string]string{
			"message": "Provider not found: " + req.Provider,
		})
		return
	}

	if !prov.IsConfigured() {
		c.JSON(consts.StatusBadRequest, map[string]string{
			"message": "Provider is not configured. Set the required environment variables.",
		})
		return
	}

	model := prov.GetDefaultModel()
	if req.Model != nil {
		model = *req.Model
	}

	c.JSON(consts.StatusOK, map[string]interface{}{
		"provider": req.Provider,
		"model":    model,
		"message":  "Provider set successfully",
	})
}

// CheckProviderRequest is the request body for checking a provider
type CheckProviderRequest struct {
	Provider string `json:"provider"`
}

// CheckProvider handles POST /config/check_provider
func (r *ConfigRoutes) CheckProvider(ctx context.Context, c *app.RequestContext) {
	var req CheckProviderRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(consts.StatusBadRequest, map[string]string{
			"message": "Invalid request body: " + err.Error(),
		})
		return
	}

	if req.Provider == "" {
		c.JSON(consts.StatusBadRequest, map[string]string{
			"message": "provider is required",
		})
		return
	}

	prov, ok := r.agentManager.GetProvider(req.Provider)
	if !ok {
		c.JSON(consts.StatusNotFound, map[string]string{
			"message": "Provider not found: " + req.Provider,
		})
		return
	}

	c.JSON(consts.StatusOK, map[string]interface{}{
		"provider":      req.Provider,
		"is_configured": prov.IsConfigured(),
		"default_model": prov.GetDefaultModel(),
	})
}

// DetectProviderRequest is the request body for detecting a provider
type DetectProviderRequest struct {
	APIKey string `json:"api_key"`
}

// DetectProvider handles POST /config/detect-provider
func (r *ConfigRoutes) DetectProvider(ctx context.Context, c *app.RequestContext) {
	var req DetectProviderRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(consts.StatusBadRequest, map[string]string{
			"message": "Invalid request body: " + err.Error(),
		})
		return
	}

	if req.APIKey == "" {
		c.JSON(consts.StatusBadRequest, map[string]string{
			"message": "api_key is required",
		})
		return
	}

	// Simple detection based on key prefix
	var detected string
	switch {
	case len(req.APIKey) > 3 && req.APIKey[:3] == "sk-":
		if len(req.APIKey) > 7 && req.APIKey[:7] == "sk-ant-" {
			detected = "anthropic"
		} else {
			detected = "openai"
		}
	default:
		c.JSON(consts.StatusOK, map[string]interface{}{
			"detected": false,
			"message":  "Could not detect provider from API key format",
		})
		return
	}

	meta := provider.GetProviderMetadata(detected)
	if meta == nil {
		c.JSON(consts.StatusOK, map[string]interface{}{
			"detected": false,
			"message":  "Provider not recognized",
		})
		return
	}

	c.JSON(consts.StatusOK, map[string]interface{}{
		"detected":      true,
		"provider":      meta.Name,
		"display_name":  meta.DisplayName,
		"default_model": meta.DefaultModel,
	})
}

// GetExtensions handles GET /config/extensions
func (r *ConfigRoutes) GetExtensions(ctx context.Context, c *app.RequestContext) {
	// TODO: Implement in Phase 7
	c.JSON(consts.StatusOK, []interface{}{})
}

// AddExtension handles POST /config/extensions
func (r *ConfigRoutes) AddExtension(ctx context.Context, c *app.RequestContext) {
	// TODO: Implement in Phase 7
	c.JSON(consts.StatusNotImplemented, map[string]string{
		"message": "Not implemented - Phase 7",
	})
}

// RemoveExtension handles DELETE /config/extensions/:name
func (r *ConfigRoutes) RemoveExtension(ctx context.Context, c *app.RequestContext) {
	// TODO: Implement in Phase 7
	c.JSON(consts.StatusNotImplemented, map[string]string{
		"message": "Not implemented - Phase 7",
	})
}

// UpdatePermissions handles POST /config/permissions
func (r *ConfigRoutes) UpdatePermissions(ctx context.Context, c *app.RequestContext) {
	// TODO: Implement in Phase 7
	c.JSON(consts.StatusNotImplemented, map[string]string{
		"message": "Not implemented - Phase 7",
	})
}

// GetSlashCommands handles GET /config/slash_commands
func (r *ConfigRoutes) GetSlashCommands(ctx context.Context, c *app.RequestContext) {
	// TODO: Implement in Phase 7
	c.JSON(consts.StatusOK, []interface{}{})
}
