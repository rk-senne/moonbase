package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Config struct {
	AgentsDir      string            `json:"agentsDir"`
	DefaultBackend string            `json:"defaultBackend"`
	Theme          string            `json:"theme"`
	Backends       map[string]Backend `json:"backends"`
}

type Backend struct {
	APIKey string `json:"apiKey,omitempty"`
	Model  string `json:"model,omitempty"`
	Binary string `json:"binary,omitempty"`
}

func DefaultConfig() Config {
	return Config{
		AgentsDir:      "./agents",
		DefaultBackend: "kiro-cli",
		Theme:          "moonbase",
		Backends: map[string]Backend{
			"kiro":      {Binary: "kiro-cli"},
			"openai":    {Model: "gpt-4o"},
			"anthropic": {Model: "claude-sonnet-4-20250514"},
			"ollama":    {Model: "llama3.1"},
		},
	}
}

func ConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "moonbase", "config.json")
}

func Load() Config {
	cfg := DefaultConfig()
	data, err := os.ReadFile(ConfigPath())
	if err != nil {
		return cfg
	}
	json.Unmarshal(data, &cfg)
	return cfg
}

func Save(cfg Config) error {
	path := ConfigPath()
	os.MkdirAll(filepath.Dir(path), 0755)
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
