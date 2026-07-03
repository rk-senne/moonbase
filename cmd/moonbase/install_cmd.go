package main

import (
	"github.com/spf13/cobra"
)

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install agents to .kiro/agents/",
	Long:  "Copy moonbase agent .md files into a project's .kiro/agents/ directory.\n\nFlags:\n  --all     Install all agents\n  --global  Install to ~/.kiro/agents/ for global kiro-cli access",
	Run: func(cmd *cobra.Command, args []string) {
		runInstall()
	},
}
