package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/key"
)

func TestHelpView_ContainsEveryBinding(t *testing.T) {
	app := NewApp()
	app.view = ViewHelp
	app.ready = true
	app.width = 120
	app.height = 50

	rendered := app.renderHelp()

	// Every binding in FullHelp() should have its help description present
	groups := app.keys.FullHelp()
	for gi, group := range groups {
		for bi, b := range group {
			h := b.Help()
			if h.Desc == "" {
				continue
			}
			if !strings.Contains(rendered, h.Desc) {
				t.Errorf("FullHelp group[%d][%d]: help description %q not found in rendered help view", gi, bi, h.Desc)
			}
			if !strings.Contains(rendered, h.Key) {
				t.Errorf("FullHelp group[%d][%d]: help key %q not found in rendered help view", gi, bi, h.Key)
			}
		}
	}

	// Verify brand prose is preserved
	if !strings.Contains(rendered, "Operations Manual") {
		t.Error("help view missing 'Operations Manual' header")
	}
	if !strings.Contains(rendered, "THE KND WAY") {
		t.Error("help view missing 'THE KND WAY' footer")
	}
	if !strings.Contains(rendered, "We fight for kids everywhere") {
		t.Error("help view missing brand tagline")
	}
}

func TestFooterShowsOnlyActiveViewKeys(t *testing.T) {
	tests := []struct {
		name       string
		view       View
		searching  bool
		termActive bool
		browsing   bool
		mustHave   []key.Binding
		mustNotHave []key.Binding
	}{
		{
			name: "dashboard normal",
			view: ViewDashboard,
			mustHave: []key.Binding{
				DefaultKeyMap().Help,
				DefaultKeyMap().Enter,
				DefaultKeyMap().Quit,
			},
			mustNotHave: []key.Binding{
				DefaultKeyMap().SendMessage,
				DefaultKeyMap().AttachFile,
				DefaultKeyMap().NextPhase,
			},
		},
		{
			name:       "dashboard searching",
			view:       ViewDashboard,
			searching:  true,
			mustHave:   []key.Binding{DefaultKeyMap().SearchConfirm, DefaultKeyMap().SearchCancel},
			mustNotHave: []key.Binding{DefaultKeyMap().NewMission, DefaultKeyMap().Quit, DefaultKeyMap().Help},
		},
		{
			name:       "dashboard terminal",
			view:       ViewDashboard,
			termActive: true,
			mustHave:   []key.Binding{DefaultKeyMap().TerminalSubmit, DefaultKeyMap().TerminalToBrowser, DefaultKeyMap().TerminalEsc},
			mustNotHave: []key.Binding{DefaultKeyMap().NewMission, DefaultKeyMap().SendMessage},
		},
		{
			name:     "dashboard browsing",
			view:     ViewDashboard,
			browsing: true,
			mustHave: []key.Binding{DefaultKeyMap().BrowserUp, DefaultKeyMap().BrowserDown, DefaultKeyMap().BrowserEnter},
			mustNotHave: []key.Binding{DefaultKeyMap().NewMission, DefaultKeyMap().SendMessage, DefaultKeyMap().NextPhase},
		},
		{
			name: "pipeline view",
			view: ViewPipeline,
			mustHave: []key.Binding{
				DefaultKeyMap().NextPhase,
				DefaultKeyMap().RetryPhase,
				DefaultKeyMap().SkipPhase,
				DefaultKeyMap().Back,
			},
			mustNotHave: []key.Binding{
				DefaultKeyMap().SendMessage,
				DefaultKeyMap().LaunchLazygit,
				DefaultKeyMap().Search,
			},
		},
		{
			name:    "comms view",
			view:    ViewComms,
			mustHave: []key.Binding{DefaultKeyMap().SendMessage, DefaultKeyMap().AttachFile, DefaultKeyMap().Back},
			mustNotHave: []key.Binding{DefaultKeyMap().NewMission, DefaultKeyMap().NextPhase, DefaultKeyMap().LaunchLazygit},
		},
		{
			name:    "help view",
			view:    ViewHelp,
			mustHave: []key.Binding{DefaultKeyMap().Back},
			mustNotHave: []key.Binding{DefaultKeyMap().NewMission, DefaultKeyMap().SendMessage, DefaultKeyMap().NextPhase},
		},
		{
			name:    "dossier view",
			view:    ViewDossier,
			mustHave: []key.Binding{DefaultKeyMap().Enter, DefaultKeyMap().CopyPrompt, DefaultKeyMap().Back},
			mustNotHave: []key.Binding{DefaultKeyMap().SendMessage, DefaultKeyMap().NextPhase, DefaultKeyMap().LaunchLazygit},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			km := DefaultKeyMap()
			bindings := km.keysFor(tc.view, tc.searching, tc.termActive, tc.browsing)

			// Build a set of help descriptions that are in this view's keysFor
			activeDescs := make(map[string]bool)
			for _, b := range bindings {
				activeDescs[b.Help().Desc] = true
			}

			// Verify mustHave bindings are present
			for _, b := range tc.mustHave {
				if !activeDescs[b.Help().Desc] {
					t.Errorf("expected binding %q (key=%s) to be present in %s footer", b.Help().Desc, b.Help().Key, tc.name)
				}
			}

			// Verify mustNotHave bindings are absent
			for _, b := range tc.mustNotHave {
				if activeDescs[b.Help().Desc] {
					t.Errorf("binding %q (key=%s) should NOT be in %s footer", b.Help().Desc, b.Help().Key, tc.name)
				}
			}

			// Also verify the rendered footer does not contain keys from mustNotHave
			app := NewApp()
			app.view = tc.view
			app.search.Active = tc.searching
			app.terminal.Active = tc.termActive
			app.browsing = tc.browsing
			app.width = 120
			app.height = 40
			app.ready = true

			footer := app.renderContextualStatusBar()
			for _, b := range tc.mustNotHave {
				desc := b.Help().Desc
				// Only check non-generic descriptions (avoid matching "up"/"down" substrings)
				if len(desc) > 4 && strings.Contains(footer, desc) {
					t.Errorf("rendered footer for %s contains %q which should not be visible", tc.name, desc)
				}
			}
		})
	}
}

