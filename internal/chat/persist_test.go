package chat

import (
	"os"
	"path/filepath"
	"testing"
)

// overrideHome sets HOME to a temp dir so chatDir() resolves there.
// Returns a cleanup function that restores the original value.
func overrideHome(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	original := os.Getenv("HOME")
	t.Setenv("HOME", tmpDir)
	_ = original
	return tmpDir
}

func TestSaveConversation_WritesToCorrectPathWithPermissions(t *testing.T) {
	home := overrideHome(t)

	conv := NewConversation("numbuh-3", "You are an implementer.")
	conv.Add(RoleUser, "Hello")
	conv.Add(RoleAssistant, "Hi there!")

	if err := Save(conv); err != nil {
		t.Fatalf("Save() returned error: %v", err)
	}

	expectedPath := filepath.Join(home, ".config", "moonbase", "chats", "numbuh-3.json")
	info, err := os.Stat(expectedPath)
	if err != nil {
		t.Fatalf("expected file at %s, got error: %v", expectedPath, err)
	}

	// Check file permissions (mask out type bits)
	perm := info.Mode().Perm()
	if perm != 0600 {
		t.Errorf("expected permissions 0600, got %04o", perm)
	}
}

func TestSaveAndLoad_RoundTrip(t *testing.T) {
	overrideHome(t)

	conv := NewConversation("numbuh-4", "You are QA.")
	conv.Add(RoleUser, "Run the tests")
	conv.Add(RoleAssistant, "All tests pass.")

	if err := Save(conv); err != nil {
		t.Fatalf("Save() returned error: %v", err)
	}

	loaded := Load("numbuh-4", "Updated system prompt")
	if loaded == nil {
		t.Fatal("Load() returned nil, expected a conversation")
	}

	if loaded.Agent != "numbuh-4" {
		t.Errorf("expected agent 'numbuh-4', got %q", loaded.Agent)
	}

	// System prompt should be updated to the new value
	if loaded.System != "Updated system prompt" {
		t.Errorf("expected system prompt 'Updated system prompt', got %q", loaded.System)
	}

	if len(loaded.Messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(loaded.Messages))
	}

	if loaded.Messages[0].Role != RoleUser || loaded.Messages[0].Content != "Run the tests" {
		t.Errorf("message[0] mismatch: got role=%q content=%q", loaded.Messages[0].Role, loaded.Messages[0].Content)
	}

	if loaded.Messages[1].Role != RoleAssistant || loaded.Messages[1].Content != "All tests pass." {
		t.Errorf("message[1] mismatch: got role=%q content=%q", loaded.Messages[1].Role, loaded.Messages[1].Content)
	}
}

func TestLoad_ReturnsNilForNonExistentAgent(t *testing.T) {
	overrideHome(t)

	loaded := Load("nonexistent-agent", "some prompt")
	if loaded != nil {
		t.Errorf("expected nil for non-existent agent, got %+v", loaded)
	}
}
