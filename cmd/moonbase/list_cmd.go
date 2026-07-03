package main

import (
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "Show operative roster",
	Run: func(cmd *cobra.Command, args []string) {
		runList()
	},
}
