package main

import (
	"strings"

	"github.com/spf13/cobra"
)

var missionDryRun bool

var missionCmd = &cobra.Command{
	Use:   "mission <task description>",
	Short: "Run full KND Council pipeline on a task",
	Long:  "Execute the full KND Council pipeline: Analysis → Architecture → Implementation → QA → Review.\n\nExamples:\n  moonbase mission \"add rate limiting to the API\"\n  moonbase mission --dry-run \"add pagination\"",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		task := strings.Join(args, " ")
		if missionDryRun {
			runMissionDryRun(task)
		} else {
			runMission(task)
		}
	},
}

func init() {
	missionCmd.Flags().BoolVar(&missionDryRun, "dry-run", false, "print execution plan without invoking backends")
}