func TestKeysFor_ReturnsNonEmpty(t *testing.T) {
	km := DefaultKeyMap()
	views := []struct {
		name string
		view View
	}{
		{"Dashboard", ViewDashboard},
		{"Dossier", ViewDossier},
		{"Pipeline", ViewPipeline},
		{"Help", ViewHelp},
		{"Mission", ViewMission},
		{"Comms", ViewComms},
		{"History", ViewHistory},
		{"Docs", ViewDocs},
		{"Projects", ViewProjects},
		{"Protocol", ViewProtocol},
	}

	for _, v := range views {
		t.Run(v.name, func(t *testing.T) {
			bindings := km.keysFor(v.view, false, false, false)
			if len(bindings) == 0 {
				t.Errorf("keysFor(%s) returned empty slice", v.name)
			}
		})
	}
}

func TestContextualStatusBar_RendersWithoutPanic(t *testing.T) {
	views := []struct {
		name       string
		view       View
		searching  bool
		termActive bool
		browsing   bool
	}{
		{"Dashboard", ViewDashboard, false, false, false},
		{"Dashboard searching", ViewDashboard, true, false, false},
		{"Dashboard terminal", ViewDashboard, false, true, false},
		{"Dashboard browsing", ViewDashboard, false, false, true},
		{"Dossier", ViewDossier, false, false, false},
		{"Pipeline", ViewPipeline, false, false, false},
		{"Help", ViewHelp, false, false, false},
		{"Mission", ViewMission, false, false, false},
		{"Comms", ViewComms, false, false, false},
		{"History", ViewHistory, false, false, false},
		{"Docs", ViewDocs, false, false, false},
		{"Projects", ViewProjects, false, false, false},
		{"Protocol", ViewProtocol, false, false, false},
	}

	for _, v := range views {
		t.Run(v.name, func(t *testing.T) {
			app := NewApp()
			app.view = v.view
			app.search.Active = v.searching
			app.terminal.Active = v.termActive
			app.browsing = v.browsing
			app.width = 100
			app.height = 40
			app.ready = true

			result := app.renderContextualStatusBar()
			if result == "" {
				t.Errorf("renderContextualStatusBar returned empty for %s", v.name)
			}
		})
	}
}
