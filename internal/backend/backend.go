package backend

import (
	"github.com/f5508037/moonbase/internal/agents"
	"github.com/f5508037/moonbase/internal/discovery"
)

// Backend is the interface all AI tool integrations implement.
type Backend interface {
	Name() string
	Available() bool
	// Deploy sends an agent with composed context to the AI backend.
	// Returns the agent's output or an error.
	Deploy(agent agents.Agent, context *discovery.ProjectContext, task string) (string, error)
}

// DetectAll checks which backends are available on this system.
func DetectAll() []Backend {
	all := []Backend{
		&Kiro{},
		&Codex{},
		&OpenAI{},
		&Anthropic{},
		&Ollama{},
		&Clipboard{},
	}
	return all
}

// DetectAvailable returns only the backends that are currently available.
func DetectAvailable() []Backend {
	var available []Backend
	for _, b := range DetectAll() {
		if b.Available() {
			available = append(available, b)
		}
	}
	return available
}

// Preferred returns the best available backend (kiro > codex > ollama > clipboard).
func Preferred() Backend {
	priority := DetectAll()
	for _, b := range priority {
		if b.Available() && b.Name() != "clipboard" {
			return b
		}
	}
	// Fallback to clipboard (always available)
	return &Clipboard{}
}
