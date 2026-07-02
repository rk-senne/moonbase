package chat

import (
	"encoding/json"
	"os"
	"path/filepath"
)

func chatDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "moonbase", "chats")
}

func chatPath(agent string) string {
	return filepath.Join(chatDir(), agent+".json")
}

// Save persists a conversation to disk
func Save(conv *Conversation) error {
	os.MkdirAll(chatDir(), 0700)
	data, err := json.MarshalIndent(conv, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(chatPath(conv.Agent), data, 0600)
}

// Load restores a conversation from disk, or returns nil if none exists
func Load(agent, systemPrompt string) *Conversation {
	data, err := os.ReadFile(chatPath(agent))
	if err != nil {
		return nil
	}
	var conv Conversation
	if err := json.Unmarshal(data, &conv); err != nil {
		return nil
	}
	// Update system prompt in case it changed
	conv.System = systemPrompt
	return &conv
}
