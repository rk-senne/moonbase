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

// FindAllAgentDirs returns all discoverable agent directories in priority order:
//   - [0] built-in (relative to executable or CWD)
//   - [1] user (~/.moonbase/agents/)
//   - [2] project (.kiro/agents/ in CWD)
//
// Empty strings are returned for directories that don't exist.
// At least one directory must exist or an error is returned.
func FindAllAgentDirs(configPath string) (builtIn, user, project string, err error) {
	// Built-in: use FindAgentsDir logic (config override → exe-relative → CWD-relative)
	builtIn, _ = FindAgentsDir(configPath)

	// User: ~/.moonbase/agents/
	home, _ := os.UserHomeDir()
	if home != "" {
		userDir := filepath.Join(home, ".moonbase", "agents")
		if fi, err := os.Stat(userDir); err == nil && fi.IsDir() {
			user = userDir
		}
	}

	// Project: .kiro/agents/ in CWD
	if cwd, cwdErr := os.Getwd(); cwdErr == nil {
		projectDir := filepath.Join(cwd, ".kiro", "agents")
		if fi, statErr := os.Stat(projectDir); statErr == nil && fi.IsDir() {
			project = projectDir
		}
	}

	// Ensure at least one directory was found
	if builtIn == "" && user == "" && project == "" {
		err = fmt.Errorf("cannot find any agents directory; run from moonbase project or install agents first")
	}

	// Don't duplicate: if built-in resolved to user or project dir, clear it
	if builtIn != "" && builtIn == user {
		user = ""
	}
	if builtIn != "" && builtIn == project {
		project = ""
	}

	return
}

