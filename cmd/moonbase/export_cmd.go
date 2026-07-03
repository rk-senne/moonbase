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
		id, _ := strconv.Atoi(args[0])
		fmt.Println(history.Export(id))
	},
}
