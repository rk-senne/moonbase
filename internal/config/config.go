package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config holds moonbase user preferences.
// No API keys or secrets — those come from environment variables.
type Config struct {
	DefaultBackend string   `yaml:"default_backend"`
	Theme          string   `yaml:"theme"`
	AgentsDir      string   `yaml:"agents_dir,omitempty"` // empty = auto-detect
	AgentOrder     []string `yaml:"agent_order,omitempty"`
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

// OldJSONPath returns the legacy JSON config path (for migration).
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
func Save(cfg Config) error {
	path := Path()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	header := []byte("# Moonbase configuration\n# https://github.com/f5508037/moonbase\n\n")
	content := append(header, data...)

	return os.WriteFile(path, content, 0o644)
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
