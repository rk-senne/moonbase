package main

import "github.com/spf13/cobra"

var guideCmd = &cobra.Command{
	Use:   "guide [agent]",
	Short: "Show usage guide for agents or the full operations manual",
	Long: `Show how to use moonbase agents. Without arguments, shows the general
operations overview. With an agent name or number, shows that agent's
specific guide.

Examples:
  moonbase guide              Show general operations manual
  moonbase guide 1            Show Numbuh 1's usage guide
  moonbase guide 274          Show Numbuh 274's usage guide
  moonbase guide z            Show Sector Z's usage guide
  moonbase guide council      Show KND Council's usage guide
  moonbase guide --all        Show all agents with usage examples`,
	Aliases: []string{"man", "howto"},
	Run: func(cmd *cobra.Command, args []string) {
		all, _ := cmd.Flags().GetBool("all")
		if all {
			runGuideAll()
		} else if len(args) > 0 {
			runGuideAgent(args[0])
		} else {
			runGuideOverview()
		}
	},
}

func init() {
	guideCmd.Flags().Bool("all", false, "show guide for all agents")
}
