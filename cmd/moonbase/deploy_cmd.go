package main

import (
	"github.com/spf13/cobra"
)

var deployCmd = &cobra.Command{
	Use:   "deploy <numbuh> [task...]",
	Short: "Deploy operative by numbuh",
	Long:  "Deploy a KND operative to an interactive AI session.\n\nExamples:\n  moonbase deploy 4\n  moonbase deploy 1 \"analyze the user auth flow\"",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		runDeploy(args[0])
	},
}
