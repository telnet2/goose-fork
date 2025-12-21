package provider

import (
	"context"
	"os"

	"github.com/cloudwego/eino-ext/components/model/claude"
	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/rs/zerolog/log"
)

// ProviderFactory creates provider instances
type ProviderFactory struct{}

// NewProviderFactory creates a new provider factory
func NewProviderFactory() *ProviderFactory {
	return &ProviderFactory{}
}

// CreateOpenAI creates an OpenAI provider
func (f *ProviderFactory) CreateOpenAI(ctx context.Context) (Provider, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Debug().Msg("OPENAI_API_KEY not set, OpenAI provider not configured")
		return &EinoAdapter{
			name:          "openai",
			displayName:   "OpenAI",
			models:        openAIModels,
			defaultModel:  "gpt-4o",
			isConfigured:  false,
			supportsTools: true,
		}, nil
	}

	baseURL := os.Getenv("OPENAI_BASE_URL")
	model := os.Getenv("OPENAI_MODEL")
	if model == "" {
		model = "gpt-4o"
	}

	cfg := &openai.ChatModelConfig{
		APIKey: apiKey,
		Model:  model,
	}
	if baseURL != "" {
		cfg.BaseURL = baseURL
	}

	chatModel, err := openai.NewChatModel(ctx, cfg)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create OpenAI chat model")
		return nil, NewProviderError(ErrRequestFailed, "failed to create OpenAI provider", err)
	}

	return NewEinoAdapter(EinoAdapterConfig{
		Name:              "openai",
		DisplayName:       "OpenAI",
		ChatModel:         chatModel,
		Models:            openAIModels,
		DefaultModel:      model,
		SupportsStreaming: true,
		SupportsTools:     true,
	}), nil
}

// CreateAnthropic creates an Anthropic/Claude provider
func (f *ProviderFactory) CreateAnthropic(ctx context.Context) (Provider, error) {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		log.Debug().Msg("ANTHROPIC_API_KEY not set, Anthropic provider not configured")
		return &EinoAdapter{
			name:          "anthropic",
			displayName:   "Anthropic",
			models:        anthropicModels,
			defaultModel:  "claude-sonnet-4-5-20250514",
			isConfigured:  false,
			supportsTools: true,
		}, nil
	}

	model := os.Getenv("ANTHROPIC_MODEL")
	if model == "" {
		model = "claude-sonnet-4-5-20250514"
	}

	maxTokens := 8192
	chatModel, err := claude.NewChatModel(ctx, &claude.Config{
		APIKey:    apiKey,
		Model:     model,
		MaxTokens: maxTokens,
	})
	if err != nil {
		log.Error().Err(err).Msg("Failed to create Anthropic chat model")
		return nil, NewProviderError(ErrRequestFailed, "failed to create Anthropic provider", err)
	}

	return NewEinoAdapter(EinoAdapterConfig{
		Name:              "anthropic",
		DisplayName:       "Anthropic",
		ChatModel:         chatModel,
		Models:            anthropicModels,
		DefaultModel:      model,
		SupportsStreaming: true,
		SupportsTools:     true,
	}), nil
}

// CreateAzureOpenAI creates an Azure OpenAI provider
func (f *ProviderFactory) CreateAzureOpenAI(ctx context.Context) (Provider, error) {
	apiKey := os.Getenv("AZURE_OPENAI_API_KEY")
	endpoint := os.Getenv("AZURE_OPENAI_ENDPOINT")

	if apiKey == "" || endpoint == "" {
		log.Debug().Msg("Azure OpenAI credentials not set")
		return &EinoAdapter{
			name:          "azure_openai",
			displayName:   "Azure OpenAI",
			models:        azureModels,
			defaultModel:  "gpt-4o",
			isConfigured:  false,
			supportsTools: true,
		}, nil
	}

	model := os.Getenv("AZURE_OPENAI_MODEL")
	if model == "" {
		model = "gpt-4o"
	}

	apiVersion := os.Getenv("AZURE_OPENAI_API_VERSION")
	if apiVersion == "" {
		apiVersion = "2024-06-01"
	}

	chatModel, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		APIKey:     apiKey,
		BaseURL:    endpoint,
		Model:      model,
		ByAzure:    true,
		APIVersion: apiVersion,
	})
	if err != nil {
		log.Error().Err(err).Msg("Failed to create Azure OpenAI chat model")
		return nil, NewProviderError(ErrRequestFailed, "failed to create Azure OpenAI provider", err)
	}

	return NewEinoAdapter(EinoAdapterConfig{
		Name:              "azure_openai",
		DisplayName:       "Azure OpenAI",
		ChatModel:         chatModel,
		Models:            azureModels,
		DefaultModel:      model,
		SupportsStreaming: true,
		SupportsTools:     true,
	}), nil
}

