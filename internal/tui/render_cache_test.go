package tui

import (
	"fmt"
	"testing"
	"time"
)

// renderMarkdown must be deterministic and cache-stable: the same input yields
// identical output across repeated calls, and distinct widths do not collide.
func TestRenderMarkdown_CacheStable(t *testing.T) {
	const md = "# Title\n\nSome **bold** text and a list:\n\n- one\n- two\n"

	first := renderMarkdown(md, 60)
	second := renderMarkdown(md, 60)
	if first != second {
		t.Error("expected identical cached output for repeated renderMarkdown calls")
	}

	// Different width is a different cache entry and should still render.
	other := renderMarkdown(md, 40)
	if other == "" {
		t.Error("expected non-empty render for a different width")
	}

	// Eviction path: many distinct inputs must not panic or corrupt the cache.
	for i := 0; i < mdCacheMax+50; i++ {
		_ = renderMarkdown(fmt.Sprintf("line %d content", i), 60)
	}
	if again := renderMarkdown(md, 60); again != first {
		// The original may have been evicted, but a re-render must still equal
		// the deterministic output.
		t.Error("expected stable deterministic output after cache churn")
	}
}

// The 30s tool-cache refresh must dispatch asynchronously (return a command)
// and the resulting toolCacheMsg must update the cache.
func TestClockTick_AsyncToolCacheRefresh(t *testing.T) {
	app := NewApp()
	app.boot.Ready = true
	app.env.Infra.ToolCacheTime = time.Now().Add(-time.Minute) // force stale

	_, cmd := app.Update(clockTickMsg(time.Now()))
	if cmd == nil {
		t.Fatal("expected a non-nil command to refresh the tool cache asynchronously")
	}

	// The command produces a toolCacheMsg; feeding it back updates the cache.
	msg := cmd()
	tcm, ok := msg.(toolCacheMsg)
	if !ok {
		t.Fatalf("expected toolCacheMsg from refresh command, got %T", msg)
	}
	model, _ := app.Update(tcm)
	if model.(App).env.Infra.ToolCache == nil {
		t.Error("expected tool cache to be populated after toolCacheMsg")
	}
}

// A fresh tool cache (recently stamped) must NOT dispatch a refresh.
func TestClockTick_NoRefreshWhenFresh(t *testing.T) {
	app := NewApp()
	app.boot.Ready = true
	app.env.Infra.ToolCacheTime = time.Now()

	_, cmd := app.Update(clockTickMsg(time.Now()))
	if cmd != nil {
		if _, ok := cmd().(toolCacheMsg); ok {
			t.Error("did not expect a tool-cache refresh when cache is fresh")
		}
	}
}
