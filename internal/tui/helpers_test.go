package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestCommsKeys_SnippetPicker(t *testing.T) {
	app := NewApp()
	app.view = ViewComms
	app.boot.Ready = true
	app.views.Comms.State = newCommsState("test-agent", "system prompt", 80, 40)
	app.views.Comms.Input.Focus()
	app.views.SnippetPick.Active = true
	app.views.SnippetPick.List = nil

	// Navigate in snippet picker
	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	result := model.(App)
	_ = result

	// Esc exits snippet picker
	model, _ = app.Update(tea.KeyMsg{Type: tea.KeyEscape})
	result = model.(App)
	if result.views.SnippetPick.Active {
		t.Error("expected snippetPick.Active=false after esc")
	}
}

func TestCommsKeys_ContextFile(t *testing.T) {
	app := NewApp()
	app.view = ViewComms
	app.boot.Ready = true
	app.views.Comms.State = newCommsState("test-agent", "system prompt", 80, 40)
	app.views.Comms.Input.Focus()
	app.views.CtxFile.Active = true
	app.views.CtxFile.Input.Focus()

	// Esc exits context file mode
	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyEscape})
	result := model.(App)
	if result.views.CtxFile.Active {
		t.Error("expected ctxFile.Active=false after esc")
	}
}

func TestCommsKeys_ContextFileEnter(t *testing.T) {
	app := NewApp()
	app.view = ViewComms
	app.boot.Ready = true
	app.views.Comms.State = newCommsState("test-agent", "system prompt", 80, 40)
	app.views.Comms.Input.Focus()
	app.views.CtxFile.Active = true
	app.views.CtxFile.Input.Focus()
	app.views.CtxFile.Input.SetValue("/nonexistent/path")

	// Enter with invalid path should exit context file mode
	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	result := model.(App)
	if result.views.CtxFile.Active {
		t.Error("expected ctxFile.Active=false after enter")
	}
}

func TestCommsKeys_AtSwitch(t *testing.T) {
	app := NewApp()
	app.view = ViewComms
	app.boot.Ready = true
	app.views.Comms.State = newCommsState("test-agent", "system prompt", 80, 40)
	app.views.Comms.Input.Focus()
	app.views.Comms.Input.SetValue("@numbuh-1")

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	result := model.(App)
	// Should have reset input after @ command
	if result.views.Comms.Input.Value() != "" {
		t.Errorf("expected comms input reset after @ command, got '%s'", result.views.Comms.Input.Value())
	}
}

func TestFileBrowserKeys_Esc(t *testing.T) {
	app := NewApp()
	app.view = ViewDashboard
	app.boot.Ready = true
	app.views.Browser.Active = true
	app.views.Terminal.Active = false

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyEscape})
	result := model.(App)
	if result.views.Browser.Active {
		t.Error("expected browsing=false after esc in file browser")
	}
}

func TestFileBrowserKeys_Navigate(t *testing.T) {
	app := NewApp()
	app.view = ViewDashboard
	app.boot.Ready = true
	app.views.Browser.Active = true
	app.views.Terminal.Active = false

	// Navigate down
	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	result := model.(App)
	_ = result // just ensure no crash
}

func TestTerminalKeys_Esc(t *testing.T) {
	app := NewApp()
	app.view = ViewDashboard
	app.boot.Ready = true
	app.views.Browser.Active = false
	app.views.Terminal.Active = true
	app.views.Terminal.Input.Focus()

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyEscape})
	result := model.(App)
	if result.views.Terminal.Active {
		t.Error("expected termActive=false after esc in terminal")
	}
}

func TestTerminalKeys_Backtick(t *testing.T) {
	app := NewApp()
	app.view = ViewDashboard
	app.boot.Ready = true
	app.views.Browser.Active = false
	app.views.Terminal.Active = true
	app.views.Terminal.Input.Focus()

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'`'}})
	result := model.(App)
	if result.views.Terminal.Active {
		t.Error("expected termActive=false after backtick")
	}
	if !result.views.Browser.Active {
		t.Error("expected browsing=true after backtick from terminal")
	}
}

