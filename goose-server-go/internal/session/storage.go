package session

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/block/goose-server-go/internal/models"
	_ "github.com/mattn/go-sqlite3"
)

// Storage handles session persistence using SQLite
type Storage struct {
	db *sql.DB
}

// NewStorage creates a new session storage
func NewStorage(dbPath string) (*Storage, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	s := &Storage{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	return s, nil
}

// migrate creates the necessary tables
func (s *Storage) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS sessions (
		id TEXT PRIMARY KEY,
		working_dir TEXT NOT NULL,
		name TEXT NOT NULL,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL,
		extension_data TEXT NOT NULL DEFAULT '{}',
		message_count INTEGER NOT NULL DEFAULT 0,
		conversation TEXT NOT NULL DEFAULT '[]',
		input_tokens INTEGER,
		output_tokens INTEGER,
		total_tokens INTEGER,
		accumulated_input_tokens INTEGER,
		accumulated_output_tokens INTEGER,
		accumulated_total_tokens INTEGER,
		provider_name TEXT,
		model_config TEXT,
		recipe TEXT,
		schedule_id TEXT,
		session_type TEXT DEFAULT 'user',
		user_recipe_values TEXT,
		user_set_name INTEGER DEFAULT 0
	);

	CREATE INDEX IF NOT EXISTS idx_sessions_created_at ON sessions(created_at DESC);
	CREATE INDEX IF NOT EXISTS idx_sessions_schedule_id ON sessions(schedule_id);
	`

	_, err := s.db.Exec(schema)
	return err
}

// Close closes the database connection
func (s *Storage) Close() error {
	return s.db.Close()
}

// Create creates a new session
func (s *Storage) Create(session *models.Session) error {
	extensionData, _ := json.Marshal(session.ExtensionData)
	conversation, _ := json.Marshal(session.Conversation)
	modelConfig, _ := json.Marshal(session.ModelConfig)
	recipe, _ := json.Marshal(session.Recipe)
	userRecipeValues, _ := json.Marshal(session.UserRecipeValues)

	_, err := s.db.Exec(`
		INSERT INTO sessions (
			id, working_dir, name, created_at, updated_at, extension_data,
			message_count, conversation, input_tokens, output_tokens, total_tokens,
			accumulated_input_tokens, accumulated_output_tokens, accumulated_total_tokens,
			provider_name, model_config, recipe, schedule_id, session_type,
			user_recipe_values, user_set_name
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		session.ID, session.WorkingDir, session.Name,
		session.CreatedAt, session.UpdatedAt,
		string(extensionData), session.MessageCount, string(conversation),
		session.InputTokens, session.OutputTokens, session.TotalTokens,
		session.AccumulatedInputTokens, session.AccumulatedOutputTokens, session.AccumulatedTotalTokens,
		session.ProviderName, string(modelConfig), string(recipe),
		session.ScheduleID, session.SessionType, string(userRecipeValues),
		session.UserSetName,
	)
	return err
}

