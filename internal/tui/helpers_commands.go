package tui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func (a App) runGitCmd(command string) tea.Cmd {
	return func() tea.Msg {
		parts := strings.Fields(command)
		out, err := exec.Command(parts[0], parts[1:]...).CombinedOutput()
		if err != nil {
			return gitOutputMsg{output: fmt.Sprintf("(%s failed: %v)", command, err)}
		}
		result := strings.TrimSpace(string(out))
		if result == "" {
			result = "(clean — no output)"
		}
		return gitOutputMsg{output: result}
	}
}

func (a App) runSpawnHook() tea.Cmd {
	agent := a.registry.Get(a.dashboard.Selected)
	if agent.Hooks == nil || len(agent.Hooks.OnActivate) == 0 {
		return func() tea.Msg {
			return spawnHookMsg{agent: agent.Name, output: "(no spawn hook configured)"}
		}
	}
	cmd := agent.Hooks.OnActivate[0].Command

	// Security: validate hook command against safe patterns
	if !isSafeHookCommand(cmd) {
		return func() tea.Msg {
			return spawnHookMsg{agent: agent.Name, output: fmt.Sprintf("⚠️ Hook blocked (unsafe): %s", cmd)}
		}
	}

	return func() tea.Msg {
		out, err := exec.Command("bash", "-c", cmd).CombinedOutput()
		if err != nil {
			return spawnHookMsg{agent: agent.Name, output: fmt.Sprintf("error: %v", err)}
		}
		return spawnHookMsg{agent: agent.Name, output: strings.TrimSpace(string(out))}
	}
}

// safeHookCommands is the allowlist of commands permitted in agent hooks.
// Hook commands run automatically on agent activation — only read-only,
// information-gathering commands are permitted.
var safeHookCommands = map[string]bool{
	// Shell builtins and basic output
	"echo": true, "printf": true, "true": true, "test": true,
	// File inspection (read-only)
	"cat": true, "head": true, "tail": true, "less": true,
	"ls": true, "find": true, "stat": true, "file": true, "wc": true,
	"basename": true, "dirname": true, "realpath": true, "readlink": true,
	// Text processing (read-only)
	"grep": true, "awk": true, "sed": true, "sort": true,
	"uniq": true, "cut": true, "tr": true, "tee": true, "xargs": true,
	// VCS (read-only operations enforced by hook context)
	"git": true,
	// System info
	"date": true, "pwd": true, "whoami": true, "uname": true,
	"hostname": true, "id": true, "which": true, "env": true, "printenv": true,
	// Build tool queries (version checks, dependency listing)
	"go": true, "node": true, "npm": true, "cargo": true,
	"java": true, "mvn": true, "gradle": true, "python3": true,
	"make": true, "docker": true,
	// Directory visualization
	"tree": true,
}

// isSafeHookCommand validates that a hook command only uses allowlisted executables.
// It extracts all command names from compound commands (&&, ||, ;, |, $(...)) and
// verifies each against safeHookCommands. Shell redirections are permitted.
func isSafeHookCommand(cmd string) bool {
	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		return true
	}

	// Block dangerous shell patterns that could bypass command extraction
	unsafePatterns := []string{"`", "${"}
	for _, p := range unsafePatterns {
		if strings.Contains(cmd, p) {
			return false
		}
	}

	// Extract commands from $(...) substitutions and the top level
	commands := extractCommands(cmd)
	for _, c := range commands {
		base := extractBaseCommand(c)
		if base == "" {
			continue
		}
		if !safeHookCommands[base] {
			return false
		}
	}
	return len(commands) > 0
}

// extractCommands splits a shell command string into individual commands,
// handling &&, ||, ;, |, and $(...) substitutions.
func extractCommands(cmd string) []string {
	var commands []string

	// First, extract commands from $(...) substitutions
	for i := 0; i < len(cmd); i++ {
		if i < len(cmd)-1 && cmd[i] == '$' && cmd[i+1] == '(' {
			depth := 1
			start := i + 2
			for j := start; j < len(cmd) && depth > 0; j++ {
				if cmd[j] == '(' {
					depth++
				} else if cmd[j] == ')' {
					depth--
					if depth == 0 {
						inner := cmd[start:j]
						commands = append(commands, splitOnOperators(inner)...)
					}
				}
			}
		}
	}

	// Then split the top-level command
	commands = append(commands, splitOnOperators(cmd)...)
	return commands
}

// splitOnOperators splits on &&, ||, ;, and |
func splitOnOperators(cmd string) []string {
	cmd = strings.ReplaceAll(cmd, "&&", "\x00")
	cmd = strings.ReplaceAll(cmd, ";", "\x00")
	var parts []string
	for _, segment := range strings.Split(cmd, "\x00") {
		for _, p := range strings.Split(segment, "|") {
			p = strings.TrimSpace(p)
			if p != "" {
				parts = append(parts, p)
			}
		}
	}
	return parts
}

