package main

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	moonbase "github.com/rk-senne/moonbase"
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
	// Resolve the agent source. Prefer an on-disk repository checkout so local
	// edits are picked up during development. Fall back to the agents embedded in
	// the binary when there is no repo source, or when the resolved source is the
	// same directory as the target — the latter is the case for `moonbase setup`
	// run outside the repo, where copying a directory onto itself is both unsafe
	// and pointless.
	agentsSource, srcErr := findAgentsSource()
	var files []string
	if srcErr == nil && !sameDir(agentsSource, targetDir) {
		files, _ = filepath.Glob(filepath.Join(agentsSource, "*.md"))
	}
	if len(files) == 0 {
		installEmbeddedAgentsTo(targetDir, global)
		return
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

// sameDir reports whether two paths resolve to the same directory.
func sameDir(a, b string) bool {
	aa, e1 := filepath.Abs(a)
	bb, e2 := filepath.Abs(b)
	return e1 == nil && e2 == nil && aa == bb
}

// installEmbeddedAgentsTo writes the agents embedded in the binary to targetDir.
// This is the fallback used when no repository checkout is available on disk
// (e.g. `moonbase setup` run from an arbitrary directory).
func installEmbeddedAgentsTo(targetDir string, global bool) {
	fmt.Printf("   Source: (embedded in binary)\n")
	fmt.Printf("   Target: %s\n", targetDir)
	fmt.Println()

	installed, err := writeEmbeddedAgents(targetDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ %v\n", err)
		osExit(1)
	}
	if installed == 0 {
		fmt.Fprintf(os.Stderr, "❌ no embedded agents found in this binary\n")
		osExit(1)
	}

	fmt.Printf("🌙 %d agent(s) installed to %s (embedded)\n", installed, targetDir)
	if !global {
		fmt.Println("   Agents are now available to kiro-cli in this project.")
	}
}

// writeEmbeddedAgents writes every embedded agent .md file into targetDir with
// 0644 permissions and returns the count written. Each file is read fully into
// memory before writing, so it is safe even if targetDir already holds a
// same-named file (this structurally avoids the truncate-on-open self-copy bug).
func writeEmbeddedAgents(targetDir string) (int, error) {
	efs, err := moonbase.AgentsFS()
	if err != nil {
		return 0, fmt.Errorf("loading embedded agents: %w", err)
	}
	entries, err := fs.Glob(efs, "*.md")
	if err != nil {
		return 0, fmt.Errorf("listing embedded agents: %w", err)
	}
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return 0, fmt.Errorf("creating %s: %w", targetDir, err)
	}

	installed := 0
	for _, name := range entries {
		base := filepath.Base(name)
		// SECURITY: reject any unexpected path components (embedded names are
		// controlled, but keep the guard consistent with the filesystem path).
		if strings.Contains(base, "..") || strings.ContainsAny(base, `/\`) {
			continue
		}
		data, rerr := fs.ReadFile(efs, name)
		if rerr != nil {
			fmt.Fprintf(os.Stderr, "  ⚠️  %s: %v\n", base, rerr)
			continue
		}
		if werr := os.WriteFile(filepath.Join(targetDir, base), data, 0o644); werr != nil {
			fmt.Fprintf(os.Stderr, "  ⚠️  %s: %v\n", base, werr)
			continue
		}
		fmt.Printf("  ✅ %s\n", strings.TrimSuffix(base, ".md"))
		installed++
	}
	return installed, nil
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
	cwd := mustGetwd()
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
	home := mustUserHomeDir()
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