// Get retrieves a session by ID
func (s *Storage) Get(id string, includeConversation bool) (*models.Session, error) {
	session := &models.Session{}
	var extensionDataStr, conversationStr, modelConfigStr, recipeStr, userRecipeValuesStr sql.NullString
	var sessionType sql.NullString
	var userSetName sql.NullInt64

	err := s.db.QueryRow(`
		SELECT id, working_dir, name, created_at, updated_at, extension_data,
			   message_count, conversation, input_tokens, output_tokens, total_tokens,
			   accumulated_input_tokens, accumulated_output_tokens, accumulated_total_tokens,
			   provider_name, model_config, recipe, schedule_id, session_type,
			   user_recipe_values, user_set_name
		FROM sessions WHERE id = ?
	`, id).Scan(
		&session.ID, &session.WorkingDir, &session.Name,
		&session.CreatedAt, &session.UpdatedAt,
		&extensionDataStr, &session.MessageCount, &conversationStr,
		&session.InputTokens, &session.OutputTokens, &session.TotalTokens,
		&session.AccumulatedInputTokens, &session.AccumulatedOutputTokens, &session.AccumulatedTotalTokens,
		&session.ProviderName, &modelConfigStr, &recipeStr,
		&session.ScheduleID, &sessionType, &userRecipeValuesStr,
		&userSetName,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	// Parse JSON fields
	if extensionDataStr.Valid {
		json.Unmarshal([]byte(extensionDataStr.String), &session.ExtensionData)
	}
	if session.ExtensionData == nil {
		session.ExtensionData = make(map[string]any)
	}

	if includeConversation && conversationStr.Valid {
		json.Unmarshal([]byte(conversationStr.String), &session.Conversation)
	}

	if modelConfigStr.Valid && modelConfigStr.String != "" && modelConfigStr.String != "null" {
		var mc models.ModelConfig
		if json.Unmarshal([]byte(modelConfigStr.String), &mc) == nil {
			session.ModelConfig = &mc
		}
	}

	if recipeStr.Valid && recipeStr.String != "" && recipeStr.String != "null" {
		var r models.Recipe
		if json.Unmarshal([]byte(recipeStr.String), &r) == nil {
			session.Recipe = &r
		}
	}

	if userRecipeValuesStr.Valid {
		json.Unmarshal([]byte(userRecipeValuesStr.String), &session.UserRecipeValues)
	}

	if sessionType.Valid {
		st := models.SessionType(sessionType.String)
		session.SessionType = &st
	}

	session.UserSetName = userSetName.Valid && userSetName.Int64 == 1

	return session, nil
}

// List retrieves all sessions
func (s *Storage) List() ([]models.Session, error) {
	rows, err := s.db.Query(`
		SELECT id, working_dir, name, created_at, updated_at, extension_data,
			   message_count, input_tokens, output_tokens, total_tokens,
			   accumulated_input_tokens, accumulated_output_tokens, accumulated_total_tokens,
			   provider_name, model_config, recipe, schedule_id, session_type,
			   user_recipe_values, user_set_name
		FROM sessions
		WHERE session_type != 'hidden' OR session_type IS NULL
		ORDER BY updated_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []models.Session
	for rows.Next() {
		var session models.Session
		var extensionDataStr, modelConfigStr, recipeStr, userRecipeValuesStr sql.NullString
		var sessionType sql.NullString
		var userSetName sql.NullInt64

		err := rows.Scan(
			&session.ID, &session.WorkingDir, &session.Name,
			&session.CreatedAt, &session.UpdatedAt,
			&extensionDataStr, &session.MessageCount,
			&session.InputTokens, &session.OutputTokens, &session.TotalTokens,
			&session.AccumulatedInputTokens, &session.AccumulatedOutputTokens, &session.AccumulatedTotalTokens,
			&session.ProviderName, &modelConfigStr, &recipeStr,
			&session.ScheduleID, &sessionType, &userRecipeValuesStr,
			&userSetName,
		)
		if err != nil {
			return nil, err
		}

		// Parse JSON fields
		if extensionDataStr.Valid {
			json.Unmarshal([]byte(extensionDataStr.String), &session.ExtensionData)
		}
		if session.ExtensionData == nil {
			session.ExtensionData = make(map[string]any)
		}

		if sessionType.Valid {
			st := models.SessionType(sessionType.String)
			session.SessionType = &st
		}

		session.UserSetName = userSetName.Valid && userSetName.Int64 == 1

		sessions = append(sessions, session)
	}

	if sessions == nil {
		sessions = []models.Session{}
	}

	return sessions, nil
}

// Update updates an existing session
func (s *Storage) Update(session *models.Session) error {
	extensionData, _ := json.Marshal(session.ExtensionData)
	conversation, _ := json.Marshal(session.Conversation)
	modelConfig, _ := json.Marshal(session.ModelConfig)
	recipe, _ := json.Marshal(session.Recipe)
	userRecipeValues, _ := json.Marshal(session.UserRecipeValues)

	session.UpdatedAt = time.Now()

	_, err := s.db.Exec(`
		UPDATE sessions SET
			working_dir = ?, name = ?, updated_at = ?, extension_data = ?,
			message_count = ?, conversation = ?, input_tokens = ?, output_tokens = ?,
			total_tokens = ?, accumulated_input_tokens = ?, accumulated_output_tokens = ?,
			accumulated_total_tokens = ?, provider_name = ?, model_config = ?,
			recipe = ?, schedule_id = ?, session_type = ?, user_recipe_values = ?,
			user_set_name = ?
		WHERE id = ?
	`,
		session.WorkingDir, session.Name, session.UpdatedAt, string(extensionData),
		session.MessageCount, string(conversation), session.InputTokens, session.OutputTokens,
		session.TotalTokens, session.AccumulatedInputTokens, session.AccumulatedOutputTokens,
		session.AccumulatedTotalTokens, session.ProviderName, string(modelConfig),
		string(recipe), session.ScheduleID, session.SessionType, string(userRecipeValues),
		session.UserSetName, session.ID,
	)
	return err
}

// Delete deletes a session by ID
func (s *Storage) Delete(id string) error {
	result, err := s.db.Exec("DELETE FROM sessions WHERE id = ?", id)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return sql.ErrNoRows
	}

	return nil
}

// UpdateName updates only the session name
func (s *Storage) UpdateName(id, name string) error {
	result, err := s.db.Exec(
		"UPDATE sessions SET name = ?, user_set_name = 1, updated_at = ? WHERE id = ?",
		name, time.Now(), id,
	)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return sql.ErrNoRows
	}

	return nil
}

// GetInsights returns aggregate session statistics
func (s *Storage) GetInsights() (*models.SessionInsights, error) {
	var insights models.SessionInsights
	var totalTokens sql.NullInt64

	err := s.db.QueryRow(`
		SELECT COUNT(*), COALESCE(SUM(accumulated_total_tokens), 0)
		FROM sessions
		WHERE session_type != 'hidden' OR session_type IS NULL
	`).Scan(&insights.TotalSessions, &totalTokens)

	if err != nil {
		return nil, err
	}

	if totalTokens.Valid {
		insights.TotalTokens = totalTokens.Int64
	}

	return &insights, nil
}

// GetByScheduleID returns sessions for a given schedule
func (s *Storage) GetByScheduleID(scheduleID string, limit int) ([]models.SessionDisplayInfo, error) {
	rows, err := s.db.Query(`
		SELECT id, name, created_at, working_dir, message_count,
			   input_tokens, output_tokens, total_tokens,
			   accumulated_input_tokens, accumulated_output_tokens, accumulated_total_tokens,
			   schedule_id
		FROM sessions
		WHERE schedule_id = ?
		ORDER BY created_at DESC
		LIMIT ?
	`, scheduleID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []models.SessionDisplayInfo
	for rows.Next() {
		var s models.SessionDisplayInfo
		var createdAt time.Time

		err := rows.Scan(
			&s.ID, &s.Name, &createdAt, &s.WorkingDir, &s.MessageCount,
			&s.InputTokens, &s.OutputTokens, &s.TotalTokens,
			&s.AccumulatedInputTokens, &s.AccumulatedOutputTokens, &s.AccumulatedTotalTokens,
			&s.ScheduleID,
		)
		if err != nil {
			return nil, err
		}

		s.CreatedAt = createdAt.Format(time.RFC3339)
		sessions = append(sessions, s)
	}

	if sessions == nil {
		sessions = []models.SessionDisplayInfo{}
	}

	return sessions, nil
}
