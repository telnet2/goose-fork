package session

import (
	"os"
	"testing"
	"time"

	"github.com/block/goose-server-go/internal/models"
)

func TestStorage_CreateAndGet(t *testing.T) {
	// Create temp db file
	tmpFile, err := os.CreateTemp("", "test-sessions-*.db")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	storage, err := NewStorage(tmpFile.Name())
	if err != nil {
		t.Fatalf("NewStorage failed: %v", err)
	}
	defer storage.Close()

	// Create session
	session := models.NewSession("/tmp/test")
	session.Name = "Test Session"

	err = storage.Create(session)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Get session
	got, err := storage.Get(session.ID, true)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if got == nil {
		t.Fatal("Get returned nil")
	}

	if got.ID != session.ID {
		t.Errorf("ID = %q, want %q", got.ID, session.ID)
	}

	if got.Name != session.Name {
		t.Errorf("Name = %q, want %q", got.Name, session.Name)
	}

	if got.WorkingDir != session.WorkingDir {
		t.Errorf("WorkingDir = %q, want %q", got.WorkingDir, session.WorkingDir)
	}
}

func TestStorage_List(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test-sessions-*.db")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	storage, err := NewStorage(tmpFile.Name())
	if err != nil {
		t.Fatalf("NewStorage failed: %v", err)
	}
	defer storage.Close()

	// Create multiple sessions
	for i := 0; i < 3; i++ {
		session := models.NewSession("/tmp/test")
		session.Name = "Test Session"
		if err := storage.Create(session); err != nil {
			t.Fatalf("Create failed: %v", err)
		}
	}

	// List
	sessions, err := storage.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(sessions) != 3 {
		t.Errorf("len(sessions) = %d, want 3", len(sessions))
	}
}

func TestStorage_Update(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test-sessions-*.db")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	storage, err := NewStorage(tmpFile.Name())
	if err != nil {
		t.Fatalf("NewStorage failed: %v", err)
	}
	defer storage.Close()

	// Create session
	session := models.NewSession("/tmp/test")
	session.Name = "Original"

	if err := storage.Create(session); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Update
	session.Name = "Updated"
	session.Conversation = append(session.Conversation, models.NewUserMessage("Hello"))
	session.MessageCount = 1

	if err := storage.Update(session); err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	// Verify
	got, err := storage.Get(session.ID, true)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if got.Name != "Updated" {
		t.Errorf("Name = %q, want %q", got.Name, "Updated")
	}

	if got.MessageCount != 1 {
		t.Errorf("MessageCount = %d, want 1", got.MessageCount)
	}

	if len(got.Conversation) != 1 {
		t.Errorf("len(Conversation) = %d, want 1", len(got.Conversation))
	}
}

