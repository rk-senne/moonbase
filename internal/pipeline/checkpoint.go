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

// currentCheckpointVersion is the version written to all new Checkpoint records.
// Evolution contract: new optional fields may be added without bumping the version.
// Removing or renaming an existing field requires incrementing the version.
// Readers must tolerate SchemaVersion == 0 (legacy checkpoints written before versioning).
const currentCheckpointVersion = 1

// Checkpoint captures pipeline state for persistence and recovery.
// Stored as JSON in ~/.moonbase/checkpoints/{traceID}.json.
type Checkpoint struct {
	SchemaVersion  int            `json:"v"`
	TraceID        string         `json:"trace_id"`
	Task           string         `json:"task"`
	Current        int            `json:"current"`
	PhaseStatuses  map[int]string `json:"phase_statuses"`
	PhaseOutputs   map[int]string `json:"phase_outputs"`
	ReworkCount    int            `json:"rework_count"`
	RiskLevel      string         `json:"risk_level"`
	CreatedAt      time.Time      `json:"created_at"`
}

// SaveCheckpoint serializes the pipeline state to a JSON file.
// Files are stored in dir/{traceID}.json with 0600 permissions.
// Uses write-to-temp-then-rename for crash safety.
func SaveCheckpoint(p *Pipeline, dir string) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("creating checkpoint directory: %w", err)
	}

	statuses := make(map[int]string)
	for _, phase := range p.Phases {
		statuses[phase.Number] = statusName(phase.Status)
	}

	cp := Checkpoint{
		SchemaVersion:  currentCheckpointVersion,
		TraceID:        p.TraceID,
		Task:           p.Task,
		Current:        p.Current,
		PhaseStatuses:  statuses,
		PhaseOutputs:   p.Context.PhaseOutputs,
		ReworkCount:    p.Context.ReworkCount,
		RiskLevel:      p.Context.RiskLevel,
		CreatedAt:      time.Now().UTC(),
	}

	data, err := json.MarshalIndent(cp, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling checkpoint: %w", err)
	}

	path := filepath.Join(dir, cp.TraceID+".json")
	tmpPath := path + ".tmp"

	if err := os.WriteFile(tmpPath, data, 0o600); err != nil {
		return fmt.Errorf("writing temp checkpoint file: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("renaming checkpoint file: %w", err)
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
