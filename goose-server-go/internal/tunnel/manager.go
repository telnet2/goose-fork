package tunnel

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/gofrs/flock"
)

// Sentinel errors
var (
	ErrTunnelAlreadyRunning = errors.New("tunnel is already running")
	ErrTunnelNotRunning     = errors.New("tunnel is not running")
	ErrTunnelDisabled       = errors.New("tunnel is disabled")
	ErrTunnelFailedToStart  = errors.New("failed to start tunnel")
)

// TunnelState represents the current state of the tunnel
type TunnelState string

const (
	StateIdle     TunnelState = "idle"
	StateStarting TunnelState = "starting"
	StateRunning  TunnelState = "running"
	StateError    TunnelState = "error"
	StateDisabled TunnelState = "disabled"
)

// TunnelInfo contains information about the tunnel
type TunnelInfo struct {
	State    TunnelState `json:"state"`
	URL      string      `json:"url,omitempty"`
	Hostname string      `json:"hostname,omitempty"`
	Secret   string      `json:"secret,omitempty"`
}

// Manager manages the tunnel connection
type Manager struct {
	state          TunnelState
	info           *TunnelInfo
	configDir      string
	lockFile       *flock.Flock
	client         *LapstoneClient
	stopChan       chan struct{}
	watchdogStop   chan struct{}
	mu             sync.RWMutex
	serverPort     int
	serverSecret   string
}

// NewManager creates a new tunnel manager
func NewManager(configDir string, serverPort int, serverSecret string) (*Manager, error) {
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	m := &Manager{
		state:        StateIdle,
		configDir:    configDir,
		serverPort:   serverPort,
		serverSecret: serverSecret,
	}

	// Check if tunnel is disabled
	if isDisabled() {
		m.state = StateDisabled
	}

	return m, nil
}

// isDisabled checks if tunnel is disabled via environment variable
func isDisabled() bool {
	val := os.Getenv("GOOSE_TUNNEL")
	return val == "no" || val == "none"
}

// Start starts the tunnel
func (m *Manager) Start() (*TunnelInfo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.state == StateDisabled {
		return &TunnelInfo{State: StateDisabled}, fmt.Errorf("tunnel is disabled")
	}

	if m.state == StateRunning {
		return m.info, nil
	}

	// Acquire lock
	lockPath := filepath.Join(m.configDir, "tunnel.lock")
	m.lockFile = flock.New(lockPath)
	locked, err := m.lockFile.TryLock()
	if err != nil {
		return nil, fmt.Errorf("failed to acquire lock: %w", err)
	}
	if !locked {
		return nil, fmt.Errorf("tunnel is locked by another process")
	}

	// Write PID to lock file
	os.WriteFile(lockPath, []byte(fmt.Sprintf("%d", os.Getpid())), 0644)

	m.state = StateStarting

	// Get or generate tunnel secret
	tunnelSecret := m.getTunnelSecret()
	if tunnelSecret == "" {
		tunnelSecret = generateSecret()
		m.setTunnelSecret(tunnelSecret)
	}

	// Create lapstone client
	m.client = NewLapstoneClient(m.serverPort, m.serverSecret, tunnelSecret)

	// Start the tunnel
	m.stopChan = make(chan struct{})
	info, err := m.client.Connect(m.stopChan)
	if err != nil {
		m.state = StateError
		m.lockFile.Unlock()
		return nil, fmt.Errorf("failed to connect tunnel: %w", err)
	}

	m.state = StateRunning
	m.info = &TunnelInfo{
		State:    StateRunning,
		URL:      info.URL,
		Hostname: info.Hostname,
		Secret:   tunnelSecret,
	}

	// Start watchdog
	m.watchdogStop = make(chan struct{})
	go m.watchdog()

	// Enable auto-start
	m.setAutoStart(true)

	return m.info, nil
}

// Stop stops the tunnel
func (m *Manager) Stop(clearAutoStart bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.state != StateRunning {
		return nil
	}

	// Stop watchdog
	if m.watchdogStop != nil {
		close(m.watchdogStop)
		m.watchdogStop = nil
	}

	// Stop client
	if m.stopChan != nil {
		close(m.stopChan)
		m.stopChan = nil
	}

	if m.client != nil {
		m.client.Close()
		m.client = nil
	}

	// Release lock
	if m.lockFile != nil {
		m.lockFile.Unlock()
		m.lockFile = nil
	}

	m.state = StateIdle
	m.info = nil

	if clearAutoStart {
		m.setAutoStart(false)
	}

	return nil
}

// GetInfo returns current tunnel information
func (m *Manager) GetInfo() *TunnelInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.state == StateDisabled {
		return &TunnelInfo{State: StateDisabled}
	}

	// Check if locked by another process
	if m.state == StateIdle {
		if m.isLockedByAnother() {
			return &TunnelInfo{State: StateRunning}
		}
	}

	if m.info != nil {
		return m.info
	}

	return &TunnelInfo{State: m.state}
}

// isLockedByAnother checks if the tunnel lock is held by another process
func (m *Manager) isLockedByAnother() bool {
	lockPath := filepath.Join(m.configDir, "tunnel.lock")
	fl := flock.New(lockPath)
	locked, err := fl.TryLock()
	if err != nil {
		return false
	}
	if locked {
		fl.Unlock()
		return false
	}
	return true
}

