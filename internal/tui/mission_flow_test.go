package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
)

// TestEndToEnd_MissionSubmitFlow drives the real Update path a user takes:
// press 'm' to open the briefing (even with the file browser active), type an
// objective, then press Enter to deploy. This guards the reported "submitting a
// mission doesn't work after pressing enter" regression end-to-end.
func TestEndToEnd_MissionSubmitFlow(t *testing.T) {
	app := NewApp()
	app.boot.Ready = true
	app.view = ViewDashboard
	app.registry = newTestRegistry()
	// Worst case: the file browser owns the panel. 'm' must still open briefing.
	app.views.Browser.Active = true

	// 1. 'm' opens the mission briefing and releases the browser.
	m, _ := app.Update(tea.KeyPressMsg{Code: 'm', Text: "m"})
	st := m.(App)
	if st.view != ViewMission {
		t.Fatalf("expected ViewMission after 'm', got %d", st.view)
	}
	if st.views.Browser.Active {
		t.Fatal("expected file browser released in mission briefing")
	}

	// 2. Typing is captured by the focused input (not stolen by global keys).
	for _, r := range "add pagination" {
		m, _ = st.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
		st = m.(App)
	}
	if got := st.views.Mission.Input.Value(); got != "add pagination" {
		t.Fatalf("expected typed objective captured, got %q", got)
	}

	// 3. Enter submits → pipeline view with the mission state populated.
	m, _ = st.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	st = m.(App)
	if st.view != ViewPipeline {
		t.Fatalf("expected ViewPipeline after enter-submit, got %d", st.view)
	}
	if st.views.Pipeline.State == nil {
		t.Fatal("expected pipeline state to be created on submit")
	}
	if st.views.Pipeline.State.Task != "add pagination" {
		t.Errorf("expected task 'add pagination', got %q", st.views.Pipeline.State.Task)
	}
}

// Tab must be reachable on the dashboard (was swallowed when the file browser
// was active by default). It cycles the focused panel.
func TestDashboardKeys_TabCyclesFocus(t *testing.T) {
	app := NewApp()
	app.boot.Ready = true
	app.view = ViewDashboard
	app.views.Browser.Active = false
	app.views.Terminal.Active = false

	before := app.chrome.Focus
	m, _ := app.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	after := m.(App).chrome.Focus
	if after == before {
		t.Errorf("expected Tab to change panel focus, stayed at %d", before)
	}
}

// Backtick opens the (now opt-in) file browser from the dashboard.
func TestDashboardKeys_BacktickOpensBrowser(t *testing.T) {
	app := NewApp()
	app.boot.Ready = true
	app.view = ViewDashboard
	app.views.Browser.Active = false
	app.views.Terminal.Active = false

	m, _ := app.Update(tea.KeyPressMsg{Code: '`', Text: "`"})
	if !m.(App).views.Browser.Active {
		t.Error("expected backtick to open the file browser from the dashboard")
	}
}
