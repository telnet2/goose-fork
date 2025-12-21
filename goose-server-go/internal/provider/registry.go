package provider

import (
	"context"
	"sync"

	"github.com/rs/zerolog/log"
)

// Registry manages provider instances
type Registry struct {
	providers map[string]Provider
	factory   *ProviderFactory
	mu        sync.RWMutex
}

// NewRegistry creates a new provider registry
func NewRegistry() *Registry {
	return &Registry{
		providers: make(map[string]Provider),
		factory:   NewProviderFactory(),
	}
}

// Initialize initializes all available providers
func (r *Registry) Initialize(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Initialize OpenAI
	if provider, err := r.factory.CreateOpenAI(ctx); err == nil && provider != nil {
		r.providers["openai"] = provider
		log.Info().Bool("configured", provider.IsConfigured()).Msg("OpenAI provider initialized")
	} else if err != nil {
		log.Warn().Err(err).Msg("Failed to initialize OpenAI provider")
	}

	// Initialize Anthropic
	if provider, err := r.factory.CreateAnthropic(ctx); err == nil && provider != nil {
		r.providers["anthropic"] = provider
		log.Info().Bool("configured", provider.IsConfigured()).Msg("Anthropic provider initialized")
	} else if err != nil {
		log.Warn().Err(err).Msg("Failed to initialize Anthropic provider")
	}

	// Initialize Azure OpenAI
	if provider, err := r.factory.CreateAzureOpenAI(ctx); err == nil && provider != nil {
		r.providers["azure_openai"] = provider
		log.Info().Bool("configured", provider.IsConfigured()).Msg("Azure OpenAI provider initialized")
	} else if err != nil {
		log.Warn().Err(err).Msg("Failed to initialize Azure OpenAI provider")
	}

	return nil
}

// Get returns a provider by name
func (r *Registry) Get(name string) (Provider, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.providers[name]
	return p, ok
}

// GetConfigured returns a configured provider by name, or returns an error
func (r *Registry) GetConfigured(name string) (Provider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	p, ok := r.providers[name]
	if !ok {
		return nil, NewProviderError(ErrModelNotFound, "provider not found: "+name, nil)
	}
	if !p.IsConfigured() {
		return nil, NewProviderError(ErrNotConfigured, "provider not configured: "+name, nil)
	}
	return p, nil
}

// GetDefault returns the default configured provider
func (r *Registry) GetDefault() (Provider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Priority order: anthropic, openai, azure_openai
	priorities := []string{"anthropic", "openai", "azure_openai"}
	for _, name := range priorities {
		if p, ok := r.providers[name]; ok && p.IsConfigured() {
			return p, nil
		}
	}

	return nil, NewProviderError(ErrNotConfigured, "no provider is configured", nil)
}

// List returns all registered providers
func (r *Registry) List() []Provider {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]Provider, 0, len(r.providers))
	for _, p := range r.providers {
		result = append(result, p)
	}
	return result
}

// ListConfigured returns all configured providers
func (r *Registry) ListConfigured() []Provider {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []Provider
	for _, p := range r.providers {
		if p.IsConfigured() {
			result = append(result, p)
		}
	}
	return result
}

// Register adds a provider to the registry
func (r *Registry) Register(name string, provider Provider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers[name] = provider
}

// GetMetadata returns metadata for all providers
func (r *Registry) GetMetadata() []ProviderMetadata {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []ProviderMetadata
	for name, p := range r.providers {
		meta := ProviderMetadata{
			Name:         name,
			DisplayName:  p.DisplayName(),
			DefaultModel: p.GetDefaultModel(),
			IsConfigured: p.IsConfigured(),
		}

		// Get known models
		for _, m := range p.GetModels() {
			meta.KnownModels = append(meta.KnownModels, m)
		}

		// Get config keys from static metadata
		if staticMeta := GetProviderMetadata(name); staticMeta != nil {
			meta.Description = staticMeta.Description
			meta.ConfigKeys = staticMeta.ConfigKeys
			meta.ModelDocLink = staticMeta.ModelDocLink
		}

		result = append(result, meta)
	}
	return result
}

// Reload reinitializes a specific provider
func (r *Registry) Reload(ctx context.Context, name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	var provider Provider
	var err error

	switch name {
	case "openai":
		provider, err = r.factory.CreateOpenAI(ctx)
	case "anthropic":
		provider, err = r.factory.CreateAnthropic(ctx)
	case "azure_openai":
		provider, err = r.factory.CreateAzureOpenAI(ctx)
	default:
		return NewProviderError(ErrModelNotFound, "unknown provider: "+name, nil)
	}

	if err != nil {
		return err
	}
	if provider != nil {
		r.providers[name] = provider
	}
	return nil
}

// ReloadAll reinitializes all providers
func (r *Registry) ReloadAll(ctx context.Context) error {
	return r.Initialize(ctx)
}
