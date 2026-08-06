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
	if len(result.views.Settings.Catalog) == 0 {
		t.Error("expected the dev catalog to be populated in Settings")
	}
	if result.views.Browser.Active {
		t.Error("expected the file browser released for the settings view")
	}
}

func TestSettingsKeys_Navigation(t *testing.T) {
	app := NewApp()
	app.boot.Ready = true
	app.enterSettingsView()

	// Cursor starts on the reboot action (0); down moves into the catalog.
	model, _ := app.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	if model.(App).views.Settings.Cursor != 1 {
		t.Errorf("expected cursor 1 after down, got %d", model.(App).views.Settings.Cursor)
	}
	model, _ = model.(App).Update(tea.KeyPressMsg{Code: tea.KeyUp})
	if model.(App).views.Settings.Cursor != 0 {
		t.Errorf("expected cursor back to 0 (reboot action) after up, got %d", model.(App).views.Settings.Cursor)
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
	for _, want := range []string{"SETTINGS", "Reboot", "Homebrew", "Python", "Node", "MOONBASE", "DEV ENVIRONMENT"} {
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
