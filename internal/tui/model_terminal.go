package tui

import (
	"os"
	"os/exec"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// TerminalModel owns the embedded terminal state: input, output buffer, active flag,
// and current working directory. It is a value type with Update/View methods —
// deliberately NOT a tea.Model implementation (concrete return type for value semantics).
type TerminalModel struct {
	Input  textinput.Model
	Output []string
	Active bool
	Cwd    string
}

// NewTerminalModel constructs a TerminalModel with defaults.
func NewTerminalModel() TerminalModel {
	ti := textinput.New()
	ti.Placeholder = "$ "
	ti.CharLimit = 500
	ti.SetWidth(80)

	cwd, _ := os.Getwd()
	return TerminalModel{
		Input:  ti,
		Output: nil,
		Active: false,
		Cwd:    cwd,
	}
}

// Update handles key messages when the terminal is active.
// Returns the updated TerminalModel and any tea.Cmd to execute.
func (m TerminalModel) Update(msg tea.KeyPressMsg, ctx AppContext) (TerminalModel, tea.Cmd) {
	switch {
	case key.Matches(msg, ctx.Keys.TerminalEsc):
		m.Active = false
		m.Input.Blur()
	case key.Matches(msg, ctx.Keys.TerminalToBrowser):
		m.Active = false
		m.Input.Blur()
		// The browsing=true transition is handled by App after receiving the updated model
	case key.Matches(msg, ctx.Keys.TerminalSubmit):
		cmd := m.Input.Value()
		m.Input.Reset()
		if cmd != "" {
			return m, m.execCmd(cmd, ctx)
		}
	default:
		var c tea.Cmd
		m.Input, c = m.Input.Update(msg)
		return m, c
	}
	return m, nil
}

// HandleOutput processes a termOutputMsg and appends it to the output buffer.
func (m TerminalModel) HandleOutput(msg termOutputMsg, themeData Theme) TerminalModel {
	m.Output = append(m.Output, lipgloss.NewStyle().Foreground(themeData.Active).Render("$ "+msg.cmd))
	m.Output = append(m.Output, strings.Split(msg.output, "\n")...)
	if len(m.Output) > maxTerminalLines {
		m.Output = m.Output[len(m.Output)-maxTerminalLines:]
	}
	return m
}

// View renders the terminal panel content (output lines + input prompt).
// The caller (App) is responsible for framing this content within the panel border and
// interleaving intel entries.
func (m TerminalModel) View(ctx AppContext) string {
	// This is intentionally minimal — the actual terminal panel rendering interleaves
	// intel entries and is handled by App's renderMainPanel. This method is available
	// for future use when the rendering can be fully separated.
	return ""
}

// execCmd handles command execution in the embedded terminal.
//
// SECURITY TRUST BOUNDARY: passes user input directly to bash -c.
// This is INTENTIONAL — it is a local terminal emulator for the TUI operator.
// The trust model is identical to the user opening a terminal: the operator IS
// the user. Input comes only from the local keyboard via the TUI text input widget.
func (m TerminalModel) execCmd(input string, ctx AppContext) tea.Cmd {
	// Handle built-in cd
	if strings.HasPrefix(input, "cd ") {
		dir := strings.TrimPrefix(input, "cd ")
		dir = strings.TrimSpace(dir)
		if dir == "~" {
			dir, _ = os.UserHomeDir()
		} else if strings.HasPrefix(dir, "~/") {
			home, _ := os.UserHomeDir()
			dir = home + dir[1:]
		}
		// cd is handled synchronously since it changes state
		return func() tea.Msg {
			if err := os.Chdir(dir); err != nil {
				return termCdMsg{input: input, err: err}
			}
			newCwd, _ := os.Getwd()
			return termCdMsg{input: input, newCwd: newCwd}
		}
	}
	// Handle clear
	if input == "clear" {
		return func() tea.Msg {
			return termClearMsg{}
		}
	}

	return func() tea.Msg {
		out, err := exec.Command("bash", "-c", input).CombinedOutput()
		result := strings.TrimRight(string(out), "\n")
		if err != nil && result == "" {
			result = err.Error()
		}
		return termOutputMsg{cmd: input, output: result}
	}
}

// HandleCd processes a cd result and updates terminal state.
func (m TerminalModel) HandleCd(msg termCdMsg, themeData Theme) TerminalModel {
	if msg.err != nil {
		m.Output = append(m.Output,
			lipgloss.NewStyle().Foreground(themeData.Active).Render("$ "+msg.input),
			lipgloss.NewStyle().Foreground(themeData.Error).Render(msg.err.Error()))
	} else {
		m.Cwd = msg.newCwd
		m.Output = append(m.Output,
			lipgloss.NewStyle().Foreground(themeData.Active).Render("$ "+msg.input))
	}
	return m
}

// HandleClear clears the terminal output buffer.
func (m TerminalModel) HandleClear() TerminalModel {
	m.Output = nil
	return m
}

// termCdMsg is sent when a cd command completes.
type termCdMsg struct {
	input  string
	err    error
	newCwd string
}

// termClearMsg is sent when the clear command is executed.
type termClearMsg struct{}
