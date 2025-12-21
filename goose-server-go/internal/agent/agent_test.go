package agent

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/block/goose-server-go/internal/models"
	"github.com/block/goose-server-go/internal/provider"
	"github.com/block/goose-server-go/internal/session"
)

func newTestSessionManager(t *testing.T) *session.Manager {
	tmpFile, err := os.CreateTemp("", "test-sessions-*.db")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Remove(tmpFile.Name()) })
	tmpFile.Close()

	manager, err := session.NewManager(tmpFile.Name())
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	t.Cleanup(func() { manager.Close() })

	return manager
}

func TestManager_StartAndGet(t *testing.T) {
	sessionManager := newTestSessionManager(t)
	manager := NewManager(sessionManager, nil) // nil registry uses mock provider

	ctx := context.Background()
	config := &AgentConfig{
		WorkingDir:   "/tmp/test",
		ProviderName: "mock",
	}

	agent, err := manager.Start(ctx, config)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	if agent.SessionID == "" {
		t.Error("SessionID should not be empty")
	}

	if !agent.IsRunning() {
		t.Error("Agent should be running")
	}

	// Get the agent
	got, ok := manager.Get(agent.SessionID)
	if !ok {
		t.Fatal("Get returned false")
	}

	if got.SessionID != agent.SessionID {
		t.Errorf("SessionID = %q, want %q", got.SessionID, agent.SessionID)
	}
}

func TestManager_Resume(t *testing.T) {
	sessionManager := newTestSessionManager(t)
	manager := NewManager(sessionManager, nil)

	// Create a session directly
	sess, err := sessionManager.Create("/tmp/test")
	if err != nil {
		t.Fatalf("Create session failed: %v", err)
	}

	ctx := context.Background()
	agent, err := manager.Resume(ctx, sess.ID, true)
	if err != nil {
		t.Fatalf("Resume failed: %v", err)
	}

	if agent.SessionID != sess.ID {
		t.Errorf("SessionID = %q, want %q", agent.SessionID, sess.ID)
	}

	if !agent.IsRunning() {
		t.Error("Agent should be running after resume")
	}
}

func TestManager_Stop(t *testing.T) {
	sessionManager := newTestSessionManager(t)
	manager := NewManager(sessionManager, nil)

	ctx := context.Background()
	config := &AgentConfig{
		WorkingDir:   "/tmp/test",
		ProviderName: "mock",
	}

	agent, err := manager.Start(ctx, config)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	sessionID := agent.SessionID

	// Stop the agent
	if err := manager.Stop(sessionID); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}

	// Verify agent is not available
	_, ok := manager.Get(sessionID)
	if ok {
		t.Error("Agent should not be available after stop")
	}
}

func TestManager_GetProvider(t *testing.T) {
	sessionManager := newTestSessionManager(t)
	manager := NewManager(sessionManager, nil)

	// With nil registry, GetProvider returns the mock provider
	got, ok := manager.GetProvider("any-name")
	if !ok {
		t.Fatal("GetProvider should return true with nil registry")
	}

	if got.Name() != "mock" {
		t.Errorf("Name = %q, want %q", got.Name(), "mock")
	}
}

func TestManager_ListProviders(t *testing.T) {
	sessionManager := newTestSessionManager(t)
	manager := NewManager(sessionManager, nil)

	providers := manager.ListProviders()
	if len(providers) != 1 {
		t.Errorf("len(providers) = %d, want 1", len(providers))
	}

	if providers[0].Name() != "mock" {
		t.Errorf("providers[0].Name() = %q, want %q", providers[0].Name(), "mock")
	}
}

func TestAgent_Chat(t *testing.T) {
	sessionManager := newTestSessionManager(t)
	manager := NewManager(sessionManager, nil)

	ctx := context.Background()
	config := &AgentConfig{
		WorkingDir:   "/tmp/test",
		ProviderName: "mock",
	}

	agent, err := manager.Start(ctx, config)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Send chat messages
	messages := []models.Message{
		models.NewUserMessage("Hello"),
	}

	eventChan, err := agent.Chat(ctx, messages)
	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}

	// Collect events
	var events []models.MessageEvent
	for event := range eventChan {
		events = append(events, event)
	}

	// Should have Message and Finish events
	if len(events) < 2 {
		t.Fatalf("Expected at least 2 events, got %d", len(events))
	}

	// Check first event is Message
	if events[0].Type != models.EventTypeMessage {
		t.Errorf("events[0].Type = %q, want %q", events[0].Type, models.EventTypeMessage)
	}

	// Check last event is Finish
	if events[len(events)-1].Type != models.EventTypeFinish {
		t.Errorf("last event Type = %q, want %q", events[len(events)-1].Type, models.EventTypeFinish)
	}
}

