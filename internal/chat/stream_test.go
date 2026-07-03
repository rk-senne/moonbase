package chat

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// mockSSEServer returns an httptest.Server that streams Anthropic-format SSE events.
// Each event string is sent as a "data: <event>\n\n" line. A final "data: [DONE]\n\n"
// is appended unless the caller includes it explicitly.
func mockSSEServer(events []string, appendDone bool) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		flusher, _ := w.(http.Flusher)
		for _, event := range events {
			fmt.Fprintf(w, "data: %s\n\n", event)
			flusher.Flush()
		}
		if appendDone {
			fmt.Fprintf(w, "data: [DONE]\n\n")
			flusher.Flush()
		}
	}))
}

// mockErrorServer returns an httptest.Server that responds with the given status code and body.
func mockErrorServer(statusCode int, body string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(statusCode)
		w.Write([]byte(body))
	}))
}

// contentBlockDelta builds a JSON SSE event string matching Anthropic's content_block_delta format.
func contentBlockDelta(text string) string {
	return fmt.Sprintf(`{"type":"content_block_delta","delta":{"type":"text_delta","text":"%s"}}`, text)
}

// messageStop returns the JSON event that signals end of message.
func messageStop() string {
	return `{"type":"message_stop"}`
}

// testConversation creates a minimal conversation for testing.
func testConversation() *Conversation {
	conv := NewConversation("test-agent", "You are a test agent.")
	conv.Add(RoleUser, "Hello")
	return conv
}

// collectChunks reads all chunks from the channel with a timeout.
func collectChunks(ch <-chan StreamChunk, timeout time.Duration) []StreamChunk {
	var chunks []StreamChunk
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	for {
		select {
		case chunk, ok := <-ch:
			if !ok {
				return chunks
			}
			chunks = append(chunks, chunk)
		case <-timer.C:
			return chunks
		}
	}
}

func TestStream_Success(t *testing.T) {
	events := []string{
		contentBlockDelta("Hello"),
		contentBlockDelta(", "),
		contentBlockDelta("world!"),
	}
	server := mockSSEServer(events, true)
	defer server.Close()

	conv := testConversation()
	ch := streamFrom(server.URL, "test-key", "test-model", conv, server.Client())

	chunks := collectChunks(ch, 5*time.Second)

	// Expect 3 text chunks + 1 done chunk (from [DONE])
	textChunks := []StreamChunk{}
	var doneChunk *StreamChunk
	for i := range chunks {
		if chunks[i].Done {
			doneChunk = &chunks[i]
		} else if chunks[i].Text != "" {
			textChunks = append(textChunks, chunks[i])
		}
	}

	if len(textChunks) != 3 {
		t.Fatalf("expected 3 text chunks, got %d: %+v", len(textChunks), chunks)
	}

	expectedTexts := []string{"Hello", ", ", "world!"}
	for i, expected := range expectedTexts {
		if textChunks[i].Text != expected {
			t.Errorf("chunk[%d]: expected %q, got %q", i, expected, textChunks[i].Text)
		}
	}

	if doneChunk == nil {
		t.Error("expected a done chunk, got none")
	}
}

func TestStream_EmptyResponse(t *testing.T) {
	// Server immediately sends [DONE] with no content events
	server := mockSSEServer([]string{}, true)
	defer server.Close()

	conv := testConversation()
	ch := streamFrom(server.URL, "test-key", "test-model", conv, server.Client())

	chunks := collectChunks(ch, 5*time.Second)

	// Should only get a done chunk
	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk (done), got %d: %+v", len(chunks), chunks)
	}
	if !chunks[0].Done {
		t.Errorf("expected done chunk, got %+v", chunks[0])
	}
}

func TestStream_ServerError(t *testing.T) {
	server := mockErrorServer(500, `{"error":{"message":"internal server error"}}`)
	defer server.Close()

	conv := testConversation()
	ch := streamFrom(server.URL, "test-key", "test-model", conv, server.Client())

	chunks := collectChunks(ch, 5*time.Second)

	if len(chunks) != 1 {
		t.Fatalf("expected 1 error chunk, got %d: %+v", len(chunks), chunks)
	}
	if chunks[0].Err == nil {
		t.Fatal("expected error in chunk, got nil")
	}
	if !strings.Contains(chunks[0].Err.Error(), "500") {
		t.Errorf("error should contain status code 500, got: %v", chunks[0].Err)
	}
	if !strings.Contains(chunks[0].Err.Error(), "internal server error") {
		t.Errorf("error should contain response body, got: %v", chunks[0].Err)
	}
}

func TestStream_AuthError(t *testing.T) {
	server := mockErrorServer(401, `{"error":{"type":"authentication_error","message":"invalid x-api-key"}}`)
	defer server.Close()

	conv := testConversation()
	ch := streamFrom(server.URL, "bad-key", "test-model", conv, server.Client())

	chunks := collectChunks(ch, 5*time.Second)

	if len(chunks) != 1 {
		t.Fatalf("expected 1 error chunk, got %d: %+v", len(chunks), chunks)
	}
	if chunks[0].Err == nil {
		t.Fatal("expected error in chunk, got nil")
	}
	errMsg := chunks[0].Err.Error()
	if !strings.Contains(errMsg, "401") {
		t.Errorf("error should contain status code 401, got: %v", errMsg)
	}
	if !strings.Contains(errMsg, "invalid x-api-key") {
		t.Errorf("error should contain meaningful auth error message, got: %v", errMsg)
	}
}

