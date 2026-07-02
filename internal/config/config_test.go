package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.DefaultBackend != "kiro-cli" {
		t.Errorf("expected default backend 'kiro-cli', got: %s", cfg.DefaultBackend)
	}
	if cfg.Theme != "moonbase" {
		t.Errorf("expected theme 'moonbase', got: %s", cfg.Theme)
	}
	if len(cfg.AgentOrder) != 14 {
		t.Errorf("expected 14 agents in order, got: %d", len(cfg.AgentOrder))
	}
	if cfg.AgentsDir != "" {
		t.Error("expected empty agents_dir (auto-detect)")
	}
}

func TestSaveAndLoad(t *testing.T) {
	// Use temp dir
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Override Path for test
	origPath := Path()
	defer func() { _ = origPath }()

	cfg := DefaultConfig()
	cfg.Theme = "dark"
	cfg.DefaultBackend = "ollama"

	// Save directly to temp path
	os.MkdirAll(filepath.Dir(configPath), 0o755)
	data, _ := marshalConfig(cfg)
	os.WriteFile(configPath, data, 0o644)

	// Load from temp path
	loaded, err := loadFromPath(configPath)
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}

	if loaded.Theme != "dark" {
		t.Errorf("expected theme 'dark', got: %s", loaded.Theme)
	}
	if loaded.DefaultBackend != "ollama" {
		t.Errorf("expected backend 'ollama', got: %s", loaded.DefaultBackend)
	}
}

func TestLoadMissingFile(t *testing.T) {
	cfg, err := loadFromPath("/nonexistent/path/config.yaml")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
	// Should still get usable zero-value config
	_ = cfg
}

func TestLoadMalformedYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Write garbage YAML
	os.WriteFile(configPath, []byte("{{{{ not yaml at all {{{{"), 0o644)

	_, err := loadFromPath(configPath)
	if err == nil {
		t.Fatal("expected error for malformed YAML")
	}
}

func TestMigrateFromJSON(t *testing.T) {
	tmpDir := t.TempDir()
	jsonPath := filepath.Join(tmpDir, "config.json")
	yamlPath := filepath.Join(tmpDir, "config.yaml")

	// Write old JSON config with API keys
	old := oldJSONConfig{
		AgentsDir:      "./my-agents",
		DefaultBackend: "anthropic",
		Theme:          "dark",
		Backends: map[string]oldJSONBackend{
			"openai":    {APIKey: "sk-12345", Model: "gpt-4o"},
			"anthropic": {APIKey: "sk-ant-xxx", Model: "claude-3"},
		},
	}
	data, _ := json.MarshalIndent(old, "", "  ")
	os.WriteFile(jsonPath, data, 0o644)

	// Run migration
	migrated, err := migrateFromJSONPaths(jsonPath, yamlPath)
	if err != nil {
		t.Fatalf("migration failed: %v", err)
	}
	if !migrated {
		t.Fatal("expected migration to happen")
	}

	// Verify YAML was created
	yamlData, err := os.ReadFile(yamlPath)
	if err != nil {
		t.Fatal("YAML config not created")
	}

	yamlStr := string(yamlData)
	if !strings.Contains(yamlStr, "anthropic") {
		t.Error("expected default_backend preserved")
	}
	if !strings.Contains(yamlStr, "dark") {
		t.Error("expected theme preserved")
	}
	// API keys must NOT be in the YAML
	if strings.Contains(yamlStr, "sk-12345") {
		t.Error("API key leaked into YAML config!")
	}
	if strings.Contains(yamlStr, "sk-ant") {
		t.Error("API key leaked into YAML config!")
	}

	// Verify JSON was renamed to .bak
	if _, err := os.Stat(jsonPath); !os.IsNotExist(err) {
		t.Error("old JSON should have been renamed")
	}
	if _, err := os.Stat(jsonPath + ".bak"); os.IsNotExist(err) {
		t.Error("backup .bak file should exist")
	}
}

func TestMigrateNoOldFile(t *testing.T) {
	migrated, err := migrateFromJSONPaths("/nonexistent/config.json", "/nonexistent/config.yaml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if migrated {
		t.Error("expected no migration when file doesn't exist")
	}
}

func TestShow(t *testing.T) {
	cfg := DefaultConfig()
	output := Show(cfg)

	if !strings.Contains(output, "default_backend") {
		t.Error("show output should contain default_backend")
	}
	if !strings.Contains(output, "kiro-cli") {
		t.Error("show output should contain kiro-cli")
	}
	if !strings.Contains(output, "theme") {
		t.Error("show output should contain theme")
	}
}

// --- Test helpers (internal functions exposed for testing) ---

func marshalConfig(cfg Config) ([]byte, error) {
	return marshalYAML(cfg)
}

func loadFromPath(path string) (Config, error) {
	return loadFromFile(path)
}

func migrateFromJSONPaths(jsonPath, yamlPath string) (bool, error) {
	return migrateJSON(jsonPath, yamlPath)
}
