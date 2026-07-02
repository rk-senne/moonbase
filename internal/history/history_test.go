package history

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// setHistoryPath overrides the package-level historyPath var to point at a temp dir.
func setHistoryPath(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "history.json")
	historyPath = path
	return path
}

func TestLoad_ReturnsEmptySliceWhenNoFileExists(t *testing.T) {
	setHistoryPath(t)

	missions := Load()
	if missions != nil {
		t.Errorf("expected nil when file doesn't exist, got %v", missions)
	}
}

func TestExport_ReturnsFormattedStringForValidMission(t *testing.T) {
	path := setHistoryPath(t)

	start := time.Date(2026, 7, 1, 10, 0, 0, 0, time.UTC)
	end := time.Date(2026, 7, 1, 10, 30, 0, 0, time.UTC)

	missions := []Mission{
		{
			ID:        1,
			Task:      "Add login feature",
			StartTime: start,
			EndTime:   end,
			Duration:  "30m",
			Outcome:   "complete",
			Phases: []Phase{
				{Name: "Requirements", Status: "done", Duration: "5m"},
				{Name: "Implementation", Status: "done", Duration: "20m"},
				{Name: "QA", Status: "done", Duration: "5m"},
			},
		},
	}

	data, err := json.MarshalIndent(missions, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal test data: %v", err)
	}
	os.MkdirAll(filepath.Dir(path), 0700)
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	result := Export(1)

	if !strings.Contains(result, "Mission #1") {
		t.Errorf("expected 'Mission #1' in output, got:\n%s", result)
	}
	if !strings.Contains(result, "Add login feature") {
		t.Errorf("expected task name in output, got:\n%s", result)
	}
	if !strings.Contains(result, "complete") {
		t.Errorf("expected outcome 'complete' in output, got:\n%s", result)
	}
	if !strings.Contains(result, "Requirements") {
		t.Errorf("expected phase 'Requirements' in output, got:\n%s", result)
	}
	if !strings.Contains(result, "30m") {
		t.Errorf("expected duration '30m' in output, got:\n%s", result)
	}
}

func TestExport_HandlesOutOfRangeGracefully(t *testing.T) {
	path := setHistoryPath(t)

	missions := []Mission{
		{
			ID:        1,
			Task:      "Only mission",
			StartTime: time.Now(),
			Outcome:   "complete",
		},
	}

	data, _ := json.MarshalIndent(missions, "", "  ")
	os.MkdirAll(filepath.Dir(path), 0700)
	os.WriteFile(path, data, 0600)

	result := Export(999)

	if !strings.Contains(result, "not found") {
		t.Errorf("expected 'not found' message for invalid ID, got: %q", result)
	}
}

func TestExport_HandlesEmptyHistory(t *testing.T) {
	setHistoryPath(t)

	result := Export(1)

	if !strings.Contains(result, "not found") {
		t.Errorf("expected 'not found' for empty history, got: %q", result)
	}
}
