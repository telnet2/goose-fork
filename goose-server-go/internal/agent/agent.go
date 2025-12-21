package agent

import (
	"context"
	"fmt"
	"sync"

	"github.com/block/goose-server-go/internal/models"
	"github.com/block/goose-server-go/internal/provider"
	"github.com/block/goose-server-go/internal/session"
)

// Agent represents an active agent session that can communicate with an LLM provider
type Agent struct {
	ID        string
	SessionID string
	Provider  provider.Provider
	Config    *AgentConfig
	Tools     []models.ToolInfo

	mu       sync.RWMutex
	running  bool
	stopChan chan struct{}
}

// AgentConfig holds agent configuration
type AgentConfig struct {
	WorkingDir     string
	ProviderName   string
	ModelName      string
	Recipe         *models.Recipe
	ExtensionNames []string
	SystemPrompt   string
}

// Manager manages active agents
type Manager struct {
	agents           sync.Map // map[string]*Agent (sessionID -> Agent)
	sessionManager   *session.Manager
	providerRegistry *provider.Registry
	mockProvider     provider.Provider
	mu               sync.RWMutex
}

// NewManager creates a new agent manager
func NewManager(sessionManager *session.Manager, providerRegistry *provider.Registry) *Manager {
	return &Manager{
		sessionManager:   sessionManager,
		providerRegistry: providerRegistry,
		mockProvider:     NewMockProviderAdapter(),
	}
}

// GetProvider returns a provider by name
func (m *Manager) GetProvider(name string) (provider.Provider, bool) {
	if m.providerRegistry == nil {
		return m.mockProvider, true
	}
	return m.providerRegistry.Get(name)
}

// GetConfiguredProvider returns a configured provider by name
func (m *Manager) GetConfiguredProvider(name string) (provider.Provider, error) {
	if m.providerRegistry == nil {
		return m.mockProvider, nil
	}
	return m.providerRegistry.GetConfigured(name)
}

// GetDefaultProvider returns the default configured provider
func (m *Manager) GetDefaultProvider() (provider.Provider, error) {
	if m.providerRegistry == nil {
		return m.mockProvider, nil
	}
	return m.providerRegistry.GetDefault()
}

// ListProviders returns all registered providers
func (m *Manager) ListProviders() []provider.Provider {
	if m.providerRegistry == nil {
		return []provider.Provider{m.mockProvider}
	}
	return m.providerRegistry.List()
}

// ListConfiguredProviders returns all configured providers
func (m *Manager) ListConfiguredProviders() []provider.Provider {
	if m.providerRegistry == nil {
		return []provider.Provider{}
	}
	return m.providerRegistry.ListConfigured()
}

// GetProviderMetadata returns metadata for all providers
func (m *Manager) GetProviderMetadata() []provider.ProviderMetadata {
	if m.providerRegistry == nil {
		return []provider.ProviderMetadata{}
	}
	return m.providerRegistry.GetMetadata()
}

