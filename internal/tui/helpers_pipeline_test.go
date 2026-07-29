package tui

import (
	"strings"
	"testing"

	"github.com/rk-senne/moonbase/internal/pipeline"
)

func TestStatusIcon_AllStatuses(t *testing.T) {
	tests := []struct {
		status pipeline.PhaseStatus
		icon   string
	}{
		{pipeline.StatusPending, "⏳"},
		{pipeline.StatusRunning, "🔄"},
		{pipeline.StatusComplete, "✅"},
		{pipeline.StatusSkipped, "⏭️"},
		{pipeline.StatusFailed, "❌"},
		{pipeline.StatusRework, "🔁"},
		{pipeline.PhaseStatus(99), "?"},
	}

	for _, tt := range tests {
		got := StatusIcon(tt.status)
		if got != tt.icon {
			t.Errorf("StatusIcon(%d) = %q, want %q", tt.status, got, tt.icon)
		}
	}
}

func TestPipelineStatusSummary(t *testing.T) {
	p := pipeline.New("test mission")
	p.Phases[0].Status = pipeline.StatusComplete
	p.Phases[1].Status = pipeline.StatusRunning

	summary := PipelineStatusSummary(p)

	if !strings.Contains(summary, "test mission") {
		t.Error("expected task in summary")
	}
	if !strings.Contains(summary, "✅") {
		t.Error("expected complete icon")
	}
	if !strings.Contains(summary, "🔄") {
		t.Error("expected running icon")
	}
	if !strings.Contains(summary, "(conditional)") {
		t.Error("expected conditional tag for specialist phases")
	}
	if !strings.Contains(summary, "Phase 1") {
		t.Error("expected Phase 1 in summary")
	}
}
