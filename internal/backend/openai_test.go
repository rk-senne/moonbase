package backend

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/rk-senne/moonbase/internal/agents"
	"github.com/rk-senne/moonbase/internal/discovery"
)

func TestOpenAI_Deploy_StreamsResponse(t *testing.T) {
	// Mock SSE server that returns streamed chunks
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Validate request
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/chat/completions" {
			t.Errorf("expected /chat/completions, got %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-key-123" {
			t.Errorf("unexpected auth header: %s", r.Header.Get("Authorization"))
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("unexpected content-type: %s", r.Header.Get("Content-Type"))
		}

		// Validate request body
		var reqBody openaiRequest
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if reqBody.Model != "gpt-4o-test" {
			t.Errorf("expected model gpt-4o-test, got %s", reqBody.Model)
		}
		if !reqBody.Stream {
			t.Error("expected stream: true")
		}
		if len(reqBody.Messages) != 2 {
			t.Fatalf("expected 2 messages, got %d", len(reqBody.Messages))
		}
		if reqBody.Messages[0].Role != "system" {
			t.Errorf("expected first message role=system, got %s", reqBody.Messages[0].Role)
		}
		if reqBody.Messages[1].Role != "user" {
			t.Errorf("expected second message role=user, got %s", reqBody.Messages[1].Role)
		}

		// Write SSE response
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		chunks := []string{"Hello", " World", "!"}
		for _, chunk := range chunks {
			data := fmt.Sprintf(`{"choices":[{"delta":{"content":"%s"},"finish_reason":null}]}`, chunk)
			fmt.Fprintf(w, "data: %s\n\n", data)
		}
		// Final chunk with finish_reason
		fmt.Fprintf(w, "data: [DONE]\n\n")
	}))
	defer server.Close()

	// Set env vars for test
	t.Setenv("OPENAI_API_KEY", "test-key-123")
	t.Setenv("OPENAI_BASE_URL", server.URL)
	t.Setenv("OPENAI_MODEL", "gpt-4o-test")

	o := &OpenAI{}
	agent := agents.Agent{
		Name:   "test-agent",
		Role:   "Test",
		Prompt: "You are a test agent.",
	}
	ctx := &discovery.ProjectContext{}

	output, err := o.Deploy(agent, ctx, "say hello")
	if err != nil {
		t.Fatalf("Deploy failed: %v", err)
	}

	if output != "Hello World!" {
		t.Errorf("expected 'Hello World!', got: %q", output)
	}
}

func TestOpenAI_Deploy_HandlesFinishReason(t *testing.T) {
	// Some endpoints use finish_reason: stop instead of [DONE]
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		stop := "stop"
		chunks := []struct {
			content      string
			finishReason *string
		}{
			{"Part 1", nil},
			{" Part 2", &stop},
		}
		for _, chunk := range chunks {
			fr := "null"
			if chunk.finishReason != nil {
				fr = fmt.Sprintf(`"%s"`, *chunk.finishReason)
			}
			data := fmt.Sprintf(`{"choices":[{"delta":{"content":"%s"},"finish_reason":%s}]}`, chunk.content, fr)
			fmt.Fprintf(w, "data: %s\n\n", data)
		}
	}))
	defer server.Close()

	t.Setenv("OPENAI_API_KEY", "test-key")
	t.Setenv("OPENAI_BASE_URL", server.URL)

	o := &OpenAI{}
	agent := agents.Agent{Name: "test", Prompt: "test"}
	output, err := o.Deploy(agent, &discovery.ProjectContext{}, "test")
	if err != nil {
		t.Fatalf("Deploy failed: %v", err)
	}
	if output != "Part 1 Part 2" {
		t.Errorf("expected 'Part 1 Part 2', got: %q", output)
	}
}

func TestOpenAI_Deploy_HandlesHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, `{"error": "invalid api key"}`)
	}))
	defer server.Close()

	t.Setenv("OPENAI_API_KEY", "bad-key")
	t.Setenv("OPENAI_BASE_URL", server.URL)

	o := &OpenAI{}
	agent := agents.Agent{Name: "test", Prompt: "test"}
	_, err := o.Deploy(agent, &discovery.ProjectContext{}, "test")
	if err == nil {
		t.Fatal("expected error for 401 response")
	}

	// Should be a DeployError with status code
	deployErr, ok := err.(*DeployError)
	if !ok {
		t.Fatalf("expected *DeployError, got %T: %v", err, err)
	}
	if deployErr.StatusCode != 401 {
		t.Errorf("expected status 401, got %d", deployErr.StatusCode)
	}
}

func TestOpenAI_Deploy_HandlesRateLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		fmt.Fprint(w, `{"error": "rate limit exceeded"}`)
	}))
	defer server.Close()

	t.Setenv("OPENAI_API_KEY", "test-key")
	t.Setenv("OPENAI_BASE_URL", server.URL)

	o := &OpenAI{}
	agent := agents.Agent{Name: "test", Prompt: "test"}
	_, err := o.Deploy(agent, &discovery.ProjectContext{}, "test")
	if err == nil {
		t.Fatal("expected error for 429 response")
	}

	// 429 should be retryable
	if !IsRetryable(err) {
		t.Error("expected 429 error to be retryable")
	}
}