// Known models for each provider
var openAIModels = []ModelInfo{
	{Name: "gpt-4o", DisplayName: "GPT-4o", ContextLength: 128000, SupportsTools: true, SupportsVision: true, InputCostPer1M: 2.50, OutputCostPer1M: 10.00},
	{Name: "gpt-4o-mini", DisplayName: "GPT-4o Mini", ContextLength: 128000, SupportsTools: true, SupportsVision: true, InputCostPer1M: 0.15, OutputCostPer1M: 0.60},
	{Name: "gpt-4-turbo", DisplayName: "GPT-4 Turbo", ContextLength: 128000, SupportsTools: true, SupportsVision: true, InputCostPer1M: 10.00, OutputCostPer1M: 30.00},
	{Name: "gpt-4", DisplayName: "GPT-4", ContextLength: 8192, SupportsTools: true, SupportsVision: false, InputCostPer1M: 30.00, OutputCostPer1M: 60.00},
	{Name: "gpt-3.5-turbo", DisplayName: "GPT-3.5 Turbo", ContextLength: 16385, SupportsTools: true, SupportsVision: false, InputCostPer1M: 0.50, OutputCostPer1M: 1.50},
	{Name: "o1", DisplayName: "o1", ContextLength: 200000, SupportsTools: true, SupportsVision: true, InputCostPer1M: 15.00, OutputCostPer1M: 60.00},
	{Name: "o1-mini", DisplayName: "o1 Mini", ContextLength: 128000, SupportsTools: true, SupportsVision: true, InputCostPer1M: 3.00, OutputCostPer1M: 12.00},
	{Name: "o3-mini", DisplayName: "o3 Mini", ContextLength: 200000, SupportsTools: true, SupportsVision: true, InputCostPer1M: 1.10, OutputCostPer1M: 4.40},
}

var anthropicModels = []ModelInfo{
	{Name: "claude-sonnet-4-5-20250514", DisplayName: "Claude Sonnet 4.5", ContextLength: 200000, SupportsTools: true, SupportsVision: true, InputCostPer1M: 3.00, OutputCostPer1M: 15.00},
	{Name: "claude-opus-4-5-20250514", DisplayName: "Claude Opus 4.5", ContextLength: 200000, SupportsTools: true, SupportsVision: true, InputCostPer1M: 15.00, OutputCostPer1M: 75.00},
	{Name: "claude-3-5-sonnet-20241022", DisplayName: "Claude 3.5 Sonnet", ContextLength: 200000, SupportsTools: true, SupportsVision: true, InputCostPer1M: 3.00, OutputCostPer1M: 15.00},
	{Name: "claude-3-5-haiku-20241022", DisplayName: "Claude 3.5 Haiku", ContextLength: 200000, SupportsTools: true, SupportsVision: true, InputCostPer1M: 0.80, OutputCostPer1M: 4.00},
	{Name: "claude-3-opus-20240229", DisplayName: "Claude 3 Opus", ContextLength: 200000, SupportsTools: true, SupportsVision: true, InputCostPer1M: 15.00, OutputCostPer1M: 75.00},
	{Name: "claude-3-sonnet-20240229", DisplayName: "Claude 3 Sonnet", ContextLength: 200000, SupportsTools: true, SupportsVision: true, InputCostPer1M: 3.00, OutputCostPer1M: 15.00},
	{Name: "claude-3-haiku-20240307", DisplayName: "Claude 3 Haiku", ContextLength: 200000, SupportsTools: true, SupportsVision: true, InputCostPer1M: 0.25, OutputCostPer1M: 1.25},
}

