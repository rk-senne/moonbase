// Kimi backend for Moonshot AI's Kimi API.
//
// Kimi uses an OpenAI-compatible chat completions endpoint at
// https://api.moonshot.ai/v1/chat/completions with SSE streaming.
// Authentication via MOONSHOT_API_KEY environment variable.
//
// This reuses the same secure HTTP client and SSE parsing logic as the
// OpenAI backend since the wire protocol is identical.
package backend

import (
	"errors"
	"fmt"
	"os"

	"github.com/rk-senne/moonbase/internal/agents"
	"github.com/rk-senne/moonbase/internal/discovery"
)

// Kimi deploys agents via Moonshot AI's Kimi API (OpenAI-compatible).
// Configuration: MOONSHOT_API_KEY (required), KIMI_MODEL (optional, default: kimi-k3).
//
// Available models:
//   - kimi-k3 (1M context, reasoning, tool use)
//   - kimi-k2.7-code (256K context, code-focused, always-on thinking)
//   - kimi-k2.6 (256K context, configurable thinking)
//   - kimi-k2.5 (256K context)
//
// API docs: https://platform.kimi.ai/docs/api/overview
type Kimi struct{}

func (k *Kimi) Name() string   { return "kimi" }
func (k *Kimi) Available() bool { return envExists("MOONSHOT_API_KEY") }

// Deploy sends the agent prompt + task to Kimi's Chat Completions API with streaming.
//
// SECURITY TRUST BOUNDARY:
// - API key comes from environment (never stored in config/code)
// - Uses the same secure HTTP client as the OpenAI backend (TLS 1.2+, timeouts)
// - Response is parsed as SSE; malformed data is discarded
// - Error response body is size-limited to prevent OOM
func (k *Kimi) Deploy(agent agents.Agent, context *discovery.ProjectContext, task string) (string, error) {
	output, _, err := k.deployWithUsage(agent, context, task)
	return output, err
}

// DeployWithUsage returns the response plus token usage from the Kimi API.
// Implements the UsageReporter optional interface.
func (k *Kimi) DeployWithUsage(agent agents.Agent, context *discovery.ProjectContext, task string) (string, *UsageInfo, error) {
	return k.deployWithUsage(agent, context, task)
}

// DeployRawWithUsage sends a pre-composed prompt and returns token usage.
// Implements the RawUsageReporter optional interface.
func (k *Kimi) DeployRawWithUsage(composed string, task string) (string, *UsageInfo, error) {
	apiKey := os.Getenv("MOONSHOT_API_KEY")
	if apiKey == "" {
		return "", nil, fmt.Errorf("MOONSHOT_API_KEY not set")
	}

	model := os.Getenv("KIMI_MODEL")
	if model == "" {
		model = "kimi-k3"
	}

	result, usage, err := streamChatCompletion(openaiHTTPClient, "https://api.moonshot.ai/v1", apiKey, model, composed, task)
	if err != nil {
		var deployErr *DeployError
		if errors.As(err, &deployErr) {
			return result, usage, deployErr
		}
		return result, usage, fmt.Errorf("kimi: %w", err)
	}
	return result, usage, nil
}

// deployWithUsage is the shared implementation for Deploy and DeployWithUsage.
func (k *Kimi) deployWithUsage(agent agents.Agent, context *discovery.ProjectContext, task string) (string, *UsageInfo, error) {
	apiKey := os.Getenv("MOONSHOT_API_KEY")
	if apiKey == "" {
		return "", nil, fmt.Errorf("MOONSHOT_API_KEY not set")
	}

	model := os.Getenv("KIMI_MODEL")
	if model == "" {
		model = "kimi-k3"
	}

	composed := discovery.ComposePrompt(agent.Prompt, context, task)

	result, usage, err := streamChatCompletion(openaiHTTPClient, "https://api.moonshot.ai/v1", apiKey, model, composed, task)
	if err != nil {
		var deployErr *DeployError
		if errors.As(err, &deployErr) {
			return result, usage, deployErr
		}
		return result, usage, fmt.Errorf("kimi: %w", err)
	}
	return result, usage, nil
}

// Compile-time interface assertions for Kimi.
var (
	_ Backend          = (*Kimi)(nil)
	_ UsageReporter    = (*Kimi)(nil)
	_ RawUsageReporter = (*Kimi)(nil)
)
