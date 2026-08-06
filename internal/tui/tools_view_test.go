package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestGlobalKeys_OpensTools(t *testing.T) {
	app := NewApp()
	app.boot.Ready = true
	app.view = ViewDashboard
	app.views.Browser.Active = true // even with the browser active, 'i' must work

	model, _ := app.Update(tea.KeyPressMsg{Code: 'i', Text: "i"})
	result := model.(App)
	if result.view != ViewTools {
		t.Fatalf("expected ViewTools after 'i', got %d", result.view)
	}
	if len(result.views.Tools.Catalog) == 0 {
		t.Error("expected the tools catalog to be populated")
	}
	if result.views.Browser.Active {
		t.Error("expected the file browser to be released for the tools view")
	}
}

func TestToolsKeys_Navigation(t *testing.T) {
	app := NewApp()
	app.boot.Ready = true
	app.enterToolsView()

	model, _ := app.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	result := model.(App)
	if result.views.Tools.Cursor != 1 {
		t.Errorf("expected cursor 1 after down, got %d", result.views.Tools.Cursor)
	}
	model, _ = result.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	result = model.(App)
	if result.views.Tools.Cursor != 0 {
		t.Errorf("expected cursor 0 after up, got %d", result.views.Tools.Cursor)
	}
}

func TestToolsKeys_Back(t *testing.T) {
	app := NewApp()
	app.boot.Ready = true
	app.enterToolsView()

	model, _ := app.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if model.(App).view != ViewDashboard {
		t.Errorf("expected ViewDashboard after esc, got %d", model.(App).view)
	}
}

func TestToolsKeys_ConfirmCancel(t *testing.T) {
	app := NewApp()
	app.boot.Ready = true
	app.enterToolsView()
	app.views.Tools.Confirm = "git"

	model, _ := app.Update(tea.KeyPressMsg{Code: 'n', Text: "n"})
	result := model.(App)
	if result.views.Tools.Confirm != "" {
		t.Error("expected confirmation cleared after 'n'")
	}
	if !strings.Contains(strings.ToLower(result.views.Tools.Result), "cancel") {
		t.Errorf("expected a cancellation message, got %q", result.views.Tools.Result)
	}
}

func TestToolsKeys_ConfirmProceedClearsAndRuns(t *testing.T) {
	app := NewApp()
	app.boot.Ready = true
	app.enterToolsView()
	app.views.Tools.Confirm = "git"

	model, cmd := app.Update(tea.KeyPressMsg{Code: 'y', Text: "y"})
	result := model.(App)
	if result.views.Tools.Confirm != "" {
		t.Error("expected confirmation cleared after 'y'")
	}
	if cmd == nil {
		t.Error("expected an install command to be dispatched after 'y'")
	}
}

// Pressing 'i' inside the Tools view must install (be handled by the tools view),
// NOT be stolen by the global handler and re-open the view.
func TestToolsKeys_InstallKeyNotStolen(t *testing.T) {
	app := NewApp()
	app.boot.Ready = true
	app.enterToolsView()

	model, _ := app.Update(tea.KeyPressMsg{Code: 'i', Text: "i"})
	result := model.(App)
	if result.view != ViewTools {
		t.Fatalf("expected to remain in ViewTools, got %d", result.view)
	}
	// requestInstall always sets either a confirmation or a result message.
	if result.views.Tools.Confirm == "" && result.views.Tools.Result == "" {
		t.Error("expected the tools handler to process 'i' (confirm or result set)")
	}
}

func TestHandleToolInstallDone(t *testing.T) {
	app := NewApp()
	app.boot.Ready = true
	app.enterToolsView()

	// Success path.
	model, cmd := app.handleToolInstallDone(toolInstallDoneMsg{display: "jq", ok: true})
	result := model.(App)
	if !strings.Contains(result.views.Tools.Result, "installed") {
		t.Errorf("expected success result, got %q", result.views.Tools.Result)
	}
	if cmd == nil {
		t.Error("expected a tool-cache refresh command after install")
	}

	// Manual/no-manager note path.
	model2, _ := app.handleToolInstallDone(toolInstallDoneMsg{display: "oh-my-posh", note: "run the script"})
	if !strings.Contains(model2.(App).views.Tools.Result, "run the script") {
		t.Errorf("expected the note surfaced, got %q", model2.(App).views.Tools.Result)
	}
}

func TestRenderTools_ContainsCatalog(t *testing.T) {
	app := NewApp()
	app.boot.Ready = true
	app.width = 100
	app.height = 30
	app.enterToolsView()

	out := app.renderTools()
	for _, want := range []string{"TOOLS ARSENAL", "CRITICAL", "COOL", "oh-my-posh", "lazygit"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected tools render to contain %q", want)
		}
	}
}
