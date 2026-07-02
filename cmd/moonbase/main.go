package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/f5508037/moonbase/internal/agents"
	"github.com/f5508037/moonbase/internal/backend"
	clip "github.com/f5508037/moonbase/internal/clipboard"
	"github.com/f5508037/moonbase/internal/config"
	"github.com/f5508037/moonbase/internal/discovery"
	"github.com/f5508037/moonbase/internal/history"
	"github.com/f5508037/moonbase/internal/tui"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "install":
			runInstall()
		case "list":
			runList()
		case "deploy":
			if len(os.Args) < 3 {
				fmt.Fprintln(os.Stderr, "Usage: moonbase deploy <numbuh>")
				os.Exit(1)
			}
			runDeploy(os.Args[2])
		case "mission":
			task := strings.Join(os.Args[2:], " ")
			if task == "" {
				fmt.Fprintln(os.Stderr, "Usage: moonbase mission <task description>")
				os.Exit(1)
			}
			runMission(task)
		case "export":
			if len(os.Args) < 3 {
				fmt.Fprintln(os.Stderr, "Usage: moonbase export <mission-id>")
				os.Exit(1)
			}
			id, _ := strconv.Atoi(os.Args[2])
			fmt.Println(history.Export(id))
		case "snippet":
			runSnippet()
		case "help", "--help", "-h":
			runHelp()
		case "config":
			runConfig()
		default:
			fmt.Fprintf(os.Stderr, "Unknown command: %s\nRun 'moonbase help' for usage.\n", os.Args[1])
			os.Exit(1)
		}
		return
	}

	// Pipe mode: if stdin is not a TTY, read it and deploy
	if !isTerminal() {
		input, _ := io.ReadAll(os.Stdin)
		task := strings.TrimSpace(string(input))
		if task != "" {
			fmt.Printf("🌙 Pipe mode — task: %s\n", task)
			fmt.Println("Deploy to kiro-cli with knd-council...")
			if kiro, err := exec.LookPath("kiro-cli"); err == nil {
				cmd := exec.Command(kiro, "chat", "--agent", "knd-council")
				cmd.Stdin = strings.NewReader(task)
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				cmd.Run()
			} else {
				// Copy task to clipboard
				clip.Copy(task)
				fmt.Println("✓ Task copied to clipboard")
			}
		}
		return
	}

	// Default: launch TUI
	p := tea.NewProgram(tui.NewApp(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// runInstall is defined in install.go

func runList() {
	fmt.Println("🌙 KND MOONBASE — OPERATIVE ROSTER")
	fmt.Println("═══════════════════════════════════════")
	fmt.Println()
	fmt.Println("  SECTOR V")

	agents := []struct {
		num, name, role string
	}{
		{"0", "Monty Uno", "System Architect"},
		{"1", "Nigel Uno", "Analyst"},
		{"2", "Hoagie Gilligan", "Architect"},
		{"3", "Kuki Sanban", "Implementer"},
		{"4", "Wallabee Beatles", "QA"},
		{"5", "Abigail Lincoln", "Reviewer"},
	}

	for _, a := range agents {
		fmt.Printf("  [%s] %-18s %s\n", a.num, a.name, a.role)
	}

	fmt.Println()
	fmt.Println("  SPECIALISTS")

	specialists := []struct {
		num, name, role string
	}{
		{"362", "Rachel McKenzie", "DevOps"},
		{"274", "Chad Dickson", "Security"},
		{"86", "Fanny Fulbright", "Tech Debt"},
		{"999", "Pioneer", "Documentation"},
		{"13", "The Jinx", "Chaos Tester"},
		{"Z", "Sector Z", "Legacy Archaeologist"},
		{"9", "Maurice", "Migration Specialist"},
	}

	for _, a := range specialists {
		fmt.Printf("  [%s] %-18s %s\n", a.num, a.name, a.role)
	}

	fmt.Println()

	// Show backends
	fmt.Println("  AI BACKENDS")
	backends := []string{"kiro-cli", "codex", "ollama"}
	for _, b := range backends {
		if _, err := exec.LookPath(b); err == nil {
			fmt.Printf("  ✓ %s\n", b)
		} else {
			fmt.Printf("  ✗ %s\n", b)
		}
	}
	if envExists("OPENAI_API_KEY") {
		fmt.Println("  ✓ openai")
	}
	if envExists("ANTHROPIC_API_KEY") {
		fmt.Println("  ✓ anthropic")
	}
	fmt.Println()
}

func runHelp() {
	fmt.Println(`🌙 Moonbase — KND Tactical Operations Terminal

USAGE:
  moonbase              Launch the TUI dashboard
  moonbase list         Show operative roster
  moonbase deploy <n>   Deploy operative by numbuh (e.g. deploy 4)
  moonbase mission <t>  Run full KND Council pipeline on a task
  moonbase install      Install agents to .kiro/agents/ (--all, --global)
  moonbase export <id>  Export mission report as markdown
  moonbase snippet      Manage saved snippets
  moonbase help         This message

PIPE MODE:
  echo "fix the bug" | moonbase     Deploy task via pipe (no TUI)
  echo "fix auth" | moonbase deploy 4   Deploy to specific agent

INSIDE THE TUI:
  ↑↓ / jk    Navigate         m    New mission     C    Open COMMS
  0-9        Jump to agent     H    Mission history P    Create PR
  enter      Dossier/deploy    w    Toggle watcher  L/B/V  Launch tools
  /          Search            T    Cycle theme     tab  Cycle focus
  ctrl+s     Snippets (COMMS)  ctrl+f  Attach file (COMMS)
  @name      Switch agent (COMMS)
  >name      Relay to agent (COMMS)   >>name msg  Relay+message
  ?          Operations manual q    Quit`)
}

func isTerminal() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return true
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

func runSnippet() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: moonbase snippet save <name>")
		fmt.Println("       moonbase snippet list")
		return
	}
	switch os.Args[2] {
	case "list":
		home, _ := os.UserHomeDir()
		path := filepath.Join(home, ".config", "moonbase", "snippets.json")
		data, err := os.ReadFile(path)
		if err != nil {
			fmt.Println("No snippets saved yet.")
			return
		}
		fmt.Println(string(data))
	case "save":
		if len(os.Args) < 4 {
			fmt.Println("Usage: moonbase snippet save <name>")
			fmt.Println("  Then type the snippet content (end with ctrl+d)")
			return
		}
		name := os.Args[3]
		scanner := bufio.NewScanner(os.Stdin)
		var lines []string
		for scanner.Scan() {
			lines = append(lines, scanner.Text())
		}
		content := strings.Join(lines, "\n")
		home, _ := os.UserHomeDir()
		path := filepath.Join(home, ".config", "moonbase", "snippets.json")
		os.MkdirAll(filepath.Dir(path), 0700)

		// Load existing
		var existing []map[string]string
		if data, err := os.ReadFile(path); err == nil {
			json.Unmarshal(data, &existing)
		}
		existing = append(existing, map[string]string{"name": name, "content": content})
		data, _ := json.MarshalIndent(existing, "", "  ")
		os.WriteFile(path, data, 0600)
		fmt.Printf("✓ Snippet saved: %s\n", name)
	}
}

