package tui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/f5508037/moonbase/internal/agents"
	"github.com/f5508037/moonbase/internal/chat"
	"github.com/f5508037/moonbase/internal/history"
)

// --- Helpers ---

func (a *App) addIntel(format string, args ...any) {
	entry := IntelEntry{
		Time:    time.Now().Format("15:04"),
		Message: fmt.Sprintf(format, args...),
	}
	a.intel = append(a.intel, entry)
	if len(a.intel) > maxIntelEntries {
		a.intel = a.intel[len(a.intel)-maxIntelEntries:]
	}
	a.anim.TriggerIntelFlash()
}

func (a *App) filterAgents() {
	query := strings.ToLower(a.searchInput.Value())
	if query == "" {
		a.filtered = nil
		return
	}
	a.filtered = nil
	for i, agent := range a.registry.All() {
		name := strings.ToLower(agent.Name)
		desc := strings.ToLower(agent.Description)
		if strings.Contains(name, query) || strings.Contains(desc, query) {
			a.filtered = append(a.filtered, i)
		}
	}
}

func (a App) gitStatus() string {
	if a.gitClean {
		return "✓ clean"
	}
	return "● dirty"
}

func (a App) uptime() string {
	d := time.Since(a.startTime)
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60
	return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
}

func (a App) detectedBackends() string {
	var names []string
	for _, b := range a.backends {
		if b.Available() {
			names = append(names, b.Name())
		}
	}
	if len(names) == 0 {
		return "clipboard only"
	}
	return strings.Join(names, ", ")
}

type copyDoneMsg struct{ agent string }
type deployDoneMsg struct{ agent string }
type gitOutputMsg struct{ output string }
type spawnHookMsg struct {
	agent  string
	output string
}
type streamChunkMsg struct {
	text string
	done bool
	err  error
}

type termOutputMsg struct {
	cmd    string
	output string
}

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

