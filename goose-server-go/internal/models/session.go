package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// SessionType represents the type of session
type SessionType string

const (
	SessionTypeUser      SessionType = "user"
	SessionTypeScheduled SessionType = "scheduled"
	SessionTypeSubAgent  SessionType = "sub_agent"
	SessionTypeHidden    SessionType = "hidden"
	SessionTypeTerminal  SessionType = "terminal"
)

// Session represents a chat session
type Session struct {
	ID                       string             `json:"id"`
	WorkingDir               string             `json:"working_dir"`
	Name                     string             `json:"name"`
	CreatedAt                time.Time          `json:"created_at"`
	UpdatedAt                time.Time          `json:"updated_at"`
	ExtensionData            map[string]any     `json:"extension_data"`
	MessageCount             uint64             `json:"message_count"`
	Conversation             Conversation       `json:"conversation,omitempty"`
	InputTokens              *int32             `json:"input_tokens,omitempty"`
	OutputTokens             *int32             `json:"output_tokens,omitempty"`
	TotalTokens              *int32             `json:"total_tokens,omitempty"`
	AccumulatedInputTokens   *int32             `json:"accumulated_input_tokens,omitempty"`
	AccumulatedOutputTokens  *int32             `json:"accumulated_output_tokens,omitempty"`
	AccumulatedTotalTokens   *int32             `json:"accumulated_total_tokens,omitempty"`
	ProviderName             *string            `json:"provider_name,omitempty"`
	ModelConfig              *ModelConfig       `json:"model_config,omitempty"`
	Recipe                   *Recipe            `json:"recipe,omitempty"`
	ScheduleID               *string            `json:"schedule_id,omitempty"`
	SessionType              *SessionType       `json:"session_type,omitempty"`
	UserRecipeValues         map[string]string  `json:"user_recipe_values,omitempty"`
	UserSetName              bool               `json:"user_set_name,omitempty"`
}

// NewSession creates a new session with defaults
func NewSession(workingDir string) *Session {
	now := time.Now()
	sessionType := SessionTypeUser
	return &Session{
		ID:            uuid.New().String(),
		WorkingDir:    workingDir,
		Name:          "New Session",
		CreatedAt:     now,
		UpdatedAt:     now,
		ExtensionData: make(map[string]any),
		MessageCount:  0,
		Conversation:  make(Conversation, 0),
		SessionType:   &sessionType,
	}
}

// ModelConfig represents model configuration
type ModelConfig struct {
	ModelName     string   `json:"model_name"`
	Toolshim      bool     `json:"toolshim"`
	ContextLimit  *uint64  `json:"context_limit,omitempty"`
	FastModel     *string  `json:"fast_model,omitempty"`
	MaxTokens     *int32   `json:"max_tokens,omitempty"`
	Temperature   *float32 `json:"temperature,omitempty"`
	ToolshimModel *string  `json:"toolshim_model,omitempty"`
}

// SessionDisplayInfo is a lightweight session info for listings
type SessionDisplayInfo struct {
	ID                      string  `json:"id"`
	Name                    string  `json:"name"`
	CreatedAt               string  `json:"createdAt"`
	WorkingDir              string  `json:"workingDir"`
	MessageCount            uint64  `json:"messageCount"`
	InputTokens             *int32  `json:"inputTokens,omitempty"`
	OutputTokens            *int32  `json:"outputTokens,omitempty"`
	TotalTokens             *int32  `json:"totalTokens,omitempty"`
	AccumulatedInputTokens  *int32  `json:"accumulatedInputTokens,omitempty"`
	AccumulatedOutputTokens *int32  `json:"accumulatedOutputTokens,omitempty"`
	AccumulatedTotalTokens  *int32  `json:"accumulatedTotalTokens,omitempty"`
	ScheduleID              *string `json:"scheduleId,omitempty"`
}

// ToDisplayInfo converts a Session to SessionDisplayInfo
func (s *Session) ToDisplayInfo() SessionDisplayInfo {
	return SessionDisplayInfo{
		ID:                      s.ID,
		Name:                    s.Name,
		CreatedAt:               s.CreatedAt.Format(time.RFC3339),
		WorkingDir:              s.WorkingDir,
		MessageCount:            s.MessageCount,
		InputTokens:             s.InputTokens,
		OutputTokens:            s.OutputTokens,
		TotalTokens:             s.TotalTokens,
		AccumulatedInputTokens:  s.AccumulatedInputTokens,
		AccumulatedOutputTokens: s.AccumulatedOutputTokens,
		AccumulatedTotalTokens:  s.AccumulatedTotalTokens,
		ScheduleID:              s.ScheduleID,
	}
}

// SessionInsights contains aggregate session statistics
type SessionInsights struct {
	TotalSessions uint64 `json:"totalSessions"`
	TotalTokens   int64  `json:"totalTokens"`
}

// SessionListResponse is the response for listing sessions
type SessionListResponse struct {
	Sessions []Session `json:"sessions"`
}

// Recipe represents a goose recipe
type Recipe struct {
	Title        string            `json:"title"`
	Description  string            `json:"description"`
	Activities   []string          `json:"activities,omitempty"`
	Author       *Author           `json:"author,omitempty"`
	Extensions   []json.RawMessage `json:"extensions,omitempty"`
	Instructions *string           `json:"instructions,omitempty"`
	Parameters   []RecipeParameter `json:"parameters,omitempty"`
	Prompt       *string           `json:"prompt,omitempty"`
	Response     *Response         `json:"response,omitempty"`
	Retry        *RetryConfig      `json:"retry,omitempty"`
	Settings     *Settings         `json:"settings,omitempty"`
	SubRecipes   []SubRecipe       `json:"sub_recipes,omitempty"`
	Version      string            `json:"version,omitempty"`
}

// Author represents a recipe author
type Author struct {
	Contact  *string `json:"contact,omitempty"`
	Metadata *string `json:"metadata,omitempty"`
}

// RecipeParameter represents a recipe parameter
type RecipeParameter struct {
	Key         string   `json:"key"`
	InputType   string   `json:"input_type"`
	Requirement string   `json:"requirement"`
	Description string   `json:"description"`
	Default     *string  `json:"default,omitempty"`
	Options     []string `json:"options,omitempty"`
}

// Response represents recipe response configuration
type Response struct {
	JSONSchema *json.RawMessage `json:"json_schema,omitempty"`
}

// RetryConfig represents retry configuration
type RetryConfig struct {
	MaxRetries              int32          `json:"max_retries"`
	Checks                  []SuccessCheck `json:"checks"`
	OnFailure               *string        `json:"on_failure,omitempty"`
	OnFailureTimeoutSeconds *int64         `json:"on_failure_timeout_seconds,omitempty"`
	TimeoutSeconds          *int64         `json:"timeout_seconds,omitempty"`
}

// SuccessCheck represents a success check
type SuccessCheck struct {
	Type    string `json:"type"`
	Command string `json:"command,omitempty"`
}

// Settings represents recipe settings
type Settings struct {
	GooseModel    *string  `json:"goose_model,omitempty"`
	GooseProvider *string  `json:"goose_provider,omitempty"`
	Temperature   *float32 `json:"temperature,omitempty"`
}

// SubRecipe represents a sub-recipe reference
type SubRecipe struct {
	Name                    string            `json:"name"`
	Path                    string            `json:"path"`
	Description             *string           `json:"description,omitempty"`
	SequentialWhenRepeated  bool              `json:"sequential_when_repeated,omitempty"`
	Values                  map[string]string `json:"values,omitempty"`
}