func agentsDir() string {
	// 1. Check relative to executable
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Join(filepath.Dir(exe), "..", "agents")
		if fi, err := os.Stat(dir); err == nil && fi.IsDir() {
			return dir
		}
		dir = filepath.Join(filepath.Dir(exe), "agents")
		if fi, err := os.Stat(dir); err == nil && fi.IsDir() {
			return dir
		}
	}
	// 2. Check relative to CWD
	if cwd, err := os.Getwd(); err == nil {
		dir := filepath.Join(cwd, "agents")
		if fi, err := os.Stat(dir); err == nil && fi.IsDir() {
			return dir
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
			return p
		}
	}
	// 4. Project-local .kiro/agents
	if cwd, err := os.Getwd(); err == nil {
		dir := filepath.Join(cwd, ".kiro", "agents")
		if fi, err := os.Stat(dir); err == nil && fi.IsDir() {
			return dir
		}
	}
	fmt.Fprintln(os.Stderr, "⚠️  Cannot find agents directory. Run from moonbase project or install agents first.")
	os.Exit(1)
	return ""
}

func runDeploy(numbuh string) {
	dir := agentsDir()

	// Security: validate agent identifier (prevent path traversal)
	if !isValidAgentID(numbuh) {
		fmt.Fprintf(os.Stderr, "Invalid agent identifier: %s\n", numbuh)
		fmt.Fprintf(os.Stderr, "Available: moonbase deploy <0-5|9|13|86|274|362|999|z|council>\n")
		os.Exit(1)
	}

	// Check if there's a task argument after the numbuh
	var task string
	if len(os.Args) > 3 {
		task = strings.Join(os.Args[3:], " ")
	}

	// Resolve agent file name from input
	var agentFile string
	switch {
	case numbuh == "council" || numbuh == "k":
		agentFile = filepath.Join(dir, "knd-council.md")
	case numbuh == "z" || numbuh == "Z":
		agentFile = filepath.Join(dir, "sector-z.md")
	default:
		agentFile = filepath.Join(dir, fmt.Sprintf("numbuh-%s.md", numbuh))
	}

	// Parse the agent .md file
	agent, err := agents.ParseAgentFile(agentFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Agent not found: %s\n  (looked for %s)\n", numbuh, agentFile)
		fmt.Fprintf(os.Stderr, "\nAvailable: moonbase deploy <0-5|9|13|86|274|362|999|z|council>\n")
		os.Exit(1)
	}

	// Discover project context
	cwd, _ := os.Getwd()
	ctx, _ := discovery.Discover(cwd)

	fmt.Printf("🌙 Deploying %s — %s (%s)\n", agent.Name, agent.Designation, agent.Role)
	if ctx != nil && (ctx.HasSpecs() || ctx.HasSteering()) {
		fmt.Printf("   Context: %s\n", ctx.Summary())
	}
	if task != "" {
		fmt.Printf("   Task: %s\n", task)
	}
	fmt.Println()

	// Compose prompt with project context and task
	composed := discovery.ComposePrompt(agent.Prompt, ctx, task)

	// Try kiro-cli with syscall.Exec (replaces this process)
	if kiro, kErr := exec.LookPath("kiro-cli"); kErr == nil {
		// Build kiro-cli args — pass composed prompt as the initial input
		args := []string{"kiro-cli", "chat"}

		// If the agent is installed in .kiro/agents/, use --agent flag
		localAgent := filepath.Join(cwd, ".kiro", "agents", agent.Name+".md")
		if _, statErr := os.Stat(localAgent); statErr == nil {
			args = append(args, "--agent", agent.Name)
		}

		// Add task as the input question
		if task != "" {
			args = append(args, task)
		}

		// syscall.Exec replaces this process — full TTY to kiro-cli
		execErr := execSyscall(kiro, args, backend.SafeEnv())
		// If exec fails, fall through to fallback
		if execErr != nil {
			fmt.Fprintf(os.Stderr, "   ⚠️  kiro-cli exec failed: %v\n", execErr)
		}
	}

	// Fallback: copy composed prompt to clipboard
	fmt.Println("   No interactive backend available. Copying prompt to clipboard...")
	fmt.Println()

	if cErr := clip.Copy(composed); cErr == nil {
		fmt.Printf("   ✅ Copied to clipboard (%d chars)\n", len(composed))
		fmt.Println()
		fmt.Printf("   Agent:    %s (%s)\n", agent.Designation, agent.Role)
		fmt.Printf("   Tools:    %s\n", strings.Join(agent.Tools, ", "))
		if task != "" {
			fmt.Printf("   Task:     %s\n", task)
		}
		fmt.Println()
		fmt.Println("   Paste into: Claude / ChatGPT / Kiro IDE / any AI tool")
	} else {
		fmt.Printf("   Agent: %s (%s)\n", agent.Name, agent.Role)
		fmt.Printf("   Prompt: %d chars\n", len(composed))
		fmt.Println("   (No clipboard available — install xclip on Linux or use kiro-cli)")
	}
}

func runConfig() {
	cfg := config.Load()
	fmt.Println("🌙 Moonbase Configuration")
	fmt.Printf("   Path: %s\n\n", config.Path())
	fmt.Println(config.Show(cfg))
}

func envExists(key string) bool {
	v := os.Getenv(key)
	return strings.TrimSpace(v) != ""
}

// execSyscall replaces the current process with the given command.
// This gives the target program full terminal control (TTY, colours, readline).
func execSyscall(binary string, args []string, env []string) error {
	return syscall.Exec(binary, args, env)
}

// isValidAgentID checks that an agent identifier contains only safe characters.
// Prevents path traversal attacks via deploy command.
func isValidAgentID(id string) bool {
	if len(id) == 0 || len(id) > 20 {
		return false
	}
	for _, c := range id {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '-') {
			return false
		}
	}
	return true
}
