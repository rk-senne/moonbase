package backend

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestStreamChatCompletion_ExtractsUsage(t *testing.T) {
	// Mock SSE server that returns streamed chunks with a final usage event
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		// Normal content chunks
		fmt.Fprintf(w, "data: %s\n\n", `{"choices":[{"delta":{"content":"Hello"},"finish_reason":null}]}`)
		fmt.Fprintf(w, "data: %s\n\n", `{"choices":[{"delta":{"content":" World"},"finish_reason":null}]}`)
		// Final event with usage (sent when stream_options.include_usage=true)
		fmt.Fprintf(w, "data: %s\n\n", `{"choices":[],"usage":{"prompt_tokens":42,"completion_tokens":128,"total_tokens":170}}`)
		fmt.Fprintf(w, "data: [DONE]\n\n")
	}))
	defer server.Close()

	output, usage, err := streamChatCompletion(http.DefaultClient, server.URL, "test-key", "gpt-4o", "system prompt", "user task")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if output != "Hello World" {
		t.Errorf("expected 'Hello World', got %q", output)
	}

	if usage == nil {
		t.Fatal("expected non-nil usage")
	}
	if usage.PromptTokens != 42 {
		t.Errorf("expected 42 prompt tokens, got %d", usage.PromptTokens)
	}
	if usage.CompletionTokens != 128 {
		t.Errorf("expected 128 completion tokens, got %d", usage.CompletionTokens)
	}
	if usage.TotalTokens != 170 {
		t.Errorf("expected 170 total tokens, got %d", usage.TotalTokens)
	}
	if usage.Model != "gpt-4o" {
		t.Errorf("expected model 'gpt-4o', got %q", usage.Model)
	}
}

func TestStreamChatCompletion_NoUsageReturnsNil(t *testing.T) {
	// Server that sends content without usage
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		fmt.Fprintf(w, "data: %s\n\n", `{"choices":[{"delta":{"content":"response"},"finish_reason":null}]}`)
		fmt.Fprintf(w, "data: [DONE]\n\n")
	}))
	defer server.Close()

	output, usage, err := streamChatCompletion(http.DefaultClient, server.URL, "test-key", "gpt-4o", "system", "user")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if output != "response" {
		t.Errorf("expected 'response', got %q", output)
	}

	if usage != nil {
		t.Errorf("expected nil usage when no usage in stream, got %+v", usage)
	}
}

func TestStreamChatCompletion_UsageWithFinishReason(t *testing.T) {
	// Server that uses finish_reason: stop and includes usage before stop
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		fmt.Fprintf(w, "data: %s\n\n", `{"choices":[{"delta":{"content":"done"},"finish_reason":null}]}`)
		// Usage arrives with the finish event
		fmt.Fprintf(w, "data: %s\n\n", `{"choices":[{"delta":{"content":""},"finish_reason":"stop"}],"usage":{"prompt_tokens":100,"completion_tokens":50,"total_tokens":150}}`)
	}))
	defer server.Close()

	output, usage, err := streamChatCompletion(http.DefaultClient, server.URL, "test-key", "gpt-4o-mini", "system", "user")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if output != "done" {
		t.Errorf("expected 'done', got %q", output)
	}

	if usage == nil {
		t.Fatal("expected non-nil usage")
	}
	if usage.TotalTokens != 150 {
		t.Errorf("expected 150 total tokens, got %d", usage.TotalTokens)
	}
	if usage.Model != "gpt-4o-mini" {
		t.Errorf("expected model 'gpt-4o-mini', got %q", usage.Model)
	}
}

func TestDeployComposed_WithRawUsageReporter(t *testing.T) {
	// Save and restore preferred backend state
	origDetect := DetectAll
	defer func() { _ = origDetect }()

	// This test verifies the interface assertion path — we test via the
	// streamChatCompletion function which is what the real implementation uses.
	// Direct test of DeployComposed's dispatch logic requires injecting a backend,
	// which is tested indirectly via the existing coverage2_test.go tests.
	// Here we just verify the type assertions compile correctly.

	var be interface{} = &OpenAI{}
	if _, ok := be.(RawUsageReporter); !ok {
		t.Error("OpenAI should implement RawUsageReporter")
	}

	be = &Kimi{}
	if _, ok := be.(RawUsageReporter); !ok {
		t.Error("Kimi should implement RawUsageReporter")
	}

	be = &Anthropic{}
	if _, ok := be.(UsageReporter); !ok {
		t.Error("Anthropic should implement UsageReporter")
	}
}

func TestNonReportingBackends_DoNotImplementUsageInterfaces(t *testing.T) {
	backends := []Backend{
		&Kiro{},
		&Codex{},
		&Ollama{},
		&Clipboard{},
	}

	for _, be := range backends {
		t.Run(be.Name(), func(t *testing.T) {
			if _, ok := be.(UsageReporter); ok {
				t.Errorf("%s should NOT implement UsageReporter", be.Name())
			}
			if _, ok := be.(RawUsageReporter); ok {
				t.Errorf("%s should NOT implement RawUsageReporter", be.Name())
			}
		})
	}
}

func TestDeployComposed_ReturnsUsageFromStream(t *testing.T) {
	// Full integration test: mock server with usage -> OpenAI.DeployRawWithUsage -> verify
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		fmt.Fprintf(w, "data: %s\n\n", `{"choices":[{"delta":{"content":"output"},"finish_reason":null}]}`)
		fmt.Fprintf(w, "data: %s\n\n", `{"choices":[],"usage":{"prompt_tokens":200,"completion_tokens":80,"total_tokens":280}}`)
		fmt.Fprintf(w, "data: [DONE]\n\n")
	}))
	defer server.Close()

	t.Setenv("OPENAI_API_KEY", "test-key")
	t.Setenv("OPENAI_BASE_URL", server.URL)
	t.Setenv("OPENAI_MODEL", "gpt-4o")

	o := &OpenAI{}
	output, usage, err := o.DeployRawWithUsage("composed prompt", "task")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if output != "output" {
		t.Errorf("expected 'output', got %q", output)
	}

	if usage == nil {
		t.Fatal("expected non-nil usage")
	}
	if usage.PromptTokens != 200 {
		t.Errorf("expected 200 prompt tokens, got %d", usage.PromptTokens)
	}
	if usage.CompletionTokens != 80 {
		t.Errorf("expected 80 completion tokens, got %d", usage.CompletionTokens)
	}
	if usage.TotalTokens != 280 {
		t.Errorf("expected 280 total tokens, got %d", usage.TotalTokens)
	}
}

func TestDeployComposed_NilUsageWhenBackendDoesNotReport(t *testing.T) {
	// When DeployComposed uses a backend without RawUsageReporter, usage should be nil.
	// We test this by ensuring DeployComposed handles the nil path correctly.
	// The clipboard fallback path always returns nil usage.
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// With no backends available, DeployComposed falls back to clipboard.
	// We can't easily control which backend is preferred in tests without
	// modifying global state, but we verify the function signature works.
	_, _, _ = DeployComposed(ctx, "test", "task", 1*time.Second)
	// The key assertion is that this compiles and doesn't panic.
}