func (a *App) cycleTheme() {
	switch a.theme {
	case "moonbase":
		a.theme = "treehouse"
		ColorActive = lipgloss.Color("#33CC33")
		ColorInfo = lipgloss.Color("#8B4513")
		ColorBrand = lipgloss.Color("#228B22")
		ColorHeader = lipgloss.Color("#006400")
	case "treehouse":
		a.theme = "classified"
		ColorActive = lipgloss.Color("#FF0000")
		ColorInfo = lipgloss.Color("#CC0000")
		ColorBrand = lipgloss.Color("#FF3333")
		ColorHeader = lipgloss.Color("#990000")
	case "classified":
		a.theme = "nerv"
		ColorActive = lipgloss.Color("#FF6600")
		ColorInfo = lipgloss.Color("#FF3399")
		ColorBrand = lipgloss.Color("#9900CC")
		ColorHeader = lipgloss.Color("#FF6600")
	default:
		a.theme = "moonbase"
		ColorActive = lipgloss.Color("#00FF88")
		ColorInfo = lipgloss.Color("#00BBFF")
		ColorBrand = lipgloss.Color("#FFD700")
		ColorHeader = lipgloss.Color("#FF6600")
	}
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

type toolExitMsg struct{ tool string }

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
	bin, err := exec.LookPath("tmux")
	if err != nil {
		return func() tea.Msg { return toolExitMsg{tool: "tmux (not found)"} }
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

func (a *App) openComms() {
	agent := a.registry.Get(a.selected)
	// Load existing chat history or start fresh
	conv := chat.Load(agent.Name, agent.Prompt)
	if conv != nil {
		a.comms = &CommsState{
			conv:     conv,
			viewport: viewport.New(a.width-6, a.height-6),
			agent:    agent.Name,
		}
		a.comms.rebuildContent()
	} else {
		a.comms = NewCommsState(agent.Name, agent.Prompt, a.width, a.height)
	}
	a.view = ViewComms
	a.commsInput.Focus()
	a.commsInput.Width = a.width - 8
}

func (a *App) sendCommsMessage() tea.Cmd {
	msg := a.commsInput.Value()
	if msg == "" {
		return nil
	}
	a.commsInput.Reset()
	a.comms.AddUserMessage(msg)
	a.comms.streaming = true

	// Start streaming and store channel for continued polling
	a.streamCh = chat.Stream(a.comms.conv)
	return pollStream(a.streamCh)
}

func pollStream(ch <-chan chat.StreamChunk) tea.Cmd {
	return func() tea.Msg {
		chunk, ok := <-ch
		if !ok {
			return streamChunkMsg{done: true}
		}
		return streamChunkMsg{text: chunk.Text, done: chunk.Done, err: chunk.Err}
	}
}

// --- GitHub PR ---

type prCreatedMsg struct{ output string }

func (a App) createPR() tea.Cmd {
	return func() tea.Msg {
		if _, err := exec.LookPath("gh"); err != nil {
			return prCreatedMsg{output: "gh CLI not found. Install: https://cli.github.com"}
		}
		branch, _ := exec.Command("git", "branch", "--show-current").Output()
		branchName := strings.TrimSpace(string(branch))
		if branchName == "main" || branchName == "master" {
			return prCreatedMsg{output: "Cannot create PR from main/master. Switch to a feature branch."}
		}
		out, err := exec.Command("gh", "pr", "create", "--fill").CombinedOutput()
		if err != nil {
			return prCreatedMsg{output: fmt.Sprintf("PR failed: %s", string(out))}
		}
		return prCreatedMsg{output: strings.TrimSpace(string(out))}
	}
}

// --- Multi-Agent COMMS ---

func (a *App) switchCommsAgent(name string) {
	for _, agent := range a.registry.All() {
		if strings.Contains(strings.ToLower(agent.Name), strings.ToLower(name)) {
			a.comms.agent = agent.Name
			a.comms.conv.System = agent.Prompt
			a.addIntel("COMMS switched to: %s", agent.Name)
			return
		}
	}
	a.addIntel("Agent not found: %s", name)
}

// relayToAgent sends a message to another agent's conversation and streams the response.
func (a *App) relayToAgent(targetName, msg string) tea.Cmd {
	var target *agents.Agent
	for _, agent := range a.registry.All() {
		if strings.Contains(strings.ToLower(agent.Name), strings.ToLower(targetName)) {
			ag := agent
			target = &ag
			break
		}
	}
	if target == nil {
		a.addIntel("Relay failed — agent not found: %s", targetName)
		return nil
	}

	fromAgent := a.comms.agent
	if msg == "" {
		for i := len(a.comms.conv.Messages) - 1; i >= 0; i-- {
			if a.comms.conv.Messages[i].Role == chat.RoleAssistant {
				msg = a.comms.conv.Messages[i].Content
				break
			}
		}
		if msg == "" {
			a.addIntel("Relay failed — no response to relay")
			return nil
		}
	}

	relayMsg := fmt.Sprintf("[Relayed from %s]\n%s", fromAgent, msg)

	conv := chat.Load(target.Name, target.Prompt)
	if conv == nil {
		conv = chat.NewConversation(target.Name, target.Prompt)
	}
	a.comms = &CommsState{
		conv:     conv,
		viewport: viewport.New(a.width-6, a.height-6),
		agent:    target.Name,
	}
	a.comms.rebuildContent()

	a.comms.AddUserMessage(relayMsg)
	a.comms.streaming = true
	a.addIntel("Relayed to %s from %s", target.Name, fromAgent)

	a.streamCh = chat.Stream(a.comms.conv)
	return pollStream(a.streamCh)
}

// --- Mission History View ---

func (a App) renderHistory() string {
	header := a.renderHeader("Mission History")

	var b strings.Builder
	titleStyle := lipgloss.NewStyle().Foreground(ColorBrand).Bold(true)
	dimStyle := lipgloss.NewStyle().Foreground(ColorDim)
	labelStyle := lipgloss.NewStyle().Foreground(ColorInfo)

	b.WriteString("\n")
	b.WriteString(titleStyle.Render("  ◆ MISSION LOG") + "\n")
	b.WriteString(labelStyle.Render("  ─────────────────────────────────────────────") + "\n\n")

	missions := history.Load()
	if len(missions) == 0 {
		b.WriteString(dimStyle.Render("  No missions logged yet.") + "\n")
		b.WriteString(dimStyle.Render("  Run a pipeline to start recording history.") + "\n")
	} else {
		b.WriteString(fmt.Sprintf("  %-4s %-30s %-10s %s\n", "ID", "TASK", "OUTCOME", "DURATION"))
		b.WriteString(labelStyle.Render("  ─────────────────────────────────────────────") + "\n")
		for i := len(missions) - 1; i >= 0 && i >= len(missions)-20; i-- {
			m := missions[i]
			status := "✅"
			if m.Outcome == "aborted" {
				status = "❌"
			}
			task := m.Task
			if len(task) > 28 {
				task = task[:28] + ".."
			}
			b.WriteString(fmt.Sprintf("  %-4d %-30s %s %-10s %s\n",
				m.ID, task, status, m.Outcome, m.Duration))
		}
	}

	b.WriteString("\n")
	b.WriteString(dimStyle.Render("  [esc] Back") + "\n")

	body := StylePanel.Width(a.width - 4).Render(b.String())
	statusBar := a.renderStatusBar("[esc] BACK TO DASHBOARD")
	return lipgloss.JoinVertical(lipgloss.Left, header, "\n"+body, statusBar)
}

// --- Embedded Terminal ---
//
// SECURITY TRUST BOUNDARY: execTermCmd passes user input directly to bash -c.
// This is INTENTIONAL — it is a local terminal emulator for the TUI operator.
// The trust model is identical to the user opening a terminal: the operator IS
// the user. This is NOT exposed to network input, AI-generated commands, or
// untrusted sources. Input comes only from the local keyboard via the TUI
// text input widget (a.termInput). No remote or programmatic callers exist.

func (a *App) execTermCmd(input string) tea.Cmd {
	// Handle built-in cd
	if strings.HasPrefix(input, "cd ") {
		dir := strings.TrimPrefix(input, "cd ")
		dir = strings.TrimSpace(dir)
		if dir == "~" {
			dir, _ = os.UserHomeDir()
		} else if strings.HasPrefix(dir, "~/") {
			home, _ := os.UserHomeDir()
			dir = home + dir[1:]
		}
		if err := os.Chdir(dir); err != nil {
			a.termOutput = append(a.termOutput,
				lipgloss.NewStyle().Foreground(ColorActive).Render("$ "+input),
				lipgloss.NewStyle().Foreground(ColorError).Render(err.Error()))
		} else {
			a.cwd, _ = os.Getwd()
			a.termOutput = append(a.termOutput,
				lipgloss.NewStyle().Foreground(ColorActive).Render("$ "+input))
			a.addIntel("cd → %s", a.cwd)
		}
		return nil
	}
	// Handle clear
	if input == "clear" {
		a.termOutput = nil
		return nil
	}

	return func() tea.Msg {
		out, err := exec.Command("bash", "-c", input).CombinedOutput()
		result := strings.TrimRight(string(out), "\n")
		if err != nil && result == "" {
			result = err.Error()
		}
		return termOutputMsg{cmd: input, output: result}
	}
}
