package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// runSetup installs agents globally to ~/.moonbase/agents/ so that moonbase
// can be used from any project directory without per-project installation.
func runSetup() {
	fmt.Println("🌙 Moonbase Global Setup")
	fmt.Println()

	// Find the source agents directory
	agentsSource, err := findAgentsSource()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ %v\n", err)
		fmt.Fprintf(os.Stderr, "\n   Run this from the moonbase source directory, or ensure\n")
		fmt.Fprintf(os.Stderr, "   agents are bundled with the binary.\n")
		osExit(1)
	}

	// List available agents
	files, err := filepath.Glob(filepath.Join(agentsSource, "*.md"))
	if err != nil || len(files) == 0 {
		fmt.Fprintf(os.Stderr, "❌ No agent .md files found in %s\n", agentsSource)
		osExit(1)
	}

	// Target: ~/.moonbase/agents/
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Cannot determine home directory: %v\n", err)
		osExit(1)
	}
	targetDir := filepath.Join(home, ".moonbase", "agents")

	// Create target directory
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Cannot create %s: %v\n", targetDir, err)
		osExit(1)
	}

	fmt.Printf("   Source: %s\n", agentsSource)
	fmt.Printf("   Target: %s\n", targetDir)
	fmt.Println()

	// Copy all agents
	installed := 0
	for _, src := range files {
		base := filepath.Base(src)

		// Safety check
		if strings.Contains(base, "..") || strings.ContainsAny(base, `/\`) {
			fmt.Fprintf(os.Stderr, "  ⚠️  %s: suspicious filename, skipping\n", base)
			continue
		}

		dst := filepath.Join(targetDir, base)
		if err := copyFile(src, dst); err != nil {
			fmt.Fprintf(os.Stderr, "  ⚠️  %s: %v\n", base, err)
			continue
		}
		name := strings.TrimSuffix(base, ".md")
		fmt.Printf("  ✅ %s\n", name)
		installed++
	}

	fmt.Println()
	fmt.Printf("🌙 %d agent(s) installed globally to %s\n", installed, targetDir)
	fmt.Println()
	fmt.Println("   You can now run moonbase from any project directory:")
	fmt.Println("     moonbase mission \"your task\"")
	fmt.Println("     moonbase deploy 1 \"analyze auth flow\"")
}
