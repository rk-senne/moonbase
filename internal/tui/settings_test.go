package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestGlobalKeys_OpensSettings(t *testing.T) {
	app := NewApp()
	app.boot.Ready = true
	app.view = ViewDashboard
	app.views.Browser.Active = true // even with the browser active, 'S' must work

	model, _ := app.Update(tea.KeyPressMsg{Code: 'S', Text: "S"})
	result := model.(App)
	if result.view != ViewSettings {
		t.Fatalf("expected ViewSettings after 'S', got %d", result.view)
	}
	if len(result.views.Settings.Rows) == 0 {
		t.Error("expected the dev catalog rows to be populated in Settings")
	}
	if result.views.Browser.Active {
		t.Error("expected the file browser released for the settings view")
	}
}

func TestSettingsKeys_Navigation(t *testing.T) {
	app := NewApp()
	app.boot.Ready = true
	app.enterSettingsView()
	sm := app.views.Settings

	// Cursor starts on the reboot action (0); down moves to the next SELECTABLE
	// row — which is host-dependent (the current OS's section), so assert by
	// selectability rather than a fixed index (CI runs on Linux, dev on macOS).
	model, _ := app.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	down := model.(App).views.Settings
	if down.Cursor == settingsActionReboot {
		t.Fatal("expected cursor to advance off the reboot row on down")
	}
	if !down.Rows[down.Cursor].selectable(sm.CurrentOS) {
		t.Errorf("cursor landed on a non-selectable (grayed) row %d", down.Cursor)
	}
	// The landing row must belong to the current OS section (not the grayed one).
	if down.Rows[down.Cursor].OS != sm.CurrentOS {
		t.Errorf("expected navigation to stay in the %s section, got OS %q", sm.CurrentOS, down.Rows[down.Cursor].OS)
	}
	// Up returns to the reboot action.
	model, _ = model.(App).Update(tea.KeyPressMsg{Code: tea.KeyUp})
	if model.(App).views.Settings.Cursor != settingsActionReboot {
		t.Errorf("expected cursor back to reboot action after up, got %d", model.(App).views.Settings.Cursor)
	}
}

// The non-current OS section is present but never selectable (grayed out).
func TestSettingsModel_OtherOSGrayed(t *testing.T) {
	m := newSettingsModelFor("linux")
	var sawMac, sawLinuxSelectable bool
	for _, r := range m.Rows {
		if r.OS == "darwin" {
			sawMac = true
			if r.selectable("linux") {
				t.Error("macOS rows must not be selectable when running on Linux")
			}
		}
		if r.OS == "linux" && r.selectable("linux") {
			sawLinuxSelectable = true
		}
	}
	if !sawMac {
		t.Error("expected a macOS section to exist even on Linux")
	}
	if !sawLinuxSelectable {
		t.Error("expected the Linux section to be selectable on Linux")
	}
}

// Enter on an install-all row (current OS) arms the batch confirmation.
func TestSettingsKeys_InstallAllArm(t *testing.T) {
	app := NewApp()
	app.boot.Ready = true
	app.enterSettingsView()
	sm := &app.views.Settings

	// Move the cursor to the current OS's install-all row.
	idx := -1
	for i, r := range sm.Rows {
		if r.Kind == rowInstallAll && r.OS == sm.CurrentOS {
			idx = i
			break
		}
	}
	if idx < 0 {
		t.Fatal("no install-all row for the current OS")
	}
	sm.Cursor = idx

	model, _ := app.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	got := model.(App).views.Settings.Confirm
	// Either armed for a batch install, or a benign "nothing to install" result
	// when every recommended tool already happens to be present in the env.
	if got != "installall:"+sm.CurrentOS && got != "" {
		t.Fatalf("unexpected confirm state after install-all enter: %q", got)
	}
}

func TestSettingsKeys_Back(t *testing.T) {
	app := NewApp()
	app.boot.Ready = true
	app.enterSettingsView()
	model, _ := app.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if model.(App).view != ViewDashboard {
		t.Errorf("expected ViewDashboard after esc, got %d", model.(App).view)
	}
}

func TestSettingsKeys_RebootConfirmArmAndCancel(t *testing.T) {
	app := NewApp()
	app.boot.Ready = true
	app.enterSettingsView()
	app.views.Settings.Cursor = settingsActionReboot

	// Enter on the reboot action arms the confirmation.
	model, _ := app.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	result := model.(App)
	if result.views.Settings.Confirm != "reboot" {
		t.Fatalf("expected reboot confirmation armed, got %q", result.views.Settings.Confirm)
	}

	// 'n' cancels without rebooting.
	model, _ = result.Update(tea.KeyPressMsg{Code: 'n', Text: "n"})
	result = model.(App)
	if result.views.Settings.Confirm != "" {
		t.Error("expected confirmation cleared after 'n'")
	}
	if _, want := result.RebootInfo(); want {
		t.Error("did not expect a reboot to be requested after cancel")
	}
}

func TestHandleRebootDone_SuccessFlagsRebootAndQuits(t *testing.T) {
	app := NewApp()
	app.boot.Ready = true
	app.enterSettingsView()
	app.views.Settings.RebootBin = "/home/op/.local/bin/moonbase"

	model, cmd := app.handleRebootDone(rebootDoneMsg{err: nil})
	result := model.(App)

	bin, want := result.RebootInfo()
	if !want {
		t.Fatal("expected RebootInfo to report a requested reboot on success")
	}
	if bin != "/home/op/.local/bin/moonbase" {
		t.Errorf("expected reboot bin preserved, got %q", bin)
	}
	if cmd == nil {
		t.Error("expected a quit command to be returned on reboot success")
	}
}

func TestHandleRebootDone_FailureStays(t *testing.T) {
	app := NewApp()
	app.boot.Ready = true
	app.enterSettingsView()

	model, _ := app.handleRebootDone(rebootDoneMsg{err: errTest})
	result := model.(App)
	if _, want := result.RebootInfo(); want {
		t.Error("did not expect a reboot request after a failed reinstall")
	}
	if !strings.Contains(result.views.Settings.Result, "failed") {
		t.Errorf("expected failure surfaced in result, got %q", result.views.Settings.Result)
	}
}

func TestRenderSettings_Content(t *testing.T) {
	app := NewApp()
	app.boot.Ready = true
	app.width = 100
	app.height = 32
	app.enterSettingsView()

	out := app.renderSettings()
	for _, want := range []string{"SETTINGS", "Current OS", "Reboot", "MOONBASE", "DEV ENVIRONMENT", "macOS", "Linux", "Install all", "Homebrew", "Python"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected settings render to contain %q", want)
		}
	}
}

func TestSetVersion(t *testing.T) {
	orig := buildVersion
	defer SetVersion(orig)
	SetVersion("1.9.9")
	if buildVersion != "1.9.9" {
		t.Errorf("expected buildVersion updated, got %q", buildVersion)
	}
	SetVersion("") // empty must not clobber
	if buildVersion != "1.9.9" {
		t.Errorf("empty SetVersion should be a no-op, got %q", buildVersion)
	}
}

var errTest = &testErr{"boom"}

type testErr struct{ s string }

func (e *testErr) Error() string { return e.s }
