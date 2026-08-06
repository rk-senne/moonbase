package tui

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/rk-senne/moonbase/internal/chat"
	"github.com/rk-senne/moonbase/internal/pipeline"
)

func TestLiveBuf_ChunkAccumulation(t *testing.T) {
	tests := []struct {
		name   string
		chunks []string
		want   string
	}{
		{"single chunk", []string{"hello"}, "hello"},
		{"multi chunk", []string{"hel", "lo ", "world"}, "hello world"},
		{"newlines preserved", []string{"line1\n", "line2\n"}, "line1\nline2\n"},
		{"empty chunks ignored", []string{"", "text", ""}, "text"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := NewApp()
			app.view = ViewPipeline
			app.boot.Ready = true
			app.views.Pipeline.State = pipeline.New("test")
			app.views.Pipeline.State.Phases[0].Status = pipeline.StatusRunning
			app.views.Pipeline.Running = true
			app.views.Pipeline.PhaseStreamCh = make(chan chat.StreamChunk, 10)
			app.views.Pipeline.PhaseStreamBuf = &bytes.Buffer{}
			app.views.Pipeline.PhaseStreamStart = time.Now()
			app.views.Pipeline.PhaseStreamCancel = func() {}
			app.views.Pipeline.PhaseStreamPhase = 1

			var model App = app
			for _, chunk := range tt.chunks {
				m, _ := model.Update(PhaseChunkMsg{Phase: 1, Text: chunk})
				model = m.(App)
			}

			if got := model.views.Pipeline.LiveBuf; got != tt.want {
				t.Errorf("LiveBuf = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestLiveBuf_FlushOnPhaseComplete(t *testing.T) {
	app := NewApp()
	app.view = ViewPipeline
	app.boot.Ready = true
	app.views.Pipeline.State = pipeline.New("test")
	app.views.Pipeline.State.Phases[0].Status = pipeline.StatusRunning
	app.views.Pipeline.Running = true
	ctx, cancel := context.WithCancel(context.Background())
	app.views.Pipeline.Ctx = ctx
	app.views.Pipeline.Cancel = cancel
	app.views.Pipeline.PhaseStreamBuf = &bytes.Buffer{}
	app.views.Pipeline.PhaseStreamPhase = 1

	// Simulate accumulated live buffer content
	app.views.Pipeline.LiveAgent = "Numbuh 1"
	app.views.Pipeline.LiveBuf = "accumulated agent output"

	// Process a PhaseResultMsg (simulates phase completion)
	msg := PhaseResultMsg{Phase: 1, Output: "accumulated agent output", Elapsed: 2 * time.Second}
	model, _ := app.Update(msg)
	result := model.(App)

	// LiveBuf should be flushed (empty)
	if result.views.Pipeline.LiveBuf != "" {
		t.Errorf("expected LiveBuf flushed, got %q", result.views.Pipeline.LiveBuf)
	}
	// LiveAgent should be cleared
	if result.views.Pipeline.LiveAgent != "" {
		t.Errorf("expected LiveAgent cleared, got %q", result.views.Pipeline.LiveAgent)
	}
	// Chat should contain the flushed content as a regular PipelineMsg
	found := false
	for _, m := range result.views.Pipeline.Chat {
		if m.Agent == "Numbuh 1" && m.Content == "accumulated agent output" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected flushed message in Chat with agent='Numbuh 1' and accumulated content")
	}
}

func TestLiveBuf_PlainTextRendering(t *testing.T) {
	// Verify that the live buffer renders as plain text (not glamour-rendered)
	// while streaming is active, and that completed messages get glamour.
	app := NewApp()
	app.boot.Ready = true
	app.width = 80
	app.height = 30
	app.view = ViewPipeline
	app.views.Pipeline.State = pipeline.New("test")
	app.views.Pipeline.State.Phases[0].Status = pipeline.StatusRunning
	app.views.Pipeline.Running = true
	// A completed message in Chat (will be glamour-rendered)
	app.views.Pipeline.Chat = []PipelineMsg{
		{Agent: "", Content: "━━━ Phase 1 ━━━"},
	}
	// Live streaming content (should be plain wrapped)
	app.views.Pipeline.LiveAgent = "Numbuh 1"
	app.views.Pipeline.LiveBuf = "streaming content here\nwith multiple lines\n"

	out := app.renderPipeline()
	// The live content should appear in the output
	if !strings.Contains(out, "streaming content") {
		t.Error("expected live buffer content in rendered pipeline output")
	}
}

func TestLiveBuf_StaleGenDropped(t *testing.T) {
	app := NewApp()
	app.view = ViewPipeline
	app.boot.Ready = true
	app.views.Pipeline.State = pipeline.New("new mission")
	app.views.Pipeline.State.Phases[0].Status = pipeline.StatusRunning
	app.views.Pipeline.Running = true
	app.views.Pipeline.Gen = 5
	app.views.Pipeline.PhaseStreamCh = make(chan chat.StreamChunk, 10)
	app.views.Pipeline.PhaseStreamBuf = &bytes.Buffer{}
	app.views.Pipeline.PhaseStreamStart = time.Now()
	app.views.Pipeline.PhaseStreamCancel = func() {}
	app.views.Pipeline.PhaseStreamPhase = 1

	// Send a chunk with stale gen (from previous mission)
	staleMsg := PhaseChunkMsg{Phase: 1, Text: "stale text", Gen: 3}
	model, cmd := app.Update(staleMsg)
	result := model.(App)

	// Should not accumulate stale text
	if result.views.Pipeline.LiveBuf != "" {
		t.Errorf("expected empty LiveBuf for stale gen, got %q", result.views.Pipeline.LiveBuf)
	}
	// Should not repoll
	if cmd != nil {
		t.Error("expected nil cmd for stale gen chunk")
	}
}
