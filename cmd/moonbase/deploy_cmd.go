package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/rk-senne/moonbase/internal/agents"
	"github.com/spf13/cobra"
)

var deployTask string
var deployCmux bool
var deployNative bool

var deployCmd = &cobra.Command{
	Use:     "deploy [numbuh] [task...]",
	Aliases: []string{"d", "run"},
	Short:   "Deploy operative by numbuh",
	Long:  "Deploy a KND operative to an interactive AI session.\nIf no numbuh is provided, an interactive picker is shown.\n\nExamples:\n  moonbase deploy 4\n  moonbase deploy 1 \"analyze the user auth flow\"\n  moonbase d 3                          (alias)\n  moonbase deploy -t \"check security\" 274\n  moonbase deploy                       (interactive picker)",
	Args:  cobra.ArbitraryArgs,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			// Interactive picker when no args provided
			if !isTerminal() {
				fmt.Fprintln(os.Stderr, "❌ No operative specified and stdin is not a terminal.")
				fmt.Fprintln(os.Stderr, "")
				fmt.Fprintln(os.Stderr, "   Usage: moonbase deploy <numbuh> [task...]")
				fmt.Fprintln(os.Stderr, "   Try:   moonbase deploy 1 \"your task\"")
				fmt.Fprintln(os.Stderr, "   Or:    moonbase list   (to see available operatives)")
				osExit(1)
			}
			numbuh := selectAgent()
			if numbuh == "" {
				return // cancelled
			}
			runDeploy(numbuh, "")
		} else {
			// args[0] is numbuh, args[1:] is the task
			task := ""
			if len(args) > 1 {
				task = strings.Join(args[1:], " ")
			}
			runDeploy(args[0], task)
		}
	},
}

func init() {
	deployCmd.Flags().StringVarP(&deployTask, "task", "t", "", "task description for the operative")
	deployCmd.Flags().BoolVar(&deployCmux, "cmux", false, "deploy agent in a cmux split pane")
	deployCmd.Flags().BoolVar(&deployNative, "native", false, "deploy via Kiro-native compiled JSON (kiro-cli chat --agent)")
}

// selectAgent shows an interactive huh select form to pick an agent.
func selectAgent() string {
	dir := agentsDir()
	reg := agents.NewRegistry(dir)
	reg.Reload()
	all := reg.All()

	if len(all) == 0 {
		fmt.Fprintln(os.Stderr, "❌ No agents found.")
		osExit(1)
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
