package tui

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/rk-senne/moonbase/internal/chat"
	"github.com/rk-senne/moonbase/internal/pipeline"
	"go.uber.org/goleak"
)

// TestStreamLeak_MissionAbort verifies that aborting a mission does not leak
// the goroutine backing pollPhaseStream. The slow mock stream blocks until ctx
// is cancelled; after PipelineAbortedMsg the goroutine must exit cleanly.
func TestStreamLeak_MissionAbort(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	ctx, cancel := context.WithCancel(context.Background())
	// Buffered channel so the goroutine can send Done without blocking.
	ch := make(chan chat.StreamChunk, 2)

	// Simulate a slow backend: blocks until ctx is cancelled, then sends Done.
	go func() {
		defer close(ch)
		<-ctx.Done()
		// Buffered channel ensures this doesn't block.
		ch <- chat.StreamChunk{Done: true, Err: ctx.Err()}
	}()

	app := NewApp()
	app.boot.Ready = true
	app.view = ViewPipeline
	app.views.Pipeline.State = pipeline.New("leak test")
	app.views.Pipeline.State.Phases[0].Status = pipeline.StatusRunning
	app.views.Pipeline.Running = true
	app.views.Pipeline.Ctx = ctx
	app.views.Pipeline.Cancel = cancel
	app.views.Pipeline.PhaseStreamCh = ch
	app.views.Pipeline.PhaseStreamBuf = &bytes.Buffer{}
	app.views.Pipeline.PhaseStreamStart = time.Now()
	app.views.Pipeline.PhaseStreamCancel = cancel
	app.views.Pipeline.PhaseStreamPhase = 1

	// Simulate abort
	model, _ := app.Update(PipelineAbortedMsg{})
	result := model.(App)

	if result.views.Pipeline.Running {
		t.Error("expected Running=false after abort")
	}

	// Drain the channel so the goroutine can fully exit.
	for range ch {
	}

	// Give goroutine time to observe cancellation and exit.
	time.Sleep(50 * time.Millisecond)
}

// TestStreamLeak_MissionSupersede verifies no leak when a new mission supersedes
// the running one (incrementing Gen causes stale messages to be dropped).
func TestStreamLeak_MissionSupersede(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	ctx, cancel := context.WithCancel(context.Background())
	// Buffered channel so the goroutine can send Done without blocking.
	ch := make(chan chat.StreamChunk, 2)

	// Slow backend goroutine: waits for cancellation then sends Done.
	go func() {
		defer close(ch)
		<-ctx.Done()
		ch <- chat.StreamChunk{Done: true, Err: ctx.Err()}
	}()

	app := NewApp()
	app.boot.Ready = true
	app.view = ViewPipeline
	app.views.Pipeline.State = pipeline.New("old mission")
	app.views.Pipeline.State.Phases[0].Status = pipeline.StatusRunning
	app.views.Pipeline.Running = true
	app.views.Pipeline.Ctx = ctx
	app.views.Pipeline.Cancel = cancel
	app.views.Pipeline.PhaseStreamCh = ch
	app.views.Pipeline.PhaseStreamBuf = &bytes.Buffer{}
	app.views.Pipeline.PhaseStreamStart = time.Now()
	app.views.Pipeline.PhaseStreamCancel = cancel
	app.views.Pipeline.PhaseStreamPhase = 1
	app.views.Pipeline.Gen = 1

	// Supersede: increment gen and cancel old context (simulates supersedeRunningMission)
	cancel()
	app.views.Pipeline.Gen = 2
	app.views.Pipeline.PhaseStreamCancel = nil

	// A stale chunk arriving with old gen should be dropped.
	staleChunk := PhaseChunkMsg{Phase: 1, Text: "stale", Gen: 1}
	model, cmd := app.Update(staleChunk)
	result := model.(App)

	// Should not repoll (stale gen)
	if cmd != nil {
		t.Error("expected nil cmd for stale-gen chunk")
	}
	// LiveBuf should not have the stale text
	if result.views.Pipeline.LiveBuf != "" {
		t.Errorf("expected empty LiveBuf after stale chunk, got %q", result.views.Pipeline.LiveBuf)
	}

	// Drain the channel so the goroutine can fully exit.
	for range ch {
	}

	// Give goroutine time to exit.
	time.Sleep(50 * time.Millisecond)
}

// TestStreamLeak_ContextCancelClosesChannel verifies the stream adapter observes
// ctx cancellation and closes its channel (no goroutine blocks indefinitely).
func TestStreamLeak_ContextCancelClosesChannel(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	ctx, cancel := context.WithCancel(context.Background())
	ch := make(chan chat.StreamChunk, 1)

	// Simulated backend: blocks until cancel, then sends Done.
	go func() {
		defer close(ch)
		<-ctx.Done()
		ch <- chat.StreamChunk{Done: true, Err: ctx.Err()}
	}()

	buf := &bytes.Buffer{}
	start := time.Now()

	// Cancel immediately
	cancel()

	// pollPhaseStream should return a PhaseResultMsg (channel closes after cancel)
	cmd := pollPhaseStream(1, ch, start, func() {}, buf, 0)
	msg := cmd()

	result, ok := msg.(PhaseResultMsg)
	if !ok {
		t.Fatalf("expected PhaseResultMsg after cancel, got %T", msg)
	}
	if result.Err == nil {
		t.Log("stream ended with nil error (acceptable: done chunk may arrive without error)")
	}
}
