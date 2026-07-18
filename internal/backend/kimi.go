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
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/f5508037/moonbase/internal/agents"
	"github.com/f5508037/moonbase/internal/discovery"
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
	apiKey := os.Getenv("MOONSHOT_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("MOONSHOT_API_KEY not set")
	}

	model := os.Getenv("KIMI_MODEL")
	if model == "" {
		model = "kimi-k3"
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

	url := "https://api.moonshot.ai/v1/chat/completions"
	req, err := http.NewRequest("POST", url, bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	// Reuse the same secure HTTP client as the OpenAI backend
	resp, err := openaiHTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("kimi request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		limited := io.LimitReader(resp.Body, openaiMaxErrorBody)
		var buf bytes.Buffer
		buf.ReadFrom(limited)
		return "", &DeployError{
			StatusCode: resp.StatusCode,
			Message:    fmt.Sprintf("Kimi API error %d: %s", resp.StatusCode, buf.String()),
		}
	}

	// Parse SSE stream — same format as OpenAI (data: {json}\ndata: [DONE])
	var result strings.Builder
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()

		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")

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
			continue
		}

		if len(event.Choices) > 0 {
			if event.Choices[0].Delta.Content != "" {
				result.WriteString(event.Choices[0].Delta.Content)
			}
			if event.Choices[0].FinishReason != nil && *event.Choices[0].FinishReason == "stop" {
				break
			}
		}
	}

	if err := scanner.Err(); err != nil && result.Len() == 0 {
		return "", fmt.Errorf("kimi stream read error: %w", err)
	}

	return result.String(), nil
}
