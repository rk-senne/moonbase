package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/rk-senne/moonbase/internal/pipeline"
)

// 'm' must open the mission view even when the file browser is active (the
// default dashboard state) — this was the root cause of "can't press m at any
// stage": the browser handler swallowed the key.
func TestGlobalKeys_MissionFromActiveBrowser(t *testing.T) {
	app := NewApp()
	app.view = ViewDashboard
	app.boot.Ready = true
	app.views.Browser.Active = true // default dashboard state after boot

	model, _ := app.Update(tea.KeyPressMsg{Code: 'm', Text: "m"})
	result := model.(App)
	if result.view != ViewMission {
		t.Fatalf("expected ViewMission after 'm' with active browser, got %d", result.view)
	}
	if result.views.Browser.Active {
		t.Error("expected file browser to be released when entering mission view")
	}
}

// 'm' must work from non-dashboard views too.
func TestGlobalKeys_MissionFromOtherViews(t *testing.T) {
	for _, v := range []View{ViewProjects, ViewDocs, ViewPipeline, ViewHistory, ViewDossier, ViewHelp} {
		app := NewApp()
		app.boot.Ready = true
		app.view = v
		app.views.Browser.Active = false

		model, _ := app.Update(tea.KeyPressMsg{Code: 'm', Text: "m"})
		if got := model.(App).view; got != ViewMission {
			t.Errorf("view %d: expected ViewMission after 'm', got %d", v, got)
		}
	}
}

// 'm' must remain a literal character while composing a COMMS message.
func TestGlobalKeys_MissionNotStolenInComms(t *testing.T) {
	app := NewApp()
	app.boot.Ready = true
	app.view = ViewComms
	app.views.Comms.State = newCommsState("test-agent", "sys", 80, 40)
	app.views.Comms.Input.Focus()

	model, _ := app.Update(tea.KeyPressMsg{Code: 'm', Text: "m"})
	result := model.(App)
	if result.view != ViewComms {
		t.Fatalf("expected to stay in ViewComms, got %d", result.view)
	}
	if result.views.Comms.Input.Value() != "m" {
		t.Errorf("expected 'm' typed into comms input, got %q", result.views.Comms.Input.Value())
	}
}

// Whitespace-only mission must not submit and must not navigate away (never
// "navigate backwards" on an Enter that was meant to submit).
func TestMissionKeys_EnterWhitespaceStays(t *testing.T) {
	app := NewApp()
	app.view = ViewMission
	app.boot.Ready = true
	app.views.Mission.Input.Focus()
	app.views.Mission.Input.SetValue("   ")

	model, _ := app.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	result := model.(App)
	if result.view != ViewMission {
		t.Errorf("expected to stay in ViewMission on whitespace submit, got %d", result.view)
	}
	if result.views.Pipeline.State != nil {
		t.Error("expected no pipeline state on whitespace submit")
	}
}

// Submitting a new mission supersedes a prior running one: the old context is
// cancelled and the generation increments.
func TestMissionKeys_SupersedePriorMission(t *testing.T) {
	app := NewApp()
	app.view = ViewMission
	app.boot.Ready = true
	app.views.Mission.Input.Focus()

	cancelled := false
	app.views.Pipeline.Cancel = func() { cancelled = true }
	app.views.Pipeline.Running = true
	app.views.Pipeline.Gen = 4

	app.views.Mission.Input.SetValue("second mission")
	model, _ := app.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	result := model.(App)

	if !cancelled {
		t.Error("expected prior pipeline Cancel to be called on new submit")
	}
	if result.views.Pipeline.Gen != 5 {
		t.Errorf("expected generation to increment to 5, got %d", result.views.Pipeline.Gen)
	}
	if result.view != ViewPipeline {
		t.Errorf("expected ViewPipeline after submit, got %d", result.view)
	}
}

// A stale PhaseResultMsg from a superseded mission must be ignored, not applied
// to the freshly started pipeline.
func TestPhaseResult_StaleGenerationIgnored(t *testing.T) {
	app := NewApp()
	app.view = ViewPipeline
	app.boot.Ready = true
	app.views.Pipeline.State = pipeline.New("current")
	app.views.Pipeline.State.Phases[0].Status = pipeline.StatusRunning
	app.views.Pipeline.Running = true
	app.views.Pipeline.Gen = 2

	// Message from an older generation should be dropped (no advance).
	model, _ := app.Update(PhaseResultMsg{Phase: 1, Output: "stale", Gen: 1})
	result := model.(App)
	if result.views.Pipeline.State.Current != 0 {
		t.Errorf("expected pipeline to stay at phase 0 for stale gen, got %d", result.views.Pipeline.State.Current)
	}
	if result.views.Pipeline.State.Phases[0].Status != pipeline.StatusRunning {
		t.Errorf("expected phase 0 to remain running for stale gen, got %d", result.views.Pipeline.State.Phases[0].Status)
	}
}
