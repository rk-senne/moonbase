package main

import "github.com/spf13/cobra"

var setupCmd = &cobra.Command{
	Use:     "setup",
	Short:   "Install agents globally so moonbase works from any project",
	Long:    "Copies agent .md files to ~/.moonbase/agents/ so moonbase can be used from any directory without per-project installation.",
	Aliases: []string{"global-install"},
	Run: func(cmd *cobra.Command, args []string) {
		runSetup()
	},
}
