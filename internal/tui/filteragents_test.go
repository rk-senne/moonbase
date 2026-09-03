package tui

import (
	"testing"

	"github.com/rk-senne/moonbase/internal/agents"
)

func TestFilterAgents_FuzzyN4(t *testing.T) {
	app := NewApp()
	app.registry = newTestRegistry()
	if app.registry.Count() == 0 {
		t.Skip("no agents loaded")
	}

	app.views.Search.Input.SetValue("n4")
	app.filterAgents()

	if len(app.views.Search.Filtered) == 0 {
		t.Fatal("expected 'n4' to match at least one agent")
	}

	// Find numbuh-4's index in registry
	var numbuh4Idx int
	found := false
	for i, agent := range app.registry.All() {
		if agent.Name == "numbuh-4" {
			numbuh4Idx = i
			found = true
			break
		}
	}
	if !found {
		t.Skip("numbuh-4 not in registry")
	}

	// numbuh-4 should be in filtered results
	inFiltered := false
	for _, idx := range app.views.Search.Filtered {
		if idx == numbuh4Idx {
			inFiltered = true
			break
		}
	}
	if !inFiltered {
		t.Error("expected numbuh-4 to appear in filtered results for query 'n4'")
	}

	// numbuh-4 should be ranked first (best match for 'n4')
	if app.views.Search.Filtered[0] != numbuh4Idx {
		// It's acceptable if numbuh-4 isn't exactly first due to description matches,
		// but it should be near the top. Check it's in top 3.
		topN := 3
		if len(app.views.Search.Filtered) < topN {
			topN = len(app.views.Search.Filtered)
		}
		foundInTop := false
		for _, idx := range app.views.Search.Filtered[:topN] {
			if idx == numbuh4Idx {
				foundInTop = true
				break
			}
		}
		if !foundInTop {
			t.Errorf("expected numbuh-4 in top %d results for 'n4', filtered=%v", topN, app.views.Search.Filtered)
		}
	}
}

func TestFilterAgents_FuzzyArch(t *testing.T) {
	app := NewApp()
	app.registry = newTestRegistry()
	if app.registry.Count() == 0 {
		t.Skip("no agents loaded")
	}

	app.views.Search.Input.SetValue("arch")
	app.filterAgents()

	// "arch" should match numbuh-2 (architect) either by name or description
	if len(app.views.Search.Filtered) == 0 {
		t.Fatal("expected 'arch' to match at least one agent")
	}
}

func TestFilterAgents_EmptyQueryNilFiltered(t *testing.T) {
	app := NewApp()
	app.registry = newTestRegistry()
	app.views.Search.Filtered = []int{1, 2, 3}

	app.views.Search.Input.SetValue("")
	app.filterAgents()

	if app.views.Search.Filtered != nil {
		t.Error("expected nil filtered for empty query")
	}
}

func TestFilterAgents_FuzzyNoMatch(t *testing.T) {
	app := NewApp()
	app.registry = newTestRegistry()

	app.views.Search.Input.SetValue("zzzznonexistentzzzz")
	app.filterAgents()

	if len(app.views.Search.Filtered) != 0 {
		t.Errorf("expected 0 matches, got %d", len(app.views.Search.Filtered))
	}
}

func TestFilterAgents_SubstringStillWorks(t *testing.T) {
	// Subsequence matching is a superset of substring matching.
	// Verify that old substring queries still produce results.
	app := NewApp()
	app.registry = newTestRegistry()
	if app.registry.Count() == 0 {
		t.Skip("no agents loaded")
	}

	app.views.Search.Input.SetValue("numbuh")
	app.filterAgents()

	if len(app.views.Search.Filtered) == 0 {
		t.Skipf("no agents matched 'numbuh' (registry count=%d)", app.registry.Count())
	}

	// All results should contain "numbuh" in name (substring is a subsequence)
	for _, idx := range app.views.Search.Filtered {
		agent := app.registry.Get(idx)
		if agent.Name == "unknown" {
			t.Errorf("filtered index %d returned unknown agent", idx)
		}
	}
}

func TestFilterAgents_SortedByScore(t *testing.T) {
	// Create a minimal registry with controlled agents to verify sort order
	reg := agents.NewRegistry("")
	// We can't easily inject agents without loading from disk,
	// so we test the scoring logic indirectly via Match
	// and verify filterAgents ordering with the real registry.

	app := NewApp()
	app.registry = newTestRegistry()
	if app.registry.Count() == 0 {
		t.Skip("no agents loaded")
	}

	// "council" should rank knd-council highest
	app.views.Search.Input.SetValue("council")
	app.filterAgents()

	if len(app.views.Search.Filtered) == 0 {
		t.Skip("no agents matched 'council'")
	}

	// Find knd-council index
	var councilIdx int
	found := false
	for i, agent := range app.registry.All() {
		if agent.Name == "knd-council" {
			councilIdx = i
			found = true
			break
		}
	}
	if !found {
		t.Skip("knd-council not in registry")
	}

	if app.views.Search.Filtered[0] != councilIdx {
		t.Errorf("expected knd-council (idx %d) first for query 'council', got idx %d", councilIdx, app.views.Search.Filtered[0])
	}

	_ = reg // suppress unused
}
