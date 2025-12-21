package provider

import (
	"context"
	"encoding/json"
	"io"
	"time"

	"github.com/block/goose-server-go/internal/models"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

// EinoAdapter wraps an eino ChatModel to implement the Provider interface
type EinoAdapter struct {
	name             string
	displayName      string
	chatModel        model.BaseChatModel
	models           []ModelInfo
	defaultModel     string
	supportsStreaming bool
	supportsTools    bool
	isConfigured     bool
}

// EinoAdapterConfig contains configuration for creating an EinoAdapter
type EinoAdapterConfig struct {
	Name              string
	DisplayName       string
	ChatModel         model.BaseChatModel
	Models            []ModelInfo
	DefaultModel      string
	SupportsStreaming bool
	SupportsTools     bool
}

// NewEinoAdapter creates a new EinoAdapter
func NewEinoAdapter(cfg EinoAdapterConfig) *EinoAdapter {
	return &EinoAdapter{
		name:              cfg.Name,
		displayName:       cfg.DisplayName,
		chatModel:         cfg.ChatModel,
		models:            cfg.Models,
		defaultModel:      cfg.DefaultModel,
		supportsStreaming: cfg.SupportsStreaming,
		supportsTools:     cfg.SupportsTools,
		isConfigured:      cfg.ChatModel != nil,
	}
}

// Name returns the provider name
func (a *EinoAdapter) Name() string {
	return a.name
}

// DisplayName returns the display name
func (a *EinoAdapter) DisplayName() string {
	return a.displayName
}

// GetModels returns available models
func (a *EinoAdapter) GetModels() []ModelInfo {
	return a.models
}

// GetDefaultModel returns the default model
func (a *EinoAdapter) GetDefaultModel() string {
	return a.defaultModel
}

// IsConfigured returns whether the provider is configured
func (a *EinoAdapter) IsConfigured() bool {
	return a.isConfigured
}

// SupportsStreaming returns whether streaming is supported
func (a *EinoAdapter) SupportsStreaming() bool {
	return a.supportsStreaming
}

// SupportsTools returns whether tool calling is supported
func (a *EinoAdapter) SupportsTools() bool {
	return a.supportsTools
}

// Chat sends messages and returns a stream of events
func (a *EinoAdapter) Chat(ctx context.Context, messages []models.Message, options ChatOptions) (<-chan models.MessageEvent, error) {
	if !a.isConfigured {
		return nil, NewProviderError(ErrNotConfigured, "provider is not configured", nil)
	}

	eventChan := make(chan models.MessageEvent, 10)

	go func() {
		defer close(eventChan)

		// Convert messages to eino format
		einoMsgs := convertToEinoMessages(messages, options.System)

		// Check if we should use streaming
		if a.supportsStreaming {
			a.streamChat(ctx, eventChan, einoMsgs, options)
		} else {
			a.generateChat(ctx, eventChan, einoMsgs, options)
		}
	}()

	return eventChan, nil
}

// streamChat handles streaming chat completion
func (a *EinoAdapter) streamChat(ctx context.Context, eventChan chan<- models.MessageEvent, msgs []*schema.Message, options ChatOptions) {
	reader, err := a.chatModel.Stream(ctx, msgs)
	if err != nil {
		eventChan <- models.NewErrorEvent("Stream failed: " + err.Error())
		return
	}
	defer reader.Close()

	var fullContent string
	var toolCalls []schema.ToolCall
	startTime := time.Now()

	for {
		select {
		case <-ctx.Done():
			eventChan <- models.NewErrorEvent("Request cancelled")
			return
		default:
		}

		chunk, err := reader.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			eventChan <- models.NewErrorEvent("Stream error: " + err.Error())
			return
		}

		if chunk != nil {
			fullContent += chunk.Content
			if len(chunk.ToolCalls) > 0 {
				toolCalls = append(toolCalls, chunk.ToolCalls...)
			}
		}
	}

	// Create the final message
	msg := convertFromEinoMessage(&schema.Message{
		Role:      schema.Assistant,
		Content:   fullContent,
		ToolCalls: toolCalls,
	})

	// Estimate tokens (real implementation would get this from response metadata)
	inputTokens := estimateTokens(msgs)
	outputTokens := int32(len(fullContent) / 4)
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
	eventChan <- models.NewMessageEvent(*msg, tokenState)

	// Send finish event
	_ = startTime // Could be used for latency tracking
	eventChan <- models.NewFinishEvent("stop", tokenState)
}

// generateChat handles non-streaming chat completion
func (a *EinoAdapter) generateChat(ctx context.Context, eventChan chan<- models.MessageEvent, msgs []*schema.Message, options ChatOptions) {
	resp, err := a.chatModel.Generate(ctx, msgs)
	if err != nil {
		eventChan <- models.NewErrorEvent("Generate failed: " + err.Error())
		return
	}

	// Convert response
	msg := convertFromEinoMessage(resp)

	// Estimate tokens
	inputTokens := estimateTokens(msgs)
	outputTokens := int32(len(resp.Content) / 4)
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
	eventChan <- models.NewMessageEvent(*msg, tokenState)

	// Send finish event
	eventChan <- models.NewFinishEvent("stop", tokenState)
}

