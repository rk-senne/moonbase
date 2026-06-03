package backend

import "github.com/f5508037/moonbase/internal/agents"

// Backend is the interface all AI tool integrations implement
type Backend interface {
	Name() string
	Available() bool
	Deploy(agent agents.Agent, task string) error
}

// DetectAll checks which backends are available on this system
func DetectAll() []Backend {
	all := []Backend{
		&Kiro{},
		&Codex{},
		&OpenAI{},
		&Anthropic{},
		&Ollama{},
		&Clipboard{},
	}

	var available []Backend
	for _, b := range all {
		available = append(available, b)
	}
	return available
}
