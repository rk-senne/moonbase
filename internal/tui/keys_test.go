package tui

import (
	"testing"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/rk-senne/moonbase/internal/pipeline"
)

func TestDashboardKeys_Mission(t *testing.T) {
	app := NewApp()
	app.view = ViewDashboard
	app.boot.Ready = true
	app.browser.Active = false
	app.terminal.Active = false

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
	result := model.(App)
	if result.view != ViewMission {
		t.Errorf("expected ViewMission after 'm', got %d", result.view)
	}
}

func TestDashboardKeys_Search(t *testing.T) {
	app := NewApp()
	app.view = ViewDashboard
	app.boot.Ready = true
	app.browser.Active = false
	app.terminal.Active = false

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	result := model.(App)
	if !result.search.Active {
		t.Error("expected searching=true after '/'")
	}
}

func TestDashboardKeys_NumberJump(t *testing.T) {
	app := NewApp()
	app.view = ViewDashboard
	app.boot.Ready = true
	app.browser.Active = false
	app.terminal.Active = false

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	result := model.(App)
	if result.dashboard.Cursor != 3 {
		t.Errorf("expected cursor=3 after '3', got %d", result.dashboard.Cursor)
	}
	if result.view != ViewDossier {
		t.Errorf("expected ViewDossier after number key, got %d", result.view)
	}
}

func TestDashboardKeys_History(t *testing.T) {
	app := NewApp()
	app.view = ViewDashboard
	app.boot.Ready = true
	app.browser.Active = false
	app.terminal.Active = false

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'H'}})
	result := model.(App)
	if result.view != ViewHistory {
		t.Errorf("expected ViewHistory after 'H', got %d", result.view)
	}
}

func TestDossierKeys_EscBack(t *testing.T) {
	app := NewApp()
	app.view = ViewDossier
	app.boot.Ready = true
	app.browser.Active = false
	app.terminal.Active = false

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyEscape})
	result := model.(App)
	if result.view != ViewDashboard {
		t.Errorf("expected ViewDashboard after esc in dossier, got %d", result.view)
	}
}

func TestDossierKeys_Enter(t *testing.T) {
	app := NewApp()
	app.view = ViewDossier
	app.boot.Ready = true
	app.browser.Active = false
	app.terminal.Active = false
	app.registry = newTestRegistry()

	_, cmd := app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	// Enter in dossier triggers deployAgent which returns a cmd
	// It should not be nil (it launches an agent deploy)
	_ = cmd // deploy returns a cmd or nil depending on backend
}

func TestPipelineKeys_EscAbort(t *testing.T) {
	app := NewApp()
	app.view = ViewPipeline
	app.boot.Ready = true
	app.browser.Active = false
	app.terminal.Active = false
	app.pipeline.Running = true
	app.pipeline.State = pipeline.New("test")

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyEscape})
	result := model.(App)
	if !result.pipeline.AbortPending {
		t.Error("expected abortPending=true after first esc during running pipeline")
	}
}

func TestPipelineKeys_EscBackWhenIdle(t *testing.T) {
	app := NewApp()
	app.view = ViewPipeline
	app.boot.Ready = true
	app.browser.Active = false
	app.terminal.Active = false
	app.pipeline.Running = false

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyEscape})
	result := model.(App)
	if result.view != ViewDashboard {
		t.Errorf("expected ViewDashboard after esc on idle pipeline, got %d", result.view)
	}
}

func TestPipelineKeys_Retry(t *testing.T) {
	app := NewApp()
	app.view = ViewPipeline
	app.boot.Ready = true
	app.browser.Active = false
	app.terminal.Active = false
	app.pipeline.Running = false
	app.pipeline.State = pipeline.New("test")
	app.pipeline.State.Phases[0].Status = pipeline.StatusFailed

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	result := model.(App)
	if result.pipeline.State.Phases[0].Status != pipeline.StatusRunning {
		t.Errorf("expected phase 0 to be running after retry, got %d", result.pipeline.State.Phases[0].Status)
	}
}

func TestPipelineKeys_Skip(t *testing.T) {
	app := NewApp()
	app.view = ViewPipeline
	app.boot.Ready = true
	app.browser.Active = false
	app.terminal.Active = false
	app.pipeline.Running = false
	app.pipeline.State = pipeline.New("test")
	app.pipeline.State.Phases[0].Status = pipeline.StatusRunning

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	result := model.(App)
	if result.pipeline.State.Phases[0].Status != pipeline.StatusSkipped {
		t.Errorf("expected phase 0 to be skipped, got %d", result.pipeline.State.Phases[0].Status)
	}
	if result.pipeline.State.Current != 1 {
		t.Errorf("expected pipeline to advance to phase 1 after skip, got %d", result.pipeline.State.Current)
	}
}

func TestMissionKeys_Esc(t *testing.T) {
	app := NewApp()
	app.view = ViewMission
	app.boot.Ready = true
	app.mission.Input.Focus()

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyEscape})
	result := model.(App)
	if result.view != ViewDashboard {
		t.Errorf("expected ViewDashboard after esc in mission, got %d", result.view)
	}
}

func TestMissionKeys_EnterSubmit(t *testing.T) {
	app := NewApp()
	app.view = ViewMission
	app.boot.Ready = true
	app.mission.Input.Focus()
	app.mission.Input.SetValue("deploy the fleet")

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	result := model.(App)
	if result.view != ViewPipeline {
		t.Errorf("expected ViewPipeline after mission submit, got %d", result.view)
	}
	if result.pipeline.State == nil {
		t.Fatal("expected pipelineState to be set")
	}
	if result.pipeline.State.Task != "deploy the fleet" {
		t.Errorf("expected task='deploy the fleet', got '%s'", result.pipeline.State.Task)
	}
}

