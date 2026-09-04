package backend

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/rk-senne/moonbase/internal/agents"
	"github.com/rk-senne/moonbase/internal/chat"
	clip "github.com/rk-senne/moonbase/internal/clipboard"
	"github.com/rk-senne/moonbase/internal/discovery"
)

// Codex deploys agents via OpenAI Codex CLI
type Codex struct{}

func (c *Codex) Name() string    { return "codex" }
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

// OpenAI deploys agents via OpenAI-compatible Chat Completions API with SSE streaming.
// Compatible with OpenAI, Azure OpenAI, LM Studio, and Ollama OpenAI compat mode.
// Configuration: OPENAI_API_KEY (required), OPENAI_BASE_URL (optional), OPENAI_MODEL (optional).
type OpenAI struct{}

func (o *OpenAI) Name() string    { return "openai" }
func (o *OpenAI) Available() bool { return envHasValue("OPENAI_API_KEY") }

// Anthropic deploys agents via Anthropic Messages API with streaming.
// Uses the same HTTP streaming code as the comms system (internal/chat).
type Anthropic struct{}

func (a *Anthropic) Name() string    { return "anthropic" }
func (a *Anthropic) Available() bool { return envHasValue("ANTHROPIC_API_KEY") }

// Deploy sends the agent prompt + task to Anthropic's Messages API via streaming,
// collects all response chunks, and returns the full response text.
func (a *Anthropic) Deploy(agent agents.Agent, context *discovery.ProjectContext, task string) (string, error) {
	output, _, err := a.deployWithUsage(agent, context, task)
	return output, err
}

// DeployWithUsage returns the response plus token usage from the Anthropic API.
// Implements the UsageReporter optional interface.
func (a *Anthropic) DeployWithUsage(agent agents.Agent, context *discovery.ProjectContext, task string) (string, *UsageInfo, error) {
	return a.deployWithUsage(agent, context, task)
}

// deployWithUsage is the shared implementation that streams and captures usage.
func (a *Anthropic) deployWithUsage(agent agents.Agent, context *discovery.ProjectContext, task string) (string, *UsageInfo, error) {
	composed := discovery.ComposePrompt(agent.Prompt, context, task)

	// Build a conversation with the system prompt and task as the user message
	conv := chat.NewConversation(agent.Name, composed)
	conv.Add(chat.RoleUser, task)

	// Stream the response and collect all chunks
	ch := chat.Stream(conv)
	var result strings.Builder
	var usage *UsageInfo
	for chunk := range ch {
		if chunk.Err != nil {
			return result.String(), nil, fmt.Errorf("anthropic streaming error: %w", chunk.Err)
		}
		if chunk.Done {
			if chunk.Usage != nil {
				model := os.Getenv("ANTHROPIC_MODEL")
				if model == "" {
					model = "claude-sonnet-4-20250514"
				}
				total := chunk.Usage.InputTokens + chunk.Usage.OutputTokens
				usage = &UsageInfo{
					PromptTokens:     chunk.Usage.InputTokens,
					CompletionTokens: chunk.Usage.OutputTokens,
					TotalTokens:      total,
					Model:            model,
				}
			}
			break
		}
		result.WriteString(chunk.Text)
	}

	return result.String(), usage, nil
}

// Compile-time interface assertion for Anthropic.
var _ UsageReporter = (*Anthropic)(nil)

// Ollama deploys agents via local Ollama
type Ollama struct{}

func (o *Ollama) Name() string    { return "ollama" }
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

func (c *Clipboard) Name() string    { return "clipboard" }
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
// - API keys (ANTHROPIC_API_KEY, OPENAI_API_KEY, MOONSHOT_API_KEY) — needed by backends
// - OPENAI_BASE_URL — needed for OpenAI-compatible endpoints (Azure, LM Studio, Ollama)
// - OPENAI_MODEL — needed for model selection on OpenAI-compatible backends
// - KIMI_MODEL — needed for model selection on Kimi/Moonshot backend
// - OLLAMA_HOST — needed for custom ollama endpoint
//
// NEVER add: AWS_*, DATABASE_*, GITHUB_TOKEN, SSH_*, or other sensitive vars.
// If a new backend needs additional env vars, add them explicitly here.
func SafeEnv() []string {
	allowed := []string{"HOME", "PATH", "USER", "TERM", "LANG", "SHELL",
		"ANTHROPIC_API_KEY", "OPENAI_API_KEY", "OPENAI_BASE_URL", "OPENAI_MODEL",
		"MOONSHOT_API_KEY", "KIMI_MODEL",
		"OLLAMA_HOST"}
	var env []string
	for _, key := range allowed {
		if val, ok := os.LookupEnv(key); ok {
			env = append(env, key+"="+val)
		}
	}
	return env
}

// envExists reports whether an environment variable is set, regardless of value.
// An exported-but-empty variable counts as existing — see envHasValue when what
// you need is a usable value.
func envExists(key string) bool {
	_, ok := os.LookupEnv(key)
	return ok
}

// envHasValue reports whether an environment variable is set to a non-empty,
// non-whitespace value.
//
// Backend availability must use this rather than envExists. An exported-but-empty
// API key (MOONSHOT_API_KEY= in a shell profile, or a blank CI secret) can never
// authenticate, so reporting the backend as available made auto-detection select
// it and then fail every deploy with "not set". Failing closed here keeps
// Available() consistent with the deploy-time credential check.
func envHasValue(key string) bool {
	return strings.TrimSpace(os.Getenv(key)) != ""
}