func TestStream_RateLimit(t *testing.T) {
	server := mockErrorServer(429, `{"error":{"type":"rate_limit_error","message":"Rate limit exceeded. Please retry after 30 seconds."}}`)
	defer server.Close()

	conv := testConversation()
	ch := streamFrom(server.URL, "test-key", "test-model", conv, server.Client())

	chunks := collectChunks(ch, 5*time.Second)

	if len(chunks) != 1 {
		t.Fatalf("expected 1 error chunk, got %d: %+v", len(chunks), chunks)
	}
	if chunks[0].Err == nil {
		t.Fatal("expected error in chunk, got nil")
	}
	errMsg := chunks[0].Err.Error()
	if !strings.Contains(errMsg, "429") {
		t.Errorf("error should contain status code 429, got: %v", errMsg)
	}
	if !strings.Contains(errMsg, "rate_limit") || !strings.Contains(errMsg, "Rate limit") {
		t.Errorf("error should contain rate limit info, got: %v", errMsg)
	}
}

func TestStream_MalformedSSE(t *testing.T) {
	// Mix of garbage data and valid events — should not panic
	events := []string{
		"not json at all",
		"{incomplete json",
		`{"type":"unknown_event"}`,
		`{"type":"content_block_delta","delta":{"type":"text_delta","text":"valid"}}`,
		"",
		`null`,
		`{"type":"content_block_delta","delta":{}}`,
	}
	server := mockSSEServer(events, true)
	defer server.Close()

	conv := testConversation()
	ch := streamFrom(server.URL, "test-key", "test-model", conv, server.Client())

	chunks := collectChunks(ch, 5*time.Second)

	// Should get the one valid text chunk + done, without panicking
	var textChunks []StreamChunk
	var gotDone bool
	for _, chunk := range chunks {
		if chunk.Err != nil {
			t.Fatalf("unexpected error chunk: %v", chunk.Err)
		}
		if chunk.Done {
			gotDone = true
		}
		if chunk.Text != "" {
			textChunks = append(textChunks, chunk)
		}
	}

	if len(textChunks) != 1 || textChunks[0].Text != "valid" {
		t.Errorf("expected 1 valid text chunk with 'valid', got %+v", textChunks)
	}
	if !gotDone {
		t.Error("expected done chunk")
	}
}

func TestStream_ConnectionDrop(t *testing.T) {
	// Server sends one event then abruptly closes connection (no [DONE])
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, _ := w.(http.Flusher)
		fmt.Fprintf(w, "data: %s\n\n", contentBlockDelta("partial"))
		flusher.Flush()
		// Close connection immediately via hijack
		hj, ok := w.(http.Hijacker)
		if ok {
			conn, _, _ := hj.Hijack()
			conn.Close()
		}
	}))
	defer server.Close()

	conv := testConversation()
	ch := streamFrom(server.URL, "test-key", "test-model", conv, server.Client())

	chunks := collectChunks(ch, 5*time.Second)

	// Should get the partial text chunk + a done chunk (from scanner exhaustion)
	// The stream.go code sends Done when scanner finishes (connection closed)
	var gotText bool
	var gotDone bool
	for _, chunk := range chunks {
		if chunk.Text == "partial" {
			gotText = true
		}
		if chunk.Done {
			gotDone = true
		}
	}

	if !gotText {
		t.Error("expected to receive the partial text chunk before connection drop")
	}
	if !gotDone {
		t.Error("expected done chunk after connection drop (scanner exhaustion)")
	}
}

func TestStream_LargePayload(t *testing.T) {
	// Generate a string larger than 64KB
	largeText := strings.Repeat("x", 70000)
	events := []string{
		fmt.Sprintf(`{"type":"content_block_delta","delta":{"type":"text_delta","text":"%s"}}`, largeText),
	}
	server := mockSSEServer(events, true)
	defer server.Close()

	conv := testConversation()
	ch := streamFrom(server.URL, "test-key", "test-model", conv, server.Client())

	chunks := collectChunks(ch, 10*time.Second)

	// Should handle the large payload (bufio.Scanner default is 64KB, but SSE lines
	// could exceed this — check if we get the content or a graceful failure)
	var gotText bool
	var gotDone bool
	var gotErr bool
	for _, chunk := range chunks {
		if chunk.Text != "" {
			gotText = true
			if len(chunk.Text) != 70000 {
				t.Errorf("expected text length 70000, got %d", len(chunk.Text))
			}
		}
		if chunk.Done {
			gotDone = true
		}
		if chunk.Err != nil {
			gotErr = true
		}
	}

	// The default bufio.Scanner buffer is 64KB. If a line exceeds this, the scanner
	// will return an error and stop. In that case, the code sends Done from the
	// deferred close path. Either way, it should handle gracefully.
	if !gotText && !gotDone && !gotErr {
		t.Fatal("expected either text chunk, done chunk, or graceful error — got nothing")
	}

	// If we got text, it should be the full 70KB content
	if gotText && !gotDone {
		t.Log("large payload received successfully but no done signal — possible scanner limit")
	}
}