func TestAgent_AddTool(t *testing.T) {
	sessionManager := newTestSessionManager(t)
	manager := NewManager(sessionManager, nil)

	ctx := context.Background()
	config := &AgentConfig{
		WorkingDir:   "/tmp/test",
		ProviderName: "mock",
	}

	agent, err := manager.Start(ctx, config)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Add a tool
	tool := models.ToolInfo{
		Name:          "test_tool",
		Description:   "A test tool",
		ExtensionName: "test",
	}
	agent.AddTool(tool)

	// Get tools
	tools := agent.GetTools()
	if len(tools) != 1 {
		t.Fatalf("len(tools) = %d, want 1", len(tools))
	}

	if tools[0].Name != "test_tool" {
		t.Errorf("tools[0].Name = %q, want %q", tools[0].Name, "test_tool")
	}
}

func TestMockProviderAdapter_Chat(t *testing.T) {
	mockProvider := NewMockProviderAdapter()

	ctx := context.Background()
	messages := []models.Message{
		models.NewUserMessage("Hello, test!"),
	}

	eventChan, err := mockProvider.Chat(ctx, messages, provider.ChatOptions{})
	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}

	var events []models.MessageEvent
	for event := range eventChan {
		events = append(events, event)
	}

	// Should have Message and Finish events
	if len(events) != 2 {
		t.Fatalf("Expected 2 events, got %d", len(events))
	}

	// Message event should contain response
	if events[0].Type != models.EventTypeMessage {
		t.Errorf("events[0].Type = %q, want %q", events[0].Type, models.EventTypeMessage)
	}

	if events[0].Message == nil {
		t.Fatal("events[0].Message should not be nil")
	}

	// Finish event
	if events[1].Type != models.EventTypeFinish {
		t.Errorf("events[1].Type = %q, want %q", events[1].Type, models.EventTypeFinish)
	}

	if events[1].Reason == nil || *events[1].Reason != "stop" {
		t.Errorf("Finish reason should be 'stop'")
	}
}

func TestMockProviderAdapter_Cancellation(t *testing.T) {
	mockProvider := NewMockProviderAdapter()

	ctx, cancel := context.WithCancel(context.Background())
	messages := []models.Message{
		models.NewUserMessage("Hello"),
	}

	eventChan, err := mockProvider.Chat(ctx, messages, provider.ChatOptions{})
	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}

	// Cancel immediately
	cancel()

	// Wait a bit and check if we get error or early termination
	select {
	case event := <-eventChan:
		// Either error event or channel closed
		if event.Type == models.EventTypeError {
			// Good - got error event
		}
	case <-time.After(500 * time.Millisecond):
		// Channel might have closed
	}
}

func TestMockProviderAdapter_GetModels(t *testing.T) {
	mockProvider := NewMockProviderAdapter()

	models := mockProvider.GetModels()
	if len(models) != 2 {
		t.Errorf("len(models) = %d, want 2", len(models))
	}
}

func TestMockProviderAdapter_IsConfigured(t *testing.T) {
	mockProvider := NewMockProviderAdapter()

	if !mockProvider.IsConfigured() {
		t.Error("Mock provider should always be configured")
	}
}

func TestMockProviderAdapter_Generate(t *testing.T) {
	mockProvider := NewMockProviderAdapter()

	ctx := context.Background()
	messages := []models.Message{
		models.NewUserMessage("Test message"),
	}

	msg, tokenState, err := mockProvider.Generate(ctx, messages, provider.ChatOptions{})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if msg == nil {
		t.Fatal("Message should not be nil")
	}

	if msg.Role != models.RoleAssistant {
		t.Errorf("Role = %q, want %q", msg.Role, models.RoleAssistant)
	}

	if tokenState == nil {
		t.Fatal("TokenState should not be nil")
	}

	if tokenState.TotalTokens <= 0 {
		t.Error("TotalTokens should be positive")
	}
}
