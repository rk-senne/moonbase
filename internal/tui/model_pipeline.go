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
}

// NewPipelineModel constructs a PipelineModel with defaults.
func NewPipelineModel() PipelineModel {
	return PipelineModel{}
}

// IsAbortConfirmed checks if a second abort was requested within the 3s window.
func (m PipelineModel) IsAbortConfirmed() bool {
	return m.AbortPending && time.Since(m.AbortAt) < 3*time.Second
}
