package history

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Mission struct {
	ID        int       `json:"id"`
	Task      string    `json:"task"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time,omitempty"`
	Duration  string    `json:"duration,omitempty"`
	Phases    []Phase   `json:"phases"`
	Outcome   string    `json:"outcome"` // "complete", "aborted", "in-progress"
}

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

func Load() []Mission {
	data, err := os.ReadFile(historyPath)
	if err != nil {
		return nil
	}
	var missions []Mission
	json.Unmarshal(data, &missions)
	return missions
}

func save(missions []Mission) error {
	os.MkdirAll(filepath.Dir(historyPath), 0755)
	data, err := json.MarshalIndent(missions, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(historyPath, data, 0644)
}

func LogMission(task string, phases []Phase, outcome string, duration time.Duration) {
	all := Load()
	id := len(all) + 1
	m := Mission{
		ID:        id,
		Task:      task,
		StartTime: time.Now().Add(-duration),
		EndTime:   time.Now(),
		Duration:  duration.Round(time.Second).String(),
		Phases:    phases,
		Outcome:   outcome,
	}
	all = append(all, m)
	save(all)
}

func Export(id int) string {
	all := Load()
	for _, m := range all {
		if m.ID == id {
			return formatMission(m)
		}
	}
	return fmt.Sprintf("Mission #%d not found.", id)
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
