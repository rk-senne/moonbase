package tui

import (
	"bytes"
	"context"
	"time"

	"github.com/rk-senne/moonbase/internal/chat"
	"github.com/rk-senne/moonbase/internal/pipeline"
)

// PipelineModel owns pipeline execution state: the pipeline itself, its chat/output
// buffers, abort tracking, and the execution context. It is a value type with grouped
// fields — deliberately NOT a tea.Model implementation (concrete return type).
type PipelineModel struct {
	State        *pipeline.Pipeline
	Output       []string
	Chat         []PipelineMsg
	Running      bool
	AbortPending bool
	AbortAt      time.Time
	MissionStart time.Time
	Ctx          context.Context
	Cancel       context.CancelFunc
	StreamCh     <-chan chat.StreamChunk

	// Streaming pipeline phase state (separate from COMMS StreamCh above).
	PhaseStreamCh     <-chan chat.StreamChunk
	PhaseStreamBuf    *bytes.Buffer
	PhaseStreamStart  time.Time
	PhaseStreamCancel context.CancelFunc
	PhaseStreamPhase  int

	// LiveAgent is the operative currently streaming. LiveBuf accumulates
	// in-flight chunks so they can be rendered as plain wrapped text (no
	// glamour) on each frame. On phase completion the buffer is flushed into
	// a normal PipelineMsg (which renders through cached glamour once).
	//
	// LiveBuf is a plain string, NOT a strings.Builder: the whole App (and thus
	// this struct) is copied by value on every Bubble Tea Update, and copying a
	// non-zero strings.Builder panics ("illegal use of non-zero Builder copied
	// by value"). A string is value-copy-safe; per-phase output is bounded
	// (MaxOutputSize), so incremental concatenation cost is negligible.
	LiveAgent string
	LiveBuf   string

	// ChatScroll is how many lines the operator has scrolled UP from the bottom
	// of the conversation. 0 = pinned to the latest output (auto-follow); >0 =
	// reviewing history. Clamped to the available range on each key event.
	ChatScroll int

	// Gen is a monotonically increasing mission generation. It is incremented
	// each time a new mission is submitted so that in-flight phase messages
	// from a prior (cancelled) mission are ignored instead of corrupting the
	// freshly started pipeline. See app.go message guards.
	Gen int
}

// NewPipelineModel constructs a PipelineModel with defaults.
func NewPipelineModel() PipelineModel {
	return PipelineModel{}
}

// IsAbortConfirmed checks if a second abort was requested within the 3s window.
func (m PipelineModel) IsAbortConfirmed() bool {
	return m.AbortPending && time.Since(m.AbortAt) < 3*time.Second
}
