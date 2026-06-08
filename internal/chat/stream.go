package chat

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
)

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

// Stream sends a conversation to Anthropic and returns a channel of text chunks
func Stream(conv *Conversation) <-chan StreamChunk {
	ch := make(chan StreamChunk, 64)

	go func() {
		defer close(ch)

		apiKey := os.Getenv("ANTHROPIC_API_KEY")
		if apiKey == "" {
			ch <- StreamChunk{Err: fmt.Errorf("ANTHROPIC_API_KEY not set")}
			return
		}

		model := os.Getenv("ANTHROPIC_MODEL")
		if model == "" {
			model = "claude-sonnet-4-20250514"
		}

		body := anthropicRequest{
			Model:     model,
			MaxTokens: 4096,
			System:    conv.System,
			Messages:  conv.APIMessages(),
			Stream:    true,
		}

		payload, _ := json.Marshal(body)

		req, err := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", bytes.NewReader(payload))
		if err != nil {
			ch <- StreamChunk{Err: err}
			return
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("x-api-key", apiKey)
		req.Header.Set("anthropic-version", "2023-06-01")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			ch <- StreamChunk{Err: err}
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			var buf bytes.Buffer
			buf.ReadFrom(resp.Body)
			ch <- StreamChunk{Err: fmt.Errorf("API error %d: %s", resp.StatusCode, buf.String())}
			return
		}

		// Parse SSE stream
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
				continue
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
