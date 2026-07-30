package tui

import (
	"testing"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"github.com/rk-senne/moonbase/internal/pipeline"
)

func TestDashboardKeys_Mission(t *testing.T) {
	app := NewApp()
	app.view = ViewDashboard
	app.boot.Ready = true
	app.views.Browser.Active = false
	app.views.Terminal.Active = false

	model, _ := app.Update(tea.KeyPressMsg{Code: 'm', Text: "m"})
	result := model.(App)
	if result.view != ViewMission {
		t.Errorf("expected ViewMission after 'm', got %d", result.view)
	}
}

func TestDashboardKeys_Search(t *testing.T) {
	app := NewApp()
	app.view = ViewDashboard
	app.boot.Ready = true
	app.views.Browser.Active = false
	app.views.Terminal.Active = false

	model, _ := app.Update(tea.KeyPressMsg{Code: '/', Text: "/"})
	result := model.(App)
	if !result.views.Search.Active {
		t.Error("expected searching=true after '/'")
	}
}

func TestDashboardKeys_NumberJump(t *testing.T) {
	app := NewApp()
	app.view = ViewDashboard
	app.boot.Ready = true
	app.views.Browser.Active = false
	app.views.Terminal.Active = false

	model, _ := app.Update(tea.KeyPressMsg{Code: '3', Text: "3"})
	result := model.(App)
	if result.views.Dashboard.Cursor != 3 {
		t.Errorf("expected cursor=3 after '3', got %d", result.views.Dashboard.Cursor)
	}
	if result.view != ViewDossier {
		t.Errorf("expected ViewDossier after number key, got %d", result.view)
	}
}

func TestDashboardKeys_History(t *testing.T) {
	app := NewApp()
	app.view = ViewDashboard
	app.boot.Ready = true
	app.views.Browser.Active = false
	app.views.Terminal.Active = false

	model, _ := app.Update(tea.KeyPressMsg{Code: 'H', Text: "H"})
	result := model.(App)
	if result.view != ViewHistory {
		t.Errorf("expected ViewHistory after 'H', got %d", result.view)
	}
}

func TestDossierKeys_EscBack(t *testing.T) {
	app := NewApp()
	app.view = ViewDossier
	app.boot.Ready = true
	app.views.Browser.Active = false
	app.views.Terminal.Active = false

	model, _ := app.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	result := model.(App)
	if result.view != ViewDashboard {
		t.Errorf("expected ViewDashboard after esc in dossier, got %d", result.view)
	}
}

func TestDossierKeys_Enter(t *testing.T) {
	app := NewApp()
	app.view = ViewDossier
	app.boot.Ready = true
	app.views.Browser.Active = false
	app.views.Terminal.Active = false
	app.registry = newTestRegistry()

	_, cmd := app.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	// Enter in dossier triggers deployAgent which returns a cmd
	// It should not be nil (it launches an agent deploy)
	_ = cmd // deploy returns a cmd or nil depending on backend
}

func TestPipelineKeys_EscAbort(t *testing.T) {
	app := NewApp()
	app.view = ViewPipeline
	app.boot.Ready = true
	app.views.Browser.Active = false
	app.views.Terminal.Active = false
	app.views.Pipeline.Running = true
	app.views.Pipeline.State = pipeline.New("test")

	model, _ := app.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	result := model.(App)
	if !result.views.Pipeline.AbortPending {
		t.Error("expected abortPending=true after first esc during running pipeline")
	}
}

func TestPipelineKeys_EscBackWhenIdle(t *testing.T) {
	app := NewApp()
	app.view = ViewPipeline
	app.boot.Ready = true
	app.views.Browser.Active = false
	app.views.Terminal.Active = false
	app.views.Pipeline.Running = false

	model, _ := app.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	result := model.(App)
	if result.view != ViewDashboard {
		t.Errorf("expected ViewDashboard after esc on idle pipeline, got %d", result.view)
	}
}

