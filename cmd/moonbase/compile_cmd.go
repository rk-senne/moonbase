package main

import (
	"github.com/spf13/cobra"
)

var (
	compileOut      string
	compileValidate bool
	compileAgent    string
)

var compileCmd = &cobra.Command{
	Use:   "compile",
	Short: "Compile agents to Kiro-native JSON",
	Long: `Translate moonbase .md agent definitions into Kiro-native agent JSON.

Emits <name>.json + <name>.prompt.md per agent into the target directory.
The JSON passes kiro-cli agent validate and can be deployed via kiro-cli chat --agent.

Examples:
  moonbase compile                          # compile all 14 agents to .kiro/agents/
  moonbase compile --out /tmp/agents        # compile to custom directory
  moonbase compile --validate               # compile + run kiro-cli agent validate
  moonbase compile --agent numbuh-4         # compile a single agent`,
	Run: func(cmd *cobra.Command, args []string) {
		runCompile()
	},
}

func init() {
	compileCmd.Flags().StringVarP(&compileOut, "out", "o", ".kiro/agents", "target directory for compiled agents")
	compileCmd.Flags().BoolVar(&compileValidate, "validate", false, "run kiro-cli agent validate on each emitted file")
	compileCmd.Flags().StringVar(&compileAgent, "agent", "", "compile a single agent by name (default: all)")
}
