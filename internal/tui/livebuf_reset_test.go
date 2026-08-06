package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/rk-senne/moonbase/internal/pipeline"
)

// Superseding a running mission must clear the live streaming buffer so a prior
// mission's partial output cannot bleed into the next mission (under the wrong
// persona).
func TestSupersede_ClearsLiveBuf(t *testing.T) {
	app := NewApp()
	app.view = ViewMission
	app.boot.Ready = true
	app.views.Mission.Input.Focus()

	// Simulate an in-flight streaming mission.
	app.views.Pipeline.Running = true
	app.views.Pipeline.LiveAgent = "Numbuh 1"
	app.views.Pipeline.LiveBuf = "half-written analysis…"

	app.views.Mission.Input.SetValue("brand new mission")
	model, _ := app.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	result := model.(App)

	if result.views.Pipeline.LiveBuf != "" {
		t.Errorf("expected LiveBuf cleared on supersede, got %q", result.views.Pipeline.LiveBuf)
	}
	if result.views.Pipeline.LiveAgent != "" {
		t.Errorf("expected LiveAgent cleared on supersede, got %q", result.views.Pipeline.LiveAgent)
	}
}

// Aborting a mission must flush the partial live text into the chat history
// (preserved as a completed message) and clear the live buffer so it is not
// rendered as stale "live" output.
func TestAbort_FlushesLiveBuf(t *testing.T) {
	app := NewApp()
	app.view = ViewPipeline
	app.boot.Ready = true
	app.views.Pipeline.State = pipeline.New("abort test")
	app.views.Pipeline.Running = true
	app.views.Pipeline.LiveAgent = "Numbuh 3"
	app.views.Pipeline.LiveBuf = "partial implementation output"

	model, _ := app.Update(PipelineAbortedMsg{})
	result := model.(App)

	if result.views.Pipeline.LiveBuf != "" {
		t.Errorf("expected LiveBuf cleared after abort, got %q", result.views.Pipeline.LiveBuf)
	}
	// The partial text should have been preserved in the chat history.
	found := false
	for _, m := range result.views.Pipeline.Chat {
		if m.Agent == "Numbuh 3" && m.Content == "partial implementation output" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected partial live output to be flushed into chat on abort")
	}
}