func TestFileBrowserKeys_Backtick(t *testing.T) {
	app := NewApp()
	app.view = ViewDashboard
	app.boot.Ready = true
	app.views.Browser.Active = true
	app.views.Terminal.Active = false

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'`'}})
	result := model.(App)
	if result.views.Browser.Active {
		t.Error("expected browsing=false after backtick from browser")
	}
	if !result.views.Terminal.Active {
		t.Error("expected termActive=true after backtick from browser")
	}
}

func TestApp_StreamChunkMsg_NilComms(t *testing.T) {
	app := NewApp()
	app.boot.Ready = true
	app.views.Comms.State = nil

	msg := streamChunkMsg{text: "hello", done: false}
	model, _ := app.Update(msg)
	result := model.(App)
	// Should not crash with nil comms
	_ = result
}

func TestApp_StreamChunkMsg_Done(t *testing.T) {
	app := NewApp()
	app.boot.Ready = true
	app.views.Comms.State = newCommsState("test-agent", "system prompt", 80, 40)
	app.views.Comms.State.streaming = true

	msg := streamChunkMsg{done: true}
	model, _ := app.Update(msg)
	result := model.(App)
	if result.views.Comms.State.streaming {
		t.Error("expected streaming=false after done message")
	}
}

func TestApp_StreamChunkMsg_Error(t *testing.T) {
	app := NewApp()
	app.boot.Ready = true
	app.views.Comms.State = newCommsState("test-agent", "system prompt", 80, 40)
	app.views.Comms.State.streaming = true

	msg := streamChunkMsg{err: errTestDummy}
	model, _ := app.Update(msg)
	result := model.(App)
	if result.views.Comms.State.streaming {
		t.Error("expected streaming=false after error")
	}
}

var errTestDummy = dummyError{}

type dummyError struct{}

func (d dummyError) Error() string { return "test error" }

func TestApp_WordWrap(t *testing.T) {
	tests := []struct {
		name   string
		text   string
		width  int
		expect int // minimum number of lines
	}{
		{"short text", "hello world", 80, 1},
		{"wrap needed", "this is a very long sentence that should definitely wrap around", 20, 3},
		{"zero width", "text", 0, 1},
		{"empty", "", 80, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := wordWrap(tt.text, tt.width)
			if tt.text == "" && result != "" {
				t.Errorf("expected empty result for empty input")
			}
			if tt.expect > 0 {
				lines := len(splitLines(result))
				if lines < tt.expect {
					t.Errorf("expected at least %d lines, got %d", tt.expect, lines)
				}
			}
		})
	}
}

func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

func TestApp_ExtractPersonality(t *testing.T) {
	tests := []struct {
		name   string
		prompt string
		empty  bool
	}{
		{"with personality", "## Identity\nPersonality: Bold and brave\n## Purpose", false},
		{"no personality", "## Identity\n## Purpose", true},
		{"empty prompt", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractPersonality(tt.prompt)
			if tt.empty && result != "" {
				t.Errorf("expected empty result, got %q", result)
			}
			if !tt.empty && result == "" {
				t.Error("expected non-empty personality extraction")
			}
		})
	}
}

func TestApp_ThreatGauge(t *testing.T) {
	app := NewApp()
	app.width = 100
	app.height = 40

	tests := []struct {
		name   string
		system SystemModel
		want   string
	}{
		{"clean", SystemModel{Clean: true}, "LOW"},
		{"medium", SystemModel{ChangedLines: 150, FilesChanged: 6}, "MEDIUM"},
		{"sensitive forces high", SystemModel{FilesChanged: 1, SensitiveHits: 1}, "HIGH"},
		{"critical", SystemModel{ChangedLines: 600, FilesChanged: 12, UntrackedFiles: 5, SensitiveHits: 2}, "CRITICAL"},
	}

	for _, tt := range tests {
		app.env.System = tt.system
		gauge := app.renderThreatGauge(30)
		if !strings.Contains(gauge, tt.want) {
			t.Errorf("%s: expected gauge to contain %q, got %q", tt.name, tt.want, gauge)
		}
	}
}