func TestStorage_Delete(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test-sessions-*.db")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	storage, err := NewStorage(tmpFile.Name())
	if err != nil {
		t.Fatalf("NewStorage failed: %v", err)
	}
	defer storage.Close()

	// Create session
	session := models.NewSession("/tmp/test")
	if err := storage.Create(session); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Delete
	if err := storage.Delete(session.ID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify deleted
	got, err := storage.Get(session.ID, false)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if got != nil {
		t.Error("Session should be deleted")
	}
}

func TestStorage_UpdateName(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test-sessions-*.db")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	storage, err := NewStorage(tmpFile.Name())
	if err != nil {
		t.Fatalf("NewStorage failed: %v", err)
	}
	defer storage.Close()

	// Create session
	session := models.NewSession("/tmp/test")
	session.Name = "Original"
	if err := storage.Create(session); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Update name
	if err := storage.UpdateName(session.ID, "New Name"); err != nil {
		t.Fatalf("UpdateName failed: %v", err)
	}

	// Verify
	got, err := storage.Get(session.ID, false)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if got.Name != "New Name" {
		t.Errorf("Name = %q, want %q", got.Name, "New Name")
	}

	if !got.UserSetName {
		t.Error("UserSetName should be true")
	}
}

func TestStorage_GetInsights(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test-sessions-*.db")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	storage, err := NewStorage(tmpFile.Name())
	if err != nil {
		t.Fatalf("NewStorage failed: %v", err)
	}
	defer storage.Close()

	// Create sessions with tokens
	for i := 0; i < 3; i++ {
		session := models.NewSession("/tmp/test")
		tokens := int32(100 * (i + 1))
		session.AccumulatedTotalTokens = &tokens
		if err := storage.Create(session); err != nil {
			t.Fatalf("Create failed: %v", err)
		}
	}

	// Get insights
	insights, err := storage.GetInsights()
	if err != nil {
		t.Fatalf("GetInsights failed: %v", err)
	}

	if insights.TotalSessions != 3 {
		t.Errorf("TotalSessions = %d, want 3", insights.TotalSessions)
	}

	// 100 + 200 + 300 = 600
	if insights.TotalTokens != 600 {
		t.Errorf("TotalTokens = %d, want 600", insights.TotalTokens)
	}
}

func TestStorage_Conversation(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test-sessions-*.db")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	storage, err := NewStorage(tmpFile.Name())
	if err != nil {
		t.Fatalf("NewStorage failed: %v", err)
	}
	defer storage.Close()

	// Create session with conversation
	session := models.NewSession("/tmp/test")
	session.Conversation = models.Conversation{
		models.NewUserMessage("Hello"),
		models.NewAssistantMessage("Hi there!"),
	}
	session.MessageCount = 2

	if err := storage.Create(session); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Get with conversation
	got, err := storage.Get(session.ID, true)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if len(got.Conversation) != 2 {
		t.Fatalf("len(Conversation) = %d, want 2", len(got.Conversation))
	}

	if got.Conversation[0].Role != models.RoleUser {
		t.Errorf("Conversation[0].Role = %q, want %q", got.Conversation[0].Role, models.RoleUser)
	}

	if got.Conversation[1].Role != models.RoleAssistant {
		t.Errorf("Conversation[1].Role = %q, want %q", got.Conversation[1].Role, models.RoleAssistant)
	}

	// Get without conversation
	got2, err := storage.Get(session.ID, false)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if len(got2.Conversation) != 0 {
		t.Errorf("len(Conversation) = %d, want 0 when includeConversation=false", len(got2.Conversation))
	}
}

func TestStorage_SessionType(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test-sessions-*.db")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	storage, err := NewStorage(tmpFile.Name())
	if err != nil {
		t.Fatalf("NewStorage failed: %v", err)
	}
	defer storage.Close()

	// Create hidden session
	hiddenSession := models.NewSession("/tmp/test")
	hiddenType := models.SessionTypeHidden
	hiddenSession.SessionType = &hiddenType
	if err := storage.Create(hiddenSession); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Create regular session
	regularSession := models.NewSession("/tmp/test")
	if err := storage.Create(regularSession); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// List should only return regular session
	sessions, err := storage.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(sessions) != 1 {
		t.Errorf("len(sessions) = %d, want 1 (hidden should be excluded)", len(sessions))
	}
}

func TestStorage_GetNotFound(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test-sessions-*.db")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	storage, err := NewStorage(tmpFile.Name())
	if err != nil {
		t.Fatalf("NewStorage failed: %v", err)
	}
	defer storage.Close()

	got, err := storage.Get("nonexistent-id", false)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if got != nil {
		t.Error("Get should return nil for nonexistent session")
	}
}

func TestStorage_TimestampPersistence(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test-sessions-*.db")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	storage, err := NewStorage(tmpFile.Name())
	if err != nil {
		t.Fatalf("NewStorage failed: %v", err)
	}
	defer storage.Close()

	// Create session
	session := models.NewSession("/tmp/test")
	originalCreated := session.CreatedAt.Truncate(time.Second)

	if err := storage.Create(session); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Get and verify timestamp
	got, err := storage.Get(session.ID, false)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	gotCreated := got.CreatedAt.Truncate(time.Second)
	if !gotCreated.Equal(originalCreated) {
		t.Errorf("CreatedAt = %v, want %v", gotCreated, originalCreated)
	}
}
