// Package backend provides AI tool integrations for deploying agents.
// It supports multiple backends (kiro-cli, codex, ollama, openai, anthropic, kimi)
// with automatic detection and a clipboard fallback for universal compatibility.
package backend

import (
	"github.com/rk-senne/moonbase/internal/agents"
	"github.com/rk-senne/moonbase/internal/discovery"
	"github.com/rk-senne/moonbase/internal/logging"
)

// Backend is the interface all AI tool integrations implement.
type Backend interface {
	Name() string
	Available() bool
	// Deploy sends an agent with composed context to the AI backend.
	// Returns the agent's output or an error.
	Deploy(agent agents.Agent, context *discovery.ProjectContext, task string) (string, error)
}

// RawDeployer is an optional interface backends can implement to accept a
// pre-composed prompt without re-running ComposePrompt. This avoids double-
// composition when the caller (e.g., the mission pipeline) has already built
// the full prompt including project context, steering, and per-phase input.
type RawDeployer interface {
	DeployRaw(composed string, task string) (string, error)
}

// DetectAll checks which backends are available on this system.
func DetectAll() []Backend {
	return DetectAllWithConfig(true) // default: trust tools enabled
}

// DetectAllWithConfig checks which backends are available, with configurable trust.
func DetectAllWithConfig(trustTools bool) []Backend {
	all := []Backend{
		&Kiro{TrustTools: trustTools},
		&Codex{},
		&OpenAI{},
		&Anthropic{},
		&Kimi{},
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
			if logging.Logger != nil {
				logging.Logger.Info("backend selected", "name", b.Name())
			}
			return b
		}
	}
	// Fallback to clipboard (always available)
	if logging.Logger != nil {
		logging.Logger.Info("backend selected", "name", "clipboard", "reason", "no other backend available")
	}
	return &Clipboard{}
}
