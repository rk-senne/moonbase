package backend

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
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
	Model    string                `json:"model"`
	Messages []openaiMessage       `json:"messages"`
	Stream   bool                  `json:"stream"`
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
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("OPENAI_API_KEY not set")
	}

	baseURL := os.Getenv("OPENAI_BASE_URL")
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	// Remove trailing slash for clean URL construction
	baseURL = strings.TrimRight(baseURL, "/")

	model := os.Getenv("OPENAI_MODEL")
	if model == "" {
		model = "gpt-4o"
	}

	composed := discovery.ComposePrompt(agent.Prompt, context, task)

	body := openaiRequest{
		Model: model,
		Messages: []openaiMessage{
			{Role: "system", Content: composed},
			{Role: "user", Content: task},
		},
		Stream: true,
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("marshaling request: %w", err)
	}

	url := baseURL + "/chat/completions"
	req, err := http.NewRequest("POST", url, bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := openaiHTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("openai request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// SECURITY: Limit error body read to prevent OOM from oversized responses.
		limited := io.LimitReader(resp.Body, openaiMaxErrorBody)
		var buf bytes.Buffer
		buf.ReadFrom(limited)
		return "", &DeployError{
			StatusCode: resp.StatusCode,
			Message:    fmt.Sprintf("OpenAI API error %d: %s", resp.StatusCode, buf.String()),
		}
	}

	// Parse SSE stream — extract content deltas until [DONE] or stream close.
	var result strings.Builder
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()

		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")

		// [DONE] signals end of stream (OpenAI standard)
		if data == "[DONE]" {
			break
		}

		var event struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
				FinishReason *string `json:"finish_reason"`
			} `json:"choices"`
		}

		if err := json.Unmarshal([]byte(data), &event); err != nil {
			continue // Discard malformed events
		}

		if len(event.Choices) > 0 {
			if event.Choices[0].Delta.Content != "" {
				result.WriteString(event.Choices[0].Delta.Content)
			}
			// Some endpoints use finish_reason: stop instead of [DONE]
			if event.Choices[0].FinishReason != nil && *event.Choices[0].FinishReason == "stop" {
				break
			}
		}
	}

	// Handle scanner errors (connection close without [DONE] — Ollama compat)
	if err := scanner.Err(); err != nil && result.Len() == 0 {
		return "", fmt.Errorf("openai stream read error: %w", err)
	}

	return result.String(), nil
}
