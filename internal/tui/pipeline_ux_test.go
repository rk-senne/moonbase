package tui

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/rk-senne/moonbase/internal/chat"
	"github.com/rk-senne/moonbase/internal/pipeline"
)

// The live streaming block must be labelled with the currently-streaming
// operative so agent handoffs are visible mid-stream.
func TestPipelineChatLines_LiveHeaderShowsCurrentAgent(t *testing.T) {
	app := NewApp()
	app.boot.Ready = true
	app.width, app.height = 100, 30
	app.view = ViewPipeline
	app.views.Pipeline.State = pipeline.New("x")
	app.views.Pipeline.Running = true
	app.views.Pipeline.LiveAgent = "Numbuh 2"
	app.views.Pipeline.LiveBuf = "designing the blueprint…"

	joined := strings.Join(app.pipelineChatLines(app.pipelineMainWidth()), "\n")
	for _, want := range []string{"Numbuh 2", "Hoagie Gilligan", "streaming"} {
		if !strings.Contains(joined, want) {
			t.Errorf("expected live streaming header to contain %q", want)
		}
	}
}

func TestPipelineScroll_UpEndAndClamp(t *testing.T) {
	app := NewApp()
	app.boot.Ready = true
	app.width, app.height = 100, 20
	app.view = ViewPipeline
	app.views.Pipeline.State = pipeline.New("x")
	for i := 0; i < 60; i++ {
		app.views.Pipeline.Chat = append(app.views.Pipeline.Chat, PipelineMsg{"", fmt.Sprintf("line %d", i)})
	}

	// Up scrolls back one line.
	m, _, handled := app.handlePipelineScroll(tea.KeyPressMsg{Code: tea.KeyUp})
	if !handled {
		t.Fatal("expected 'up' handled as scroll in the pipeline view")
	}
	if got := m.(App).views.Pipeline.ChatScroll; got != 1 {
		t.Errorf("expected ChatScroll 1 after up, got %d", got)
	}

	// End jumps back to following the latest (0).
	m, _, _ = m.(App).handlePipelineScroll(tea.KeyPressMsg{Code: tea.KeyEnd})
	if got := m.(App).views.Pipeline.ChatScroll; got != 0 {
		t.Errorf("expected End to reset scroll to 0, got %d", got)
	}

	// Down at the bottom clamps at 0.
	m, _, _ = m.(App).handlePipelineScroll(tea.KeyPressMsg{Code: tea.KeyDown})
	if got := m.(App).views.Pipeline.ChatScroll; got != 0 {
		t.Errorf("expected scroll to stay clamped at 0, got %d", got)
	}

	// Non-scroll keys are not consumed by the scroll handler.
	if _, _, h := app.handlePipelineScroll(tea.KeyPressMsg{Code: 'n', Text: "n"}); h {
		t.Error("expected 'n' to fall through (not handled as scroll)")
	}
}

func TestPipelineSkip_InterruptsRunningPhase(t *testing.T) {
	app := NewApp()
	app.boot.Ready = true
	app.view = ViewPipeline
	app.views.Pipeline.State = pipeline.New("x")
	app.views.Pipeline.State.Phases[0].Status = pipeline.StatusRunning
	app.views.Pipeline.Running = true
	app.views.Pipeline.Gen = 1
	cancelled := false
	app.views.Pipeline.PhaseStreamCancel = func() { cancelled = true }

	model, _ := app.handlePipelineSkip()
	result := model.(App)

	if !cancelled {
		t.Error("expected the running phase's backend stream to be cancelled on skip")
	}
	if result.views.Pipeline.Gen != 2 {
		t.Errorf("expected generation bumped to 2 (discard interrupted phase msgs), got %d", result.views.Pipeline.Gen)
	}
	if result.views.Pipeline.State.Phases[0].Status != pipeline.StatusSkipped {
		t.Errorf("expected phase 0 marked skipped, got %d", result.views.Pipeline.State.Phases[0].Status)
	}
}

func TestPollPhaseStream_CoalescesAvailableChunks(t *testing.T) {
	ch := make(chan chat.StreamChunk, 8)
	ch <- chat.StreamChunk{Text: "a"}
	ch <- chat.StreamChunk{Text: "b"}
	ch <- chat.StreamChunk{Text: "c"}
	buf := &bytes.Buffer{}

	msg := pollPhaseStream(1, ch, time.Now(), func() {}, buf, 0)()
	pc, ok := msg.(PhaseChunkMsg)
	if !ok {
		t.Fatalf("expected PhaseChunkMsg, got %T", msg)
	}
	if pc.Text != "abc" {
		t.Errorf("expected coalesced text 'abc', got %q", pc.Text)
	}
}

func TestPollPhaseStream_CoalesceThenDoneFlushesBuffer(t *testing.T) {
	ch := make(chan chat.StreamChunk, 4)
	ch <- chat.StreamChunk{Text: "partial"}
	ch <- chat.StreamChunk{Done: true}
	close(ch)
	buf := &bytes.Buffer{}

	msg := pollPhaseStream(2, ch, time.Now(), func() {}, buf, 0)()
	pr, ok := msg.(PhaseResultMsg)
	if !ok {
		t.Fatalf("expected PhaseResultMsg when stream ends mid-drain, got %T", msg)
	}
	if pr.Output != "partial" {
		t.Errorf("expected coalesced text flushed to output ('partial'), got %q", pr.Output)
	}
}
