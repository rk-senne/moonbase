package tui

import (
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
)

// handleGlobalKeys intercepts keys that must work from ANY stage of the TUI,
// regardless of which view or sub-mode currently owns the panel. It runs after
// the text-entry guards (search / embedded terminal) and before the file
// browser + view-specific handlers.
//
// It deliberately does NOT fire while a text input is focused (mission briefing
// or COMMS message input) so those views can receive the literal characters.
// The returned bool reports whether the key was handled.
func (a App) handleGlobalKeys(msg tea.KeyPressMsg) (tea.Model, tea.Cmd, bool) {
	// Never steal keys while the operator is typing into a text field.
	if a.view == ViewMission || a.view == ViewComms {
		return a, nil, false
	}

	switch {
	case key.Matches(msg, a.keys.NewMission):
		cmd := a.enterMissionView()
		return a, cmd, true
	case key.Matches(msg, a.keys.Tools):
		// Inside the Tools view, 'i' is "install selected" — let that view own it.
		if a.view == ViewTools {
			return a, nil, false
		}
		a.enterToolsView()
		return a, nil, true
	case key.Matches(msg, a.keys.Settings):
		// Inside Settings, 'S' selects the current row — let that view own it.
		if a.view == ViewSettings {
			return a, nil, false
		}
		a.enterSettingsView()
		return a, nil, true
	}

	return a, nil, false
}

// enterMissionView switches to the mission briefing view with a fresh, focused
// input. It also releases the file browser so the dashboard's key capture can
// never swallow the briefing keystrokes. Safe to call from any view.
func (a *App) enterMissionView() tea.Cmd {
	a.views.Browser.Active = false
	a.view = ViewMission
	a.views.Mission.Input.Reset()
	a.views.Mission.Input.Focus()
	return textinput.Blink
}
