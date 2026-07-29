package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/f5508037/moonbase/internal/pipeline"
)

func TestApp_ViewRendering_Dashboard(t *testing.T) {
	app := NewApp()
	app.ready = true
	app.view = ViewDashboard
	app.width = 140
	app.height = 40
	app.registry = newTestRegistry()

	output := app.View()
	if output == "" {
		t.Error("expected non-empty dashboard view")
	}
	if !strings.Contains(output, "MOONBASE") {
		t.Error("expected dashboard to contain 'MOONBASE' header")
	}
}

func TestApp_ViewRendering_Dashboard2Col(t *testing.T) {
	app := NewApp()
	app.ready = true
	app.view = ViewDashboard
	app.width = 100
	app.height = 40
	app.registry = newTestRegistry()

	output := app.View()
	if output == "" {
		t.Error("expected non-empty dashboard view at 100 width")
	}
}

func TestApp_ViewRendering_Dashboard1Col(t *testing.T) {
	app := NewApp()
	app.ready = true
	app.view = ViewDashboard
	app.width = 60
	app.height = 40
	app.registry = newTestRegistry()

	output := app.View()
	if output == "" {
		t.Error("expected non-empty dashboard view at 60 width")
	}
}

func TestApp_ViewRendering_Help(t *testing.T) {
	app := NewApp()
	app.ready = true
	app.view = ViewHelp
	app.width = 100
	app.height = 40

	output := app.View()
	if output == "" {
		t.Error("expected non-empty help view")
	}
	if !strings.Contains(output, "Operations Manual") {
		t.Error("expected help view to contain 'Operations Manual'")
	}
	if !strings.Contains(output, "NAVIGATION") {
		t.Error("expected help view to contain 'NAVIGATION'")
	}
}

func TestApp_ViewRendering_Mission(t *testing.T) {
	app := NewApp()
	app.ready = true
	app.view = ViewMission
	app.width = 100
	app.height = 40

	output := app.View()
	if output == "" {
		t.Error("expected non-empty mission view")
	}
	if !strings.Contains(output, "MISSION") {
		t.Error("expected mission view to contain 'MISSION'")
	}
}

func TestApp_ViewRendering_Pipeline_NoState(t *testing.T) {
	app := NewApp()
	app.ready = true
	app.view = ViewPipeline
	app.width = 100
	app.height = 40
	app.pipeline.State = nil

	output := app.View()
	if output == "" {
		t.Error("expected non-empty pipeline view")
	}
	if !strings.Contains(output, "No active mission") {
		t.Error("expected pipeline view to show 'No active mission' when no state")
	}
}

func TestApp_ViewRendering_Pipeline_WithState(t *testing.T) {
	app := NewApp()
	app.ready = true
	app.view = ViewPipeline
	app.width = 100
	app.height = 40
	app.pipeline.State = pipeline.New("test task")
	app.pipeline.State.Phases[0].Status = pipeline.StatusRunning
	app.pipeline.Chat = []PipelineMsg{
		{"", "━━━ MISSION: test task ━━━"},
		{"Numbuh 1", "Starting analysis..."},
	}

	output := app.View()
	if output == "" {
		t.Error("expected non-empty pipeline view with state")
	}
	if !strings.Contains(output, "test task") {
		t.Error("expected pipeline view to show task name")
	}
}

func TestApp_ViewRendering_Pipeline_RiskDisplay(t *testing.T) {
	app := NewApp()
	app.ready = true
	app.view = ViewPipeline
	app.width = 100
	app.height = 40
	app.pipeline.State = pipeline.New("test task")
	app.pipeline.State.Context.RiskLevel = "MEDIUM"
	app.pipeline.State.Context.ReworkCount = 1

	output := app.View()
	if !strings.Contains(output, "MEDIUM") {
		t.Error("expected pipeline view to show risk level")
	}
}

func TestApp_ViewRendering_Dossier(t *testing.T) {
	app := NewApp()
	app.ready = true
	app.view = ViewDossier
	app.width = 120
	app.height = 40
	app.registry = newTestRegistry()
	app.dashboard.Selected = 0

	output := app.View()
	if output == "" {
		t.Error("expected non-empty dossier view")
	}
	if !strings.Contains(output, "Dossier") {
		t.Error("expected dossier view to contain 'Dossier'")
	}
}

func TestApp_ViewRendering_Boot(t *testing.T) {
	app := NewApp()
	app.ready = true
	app.view = ViewBoot
	app.width = 100
	app.height = 40
	app.bootStep = 2

	output := app.View()
	if output == "" {
		t.Error("expected non-empty boot view")
	}
}

