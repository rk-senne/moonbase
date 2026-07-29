package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

func TestTerminalModel_Update(t *testing.T) {
	makeCtx := func() AppContext {
		return AppContext{
			Keys:   DefaultKeyMap(),
			Styles: NewStyles(moonbaseTheme),
		}
	}

	tests := []struct {
		name       string
		msg        tea.KeyMsg
		wantActive bool
		wantReset  bool // input was reset (submit case)
	}{
		{
			name:       "TerminalEsc deactivates",
			msg:        tea.KeyMsg{Type: tea.KeyEscape},
			wantActive: false,
		},
		{
			name:       "TerminalToBrowser deactivates",
			msg:        tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("`")},
			wantActive: false,
		},
		{
			name:       "TerminalSubmit with empty input does nothing",
			msg:        tea.KeyMsg{Type: tea.KeyEnter},
			wantActive: true,
		},
		{
			name:       "Default key typing keeps active",
			msg:        tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")},
			wantActive: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewTerminalModel()
			m.Active = true
			m.Input.Focus()

			ctx := makeCtx()
			result, _ := m.Update(tt.msg, ctx)

			if result.Active != tt.wantActive {
				t.Errorf("Active = %v, want %v", result.Active, tt.wantActive)
			}
		})
	}
}

func TestTerminalModel_Update_SubmitWithInput(t *testing.T) {
	m := NewTerminalModel()
	m.Active = true
	m.Input.Focus()
	m.Input.SetValue("echo hello")

	ctx := AppContext{
		Keys:   DefaultKeyMap(),
		Styles: NewStyles(moonbaseTheme),
	}

	result, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter}, ctx)
	if result.Input.Value() != "" {
		t.Errorf("expected input reset after submit, got %q", result.Input.Value())
	}
	if cmd == nil {
		t.Error("expected non-nil cmd after submit with input")
	}
}

func TestTerminalModel_execCmd_Builtins(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantMsg interface{} // type to check
	}{
		{
			name:    "cd with directory",
			input:   "cd /tmp",
			wantMsg: termCdMsg{},
		},
		{
			name:    "cd with tilde",
			input:   "cd ~",
			wantMsg: termCdMsg{},
		},
		{
			name:    "clear",
			input:   "clear",
			wantMsg: termClearMsg{},
		},
		{
			name:    "normal command",
			input:   "echo hello",
			wantMsg: termOutputMsg{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewTerminalModel()
			ctx := AppContext{
				Keys:   DefaultKeyMap(),
				Styles: NewStyles(moonbaseTheme),
			}

			cmd := m.execCmd(tt.input, ctx)
			if cmd == nil {
				t.Fatal("expected non-nil cmd")
			}
			msg := cmd()
			switch tt.wantMsg.(type) {
			case termCdMsg:
				if _, ok := msg.(termCdMsg); !ok {
					t.Errorf("expected termCdMsg, got %T", msg)
				}
			case termClearMsg:
				if _, ok := msg.(termClearMsg); !ok {
					t.Errorf("expected termClearMsg, got %T", msg)
				}
			case termOutputMsg:
				if _, ok := msg.(termOutputMsg); !ok {
					t.Errorf("expected termOutputMsg, got %T", msg)
				}
			}
		})
	}
}

func TestTerminalModel_execCmd_CdTildePath(t *testing.T) {
	m := NewTerminalModel()
	ctx := AppContext{Keys: DefaultKeyMap(), Styles: NewStyles(moonbaseTheme)}

	cmd := m.execCmd("cd ~/Documents", ctx)
	if cmd == nil {
		t.Fatal("expected non-nil cmd")
	}
	msg := cmd()
	cdMsg, ok := msg.(termCdMsg)
	if !ok {
		t.Fatalf("expected termCdMsg, got %T", msg)
	}
	// Either succeeds (newCwd set) or fails (err set) — either path proves code ran.
	if cdMsg.err == nil && cdMsg.newCwd == "" {
		t.Error("expected either error or newCwd to be set")
	}
}

