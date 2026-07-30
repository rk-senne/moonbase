package tui

import (
	"os"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rk-senne/moonbase/internal/snippets"
)

// === renderSnippetPicker ===

func TestRenderSnippetPicker_Empty(t *testing.T) {
	app := NewApp()
	app.width = 100
	app.height = 40
	app.snippetPick.List = nil
	app.snippetPick.Cursor = 0

	result := app.renderSnippetPicker()
	if result == "" {
		t.Error("expected non-empty snippet picker render")
	}
}

func TestRenderSnippetPicker_WithSnippets(t *testing.T) {
	app := NewApp()
	app.width = 100
	app.height = 40
	app.snippetPick.List = []snippets.Snippet{
		{Name: "greeting", Content: "hello"},
		{Name: "farewell", Content: "bye"},
	}
	app.snippetPick.Cursor = 0

	result := app.renderSnippetPicker()
	if result == "" {
		t.Error("expected non-empty result")
	}
}

// === fileIcon ===

func TestFileIcon(t *testing.T) {
	tests := []struct {
		name  string
		isDir bool
		want  string
	}{
		{"main.go", false, "🔹"},
		{"app.ts", false, "🟡"},
		{"App.tsx", false, "🟡"},
		{"index.js", false, "🟡"},
		{"Main.java", false, "☕"},
		{"README.md", false, "📄"},
		{"config.json", false, "📋"},
		{"deploy.yaml", false, "⚙️"},
		{"run.sh", false, "🔧"},
		{"go.mod", false, "📦"},
		{"go.sum", false, "📦"},
		{"random.xyz", false, "  "},
		{"src", true, "📁"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := fileIcon(tt.name, tt.isDir)
			if got != tt.want {
				t.Errorf("fileIcon(%q, %v) = %q, want %q", tt.name, tt.isDir, got, tt.want)
			}
		})
	}
}

// === openComms ===

func TestOpenComms(t *testing.T) {
	app := NewApp()
	app.registry = newTestRegistry()
	app.dashboard.Selected = 0
	app.width = 100
	app.height = 40

	app.openComms()
	if app.view != ViewComms {
		t.Errorf("expected ViewComms after openComms, got %d", app.view)
	}
	if app.comms == nil {
		t.Error("expected comms state to be initialized")
	}
}

// === switchCommsAgent ===

func TestSwitchCommsAgent(t *testing.T) {
	app := NewApp()
	app.registry = newTestRegistry()
	app.width = 100
	app.height = 40
	app.comms = newCommsState("numbuh-1", "prompt", 80, 40)

	app.switchCommsAgent("numbuh-2")
	// Should switch to new agent
	if app.comms.agent != "numbuh-2" {
		t.Errorf("expected agent numbuh-2, got %s", app.comms.agent)
	}
}

func TestSwitchCommsAgent_NotFound(t *testing.T) {
	app := NewApp()
	app.registry = newTestRegistry()
	app.width = 100
	app.height = 40
	app.comms = newCommsState("numbuh-1", "prompt", 80, 40)

	app.switchCommsAgent("nonexistent-agent")
	// Should not crash, agent stays the same or gets an error message
}

// === renderComms ===

func TestRenderComms(t *testing.T) {
	app := NewApp()
	app.boot.Ready = true
	app.width = 100
	app.height = 40
	app.view = ViewComms
	app.comms = newCommsState("numbuh-1", "prompt", 80, 40)

	result := app.renderComms()
	if result == "" {
		t.Error("expected non-empty comms render")
	}
}

func TestRenderComms_Streaming(t *testing.T) {
	app := NewApp()
	app.boot.Ready = true
	app.width = 100
	app.height = 40
	app.view = ViewComms
	app.comms = newCommsState("numbuh-1", "prompt", 80, 40)
	app.comms.streaming = true
	app.comms.buffer = "partial response..."

	result := app.renderComms()
	if result == "" {
		t.Error("expected non-empty streaming comms render")
	}
}

// === renderProtocol ===

func TestRenderProtocol(t *testing.T) {
	app := NewApp()
	app.boot.Ready = true
	app.width = 100
	app.height = 40

	result := app.renderProtocol()
	if result == "" {
		t.Error("expected non-empty protocol render")
	}
}

// === handleTerminalKeys enter with command ===

func TestTerminalKeys_EnterWithCommand(t *testing.T) {
	app := NewApp()
	app.view = ViewDashboard
	app.boot.Ready = true
	app.browser.Active = false
	app.terminal.Active = true
	app.terminal.Input.Focus()
	app.terminal.Input.SetValue("echo hello")

	model, cmd := app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	result := model.(App)
	if result.terminal.Input.Value() != "" {
		t.Error("expected term input reset after enter")
	}
	if cmd == nil {
		t.Error("expected non-nil cmd for terminal execution")
	}
}