// CheckAutoStart checks and starts tunnel if auto-start is enabled
func (m *Manager) CheckAutoStart() {
	m.mu.RLock()
	if m.state != StateIdle {
		m.mu.RUnlock()
		return
	}
	m.mu.RUnlock()

	if !m.getAutoStart() {
		return
	}

	if m.isLockedByAnother() {
		return
	}

	m.Start()
}

// watchdog monitors the tunnel and restarts if needed
func (m *Manager) watchdog() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-m.watchdogStop:
			return
		case <-ticker.C:
			m.mu.RLock()
			running := m.state == StateRunning
			autoStart := m.getAutoStart()
			m.mu.RUnlock()

			if !running && autoStart {
				// Try to restart
				m.Start()
			}
		}
	}
}

// getTunnelSecret reads the tunnel secret from storage
func (m *Manager) getTunnelSecret() string {
	path := filepath.Join(m.configDir, "tunnel_secret")
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(data)
}

// setTunnelSecret stores the tunnel secret
func (m *Manager) setTunnelSecret(secret string) {
	path := filepath.Join(m.configDir, "tunnel_secret")
	os.WriteFile(path, []byte(secret), 0600)
}

// getAutoStart reads the auto-start setting
func (m *Manager) getAutoStart() bool {
	path := filepath.Join(m.configDir, "tunnel_auto_start")
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	return string(data) == "true"
}

// setAutoStart stores the auto-start setting
func (m *Manager) setAutoStart(enabled bool) {
	path := filepath.Join(m.configDir, "tunnel_auto_start")
	value := "false"
	if enabled {
		value = "true"
	}
	os.WriteFile(path, []byte(value), 0644)
}

// generateSecret generates a random secret
func generateSecret() string {
	// Generate a random 32-character hex string
	data := make([]byte, 16)
	for i := range data {
		data[i] = byte(time.Now().UnixNano() % 256)
	}
	return fmt.Sprintf("%x", data)
}

// ConnectionInfo contains tunnel connection details
type ConnectionInfo struct {
	URL      string
	Hostname string
}

// LapstoneClient handles WebSocket tunnel communication
type LapstoneClient struct {
	serverPort   int
	serverSecret string
	tunnelSecret string
	httpClient   *http.Client
	stopChan     chan struct{}
}

// NewLapstoneClient creates a new lapstone client
func NewLapstoneClient(serverPort int, serverSecret, tunnelSecret string) *LapstoneClient {
	return &LapstoneClient{
		serverPort:   serverPort,
		serverSecret: serverSecret,
		tunnelSecret: tunnelSecret,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Connect establishes the tunnel connection
func (c *LapstoneClient) Connect(stopChan chan struct{}) (*ConnectionInfo, error) {
	c.stopChan = stopChan

	// Get worker URL from environment or use default
	workerURL := os.Getenv("GOOSE_TUNNEL_WORKER_URL")
	if workerURL == "" {
		workerURL = "https://tunnel.block.xyz"
	}

	// Register with the tunnel service
	registerURL := workerURL + "/register"
	req, err := http.NewRequest("POST", registerURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("X-Tunnel-Secret", c.tunnelSecret)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to register tunnel: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("registration failed: %s", string(body))
	}

	var result struct {
		URL      string `json:"url"`
		Hostname string `json:"hostname"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to parse registration response: %w", err)
	}

	// Start WebSocket connection in background
	go c.runWebSocket(workerURL)

	return &ConnectionInfo{
		URL:      result.URL,
		Hostname: result.Hostname,
	}, nil
}

// runWebSocket maintains the WebSocket connection
func (c *LapstoneClient) runWebSocket(workerURL string) {
	// Placeholder for WebSocket implementation
	// In a real implementation, this would:
	// 1. Connect to WebSocket endpoint
	// 2. Receive incoming HTTP requests
	// 3. Forward to local server
	// 4. Send responses back through WebSocket

	<-c.stopChan
}

// Close closes the client
func (c *LapstoneClient) Close() {
	// Cleanup
}

// TunnelMessage represents a message from the tunnel
type TunnelMessage struct {
	RequestID string            `json:"requestId"`
	Method    string            `json:"method"`
	Path      string            `json:"path"`
	Headers   map[string]string `json:"headers,omitempty"`
	Body      string            `json:"body,omitempty"`
}

// TunnelResponse represents a response to the tunnel
type TunnelResponse struct {
	RequestID   string            `json:"requestId"`
	Status      int               `json:"status"`
	Headers     map[string]string `json:"headers,omitempty"`
	Body        string            `json:"body,omitempty"`
	Error       string            `json:"error,omitempty"`
	ChunkIndex  *int              `json:"chunkIndex,omitempty"`
	TotalChunks *int              `json:"totalChunks,omitempty"`
	IsChunked   bool              `json:"isChunked"`
	IsStreaming bool              `json:"isStreaming"`
	IsFirstChunk bool             `json:"isFirstChunk"`
	IsLastChunk  bool             `json:"isLastChunk"`
}