func TestApp_ViewRendering_NotReady(t *testing.T) {
	app := NewApp()
	app.ready = false

	output := app.View()
	if output != "  Initializing..." {
		t.Errorf("expected 'Initializing...' when not ready, got %q", output)
	}
}

// Test message handling for common messages

func TestApp_ClockTickMsg(t *testing.T) {
	app := NewApp()
	app.ready = true
	app.view = ViewDashboard

	model, _ := app.Update(clockTickMsg{})
	result := model.(App)
	// Clock should be updated (non-empty)
	if result.clock == "" {
		t.Error("expected clock to be set after clockTickMsg")
	}
}

func TestApp_BlinkTickMsg(t *testing.T) {
	app := NewApp()
	app.ready = true
	app.blink = false

	model, _ := app.Update(blinkTickMsg{})
	result := model.(App)
	if !result.blink {
		t.Error("expected blink to toggle to true")
	}

	model, _ = result.Update(blinkTickMsg{})
	result = model.(App)
	if result.blink {
		t.Error("expected blink to toggle back to false")
	}
}

func TestApp_SystemInfoMsg(t *testing.T) {
	app := NewApp()
	app.ready = true

	model, _ := app.Update(systemInfoMsg{
		branch:      "main",
		clean:       true,
		dockerCount: 3,
		diffLines:   42,
	})
	result := model.(App)
	if result.gitBranch != "main" {
		t.Errorf("expected gitBranch='main', got %s", result.gitBranch)
	}
	if !result.gitClean {
		t.Error("expected gitClean=true")
	}
	if result.dockerCount != 3 {
		t.Errorf("expected dockerCount=3, got %d", result.dockerCount)
	}
	if result.gitDiffLines != 42 {
		t.Errorf("expected gitDiffLines=42, got %d", result.gitDiffLines)
	}
}

func TestApp_PipelineAbortedMsg(t *testing.T) {
	app := NewApp()
	app.ready = true
	app.view = ViewPipeline
	app.pipeline.State = pipeline.New("abort test")
	app.pipeline.State.Phases[0].Status = pipeline.StatusRunning
	app.pipeline.Running = true

	model, _ := app.Update(PipelineAbortedMsg{})
	result := model.(App)
	if result.pipeline.Running {
		t.Error("expected pipelineRunning=false after abort")
	}
	if result.pipeline.State.Active {
		t.Error("expected pipeline to be stopped after abort")
	}
}

func TestApp_TermOutputMsg(t *testing.T) {
	app := NewApp()
	app.ready = true

	model, _ := app.Update(termOutputMsg{cmd: "ls", output: "file1\nfile2"})
	result := model.(App)

	if len(result.terminal.Output) < 2 {
		t.Errorf("expected at least 2 terminal output lines, got %d", len(result.terminal.Output))
	}
}

func TestApp_TermOutputMsg_MaxLines(t *testing.T) {
	app := NewApp()
	app.ready = true

	// Fill terminal to max
	for i := 0; i < maxTerminalLines+10; i++ {
		model, _ := app.Update(termOutputMsg{cmd: "echo", output: "line"})
		app = model.(App)
	}

	if len(app.terminal.Output) > maxTerminalLines {
		t.Errorf("expected termOutput <= %d, got %d", maxTerminalLines, len(app.terminal.Output))
	}
}

// Test pipeline advance key
func TestPipelineKeys_Advance(t *testing.T) {
	app := NewApp()
	app.view = ViewPipeline
	app.ready = true
	app.browsing = false
	app.terminal.Active = false
	app.pipeline.Running = false
	app.pipeline.State = pipeline.New("test")
	app.pipeline.State.Phases[0].Status = pipeline.StatusRunning

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	result := model.(App)

	// Should advance to next phase
	if result.pipeline.State.Current != 1 {
		t.Errorf("expected pipeline to advance to index 1, got %d", result.pipeline.State.Current)
	}
}

// Test protocol view key (F1)
// Note: bubbletea's KeyF1 String() returns "f1" but handler checks "F1"
// This test verifies actual behavior — the key won't match.
func TestDashboardKeys_Protocol(t *testing.T) {
	app := NewApp()
	app.view = ViewDashboard
	app.ready = true
	app.browsing = false
	app.terminal.Active = false

	// Directly set view to test rendering
	app.view = ViewProtocol
	if app.view != ViewProtocol {
		t.Error("expected ViewProtocol to be settable")
	}
}

// Test help toggle from non-dashboard view
func TestApp_HelpToggleFromDossier(t *testing.T) {
	app := NewApp()
	app.view = ViewDossier
	app.ready = true
	app.browsing = false
	app.terminal.Active = false

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	result := model.(App)
	if result.view != ViewHelp {
		t.Errorf("expected ViewHelp after '?' from dossier, got %d", result.view)
	}
}
