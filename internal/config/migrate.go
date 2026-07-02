package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// oldJSONConfig matches the legacy JSON config format.
type oldJSONConfig struct {
	AgentsDir      string                    `json:"agentsDir"`
	DefaultBackend string                    `json:"defaultBackend"`
	Theme          string                    `json:"theme"`
	Backends       map[string]oldJSONBackend `json:"backends"`
}

type oldJSONBackend struct {
	APIKey string `json:"apiKey,omitempty"`
	Model  string `json:"model,omitempty"`
	Binary string `json:"binary,omitempty"`
}

// MigrateFromJSON detects the legacy config.json and migrates it to config.yaml.
// API keys are intentionally dropped during migration (they now come from env vars).
// Returns (true, nil) if migration succeeded, (false, nil) if no old file was found.
func MigrateFromJSON() (bool, error) {
	return migrateJSON(OldJSONPath(), Path())
}

// migrateJSON is the testable implementation of MigrateFromJSON.
func migrateJSON(jsonPath, yamlPath string) (bool, error) {
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		return false, nil // No old file — nothing to migrate
	}

	var old oldJSONConfig
	if err := json.Unmarshal(data, &old); err != nil {
		return false, fmt.Errorf("parsing old config: %w", err)
	}

	// Convert to new format (drop API keys — those come from env now)
	cfg := DefaultConfig()
	if old.DefaultBackend != "" {
		cfg.DefaultBackend = old.DefaultBackend
	}
	if old.Theme != "" {
		cfg.Theme = old.Theme
	}
	if old.AgentsDir != "" {
		cfg.AgentsDir = old.AgentsDir
	}

	// Save as YAML to specified path
	os.MkdirAll(filepath.Dir(yamlPath), 0o700)
	yamlData, err := marshalYAML(cfg)
	if err != nil {
		return false, fmt.Errorf("marshaling config: %w", err)
	}
	header := []byte("# Moonbase configuration (migrated from JSON)\n\n")
	if err := os.WriteFile(yamlPath, append(header, yamlData...), 0o600); err != nil {
		return false, fmt.Errorf("saving migrated config: %w", err)
	}

	// Rename old file to .bak
	backupPath := jsonPath + ".bak"
	if err := os.Rename(jsonPath, backupPath); err != nil {
		fmt.Fprintf(os.Stderr, "⚠️  Could not rename old config: %v\n", err)
	}

	fmt.Fprintf(os.Stderr, "🌙 Config migrated: %s → %s\n", jsonPath, yamlPath)
	if old.Backends != nil {
		fmt.Fprintf(os.Stderr, "   Note: API keys are no longer stored in config.\n")
		fmt.Fprintf(os.Stderr, "   Use environment variables: OPENAI_API_KEY, ANTHROPIC_API_KEY\n")
	}

	return true, nil
}
