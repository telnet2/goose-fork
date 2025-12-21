package tunnel

import (
	"os"
	"path/filepath"
	"testing"
)

func setupTestManager(t *testing.T) (*Manager, string, func()) {
	tempDir, err := os.MkdirTemp("", "tunnel_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	manager, err := NewManager(tempDir, 3000, "test-secret")
	if err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("failed to create manager: %v", err)
	}

	cleanup := func() {
		manager.Stop(true)
		os.RemoveAll(tempDir)
	}

	return manager, tempDir, cleanup
}

func TestNewManager(t *testing.T) {
	manager, _, cleanup := setupTestManager(t)
	defer cleanup()

	if manager == nil {
		t.Fatal("expected non-nil manager")
	}
}

func TestManager_InitialStatus(t *testing.T) {
	manager, _, cleanup := setupTestManager(t)
	defer cleanup()

	info := manager.GetInfo()

	if info.State != StateIdle {
		t.Errorf("expected initial state to be Idle, got %s", info.State)
	}

	if info.URL != "" {
		t.Errorf("expected empty URL initially, got %s", info.URL)
	}

	if info.Hostname != "" {
		t.Errorf("expected empty hostname initially, got %s", info.Hostname)
	}
}

func TestManager_Stop_NotRunning(t *testing.T) {
	manager, _, cleanup := setupTestManager(t)
	defer cleanup()

	// Stop should not error when not running
	if err := manager.Stop(false); err != nil {
		t.Errorf("unexpected error stopping non-running tunnel: %v", err)
	}

	// State should remain idle
	info := manager.GetInfo()
	if info.State != StateIdle {
		t.Errorf("expected state to be Idle, got %s", info.State)
	}
}

func TestManager_CheckAutoStart(t *testing.T) {
	manager, _, cleanup := setupTestManager(t)
	defer cleanup()

	// This should not panic or error
	manager.CheckAutoStart()
}

func TestTunnelState_String(t *testing.T) {
	tests := []struct {
		state    TunnelState
		expected string
	}{
		{StateIdle, "idle"},
		{StateStarting, "starting"},
		{StateRunning, "running"},
		{StateError, "error"},
		{StateDisabled, "disabled"},
	}

	for _, tt := range tests {
		t.Run(string(tt.state), func(t *testing.T) {
			if string(tt.state) != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, tt.state)
			}
		})
	}
}

func TestTunnelInfo_Fields(t *testing.T) {
	info := TunnelInfo{
		State:    StateRunning,
		URL:      "https://example.goose.run",
		Hostname: "example.goose.run",
		Secret:   "secret123",
	}

	// Verify the struct can be created with expected values
	if info.State != StateRunning {
		t.Errorf("expected state Running, got %s", info.State)
	}
	if info.URL != "https://example.goose.run" {
		t.Errorf("unexpected URL: %s", info.URL)
	}
	if info.Hostname != "example.goose.run" {
		t.Errorf("unexpected Hostname: %s", info.Hostname)
	}
	if info.Secret != "secret123" {
		t.Errorf("unexpected Secret: %s", info.Secret)
	}
}

func TestManager_ConfigDir(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "tunnel_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Test that manager creates the config directory
	configDir := filepath.Join(tempDir, "subdir")
	manager, err := NewManager(configDir, 3000, "secret")
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}
	defer manager.Stop(true)

	// Verify directory was created
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		t.Error("expected config directory to be created")
	}
}

func TestTunnelError_Messages(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{
			name: "already running",
			err:  ErrTunnelAlreadyRunning,
			want: "tunnel is already running",
		},
		{
			name: "not running",
			err:  ErrTunnelNotRunning,
			want: "tunnel is not running",
		},
		{
			name: "disabled",
			err:  ErrTunnelDisabled,
			want: "tunnel is disabled",
		},
		{
			name: "failed to start",
			err:  ErrTunnelFailedToStart,
			want: "failed to start tunnel",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Error() != tt.want {
				t.Errorf("expected error message '%s', got '%s'", tt.want, tt.err.Error())
			}
		})
	}
}

// Test concurrent access to manager
func TestManager_Concurrent(t *testing.T) {
	manager, _, cleanup := setupTestManager(t)
	defer cleanup()

	done := make(chan bool)

	// Multiple goroutines accessing info
	for i := 0; i < 10; i++ {
		go func() {
			_ = manager.GetInfo()
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

// Test that manager handles missing config dir gracefully
func TestNewManager_CreateDir(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "tunnel_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create manager with non-existent subdirectory
	newDir := filepath.Join(tempDir, "new", "nested", "dir")
	manager, err := NewManager(newDir, 3000, "secret")
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}
	defer manager.Stop(true)

	// Directory should have been created
	if _, err := os.Stat(newDir); os.IsNotExist(err) {
		t.Error("expected directory to be created")
	}
}

func TestManager_GetInfo_NoMutation(t *testing.T) {
	manager, _, cleanup := setupTestManager(t)
	defer cleanup()

	info1 := manager.GetInfo()
	info2 := manager.GetInfo()

	// Both should have the same state
	if info1.State != info2.State {
		t.Error("expected same state from multiple GetInfo calls")
	}
}