func TestStream_MultipleDataFields(t *testing.T) {
	// SSE spec: multiple "data:" lines before a blank line form a single event
	// joined by newlines. However, our implementation uses bufio.Scanner line-by-line
	// and expects each "data:" line to be a complete event.
	// This test verifies behavior when multiple data: lines appear —
	// each line is treated as a separate event per our implementation.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, _ := w.(http.Flusher)

		// Standard format: each data line is its own event separated by double newline
		fmt.Fprintf(w, "data: %s\n\n", contentBlockDelta("first"))
		flusher.Flush()

		// Two data: lines before blank line (SSE spec concatenation)
		// Our scanner reads line-by-line, so each "data: " line is processed independently
		fmt.Fprintf(w, "data: %s\n", contentBlockDelta("second"))
		fmt.Fprintf(w, "data: %s\n\n", contentBlockDelta("third"))
		flusher.Flush()

		fmt.Fprintf(w, "data: [DONE]\n\n")
		flusher.Flush()
	}))
	defer server.Close()

	conv := testConversation()
	ch := streamFrom(server.URL, "test-key", "test-model", conv, server.Client())

	chunks := collectChunks(ch, 5*time.Second)

	// Our line-by-line scanner should pick up each "data: " line independently
	var textChunks []string
	for _, chunk := range chunks {
		if chunk.Text != "" {
			textChunks = append(textChunks, chunk.Text)
		}
	}

	// All three data lines have valid JSON, so all should be parsed
	if len(textChunks) != 3 {
		t.Fatalf("expected 3 text chunks (line-by-line processing), got %d: %v", len(textChunks), textChunks)
	}

	expected := []string{"first", "second", "third"}
	for i, exp := range expected {
		if textChunks[i] != exp {
			t.Errorf("chunk[%d]: expected %q, got %q", i, exp, textChunks[i])
		}
	}
}

func TestStream_MessageStopEvent(t *testing.T) {
	// Test that message_stop event properly terminates the stream
	events := []string{
		contentBlockDelta("response text"),
		messageStop(),
	}
	// Don't append [DONE] — message_stop should terminate first
	server := mockSSEServer(events, false)
	defer server.Close()

	conv := testConversation()
	ch := streamFrom(server.URL, "test-key", "test-model", conv, server.Client())

	chunks := collectChunks(ch, 5*time.Second)

	var gotText bool
	var gotDone bool
	for _, chunk := range chunks {
		if chunk.Text == "response text" {
			gotText = true
		}
		if chunk.Done {
			gotDone = true
		}
	}

	if !gotText {
		t.Error("expected text chunk with 'response text'")
	}
	if !gotDone {
		t.Error("expected done chunk from message_stop event")
	}
}

func TestStream_NoAPIKey(t *testing.T) {
	// Test the Stream function directly with no API key set
	t.Setenv("ANTHROPIC_API_KEY", "")

	conv := testConversation()
	ch := Stream(conv)

	chunks := collectChunks(ch, 5*time.Second)

	if len(chunks) != 1 {
		t.Fatalf("expected 1 error chunk, got %d: %+v", len(chunks), chunks)
	}
	if chunks[0].Err == nil {
		t.Fatal("expected error for missing API key")
	}
	if !strings.Contains(chunks[0].Err.Error(), "ANTHROPIC_API_KEY not set") {
		t.Errorf("expected ANTHROPIC_API_KEY error, got: %v", chunks[0].Err)
	}
}

func TestStream_NonSSELines(t *testing.T) {
	// Server sends event/id/retry fields mixed with data lines
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, _ := w.(http.Flusher)

		fmt.Fprintf(w, "event: content_block_delta\n")
		fmt.Fprintf(w, "id: 1\n")
		fmt.Fprintf(w, "data: %s\n\n", contentBlockDelta("hello"))
		flusher.Flush()

		fmt.Fprintf(w, ": this is a comment\n")
		fmt.Fprintf(w, "retry: 5000\n")
		fmt.Fprintf(w, "data: %s\n\n", contentBlockDelta(" world"))
		flusher.Flush()

		fmt.Fprintf(w, "data: [DONE]\n\n")
		flusher.Flush()
	}))
	defer server.Close()

	conv := testConversation()
	ch := streamFrom(server.URL, "test-key", "test-model", conv, server.Client())

	chunks := collectChunks(ch, 5*time.Second)

	var texts []string
	for _, chunk := range chunks {
		if chunk.Text != "" {
			texts = append(texts, chunk.Text)
		}
	}

	if len(texts) != 2 {
		t.Fatalf("expected 2 text chunks, got %d: %v", len(texts), texts)
	}
	if texts[0] != "hello" || texts[1] != " world" {
		t.Errorf("expected ['hello', ' world'], got %v", texts)
	}
}