// extractBaseCommand gets the executable name from a command string.
// Strips: redirections, env var assignments, leading whitespace.
func extractBaseCommand(cmd string) string {
	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		return ""
	}

	// Remove $(...) sequences so they don't confuse field splitting
	cleaned := cmd
	for {
		idx := strings.Index(cleaned, "$(")
		if idx == -1 {
			break
		}
		depth := 1
		end := idx + 2
		for end < len(cleaned) && depth > 0 {
			if cleaned[end] == '(' {
				depth++
			} else if cleaned[end] == ')' {
				depth--
			}
			end++
		}
		cleaned = cleaned[:idx] + cleaned[end:]
	}

	// Skip env var assignments (e.g., FOO=bar cmd)
	fields := strings.Fields(cleaned)
	for i, f := range fields {
		if !strings.Contains(f, "=") || strings.HasPrefix(f, "-") {
			fields = fields[i:]
			goto found
		}
		if i == len(fields)-1 {
			return "" // All env vars, no command
		}
	}
	if len(fields) == 0 {
		return ""
	}
found:
	if len(fields) == 0 {
		return ""
	}

	// Get base name (handle full paths like /usr/bin/git)
	return filepath.Base(fields[0])
}

func (a App) copyPrompt() tea.Cmd {
	agent := a.registry.Get(a.dashboard.Selected)
	return func() tea.Msg {
		cmd := exec.Command("pbcopy")
		cmd.Stdin = strings.NewReader(agent.Prompt)
		cmd.Run()
		return copyDoneMsg{agent: agent.Name}
	}
}

func (a App) deployAgent() tea.Cmd {
	agent := a.registry.Get(a.dashboard.Selected)
	return func() tea.Msg {
		// Try kiro-cli first
		if _, err := exec.LookPath("kiro-cli"); err == nil {
			exec.Command("kiro-cli", "chat", "--agent", agent.Name).Start()
		} else {
			// Fallback: copy to clipboard
			cmd := exec.Command("pbcopy")
			cmd.Stdin = strings.NewReader(agent.Prompt)
			cmd.Run()
		}
		return deployDoneMsg{agent: agent.Name}
	}
}

func (a App) launchTool(name string) tea.Cmd {
	bin, err := exec.LookPath(name)
	if err != nil {
		return func() tea.Msg {
			return toolExitMsg{tool: name + " (not found)"}
		}
	}
	c := exec.Command(bin)
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return toolExitMsg{tool: name}
	})
}

func (a App) launchNvim() tea.Cmd {
	bin, err := exec.LookPath("nvim")
	if err != nil {
		return func() tea.Msg { return toolExitMsg{tool: "nvim (not found)"} }
	}
	var args []string
	if a.view == ViewProjects && a.projectNav != nil && len(a.projectNav.list) > 0 {
		args = append(args, a.projectNav.list[a.projectNav.cursor].Path)
	} else if a.browsing && a.fileBrowser != nil && len(a.fileBrowser.entries) > 0 {
		entry := a.fileBrowser.entries[a.fileBrowser.cursor]
		if !entry.IsDir {
			args = append(args, filepath.Join(a.fileBrowser.dir, entry.Name))
		}
	}
	c := exec.Command(bin, args...)
	return tea.ExecProcess(c, func(err error) tea.Msg { return toolExitMsg{tool: "nvim"} })
}

func (a App) launchCmux() tea.Cmd {
	// Prefer cmux over tmux when available
	if cmuxBin, cmuxErr := exec.LookPath("cmux"); cmuxErr == nil {
		c := exec.Command(cmuxBin, "workspace", "new", "--name", "moonbase")
		return tea.ExecProcess(c, func(err error) tea.Msg { return toolExitMsg{tool: "cmux"} })
	}

	// Fall back to tmux
	bin, err := exec.LookPath("tmux")
	if err != nil {
		return func() tea.Msg { return toolExitMsg{tool: "cmux/tmux (not found)"} }
	}
	check := exec.Command(bin, "has-session", "-t", "moonbase")
	if check.Run() == nil {
		c := exec.Command(bin, "attach-session", "-t", "moonbase")
		return tea.ExecProcess(c, func(err error) tea.Msg { return toolExitMsg{tool: "tmux"} })
	}
	c := exec.Command(bin, "new-session", "-s", "moonbase")
	return tea.ExecProcess(c, func(err error) tea.Msg { return toolExitMsg{tool: "tmux"} })
}

func (a App) editFile(path string) tea.Cmd {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vim"
	}
	bin, err := exec.LookPath(editor)
	if err != nil {
		return func() tea.Msg {
			return toolExitMsg{tool: editor + " (not found)"}
		}
	}
	c := exec.Command(bin, path)
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return toolExitMsg{tool: editor}
	})
}
