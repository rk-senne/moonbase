package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print moonbase version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("moonbase %s (commit: %s, built: %s)\n", version, commit, date)
	},
}
