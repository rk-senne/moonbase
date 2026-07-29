package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var snippetCmd = &cobra.Command{
	Use:   "snippet",
	Short: "Manage saved prompt snippets",
	Long:  "Save and list reusable prompt snippets.\n\nExamples:\n  moonbase snippet save my-prompt\n  moonbase snippet list",
}

var snippetSaveCmd = &cobra.Command{
	Use:   "save <name>",
	Short: "Save a snippet (reads content from stdin)",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]

		// Validate snippet name
		if len(name) > 100 {
			fmt.Fprintf(os.Stderr, "❌ Snippet name too long (%d chars, max 100)\n", len(name))
			osExit(1)
		}
		if strings.ContainsAny(name, "/\\") {
			fmt.Fprintf(os.Stderr, "❌ Snippet name must not contain path separators (/ or \\)\n")
			osExit(1)
		}
		for _, r := range name {
			if r < 32 {
				fmt.Fprintf(os.Stderr, "❌ Snippet name must not contain control characters\n")
				osExit(1)
			}
		}

		scanner := bufio.NewScanner(os.Stdin)
		var lines []string
		for scanner.Scan() {
			lines = append(lines, scanner.Text())
		}
		content := strings.Join(lines, "\n")
		home := mustUserHomeDir()
		path := filepath.Join(home, ".config", "moonbase", "snippets.json")
		os.MkdirAll(filepath.Dir(path), 0700)

		// Load existing
		var existing []map[string]string
		if data, err := os.ReadFile(path); err == nil {
			json.Unmarshal(data, &existing)
		}
		existing = append(existing, map[string]string{"name": name, "content": content})
		data, _ := json.MarshalIndent(existing, "", "  ")
		os.WriteFile(path, data, 0600)
		fmt.Printf("✓ Snippet saved: %s\n", name)
	},
}

var snippetListCmd = &cobra.Command{
	Use:   "list",
	Short: "List saved snippets",
	Run: func(cmd *cobra.Command, args []string) {
		home := mustUserHomeDir()
		path := filepath.Join(home, ".config", "moonbase", "snippets.json")
		data, err := os.ReadFile(path)
		if err != nil {
			fmt.Println("No snippets saved yet.")
			return
		}
		var snippets []map[string]string
		if err := json.Unmarshal(data, &snippets); err != nil {
			fmt.Println("No snippets saved yet.")
			return
		}
		if len(snippets) == 0 {
			fmt.Println("No snippets saved yet.")
			return
		}
		fmt.Printf("%-4s  %-20s  %s\n", "#", "NAME", "PREVIEW")
		fmt.Printf("%-4s  %-20s  %s\n", "──", "────────────────────", "───────────────────────────────────")
		for i, s := range snippets {
			name := s["name"]
			content := s["content"]
			// Truncate preview
			preview := strings.ReplaceAll(content, "\n", " ")
			if len(preview) > 40 {
				preview = preview[:40] + "..."
			}
			fmt.Printf("%-4d  %-20s  %s\n", i+1, name, preview)
		}
		fmt.Printf("\n%d snippet(s) saved.\n", len(snippets))
	},
}

func init() {
	snippetCmd.AddCommand(snippetSaveCmd)
	snippetCmd.AddCommand(snippetListCmd)
}
