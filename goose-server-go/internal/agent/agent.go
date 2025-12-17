package agent

import (
	"context"
	"fmt"
	"sync"

	"github.com/block/goose-server-go/internal/models"
	"github.com/block/goose-server-go/internal/session"
)

// Agent represents an active agent session that can communicate with an LLM provider
type Agent struct {
	ID        string
	SessionID string
	Provider  Provider
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
}

// Provider defines the interface for LLM providers
type Provider interface {
	// Name returns the provider name
	Name() string

	// Chat sends messages and returns a stream of events
	Chat(ctx context.Context, messages []models.Message, options ChatOptions) (<-chan models.MessageEvent, error)

	// GetModels returns available models for this provider
	GetModels() []string

	// IsConfigured returns whether the provider is properly configured
	IsConfigured() bool
}

// ChatOptions contains options for a chat request
type ChatOptions struct {
	Model       string
	MaxTokens   int
	Temperature float64
	Tools       []models.ToolInfo
}

// Manager manages active agents
type Manager struct {
	agents         sync.Map // map[string]*Agent (sessionID -> Agent)
	sessionManager *session.Manager
	providers      map[string]Provider
	mu             sync.RWMutex
}

// NewManager creates a new agent manager
func NewManager(sessionManager *session.Manager) *Manager {
	return &Manager{
		sessionManager: sessionManager,
		providers:      make(map[string]Provider),
	}
}

// RegisterProvider registers an LLM provider
func (m *Manager) RegisterProvider(provider Provider) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.providers[provider.Name()] = provider
}

// GetProvider returns a provider by name
func (m *Manager) GetProvider(name string) (Provider, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	p, ok := m.providers[name]
	return p, ok
}

// ListProviders returns all registered providers
func (m *Manager) ListProviders() []Provider {
	m.mu.RLock()
	defer m.mu.RUnlock()

	providers := make([]Provider, 0, len(m.providers))
	for _, p := range m.providers {
		providers = append(providers, p)
	}
	return providers
}

// Start starts a new agent for a session
func (m *Manager) Start(ctx context.Context, config *AgentConfig) (*Agent, error) {
	// Create session
	sess, err := m.sessionManager.Create(config.WorkingDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	// Get provider
	provider, ok := m.GetProvider(config.ProviderName)
	if !ok {
		// Use mock provider if no real provider configured
		provider = NewMockProvider()
	}

	agent := &Agent{
		ID:        sess.ID,
		SessionID: sess.ID,
		Provider:  provider,
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
	providerName := "mock"
	if sess.ProviderName != nil {
		providerName = *sess.ProviderName
	}

	provider, ok := m.GetProvider(providerName)
	if !ok {
		provider = NewMockProvider()
	}

	config := &AgentConfig{
		WorkingDir:   sess.WorkingDir,
		ProviderName: providerName,
		Recipe:       sess.Recipe,
	}

	agent := &Agent{
		ID:        sessionID,
		SessionID: sessionID,
		Provider:  provider,
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

// Chat sends messages to the agent and returns a stream of events
func (a *Agent) Chat(ctx context.Context, messages []models.Message) (<-chan models.MessageEvent, error) {
	a.mu.RLock()
	if !a.running {
		a.mu.RUnlock()
		return nil, fmt.Errorf("agent is not running")
	}
	a.mu.RUnlock()

	options := ChatOptions{
		Model: a.Config.ModelName,
		Tools: a.Tools,
	}

	return a.Provider.Chat(ctx, messages, options)
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
