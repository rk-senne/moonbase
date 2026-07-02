package snippets

import (
	"path/filepath"
	"testing"
)

// setSnippetsPath overrides the package-level snippetsPath var to point at a temp dir.
func setSnippetsPath(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "snippets.json")
	snippetsPath = path
	return path
}

func TestLoad_ReturnsNilWhenNoFileExists(t *testing.T) {
	setSnippetsPath(t)

	result := Load()
	if result != nil {
		t.Errorf("expected nil when file doesn't exist, got %v", result)
	}
}

func TestSaveAndLoad_RoundTrip(t *testing.T) {
	setSnippetsPath(t)

	original := []Snippet{
		{Name: "greeting", Content: "Hello, operator.", Agent: "numbuh-1"},
		{Name: "footer", Content: "End of report.", Agent: ""},
	}

	if err := Save(original); err != nil {
		t.Fatalf("Save() returned error: %v", err)
	}

	loaded := Load()
	if loaded == nil {
		t.Fatal("Load() returned nil after Save()")
	}

	if len(loaded) != 2 {
		t.Fatalf("expected 2 snippets, got %d", len(loaded))
	}

	if loaded[0].Name != "greeting" || loaded[0].Content != "Hello, operator." || loaded[0].Agent != "numbuh-1" {
		t.Errorf("snippet[0] mismatch: %+v", loaded[0])
	}

	if loaded[1].Name != "footer" || loaded[1].Content != "End of report." || loaded[1].Agent != "" {
		t.Errorf("snippet[1] mismatch: %+v", loaded[1])
	}
}

func TestAdd_AppendsToExistingSnippets(t *testing.T) {
	setSnippetsPath(t)

	// Start with one snippet
	initial := []Snippet{
		{Name: "first", Content: "I'm first", Agent: ""},
	}
	if err := Save(initial); err != nil {
		t.Fatalf("Save() returned error: %v", err)
	}

	// Add a new one
	if err := Add("second", "I'm second", "numbuh-4"); err != nil {
		t.Fatalf("Add() returned error: %v", err)
	}

	loaded := Load()
	if len(loaded) != 2 {
		t.Fatalf("expected 2 snippets after Add(), got %d", len(loaded))
	}

	if loaded[1].Name != "second" || loaded[1].Content != "I'm second" || loaded[1].Agent != "numbuh-4" {
		t.Errorf("added snippet mismatch: %+v", loaded[1])
	}
}

func TestForAgent_FiltersCorrectly(t *testing.T) {
	setSnippetsPath(t)

	all := []Snippet{
		{Name: "global-one", Content: "Available to all", Agent: ""},
		{Name: "numbuh1-specific", Content: "Only for numbuh-1", Agent: "numbuh-1"},
		{Name: "numbuh4-specific", Content: "Only for numbuh-4", Agent: "numbuh-4"},
		{Name: "global-two", Content: "Also available to all", Agent: ""},
	}

	if err := Save(all); err != nil {
		t.Fatalf("Save() returned error: %v", err)
	}

	// ForAgent("numbuh-1") should return global + numbuh-1 specific
	result := ForAgent("numbuh-1")
	if len(result) != 3 {
		t.Fatalf("expected 3 snippets for numbuh-1, got %d: %+v", len(result), result)
	}

	// Verify we get the right ones
	names := make(map[string]bool)
	for _, s := range result {
		names[s.Name] = true
	}

	if !names["global-one"] || !names["global-two"] || !names["numbuh1-specific"] {
		t.Errorf("unexpected snippets for numbuh-1: %+v", result)
	}
	if names["numbuh4-specific"] {
		t.Errorf("numbuh-4 snippet should not appear for numbuh-1")
	}
}

func TestForAgent_ReturnsNilWhenNoFile(t *testing.T) {
	setSnippetsPath(t)

	result := ForAgent("numbuh-5")
	if result != nil {
		t.Errorf("expected nil when no file exists, got %v", result)
	}
}
