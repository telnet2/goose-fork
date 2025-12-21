package session

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/block/goose-server-go/internal/models"
)

// Manager handles session lifecycle operations
type Manager struct {
	storage *Storage
	cache   sync.Map // map[string]*models.Session for quick access
}

// NewManager creates a new session manager
func NewManager(dbPath string) (*Manager, error) {
	storage, err := NewStorage(dbPath)
	if err != nil {
		return nil, err
	}

	return &Manager{
		storage: storage,
	}, nil
}

// Close closes the manager and its storage
func (m *Manager) Close() error {
	return m.storage.Close()
}

// Create creates a new session
func (m *Manager) Create(workingDir string) (*models.Session, error) {
	session := models.NewSession(workingDir)

	if err := m.storage.Create(session); err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	m.cache.Store(session.ID, session)
	return session, nil
}

// Get retrieves a session by ID
func (m *Manager) Get(id string, includeConversation bool) (*models.Session, error) {
	// Check cache first (for frequently accessed sessions)
	if cached, ok := m.cache.Load(id); ok {
		session := cached.(*models.Session)
		if !includeConversation {
			// Return copy without conversation
			copy := *session
			copy.Conversation = nil
			return &copy, nil
		}
		return session, nil
	}

	session, err := m.storage.Get(id, includeConversation)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	if session == nil {
		return nil, nil
	}

	// Cache if conversation was loaded
	if includeConversation {
		m.cache.Store(id, session)
	}

	return session, nil
}

// List returns all sessions
func (m *Manager) List() ([]models.Session, error) {
	return m.storage.List()
}

// Update updates a session
func (m *Manager) Update(session *models.Session) error {
	if err := m.storage.Update(session); err != nil {
		return fmt.Errorf("failed to update session: %w", err)
	}

	m.cache.Store(session.ID, session)
	return nil
}

// Delete deletes a session
func (m *Manager) Delete(id string) error {
	if err := m.storage.Delete(id); err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	m.cache.Delete(id)
	return nil
}

// UpdateName updates only the session name
func (m *Manager) UpdateName(id, name string) error {
	if err := m.storage.UpdateName(id, name); err != nil {
		return err
	}

	// Update cache if present
	if cached, ok := m.cache.Load(id); ok {
		session := cached.(*models.Session)
		session.Name = name
		session.UserSetName = true
	}

	return nil
}

// GetInsights returns aggregate session statistics
func (m *Manager) GetInsights() (*models.SessionInsights, error) {
	return m.storage.GetInsights()
}

// Export exports a session as formatted text
func (m *Manager) Export(id string) (string, error) {
	session, err := m.Get(id, true)
	if err != nil {
		return "", err
	}
	if session == nil {
		return "", fmt.Errorf("session not found")
	}

	// Format as readable text
	result := fmt.Sprintf("# Session: %s\n\n", session.Name)
	result += fmt.Sprintf("Created: %s\n", session.CreatedAt.Format("2006-01-02 15:04:05"))
	result += fmt.Sprintf("Working Directory: %s\n\n", session.WorkingDir)
	result += "---\n\n"

	for _, msg := range session.Conversation {
		role := "User"
		if msg.Role == models.RoleAssistant {
			role = "Assistant"
		}
		result += fmt.Sprintf("## %s\n\n", role)

		for _, content := range msg.Content {
			switch content.Type {
			case "text":
				if content.Text != nil {
					result += *content.Text + "\n\n"
				}
			case "toolRequest":
				if content.ToolName != nil {
					result += fmt.Sprintf("*Tool Request: %s*\n\n", *content.ToolName)
				}
			case "toolResponse":
				result += "*Tool Response*\n\n"
			}
		}
	}

	return result, nil
}

// Import imports a session from JSON
func (m *Manager) Import(jsonData string) (*models.Session, error) {
	var session models.Session
	if err := json.Unmarshal([]byte(jsonData), &session); err != nil {
		return nil, fmt.Errorf("invalid session JSON: %w", err)
	}

	// Generate new ID to avoid conflicts
	newSession := models.NewSession(session.WorkingDir)
	newSession.Name = session.Name
	newSession.Conversation = session.Conversation
	newSession.MessageCount = uint64(len(session.Conversation))
	newSession.ExtensionData = session.ExtensionData

	if err := m.storage.Create(newSession); err != nil {
		return nil, fmt.Errorf("failed to import session: %w", err)
	}

	return newSession, nil
}

// AddMessage adds a message to a session
func (m *Manager) AddMessage(id string, message models.Message) error {
	session, err := m.Get(id, true)
	if err != nil {
		return err
	}
	if session == nil {
		return fmt.Errorf("session not found")
	}

	session.Conversation = append(session.Conversation, message)
	session.MessageCount = uint64(len(session.Conversation))

	return m.Update(session)
}

// UpdateTokens updates the token counts for a session
func (m *Manager) UpdateTokens(id string, input, output int32) error {
	session, err := m.Get(id, false)
	if err != nil {
		return err
	}
	if session == nil {
		return fmt.Errorf("session not found")
	}

	total := input + output
	session.InputTokens = &input
	session.OutputTokens = &output
	session.TotalTokens = &total

	// Update accumulated
	var accInput, accOutput, accTotal int32
	if session.AccumulatedInputTokens != nil {
		accInput = *session.AccumulatedInputTokens
	}
	if session.AccumulatedOutputTokens != nil {
		accOutput = *session.AccumulatedOutputTokens
	}
	if session.AccumulatedTotalTokens != nil {
		accTotal = *session.AccumulatedTotalTokens
	}

	accInput += input
	accOutput += output
	accTotal += total

	session.AccumulatedInputTokens = &accInput
	session.AccumulatedOutputTokens = &accOutput
	session.AccumulatedTotalTokens = &accTotal

	return m.Update(session)
}

// GetByScheduleID returns sessions for a given schedule
func (m *Manager) GetByScheduleID(scheduleID string, limit int) ([]models.SessionDisplayInfo, error) {
	return m.storage.GetByScheduleID(scheduleID, limit)
}

// ClearCache clears the session cache
func (m *Manager) ClearCache() {
	m.cache = sync.Map{}
}
