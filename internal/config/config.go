// Package config manages moonbase user preferences stored as YAML.
// No secrets or API keys are stored in config — those come exclusively from
// environment variables. Supports automatic migration from the legacy JSON format.
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/rk-senne/moonbase/internal/agents"
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

	// Pipeline execution options (Enhancement 1, 2, 7)
	TrustTools      bool   `yaml:"trust_tools,omitempty"`      // Pass --trust-all-tools to kiro-cli (enables headless execution)
	PipelineBackend string `yaml:"pipeline_backend,omitempty"` // Backend for analysis phases (anthropic/openai); kiro-cli used for implementation
	FastThreshold   int    `yaml:"fast_threshold,omitempty"`   // Word count below which --fast mode auto-engages (0=disabled)

	// Pipeline orchestration options.
	// These control resilience and observability for multi-phase mission execution.
	// Derived from production agent patterns (LangGraph state machines, AWS AgentCore).
	PhaseTimeout  int  `yaml:"phase_timeout_seconds,omitempty"` // Max seconds per phase (default 300 = 5 min)
	MaxOutputSize int  `yaml:"max_output_size,omitempty"`       // Max output bytes per phase (default 100000)
	EnableTrace   bool `yaml:"enable_trace,omitempty"`          // Enable trace ID generation for pipeline runs
	MaxRetries    int  `yaml:"max_retries,omitempty"`           // Max retries per phase before failure (default 1)

	// cmux integration (manaflow-ai/cmux macOS terminal for AI agents).
	UseCmux bool `yaml:"use_cmux,omitempty"` // Auto-enable cmux features when true (notifications, split panes)
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		DefaultBackend:  "kiro-cli",
		Theme:           "moonbase",
		AgentsDir:       "",
		TrustTools:      true, // Enable headless execution by default
		PipelineBackend: "",   // Empty = use default_backend for all phases
		FastThreshold:   0,    // 0 = disabled (user must pass --fast explicitly)
		PhaseTimeout:    300,  // 5 minutes per phase
		MaxOutputSize:   100000, // 100KB max output per phase
		EnableTrace:     true,   // Trace IDs enabled by default for observability
		MaxRetries:      1,      // One retry per phase before failure
		AgentOrder: agents.DefaultAgentOrder,
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

	header := []byte("# Moonbase configuration\n# https://github.com/rk-senne/moonbase\n\n")
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