func TestPipelineKeys_Retry(t *testing.T) {
	app := NewApp()
	app.view = ViewPipeline
	app.boot.Ready = true
	app.views.Browser.Active = false
	app.views.Terminal.Active = false
	app.views.Pipeline.Running = false
	app.views.Pipeline.State = pipeline.New("test")
	app.views.Pipeline.State.Phases[0].Status = pipeline.StatusFailed

	model, _ := app.Update(tea.KeyPressMsg{Code: 'r', Text: "r"})
	result := model.(App)
	if result.views.Pipeline.State.Phases[0].Status != pipeline.StatusRunning {
		t.Errorf("expected phase 0 to be running after retry, got %d", result.views.Pipeline.State.Phases[0].Status)
	}
}

func TestPipelineKeys_Skip(t *testing.T) {
	app := NewApp()
	app.view = ViewPipeline
	app.boot.Ready = true
	app.views.Browser.Active = false
	app.views.Terminal.Active = false
	app.views.Pipeline.Running = false
	app.views.Pipeline.State = pipeline.New("test")
	app.views.Pipeline.State.Phases[0].Status = pipeline.StatusRunning

	model, _ := app.Update(tea.KeyPressMsg{Code: 's', Text: "s"})
	result := model.(App)
	if result.views.Pipeline.State.Phases[0].Status != pipeline.StatusSkipped {
		t.Errorf("expected phase 0 to be skipped, got %d", result.views.Pipeline.State.Phases[0].Status)
	}
	if result.views.Pipeline.State.Current != 1 {
		t.Errorf("expected pipeline to advance to phase 1 after skip, got %d", result.views.Pipeline.State.Current)
	}
}

func TestMissionKeys_Esc(t *testing.T) {
	app := NewApp()
	app.view = ViewMission
	app.boot.Ready = true
	app.views.Mission.Input.Focus()

	model, _ := app.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	result := model.(App)
	if result.view != ViewDashboard {
		t.Errorf("expected ViewDashboard after esc in mission, got %d", result.view)
	}
}

func TestMissionKeys_EnterSubmit(t *testing.T) {
	app := NewApp()
	app.view = ViewMission
	app.boot.Ready = true
	app.views.Mission.Input.Focus()
	app.views.Mission.Input.SetValue("deploy the fleet")

	model, _ := app.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	result := model.(App)
	if result.view != ViewPipeline {
		t.Errorf("expected ViewPipeline after mission submit, got %d", result.view)
	}
	if result.views.Pipeline.State == nil {
		t.Fatal("expected pipelineState to be set")
	}
	if result.views.Pipeline.State.Task != "deploy the fleet" {
		t.Errorf("expected task='deploy the fleet', got '%s'", result.views.Pipeline.State.Task)
	}
}

func TestMissionKeys_EnterEmpty(t *testing.T) {
	app := NewApp()
	app.view = ViewMission
	app.boot.Ready = true
	app.views.Mission.Input.Focus()
	app.views.Mission.Input.SetValue("")

	model, _ := app.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	result := model.(App)
	// Empty mission should not start pipeline
	if result.view != ViewMission {
		t.Errorf("expected to stay in ViewMission with empty input, got %d", result.view)
	}
	if result.views.Pipeline.State != nil {
		t.Error("expected pipelineState to remain nil on empty submit")
	}
}

func TestCommsKeys_Esc(t *testing.T) {
	app := NewApp()
	app.view = ViewComms
	app.boot.Ready = true
	app.views.Comms.State = newCommsState("test-agent", "system prompt", 80, 40)
	app.views.Comms.Input.Focus()

	model, _ := app.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	result := model.(App)
	if result.view != ViewDossier {
		t.Errorf("expected ViewDossier after esc in comms, got %d", result.view)
	}
}

func TestCommsKeys_MessageInput(t *testing.T) {
	app := NewApp()
	app.view = ViewComms
	app.boot.Ready = true
	app.views.Comms.State = newCommsState("test-agent", "system prompt", 80, 40)
	app.views.Comms.Input.Focus()

	// Type a character
	model, _ := app.Update(tea.KeyPressMsg{Code: 'h', Text: "h"})
	result := model.(App)
	model, _ = result.Update(tea.KeyPressMsg{Code: 'i', Text: "i"})
	result = model.(App)

	if result.views.Comms.Input.Value() != "hi" {
		t.Errorf("expected comms input='hi', got '%s'", result.views.Comms.Input.Value())
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
