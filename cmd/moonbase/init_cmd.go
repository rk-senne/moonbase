package main

import (
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Scaffold .kiro/ in any project",
	Long:  "Initialize a project for agent-driven development by creating .kiro/ with specs, steering, and agents.",
	Run: func(cmd *cobra.Command, args []string) {
		runInit()
	},
}