// Start starts a new agent for a session
func (m *Manager) Start(ctx context.Context, config *AgentConfig) (*Agent, error) {
	// Create session
	sess, err := m.sessionManager.Create(config.WorkingDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	// Get provider
	var prov provider.Provider
	if config.ProviderName != "" {
		prov, _ = m.GetProvider(config.ProviderName)
	}
	if prov == nil || !prov.IsConfigured() {
		// Try to get default configured provider
		if defaultProv, err := m.GetDefaultProvider(); err == nil && defaultProv.IsConfigured() {
			prov = defaultProv
			config.ProviderName = prov.Name()
		} else {
			// Fall back to mock provider
			prov = m.mockProvider
			config.ProviderName = "mock"
		}
	}

	// Set default model if not specified
	if config.ModelName == "" {
		config.ModelName = prov.GetDefaultModel()
	}

	agent := &Agent{
		ID:        sess.ID,
		SessionID: sess.ID,
		Provider:  prov,
		Config:    config,
		Tools:     []models.ToolInfo{},
		running:   true,
		stopChan:  make(chan struct{}),
	}

	// Store agent
	m.agents.Store(sess.ID, agent)

	// Update session with provider info
	sess.ProviderName = &config.ProviderName
	if config.Recipe != nil {
		sess.Recipe = config.Recipe
	}
	if err := m.sessionManager.Update(sess); err != nil {
		return nil, fmt.Errorf("failed to update session: %w", err)
	}

	return agent, nil
}

// Resume resumes an existing agent session
func (m *Manager) Resume(ctx context.Context, sessionID string, loadModelAndExtensions bool) (*Agent, error) {
	// Check if agent is already running
	if existing, ok := m.agents.Load(sessionID); ok {
		return existing.(*Agent), nil
	}

	// Get session
	sess, err := m.sessionManager.Get(sessionID, true)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}
	if sess == nil {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}

	// Determine provider
	providerName := ""
	if sess.ProviderName != nil {
		providerName = *sess.ProviderName
	}

	var prov provider.Provider
	if providerName != "" {
		prov, _ = m.GetProvider(providerName)
	}
	if prov == nil || !prov.IsConfigured() {
		// Try default provider
		if defaultProv, err := m.GetDefaultProvider(); err == nil && defaultProv.IsConfigured() {
			prov = defaultProv
			providerName = prov.Name()
		} else {
			prov = m.mockProvider
			providerName = "mock"
		}
	}

	config := &AgentConfig{
		WorkingDir:   sess.WorkingDir,
		ProviderName: providerName,
		ModelName:    prov.GetDefaultModel(),
		Recipe:       sess.Recipe,
	}

	agent := &Agent{
		ID:        sessionID,
		SessionID: sessionID,
		Provider:  prov,
		Config:    config,
		Tools:     []models.ToolInfo{},
		running:   true,
		stopChan:  make(chan struct{}),
	}

	// Store agent
	m.agents.Store(sessionID, agent)

	return agent, nil
}

// Get returns an agent by session ID
func (m *Manager) Get(sessionID string) (*Agent, bool) {
	if agent, ok := m.agents.Load(sessionID); ok {
		return agent.(*Agent), true
	}
	return nil, false
}

// Stop stops an agent
func (m *Manager) Stop(sessionID string) error {
	if agentVal, ok := m.agents.Load(sessionID); ok {
		agent := agentVal.(*Agent)
		agent.mu.Lock()
		agent.running = false
		close(agent.stopChan)
		agent.mu.Unlock()
		m.agents.Delete(sessionID)
	}
	return nil
}

// UpdateProvider updates the provider for an agent
func (m *Manager) UpdateProvider(sessionID string, providerName string, modelName string) error {
	agentVal, ok := m.agents.Load(sessionID)
	if !ok {
		return fmt.Errorf("agent not found: %s", sessionID)
	}

	agent := agentVal.(*Agent)
	prov, ok := m.GetProvider(providerName)
	if !ok {
		return fmt.Errorf("provider not found: %s", providerName)
	}

	agent.mu.Lock()
	agent.Provider = prov
	agent.Config.ProviderName = providerName
	if modelName != "" {
		agent.Config.ModelName = modelName
	} else {
		agent.Config.ModelName = prov.GetDefaultModel()
	}
	agent.mu.Unlock()

	return nil
}

// Chat sends messages to the agent and returns a stream of events
func (a *Agent) Chat(ctx context.Context, messages []models.Message) (<-chan models.MessageEvent, error) {
	a.mu.RLock()
	if !a.running {
		a.mu.RUnlock()
		return nil, fmt.Errorf("agent is not running")
	}
	prov := a.Provider
	config := a.Config
	tools := a.Tools
	a.mu.RUnlock()

	options := provider.ChatOptions{
		Model:  config.ModelName,
		Tools:  tools,
		System: config.SystemPrompt,
	}

	return prov.Chat(ctx, messages, options)
}

// AddTool adds a tool to the agent
func (a *Agent) AddTool(tool models.ToolInfo) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.Tools = append(a.Tools, tool)
}

// GetTools returns all available tools
func (a *Agent) GetTools() []models.ToolInfo {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.Tools
}

// IsRunning returns whether the agent is running
func (a *Agent) IsRunning() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.running
}

// SetSystemPrompt updates the system prompt for the agent
func (a *Agent) SetSystemPrompt(prompt string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.Config.SystemPrompt = prompt
}

// GetSystemPrompt returns the current system prompt
func (a *Agent) GetSystemPrompt() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.Config.SystemPrompt
}
