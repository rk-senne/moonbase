// Package config manages moonbase user preferences stored as YAML.
// No secrets or API keys are stored in config — those come exclusively from
// environment variables. Supports automatic migration from the legacy JSON format.
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config holds moonbase user preferences.
// No API keys or secrets — those come from environment variables.
//
// SECURITY: YAML deserialization safety
// gopkg.in/yaml.v3 is safe against YAML deserialization attacks because:
// 1. It does NOT support custom constructors or !!python/object-style tags
// 2. It does NOT execute code during unmarshaling
// 3. It only maps YAML scalars/sequences/mappings to Go struct fields
// 4. Unknown fields are silently ignored (no gadget chains possible)
// 5. The Config struct uses only primitive types (string, []string) —
//    no interfaces, func fields, or unsafe.Pointer that could be exploited
//
// Unlike PyYAML's yaml.load() or Java's SnakeYAML, Go's yaml.v3 has no
// known deserialization vulnerabilities. The strict typing of Go structs
// provides an additional layer of defense.
type Config struct {
	DefaultBackend string   `yaml:"default_backend"`       // Preferred AI backend (e.g., "kiro-cli")
	Theme          string   `yaml:"theme"`                 // TUI color theme name
	AgentsDir      string   `yaml:"agents_dir,omitempty"`  // Custom agents directory (empty = auto-detect)
	AgentOrder     []string `yaml:"agent_order,omitempty"` // Display order for agents in TUI sidebar
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		DefaultBackend: "kiro-cli",
		Theme:          "moonbase",
		AgentsDir:      "",
		AgentOrder: []string{
			"numbuh-0", "numbuh-1", "numbuh-2", "numbuh-3", "numbuh-4", "numbuh-5",
			"numbuh-362", "numbuh-274", "numbuh-86", "numbuh-999", "numbuh-13",
			"knd-council", "sector-z", "numbuh-9",
		},
	}
}

// Path returns the config file path.
func Path() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "moonbase", "config.yaml")
}

// OldJSONPath returns the legacy JSON config path (used for migration detection).
func OldJSONPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "moonbase", "config.json")
}

// Load reads the config from disk. Returns defaults if file doesn't exist.
// Attempts migration from JSON if YAML doesn't exist but JSON does.
func Load() Config {
	cfg := DefaultConfig()

	// Try YAML first
	data, err := os.ReadFile(Path())
	if err == nil {
		if yErr := yaml.Unmarshal(data, &cfg); yErr != nil {
			fmt.Fprintf(os.Stderr, "⚠️  Config parse error (%s), using defaults: %v\n", Path(), yErr)
			return DefaultConfig()
		}
		return cfg
	}

	// YAML doesn't exist — try migration from JSON
	if migrated, mErr := MigrateFromJSON(); mErr == nil && migrated {
		// Retry load after migration
		data, err = os.ReadFile(Path())
		if err == nil {
			yaml.Unmarshal(data, &cfg)
			return cfg
		}
	}

	return cfg
}

// Save writes the config to disk as YAML.
// SECURITY: Config is written with 0600 permissions (owner read/write only).
// Config directory is created with 0700 (owner access only).
// This prevents other users on a shared system from reading preferences.
func Save(cfg Config) error {
	path := Path()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	header := []byte("# Moonbase configuration\n# https://github.com/f5508037/moonbase\n\n")
	content := append(header, data...)

	return os.WriteFile(path, content, 0o600)
}

// Show returns the config as a formatted YAML string for display.
func Show(cfg Config) string {
	data, _ := yaml.Marshal(cfg)
	return string(data)
}

// --- Internal helpers (also used by tests) ---

// marshalYAML marshals config to YAML bytes.
func marshalYAML(cfg Config) ([]byte, error) {
	return yaml.Marshal(cfg)
}

// loadFromFile loads config from a specific path.
func loadFromFile(path string) (Config, error) {
	var cfg Config
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, err
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}
