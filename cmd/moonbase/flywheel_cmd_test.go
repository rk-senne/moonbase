package main

import (
	"testing"
	"time"

	"github.com/rk-senne/moonbase/internal/pipeline"
)

func TestComputeLeadTimeInsights_SingleMission(t *testing.T) {
	start := time.Date(2026, 7, 20, 12, 0, 0, 0, time.UTC)
	traceEntries := map[string][]pipeline.FlywheelEntry{
		"trace-1": {
			{Timestamp: start, Phase: 1, Agent: "numbuh-1", DurationMs: 1000},
			{Timestamp: start.Add(2 * time.Second), Phase: 2, Agent: "numbuh-2", DurationMs: 1500},
			{Timestamp: start.Add(5 * time.Second), Phase: 3, Agent: "numbuh-3", DurationMs: 3000},
			{Timestamp: start.Add(9 * time.Second), Phase: 4, Agent: "numbuh-4", DurationMs: 2000},
		},
	}

	stats, longest := computeLeadTimeInsights(traceEntries)

	// Lead time = (last.Timestamp - first.Timestamp) + last.DurationMs
	// = 9s + 2000ms = 11s
	expectedLeadTime := 11 * time.Second
	if stats.avg != expectedLeadTime {
		t.Errorf("avg lead time: got %s, want %s", stats.avg, expectedLeadTime)
	}
	if stats.count != 1 {
		t.Errorf("count: got %d, want 1", stats.count)
	}

	// Longest phase should be Phase 3 (3000ms avg)
	if longest.name != "Phase 3 (numbuh-3)" {
		t.Errorf("longest phase: got %q, want %q", longest.name, "Phase 3 (numbuh-3)")
	}
	if longest.duration != 3*time.Second {
		t.Errorf("longest duration: got %s, want %s", longest.duration, 3*time.Second)
	}
}

func TestComputeLeadTimeInsights_MultipleMissions(t *testing.T) {
	start1 := time.Date(2026, 7, 20, 12, 0, 0, 0, time.UTC)
	start2 := time.Date(2026, 7, 20, 14, 0, 0, 0, time.UTC)

	traceEntries := map[string][]pipeline.FlywheelEntry{
		"trace-1": {
			{Timestamp: start1, Phase: 1, Agent: "numbuh-1", DurationMs: 1000},
			{Timestamp: start1.Add(2 * time.Second), Phase: 4, Agent: "numbuh-4", DurationMs: 2000},
		},
		"trace-2": {
			{Timestamp: start2, Phase: 1, Agent: "numbuh-1", DurationMs: 500},
			{Timestamp: start2.Add(10 * time.Second), Phase: 4, Agent: "numbuh-4", DurationMs: 4000},
		},
	}

	stats, _ := computeLeadTimeInsights(traceEntries)

	// trace-1 lead time: 2s + 2000ms = 4s
	// trace-2 lead time: 10s + 4000ms = 14s
	// avg = (4s + 14s) / 2 = 9s
	expectedAvg := 9 * time.Second
	if stats.avg != expectedAvg {
		t.Errorf("avg lead time: got %s, want %s", stats.avg, expectedAvg)
	}
	if stats.count != 2 {
		t.Errorf("count: got %d, want 2", stats.count)
	}
}

func TestComputeLeadTimeInsights_EmptyInput(t *testing.T) {
	traceEntries := map[string][]pipeline.FlywheelEntry{}

	stats, longest := computeLeadTimeInsights(traceEntries)

	if stats.count != 0 {
		t.Errorf("count: got %d, want 0", stats.count)
	}
	if stats.avg != 0 {
		t.Errorf("avg: got %s, want 0", stats.avg)
	}
	if longest.name != "" {
		t.Errorf("longest name: got %q, want empty", longest.name)
	}
}
