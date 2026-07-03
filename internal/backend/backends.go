package backend

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/f5508037/moonbase/internal/agents"
	"github.com/f5508037/moonbase/internal/chat"
	clip "github.com/f5508037/moonbase/internal/clipboard"
	"github.com/f5508037/moonbase/internal/discovery"
)

// Kiro deploys agents via kiro-cli
type Kiro struct{}

func (k *Kiro) Name() string   { return "kiro-cli" }
func (k *Kiro) Available() bool { _, err := exec.LookPath("kiro-cli"); return err == nil }

func (k *Kiro) Deploy(agent agents.Agent, context *discovery.ProjectContext, task string) (string, error) {
	composed := discovery.ComposePrompt(agent.Prompt, context, task)

	// Write the composed prompt to a temp file for kiro-cli to consume
	tmpFile, err := os.CreateTemp("", "moonbase-prompt-*.md")
	if err != nil {
		return "", fmt.Errorf("creating temp prompt file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(composed); err != nil {
		tmpFile.Close()
		return "", fmt.Errorf("writing prompt: %w", err)
	}
	tmpFile.Close()

	// Execute kiro-cli with the prompt
	cmd := exec.Command("kiro-cli", "chat",
		"--system-prompt", tmpFile.Name(),
		"--message", task,
	)
	// SECURITY: SafeEnv prevents leaking user's full environment to child process.
	cmd.Env = SafeEnv()

	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("kiro-cli execution: %w\noutput: %s", err, string(output))
	}

	return string(output), nil
}

// Codex deploys agents via OpenAI Codex CLI
type Codex struct{}

func (c *Codex) Name() string   { return "codex" }
func (c *Codex) Available() bool { _, err := exec.LookPath("codex"); return err == nil }

func (c *Codex) Deploy(agent agents.Agent, context *discovery.ProjectContext, task string) (string, error) {
	composed := discovery.ComposePrompt(agent.Prompt, context, task)

	cmd := exec.Command("codex",
		"--system-prompt", composed,
		task,
	)
	// SECURITY: SafeEnv prevents leaking user's full environment to child process.
	cmd.Env = SafeEnv()

	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("codex execution: %w", err)
	}
	return string(output), nil
}

// OpenAI deploys agents via OpenAI API (placeholder — requires full API client)
type OpenAI struct{}

func (o *OpenAI) Name() string   { return "openai" }
func (o *OpenAI) Available() bool { return envExists("OPENAI_API_KEY") }

func (o *OpenAI) Deploy(agent agents.Agent, context *discovery.ProjectContext, task string) (string, error) {
	return "", fmt.Errorf("openai backend not yet wired — requires OpenAI streaming client. Use anthropic, kiro-cli, or clipboard instead")
}

// Anthropic deploys agents via Anthropic Messages API with streaming.
// Uses the same HTTP streaming code as the comms system (internal/chat).
type Anthropic struct{}

func (a *Anthropic) Name() string   { return "anthropic" }
func (a *Anthropic) Available() bool { return envExists("ANTHROPIC_API_KEY") }

// Deploy sends the agent prompt + task to Anthropic's Messages API via streaming,
// collects all response chunks, and returns the full response text.
func (a *Anthropic) Deploy(agent agents.Agent, context *discovery.ProjectContext, task string) (string, error) {
	composed := discovery.ComposePrompt(agent.Prompt, context, task)

	// Build a conversation with the system prompt and task as the user message
	conv := chat.NewConversation(agent.Name, composed)
	conv.Add(chat.RoleUser, task)

	// Stream the response and collect all chunks
	ch := chat.Stream(conv)
	var result strings.Builder
	for chunk := range ch {
		if chunk.Err != nil {
			return result.String(), fmt.Errorf("anthropic streaming error: %w", chunk.Err)
		}
		if chunk.Done {
			break
		}
		result.WriteString(chunk.Text)
	}

	return result.String(), nil
}

// Ollama deploys agents via local Ollama
type Ollama struct{}

func (o *Ollama) Name() string   { return "ollama" }
func (o *Ollama) Available() bool { _, err := exec.LookPath("ollama"); return err == nil }

func (o *Ollama) Deploy(agent agents.Agent, context *discovery.ProjectContext, task string) (string, error) {
	composed := discovery.ComposePrompt(agent.Prompt, context, task)

	cmd := exec.Command("ollama", "run", "llama3.1")
	cmd.Stdin = strings.NewReader(composed)
	// SECURITY: SafeEnv prevents leaking user's full environment to child process.
	cmd.Env = SafeEnv()

	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("ollama execution: %w", err)
	}
	return string(output), nil
}

// Clipboard copies the composed prompt to clipboard as a universal fallback.
type Clipboard struct{}

func (c *Clipboard) Name() string   { return "clipboard" }
func (c *Clipboard) Available() bool { return clip.Available() }

func (c *Clipboard) Deploy(agent agents.Agent, context *discovery.ProjectContext, task string) (string, error) {
	composed := discovery.ComposePrompt(agent.Prompt, context, task)

	if err := clip.Copy(composed); err != nil {
		return "", fmt.Errorf("clipboard copy failed: %w", err)
	}

	summary := fmt.Sprintf("Copied to clipboard (%d chars). Agent: %s (%s). Task: %s",
		len(composed), agent.Name, agent.Role, truncate(task, 80))
	return summary, nil
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// SafeEnv returns a filtered environment containing only variables needed for
// backend execution. This prevents leaking sensitive env vars to child processes.
//
// SECURITY TRUST BOUNDARY:
// All child process execution (kiro-cli, codex, ollama) MUST use SafeEnv().
// This is the primary defense against env var leakage. The allowlist contains:
// - System vars (HOME, PATH, USER, TERM, LANG, SHELL) — needed for basic operation
// - API keys (ANTHROPIC_API_KEY, OPENAI_API_KEY) — needed by backends
// - OLLAMA_HOST — needed for custom ollama endpoint
//
// NEVER add: AWS_*, DATABASE_*, GITHUB_TOKEN, SSH_*, or other sensitive vars.
// If a new backend needs additional env vars, add them explicitly here.
func SafeEnv() []string {
	allowed := []string{"HOME", "PATH", "USER", "TERM", "LANG", "SHELL",
		"ANTHROPIC_API_KEY", "OPENAI_API_KEY", "OLLAMA_HOST"}
	var env []string
	for _, key := range allowed {
		if val, ok := os.LookupEnv(key); ok {
			env = append(env, key+"="+val)
		}
	}
	return env
}

func envExists(key string) bool {
	_, ok := os.LookupEnv(key)
	return ok
}
