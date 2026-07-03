package main

import (
	"github.com/spf13/cobra"
)

var lintCmd = &cobra.Command{
	Use:   "lint",
	Short: "Validate all agent .md files",
	Run: func(cmd *cobra.Command, args []string) {
		runLint()
	},
}
