package tui

import (
	"fmt"
	"os/exec"
	"runtime"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"github.com/rk-senne/moonbase/internal/tools"
)

// toolInstallDoneMsg reports the outcome of an install attempt.
type toolInstallDoneMsg struct {
	display string
	ok      bool
	err     error
	note    string // set when install was not attempted (manual / no manager)
}

// enterToolsView opens the Tools arsenal with a fresh model. Releases the file
// browser so it cannot capture keys behind the view.
func (a *App) enterToolsView() {
	a.views.Browser.Active = false
	a.views.Tools = NewToolsModel()
	a.view = ViewTools
}

// handleToolsKeys handles keys for the Tools view, including the install
// confirmation sub-mode. moonbase NEVER installs without an explicit 'y'.
func (a App) handleToolsKeys(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	tm := &a.views.Tools
	n := len(tm.Catalog)

	// Confirmation sub-mode: only 'y' proceeds; everything else cancels.
	if tm.Confirm != "" {
		switch msg.String() {
		case "y", "Y":
			tool, ok := tm.toolByID(tm.Confirm)
			tm.Confirm = ""
			if !ok {
				return a, nil
			}
			tm.Result = fmt.Sprintf("Installing %s…", tool.Display)
			return a, a.installTool(tool)
		default:
			tm.Confirm = ""
			tm.Result = "Install cancelled."
			return a, nil
		}
	}

	switch {
	case key.Matches(msg, a.keys.Back):
		a.view = ViewDashboard
	case key.Matches(msg, a.keys.Up):
		if tm.Cursor > 0 {
			tm.Cursor--
		}
	case key.Matches(msg, a.keys.Down):
		if tm.Cursor < n-1 {
			tm.Cursor++
		}
	case key.Matches(msg, a.keys.Enter), key.Matches(msg, a.keys.Tools):
		if n == 0 {
			return a, nil
		}
		a.requestInstall(tm.Catalog[tm.Cursor])
	}
	return a, nil
}

// requestInstall validates that a tool can be installed on this host and, if so,
// arms the confirmation prompt. Otherwise it records a helpful result message
// (already installed / macOS-only / manual / no package manager) — never an
// install without confirmation.
func (a *App) requestInstall(tool tools.Tool) {
	tm := &a.views.Tools

	if tools.IsInstalled(tool.ID) {
		tm.Result = fmt.Sprintf("%s is already installed.", tool.Display)
		return
	}
	if tool.MacOnly && runtime.GOOS != "darwin" {
		tm.Result = fmt.Sprintf("%s is macOS-only — %s", tool.Display, tool.Manual)
		return
	}
	mgr, ok := tools.DetectManager()
	if !ok {
		tm.Result = "No supported package manager detected — " + tool.Manual
		return
	}
	if _, ok, reason := tools.BuildInstall(tool, mgr); !ok {
		tm.Result = "Manual install required — " + reason
		return
	}
	// Everything checks out — ask for confirmation.
	tm.Confirm = tool.ID
	tm.Result = ""
}

// installTool runs the package-manager install for a tool via ExecProcess so the
// operator sees live output (and any sudo prompt). Commands are built from the
// curated catalog + detected manager only — never from user input.
func (a App) installTool(t tools.Tool) tea.Cmd {
	mgr, ok := tools.DetectManager()
	if !ok {
		return func() tea.Msg {
			return toolInstallDoneMsg{display: t.Display, note: "no package manager"}
		}
	}
	plan, ok, reason := tools.BuildInstall(t, mgr)
	if !ok {
		return func() tea.Msg {
			return toolInstallDoneMsg{display: t.Display, note: reason}
		}
	}
	c := exec.Command(plan.Bin, plan.Args...)
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return toolInstallDoneMsg{display: t.Display, ok: err == nil, err: err}
	})
}

// handleToolInstallDone records the outcome and refreshes tool availability so
// the ✓/✗ markers update.
func (a App) handleToolInstallDone(msg toolInstallDoneMsg) (tea.Model, tea.Cmd) {
	switch {
	case msg.note != "":
		a.views.Tools.Result = fmt.Sprintf("⚠️ %s: %s", msg.display, msg.note)
	case msg.ok:
		a.addIntel("Installed %s", msg.display)
		a.views.Tools.Result = fmt.Sprintf("✅ %s installed.", msg.display)
	default:
		a.addIntel("Install failed: %s (%v)", msg.display, msg.err)
		a.views.Tools.Result = fmt.Sprintf("❌ %s install failed — see terminal output.", msg.display)
	}
	return a, refreshToolCacheCmd()
}
