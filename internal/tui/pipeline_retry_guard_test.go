package tui

import (
	"errors"
	"testing"
	"time"

	"github.com/rk-senne/moonbase/internal/pipeline"
)

// The auto-retry path runs when a phase has already failed. A panic there would
// replace a recoverable retry with a crashed TUI, so the nil cases are pinned.

// The retry block sits entirely inside an `if state != nil`, which means the
// retries-exhausted branch below it is reachable with a nil state. It previously
// dereferenced state.MaxRetries there and panicked.
func TestHandlePhaseResult_NilStateDoesNotPanic(t *testing.T) {
	app := NewApp()
	app.views.Pipeline.State = nil
	app.views.Pipeline.Running = true

	// Must not panic.
	cmd := app.handlePhaseResult(PhaseResultMsg{
		Phase:   1,
		Err:     errors.New("phase blew up"),
		Elapsed: time.Second,
	})

	if cmd != nil {
		t.Error("expected no follow-up command when there is no pipeline state")
	}
	if app.views.Pipeline.Running {
		t.Error("expected Running to be cleared")
	}
}

// The flywheel entry reads CurrentPhase() and Context, either of which can be
// absent on a partially initialised pipeline.
func TestHandlePhaseResult_RetryWithNilContextDoesNotPanic(t *testing.T) {
	app := NewApp()
	state := pipeline.New("task")
	state.Context = nil // flywheel entry reads ReworkCount from this
	state.MaxRetries = 2
	state.Phases[0].Status = pipeline.StatusRunning
	app.views.Pipeline.State = state
	app.views.Pipeline.Running = true

	// Must not panic, and must schedule a retry since attempts remain.
	cmd := app.handlePhaseResult(PhaseResultMsg{
		Phase:   state.Phases[0].Number,
		Err:     errors.New("transient failure"),
		Elapsed: 250 * time.Millisecond,
	})

	if cmd == nil {
		t.Fatal("expected a retry to be scheduled while attempts remain")
	}
	if got := state.Retries[state.Phases[0].Number]; got != 1 {
		t.Errorf("retry count = %d, want 1", got)
	}
}

// A zero MaxRetries must fall straight through to asking the human rather than
// retrying forever or dividing by zero in the backoff.
func TestHandlePhaseResult_ZeroMaxRetriesAsksHumanImmediately(t *testing.T) {
	app := NewApp()
	state := pipeline.New("task")
	state.MaxRetries = 0
	state.Phases[0].Status = pipeline.StatusRunning
	app.views.Pipeline.State = state
	app.views.Pipeline.Running = true

	cmd := app.handlePhaseResult(PhaseResultMsg{
		Phase:   state.Phases[0].Number,
		Err:     errors.New("hard failure"),
		Elapsed: time.Second,
	})

	if cmd != nil {
		t.Error("expected no retry to be scheduled when MaxRetries is 0")
	}
	if state.Phases[0].Status != pipeline.StatusFailed {
		t.Errorf("phase status = %v, want StatusFailed", state.Phases[0].Status)
	}
}

// A retry delivered after the mission was superseded must be discarded rather
// than re-running a phase belonging to an abandoned run.
func TestHandlePhaseRetryMsg_NilStateIsDiscarded(t *testing.T) {
	app := NewApp()
	app.views.Pipeline.State = nil

	model, cmd := app.handlePhaseRetryMsg(PhaseRetryMsg{
		Phase:   1,
		Attempt: 1,
		Gen:     app.views.Pipeline.Gen,
	})

	if cmd != nil {
		t.Error("expected the retry to be discarded when there is no state")
	}
	if model == nil {
		t.Error("expected the model to be returned")
	}
}
