package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/f5508037/moonbase/internal/backend"
	clip "github.com/f5508037/moonbase/internal/clipboard"
	"github.com/f5508037/moonbase/internal/logging"
	"github.com/f5508037/moonbase/internal/tui"
	"github.com/spf13/cobra"
)

// Build-time variables set via ldflags.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

// debug enables verbose output when --debug is passed.
var debug bool

var rootCmd = &cobra.Command{
	Use:   "moonbase",
	Short: "KND Tactical Operations Terminal",
	Long:  "🌙 Moonbase — KND Tactical Operations Terminal\n\nA 14-agent AI development pipeline with spec-driven methodology.",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		logging.Init(debug)
	},
	// When called with no subcommand: pipe mode or TUI.
	RunE: func(cmd *cobra.Command, args []string) error {
		// Pipe mode: if stdin is not a TTY, read it and deploy.
		// SECURITY: Input is size-limited to prevent OOM from large/infinite pipes.
		// The kiro-cli subprocess uses SafeEnv() to avoid leaking env vars.
		if !isTerminal() {
			limited := io.LimitReader(os.Stdin, maxPipeInputSize)
			input, _ := io.ReadAll(limited)
			task := strings.TrimSpace(string(input))
			if task == "" {
				return nil
			}
			if len(input) >= maxPipeInputSize {
				fmt.Fprintf(os.Stderr, "⚠️  Pipe input truncated at %d bytes\n", maxPipeInputSize)
			}
			fmt.Printf("🌙 Pipe mode — task: %s\n", task)
			fmt.Println("Deploy to kiro-cli with knd-council...")
			if kiro, err := exec.LookPath("kiro-cli"); err == nil {
				execCmd := exec.Command(kiro, "chat", "--agent", "knd-council")
				execCmd.Stdin = strings.NewReader(task)
				execCmd.Stdout = os.Stdout
				execCmd.Stderr = os.Stderr
				// SECURITY: Use SafeEnv to prevent leaking sensitive env vars to subprocess
				execCmd.Env = backend.SafeEnv()
				execCmd.Run()
			} else {
				// Copy task to clipboard
				clip.Copy(task)
				fmt.Println("✓ Task copied to clipboard")
			}
			return nil
		}

		// Default: launch TUI
		p := tea.NewProgram(tui.NewApp(), tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			return fmt.Errorf("TUI error: %w", err)
		}
		return nil
	},
	// Silence cobra's default error/usage printing — we handle it ourselves.
	SilenceErrors: true,
	SilenceUsage:  true,
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "enable debug output")

	// Register all subcommands
	rootCmd.AddCommand(deployCmd)
	rootCmd.AddCommand(missionCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(lintCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(snippetCmd)
	rootCmd.AddCommand(installCmd)
	rootCmd.AddCommand(exportCmd)
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(historyCmd)
	rootCmd.AddCommand(replayCmd)
}
