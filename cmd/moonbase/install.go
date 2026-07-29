package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// runInstall implements `moonbase install` — copies agent .md files into a target directory.
// Uses cobra-registered flags: installAll and installGlobal.
func runInstall() {
	// Determine target directory
	var targetDir string
	if installGlobal {
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Cannot determine home directory: %v\n", err)
			osExit(1)
		}
		targetDir = filepath.Join(home, ".kiro", "agents")
	} else {
		cwd, err := os.Getwd()
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Cannot determine working directory: %v\n", err)
			osExit(1)
		}
		targetDir = filepath.Join(cwd, ".kiro", "agents")
	}

	installAgentsTo(targetDir, installGlobal)
}

// runSetup installs agents globally to ~/.moonbase/agents/ so that moonbase
// can be used from any project directory without per-project installation.
func runSetup() {
	fmt.Println("🌙 Moonbase Global Setup")
	fmt.Println()

	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Cannot determine home directory: %v\n", err)
		osExit(1)
	}
	targetDir := filepath.Join(home, ".moonbase", "agents")
	installAgentsTo(targetDir, true)

	fmt.Println()
	fmt.Println("   You can now run moonbase from any project directory:")
	fmt.Println("     moonbase mission \"your task\"")
	fmt.Println("     moonbase deploy 1 \"analyze auth flow\"")
}

// installAgentsTo copies agent .md files from the source directory to targetDir.
// This is the shared implementation for both `install` and `setup` commands.
func installAgentsTo(targetDir string, global bool) {
	// Find the moonbase agents directory
	agentsSource, err := findAgentsSource()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ %v\n", err)
		if global {
			fmt.Fprintf(os.Stderr, "\n   Run this from the moonbase source directory, or ensure\n")
			fmt.Fprintf(os.Stderr, "   agents are bundled with the binary.\n")
		}
		osExit(1)
	}

	// List available agents
	files, err := filepath.Glob(filepath.Join(agentsSource, "*.md"))
	if err != nil || len(files) == 0 {
		fmt.Fprintf(os.Stderr, "❌ No agent .md files found in %s\n", agentsSource)
		osExit(1)
	}

	// Guard against source == target. This happens when 'moonbase setup' is run
	// outside the repo: findAgentsSource() falls back to ~/.moonbase/agents, which
	// is also the setup target. Copying a directory onto itself would truncate
	// every agent to 0 bytes. Nothing fresh to copy — tell the user where to run.
	if srcAbs, e1 := filepath.Abs(agentsSource); e1 == nil {
		if dstAbs, e2 := filepath.Abs(targetDir); e2 == nil && srcAbs == dstAbs {
			fmt.Printf("   Agents are already installed at %s\n", targetDir)
			fmt.Println("   (source and target are the same directory — nothing to copy)")
			fmt.Println("   To refresh from source, run this from the moonbase repository.")
			return
		}
	}

	// SECURITY: Create target directory with restrictive permissions (0755).
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

		// SECURITY: Validate that the source filename is safe (no path traversal).
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

	return "", fmt.Errorf("agents directory not found — run from moonbase project root or run 'moonbase init' first")
}

func isAgentsDir(path string) bool {
	files, err := filepath.Glob(filepath.Join(path, "*.md"))
	return err == nil && len(files) > 0
}

// copyFile copies src to dst with explicit file permissions.
// SECURITY: Destination is created with 0644 (owner rw, others read-only).
func copyFile(src, dst string) error {
	// Guard against copying a file onto itself. copyFile opens the destination
	// with O_TRUNC, so a self-copy would truncate the file to 0 bytes before the
	// source is read. Skip silently — the file already has the intended content.
	if srcAbs, err1 := filepath.Abs(src); err1 == nil {
		if dstAbs, err2 := filepath.Abs(dst); err2 == nil && srcAbs == dstAbs {
			return nil
		}
	}

	source, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("copying %s to %s: %w", src, dst, err)
	}
	defer source.Close()

	dest, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return fmt.Errorf("copying %s to %s: %w", src, dst, err)
	}
	defer dest.Close()

	if _, err = io.Copy(dest, source); err != nil {
		return fmt.Errorf("copying %s to %s: %w", src, dst, err)
	}
	return nil
}
