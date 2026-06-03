package backend

import (
	"os/exec"

	"github.com/f5508037/moonbase/internal/agents"
)

// Kiro deploys agents via kiro-cli
type Kiro struct{}

func (k *Kiro) Name() string      { return "kiro-cli" }
func (k *Kiro) Available() bool    { _, err := exec.LookPath("kiro-cli"); return err == nil }
func (k *Kiro) Deploy(agent agents.Agent, task string) error { return nil } // TODO

// Codex deploys agents via OpenAI Codex CLI
type Codex struct{}

func (c *Codex) Name() string      { return "codex" }
func (c *Codex) Available() bool    { _, err := exec.LookPath("codex"); return err == nil }
func (c *Codex) Deploy(agent agents.Agent, task string) error { return nil } // TODO

// OpenAI deploys agents via OpenAI API
type OpenAI struct{}

func (o *OpenAI) Name() string      { return "openai" }
func (o *OpenAI) Available() bool    { return envExists("OPENAI_API_KEY") }
func (o *OpenAI) Deploy(agent agents.Agent, task string) error { return nil } // TODO

// Anthropic deploys agents via Anthropic API
type Anthropic struct{}

func (a *Anthropic) Name() string      { return "anthropic" }
func (a *Anthropic) Available() bool    { return envExists("ANTHROPIC_API_KEY") }
func (a *Anthropic) Deploy(agent agents.Agent, task string) error { return nil } // TODO

// Ollama deploys agents via local Ollama
type Ollama struct{}

func (o *Ollama) Name() string      { return "ollama" }
func (o *Ollama) Available() bool    { _, err := exec.LookPath("ollama"); return err == nil }
func (o *Ollama) Deploy(agent agents.Agent, task string) error { return nil } // TODO

// Clipboard copies the agent prompt to clipboard as a fallback
type Clipboard struct{}

func (c *Clipboard) Name() string      { return "clipboard" }
func (c *Clipboard) Available() bool    { return true } // always available
func (c *Clipboard) Deploy(agent agents.Agent, task string) error { return nil } // TODO
