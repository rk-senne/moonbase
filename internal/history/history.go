package history

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Mission represents a pipeline execution record.
type Mission struct {
	ID        int       `json:"id"`
	Task      string    `json:"task"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time,omitempty"`
	Duration  string    `json:"duration,omitempty"`
	Phases    []Phase   `json:"phases"`
	Outcome   string    `json:"outcome"` // "complete", "aborted", "in-progress"
}

// Phase represents a single phase execution within a mission.
type Phase struct {
	Name     string `json:"name"`
	Status   string `json:"status"`
	Duration string `json:"duration,omitempty"`
}

var historyPath string

func init() {
	home, _ := os.UserHomeDir()
	historyPath = filepath.Join(home, ".config", "moonbase", "history.json")
}

// Load reads all missions from the history file.
func Load() []Mission {
	data, err := os.ReadFile(historyPath)
	if err != nil {
		return nil
	}
	var missions []Mission
	json.Unmarshal(data, &missions)
	return missions
}

// Save persists a new mission to the history file with an auto-incrementing ID.
// The mission is appended to existing history. Returns the assigned ID.
func Save(m Mission) (int, error) {
	missions := Load()
	if missions == nil {
		missions = []Mission{}
	}

	// Assign next ID
	maxID := 0
	for _, existing := range missions {
		if existing.ID > maxID {
			maxID = existing.ID
		}
	}
	m.ID = maxID + 1

	missions = append(missions, m)
	if err := writeHistory(missions); err != nil {
		return 0, fmt.Errorf("saving mission: %w", err)
	}
	return m.ID, nil
}

// Update modifies an existing mission in the history file (matched by ID).
func Update(m Mission) error {
	missions := Load()
	if missions == nil {
		return fmt.Errorf("history file not found")
	}

	found := false
	for i := range missions {
		if missions[i].ID == m.ID {
			missions[i] = m
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("mission #%d not found", m.ID)
	}

	return writeHistory(missions)
}

// GetByID returns a mission by its ID, or nil if not found.
func GetByID(id int) *Mission {
	missions := Load()
	for i := range missions {
		if missions[i].ID == id {
			return &missions[i]
		}
	}
	return nil
}

// List returns the most recent N missions (or all if limit <= 0).
func List(limit int) []Mission {
	missions := Load()
	if missions == nil {
		return nil
	}

	if limit <= 0 || limit >= len(missions) {
		return missions
	}

	// Return the last N missions
	return missions[len(missions)-limit:]
}

// Export formats a single mission for display.
func Export(id int) string {
	all := Load()
	for _, m := range all {
		if m.ID == id {
			return formatMission(m)
		}
	}
	return fmt.Sprintf("Mission #%d not found.", id)
}

// writeHistory atomically writes the missions slice to the history file.
// Uses write-to-temp-then-rename for crash safety.
func writeHistory(missions []Mission) error {
	dir := filepath.Dir(historyPath)
	// SECURITY: Directory created with 0700 (owner-only access).
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("creating history directory: %w", err)
	}

	data, err := json.MarshalIndent(missions, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling history: %w", err)
	}

	// Atomic write: write to temp file, then rename
	tmpFile := historyPath + ".tmp"
	// SECURITY: History file created with 0600 (owner-only read/write).
	if err := os.WriteFile(tmpFile, data, 0o600); err != nil {
		return fmt.Errorf("writing temp history file: %w", err)
	}

	if err := os.Rename(tmpFile, historyPath); err != nil {
		os.Remove(tmpFile) // clean up on failure
		return fmt.Errorf("renaming history file: %w", err)
	}

	return nil
}

func formatMission(m Mission) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("# Mission #%d: %s\n\n", m.ID, m.Task))
	b.WriteString(fmt.Sprintf("- **Started:** %s\n", m.StartTime.Format("2006-01-02 15:04:05")))
	b.WriteString(fmt.Sprintf("- **Duration:** %s\n", m.Duration))
	b.WriteString(fmt.Sprintf("- **Outcome:** %s\n\n", m.Outcome))
	b.WriteString("## Phases\n\n")
	for _, p := range m.Phases {
		b.WriteString(fmt.Sprintf("| %s | %s | %s |\n", p.Name, p.Status, p.Duration))
	}
	return b.String()
}
