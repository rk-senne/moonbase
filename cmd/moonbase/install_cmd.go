package main

import (
	"github.com/spf13/cobra"
)

var (
	installAll    bool
	installGlobal bool
)

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install agents to .kiro/agents/",
	Long:  "Copy moonbase agent .md files into a project's .kiro/agents/ directory.\n\nUse --all to install all agents (default behavior).\nUse --global to install to ~/.kiro/agents/ for global kiro-cli access.",
	Run: func(cmd *cobra.Command, args []string) {
		runInstall()
	},
}

func init() {
	installCmd.Flags().BoolVarP(&installAll, "all", "a", false, "install all agents")
	installCmd.Flags().BoolVarP(&installGlobal, "global", "g", false, "install to ~/.kiro/agents/ for global kiro-cli access")
}
