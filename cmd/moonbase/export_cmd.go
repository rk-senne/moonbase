package main

import (
	"fmt"
	"strconv"

	"github.com/f5508037/moonbase/internal/history"
	"github.com/spf13/cobra"
)

var exportCmd = &cobra.Command{
	Use:   "export <mission-id>",
	Short: "Export a mission's history",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		id, err := strconv.Atoi(args[0])
		if err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "❌ Invalid mission ID: %q\n", args[0])
			fmt.Fprintf(cmd.ErrOrStderr(), "   Mission ID must be a number. Use 'moonbase history' to see available missions.\n")
			osExit(1)
		}
		fmt.Println(history.Export(id))
	},
}
