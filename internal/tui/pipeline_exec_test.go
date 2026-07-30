package tui

import (
	"fmt"
	"testing"
	"time"

	"github.com/rk-senne/moonbase/internal/pipeline"
)

func TestPhaseResultMsg_Success(t *testing.T) {
	app := NewApp()
	app.view = ViewPipeline
	app.boot.Ready = true
	app.views.Pipeline.State = pipeline.New("test task")
	app.views.Pipeline.State.Phases[0].Status = pipeline.StatusRunning
	app.views.Pipeline.Running = true

	msg := PhaseResultMsg{
		Phase:   1,
		Output:  "Analysis complete. Requirements gathered.",
		Err:     nil,
		Elapsed: 2 * time.Second,
	}

	model, _ := app.Update(msg)
	result := model.(App)

	// Pipeline should have advanced
	if result.views.Pipeline.State.Current != 1 {
		t.Errorf("expected pipeline to advance to phase index 1, got %d", result.views.Pipeline.State.Current)
	}
	if result.views.Pipeline.State.Phases[0].Status != pipeline.StatusComplete {
		t.Errorf("expected phase 0 to be complete, got %d", result.views.Pipeline.State.Phases[0].Status)
	}
	// Output should be recorded in context
	if result.views.Pipeline.State.Context.PhaseOutputs[1] == "" {
		t.Error("expected phase output to be recorded in context")
	}
}

func TestPhaseResultMsg_Error(t *testing.T) {
	app := NewApp()
	app.view = ViewPipeline
	app.boot.Ready = true
	app.views.Pipeline.State = pipeline.New("test task")
	app.views.Pipeline.State.Phases[0].Status = pipeline.StatusRunning
	app.views.Pipeline.Running = true

	msg := PhaseResultMsg{
		Phase:   1,
		Output:  "",
		Err:     fmt.Errorf("agent numbuh-1 not found"),
		Elapsed: 500 * time.Millisecond,
	}

	model, _ := app.Update(msg)
	result := model.(App)

	// Pipeline should NOT advance
	if result.views.Pipeline.State.Current != 0 {
		t.Errorf("expected pipeline to stay at phase 0, got %d", result.views.Pipeline.State.Current)
	}
	// Current phase should be marked as failed
	if result.views.Pipeline.State.Phases[0].Status != pipeline.StatusFailed {
		t.Errorf("expected phase 0 to be failed, got %d", result.views.Pipeline.State.Phases[0].Status)
	}
	// Should not be running
	if result.views.Pipeline.Running {
		t.Error("expected pipelineRunning=false after error")
	}
	// Chat should contain error message
	found := false
	for _, msg := range result.views.Pipeline.Chat {
		if msg.Agent == "" && contains(msg.Content, "failed") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected pipeline chat to contain failure message")
	}
}

func TestPhaseResultMsg_RiskGate_Low(t *testing.T) {
	app := NewApp()
	app.view = ViewPipeline
	app.boot.Ready = true
	app.views.Pipeline.State = pipeline.New("test task")
	// Advance to phase 4 (QA) — set phases 0-3 as complete
	for i := 0; i < 3; i++ {
		app.views.Pipeline.State.Advance()
	}
	app.views.Pipeline.State.Phases[3].Status = pipeline.StatusRunning
	app.views.Pipeline.Running = true

	msg := PhaseResultMsg{
		Phase:   4,
		Output:  "## Verdict\nLOW\n\nAll tests pass.",
		Err:     nil,
		Elapsed: 5 * time.Second,
	}

	model, _ := app.Update(msg)
	result := model.(App)

	// LOW risk — pipeline should advance past QA to Review (phase 5)
	if result.views.Pipeline.State.Context.RiskLevel != "LOW" {
		t.Errorf("expected risk level LOW, got %s", result.views.Pipeline.State.Context.RiskLevel)
	}
	// Should have advanced (current >= 4 since phase 4 index is 3)
	if result.views.Pipeline.State.Current < 4 {
		t.Errorf("expected pipeline to advance past QA (index 3), got current=%d", result.views.Pipeline.State.Current)
	}
}

func TestPhaseResultMsg_RiskGate_Medium(t *testing.T) {
	app := NewApp()
	app.view = ViewPipeline
	app.boot.Ready = true
	app.views.Pipeline.State = pipeline.New("test task")
	for i := 0; i < 3; i++ {
		app.views.Pipeline.State.Advance()
	}
	app.views.Pipeline.State.Phases[3].Status = pipeline.StatusRunning
	app.views.Pipeline.Running = true

	msg := PhaseResultMsg{
		Phase:   4,
		Output:  "## Verdict\nMEDIUM\n\nSome tests need fixing.",
		Err:     nil,
		Elapsed: 3 * time.Second,
	}

	model, _ := app.Update(msg)
	result := model.(App)

	if result.views.Pipeline.State.Context.RiskLevel != "MEDIUM" {
		t.Errorf("expected risk level MEDIUM, got %s", result.views.Pipeline.State.Context.RiskLevel)
	}
	// MEDIUM should route back to implementation (phase 3, index 2)
	if result.views.Pipeline.State.Current != 2 {
		t.Errorf("expected pipeline to route back to phase index 2 (implementation), got %d", result.views.Pipeline.State.Current)
	}
}

func TestPhaseResultMsg_RiskGate_Critical(t *testing.T) {
	app := NewApp()
	app.view = ViewPipeline
	app.boot.Ready = true
	app.views.Pipeline.State = pipeline.New("test task")
	for i := 0; i < 3; i++ {
		app.views.Pipeline.State.Advance()
	}
	app.views.Pipeline.State.Phases[3].Status = pipeline.StatusRunning
	app.views.Pipeline.Running = true

	msg := PhaseResultMsg{
		Phase:   4,
		Output:  "## Verdict\nCRITICAL\n\nSecurity breach detected.",
		Err:     nil,
		Elapsed: 4 * time.Second,
	}

	model, _ := app.Update(msg)
	result := model.(App)

	if result.views.Pipeline.State.Context.RiskLevel != "CRITICAL" {
		t.Errorf("expected risk level CRITICAL, got %s", result.views.Pipeline.State.Context.RiskLevel)
	}
	// CRITICAL should stop the pipeline
	if result.views.Pipeline.State.Active {
		t.Error("expected pipeline to be stopped on CRITICAL risk")
	}
}

func TestStartNextPhase_NoBackend(t *testing.T) {
	app := NewApp()
	app.views.Pipeline.State = pipeline.New("test task")
	app.views.Pipeline.State.Phases[0].Status = pipeline.StatusRunning
	app.env.Backend.Active = nil

	cmd := app.startNextPhase()
	if cmd != nil {
		t.Error("expected nil cmd when no backend is set (simulated mode)")
	}
	if app.views.Pipeline.Running {
		t.Error("expected pipelineRunning=false when no backend")
	}
}

func TestStartNextPhase_ConditionalSkip(t *testing.T) {
	app := NewApp()
	app.views.Pipeline.State = pipeline.New("test task")
	app.env.Backend.Active = nil
	// Move to a conditional phase (index 5 = phase 6, Oversight)
	app.views.Pipeline.State.Current = 5
	app.views.Pipeline.State.Phases[5].Status = pipeline.StatusRunning

	cmd := app.startNextPhase()
	// No backend, so nil anyway, but the conditional check should also skip
	if cmd != nil {
		t.Error("expected nil cmd for conditional phase with no backend")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstr(s, substr))
}

func containsSubstr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
