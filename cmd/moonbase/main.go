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

	tea "github.com/charmbracelet/bubbletea"
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
				clip := exec.Command("pbcopy")
				clip.Stdin = strings.NewReader(task)
				clip.Run()
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

func runInstall() {
	home, _ := os.UserHomeDir()
	kiroDir := filepath.Join(home, ".kiro", "agents")
	agentsDir := "./agents"

	// Create .kiro/agents if it doesn't exist
	os.MkdirAll(kiroDir, 0755)

	files, _ := filepath.Glob(filepath.Join(agentsDir, "*.json"))
	if len(files) == 0 {
		fmt.Println("No agent configs found in ./agents/")
		os.Exit(1)
	}

	for _, src := range files {
		name := filepath.Base(src)
		dst := filepath.Join(kiroDir, name)

		// Remove existing symlink/file
		os.Remove(dst)

		absSrc, _ := filepath.Abs(src)
		if err := os.Symlink(absSrc, dst); err != nil {
			fmt.Printf("  ✗ %s: %v\n", name, err)
		} else {
			fmt.Printf("  ✓ %s → %s\n", name, dst)
		}
	}

	fmt.Printf("\n🌙 %d agents installed to %s\n", len(files), kiroDir)
}

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
  moonbase export <id>  Export mission report as markdown
  moonbase snippet      Manage saved snippets
  moonbase install      Symlink agents to ~/.kiro/agents/
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
		os.MkdirAll(filepath.Dir(path), 0755)

		// Load existing
		var existing []map[string]string
		if data, err := os.ReadFile(path); err == nil {
			json.Unmarshal(data, &existing)
		}
		existing = append(existing, map[string]string{"name": name, "content": content})
		data, _ := json.MarshalIndent(existing, "", "  ")
		os.WriteFile(path, data, 0644)
		fmt.Printf("✓ Snippet saved: %s\n", name)
	}
}

func runDeploy(numbuh string) {
	// Map numbuh to agent filename
	agentFile := fmt.Sprintf("./agents/numbuh-%s.json", numbuh)
	if numbuh == "council" || numbuh == "k" {
		agentFile = "./agents/knd-council.json"
	} else if numbuh == "z" || numbuh == "Z" {
		agentFile = "./agents/sector-z.json"
	}

	data, err := os.ReadFile(agentFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Agent not found: numbuh-%s\n", numbuh)
		os.Exit(1)
	}

	// Extract agent name
	var agent struct {
		Name   string `json:"name"`
		Prompt string `json:"prompt"`
	}
	if err := json.Unmarshal(data, &agent); err != nil {
		fmt.Fprintf(os.Stderr, "Invalid agent config: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("🌙 Deploying %s...\n", agent.Name)

	// Try kiro-cli
	if kiro, err := exec.LookPath("kiro-cli"); err == nil {
		cmd := exec.Command(kiro, "chat", "--agent", agent.Name)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Run()
		return
	}

	// Fallback: copy prompt to clipboard
	clip := exec.Command("pbcopy")
	clip.Stdin = strings.NewReader(agent.Prompt)
	if err := clip.Run(); err == nil {
		fmt.Println("✓ Prompt copied to clipboard (kiro-cli not found)")
	} else {
		fmt.Println("Prompt:")
		fmt.Println(agent.Prompt[:min(200, len(agent.Prompt))] + "...")
	}
}

func envExists(key string) bool {
	v := os.Getenv(key)
	return strings.TrimSpace(v) != ""
}
