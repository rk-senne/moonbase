package main

import (
	"github.com/spf13/cobra"
)

var lintCmd = &cobra.Command{
	Use:     "lint",
	Aliases: []string{"validate"},
	Short:   "Validate all agent .md files",
	Run: func(cmd *cobra.Command, args []string) {
		runLint()
	},
}
