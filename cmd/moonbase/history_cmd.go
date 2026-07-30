package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/rk-senne/moonbase/internal/history"
	"github.com/spf13/cobra"
)

var historyAll bool
var historyJSON bool
var historyLimit int

var historyCmd = &cobra.Command{
	Use:     "history",
	Aliases: []string{"h", "log"},
	Short:   "Show past missions",
	Long:  "List past mission executions with ID, date, task, outcome, and duration.\n\nExamples:\n  moonbase history\n  moonbase h                             (alias)\n  moonbase history --all\n  moonbase history --limit 5\n  moonbase history --json",
	Run: func(cmd *cobra.Command, args []string) {
		runHistory()
	},
}

func init() {
	historyCmd.Flags().BoolVar(&historyAll, "all", false, "show all missions (not just last 20)")
	historyCmd.Flags().BoolVar(&historyJSON, "json", false, "output raw JSON for scripting")
	historyCmd.Flags().IntVar(&historyLimit, "limit", 20, "number of recent missions to show")
}

func runHistory() {
	limit := historyLimit
	if historyAll {
		limit = 0
	}

	missions := history.List(limit)
	if len(missions) == 0 {
		fmt.Println("No mission history found.")
		return
	}

	if historyJSON {
		data, err := json.MarshalIndent(missions, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Failed to marshal history: %v\n", err)
			osExit(1)
		}
		fmt.Println(string(data))
		return
	}

	// Table format
	fmt.Println("🌙 Mission History")
	fmt.Println("═══════════════════════════════════════════════════════════════════════════════")
	fmt.Printf("  %-4s  %-12s  %-40s  %-12s  %s\n", "ID", "Date", "Task", "Outcome", "Duration")
	fmt.Println("  ────  ────────────  ────────────────────────────────────────────  ──────────  ────────")

	for _, m := range missions {
		task := m.Task
		if len(task) > 40 {
			task = task[:37] + "..."
		}
		date := m.StartTime.Format("2006-01-02")
		fmt.Printf("  %-4d  %-12s  %-40s  %-12s  %s\n", m.ID, date, task, m.Outcome, m.Duration)
	}

	fmt.Printf("\n  Showing %d mission(s)\n", len(missions))
}
