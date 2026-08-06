package tui

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"github.com/rk-senne/moonbase/internal/config"
	"github.com/rk-senne/moonbase/internal/reboot"
	"github.com/rk-senne/moonbase/internal/tools"
	"github.com/rk-senne/moonbase/internal/updater"
)

// buildVersion is the moonbase version, injected from main via SetVersion. It
// drives the reboot strategy (dev builds rebuild from source; releases self-update).
var buildVersion = "dev"

// SetVersion records the running moonbase version for the TUI (called from main).
func SetVersion(v string) {
	if v != "" {
		buildVersion = v
	}
}

// RebootInfo reports whether the operator requested a reboot and the binary path
// to re-exec. main calls this after the tea program exits to relaunch moonbase.
func (a App) RebootInfo() (string, bool) {
	return a.views.Settings.RebootBin, a.views.Settings.RebootRequested
}

// rebootDoneMsg is emitted after a reinstall attempt completes.
type rebootDoneMsg struct{ err error }

// enterSettingsView opens the Settings view with a fresh model.
func (a *App) enterSettingsView() {
	a.views.Browser.Active = false
	a.views.Settings = NewSettingsModel()
	a.view = ViewSettings
}

// handleSettingsKeys handles keys for the Settings view, including the reboot
// and install confirmation sub-modes. Nothing runs without an explicit 'y'.
func (a App) handleSettingsKeys(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	sm := &a.views.Settings

	// Confirmation sub-mode.
	if sm.Confirm != "" {
		switch msg.String() {
		case "y", "Y":
			pending := sm.Confirm
			sm.Confirm = ""
			if pending == "reboot" {
				return a.triggerReboot()
			}
			if tool, ok := sm.toolByID(pending); ok {
				sm.Result = fmt.Sprintf("Installing %s…", tool.Display)
				return a.settingsInstall(tool)
			}
			return a, nil
		default:
			sm.Confirm = ""
			sm.Result = "Cancelled."
			return a, nil
		}
	}

	switch {
	case key.Matches(msg, a.keys.Back):
		a.view = ViewDashboard
	case key.Matches(msg, a.keys.Up):
		if sm.Cursor > 0 {
			sm.Cursor--
		}
	case key.Matches(msg, a.keys.Down):
		if sm.Cursor < sm.rowCount()-1 {
			sm.Cursor++
		}
	case key.Matches(msg, a.keys.Enter), key.Matches(msg, a.keys.Settings):
		if sm.Cursor == settingsActionReboot {
			sm.Confirm = "reboot"
			sm.Result = ""
			return a, nil
		}
		if tool, ok := sm.selectedTool(); ok {
			a.requestSettingsInstall(tool)
		}
	}
	return a, nil
}

// requestSettingsInstall validates a tool can be installed and arms the y/n
// confirmation, or records guidance. Homebrew (bootstrap) is always confirmable.
func (a *App) requestSettingsInstall(tool tools.Tool) {
	sm := &a.views.Settings

	if tools.IsInstalled(tool.ID) {
		sm.Result = fmt.Sprintf("%s is already installed.", tool.Display)
		return
	}
	if tool.Bootstrap { // Homebrew — official installer, confirm required.
		sm.Confirm = tool.ID
		sm.Result = ""
		return
	}
	if tool.MacOnly && runtime.GOOS != "darwin" {
		sm.Result = fmt.Sprintf("%s is macOS-only — %s", tool.Display, tool.Manual)
		return
	}
	mgr, ok := tools.DetectManager()
	if !ok {
		sm.Result = "No supported package manager detected — " + tool.Manual
		return
	}
	if _, ok, reason := tools.BuildInstall(tool, mgr); !ok {
		sm.Result = "Manual install required — " + reason
		return
	}
	sm.Confirm = tool.ID
	sm.Result = ""
}

// settingsInstall runs the install for a dev-catalog tool via ExecProcess so the
// operator sees output. Homebrew uses its official bootstrap installer.
func (a App) settingsInstall(tool tools.Tool) (tea.Model, tea.Cmd) {
	if tool.Bootstrap {
		plan := tools.HomebrewInstallPlan()
		c := exec.Command(plan.Bin, plan.Args...)
		return a, tea.ExecProcess(c, func(err error) tea.Msg {
			return toolInstallDoneMsg{display: tool.Display, ok: err == nil, err: err}
		})
	}
	return a, a.installTool(tool)
}

// triggerReboot selects the reinstall strategy and runs it, then (on success)
// flags a reboot so main re-execs the fresh binary.
func (a App) triggerReboot() (tea.Model, tea.Cmd) {
	exe, err := os.Executable()
	if err != nil || exe == "" {
		a.views.Settings.Result = "❌ Cannot locate the moonbase binary to reinstall."
		return a, nil
	}
	cfg := config.Load()
	plan := reboot.SelectPlan(buildVersion, cfg.SourceDir, exe)

	switch plan.Strategy {
	case reboot.StrategySource:
		a.views.Settings.RebootBin = plan.TargetBin
		a.views.Settings.Result = "⟳ Rebooting via " + plan.Reason
		script := reboot.ReinstallScript(plan.SourceDir, plan.TargetBin)
		c := exec.Command("bash", "-c", script)
		return a, tea.ExecProcess(c, func(err error) tea.Msg { return rebootDoneMsg{err: err} })
	case reboot.StrategyRelease:
		a.views.Settings.RebootBin = plan.TargetBin
		a.views.Settings.Result = "⟳ Downloading the latest release…"
		return a, func() tea.Msg {
			_, uerr := updater.Update(buildVersion)
			return rebootDoneMsg{err: uerr}
		}
	default:
		a.views.Settings.Result = "⚠️ " + plan.Reason
		return a, nil
	}
}

// handleRebootDone finalizes a reinstall: on success it flags a reboot and quits
// (main then re-execs); on failure it surfaces the error and stays put.
func (a App) handleRebootDone(msg rebootDoneMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		a.views.Settings.Result = "❌ Reboot failed: " + msg.err.Error()
		return a, nil
	}
	a.views.Settings.RebootRequested = true
	a.addIntel("Rebooting moonbase…")
	return a, tea.Quit
}
