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
	agent := a.registry.Get(a.selected)
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

// isSafeHookCommand validates that a hook command only uses safe read-only operations.
func isSafeHookCommand(cmd string) bool {
	dangerous := []string{
		"curl ", "wget ", "rm ", "rm -", "mv ", "cp ",
		"chmod ", "chown ", "dd ", "mkfs",
		"python", "node ", "ruby ", "perl ",
		"eval ", "> ", ">> ", "| sh", "| bash",
		"$(curl", "$(wget", "${", "`curl", "`wget",
		"nc ", "ncat ", "socat ",
		"base64", "openssl",
		"/dev/tcp", "/dev/udp",
	}
	for _, d := range dangerous {
		if strings.Contains(cmd, d) {
			return false
		}
	}
	return true
}

func (a App) copyPrompt() tea.Cmd {
	agent := a.registry.Get(a.selected)
	return func() tea.Msg {
		cmd := exec.Command("pbcopy")
		cmd.Stdin = strings.NewReader(agent.Prompt)
		cmd.Run()
		return copyDoneMsg{agent: agent.Name}
	}
}

func (a App) deployAgent() tea.Cmd {
	agent := a.registry.Get(a.selected)
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