// Generate sends messages and returns a single response
func (a *EinoAdapter) Generate(ctx context.Context, messages []models.Message, options ChatOptions) (*models.Message, *models.TokenState, error) {
	if !a.isConfigured {
		return nil, nil, NewProviderError(ErrNotConfigured, "provider is not configured", nil)
	}

	// Convert messages to eino format
	einoMsgs := convertToEinoMessages(messages, options.System)

	// Generate response
	resp, err := a.chatModel.Generate(ctx, einoMsgs)
	if err != nil {
		return nil, nil, NewProviderError(ErrRequestFailed, "generation failed", err)
	}

	// Convert response
	msg := convertFromEinoMessage(resp)

	// Estimate tokens
	inputTokens := estimateTokens(einoMsgs)
	outputTokens := int32(len(resp.Content) / 4)
	totalTokens := inputTokens + outputTokens

	tokenState := &models.TokenState{
		InputTokens:             inputTokens,
		OutputTokens:            outputTokens,
		TotalTokens:             totalTokens,
		AccumulatedInputTokens:  inputTokens,
		AccumulatedOutputTokens: outputTokens,
		AccumulatedTotalTokens:  totalTokens,
	}

	return msg, tokenState, nil
}

// convertToEinoMessages converts our message format to eino format
func convertToEinoMessages(messages []models.Message, systemPrompt string) []*schema.Message {
	var result []*schema.Message

	// Add system message if provided
	if systemPrompt != "" {
		result = append(result, &schema.Message{
			Role:    schema.System,
			Content: systemPrompt,
		})
	}

	for _, msg := range messages {
		einoMsg := &schema.Message{}

		// Set role
		switch msg.Role {
		case models.RoleUser:
			einoMsg.Role = schema.User
		case models.RoleAssistant:
			einoMsg.Role = schema.Assistant
		default:
			einoMsg.Role = schema.User
		}

		// Extract text content
		for _, content := range msg.Content {
			if content.Type == "text" && content.Text != nil {
				einoMsg.Content += *content.Text
			}
			// Handle tool calls
			if content.Type == "toolRequest" && content.ToolCall != nil {
				var toolCall struct {
					Name      string          `json:"name"`
					Arguments json.RawMessage `json:"arguments"`
				}
				if err := json.Unmarshal(*content.ToolCall, &toolCall); err == nil {
					einoMsg.ToolCalls = append(einoMsg.ToolCalls, schema.ToolCall{
						ID:   getToolCallID(content),
						Type: "function",
						Function: schema.FunctionCall{
							Name:      toolCall.Name,
							Arguments: string(toolCall.Arguments),
						},
					})
				}
			}
			// Handle tool results
			if content.Type == "toolResult" && content.ToolResult != nil {
				einoMsg.Role = schema.Tool
				einoMsg.ToolCallID = getToolCallID(content)
				if content.Text != nil {
					einoMsg.Content = *content.Text
				} else {
					einoMsg.Content = string(*content.ToolResult)
				}
			}
		}

		result = append(result, einoMsg)
	}

	return result
}

// getToolCallID extracts tool call ID from content
func getToolCallID(content models.MessageContent) string {
	if content.ID != nil {
		return *content.ID
	}
	return ""
}

// convertFromEinoMessage converts eino message to our format
func convertFromEinoMessage(msg *schema.Message) *models.Message {
	result := models.NewAssistantMessage(msg.Content)

	// Handle tool calls
	for _, tc := range msg.ToolCalls {
		toolCall := map[string]interface{}{
			"id":   tc.ID,
			"type": tc.Type,
			"name": tc.Function.Name,
		}
		if tc.Function.Arguments != "" {
			var args interface{}
			if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err == nil {
				toolCall["arguments"] = args
			} else {
				toolCall["arguments"] = tc.Function.Arguments
			}
		}
		tcBytes, _ := json.Marshal(toolCall)
		rawMsg := json.RawMessage(tcBytes)
		result.Content = append(result.Content, models.MessageContent{
			Type:     "toolRequest",
			ToolCall: &rawMsg,
		})
	}

	// Handle reasoning/thinking content
	if msg.ReasoningContent != "" {
		result.Content = append(result.Content, models.MessageContent{
			Type:     "thinking",
			Thinking: &msg.ReasoningContent,
		})
	}

	return &result
}

// estimateTokens provides a rough token estimate
func estimateTokens(msgs []*schema.Message) int32 {
	var total int
	for _, msg := range msgs {
		total += len(msg.Content) / 4 // Rough estimate: 4 chars per token
	}
	return int32(total)
}
