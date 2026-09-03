package pipeline

import (
	"strings"
	"testing"
)

// Checkpointing runs on failure paths — a blown token budget, an aborted phase.
// An incomplete pipeline must produce an error there, not a panic that also
// destroys the checkpoint being written.

func TestSaveCheckpoint_NilPipelineReturnsError(t *testing.T) {
	err := SaveCheckpoint(nil, t.TempDir())
	if err == nil {
		t.Fatal("expected an error for a nil pipeline")
	}
	if !strings.Contains(err.Error(), "nil pipeline") {
		t.Errorf("error = %q, want it to mention a nil pipeline", err)
	}
}

func TestSaveCheckpoint_NilContextReturnsErrorNotPanic(t *testing.T) {
	p := &Pipeline{TraceID: "trace-abc", Task: "t"} // Context deliberately nil

	err := SaveCheckpoint(p, t.TempDir())
	if err == nil {
		t.Fatal("expected an error for a pipeline with no context")
	}
	if !strings.Contains(err.Error(), "trace-abc") {
		t.Errorf("error = %q, want it to identify the trace", err)
	}
}

func TestSaveCheckpoint_CompletePipelineSucceeds(t *testing.T) {
	p := &Pipeline{
		TraceID: "trace-ok",
		Task:    "do the thing",
		Context: &PipelineContext{},
		Phases:  []Phase{{Number: 1, AgentName: "numbuh-1"}},
	}

	if err := SaveCheckpoint(p, t.TempDir()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
