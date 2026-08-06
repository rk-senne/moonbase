package tui

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

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
			switch {
			case pending == "reboot":
				return a.triggerReboot()
			case strings.HasPrefix(pending, "installall:"):
				return a.settingsInstallAll(strings.TrimPrefix(pending, "installall:"))
			default:
				if tool, ok := sm.toolByID(pending); ok {
					sm.Result = fmt.Sprintf("Installing %s…", tool.Display)
					return a.settingsInstall(tool)
				}
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
		sm.moveCursor(-1)
	case key.Matches(msg, a.keys.Down):
		sm.moveCursor(1)
	case key.Matches(msg, a.keys.Enter), key.Matches(msg, a.keys.Settings):
		row := sm.current()
		switch row.Kind {
		case rowReboot:
			sm.Confirm = "reboot"
			sm.Result = ""
		case rowInstallAll:
			a.requestInstallAll(row.OS)
		case rowTool:
			a.requestSettingsInstall(row.Tool)
		}
	}
	return a, nil
}

// requestInstallAll arms the y/n confirmation for a batch install of every
// missing tool in the given OS section, or records why there is nothing to do.
func (a *App) requestInstallAll(goos string) {
	sm := &a.views.Settings
	mgr, ok := tools.DetectManager()
	if !ok {
		sm.Result = "No supported package manager detected — install one first (e.g. Homebrew)."
		return
	}
	_, _, ok = tools.InstallAllPlan(tools.ToolsForOS(goos), mgr)
	if !ok {
		sm.Result = "Nothing to install — all recommended tools are already present."
		return
	}
	sm.Confirm = "installall:" + goos
	sm.Result = ""
}

// settingsInstallAll runs the batch package-manager install for the OS section
// via ExecProcess so the operator sees output (and any sudo prompt).
func (a App) settingsInstallAll(goos string) (tea.Model, tea.Cmd) {
	mgr, ok := tools.DetectManager()
	if !ok {
		a.views.Settings.Result = "No supported package manager detected."
		return a, nil
	}
	plan, skipped, ok := tools.InstallAllPlan(tools.ToolsForOS(goos), mgr)
	if !ok {
		a.views.Settings.Result = "Nothing to install — all recommended tools are already present."
		return a, nil
	}
	a.views.Settings.Result = fmt.Sprintf("Installing all missing tools via %s…", plan.Manager)
	label := "all recommended tools"
	if len(skipped) > 0 {
		label = fmt.Sprintf("all recommended tools (skipping %d manual/bootstrap)", len(skipped))
	}
	c := exec.Command(plan.Bin, plan.Args...)
	return a, tea.ExecProcess(c, func(err error) tea.Msg {
		return toolInstallDoneMsg{display: label, ok: err == nil, err: err}
	})
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
