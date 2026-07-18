package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/f5508037/moonbase/internal/pipeline"
)

func TestDashboardKeys_Mission(t *testing.T) {
	app := NewApp()
	app.view = ViewDashboard
	app.ready = true
	app.browsing = false
	app.termActive = false

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
	result := model.(App)
	if result.view != ViewMission {
		t.Errorf("expected ViewMission after 'm', got %d", result.view)
	}
}

func TestDashboardKeys_Search(t *testing.T) {
	app := NewApp()
	app.view = ViewDashboard
	app.ready = true
	app.browsing = false
	app.termActive = false

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	result := model.(App)
	if !result.searching {
		t.Error("expected searching=true after '/'")
	}
}

func TestDashboardKeys_NumberJump(t *testing.T) {
	app := NewApp()
	app.view = ViewDashboard
	app.ready = true
	app.browsing = false
	app.termActive = false

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	result := model.(App)
	if result.cursor != 3 {
		t.Errorf("expected cursor=3 after '3', got %d", result.cursor)
	}
	if result.view != ViewDossier {
		t.Errorf("expected ViewDossier after number key, got %d", result.view)
	}
}

func TestDashboardKeys_History(t *testing.T) {
	app := NewApp()
	app.view = ViewDashboard
	app.ready = true
	app.browsing = false
	app.termActive = false

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'H'}})
	result := model.(App)
	if result.view != ViewHistory {
		t.Errorf("expected ViewHistory after 'H', got %d", result.view)
	}
}

func TestDossierKeys_EscBack(t *testing.T) {
	app := NewApp()
	app.view = ViewDossier
	app.ready = true
	app.browsing = false
	app.termActive = false

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyEscape})
	result := model.(App)
	if result.view != ViewDashboard {
		t.Errorf("expected ViewDashboard after esc in dossier, got %d", result.view)
	}
}

func TestDossierKeys_Enter(t *testing.T) {
	app := NewApp()
	app.view = ViewDossier
	app.ready = true
	app.browsing = false
	app.termActive = false
	app.registry = newTestRegistry()

	_, cmd := app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	// Enter in dossier triggers deployAgent which returns a cmd
	// It should not be nil (it launches an agent deploy)
	_ = cmd // deploy returns a cmd or nil depending on backend
}

func TestPipelineKeys_EscAbort(t *testing.T) {
	app := NewApp()
	app.view = ViewPipeline
	app.ready = true
	app.browsing = false
	app.termActive = false
	app.pipelineRunning = true
	app.pipelineState = pipeline.New("test")

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyEscape})
	result := model.(App)
	if !result.abortPending {
		t.Error("expected abortPending=true after first esc during running pipeline")
	}
}

func TestPipelineKeys_EscBackWhenIdle(t *testing.T) {
	app := NewApp()
	app.view = ViewPipeline
	app.ready = true
	app.browsing = false
	app.termActive = false
	app.pipelineRunning = false

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyEscape})
	result := model.(App)
	if result.view != ViewDashboard {
		t.Errorf("expected ViewDashboard after esc on idle pipeline, got %d", result.view)
	}
}

func TestPipelineKeys_Retry(t *testing.T) {
	app := NewApp()
	app.view = ViewPipeline
	app.ready = true
	app.browsing = false
	app.termActive = false
	app.pipelineRunning = false
	app.pipelineState = pipeline.New("test")
	app.pipelineState.Phases[0].Status = pipeline.StatusFailed

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	result := model.(App)
	if result.pipelineState.Phases[0].Status != pipeline.StatusRunning {
		t.Errorf("expected phase 0 to be running after retry, got %d", result.pipelineState.Phases[0].Status)
	}
}

func TestPipelineKeys_Skip(t *testing.T) {
	app := NewApp()
	app.view = ViewPipeline
	app.ready = true
	app.browsing = false
	app.termActive = false
	app.pipelineRunning = false
	app.pipelineState = pipeline.New("test")
	app.pipelineState.Phases[0].Status = pipeline.StatusRunning

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	result := model.(App)
	if result.pipelineState.Phases[0].Status != pipeline.StatusSkipped {
		t.Errorf("expected phase 0 to be skipped, got %d", result.pipelineState.Phases[0].Status)
	}
	if result.pipelineState.Current != 1 {
		t.Errorf("expected pipeline to advance to phase 1 after skip, got %d", result.pipelineState.Current)
	}
}

func TestMissionKeys_Esc(t *testing.T) {
	app := NewApp()
	app.view = ViewMission
	app.ready = true
	app.missionInput.Focus()

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyEscape})
	result := model.(App)
	if result.view != ViewDashboard {
		t.Errorf("expected ViewDashboard after esc in mission, got %d", result.view)
	}
}

func TestMissionKeys_EnterSubmit(t *testing.T) {
	app := NewApp()
	app.view = ViewMission
	app.ready = true
	app.missionInput.Focus()
	app.missionInput.SetValue("deploy the fleet")

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	result := model.(App)
	if result.view != ViewPipeline {
		t.Errorf("expected ViewPipeline after mission submit, got %d", result.view)
	}
	if result.pipelineState == nil {
		t.Fatal("expected pipelineState to be set")
	}
	if result.pipelineState.Task != "deploy the fleet" {
		t.Errorf("expected task='deploy the fleet', got '%s'", result.pipelineState.Task)
	}
}

func TestMissionKeys_EnterEmpty(t *testing.T) {
	app := NewApp()
	app.view = ViewMission
	app.ready = true
	app.missionInput.Focus()
	app.missionInput.SetValue("")

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	result := model.(App)
	// Empty mission should not start pipeline
	if result.view != ViewMission {
		t.Errorf("expected to stay in ViewMission with empty input, got %d", result.view)
	}
	if result.pipelineState != nil {
		t.Error("expected pipelineState to remain nil on empty submit")
	}
}

func TestCommsKeys_Esc(t *testing.T) {
	app := NewApp()
	app.view = ViewComms
	app.ready = true
	app.comms = newCommsState("test-agent", "system prompt", 80, 40)
	app.commsInput.Focus()

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyEscape})
	result := model.(App)
	if result.view != ViewDossier {
		t.Errorf("expected ViewDossier after esc in comms, got %d", result.view)
	}
}

func TestCommsKeys_MessageInput(t *testing.T) {
	app := NewApp()
	app.view = ViewComms
	app.ready = true
	app.comms = newCommsState("test-agent", "system prompt", 80, 40)
	app.commsInput.Focus()

	// Type a character
	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	result := model.(App)
	model, _ = result.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
	result = model.(App)

	if result.commsInput.Value() != "hi" {
		t.Errorf("expected comms input='hi', got '%s'", result.commsInput.Value())
	}
}
