package tui

import (
	"bytes"

	tea "charm.land/bubbletea/v2"
)

// handlePhaseStreamStarted processes a successfully started phase stream.
// It stores the channel, cancel func, and buffer on the PipelineModel, then
// kicks the first pollPhaseStream to begin consuming chunks.
func (a App) handlePhaseStreamStarted(msg phaseStreamStartedMsg) (tea.Model, tea.Cmd) {
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
	)
}

// handlePhaseChunk appends the chunk text to the phase stream buffer and the
// live pipeline chat, then re-polls for the next chunk.
func (a App) handlePhaseChunk(msg PhaseChunkMsg) (tea.Model, tea.Cmd) {
	if a.views.Pipeline.PhaseStreamBuf != nil {
		a.views.Pipeline.PhaseStreamBuf.WriteString(msg.Text)
	}

	// Append to live pipeline chat so the operator sees the agent "typing"
	if len(a.views.Pipeline.Chat) > 0 {
		last := &a.views.Pipeline.Chat[len(a.views.Pipeline.Chat)-1]
		// If the last message is from an agent (streaming content), append to it
		if last.Agent != "" {
			last.Content += msg.Text
		} else {
			// Start a new agent message block
			operative := ""
			if a.views.Pipeline.State != nil && a.views.Pipeline.State.CurrentPhase() != nil {
				operative = a.views.Pipeline.State.CurrentPhase().Operative
			}
			a.views.Pipeline.Chat = append(a.views.Pipeline.Chat,
				PipelineMsg{operative, msg.Text},
			)
		}
	}

	return a, pollPhaseStream(
		a.views.Pipeline.PhaseStreamPhase,
		a.views.Pipeline.PhaseStreamCh,
		a.views.Pipeline.PhaseStreamStart,
		a.views.Pipeline.PhaseStreamCancel,
		a.views.Pipeline.PhaseStreamBuf,
	)
}
