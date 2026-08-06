// Package mux provides a unified integration with terminal multiplexers —
// tmux (Linux and everywhere) and cmux (the manaflow-ai macOS terminal built for
// AI agents). It gives moonbase one API for notifications, split-pane execution,
// dedicated windows/workspaces, and send-keys, backed by per-tool command
// builders so BOTH multiplexers are used to their full capacity (previously only
// cmux had notify/split/workspace support).
//
// Selection is OS-aware: cmux is preferred on macOS (purpose-built for coding
// agents), tmux everywhere else. All operations are safe no-ops when no
// multiplexer is available.
//
// SECURITY: commands are built from a fixed set of flags plus caller-supplied
// values passed as discrete exec.Command arguments (never interpolated into a
// shell string), so there is no shell-injection surface.
package mux

import (
	"os"
	"os/exec"
	"runtime"
)

// Kind identifies which multiplexer is active.
type Kind int

const (
	// None means no supported multiplexer is available.
	None Kind = iota
	// Tmux is the tmux terminal multiplexer.
	Tmux
	// Cmux is the manaflow-ai/cmux macOS terminal.
	Cmux
)

// Direction is a split-pane direction.
type Direction string

const (
	// Right splits to a new pane on the right (side-by-side).
	Right Direction = "right"
	// Down splits to a new pane below (stacked).
	Down Direction = "down"
)

// sessionName is the tmux session moonbase targets when it is not already
// running inside a tmux session.
const sessionName = "moonbase"

// Mux is a resolved multiplexer handle.
type Mux struct {
	Kind      Kind
	Bin       string // resolved executable path
	inSession bool   // running inside a tmux session ($TMUX set)
}

// Detect resolves the multiplexer for the current host and environment.
func Detect() Mux {
	return detect(runtime.GOOS, exec.LookPath, os.Getenv)
}

// detect is the injectable core of Detect (testable across OS/env).
func detect(goos string, look func(string) (string, error), getenv func(string) string) Mux {
	tmux := func() (Mux, bool) {
		if bin, err := look("tmux"); err == nil {
			return Mux{Kind: Tmux, Bin: bin, inSession: getenv("TMUX") != ""}, true
		}
		return Mux{}, false
	}
	if goos == "darwin" {
		if bin, err := look("cmux"); err == nil {
			return Mux{Kind: Cmux, Bin: bin}
		}
		if m, ok := tmux(); ok {
			return m
		}
		return Mux{Kind: None}
	}
	if m, ok := tmux(); ok {
		return m
	}
	return Mux{Kind: None}
}

// Available reports whether a usable multiplexer is present.
func (m Mux) Available() bool { return m.Kind != None && m.Bin != "" }

// Name returns the multiplexer's name ("tmux", "cmux", or "none").
func (m Mux) Name() string {
	switch m.Kind {
	case Tmux:
		return "tmux"
	case Cmux:
		return "cmux"
	default:
		return "none"
	}
}

// InSession reports whether operations will target the current session. For
// tmux this means $TMUX is set; cmux's CLI always targets the running app.
func (m Mux) InSession() bool {
	return m.Kind == Cmux || (m.Kind == Tmux && m.inSession)
}

// tmuxTarget returns the tmux "-t <session>" args when moonbase is not already
// inside a tmux session, so operations still reach the moonbase session.
func (m Mux) tmuxTarget() []string {
	if m.inSession {
		return nil
	}
	return []string{"-t", sessionName}
}

// notifyArgs builds the notification command args for the active multiplexer.
func (m Mux) notifyArgs(title, body string) ([]string, bool) {
	switch m.Kind {
	case Cmux:
		return []string{"notify", "--title", title, "--body", body}, true
	case Tmux:
		msg := "🌙 " + title
		if body != "" {
			msg += " — " + body
		}
		args := []string{"display-message"}
		args = append(args, m.tmuxTarget()...)
		return append(args, msg), true
	default:
		return nil, false
	}
}

// splitArgs builds the split-pane command args, running command in the new pane.
func (m Mux) splitArgs(dir Direction, command string) ([]string, bool) {
	if dir != Right && dir != Down {
		dir = Right
	}
	switch m.Kind {
	case Cmux:
		args := []string{"split", "--direction", string(dir)}
		if command != "" {
			args = append(args, "--command", command)
		}
		return args, true
	case Tmux:
		flag := "-h" // horizontal split = side-by-side (right)
		if dir == Down {
			flag = "-v"
		}
		args := []string{"split-window", flag}
		args = append(args, m.tmuxTarget()...)
		if command != "" {
			args = append(args, command)
		}
		return args, true
	default:
		return nil, false
	}
}

// windowArgs builds a "new window / workspace" command running command in it.
func (m Mux) windowArgs(name, command string) ([]string, bool) {
	switch m.Kind {
	case Cmux:
		return []string{"workspace", "new", "--name", name}, true
	case Tmux:
		args := []string{"new-window", "-n", name}
		args = append(args, m.tmuxTarget()...)
		if command != "" {
			args = append(args, command)
		}
		return args, true
	default:
		return nil, false
	}
}

// sendKeysArgs builds a send-keys command (tmux only; cmux has no equivalent).
func (m Mux) sendKeysArgs(command string) ([]string, bool) {
	if m.Kind != Tmux {
		return nil, false
	}
	args := []string{"send-keys"}
	args = append(args, m.tmuxTarget()...)
	return append(args, "--", command, "Enter"), true
}

// run executes the built command, returning nil for unsupported/no-op cases.
func (m Mux) run(args []string, ok bool) error {
	if !m.Available() || !ok {
		return nil
	}
	return exec.Command(m.Bin, args...).Run()
}

// Notify posts a notification through the active multiplexer.
func (m Mux) Notify(title, body string) error {
	return m.run(m.notifyArgs(title, body))
}

// SplitRun opens a split pane and runs command in it.
func (m Mux) SplitRun(dir Direction, command string) error {
	return m.run(m.splitArgs(dir, command))
}

// NewWindow opens a new window (tmux) or workspace (cmux) named name, optionally
// running command in it.
func (m Mux) NewWindow(name, command string) error {
	return m.run(m.windowArgs(name, command))
}

// SendKeys types command into the active session and presses Enter (tmux only).
func (m Mux) SendKeys(command string) error {
	return m.run(m.sendKeysArgs(command))
}
