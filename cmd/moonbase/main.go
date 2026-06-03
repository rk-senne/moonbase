package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/f5508037/moonbase/internal/tui"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "install":
			runInstall()
		case "list":
			runList()
		case "help", "--help", "-h":
			runHelp()
		default:
			fmt.Fprintf(os.Stderr, "Unknown command: %s\nRun 'moonbase help' for usage.\n", os.Args[1])
			os.Exit(1)
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
  moonbase install      Symlink agents to ~/.kiro/agents/
  moonbase help         This message

INSIDE THE TUI:
  ↑↓ / jk              Navigate operatives
  0-9                   Jump to operative
  enter                 Open dossier / deploy
  c                     Copy prompt to clipboard
  m                     New mission
  ?                     Operations manual
  q                     Quit`)
}

func envExists(key string) bool {
	v := os.Getenv(key)
	return strings.TrimSpace(v) != ""
}