func TestTerminalModel_HandleOutput(t *testing.T) {
	m := NewTerminalModel()
	theme := moonbaseTheme

	msg := termOutputMsg{cmd: "ls", output: "file1\nfile2"}
	result := m.HandleOutput(msg, theme)

	// Should have: "$ ls" + "file1" + "file2" = 3 lines
	if len(result.Output) != 3 {
		t.Errorf("expected 3 lines, got %d", len(result.Output))
	}
}

func TestTerminalModel_HandleOutput_Truncation(t *testing.T) {
	m := NewTerminalModel()
	theme := moonbaseTheme

	// Fill beyond maxTerminalLines
	for i := 0; i < maxTerminalLines+10; i++ {
		m = m.HandleOutput(termOutputMsg{cmd: "x", output: "line"}, theme)
	}

	if len(m.Output) > maxTerminalLines {
		t.Errorf("expected output truncated to %d, got %d", maxTerminalLines, len(m.Output))
	}
}

func TestTerminalModel_HandleCd_Success(t *testing.T) {
	m := NewTerminalModel()
	theme := moonbaseTheme

	msg := termCdMsg{input: "cd /tmp", newCwd: "/tmp"}
	result := m.HandleCd(msg, theme)

	if result.Cwd != "/tmp" {
		t.Errorf("expected Cwd = /tmp, got %q", result.Cwd)
	}
	if len(result.Output) == 0 {
		t.Error("expected output to contain cd command echo")
	}
}

func TestTerminalModel_HandleCd_Error(t *testing.T) {
	m := NewTerminalModel()
	theme := moonbaseTheme

	cdErr := &testCdError{msg: "no such directory"}
	errMsg := termCdMsg{input: "cd /nonexistent", err: cdErr}
	result := m.HandleCd(errMsg, theme)

	if result.Cwd == "/nonexistent" {
		t.Error("Cwd should not change on error")
	}
	// Should have 2 lines: command echo + error
	if len(result.Output) < 2 {
		t.Errorf("expected at least 2 output lines (cmd + error), got %d", len(result.Output))
	}
}

type testCdError struct {
	msg string
}

func (e *testCdError) Error() string {
	return e.msg
}

func TestTerminalModel_HandleClear(t *testing.T) {
	m := NewTerminalModel()
	m.Output = []string{"line1", "line2", "line3"}

	result := m.HandleClear()

	if result.Output != nil {
		t.Errorf("expected nil output after clear, got %v", result.Output)
	}
}

func TestNewTerminalModel(t *testing.T) {
	m := NewTerminalModel()
	if m.Active {
		t.Error("new terminal should not be active")
	}
	if m.Input.CharLimit != 500 {
		t.Errorf("expected char limit 500, got %d", m.Input.CharLimit)
	}
	if m.Cwd == "" {
		t.Error("expected Cwd to be set to current directory")
	}
}

func TestTerminalModel_Update_EscBlursInput(t *testing.T) {
	m := NewTerminalModel()
	m.Active = true
	m.Input.Focus()

	ctx := AppContext{Keys: DefaultKeyMap(), Styles: NewStyles(moonbaseTheme)}
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyEscape}, ctx)

	if result.Active {
		t.Error("expected terminal to be inactive after esc")
	}
	// Input should be blurred
	if result.Input.Focused() {
		t.Error("expected input to be blurred after esc")
	}
}

func TestTerminalModel_Update_ToBrowserBlursInput(t *testing.T) {
	m := NewTerminalModel()
	m.Active = true
	m.Input.Focus()

	ctx := AppContext{Keys: DefaultKeyMap(), Styles: NewStyles(moonbaseTheme)}
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("`")}, ctx)

	if result.Active {
		t.Error("expected terminal to be inactive after browser switch")
	}
	if result.Input.Focused() {
		t.Error("expected input to be blurred after browser switch")
	}
}

func TestTerminalModel_execCmd_NormalCommand(t *testing.T) {
	m := NewTerminalModel()
	ctx := AppContext{Keys: DefaultKeyMap(), Styles: NewStyles(moonbaseTheme)}

	cmd := m.execCmd("echo test123", ctx)
	if cmd == nil {
		t.Fatal("expected non-nil cmd")
	}
	msg := cmd()
	result, ok := msg.(termOutputMsg)
	if !ok {
		t.Fatalf("expected termOutputMsg, got %T", msg)
	}
	if result.cmd != "echo test123" {
		t.Errorf("cmd = %q, want 'echo test123'", result.cmd)
	}
	if !strings.Contains(result.output, "test123") {
		t.Errorf("output = %q, want to contain 'test123'", result.output)
	}
}

