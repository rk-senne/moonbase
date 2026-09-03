// Flywheel session logging for pipeline self-improvement.
//
// The flywheel pattern (adapted from Kiro CLI and AWS agent best practices):
// every pipeline phase execution is logged as a JSONL entry. Over time,
// patterns emerge -- which agents get reworked most, which phases are slow,
// where risk gates trigger. This data feeds `moonbase flywheel` analysis.
//
// Log location: ~/.moonbase/flywheel.jsonl (append-only, 0600 permissions).
package pipeline

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// currentSchemaVersion is the version written to all new FlywheelEntry records.
// Evolution contract: new optional fields may be added without bumping the version.
// Removing or renaming an existing field requires incrementing the version.
// Readers must tolerate SchemaVersion == 0 (legacy entries written before versioning).
const currentSchemaVersion = 1

// FlywheelEntry records a single pipeline turn for later analysis.
// The flywheel pattern (from Kiro/AWS): log agent interactions so patterns
// of corrections, failures, and rework can be analyzed to improve config.
type FlywheelEntry struct {
	SchemaVersion int       `json:"v"`
	Timestamp     time.Time `json:"timestamp"`
	TraceID       string    `json:"trace_id"`
	Phase         int       `json:"phase"`
	Agent         string    `json:"agent"`
	Task          string    `json:"task"`
	Outcome       string    `json:"outcome"` // "complete", "rework", "failed", "skipped", "budget_exceeded"
	RiskLevel     string    `json:"risk_level,omitempty"`
	DurationMs    int64     `json:"duration_ms"`
	OutputSize    int       `json:"output_size"`
	ReworkCount   int       `json:"rework_count"`
	// Token/cost observability (added without schema version bump per evolution contract).
	// These fields are omitempty: zero values mean "no data" (not "zero tokens consumed").
	PromptTokens     int     `json:"prompt_tokens,omitempty"`
	CompletionTokens int     `json:"completion_tokens,omitempty"`
	TotalTokens      int     `json:"total_tokens,omitempty"`
	Model            string  `json:"model,omitempty"`
	EstimatedCostUSD float64 `json:"estimated_cost_usd,omitempty"`
	// Parallel specialist fan-out correlation (added without schema version bump).
	ParallelGroup string `json:"parallel_group,omitempty"` // Groups entries from same fan-out batch
	// Adaptive pipeline depth observability (added without schema version bump per evolution contract).
	Depth         string `json:"depth,omitempty"`          // Pipeline depth for this run
	DepthReason   string `json:"depth_reason,omitempty"`   // Why this depth was selected
	EscalatedFrom string `json:"escalated_from,omitempty"` // Original depth before escalation
	EscalatedTo   string `json:"escalated_to,omitempty"`   // New depth after escalation
}

// FlywheelLog manages append-only JSONL logging for flywheel analysis.
type FlywheelLog struct {
	path string
}

// NewFlywheelLog creates a flywheel logger. Logs to ~/.moonbase/flywheel.jsonl.
func NewFlywheelLog() *FlywheelLog {
	home, _ := os.UserHomeDir()
	path := filepath.Join(home, ".moonbase", "flywheel.jsonl")
	return &FlywheelLog{path: path}
}

// Append writes a flywheel entry to the log file.
// Single-writer assumed; no file locking is performed. Calls file.Sync()
// to ensure durability before closing.
func (f *FlywheelLog) Append(entry FlywheelEntry) error {
	if err := os.MkdirAll(filepath.Dir(f.path), 0o700); err != nil {
		return fmt.Errorf("creating flywheel directory: %w", err)
	}

	file, err := os.OpenFile(f.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("opening flywheel log: %w", err)
	}
	defer file.Close()

	entry.SchemaVersion = currentSchemaVersion

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("marshaling flywheel entry: %w", err)
	}

	if _, err := file.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("writing flywheel entry: %w", err)
	}

	if err := file.Sync(); err != nil {
		return fmt.Errorf("syncing flywheel log: %w", err)
	}

	return nil
}

// Path returns the flywheel log file path.
func (f *FlywheelLog) Path() string {
	return f.path
}
