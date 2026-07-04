package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
)

var missionDryRun bool

var missionCmd = &cobra.Command{
	Use:     "mission <task description>",
	Aliases: []string{"m", "go"},
	Short:   "Run full KND Council pipeline on a task",
	Long:  "Execute the full KND Council pipeline: Analysis → Architecture → Implementation → QA → Review.\n\nExamples:\n  moonbase mission \"add rate limiting to the API\"\n  moonbase m \"fix the auth bug\"          (alias)\n  moonbase mission --dry-run \"add pagination\"",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		task := strings.Join(args, " ")
		if missionDryRun {
			runMissionDryRun(task)
		} else {
			// Show confirmation before executing (only if terminal)
			if isTerminal() {
				if !confirmMission(task) {
					fmt.Println("Mission aborted.")
					return
				}
			}
			runMission(task)
		}
	},
}

func init() {
	missionCmd.Flags().BoolVar(&missionDryRun, "dry-run", false, "print execution plan without invoking backends")
}

// confirmMission shows a huh confirmation dialog before executing a mission.
func confirmMission(task string) bool {
	var confirm bool
	err := huh.NewConfirm().
		Title("Deploy KND Council on this mission?").
		Description(task).
		Affirmative("Deploy").
		Negative("Abort").
		Value(&confirm).
		Run()

	if err != nil {
		// ESC/Ctrl+C cancels
		return false
	}
	return confirm
}

// runMissionWithoutConfirm executes a mission without showing the confirmation dialog.
func runMissionWithoutConfirm(task string) {
	runMission(task)
}