func TestTerminalModel_View_ReturnsEmpty(t *testing.T) {
	m := NewTerminalModel()
	ctx := AppContext{Keys: DefaultKeyMap(), Styles: NewStyles(moonbaseTheme)}
	if v := m.View(ctx); v != "" {
		t.Errorf("expected empty view, got %q", v)
	}
}

func TestTerminalModel_HandleOutput_MultipleAppends(t *testing.T) {
	m := NewTerminalModel()
	theme := moonbaseTheme

	m = m.HandleOutput(termOutputMsg{cmd: "a", output: "out1"}, theme)
	m = m.HandleOutput(termOutputMsg{cmd: "b", output: "out2"}, theme)

	// Each command adds: "$ cmd" + output lines
	// a: "$ a", "out1" = 2
	// b: "$ b", "out2" = 2
	if len(m.Output) != 4 {
		t.Errorf("expected 4 output lines, got %d", len(m.Output))
	}
}

func TestTerminalModel_Update_DefaultPassesThrough(t *testing.T) {
	m := NewTerminalModel()
	m.Active = true
	m.Input.Focus()

	ctx := AppContext{Keys: DefaultKeyMap(), Styles: NewStyles(moonbaseTheme)}

	// Type a character — should be passed to the textinput
	result, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")}, ctx)

	if !result.Active {
		t.Error("expected terminal to remain active")
	}
	// The textinput model should have updated (cmd may be cursor blink)
	_ = cmd
	if result.Input.Value() != "x" {
		t.Errorf("expected input value 'x', got %q", result.Input.Value())
	}
}

func TestTerminalModel_execCmd_CdSuccess(t *testing.T) {
	m := NewTerminalModel()
	ctx := AppContext{Keys: DefaultKeyMap(), Styles: NewStyles(moonbaseTheme)}

	cmd := m.execCmd("cd /tmp", ctx)
	if cmd == nil {
		t.Fatal("expected non-nil cmd")
	}
	msg := cmd()
	cdMsg, ok := msg.(termCdMsg)
	if !ok {
		t.Fatalf("expected termCdMsg, got %T", msg)
	}
	if cdMsg.err != nil {
		t.Errorf("expected no error, got %v", cdMsg.err)
	}
	if cdMsg.newCwd == "" {
		t.Error("expected newCwd to be set")
	}
}

func TestTerminalModel_execCmd_CdInvalidDir(t *testing.T) {
	m := NewTerminalModel()
	ctx := AppContext{Keys: DefaultKeyMap(), Styles: NewStyles(moonbaseTheme)}

	cmd := m.execCmd("cd /nonexistent_dir_xyz_abc_123", ctx)
	if cmd == nil {
		t.Fatal("expected non-nil cmd")
	}
	msg := cmd()
	cdMsg, ok := msg.(termCdMsg)
	if !ok {
		t.Fatalf("expected termCdMsg, got %T", msg)
	}
	if cdMsg.err == nil {
		t.Error("expected error for non-existent directory")
	}
}

// Verify key.Matches integration works correctly
func TestTerminalModel_Update_KeyMatchesIntegration(t *testing.T) {
	ctx := AppContext{Keys: DefaultKeyMap(), Styles: NewStyles(moonbaseTheme)}

	// Verify the keys are properly configured
	escKey := tea.KeyMsg{Type: tea.KeyEscape}
	if !key.Matches(escKey, ctx.Keys.TerminalEsc) {
		t.Error("TerminalEsc should match escape key")
	}

	backtickKey := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("`")}
	if !key.Matches(backtickKey, ctx.Keys.TerminalToBrowser) {
		t.Error("TerminalToBrowser should match backtick key")
	}

	enterKey := tea.KeyMsg{Type: tea.KeyEnter}
	if !key.Matches(enterKey, ctx.Keys.TerminalSubmit) {
		t.Error("TerminalSubmit should match enter key")
	}
}