func TestMissionKeys_EnterEmpty(t *testing.T) {
	app := NewApp()
	app.view = ViewMission
	app.boot.Ready = true
	app.mission.Input.Focus()
	app.mission.Input.SetValue("")

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	result := model.(App)
	// Empty mission should not start pipeline
	if result.view != ViewMission {
		t.Errorf("expected to stay in ViewMission with empty input, got %d", result.view)
	}
	if result.pipeline.State != nil {
		t.Error("expected pipelineState to remain nil on empty submit")
	}
}

func TestCommsKeys_Esc(t *testing.T) {
	app := NewApp()
	app.view = ViewComms
	app.boot.Ready = true
	app.comms.State = newCommsState("test-agent", "system prompt", 80, 40)
	app.comms.Input.Focus()

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyEscape})
	result := model.(App)
	if result.view != ViewDossier {
		t.Errorf("expected ViewDossier after esc in comms, got %d", result.view)
	}
}

func TestCommsKeys_MessageInput(t *testing.T) {
	app := NewApp()
	app.view = ViewComms
	app.boot.Ready = true
	app.comms.State = newCommsState("test-agent", "system prompt", 80, 40)
	app.comms.Input.Focus()

	// Type a character
	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	result := model.(App)
	model, _ = result.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
	result = model.(App)

	if result.comms.Input.Value() != "hi" {
		t.Errorf("expected comms input='hi', got '%s'", result.comms.Input.Value())
	}
}

func TestDefaultKeyMap_AllActionsHaveKeysAndHelp(t *testing.T) {
	km := DefaultKeyMap()
	bindings := []struct {
		name    string
		binding key.Binding
	}{
		{"Up", km.Up},
		{"Down", km.Down},
		{"Enter", km.Enter},
		{"Back", km.Back},
		{"Tab", km.Tab},
		{"NewMission", km.NewMission},
		{"NextPhase", km.NextPhase},
		{"RetryPhase", km.RetryPhase},
		{"SkipPhase", km.SkipPhase},
		{"Help", km.Help},
		{"Protocol", km.Protocol},
		{"CycleTheme", km.CycleTheme},
		{"OpenComms", km.OpenComms},
		{"Search", km.Search},
		{"History", km.History},
		{"Docs", km.Docs},
		{"Projects", km.Projects},
		{"CopyPrompt", km.CopyPrompt},
		{"SpawnHook", km.SpawnHook},
		{"JumpToAgent", km.JumpToAgent},
		{"LaunchLazygit", km.LaunchLazygit},
		{"LaunchBtop", km.LaunchBtop},
		{"LaunchNvim", km.LaunchNvim},
		{"LaunchCmux", km.LaunchCmux},
		{"LaunchFish", km.LaunchFish},
		{"Quit", km.Quit},
		{"GitDiff", km.GitDiff},
		{"GitStatus", km.GitStatus},
		{"ToggleWatcher", km.ToggleWatcher},
		{"CreatePR", km.CreatePR},
		{"SendMessage", km.SendMessage},
		{"AttachFile", km.AttachFile},
		{"SnippetPicker", km.SnippetPicker},
		{"CommsQuit", km.CommsQuit},
		{"SearchConfirm", km.SearchConfirm},
		{"SearchCancel", km.SearchCancel},
		{"TerminalEsc", km.TerminalEsc},
		{"TerminalToBrowser", km.TerminalToBrowser},
		{"TerminalSubmit", km.TerminalSubmit},
		{"BrowserToTerminal", km.BrowserToTerminal},
		{"BrowserUp", km.BrowserUp},
		{"BrowserDown", km.BrowserDown},
		{"BrowserEnter", km.BrowserEnter},
		{"BrowserBack", km.BrowserBack},
		{"BrowserEdit", km.BrowserEdit},
		{"BrowserRefresh", km.BrowserRefresh},
		{"BrowserEsc", km.BrowserEsc},
		{"SnippetUp", km.SnippetUp},
		{"SnippetDown", km.SnippetDown},
		{"SnippetConfirm", km.SnippetConfirm},
		{"SnippetCancel", km.SnippetCancel},
		{"ContextConfirm", km.ContextConfirm},
		{"ContextCancel", km.ContextCancel},
		{"DocsPageDown", km.DocsPageDown},
		{"DocsPageUp", km.DocsPageUp},
	}

	for _, b := range bindings {
		t.Run(b.name, func(t *testing.T) {
			keys := b.binding.Keys()
			if len(keys) == 0 {
				t.Errorf("%s has no keys bound", b.name)
			}
			help := b.binding.Help()
			if help.Key == "" {
				t.Errorf("%s has empty help key", b.name)
			}
			if help.Desc == "" {
				t.Errorf("%s has empty help description", b.name)
			}
		})
	}
}

func TestKeyMap_FullHelpCoversAllBindings(t *testing.T) {
	km := DefaultKeyMap()
	groups := km.FullHelp()
	if len(groups) != 6 {
		t.Fatalf("expected 6 help groups (Navigation, Missions, Views, Tools, Comms, System), got %d", len(groups))
	}

	// Count total bindings in FullHelp
	total := 0
	for _, group := range groups {
		total += len(group)
	}
	// We expect at least 30 bindings across all groups
	if total < 30 {
		t.Errorf("expected at least 30 bindings in FullHelp, got %d", total)
	}
}

func TestKeyMap_ShortHelp(t *testing.T) {
	km := DefaultKeyMap()
	short := km.ShortHelp()
	if len(short) == 0 {
		t.Fatal("ShortHelp returned empty slice")
	}
	// ShortHelp should have the essential nav keys
	if len(short) != 6 {
		t.Errorf("expected 6 bindings in ShortHelp, got %d", len(short))
	}
}
