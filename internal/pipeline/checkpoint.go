// Pipeline state checkpointing for crash recovery and mission replay.
//
// Checkpoints serialize the pipeline state to JSON files under
// ~/.moonbase/checkpoints/{traceID}.json. This enables:
// - Resuming missions after unexpected termination
// - Replaying historical missions with moonbase replay
// - Post-mortem analysis of failed pipelines
//
// Files are written with 0600 permissions (owner-only).
package pipeline

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Checkpoint captures pipeline state for persistence and recovery.
// Stored as JSON in ~/.moonbase/checkpoints/{traceID}.json.
type Checkpoint struct {
	TraceID      string         `json:"trace_id"`
	Task         string         `json:"task"`
	Current      int            `json:"current"`
	PhaseStatuses map[int]string `json:"phase_statuses"`
	PhaseOutputs map[int]string `json:"phase_outputs"`
	ReworkCount  int            `json:"rework_count"`
	RiskLevel    string         `json:"risk_level"`
	CreatedAt    time.Time      `json:"created_at"`
}

// SaveCheckpoint serializes the pipeline state to a JSON file.
// Files are stored in dir/{traceID}.json with 0600 permissions.
func SaveCheckpoint(p *Pipeline, dir string) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("creating checkpoint directory: %w", err)
	}

	statuses := make(map[int]string)
	for _, phase := range p.Phases {
		statuses[phase.Number] = statusName(phase.Status)
	}

	cp := Checkpoint{
		TraceID:       p.TraceID,
		Task:          p.Task,
		Current:       p.Current,
		PhaseStatuses: statuses,
		PhaseOutputs:  p.Context.PhaseOutputs,
		ReworkCount:   p.Context.ReworkCount,
		RiskLevel:     p.Context.RiskLevel,
		CreatedAt:     time.Now().UTC(),
	}

	data, err := json.MarshalIndent(cp, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling checkpoint: %w", err)
	}

	path := filepath.Join(dir, cp.TraceID+".json")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("writing checkpoint file: %w", err)
	}

	return nil
}

// LoadCheckpoint reads a checkpoint file from dir/{traceID}.json.
func LoadCheckpoint(dir string, traceID string) (*Checkpoint, error) {
	path := filepath.Join(dir, traceID+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading checkpoint file: %w", err)
	}

	var cp Checkpoint
	if err := json.Unmarshal(data, &cp); err != nil {
		return nil, fmt.Errorf("unmarshaling checkpoint: %w", err)
	}

	return &cp, nil
}

// statusName converts a PhaseStatus to its string representation.
func statusName(s PhaseStatus) string {
	switch s {
	case StatusPending:
		return "pending"
	case StatusRunning:
		return "running"
	case StatusComplete:
		return "complete"
	case StatusSkipped:
		return "skipped"
	case StatusFailed:
		return "failed"
	case StatusRework:
		return "rework"
	default:
		return "unknown"
	}
}
