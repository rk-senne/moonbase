// Shared SSE streaming logic for OpenAI-compatible chat completion APIs.
//
// Both the OpenAI and Kimi backends use identical SSE wire protocols for
// streaming chat completions. This file extracts that shared logic to avoid
// duplication and ensures both backends benefit from the response size bound.
package backend

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// maxResponseSize is the maximum accumulated SSE response size (10 MB).
// Prevents unbounded memory growth from pathologically large streamed responses.
const maxResponseSize = 10 << 20

// streamChatCompletion sends a streaming chat completion request to an
// OpenAI-compatible endpoint and collects the full response text.
//
// Compatible with: OpenAI, Azure OpenAI, LM Studio, Ollama OpenAI compat,
// Moonshot/Kimi.
//
// The accumulated response is bounded by maxResponseSize. If the response
// exceeds this limit, streaming stops and a size-exceeded error is returned.
func streamChatCompletion(client *http.Client, baseURL, apiKey, model, composed, task string) (string, error) {
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

	url := strings.TrimRight(baseURL, "/") + "/chat/completions"
	req, err := http.NewRequest("POST", url, bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		limited := io.LimitReader(resp.Body, openaiMaxErrorBody)
		var buf bytes.Buffer
		buf.ReadFrom(limited)
		return "", &DeployError{
			StatusCode: resp.StatusCode,
			Message:    fmt.Sprintf("API error %d: %s", resp.StatusCode, buf.String()),
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
			content := event.Choices[0].Delta.Content
			if content != "" {
				if result.Len()+len(content) > maxResponseSize {
					return result.String(), fmt.Errorf("response exceeded %d bytes", maxResponseSize)
				}
				result.WriteString(content)
			}
			// Some endpoints use finish_reason: stop instead of [DONE]
			if event.Choices[0].FinishReason != nil && *event.Choices[0].FinishReason == "stop" {
				break
			}
		}
	}

	// Handle scanner errors (connection close without [DONE] — Ollama compat)
	if err := scanner.Err(); err != nil && result.Len() == 0 {
		return "", fmt.Errorf("stream read error: %w", err)
	}

	return result.String(), nil
}
