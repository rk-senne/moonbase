package agents

import (
	"fmt"
	"os"
	"path/filepath"
)

// FindAgentsDir locates the agents directory by checking multiple candidate paths.
// If configPath is non-empty, it is checked first (allows user override via config).
// Returns the resolved path or an error if no agents directory can be found.
func FindAgentsDir(configPath string) (string, error) {
	// 0. Explicit config override
	if configPath != "" {
		if fi, err := os.Stat(configPath); err == nil && fi.IsDir() {
			return configPath, nil
		}
	}

	// 1. Check relative to executable
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Join(filepath.Dir(exe), "..", "agents")
		if fi, err := os.Stat(dir); err == nil && fi.IsDir() {
			return dir, nil
		}
		dir = filepath.Join(filepath.Dir(exe), "agents")
		if fi, err := os.Stat(dir); err == nil && fi.IsDir() {
			return dir, nil
		}
	}

	// 2. Check relative to CWD
	if cwd, err := os.Getwd(); err == nil {
		dir := filepath.Join(cwd, "agents")
		if fi, err := os.Stat(dir); err == nil && fi.IsDir() {
			return dir, nil
		}
	}

	// 3. Check common install paths
	home, _ := os.UserHomeDir()
	paths := []string{
		filepath.Join(home, ".moonbase", "agents"),
		filepath.Join(home, ".config", "moonbase", "agents"),
	}
	for _, p := range paths {
		if fi, err := os.Stat(p); err == nil && fi.IsDir() {
			return p, nil
		}
	}

	// 4. Project-local .kiro/agents
	if cwd, err := os.Getwd(); err == nil {
		dir := filepath.Join(cwd, ".kiro", "agents")
		if fi, err := os.Stat(dir); err == nil && fi.IsDir() {
			return dir, nil
		}
	}

	return "", fmt.Errorf("cannot find agents directory; run from moonbase project or install agents first")
}