var azureModels = []ModelInfo{
	{Name: "gpt-4o", DisplayName: "GPT-4o", ContextLength: 128000, SupportsTools: true, SupportsVision: true},
	{Name: "gpt-4o-mini", DisplayName: "GPT-4o Mini", ContextLength: 128000, SupportsTools: true, SupportsVision: true},
	{Name: "gpt-4-turbo", DisplayName: "GPT-4 Turbo", ContextLength: 128000, SupportsTools: true, SupportsVision: true},
	{Name: "gpt-4", DisplayName: "GPT-4", ContextLength: 8192, SupportsTools: true, SupportsVision: false},
	{Name: "gpt-35-turbo", DisplayName: "GPT-3.5 Turbo", ContextLength: 16385, SupportsTools: true, SupportsVision: false},
}

// GetProviderMetadata returns metadata for a provider without creating it
func GetProviderMetadata(name string) *ProviderMetadata {
	switch name {
	case "openai":
		return &ProviderMetadata{
			Name:         "openai",
			DisplayName:  "OpenAI",
			Description:  "OpenAI GPT models including GPT-4o, o1, and o3",
			DefaultModel: "gpt-4o",
			KnownModels:  openAIModels,
			ModelDocLink: "https://platform.openai.com/docs/models",
			ConfigKeys: []ConfigKey{
				{Name: "OPENAI_API_KEY", Required: true, Secret: true, Description: "OpenAI API key"},
				{Name: "OPENAI_BASE_URL", Required: false, Secret: false, Description: "Custom API base URL"},
				{Name: "OPENAI_MODEL", Required: false, Secret: false, Description: "Default model to use"},
			},
			IsConfigured: os.Getenv("OPENAI_API_KEY") != "",
		}
	case "anthropic":
		return &ProviderMetadata{
			Name:         "anthropic",
			DisplayName:  "Anthropic",
			Description:  "Claude models from Anthropic",
			DefaultModel: "claude-sonnet-4-5-20250514",
			KnownModels:  anthropicModels,
			ModelDocLink: "https://docs.anthropic.com/en/docs/about-claude/models",
			ConfigKeys: []ConfigKey{
				{Name: "ANTHROPIC_API_KEY", Required: true, Secret: true, Description: "Anthropic API key"},
				{Name: "ANTHROPIC_MODEL", Required: false, Secret: false, Description: "Default model to use"},
			},
			IsConfigured: os.Getenv("ANTHROPIC_API_KEY") != "",
		}
	case "azure_openai":
		return &ProviderMetadata{
			Name:         "azure_openai",
			DisplayName:  "Azure OpenAI",
			Description:  "OpenAI models via Azure",
			DefaultModel: "gpt-4o",
			KnownModels:  azureModels,
			ModelDocLink: "https://learn.microsoft.com/en-us/azure/ai-services/openai/",
			ConfigKeys: []ConfigKey{
				{Name: "AZURE_OPENAI_API_KEY", Required: true, Secret: true, Description: "Azure OpenAI API key"},
				{Name: "AZURE_OPENAI_ENDPOINT", Required: true, Secret: false, Description: "Azure OpenAI endpoint URL"},
				{Name: "AZURE_OPENAI_MODEL", Required: false, Secret: false, Description: "Deployment name"},
				{Name: "AZURE_OPENAI_API_VERSION", Required: false, Secret: false, Description: "API version"},
			},
			IsConfigured: os.Getenv("AZURE_OPENAI_API_KEY") != "" && os.Getenv("AZURE_OPENAI_ENDPOINT") != "",
		}
	default:
		return nil
	}
}

// ListProviderMetadata returns metadata for all known providers
func ListProviderMetadata() []ProviderMetadata {
	providers := []string{"openai", "anthropic", "azure_openai"}
	var result []ProviderMetadata
	for _, name := range providers {
		if meta := GetProviderMetadata(name); meta != nil {
			result = append(result, *meta)
		}
	}
	return result
}
