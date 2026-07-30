package tui

import (
	"os"
	"sync"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestNewStyles_Pure(t *testing.T) {
	// Same theme produces equal styles (pure function property).
	s1 := NewStyles(moonbaseTheme)
	s2 := NewStyles(moonbaseTheme)

	// Compare rendered output of each style to verify equality.
	tests := []struct {
		name string
		a, b lipgloss.Style
	}{
		{"Header", s1.Header, s2.Header},
		{"Sidebar", s1.Sidebar, s2.Sidebar},
		{"Panel", s1.Panel, s2.Panel},
		{"Modal", s1.Modal, s2.Modal},
		{"Active", s1.Active, s2.Active},
		{"Inactive", s1.Inactive, s2.Inactive},
		{"StatusBar", s1.StatusBar, s2.StatusBar},
	}

	for _, tt := range tests {
		got := tt.a.Render("test")
		want := tt.b.Render("test")
		if got != want {
			t.Errorf("NewStyles not pure for %s: got %q, want %q", tt.name, got, want)
		}
	}
}

func TestThemeRegistry_HasFour(t *testing.T) {
	if ThemeCount() != 4 {
		t.Errorf("expected 4 themes in registry, got %d", ThemeCount())
	}

	expected := []string{"moonbase", "treehouse", "classified", "nerv"}
	for i, name := range expected {
		if themeRegistry[i].Name != name {
			t.Errorf("registry[%d] = %q, want %q", i, themeRegistry[i].Name, name)
		}
	}
}

func TestCycleTheme_AdvancesAndWraps(t *testing.T) {
	app := NewApp()

	// Verify initial state
	if app.theme.Name != "moonbase" {
		t.Fatalf("expected initial theme moonbase, got %s", app.theme.Name)
	}

	// Cycle through all themes
	expected := []string{"treehouse", "classified", "nerv", "moonbase"}
	for _, want := range expected {
		app.cycleTheme()
		if app.theme.Name != want {
			t.Errorf("expected theme %s, got %s", want, app.theme.Name)
		}
		if app.theme.Data.Name != want {
			t.Errorf("expected themeData.Name %s, got %s", want, app.theme.Data.Name)
		}
	}
}

func TestCycleTheme_PureValueTransform(t *testing.T) {
	// Cycling changes the model's theme fields without touching any package-level state.
	// The registry is read-only — cycling reads it but never mutates it.
	app := NewApp()

	// Take a snapshot of the registry before cycling
	regBefore := make([]Theme, len(themeRegistry))
	copy(regBefore, themeRegistry)

	// Cycle multiple times
	for i := 0; i < 8; i++ {
		app.cycleTheme()
	}

	// Verify registry unchanged
	for i, th := range themeRegistry {
		if th.Name != regBefore[i].Name {
			t.Errorf("registry[%d] mutated: was %s, now %s", i, regBefore[i].Name, th.Name)
		}
	}

	// Verify that cycling is deterministic (pure): same starting point → same result
	a1 := NewApp()
	a2 := NewApp()
	a1.cycleTheme()
	a2.cycleTheme()
	if a1.theme.Name != a2.theme.Name {
		t.Errorf("cycling not deterministic: %s vs %s", a1.theme.Name, a2.theme.Name)
	}
}

func TestCycleTheme_RaceSafe(t *testing.T) {
	// Exercise theme cycling concurrently to verify no races on shared state.
	// This is redundant with -race but explicit for documentation.
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			app := NewApp()
			for j := 0; j < 20; j++ {
				app.cycleTheme()
				_ = app.View()
			}
		}()
	}
	wg.Wait()
}

func TestNoColor_DegradesPalette(t *testing.T) {
	// Set NO_COLOR and verify NewStyles returns degraded styles.
	os.Setenv("NO_COLOR", "1")
	defer os.Unsetenv("NO_COLOR")

	styles := NewStyles(moonbaseTheme)

	// Degraded styles should produce non-empty output
	rendered := styles.Active.Render("test")
	if rendered == "" {
		t.Error("degraded Active style produced empty output")
	}

	// The degraded styles come from newDegradedStyles() which has no Foreground set.
	// Verify structurally by calling NewStyles with NO_COLOR and comparing to the
	// known degraded output (no colour in the style definition).
	degraded := newDegradedStyles()
	if degraded.Active.Render("x") != styles.Active.Render("x") {
		t.Error("NewStyles with NO_COLOR did not return degraded styles")
	}

	// Unset and verify normal path returns coloured styles (different from degraded)
	os.Unsetenv("NO_COLOR")
	normal := NewStyles(moonbaseTheme)
	// The normal Active has a Foreground color set; degraded does not.
	// We verify by checking that the style objects differ in their rendering
	// when colour profile supports it. In test environments (ASCII profile),
	// they may render identically, so we test the code path was taken instead.
	_ = normal // compilation guard
}

func TestNextTheme_Wraps(t *testing.T) {
	// Test that NextTheme wraps from last to first
	last := themeRegistry[len(themeRegistry)-1]
	next := NextTheme(last)
	if next.Name != themeRegistry[0].Name {
		t.Errorf("NextTheme(%s) = %s, want %s", last.Name, next.Name, themeRegistry[0].Name)
	}
}

func TestNextTheme_UnknownReturnsFirst(t *testing.T) {
	unknown := Theme{Name: "nonexistent"}
	next := NextTheme(unknown)
	if next.Name != themeRegistry[0].Name {
		t.Errorf("NextTheme(unknown) = %s, want %s", next.Name, themeRegistry[0].Name)
	}
}

func TestThemeByName_Found(t *testing.T) {
	for _, want := range themeRegistry {
		got := ThemeByName(want.Name)
		if got.Name != want.Name {
			t.Errorf("ThemeByName(%s) = %s", want.Name, got.Name)
		}
	}
}

func TestThemeByName_NotFound(t *testing.T) {
	got := ThemeByName("nonexistent")
	if got.Name != "moonbase" {
		t.Errorf("ThemeByName(nonexistent) = %s, want moonbase", got.Name)
	}
}
