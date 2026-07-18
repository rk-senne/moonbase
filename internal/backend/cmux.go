// cmux integration utilities for the manaflow-ai/cmux terminal.
//
// cmux is a macOS terminal built for AI coding agents with features like
// split panes, workspaces, notification rings, and a programmable CLI.
// These utilities allow moonbase to:
// - Deploy agents into cmux split panes (moonbase deploy --cmux)
// - Send notifications when pipeline phases complete or need attention
// - Create dedicated workspaces for mission execution
//
// All functions are no-ops when cmux is not installed.
package backend

import (
	"fmt"
	"os/exec"
)

// CmuxAvailable returns true if the cmux CLI is installed.
func CmuxAvailable() bool {
	_, err := exec.LookPath("cmux")
	return err == nil
}

// CmuxNotify sends a notification via cmux's notification system.
// This is used by the pipeline to alert the user when phases complete or need attention.
func CmuxNotify(title, body string) error {
	if !CmuxAvailable() {
		return nil // silently skip if cmux not available
	}
	cmd := exec.Command("cmux", "notify", "--title", title, "--body", body)
	return cmd.Run()
}

// CmuxSplit creates a new split pane in cmux and runs a command in it.
// direction should be "right" or "down".
func CmuxSplit(direction, command string) error {
	if !CmuxAvailable() {
		return fmt.Errorf("cmux not available")
	}
	args := []string{"split", "--direction", direction}
	if command != "" {
		args = append(args, "--command", command)
	}
	cmd := exec.Command("cmux", args...)
	return cmd.Run()
}

// CmuxWorkspace creates or focuses a cmux workspace.
func CmuxWorkspace(name string) error {
	if !CmuxAvailable() {
		return fmt.Errorf("cmux not available")
	}
	cmd := exec.Command("cmux", "workspace", "new", "--name", name)
	return cmd.Run()
}
