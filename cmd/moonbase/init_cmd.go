package main

import "github.com/spf13/cobra"

// initWithDataAccess, when true, makes 'moonbase init' also generate the opt-in
// .kiro/steering/data-access-performance.md standard. It is set by the
// --data-access flag. The file is always gitignored regardless of this flag, so
// it stays local to the projects that need it.
var initWithDataAccess bool

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Scaffold .kiro/ in any project",
	Long: "Initialize a project for agent-driven development by creating .kiro/ with specs, steering, and agents.\n\n" +
		"Examples:\n" +
		"  cd my-project && moonbase init\n" +
		"  moonbase init --data-access   # also add the data-access performance standard",
	Run: func(cmd *cobra.Command, args []string) {
		runInit()
	},
}

func init() {
	initCmd.Flags().BoolVar(&initWithDataAccess, "data-access", false,
		"also generate .kiro/steering/data-access-performance.md (for projects with a data layer)")
}
