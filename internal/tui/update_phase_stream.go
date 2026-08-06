package tui

import (
	"bytes"

	tea "charm.land/bubbletea/v2"
)

// handlePhaseStreamStarted processes a successfully started phase stream.
// It stores the channel, cancel func, and buffer on the PipelineModel, then
// kicks the first pollPhaseStream to begin consuming chunks. Streams belonging
// to a superseded mission (msg.Gen != current) are cancelled and dropped.
func (a App) handlePhaseStreamStarted(msg phaseStreamStartedMsg) (tea.Model, tea.Cmd) {
	if msg.Gen != a.views.Pipeline.Gen {
		if msg.Cancel != nil {
			msg.Cancel()
		}
		return a, nil
	}

	a.views.Pipeline.PhaseStreamCh = msg.Ch
	a.views.Pipeline.PhaseStreamStart = msg.Start
	a.views.Pipeline.PhaseStreamCancel = msg.Cancel
	a.views.Pipeline.PhaseStreamPhase = msg.Phase
	a.views.Pipeline.PhaseStreamBuf = &bytes.Buffer{}

	return a, pollPhaseStream(
		msg.Phase,
		msg.Ch,
		msg.Start,
		msg.Cancel,
		a.views.Pipeline.PhaseStreamBuf,
		a.views.Pipeline.Gen,
	)
}

// handlePhaseChunk appends the chunk text to the phase stream buffer and the
// live streaming buffer, then re-polls for the next chunk. The live buffer is
// rendered as plain wrapped text (no glamour) on each frame — glamour is only
// applied once the phase completes (AC-1.2). Chunks from a superseded mission
// (msg.Gen != current) are dropped without re-polling (AC-1.3).
func (a App) handlePhaseChunk(msg PhaseChunkMsg) (tea.Model, tea.Cmd) {
	if msg.Gen != a.views.Pipeline.Gen {
		return a, nil
	}

	if a.views.Pipeline.PhaseStreamBuf != nil {
		a.views.Pipeline.PhaseStreamBuf.WriteString(msg.Text)
	}

	// Accumulate into the live buffer for plain-text rendering (AC-1.1/1.2).
	// Determine the operative for the live buffer label.
	if a.views.Pipeline.LiveAgent == "" {
		operative := ""
		if a.views.Pipeline.State != nil && a.views.Pipeline.State.CurrentPhase() != nil {
			operative = a.views.Pipeline.State.CurrentPhase().Operative
		}
		a.views.Pipeline.LiveAgent = operative
	}
	a.views.Pipeline.LiveBuf += msg.Text

	return a, pollPhaseStream(
		a.views.Pipeline.PhaseStreamPhase,
		a.views.Pipeline.PhaseStreamCh,
		a.views.Pipeline.PhaseStreamStart,
		a.views.Pipeline.PhaseStreamCancel,
		a.views.Pipeline.PhaseStreamBuf,
		a.views.Pipeline.Gen,
	)
}
