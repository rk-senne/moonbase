package tui

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/rk-senne/moonbase/internal/chat"
	"github.com/rk-senne/moonbase/internal/pipeline"
)

func TestPollPhaseStream_DoneBuildsPhaseResult(t *testing.T) {
	// Set up a channel that emits some text then Done
	ch := make(chan chat.StreamChunk, 5)
	buf := &bytes.Buffer{}

	// Simulate accumulated output already in buffer
	buf.WriteString("line 1\nline 2\n")

	// Send Done chunk
	ch <- chat.StreamChunk{Done: true, Err: nil}
	close(ch)

	start := time.Now().Add(-2 * time.Second) // pretend started 2s ago
	cancel := func() {}                        // no-op cancel for test

	cmd := pollPhaseStream(3, ch, start, cancel, buf)
	msg := cmd()

	result, ok := msg.(PhaseResultMsg)
	if !ok {
		t.Fatalf("expected PhaseResultMsg, got %T", msg)
	}

	if result.Phase != 3 {
		t.Errorf("expected phase 3, got %d", result.Phase)
	}
	if result.Output != "line 1\nline 2\n" {
		t.Errorf("expected accumulated output, got %q", result.Output)
	}
	if result.Err != nil {
		t.Errorf("expected no error, got %v", result.Err)
	}
	if result.Elapsed < 2*time.Second {
		t.Errorf("expected elapsed >= 2s, got %s", result.Elapsed)
	}
}

func TestPollPhaseStream_DoneWithError(t *testing.T) {
	ch := make(chan chat.StreamChunk, 2)
	buf := &bytes.Buffer{}
	buf.WriteString("partial output\n")

	ch <- chat.StreamChunk{Done: true, Err: context.DeadlineExceeded}
	close(ch)

	start := time.Now()
	cancelled := false
	cancel := func() { cancelled = true }

	cmd := pollPhaseStream(1, ch, start, cancel, buf)
	msg := cmd()

	result, ok := msg.(PhaseResultMsg)
	if !ok {
		t.Fatalf("expected PhaseResultMsg, got %T", msg)
	}

	if result.Err != context.DeadlineExceeded {
		t.Errorf("expected DeadlineExceeded, got %v", result.Err)
	}
	if result.Output != "partial output\n" {
		t.Errorf("expected partial output, got %q", result.Output)
	}
	if !cancelled {
		t.Error("expected cancel to be called")
	}
}

func TestPollPhaseStream_TextChunk(t *testing.T) {
	ch := make(chan chat.StreamChunk, 2)
	buf := &bytes.Buffer{}

	ch <- chat.StreamChunk{Text: "hello world\n"}

	start := time.Now()
	cancel := func() {}

	cmd := pollPhaseStream(2, ch, start, cancel, buf)
	msg := cmd()

	chunk, ok := msg.(PhaseChunkMsg)
	if !ok {
		t.Fatalf("expected PhaseChunkMsg, got %T", msg)
	}

	if chunk.Phase != 2 {
		t.Errorf("expected phase 2, got %d", chunk.Phase)
	}
	if chunk.Text != "hello world\n" {
		t.Errorf("expected 'hello world\\n', got %q", chunk.Text)
	}
}

func TestPollPhaseStream_ClosedChannel(t *testing.T) {
	ch := make(chan chat.StreamChunk)
	close(ch) // closed immediately

	buf := &bytes.Buffer{}
	buf.WriteString("previous content\n")

	start := time.Now()
	cancelled := false
	cancel := func() { cancelled = true }

	cmd := pollPhaseStream(1, ch, start, cancel, buf)
	msg := cmd()

	result, ok := msg.(PhaseResultMsg)
	if !ok {
		t.Fatalf("expected PhaseResultMsg on closed channel, got %T", msg)
	}
	if result.Output != "previous content\n" {
		t.Errorf("expected buffer contents, got %q", result.Output)
	}
	if !cancelled {
		t.Error("expected cancel to be called on closed channel")
	}
}

func TestHandlePhaseChunk_AppendsAndRepolls(t *testing.T) {
	app := NewApp()
	app.view = ViewPipeline
	app.boot.Ready = true
	app.views.Pipeline.State = pipeline.New("test task")
	app.views.Pipeline.State.Phases[0].Status = pipeline.StatusRunning
	app.views.Pipeline.Running = true

	// Set up streaming state
	ch := make(chan chat.StreamChunk, 10)
	buf := &bytes.Buffer{}
	start := time.Now()
	cancel := func() {}

	app.views.Pipeline.PhaseStreamCh = ch
	app.views.Pipeline.PhaseStreamBuf = buf
	app.views.Pipeline.PhaseStreamStart = start
	app.views.Pipeline.PhaseStreamCancel = cancel
	app.views.Pipeline.PhaseStreamPhase = 1

	// Add an initial agent message to append to
	app.views.Pipeline.Chat = append(app.views.Pipeline.Chat,
		PipelineMsg{"numbuh-1", ""},
	)

	msg := PhaseChunkMsg{Phase: 1, Text: "streaming text\n"}
	model, cmd := app.Update(msg)
	result := model.(App)

	// Buffer should have the text appended
	if result.views.Pipeline.PhaseStreamBuf.String() != "streaming text\n" {
		t.Errorf("expected buffer to contain 'streaming text\\n', got %q",
			result.views.Pipeline.PhaseStreamBuf.String())
	}

	// Chat should have the text appended to the last agent message
	lastChat := result.views.Pipeline.Chat[len(result.views.Pipeline.Chat)-1]
	if lastChat.Content != "streaming text\n" {
		t.Errorf("expected last chat content to be 'streaming text\\n', got %q", lastChat.Content)
	}

	// Should return a re-poll command
	if cmd == nil {
		t.Error("expected non-nil cmd (re-poll)")
	}
}

