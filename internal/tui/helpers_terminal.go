package tui

import (
	"os"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// --- Embedded Terminal ---
//
// SECURITY TRUST BOUNDARY: execTermCmd passes user input directly to bash -c.
// This is INTENTIONAL — it is a local terminal emulator for the TUI operator.
// The trust model is identical to the user opening a terminal: the operator IS
// the user. This is NOT exposed to network input, AI-generated commands, or
// untrusted sources. Input comes only from the local keyboard via the TUI
// text input widget (a.termInput). No remote or programmatic callers exist.

func (a *App) execTermCmd(input string) tea.Cmd {
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
		if err := os.Chdir(dir); err != nil {
			a.termOutput = append(a.termOutput,
				lipgloss.NewStyle().Foreground(ColorActive).Render("$ "+input),
				lipgloss.NewStyle().Foreground(ColorError).Render(err.Error()))
		} else {
			a.cwd, _ = os.Getwd()
			a.termOutput = append(a.termOutput,
				lipgloss.NewStyle().Foreground(ColorActive).Render("$ "+input))
			a.addIntel("cd → %s", a.cwd)
		}
		return nil
	}
	// Handle clear
	if input == "clear" {
		a.termOutput = nil
		return nil
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
