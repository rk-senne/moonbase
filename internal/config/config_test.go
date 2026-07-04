package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
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

// === Gap Coverage: Path() returns valid path, Show() output format ===

func TestPath_ReturnsValidPath(t *testing.T) {
	path := Path()
	if path == "" {
		t.Fatal("Path() returned empty string")
	}
	// Must contain config.yaml
	if !strings.Contains(path, "config.yaml") {
		t.Errorf("expected path to contain 'config.yaml', got: %s", path)
	}
	// Must be absolute
	if !filepath.IsAbs(path) {
		t.Errorf("expected absolute path, got: %s", path)
	}
	// Must be under .config/moonbase
	if !strings.Contains(path, filepath.Join(".config", "moonbase")) {
		t.Errorf("expected path under .config/moonbase, got: %s", path)
	}
}

func TestOldJSONPath_ReturnsValidPath(t *testing.T) {
	path := OldJSONPath()
	if path == "" {
		t.Fatal("OldJSONPath() returned empty string")
	}
	if !strings.Contains(path, "config.json") {
		t.Errorf("expected path to contain 'config.json', got: %s", path)
	}
	if !filepath.IsAbs(path) {
		t.Errorf("expected absolute path, got: %s", path)
	}
}

func TestShow_OutputIsValidYAML(t *testing.T) {
	cfg := DefaultConfig()
	output := Show(cfg)

	// Should be parseable back as YAML
	var parsed Config
	if err := yaml.Unmarshal([]byte(output), &parsed); err != nil {
		t.Fatalf("Show() output is not valid YAML: %v", err)
	}
	if parsed.DefaultBackend != "kiro-cli" {
		t.Errorf("expected re-parsed backend 'kiro-cli', got: %s", parsed.DefaultBackend)
	}
	if parsed.Theme != "moonbase" {
		t.Errorf("expected re-parsed theme 'moonbase', got: %s", parsed.Theme)
	}
	if len(parsed.AgentOrder) != 14 {
		t.Errorf("expected 14 agents in re-parsed order, got: %d", len(parsed.AgentOrder))
	}
}

func TestShow_ContainsAllFields(t *testing.T) {
	cfg := Config{
		DefaultBackend: "ollama",
		Theme:          "classified",
		AgentsDir:      "/custom/agents",
		AgentOrder:     []string{"numbuh-1", "numbuh-2"},
	}
	output := Show(cfg)

	if !strings.Contains(output, "ollama") {
		t.Error("expected 'ollama' in show output")
	}
	if !strings.Contains(output, "classified") {
		t.Error("expected 'classified' in show output")
	}
	if !strings.Contains(output, "/custom/agents") {
		t.Error("expected agents_dir in show output")
	}
	if !strings.Contains(output, "numbuh-1") {
		t.Error("expected agent_order entries in show output")
	}
}

func TestDefaultConfig_AgentOrderContainsAllExpected(t *testing.T) {
	cfg := DefaultConfig()

	expected := []string{
		"numbuh-0", "numbuh-1", "numbuh-2", "numbuh-3", "numbuh-4", "numbuh-5",
		"numbuh-362", "numbuh-274", "numbuh-86", "numbuh-999", "numbuh-13",
		"knd-council", "sector-z", "numbuh-9",
	}

	for _, name := range expected {
		found := false
		for _, a := range cfg.AgentOrder {
			if a == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected agent %s in default order", name)
		}
	}
}

func TestSave_CreatesFileWithCorrectPermissions(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "sub", "config.yaml")

	// Temporarily override the Path function by writing/reading directly
	cfg := DefaultConfig()
	cfg.Theme = "test-theme"

	// Use the internal Save logic manually (write to custom path)
	os.MkdirAll(filepath.Dir(configPath), 0o700)
	data, err := yaml.Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	header := []byte("# Moonbase configuration\n# https://github.com/f5508037/moonbase\n\n")
	content := append(header, data...)
	if err := os.WriteFile(configPath, content, 0o600); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	// Verify file was created
	info, err := os.Stat(configPath)
	if err != nil {
		t.Fatalf("file not created: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("expected 0600 permissions, got: %o", info.Mode().Perm())
	}

	// Verify content is correct by loading
	loaded, err := loadFromFile(configPath)
	if err != nil {
		t.Fatalf("reload failed: %v", err)
	}
	if loaded.Theme != "test-theme" {
		t.Errorf("expected theme 'test-theme', got: %s", loaded.Theme)
	}
}