func TestHandlePhaseStreamStarted(t *testing.T) {
	app := NewApp()
	app.view = ViewPipeline
	app.boot.Ready = true
	app.views.Pipeline.State = pipeline.New("test task")
	app.views.Pipeline.Running = true

	ch := make(chan chat.StreamChunk, 5)
	start := time.Now()
	cancelled := false
	cancel := func() { cancelled = true }

	msg := phaseStreamStartedMsg{
		Phase:  1,
		Ch:     ch,
		Start:  start,
		Cancel: cancel,
	}

	model, cmd := app.Update(msg)
	result := model.(App)

	// Pipeline should have stream state set
	if result.views.Pipeline.PhaseStreamPhase != 1 {
		t.Errorf("expected PhaseStreamPhase=1, got %d", result.views.Pipeline.PhaseStreamPhase)
	}
	if result.views.Pipeline.PhaseStreamBuf == nil {
		t.Error("expected PhaseStreamBuf to be initialized")
	}
	if result.views.Pipeline.PhaseStreamCh == nil {
		t.Error("expected PhaseStreamCh to be set")
	}

	// Should return a poll command
	if cmd == nil {
		t.Error("expected non-nil cmd (initial poll)")
	}

	_ = cancelled // cancel is stored, not called yet
}

func TestComms_StreamUnchanged(t *testing.T) {
	// Verify that COMMS streaming uses streamChunkMsg (not PhaseChunkMsg) and
	// that handleStreamChunk is a separate code path from handlePhaseChunk.
	app := NewApp()
	app.view = ViewComms
	app.boot.Ready = true

	// Set up a minimal COMMS state
	app.views.Comms.State = &CommsState{
		agent:     "test-agent",
		streaming: true,
		conv:      &chat.Conversation{},
	}

	// Sending a streamChunkMsg should go through handleStreamChunk (COMMS path)
	msg := streamChunkMsg{text: "hello from comms", done: false, err: nil}
	model, _ := app.Update(msg)
	result := model.(App)

	// COMMS buffer should have been updated (via AppendStreamToken)
	if result.views.Comms.State.buffer != "hello from comms" {
		t.Errorf("expected COMMS buffer to contain 'hello from comms', got %q",
			result.views.Comms.State.buffer)
	}

	// PhaseChunkMsg should NOT affect COMMS state
	app2 := NewApp()
	app2.view = ViewPipeline
	app2.boot.Ready = true
	app2.views.Pipeline.State = pipeline.New("test")
	app2.views.Pipeline.Running = true
	app2.views.Pipeline.PhaseStreamBuf = &bytes.Buffer{}
	app2.views.Pipeline.PhaseStreamPhase = 1
	app2.views.Pipeline.PhaseStreamCh = make(chan chat.StreamChunk, 5)
	app2.views.Pipeline.PhaseStreamStart = time.Now()
	app2.views.Pipeline.PhaseStreamCancel = func() {}
	app2.views.Pipeline.Chat = append(app2.views.Pipeline.Chat,
		PipelineMsg{"agent", ""},
	)

	phaseMsg := PhaseChunkMsg{Phase: 1, Text: "pipeline text"}
	model2, _ := app2.Update(phaseMsg)
	result2 := model2.(App)

	// Pipeline buffer should have the text
	if result2.views.Pipeline.PhaseStreamBuf.String() != "pipeline text" {
		t.Errorf("expected pipeline buffer to contain 'pipeline text', got %q",
			result2.views.Pipeline.PhaseStreamBuf.String())
	}
}

func TestPipelineAbort_CancelsPhaseStream(t *testing.T) {
	app := NewApp()
	app.view = ViewPipeline
	app.boot.Ready = true
	app.views.Pipeline.State = pipeline.New("test task")
	app.views.Pipeline.Running = true

	cancelled := false
	app.views.Pipeline.PhaseStreamCancel = func() { cancelled = true }

	msg := PipelineAbortedMsg{}
	model, _ := app.Update(msg)
	result := model.(App)

	if !cancelled {
		t.Error("expected PhaseStreamCancel to be called on abort")
	}
	if result.views.Pipeline.Running {
		t.Error("expected Running=false after abort")
	}
	if result.views.Pipeline.PhaseStreamCancel != nil {
		t.Error("expected PhaseStreamCancel to be nil after abort")
	}
}
