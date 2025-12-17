package agent

import (
	"context"
	"time"

	"github.com/block/goose-server-go/internal/models"
)

// MockProvider is a mock LLM provider for testing and development
type MockProvider struct {
	name   string
	models []string
}

// NewMockProvider creates a new mock provider
func NewMockProvider() *MockProvider {
	return &MockProvider{
		name: "mock",
		models: []string{
			"mock-model-v1",
			"mock-model-v2",
		},
	}
}

// Name returns the provider name
func (p *MockProvider) Name() string {
	return p.name
}

// GetModels returns available models
func (p *MockProvider) GetModels() []string {
	return p.models
}

// IsConfigured returns whether the provider is configured
func (p *MockProvider) IsConfigured() bool {
	return true
}

// Chat sends messages and returns a stream of events
func (p *MockProvider) Chat(ctx context.Context, messages []models.Message, options ChatOptions) (<-chan models.MessageEvent, error) {
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
		totalTokens := inputTokens + outputTokens

		tokenState := &models.TokenState{
			InputTokens:             inputTokens,
			OutputTokens:            outputTokens,
			TotalTokens:             totalTokens,
			AccumulatedInputTokens:  inputTokens,
			AccumulatedOutputTokens: outputTokens,
			AccumulatedTotalTokens:  totalTokens,
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

// generateMockResponse creates a contextual mock response
func (p *MockProvider) generateMockResponse(messages []models.Message) string {
	// Get the last user message
	var lastUserMsg string
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == models.RoleUser {
			for _, content := range messages[i].Content {
				if content.Type == "text" && content.Text != nil {
					lastUserMsg = *content.Text
					break
				}
			}
			break
		}
	}

	if lastUserMsg == "" {
		return "I'm the mock provider. I received your message but couldn't extract the text content. In Phase 4+, I will be replaced with real LLM integration."
	}

	// Generate contextual response
	response := "This is a mock response from the Go server. "
	response += "I received your message: \"" + truncate(lastUserMsg, 100) + "\". "
	response += "In Phase 4+, this mock provider will be replaced with real LLM providers like OpenAI, Anthropic, etc."

	return response
}

// estimateTokens provides a rough token count estimate
func (p *MockProvider) estimateTokens(messages []models.Message) int32 {
	var totalChars int
	for _, msg := range messages {
		for _, content := range msg.Content {
			if content.Type == "text" && content.Text != nil {
				totalChars += len(*content.Text)
			}
		}
	}
	// Rough estimate: 1 token ≈ 4 characters
	return int32(totalChars / 4)
}

// truncate truncates a string to maxLen characters
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
