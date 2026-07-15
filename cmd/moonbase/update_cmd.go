package main

import "github.com/spf13/cobra"

var updateCheckOnly bool

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update moonbase to the latest release",
	Long: `Check for and install the latest moonbase release from GitHub.

Downloads the appropriate binary for your platform, verifies the SHA256
checksum, and atomically replaces the current binary.

Examples:
  moonbase update          Download and install the latest version
  moonbase update --check  Just check if an update is available`,
	Aliases: []string{"upgrade", "self-update"},
	Run: func(cmd *cobra.Command, args []string) {
		if updateCheckOnly {
			runUpdateCheck()
		} else {
			runUpdate()
		}
	},
}

func init() {
	updateCmd.Flags().BoolVar(&updateCheckOnly, "check", false, "only check for updates, don't install")
}
