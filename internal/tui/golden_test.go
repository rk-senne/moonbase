package tui

import (
	"flag"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/f5508037/moonbase/internal/pipeline"
	"github.com/muesli/termenv"
)

// updateGolden controls whether golden files are written (regenerated) or compared.
// Run with: go test ./internal/tui/ -run TestGolden -update-golden
var updateGolden = flag.Bool("update-golden", false, "regenerate golden files instead of comparing")

// goldenDir returns the absolute path to the testdata directory, resolved from the
// source file location so tests are immune to os.Chdir calls in other tests.
func goldenDir() string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "testdata")
}

// goldenAssert renders the given view string and compares it to the golden file
// at testdata/<name>.golden. If -update-golden is set, the golden file is written
// instead. The colour profile is pinned to TrueColor for cross-machine stability.
func goldenAssert(t *testing.T, name string, got string) {
	t.Helper()

	dir := goldenDir()
	goldenPath := filepath.Join(dir, name+".golden")

	if *updateGolden {
		err := os.MkdirAll(dir, 0700)
		if err != nil {
			t.Fatalf("creating testdata dir: %v", err)
		}
		err = os.WriteFile(goldenPath, []byte(got), 0600)
		if err != nil {
			t.Fatalf("writing golden file %s: %v", goldenPath, err)
		}
		t.Logf("updated golden file: %s (%d bytes)", goldenPath, len(got))
		return
	}

	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("reading golden file %s (run with -update-golden to generate): %v", goldenPath, err)
	}

	if got != string(want) {
		// Find first differing line for a useful error message
		gotLines := strings.Split(got, "\n")
		wantLines := strings.Split(string(want), "\n")
		for i := 0; i < len(gotLines) || i < len(wantLines); i++ {
			var gl, wl string
			if i < len(gotLines) {
				gl = gotLines[i]
			}
			if i < len(wantLines) {
				wl = wantLines[i]
			}
			if gl != wl {
				t.Errorf("golden mismatch in %s at line %d:\n  got:  %q\n  want: %q", name, i+1, gl, wl)
				break
			}
		}
		t.Errorf("full output length: got=%d want=%d", len(got), len(want))
	}
}

// newGoldenApp constructs a deterministic App suitable for golden testing.
// All dynamic elements (clock, uptime, spinner, animation, blink, tool cache) are
// frozen to fixed values so output is reproducible across machines and CI.
//
// Determinism strategy:
//   - clock: fixed string "12:00:00"
//   - uptime: startTime set immediately before View() call (yields "00:00:00")
//   - spinner: no running pipeline phases (avoids frame-dependent output)
//   - blink: fixed to false
//   - toolCache: all tools unavailable (avoids host-dependent exec.LookPath)
//   - file browser: disabled (browsing=false) to avoid real FS content
//   - file watcher: nil (testMode skips creation)
//   - git state: fixed clean main branch
//   - animation: zero-value AnimState (radar frame 0)
//   - cwd: fixed string so terminal panel header is stable
func newGoldenApp(t *testing.T) App {
	t.Helper()

	reg := newTestRegistry()

	// Deterministic tool cache: all tools unavailable (avoids exec.LookPath variance)
	toolCache := map[string]bool{
		"lazygit": false,
		"docker":  false,
		"btop":    false,
		"nvim":    false,
		"cmux":    false,
		"tmux":    false,
		"fish":    false,
	}

	app := App{
		keys:          DefaultKeyMap(),
		view:          ViewDashboard,
		registry:      reg,
		dashboard:     DashboardModel{Cursor: 0, Selected: 0},
		width:         100,
		height:        30,
		ready:         true,
		theme:         "moonbase",
		themeData:     moonbaseTheme,
		styles:        NewStyles(moonbaseTheme),
		clock:         "12:00:00",
		startTime:     time.Now(), // uptime() = time.Since(startTime) ≈ 0 → "00:00:00"
		focus:         FocusSidebar,
		blink:         false,
		intel:         []IntelEntry{},
		missions:      []MissionEntry{{Name: "init scaffold", Status: "✅"}, {Name: "tui views", Status: "✅"}, {Name: "pipeline+deploy", Status: "✅"}},
		toolCache:     toolCache,
		toolCacheTime: time.Now(),
		fileBrowser:   nil,      // avoid real FS reads
		browsing:      false,    // show terminal panel (deterministic with empty state)
		gitBranch:     "main",
		gitClean:      true,
		gitDiffLines:  0,
		dockerCount:   0,
		terminal:      TerminalModel{Cwd: "/home/operative/moonbase"}, // fixed path for stable header
	}

	return app
}

