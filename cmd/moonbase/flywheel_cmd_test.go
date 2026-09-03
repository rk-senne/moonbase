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

func TestFormatTokenCount(t *testing.T) {
	tests := []struct {
		input int
		want  string
	}{
		{0, "0"},
		{500, "500"},
		{1000, "1K"},
		{45000, "45K"},
		{142000, "142K"},
		{1200000, "1.2M"},
		{2500000, "2.5M"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := formatTokenCount(tt.input)
			if got != tt.want {
				t.Errorf("formatTokenCount(%d) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestTruncateTrace(t *testing.T) {
	short := "trace-123"
	if got := truncateTrace(short); got != short {
		t.Errorf("truncateTrace(%q) = %q, want %q", short, got, short)
	}

	long := "20260730T120000-very-long-trace-id-here"
	got := truncateTrace(long)
	if len(got) > 24 { // 20 + "..."
		t.Errorf("truncateTrace did not truncate: %q", got)
	}
	if got[len(got)-3:] != "..." {
		t.Error("truncateTrace should end with ...")
	}
}

func TestDisplayTokenCostInsights_NoData(t *testing.T) {
	// When no entries have token data, should not panic.
	// We can't easily capture stdout here, but we verify no panic.
	entries := []pipeline.FlywheelEntry{
		{Phase: 1, Agent: "numbuh-1", Outcome: "complete", DurationMs: 100},
		{Phase: 3, Agent: "numbuh-3", Outcome: "complete", DurationMs: 200},
	}

	// Should not panic — graceful degradation
	displayTokenCostInsights(entries)
}

func TestDisplayTokenCostInsights_WithData(t *testing.T) {
	// Entries with token data — should not panic.
	entries := []pipeline.FlywheelEntry{
		{
			TraceID:          "trace-1",
			Phase:            1,
			Agent:            "numbuh-1",
			Outcome:          "complete",
			PromptTokens:     30000,
			CompletionTokens: 8000,
			TotalTokens:      38000,
			Model:            "gpt-4o",
			EstimatedCostUSD: 0.155,
		},
		{
			TraceID:          "trace-1",
			Phase:            3,
			Agent:            "numbuh-3",
			Outcome:          "complete",
			PromptTokens:     60000,
			CompletionTokens: 18000,
			TotalTokens:      78000,
			Model:            "gpt-4o",
			EstimatedCostUSD: 0.33,
		},
		{
			TraceID:          "trace-2",
			Phase:            1,
			Agent:            "numbuh-1",
			Outcome:          "complete",
			PromptTokens:     25000,
			CompletionTokens: 6000,
			TotalTokens:      31000,
			Model:            "gpt-4o",
			EstimatedCostUSD: 0.1225,
		},
		// Entry without tokens — should be excluded from averages
		{
			TraceID: "trace-2",
			Phase:   4,
			Agent:   "numbuh-4",
			Outcome: "complete",
		},
	}

	// Should not panic, should handle mixed data gracefully
	displayTokenCostInsights(entries)
}

func TestDisplayTokenCostInsights_Empty(t *testing.T) {
	// Empty entries list
	displayTokenCostInsights(nil)
}
