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

func TestSave_CreatesFileAndAssignsID(t *testing.T) {
	setHistoryPath(t)

	m := Mission{
		Task:      "test mission",
		StartTime: time.Now(),
		Outcome:   "in-progress",
	}

	id, err := Save(m)
	if err != nil {
		t.Fatalf("Save() failed: %v", err)
	}
	if id != 1 {
		t.Errorf("expected ID 1, got %d", id)
	}

	// Save another
	m2 := Mission{
		Task:      "second mission",
		StartTime: time.Now(),
		Outcome:   "in-progress",
	}
	id2, err := Save(m2)
	if err != nil {
		t.Fatalf("Save() second failed: %v", err)
	}
	if id2 != 2 {
		t.Errorf("expected ID 2, got %d", id2)
	}

	// Verify persistence
	missions := Load()
	if len(missions) != 2 {
		t.Fatalf("expected 2 missions, got %d", len(missions))
	}
	if missions[0].Task != "test mission" {
		t.Errorf("expected first task 'test mission', got %q", missions[0].Task)
	}
	if missions[1].Task != "second mission" {
		t.Errorf("expected second task 'second mission', got %q", missions[1].Task)
	}
}

func TestSave_CreatesDirectoryWithCorrectPermissions(t *testing.T) {
	tmpDir := t.TempDir()
	historyPath = filepath.Join(tmpDir, "subdir", "history.json")

	m := Mission{
		Task:      "test",
		StartTime: time.Now(),
		Outcome:   "complete",
	}

	_, err := Save(m)
	if err != nil {
		t.Fatalf("Save() failed: %v", err)
	}

	// Check directory permissions
	info, err := os.Stat(filepath.Join(tmpDir, "subdir"))
	if err != nil {
		t.Fatalf("directory not created: %v", err)
	}
	perm := info.Mode().Perm()
	if perm != 0o700 {
		t.Errorf("expected directory permissions 0700, got %o", perm)
	}
}

func TestUpdate_ModifiesExistingMission(t *testing.T) {
	setHistoryPath(t)

	m := Mission{
		Task:      "updatable mission",
		StartTime: time.Now(),
		Outcome:   "in-progress",
	}
	id, _ := Save(m)

	// Update it
	m.ID = id
	m.Outcome = "complete"
	m.Duration = "5m"
	err := Update(m)
	if err != nil {
		t.Fatalf("Update() failed: %v", err)
	}

	// Verify
	got := GetByID(id)
	if got == nil {
		t.Fatal("GetByID returned nil after Update")
	}
	if got.Outcome != "complete" {
		t.Errorf("expected outcome 'complete', got %q", got.Outcome)
	}
	if got.Duration != "5m" {
		t.Errorf("expected duration '5m', got %q", got.Duration)
	}
}

func TestUpdate_ReturnsErrorForNonexistentID(t *testing.T) {
	setHistoryPath(t)

	m := Mission{ID: 999, Task: "ghost", Outcome: "complete"}
	err := Update(m)
	if err == nil {
		t.Error("expected error for nonexistent ID, got nil")
	}
}

func TestGetByID_ReturnsNilForMissingID(t *testing.T) {
	setHistoryPath(t)

	got := GetByID(42)
	if got != nil {
		t.Errorf("expected nil for missing ID, got %+v", got)
	}
}

func TestGetByID_ReturnsMission(t *testing.T) {
	setHistoryPath(t)

	m := Mission{Task: "find me", StartTime: time.Now(), Outcome: "complete"}
	id, _ := Save(m)

	got := GetByID(id)
	if got == nil {
		t.Fatal("expected mission, got nil")
	}
	if got.Task != "find me" {
		t.Errorf("expected task 'find me', got %q", got.Task)
	}
}

func TestList_ReturnsLastN(t *testing.T) {
	setHistoryPath(t)

	for i := 1; i <= 5; i++ {
		Save(Mission{Task: strings.Repeat("x", i), StartTime: time.Now(), Outcome: "complete"})
	}

	// Get last 3
	result := List(3)
	if len(result) != 3 {
		t.Fatalf("expected 3 missions, got %d", len(result))
	}
	// Should be missions 3, 4, 5
	if result[0].ID != 3 {
		t.Errorf("expected first result ID 3, got %d", result[0].ID)
	}
	if result[2].ID != 5 {
		t.Errorf("expected last result ID 5, got %d", result[2].ID)
	}
}

func TestList_ReturnsAllWhenLimitZero(t *testing.T) {
	setHistoryPath(t)

	for i := 0; i < 3; i++ {
		Save(Mission{Task: "all", StartTime: time.Now(), Outcome: "complete"})
	}

	result := List(0)
	if len(result) != 3 {
		t.Errorf("expected 3 missions with limit 0, got %d", len(result))
	}
}

func TestList_ReturnsAllWhenLimitExceedsCount(t *testing.T) {
	setHistoryPath(t)

	Save(Mission{Task: "only one", StartTime: time.Now(), Outcome: "complete"})

	result := List(100)
	if len(result) != 1 {
		t.Errorf("expected 1 mission, got %d", len(result))
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
