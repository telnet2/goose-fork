package agent

import (
	"context"
	"fmt"
	"time"

	"github.com/block/goose-server-go/internal/models"
	"github.com/block/goose-server-go/internal/provider"
)

// MockProviderAdapter implements provider.Provider for testing
type MockProviderAdapter struct {
	models []provider.ModelInfo
}

// NewMockProviderAdapter creates a new mock provider adapter
func NewMockProviderAdapter() *MockProviderAdapter {
	return &MockProviderAdapter{
		models: []provider.ModelInfo{
			{Name: "mock-model-v1", DisplayName: "Mock Model v1", ContextLength: 128000, SupportsTools: true},
			{Name: "mock-model-v2", DisplayName: "Mock Model v2", ContextLength: 128000, SupportsTools: true},
		},
	}
}

// Name returns the provider name
func (p *MockProviderAdapter) Name() string {
	return "mock"
}

// DisplayName returns the display name
func (p *MockProviderAdapter) DisplayName() string {
	return "Mock Provider"
}

// GetModels returns available models
func (p *MockProviderAdapter) GetModels() []provider.ModelInfo {
	return p.models
}

// GetDefaultModel returns the default model
func (p *MockProviderAdapter) GetDefaultModel() string {
	return "mock-model-v1"
}

// IsConfigured always returns true for mock
func (p *MockProviderAdapter) IsConfigured() bool {
	return true
}

// SupportsStreaming returns true
func (p *MockProviderAdapter) SupportsStreaming() bool {
	return true
}

// SupportsTools returns true
func (p *MockProviderAdapter) SupportsTools() bool {
	return true
}

// Chat sends messages and returns a stream of events
func (p *MockProviderAdapter) Chat(ctx context.Context, messages []models.Message, options provider.ChatOptions) (<-chan models.MessageEvent, error) {
	eventChan := make(chan models.MessageEvent, 10)

	go func() {
		defer close(eventChan)

		// Simulate processing time
		select {
		case <-ctx.Done():
			eventChan <- models.NewErrorEvent("Request cancelled")
			return
		case <-time.After(100 * time.Millisecond):
		}

		// Generate mock response based on input
		responseText := p.generateMockResponse(messages)

		// Create assistant message
		assistantMsg := models.NewAssistantMessage(responseText)

		// Token counts (simulated)
		inputTokens := p.estimateTokens(messages)
		outputTokens := int32(len(responseText) / 4) // Rough estimate

		tokenState := &models.TokenState{
			InputTokens:             inputTokens,
			OutputTokens:            outputTokens,
			TotalTokens:             inputTokens + outputTokens,
			AccumulatedInputTokens:  inputTokens,
			AccumulatedOutputTokens: outputTokens,
			AccumulatedTotalTokens:  inputTokens + outputTokens,
		}

		// Send message event
		select {
		case <-ctx.Done():
			eventChan <- models.NewErrorEvent("Request cancelled")
			return
		case eventChan <- models.NewMessageEvent(assistantMsg, tokenState):
		}

		// Send finish event
		select {
		case <-ctx.Done():
			return
		case eventChan <- models.NewFinishEvent("stop", tokenState):
		}
	}()

	return eventChan, nil
}

// Generate sends messages and returns a single response
func (p *MockProviderAdapter) Generate(ctx context.Context, messages []models.Message, options provider.ChatOptions) (*models.Message, *models.TokenState, error) {
	select {
	case <-ctx.Done():
		return nil, nil, ctx.Err()
	case <-time.After(100 * time.Millisecond):
	}

	responseText := p.generateMockResponse(messages)
	msg := models.NewAssistantMessage(responseText)

	inputTokens := p.estimateTokens(messages)
	outputTokens := int32(len(responseText) / 4)

	tokenState := &models.TokenState{
		InputTokens:             inputTokens,
		OutputTokens:            outputTokens,
		TotalTokens:             inputTokens + outputTokens,
		AccumulatedInputTokens:  inputTokens,
		AccumulatedOutputTokens: outputTokens,
		AccumulatedTotalTokens:  inputTokens + outputTokens,
	}

	return &msg, tokenState, nil
}

// generateMockResponse creates a contextual mock response
func (p *MockProviderAdapter) generateMockResponse(messages []models.Message) string {
	// Find the last user message for context
	var lastUserContent string
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == models.RoleUser {
			for _, content := range messages[i].Content {
				if content.Type == "text" && content.Text != nil {
					lastUserContent = *content.Text
					break
				}
			}
			break
		}
	}

	if lastUserContent == "" {
		return "This is a mock response from the Go server. The mock provider is active because no real LLM provider is configured. Set ANTHROPIC_API_KEY or OPENAI_API_KEY to use real providers."
	}

	truncated := truncate(lastUserContent, 100)
	return fmt.Sprintf("Mock response to: \"%s\"\n\nThis is a mock response from the Go goose-server. To get real responses, configure an LLM provider:\n\n- Set ANTHROPIC_API_KEY for Claude\n- Set OPENAI_API_KEY for GPT models", truncated)
}

// estimateTokens provides a rough token estimate
func (p *MockProviderAdapter) estimateTokens(messages []models.Message) int32 {
	var total int
	for _, msg := range messages {
		for _, content := range msg.Content {
			if content.Type == "text" && content.Text != nil {
				total += len(*content.Text)
			}
		}
	}
	return int32(total / 4) // Rough estimate: 4 chars per token
}

// truncate shortens a string to the given length
func truncate(s string, length int) string {
	if len(s) <= length {
		return s
	}
	return s[:length] + "..."
}
