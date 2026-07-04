package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/f5508037/moonbase/internal/history"
	"github.com/spf13/cobra"
)

var replayDryRun bool

var replayCmd = &cobra.Command{
	Use:     "replay <mission-id>",
	Aliases: []string{"re"},
	Short:   "Re-run a past mission",
	Long:  "Load a mission from history and re-execute it with the original task string.\n\nExamples:\n  moonbase replay 3\n  moonbase replay 3 --dry-run\n  moonbase re 3                          (alias)",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		runReplay(args[0])
	},
}

func init() {
	replayCmd.Flags().BoolVar(&replayDryRun, "dry-run", false, "show what would be replayed without executing")
}

func runReplay(idStr string) {
	id, err := strconv.Atoi(idStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Invalid mission ID: %s\n", idStr)
		fmt.Fprintf(os.Stderr, "   Must be a number. Run 'moonbase history' to see mission IDs.\n")
		osExit(1)
	}

	mission := history.GetByID(id)
	if mission == nil {
		fmt.Fprintf(os.Stderr, "❌ Mission #%d not found.\n", id)
		fmt.Fprintf(os.Stderr, "   Run 'moonbase history' to see available missions.\n")
		osExit(1)
	}

	fmt.Printf("🌙 Replaying mission #%d: %s\n\n", mission.ID, mission.Task)

	if replayDryRun {
		fmt.Printf("   Task:     %s\n", mission.Task)
		fmt.Printf("   Original: %s\n", mission.StartTime.Format("2006-01-02 15:04:05"))
		fmt.Printf("   Outcome:  %s\n", mission.Outcome)
		fmt.Printf("   Duration: %s\n", mission.Duration)
		fmt.Println()
		fmt.Println("   (dry-run: would re-execute this mission)")
		return
	}

	// Re-run the mission with the original task string
	runMissionWithoutConfirm(mission.Task)
}