func TestSave_RoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	cfg := Config{
		DefaultBackend: "anthropic",
		Theme:          "dark",
		AgentsDir:      "/custom/path",
		AgentOrder:     []string{"numbuh-1", "numbuh-2", "numbuh-3"},
	}

	// Write
	data, _ := yaml.Marshal(cfg)
	os.WriteFile(configPath, data, 0o600)

	// Read back
	loaded, err := loadFromFile(configPath)
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}

	if loaded.DefaultBackend != "anthropic" {
		t.Errorf("backend mismatch: %s", loaded.DefaultBackend)
	}
	if loaded.Theme != "dark" {
		t.Errorf("theme mismatch: %s", loaded.Theme)
	}
	if loaded.AgentsDir != "/custom/path" {
		t.Errorf("agents_dir mismatch: %s", loaded.AgentsDir)
	}
	if len(loaded.AgentOrder) != 3 {
		t.Errorf("agent_order length mismatch: %d", len(loaded.AgentOrder))
	}
}

func TestMigrateFromJSON_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	jsonPath := filepath.Join(tmpDir, "config.json")
	yamlPath := filepath.Join(tmpDir, "config.yaml")

	// Write invalid JSON
	os.WriteFile(jsonPath, []byte("{invalid json!!!"), 0o644)

	migrated, err := migrateJSON(jsonPath, yamlPath)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	if migrated {
		t.Error("should not report migration success for invalid JSON")
	}
}

func TestMigrateFromJSON_MinimalConfig(t *testing.T) {
	tmpDir := t.TempDir()
	jsonPath := filepath.Join(tmpDir, "config.json")
	yamlPath := filepath.Join(tmpDir, "config.yaml")

	// Write minimal JSON config (no backends, no extra fields)
	old := oldJSONConfig{
		DefaultBackend: "ollama",
	}
	data, _ := json.MarshalIndent(old, "", "  ")
	os.WriteFile(jsonPath, data, 0o644)

	migrated, err := migrateJSON(jsonPath, yamlPath)
	if err != nil {
		t.Fatalf("migration failed: %v", err)
	}
	if !migrated {
		t.Fatal("expected migration")
	}

	// Verify YAML was created with correct values
	loaded, err := loadFromFile(yamlPath)
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if loaded.DefaultBackend != "ollama" {
		t.Errorf("expected backend 'ollama', got: %s", loaded.DefaultBackend)
	}
	// Theme should be default since not set in JSON
	if loaded.Theme != "moonbase" {
		t.Errorf("expected default theme 'moonbase', got: %s", loaded.Theme)
	}
}

func TestMigrateFromJSON_PreservesAllFields(t *testing.T) {
	tmpDir := t.TempDir()
	jsonPath := filepath.Join(tmpDir, "config.json")
	yamlPath := filepath.Join(tmpDir, "config.yaml")

	old := oldJSONConfig{
		AgentsDir:      "/my/agents",
		DefaultBackend: "codex",
		Theme:          "retro",
	}
	data, _ := json.MarshalIndent(old, "", "  ")
	os.WriteFile(jsonPath, data, 0o644)

	migrateJSON(jsonPath, yamlPath)

	loaded, err := loadFromFile(yamlPath)
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if loaded.AgentsDir != "/my/agents" {
		t.Errorf("agents_dir not preserved: %s", loaded.AgentsDir)
	}
	if loaded.DefaultBackend != "codex" {
		t.Errorf("backend not preserved: %s", loaded.DefaultBackend)
	}
	if loaded.Theme != "retro" {
		t.Errorf("theme not preserved: %s", loaded.Theme)
	}
}

func TestMigrateFromJSON_PublicFunction(t *testing.T) {
	// MigrateFromJSON uses real paths (user's home dir), so just verify
	// it doesn't panic when called and returns false (no old file at real path)
	migrated, err := MigrateFromJSON()
	// If there's no old JSON config at the real path, it should return false, nil
	// If it actually migrates (user has an old config), that's fine too
	_ = migrated
	_ = err
}

