package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
)

var missionDryRun bool
var missionFast bool
var missionTrace bool

var missionCmd = &cobra.Command{
	Use:     "mission [task description]",
	Aliases: []string{"m", "go"},
	Short:   "Run full KND Council pipeline on a task",
	Long:  "Execute the full KND Council pipeline: Analysis → Architecture → Implementation → QA → Review.\n\nThe task can be provided as arguments or piped via stdin.\n\nExamples:\n  moonbase mission \"add rate limiting to the API\"\n  moonbase m \"fix the auth bug\"          (alias)\n  moonbase mission --dry-run \"add pagination\"\n  moonbase mission --fast \"fix typo in utils.ts\"\n  echo \"fix auth\" | moonbase mission",
	Args:  cobra.ArbitraryArgs,
	Run: func(cmd *cobra.Command, args []string) {
		var task string
		if len(args) > 0 {
			task = strings.Join(args, " ")
		} else if !isTerminal() {
			// Support pipe mode: read task from stdin
			limited := io.LimitReader(os.Stdin, maxPipeInputSize)
			input, _ := io.ReadAll(limited)
			task = strings.TrimSpace(string(input))
		}

		if task == "" {
			fmt.Fprintln(os.Stderr, "❌ No task provided.")
			fmt.Fprintln(os.Stderr, "")
			fmt.Fprintln(os.Stderr, "   Usage: moonbase mission \"your task description\"")
			fmt.Fprintln(os.Stderr, "   Or:    echo \"your task\" | moonbase mission")
			osExit(1)
		}

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
			if missionFast {
				runMissionFast(task)
			} else {
				runMission(task)
			}
		}
	},
}

func init() {
	missionCmd.Flags().BoolVar(&missionDryRun, "dry-run", false, "print execution plan without invoking backends")
	missionCmd.Flags().BoolVar(&missionFast, "fast", false, "skip analysis/architecture, go straight to implementation + QA")
	missionCmd.Flags().BoolVar(&missionTrace, "trace", false, "output trace-level info (TraceID, phase timestamps, output sizes)")
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
