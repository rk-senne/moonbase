package tui

import (
	"testing"
	"time"
)

func TestNewPipelineModel(t *testing.T) {
	m := NewPipelineModel()
	if m.State != nil {
		t.Error("expected nil State")
	}
	if m.Running {
		t.Error("expected Running=false")
	}
	if m.AbortPending {
		t.Error("expected AbortPending=false")
	}
	if m.Output != nil {
		t.Error("expected nil Output")
	}
	if m.Chat != nil {
		t.Error("expected nil Chat")
	}
}

func TestPipelineModel_IsAbortConfirmed(t *testing.T) {
	tests := []struct {
		name         string
		abortPending bool
		abortAt      time.Time
		want         bool
	}{
		{
			name:         "not pending",
			abortPending: false,
			abortAt:      time.Now(),
			want:         false,
		},
		{
			name:         "pending and within window",
			abortPending: true,
			abortAt:      time.Now(),
			want:         true,
		},
		{
			name:         "pending but expired",
			abortPending: true,
			abortAt:      time.Now().Add(-5 * time.Second),
			want:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := PipelineModel{
				AbortPending: tt.abortPending,
				AbortAt:      tt.abortAt,
			}
			if got := m.IsAbortConfirmed(); got != tt.want {
				t.Errorf("IsAbortConfirmed() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPipelineModel_StateManagement(t *testing.T) {
	m := NewPipelineModel()

	// Simulate pipeline start
	m.Output = []string{"Starting pipeline..."}
	m.Chat = []PipelineMsg{{Agent: "system", Content: "Pipeline initialized"}}
	m.Running = true

	if !m.Running {
		t.Error("expected Running=true after start")
	}
	if len(m.Output) != 1 {
		t.Errorf("expected 1 output line, got %d", len(m.Output))
	}
	if len(m.Chat) != 1 {
		t.Errorf("expected 1 chat msg, got %d", len(m.Chat))
	}

	// Simulate abort
	m.AbortPending = true
	m.AbortAt = time.Now()
	m.Running = false

	if m.Running {
		t.Error("expected Running=false after abort")
	}
	if !m.IsAbortConfirmed() {
		t.Error("expected abort confirmed within window")
	}
}
