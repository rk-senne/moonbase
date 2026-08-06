// Package reboot determines how moonbase updates and reinstalls itself so the
// TUI can offer a "reboot" action: pull + rebuild from a local source checkout
// for development builds, or self-update from a GitHub release otherwise.
//
// SECURITY: the reinstall script is assembled from constants plus shell-quoted,
// validated filesystem paths — never from arbitrary user text. The action is
// always gated behind an explicit confirmation in the TUI.
package reboot

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// moduleMarker identifies a moonbase source checkout via its go.mod module path.
const moduleMarker = "github.com/rk-senne/moonbase"

// Strategy is how moonbase will reinstall itself.
type Strategy int

const (
	// StrategyNone means no automatic path is available (guide the user).
	StrategyNone Strategy = iota
	// StrategySource rebuilds from a local source checkout (git pull + go build).
	StrategySource
	// StrategyRelease self-updates from the latest GitHub release binary.
	StrategyRelease
)

// Plan describes how a reboot will reinstall moonbase.
type Plan struct {
	Strategy  Strategy
	SourceDir string // repo root, for StrategySource
	TargetBin string // installed binary path to replace and re-exec
	Reason    string // human-readable explanation
}

// isMoonbaseRepo reports whether dir is the root of a moonbase source checkout.
func isMoonbaseRepo(dir string) bool {
	if dir == "" {
		return false
	}
	data, err := os.ReadFile(filepath.Join(dir, "go.mod"))
	return err == nil && strings.Contains(string(data), moduleMarker)
}

// walkUpToRepo walks up from startPath (a file or dir) looking for a moonbase
// repo root, bounded to a few levels.
func walkUpToRepo(startPath string) (string, bool) {
	dir := startPath
	if info, err := os.Stat(startPath); err == nil && !info.IsDir() {
		dir = filepath.Dir(startPath)
	}
	for i := 0; i < 10; i++ {
		if isMoonbaseRepo(dir) {
			return dir, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", false
}

// FindSourceDir locates a moonbase source checkout, in order of preference:
// the explicitly configured dir, the MOONBASE_SRC env var, then the resolved
// location of the running executable (symlinks followed) walked up to a repo
// root — which makes a `make setup` symlink install self-updating.
func FindSourceDir(configDir, exePath string) (string, bool) {
	if isMoonbaseRepo(configDir) {
		return configDir, true
	}
	if env := os.Getenv("MOONBASE_SRC"); isMoonbaseRepo(env) {
		return env, true
	}
	if exePath != "" {
		resolved := exePath
		if r, err := filepath.EvalSymlinks(exePath); err == nil {
			resolved = r
		}
		if dir, ok := walkUpToRepo(resolved); ok {
			return dir, true
		}
	}
	return "", false
}

// SelectPlan chooses how to reboot/reinstall moonbase given the build version,
// configured source dir, and the running executable path.
func SelectPlan(version, configSourceDir, exePath string) Plan {
	if src, ok := FindSourceDir(configSourceDir, exePath); ok {
		return Plan{
			Strategy:  StrategySource,
			SourceDir: src,
			TargetBin: exePath,
			Reason:    "rebuild from source (" + src + ")",
		}
	}
	if version != "" && version != "dev" {
		return Plan{
			Strategy:  StrategyRelease,
			TargetBin: exePath,
			Reason:    "self-update from the latest GitHub release",
		}
	}
	return Plan{
		Strategy:  StrategyNone,
		TargetBin: exePath,
		Reason:    "no source checkout found and this is a dev build — set source_dir in config or the MOONBASE_SRC env var",
	}
}

// shellSingleQuote safely single-quotes s for POSIX sh.
func shellSingleQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// ReinstallScript returns a POSIX sh script that pulls the latest source,
// rebuilds moonbase, and installs it over targetBin. Paths are shell-quoted.
func ReinstallScript(sourceDir, targetBin string) string {
	return fmt.Sprintf("set -e; cd %s; "+
		"echo '⟳ Pulling latest source…'; git pull --ff-only; "+
		"echo '⚙️  Building moonbase…'; go build -o bin/moonbase ./cmd/moonbase; "+
		"echo '📦 Installing…'; cp bin/moonbase %s; "+
		"echo '✅ moonbase reinstalled — rebooting…'",
		shellSingleQuote(sourceDir), shellSingleQuote(targetBin))
}