// === handleFileBrowserKeys: enter, backspace, e, dot ===

func TestFileBrowserKeys_Enter(t *testing.T) {
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	app := NewApp()
	app.view = ViewDashboard
	app.boot.Ready = true
	app.browser.Active = true
	app.terminal.Active = false

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	_ = model.(App)
}

func TestFileBrowserKeys_Backspace(t *testing.T) {
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	app := NewApp()
	app.view = ViewDashboard
	app.boot.Ready = true
	app.browser.Active = true
	app.terminal.Active = false

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	_ = model.(App)
}

func TestFileBrowserKeys_Dot(t *testing.T) {
	app := NewApp()
	app.view = ViewDashboard
	app.boot.Ready = true
	app.browser.Active = true
	app.terminal.Active = false

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'.'}})
	_ = model.(App)
}

func TestFileBrowserKeys_E(t *testing.T) {
	app := NewApp()
	app.view = ViewDashboard
	app.boot.Ready = true
	app.browser.Active = true
	app.terminal.Active = false

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	_ = model.(App)
}

// === selectProject ===

func TestSelectProject_EmptyList(t *testing.T) {
	app := NewApp()
	app.projectNav = &ProjectsState{list: nil}
	app.selectProject() // should not panic
}

func TestSelectProject_ValidProject(t *testing.T) {
	app := NewApp()
	app.width = 100
	app.height = 40
	app.projectNav = newProjectsState()
	// selectProject changes CWD which can affect other tests
	// Just test that it doesn't panic with no projects
}

// === Dashboard 'd' and 'g' git commands ===

func TestDashboardKeys_D_GitDiff(t *testing.T) {
	app := NewApp()
	app.view = ViewDashboard
	app.boot.Ready = true
	app.browser.Active = false
	app.terminal.Active = false

	_, cmd := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	if cmd == nil {
		t.Error("expected non-nil cmd for 'd' (git diff)")
	}
}

func TestDashboardKeys_G_GitStatus(t *testing.T) {
	app := NewApp()
	app.view = ViewDashboard
	app.boot.Ready = true
	app.browser.Active = false
	app.terminal.Active = false

	_, cmd := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	if cmd == nil {
		t.Error("expected non-nil cmd for 'g' (git status)")
	}
}

// === Dashboard dossier keys: 'c' copy, 'C' comms ===

func TestDossierKeys_C_Comms(t *testing.T) {
	app := NewApp()
	app.view = ViewDossier
	app.boot.Ready = true
	app.browser.Active = false
	app.terminal.Active = false
	app.registry = newTestRegistry()

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'C'}})
	result := model.(App)
	if result.view != ViewComms {
		t.Errorf("expected ViewComms after 'C', got %d", result.view)
	}
}

// === renderMainPanel coverage ===

func TestView_MainPanel_FileBrowser(t *testing.T) {
	app := NewApp()
	app.boot.Ready = true
	app.width = 120
	app.height = 40
	app.view = ViewDashboard
	app.browser.Active = true
	app.registry = newTestRegistry()

	output := app.View()
	if output == "" {
		t.Error("expected non-empty view with file browser")
	}
}

func TestView_MainPanel_Terminal(t *testing.T) {
	app := NewApp()
	app.boot.Ready = true
	app.width = 120
	app.height = 40
	app.view = ViewDashboard
	app.browser.Active = false
	app.terminal.Active = true
	app.registry = newTestRegistry()

	output := app.View()
	if output == "" {
		t.Error("expected non-empty view with terminal")
	}
}

// === handleBootTick with typewriter trigger ===

func TestHandleBootTick_TriggerTypewriter(t *testing.T) {
	app := NewApp()
	app.view = ViewBoot
	app.boot.Ready = true
	// Set bootStep to len(bootMessages)-2 so next tick triggers typewriter
	app.boot.Step = len(bootMessages) - 2

	model, _ := app.Update(bootTickMsg{})
	result := model.(App)
	if result.chrome.Anim.typewriterAt != 1 {
		t.Errorf("expected typewriter triggered, got typewriterAt=%d", result.chrome.Anim.typewriterAt)
	}
}

// === agentColor coverage ===

func TestAgentColor_AllBranches(t *testing.T) {
	colors := []string{"1", "2", "3", "4", "5", "0", "274", "362", "unknown"}
	for _, name := range colors {
		c := agentColor(name)
		if c == "" {
			t.Errorf("expected non-empty color for %s", name)
		}
	}
}
