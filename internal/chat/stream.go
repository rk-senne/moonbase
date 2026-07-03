package chat

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
)

// DefaultMaxTokens is the maximum number of tokens the model will generate per response.
const DefaultMaxTokens = 4096

// maxErrorBodySize limits how much of an error response body we read (1MB).
// Prevents memory exhaustion from malicious or malformed API responses.
const maxErrorBodySize = 1 << 20

type StreamChunk struct {
	Text string
	Done bool
	Err  error
}

type anthropicRequest struct {
	Model     string              `json:"model"`
	MaxTokens int                 `json:"max_tokens"`
	System    string              `json:"system"`
	Messages  []map[string]string `json:"messages"`
	Stream    bool                `json:"stream"`
}

// SECURITY: httpClient is configured with explicit TLS 1.2 minimum, timeouts on all
// phases of the connection, and enforced HTTPS via TLS config. This prevents:
// - Downgrade attacks (TLS 1.2 minimum)
// - Slowloris/connection exhaustion (timeouts)
// - The client only connects to api.anthropic.com over HTTPS (URL hardcoded below)
var httpClient = &http.Client{
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

// Stream sends a conversation to Anthropic and returns a channel of text chunks.
//
// SECURITY TRUST BOUNDARY:
// - API key comes from environment (never stored in config/code)
// - Connection is HTTPS-only to api.anthropic.com (hardcoded, not configurable)
// - Response is parsed as SSE; malformed data is discarded (not executed)
// - Error response body is size-limited to prevent OOM
func Stream(conv *Conversation) <-chan StreamChunk {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		ch := make(chan StreamChunk, 1)
		ch <- StreamChunk{Err: fmt.Errorf("ANTHROPIC_API_KEY not set")}
		close(ch)
		return ch
	}

	model := os.Getenv("ANTHROPIC_MODEL")
	if model == "" {
		model = "claude-sonnet-4-20250514"
	}

	// SECURITY: URL is hardcoded HTTPS — no user-controlled endpoint.
	return streamFrom("https://api.anthropic.com/v1/messages", apiKey, model, conv, httpClient)
}

// streamFrom performs the actual SSE streaming against the given URL.
// Extracted for testability — tests can pass an httptest URL and custom client.
func streamFrom(url, apiKey, model string, conv *Conversation, client *http.Client) <-chan StreamChunk {
	ch := make(chan StreamChunk, 64)

	go func() {
		defer close(ch)

		body := anthropicRequest{
			Model:     model,
			MaxTokens: DefaultMaxTokens,
			System:    conv.System,
			Messages:  conv.APIMessages(),
			Stream:    true,
		}

		payload, _ := json.Marshal(body)

		req, err := http.NewRequest("POST", url, bytes.NewReader(payload))
		if err != nil {
			ch <- StreamChunk{Err: err}
			return
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("x-api-key", apiKey)
		req.Header.Set("anthropic-version", "2023-06-01")

		resp, err := client.Do(req)
		if err != nil {
			ch <- StreamChunk{Err: err}
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			// SECURITY: Limit error body read to prevent OOM from oversized responses.
			limited := io.LimitReader(resp.Body, maxErrorBodySize)
			var buf bytes.Buffer
			buf.ReadFrom(limited)
			ch <- StreamChunk{Err: fmt.Errorf("API error %d: %s", resp.StatusCode, buf.String())}
			return
		}

		// Parse SSE stream — only extract text content, discard anything else.
		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()

			if !strings.HasPrefix(line, "data: ") {
				continue
			}

			data := strings.TrimPrefix(line, "data: ")

			if data == "[DONE]" {
				ch <- StreamChunk{Done: true}
				return
			}

			var event struct {
				Type  string `json:"type"`
				Delta struct {
					Type string `json:"type"`
					Text string `json:"text"`
				} `json:"delta"`
			}

			if err := json.Unmarshal([]byte(data), &event); err != nil {
				continue // SECURITY: Discard malformed events, don't propagate parse errors
			}

			if event.Type == "content_block_delta" && event.Delta.Text != "" {
				ch <- StreamChunk{Text: event.Delta.Text}
			}

			if event.Type == "message_stop" {
				ch <- StreamChunk{Done: true}
				return
			}
		}

		ch <- StreamChunk{Done: true}
	}()

	return ch
}
