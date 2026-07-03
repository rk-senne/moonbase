package main

import (
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show environment health check",
	Run: func(cmd *cobra.Command, args []string) {
		runStatus()
	},
}
