package main

import (
	"fmt"
	"os"

	"github.com/charmbracelet/huh"
	"github.com/f5508037/moonbase/internal/agents"
	"github.com/spf13/cobra"
)

var deployCmd = &cobra.Command{
	Use:   "deploy [numbuh] [task...]",
	Short: "Deploy operative by numbuh",
	Long:  "Deploy a KND operative to an interactive AI session.\nIf no numbuh is provided, an interactive picker is shown.\n\nExamples:\n  moonbase deploy 4\n  moonbase deploy 1 \"analyze the user auth flow\"\n  moonbase deploy  (interactive picker)",
	Args:  cobra.ArbitraryArgs,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			// Interactive picker when no args provided
			if !isTerminal() {
				fmt.Fprintln(os.Stderr, "❌ No operative specified and stdin is not a terminal.")
				os.Exit(1)
			}
			numbuh := selectAgent()
			if numbuh == "" {
				return // cancelled
			}
			runDeploy(numbuh)
		} else {
			runDeploy(args[0])
		}
	},
}

// selectAgent shows an interactive huh select form to pick an agent.
func selectAgent() string {
	dir := agentsDir()
	reg := agents.NewRegistry(dir)
	reg.Reload()
	all := reg.All()

	if len(all) == 0 {
		fmt.Fprintln(os.Stderr, "❌ No agents found.")
		os.Exit(1)
	}

	// Build options from loaded agents
	options := make([]huh.Option[string], 0, len(all))
	for _, a := range all {
		label := fmt.Sprintf("[%s] %s — %s", extractNumbuh(a.Name), a.Designation, a.Role)
		options = append(options, huh.NewOption(label, extractNumbuh(a.Name)))
	}

	var selected string
	err := huh.NewSelect[string]().
		Title("Select operative to deploy").
		Options(options...).
		Value(&selected).
		Run()

	if err != nil {
		// ESC/Ctrl+C cancels
		return ""
	}

	return selected
}