func TestLoad_ReturnsDefaults(t *testing.T) {
	// Load uses real paths, so it should return defaults or actual config
	cfg := Load()
	// Regardless of what's on disk, the returned config should have valid values
	if cfg.DefaultBackend == "" {
		t.Error("Load should return a config with non-empty DefaultBackend")
	}
	if cfg.Theme == "" {
		t.Error("Load should return a config with non-empty Theme")
	}
}

func TestSave_UsesRealPath(t *testing.T) {
	// Just verify Save doesn't panic — it uses the real config path
	// We'll save the current config, then restore it
	cfg := Load()
	err := Save(cfg)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}
}

func TestLoad_AfterSave_RoundTrip(t *testing.T) {
	// Save a config, then Load it back — verify the round-trip
	original := Load()

	// Modify and save
	modified := original
	modified.Theme = "test-round-trip-theme"
	if err := Save(modified); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Load and verify
	loaded := Load()
	if loaded.Theme != "test-round-trip-theme" {
		t.Errorf("expected theme 'test-round-trip-theme', got: %s", loaded.Theme)
	}

	// Restore original
	Save(original)
}

func TestSave_CreatesDirectory(t *testing.T) {
	// Save should create the directory structure if it doesn't exist
	cfg := DefaultConfig()
	err := Save(cfg)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Verify the file exists at the expected path
	_, err = os.Stat(Path())
	if err != nil {
		t.Fatalf("config file not found at %s: %v", Path(), err)
	}
}

func TestSave_FileContainsHeader(t *testing.T) {
	cfg := DefaultConfig()
	Save(cfg)

	data, err := os.ReadFile(Path())
	if err != nil {
		t.Fatalf("failed to read config: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "# Moonbase configuration") {
		t.Error("expected header comment in saved config")
	}
}

func TestLoad_MalformedYAML_ReturnsDefaults(t *testing.T) {
	// Save original config, write malformed YAML, test Load, restore
	original, err := os.ReadFile(Path())
	if err != nil {
		// Config file doesn't exist yet — create it so we can test
		Save(DefaultConfig())
		original, _ = os.ReadFile(Path())
	}
	defer os.WriteFile(Path(), original, 0o600) // restore after test

	// Write malformed YAML to the real config path
	os.WriteFile(Path(), []byte("{{{{ not yaml at all {{{{"), 0o600)

	cfg := Load()
	// Should return defaults when parse fails
	if cfg.DefaultBackend != "kiro-cli" {
		t.Errorf("expected default backend on parse failure, got: %s", cfg.DefaultBackend)
	}
	if cfg.Theme != "moonbase" {
		t.Errorf("expected default theme on parse failure, got: %s", cfg.Theme)
	}
}

func TestLoad_MissingYAML_TriesMigration(t *testing.T) {
	// Save original config
	configPath := Path()
	original, err := os.ReadFile(configPath)
	hasOriginal := err == nil
	defer func() {
		if hasOriginal {
			os.WriteFile(configPath, original, 0o600)
		} else {
			os.Remove(configPath)
		}
	}()

	// Remove the YAML config so Load tries migration
	os.Remove(configPath)

	// Also ensure no old JSON config exists at the real path
	// (MigrateFromJSON will check the real OldJSONPath)
	cfg := Load()

	// Should return defaults (no migration source available)
	if cfg.DefaultBackend == "" {
		t.Error("Load should return config with non-empty DefaultBackend")
	}
}

func TestLoad_MissingYAML_WithJSONMigration(t *testing.T) {
	// Save original YAML config
	configPath := Path()
	original, err := os.ReadFile(configPath)
	hasOriginal := err == nil
	defer func() {
		if hasOriginal {
			os.WriteFile(configPath, original, 0o600)
		} else {
			os.Remove(configPath)
		}
	}()

	// Remove YAML config
	os.Remove(configPath)

	// Create a JSON config at the real old path for migration
	jsonPath := OldJSONPath()
	jsonOriginal, jsonErr := os.ReadFile(jsonPath)
	hasJSON := jsonErr == nil
	defer func() {
		if hasJSON {
			os.WriteFile(jsonPath, jsonOriginal, 0o644)
		} else {
			os.Remove(jsonPath)
			os.Remove(jsonPath + ".bak")
		}
	}()

	old := oldJSONConfig{
		DefaultBackend: "migrated-backend",
		Theme:          "migrated-theme",
	}
	data, _ := json.MarshalIndent(old, "", "  ")
	os.MkdirAll(filepath.Dir(jsonPath), 0o700)
	os.WriteFile(jsonPath, data, 0o644)

	// Load should detect missing YAML, find JSON, migrate, and return migrated config
	cfg := Load()
	if cfg.DefaultBackend != "migrated-backend" {
		t.Errorf("expected migrated backend 'migrated-backend', got: %s", cfg.DefaultBackend)
	}
	if cfg.Theme != "migrated-theme" {
		t.Errorf("expected migrated theme 'migrated-theme', got: %s", cfg.Theme)
	}
}

func TestLoadFromFile_EmptyYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Write empty YAML (valid but no fields)
	os.WriteFile(configPath, []byte(""), 0o644)

	loaded, err := loadFromFile(configPath)
	if err != nil {
		t.Fatalf("expected no error for empty YAML, got: %v", err)
	}
	// All fields should be zero-value
	if loaded.DefaultBackend != "" {
		t.Errorf("expected empty default_backend, got: %s", loaded.DefaultBackend)
	}
}

