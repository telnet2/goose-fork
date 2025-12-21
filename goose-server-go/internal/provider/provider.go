package provider

import (
	"context"

	"github.com/block/goose-server-go/internal/models"
)

// Provider defines the interface for LLM providers
type Provider interface {
	// Name returns the provider name (e.g., "openai", "anthropic")
	Name() string

	// DisplayName returns a human-readable name
	DisplayName() string

	// Chat sends messages and returns a stream of events
	Chat(ctx context.Context, messages []models.Message, options ChatOptions) (<-chan models.MessageEvent, error)

	// Generate sends messages and returns a single response (non-streaming)
	Generate(ctx context.Context, messages []models.Message, options ChatOptions) (*models.Message, *models.TokenState, error)

	// GetModels returns available models for this provider
	GetModels() []ModelInfo

	// GetDefaultModel returns the default model name
	GetDefaultModel() string

	// IsConfigured returns whether the provider is properly configured
	IsConfigured() bool

	// SupportsStreaming returns whether the provider supports streaming
	SupportsStreaming() bool

	// SupportsTools returns whether the provider supports tool calling
	SupportsTools() bool
}

// ChatOptions contains options for chat completions
type ChatOptions struct {
	Model       string
	MaxTokens   int
	Temperature float64
	TopP        float64
	Tools       []models.ToolInfo
	System      string
}

// ModelInfo describes an available model
type ModelInfo struct {
	Name           string  `json:"name"`
	DisplayName    string  `json:"display_name,omitempty"`
	ContextLength  int     `json:"context_length,omitempty"`
	InputCostPer1M float64 `json:"input_cost_per_1m,omitempty"`
	OutputCostPer1M float64 `json:"output_cost_per_1m,omitempty"`
	SupportsTools  bool    `json:"supports_tools"`
	SupportsVision bool    `json:"supports_vision"`
}

// ProviderMetadata contains static metadata about a provider
type ProviderMetadata struct {
	Name          string      `json:"name"`
	DisplayName   string      `json:"display_name"`
	Description   string      `json:"description"`
	DefaultModel  string      `json:"default_model"`
	KnownModels   []ModelInfo `json:"known_models"`
	ConfigKeys    []ConfigKey `json:"config_keys"`
	IsConfigured  bool        `json:"is_configured"`
	ModelDocLink  string      `json:"model_doc_link,omitempty"`
}

// ConfigKey describes a configuration key for a provider
type ConfigKey struct {
	Name        string  `json:"name"`
	Required    bool    `json:"required"`
	Secret      bool    `json:"secret"`
	Default     *string `json:"default,omitempty"`
	Description string  `json:"description,omitempty"`
}

// ProviderError represents a provider-specific error
type ProviderError struct {
	Code    ProviderErrorCode
	Message string
	Err     error
}

// ProviderErrorCode represents the type of provider error
type ProviderErrorCode string

const (
	ErrAuthentication      ProviderErrorCode = "AUTHENTICATION"
	ErrRateLimitExceeded   ProviderErrorCode = "RATE_LIMIT_EXCEEDED"
	ErrContextLengthExceeded ProviderErrorCode = "CONTEXT_LENGTH_EXCEEDED"
	ErrServerError         ProviderErrorCode = "SERVER_ERROR"
	ErrRequestFailed       ProviderErrorCode = "REQUEST_FAILED"
	ErrNotConfigured       ProviderErrorCode = "NOT_CONFIGURED"
	ErrModelNotFound       ProviderErrorCode = "MODEL_NOT_FOUND"
)

func (e *ProviderError) Error() string {
	if e.Err != nil {
		return string(e.Code) + ": " + e.Message + ": " + e.Err.Error()
	}
	return string(e.Code) + ": " + e.Message
}

func (e *ProviderError) Unwrap() error {
	return e.Err
}

// NewProviderError creates a new provider error
func NewProviderError(code ProviderErrorCode, message string, err error) *ProviderError {
	return &ProviderError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}