func TestOpenAI_Deploy_MissingAPIKey(t *testing.T) {
	os.Unsetenv("OPENAI_API_KEY")

	o := &OpenAI{}
	agent := agents.Agent{Name: "test", Prompt: "test"}
	_, err := o.Deploy(agent, &discovery.ProjectContext{}, "test")
	if err == nil {
		t.Fatal("expected error when API key is missing")
	}
}

func TestOpenAI_Deploy_DefaultBaseURL(t *testing.T) {
	// Verify that without OPENAI_BASE_URL set, it attempts the default
	// We'll get a connection error since there's no real server, but that's fine
	t.Setenv("OPENAI_API_KEY", "test-key")
	os.Unsetenv("OPENAI_BASE_URL")

	o := &OpenAI{}
	if !o.Available() {
		t.Error("expected Available() true when OPENAI_API_KEY is set")
	}
}

func TestOpenAI_Available(t *testing.T) {
	os.Unsetenv("OPENAI_API_KEY")
	o := &OpenAI{}
	if o.Available() {
		t.Error("expected Available() false without OPENAI_API_KEY")
	}

	t.Setenv("OPENAI_API_KEY", "some-key")
	if !o.Available() {
		t.Error("expected Available() true with OPENAI_API_KEY set")
	}
}

func TestOpenAI_Deploy_StreamCloseWithoutDone(t *testing.T) {
	// Ollama compat mode may close connection without sending [DONE]
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "data: %s\n\n", `{"choices":[{"delta":{"content":"partial"},"finish_reason":null}]}`)
		// Connection closes without [DONE]
	}))
	defer server.Close()

	t.Setenv("OPENAI_API_KEY", "test-key")
	t.Setenv("OPENAI_BASE_URL", server.URL)

	o := &OpenAI{}
	agent := agents.Agent{Name: "test", Prompt: "test"}
	output, err := o.Deploy(agent, &discovery.ProjectContext{}, "test")
	if err != nil {
		t.Fatalf("Deploy should succeed on graceful close without [DONE]: %v", err)
	}
	if output != "partial" {
		t.Errorf("expected 'partial', got: %q", output)
	}
}

func TestOpenAI_Deploy_MalformedSSEData(t *testing.T) {
	// Test that malformed JSON in SSE events is silently discarded
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		// Non-data lines (should be skipped)
		fmt.Fprintf(w, ": comment line\n\n")
		fmt.Fprintf(w, "event: ping\n\n")
		// Malformed JSON (should be discarded)
		fmt.Fprintf(w, "data: {not valid json\n\n")
		// Valid chunk
		fmt.Fprintf(w, "data: %s\n\n", `{"choices":[{"delta":{"content":"good"},"finish_reason":null}]}`)
		fmt.Fprintf(w, "data: [DONE]\n\n")
	}))
	defer server.Close()

	t.Setenv("OPENAI_API_KEY", "test-key")
	t.Setenv("OPENAI_BASE_URL", server.URL)

	o := &OpenAI{}
	agent := agents.Agent{Name: "test", Prompt: "test"}
	output, err := o.Deploy(agent, &discovery.ProjectContext{}, "test")
	if err != nil {
		t.Fatalf("Deploy failed: %v", err)
	}
	if output != "good" {
		t.Errorf("expected 'good' (malformed events skipped), got: %q", output)
	}
}

func TestOpenAI_Deploy_BaseURLWithTrailingSlash(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "data: %s\n\n", `{"choices":[{"delta":{"content":"ok"},"finish_reason":null}]}`)
		fmt.Fprintf(w, "data: [DONE]\n\n")
	}))
	defer server.Close()

	// Set base URL with trailing slash
	t.Setenv("OPENAI_API_KEY", "test-key")
	t.Setenv("OPENAI_BASE_URL", server.URL+"/")

	o := &OpenAI{}
	agent := agents.Agent{Name: "test", Prompt: "test"}
	output, err := o.Deploy(agent, &discovery.ProjectContext{}, "test")
	if err != nil {
		t.Fatalf("Deploy failed: %v", err)
	}
	if output != "ok" {
		t.Errorf("expected 'ok', got: %q", output)
	}
}

func TestOpenAI_Deploy_EmptyChoices(t *testing.T) {
	// Test handling of events with empty choices array
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		// Event with empty choices
		fmt.Fprintf(w, "data: %s\n\n", `{"choices":[]}`)
		// Normal event
		fmt.Fprintf(w, "data: %s\n\n", `{"choices":[{"delta":{"content":"text"},"finish_reason":null}]}`)
		fmt.Fprintf(w, "data: [DONE]\n\n")
	}))
	defer server.Close()

	t.Setenv("OPENAI_API_KEY", "test-key")
	t.Setenv("OPENAI_BASE_URL", server.URL)

	o := &OpenAI{}
	agent := agents.Agent{Name: "test", Prompt: "test"}
	output, err := o.Deploy(agent, &discovery.ProjectContext{}, "test")
	if err != nil {
		t.Fatalf("Deploy failed: %v", err)
	}
	if output != "text" {
		t.Errorf("expected 'text', got: %q", output)
	}
}
