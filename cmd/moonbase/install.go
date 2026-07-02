package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// runInstall implements `moonbase install` — copies agent .md files into a project's .kiro/agents/ dir.
// If run with --all flag (os.Args), installs all agents.
// By default installs to .kiro/agents/ in the current working directory.
// If --global flag is used, installs to ~/.kiro/agents/ (for global kiro-cli access).
func runInstall() {
	// Parse flags
	installAll := false
	global := false
	for _, arg := range os.Args[2:] {
		switch arg {
		case "--all", "-a":
			installAll = true
		case "--global", "-g":
			global = true
		}
	}

	// Find the moonbase agents directory
	agentsSource, err := findAgentsSource()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// List available agents
	files, err := filepath.Glob(filepath.Join(agentsSource, "*.md"))
	if err != nil || len(files) == 0 {
		fmt.Fprintf(os.Stderr, "No agent .md files found in %s\n", agentsSource)
		os.Exit(1)
	}

	// Determine target directory
	var targetDir string
	if global {
		home, _ := os.UserHomeDir()
		targetDir = filepath.Join(home, ".kiro", "agents")
	} else {
		cwd, _ := os.Getwd()
		targetDir = filepath.Join(cwd, ".kiro", "agents")
	}

	// Create target directory
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating %s: %v\n", targetDir, err)
		os.Exit(1)
	}

	// Select agents to install
	var toInstall []string
	if installAll {
		toInstall = files
	} else {
		// In non-interactive mode, install all with a listing
		fmt.Println("🌙 Moonbase Agent Installation")
		fmt.Println()
		for _, f := range files {
			name := strings.TrimSuffix(filepath.Base(f), ".md")
			fmt.Printf("  → %s\n", name)
		}
		fmt.Println()
		toInstall = files
	}

	if len(toInstall) == 0 {
		fmt.Println("No agents selected.")
		return
	}

	// Copy agents
	fmt.Printf("Installing to %s\n\n", targetDir)
	installed := 0
	for _, src := range toInstall {
		base := filepath.Base(src)
		dst := filepath.Join(targetDir, base)

		if err := copyFile(src, dst); err != nil {
			fmt.Fprintf(os.Stderr, "  ⚠️  %s: %v\n", base, err)
			continue
		}
		fmt.Printf("  ✅ %s\n", base)
		installed++
	}

	fmt.Printf("\n🌙 %d agent(s) installed to %s\n", installed, targetDir)
	if !global {
		fmt.Println("   Agents are now available to kiro-cli in this project.")
	}
}

// findAgentsSource locates the moonbase agents directory.
func findAgentsSource() (string, error) {
	// 1. Check relative to executable
	exe, err := os.Executable()
	if err == nil {
		exeDir := filepath.Dir(exe)
		candidates := []string{
			filepath.Join(exeDir, "..", "agents"),
			filepath.Join(exeDir, "agents"),
		}
		for _, c := range candidates {
			if isAgentsDir(c) {
				abs, _ := filepath.Abs(c)
				return abs, nil
			}
		}
	}

	// 2. Check relative to CWD (for development)
	cwd, _ := os.Getwd()
	candidates := []string{
		filepath.Join(cwd, "agents"),
		filepath.Join(cwd, "..", "agents"),
	}
	for _, c := range candidates {
		if isAgentsDir(c) {
			abs, _ := filepath.Abs(c)
			return abs, nil
		}
	}

	// 3. Check common install paths
	home, _ := os.UserHomeDir()
	if home != "" {
		commonPaths := []string{
			filepath.Join(home, ".moonbase", "agents"),
			filepath.Join(home, ".config", "moonbase", "agents"),
			"/usr/local/share/moonbase/agents",
		}
		for _, c := range commonPaths {
			if isAgentsDir(c) {
				return c, nil
			}
		}
	}

	return "", fmt.Errorf("agents directory not found — run from moonbase project root or install moonbase first")
}

func isAgentsDir(path string) bool {
	files, err := filepath.Glob(filepath.Join(path, "*.md"))
	return err == nil && len(files) > 0
}

func copyFile(src, dst string) error {
	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	dest, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dest.Close()

	_, err = io.Copy(dest, source)
	return err
}
