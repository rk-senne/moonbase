package backend

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/rk-senne/moonbase/internal/agents"
	"github.com/rk-senne/moonbase/internal/discovery"
)

// openaiHTTPClient is configured with explicit TLS 1.2 minimum, timeouts on all
// phases of the connection, matching the security pattern from internal/chat.
//
// SECURITY: Prevents downgrade attacks (TLS 1.2 min), connection exhaustion (timeouts).
var openaiHTTPClient = &http.Client{
	Timeout: 300 * time.Second, // overall request timeout (streaming may be long)
	Transport: &http.Transport{
		TLSClientConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
		},
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second,
		IdleConnTimeout:       90 * time.Second,
		MaxIdleConns:          10,
		MaxIdleConnsPerHost:   2,
	},
}

// openaiMaxErrorBody limits how much of an error response body we read (1MB).
const openaiMaxErrorBody = 1 << 20

// openaiRequest is the request body for POST /v1/chat/completions.
type openaiRequest struct {
	Model         string          `json:"model"`
	Messages      []openaiMessage `json:"messages"`
	Stream        bool            `json:"stream"`
	StreamOptions *streamOptions  `json:"stream_options,omitempty"`
}

type openaiMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Deploy sends the agent prompt + task to the OpenAI-compatible Chat Completions API
// with streaming enabled, collects all response chunks, and returns the full response.
//
// Compatible with: OpenAI, Azure OpenAI (set OPENAI_BASE_URL to full path prefix),
// LM Studio, Ollama OpenAI compat mode.
//
// SECURITY TRUST BOUNDARY:
// - API key comes from environment (never stored in config/code)
// - Base URL is configurable for flexibility (user's responsibility for HTTPS)
// - Response is parsed as SSE; malformed data is discarded
// - Error response body is size-limited to prevent OOM
func (o *OpenAI) Deploy(agent agents.Agent, context *discovery.ProjectContext, task string) (string, error) {
	output, _, err := o.deployWithUsage(agent, context, task)
	return output, err
}

// DeployWithUsage returns the response plus token usage from the OpenAI API.
// Implements the UsageReporter optional interface.
func (o *OpenAI) DeployWithUsage(agent agents.Agent, context *discovery.ProjectContext, task string) (string, *UsageInfo, error) {
	return o.deployWithUsage(agent, context, task)
}

// DeployRawWithUsage sends a pre-composed prompt and returns token usage.
// Implements the RawUsageReporter optional interface.
func (o *OpenAI) DeployRawWithUsage(composed string, task string) (string, *UsageInfo, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return "", nil, fmt.Errorf("OPENAI_API_KEY not set")
	}

	baseURL := os.Getenv("OPENAI_BASE_URL")
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}

	model := os.Getenv("OPENAI_MODEL")
	if model == "" {
		model = "gpt-4o"
	}

	result, usage, err := streamChatCompletion(openaiHTTPClient, baseURL, apiKey, model, composed, task)
	if err != nil {
		var deployErr *DeployError
		if errors.As(err, &deployErr) {
			return result, usage, deployErr
		}
		return result, usage, fmt.Errorf("openai: %w", err)
	}
	return result, usage, nil
}

// deployWithUsage is the shared implementation for Deploy and DeployWithUsage.
func (o *OpenAI) deployWithUsage(agent agents.Agent, context *discovery.ProjectContext, task string) (string, *UsageInfo, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return "", nil, fmt.Errorf("OPENAI_API_KEY not set")
	}

	baseURL := os.Getenv("OPENAI_BASE_URL")
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}

	model := os.Getenv("OPENAI_MODEL")
	if model == "" {
		model = "gpt-4o"
	}

	composed := discovery.ComposePrompt(agent.Prompt, context, task)

	result, usage, err := streamChatCompletion(openaiHTTPClient, baseURL, apiKey, model, composed, task)
	if err != nil {
		// Preserve *DeployError type for callers that check status codes.
		var deployErr *DeployError
		if errors.As(err, &deployErr) {
			return result, usage, deployErr
		}
		return result, usage, fmt.Errorf("openai: %w", err)
	}
	return result, usage, nil
}

// Compile-time interface assertions for OpenAI.
var (
	_ Backend          = (*OpenAI)(nil)
	_ UsageReporter    = (*OpenAI)(nil)
	_ RawUsageReporter = (*OpenAI)(nil)
)