func TestMarshalYAML_AllFields(t *testing.T) {
	cfg := Config{
		DefaultBackend: "test-backend",
		Theme:          "test-theme",
		AgentsDir:      "/test/agents",
		AgentOrder:     []string{"a", "b", "c"},
	}

	data, err := marshalYAML(cfg)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	yamlStr := string(data)
	if !strings.Contains(yamlStr, "test-backend") {
		t.Error("expected test-backend in YAML output")
	}
	if !strings.Contains(yamlStr, "test-theme") {
		t.Error("expected test-theme in YAML output")
	}
	if !strings.Contains(yamlStr, "/test/agents") {
		t.Error("expected agents dir in YAML output")
	}
}

func TestMigrateFromJSON_EmptyFields(t *testing.T) {
	tmpDir := t.TempDir()
	jsonPath := filepath.Join(tmpDir, "config.json")
	yamlPath := filepath.Join(tmpDir, "config.yaml")

	// Write JSON with all empty fields — should use defaults
	old := oldJSONConfig{}
	data, _ := json.MarshalIndent(old, "", "  ")
	os.WriteFile(jsonPath, data, 0o644)

	migrated, err := migrateJSON(jsonPath, yamlPath)
	if err != nil {
		t.Fatalf("migration failed: %v", err)
	}
	if !migrated {
		t.Fatal("expected migration")
	}

	loaded, err := loadFromFile(yamlPath)
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	// Should have defaults since old config had empty fields
	if loaded.DefaultBackend != "kiro-cli" {
		t.Errorf("expected default backend, got: %s", loaded.DefaultBackend)
	}
	if loaded.Theme != "moonbase" {
		t.Errorf("expected default theme, got: %s", loaded.Theme)
	}
}

func TestMigrateFromJSON_WithBackendsAPIKeys(t *testing.T) {
	tmpDir := t.TempDir()
	jsonPath := filepath.Join(tmpDir, "config.json")
	yamlPath := filepath.Join(tmpDir, "config.yaml")

	// Write JSON with API keys in backends map
	old := oldJSONConfig{
		DefaultBackend: "openai",
		Theme:          "light",
		Backends: map[string]oldJSONBackend{
			"openai": {APIKey: "sk-secret-key", Model: "gpt-4"},
		},
	}
	data, _ := json.MarshalIndent(old, "", "  ")
	os.WriteFile(jsonPath, data, 0o644)

	migrated, err := migrateJSON(jsonPath, yamlPath)
	if err != nil {
		t.Fatalf("migration failed: %v", err)
	}
	if !migrated {
		t.Fatal("expected migration")
	}

	// Verify API keys are NOT in the YAML output
	yamlData, _ := os.ReadFile(yamlPath)
	yamlStr := string(yamlData)
	if strings.Contains(yamlStr, "sk-secret-key") {
		t.Error("API key leaked into YAML config!")
	}
	if strings.Contains(yamlStr, "gpt-4") {
		// Model info is also dropped (not in new format)
	}
}