// TestGolden_Dashboard captures the dashboard view at a fixed 100x30 state.
func TestGolden_Dashboard(t *testing.T) {
	// Pin colour profile to TrueColor for deterministic ANSI output.
	lipgloss.SetColorProfile(termenv.TrueColor)
	lipgloss.SetHasDarkBackground(true)

	app := newGoldenApp(t)
	app.view = ViewDashboard

	got := app.View()
	goldenAssert(t, "dashboard", got)
}

// TestGolden_Pipeline captures the pipeline view with a deterministic phase state.
// All phases are set to fixed statuses (no StatusRunning which would include spinner).
func TestGolden_Pipeline(t *testing.T) {
	lipgloss.SetColorProfile(termenv.TrueColor)
	lipgloss.SetHasDarkBackground(true)

	app := newGoldenApp(t)
	app.view = ViewPipeline

	// Build a deterministic pipeline state: phases 1-2 complete, 3 pending, rest pending.
	// No running phase avoids spinner frame dependency.
	ps := &pipeline.Pipeline{
		Task:       "add pagination to /users API",
		Active:     true,
		MaxRework:  2,
		TraceID:    "abc123def456",
		Context:    pipeline.NewPipelineContext("add pagination to /users API"),
		Phases: []pipeline.Phase{
			{Number: 1, Name: "Analysis", Operative: "Numbuh 1", AgentName: "numbuh-1", Status: pipeline.StatusComplete, StartedAt: time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC), CompletedAt: time.Date(2025, 1, 1, 12, 0, 3, 0, time.UTC)},
			{Number: 2, Name: "Architecture", Operative: "Numbuh 2", AgentName: "numbuh-2", Status: pipeline.StatusComplete, StartedAt: time.Date(2025, 1, 1, 12, 0, 3, 0, time.UTC), CompletedAt: time.Date(2025, 1, 1, 12, 0, 5, 0, time.UTC)},
			{Number: 3, Name: "Implementation", Operative: "Numbuh 3", AgentName: "numbuh-3", Status: pipeline.StatusPending},
			{Number: 4, Name: "QA", Operative: "Numbuh 4", AgentName: "numbuh-4", Status: pipeline.StatusPending},
			{Number: 5, Name: "Review", Operative: "Numbuh 5", AgentName: "numbuh-5", Status: pipeline.StatusPending},
			{Number: 6, Name: "Oversight", Operative: "Numbuh 0", AgentName: "numbuh-0", Status: pipeline.StatusPending, Conditional: true},
			{Number: 7, Name: "Security", Operative: "Numbuh 274", AgentName: "numbuh-274", Status: pipeline.StatusPending, Conditional: true},
			{Number: 8, Name: "Deploy Prep", Operative: "Numbuh 362", AgentName: "numbuh-362", Status: pipeline.StatusPending, Conditional: true},
		},
	}

	app.pipeline.State = ps
	app.pipeline.Chat = []PipelineMsg{
		{Agent: "", Content: "━━━ Phase 1: Analysis ━━━"},
		{Agent: "numbuh-1", Content: "Requirements gathered. 3 ACs defined."},
		{Agent: "", Content: "└── ✅ Analysis complete (3.0s)"},
		{Agent: "", Content: "━━━ Phase 2: Architecture ━━━"},
		{Agent: "numbuh-2", Content: "Blueprint ready. 2 files impacted."},
		{Agent: "", Content: "└── ✅ Architecture complete (2.0s)"},
	}

	got := app.View()
	goldenAssert(t, "pipeline", got)
}

// TestGolden_Dossier captures the dossier view for the first agent in the registry.
func TestGolden_Dossier(t *testing.T) {
	lipgloss.SetColorProfile(termenv.TrueColor)
	lipgloss.SetHasDarkBackground(true)

	app := newGoldenApp(t)
	app.view = ViewDossier
	app.dashboard.Selected = 0

	got := app.View()
	goldenAssert(t, "dossier", got)
}
