package tui

import (
	"strings"
	"testing"

	"github.com/rk-senne/moonbase/internal/pipeline"
)

func TestPersonaFor_KnownByIDAndDisplay(t *testing.T) {
	cases := []struct{ in, wantDesig string }{
		{"numbuh-4", "Wallabee Beatles"},
		{"Numbuh 4", "Wallabee Beatles"},
		{"numbuh-274", "Chad Dickson"},
		{"Numbuh 274", "Chad Dickson"},
		{"sector-z", "The Lost Operatives"},
		{"Sector Z", "The Lost Operatives"},
		{"knd-council", "Kids Next Door"},
	}
	for _, c := range cases {
		p := personaFor(c.in)
		if p.Designation != c.wantDesig {
			t.Errorf("personaFor(%q).Designation = %q, want %q", c.in, p.Designation, c.wantDesig)
		}
		if p.Personality == "" {
			t.Errorf("personaFor(%q) missing personality tagline", c.in)
		}
	}
}

func TestPersonaFor_UnknownFallback(t *testing.T) {
	if got := personaFor("mystery-op").Operative; got != "mystery-op" {
		t.Errorf("expected fallback operative label 'mystery-op', got %q", got)
	}
	if got := personaFor("").Operative; got == "" {
		t.Error("expected non-empty fallback label for empty input")
	}
}

func TestRenderPipeline_ShowsPersonaHeader(t *testing.T) {
	app := NewApp()
	app.boot.Ready = true
	app.width = 100
	app.height = 30
	app.view = ViewPipeline
	app.views.Pipeline.State = pipeline.New("test mission")
	app.views.Pipeline.Chat = []PipelineMsg{
		{Agent: "numbuh-4", Content: "Ran the tests. It holds."},
	}

	out := app.renderPipeline()
	if !strings.Contains(out, "Numbuh 4") {
		t.Error("expected operative name in pipeline render")
	}
	if !strings.Contains(out, "Wallabee Beatles") {
		t.Error("expected persona designation in pipeline render")
	}
}

func TestMissionIndicator(t *testing.T) {
	app := NewApp()

	if _, ok := app.missionIndicator(); ok {
		t.Error("expected no indicator when no mission is active")
	}

	app.views.Pipeline.State = pipeline.New("do a thing")
	app.views.Pipeline.State.Phases[0].Status = pipeline.StatusRunning

	seg, ok := app.missionIndicator()
	if !ok {
		t.Fatal("expected active mission indicator")
	}
	if !strings.Contains(seg, "P0/5") {
		t.Errorf("expected mandatory progress 'P0/5', got %q", seg)
	}
	if !strings.Contains(seg, "Analysis") {
		t.Errorf("expected running phase name 'Analysis', got %q", seg)
	}
}
