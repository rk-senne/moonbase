package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/f5508037/moonbase/internal/agents"
	"github.com/f5508037/moonbase/internal/backend"
	"github.com/f5508037/moonbase/internal/discovery"
)

// runStatus prints a quick health check of the moonbase environment.
func runStatus() {
	fmt.Println("🌙 Moonbase Status")
	fmt.Println()

	// Backend
	preferred := backend.Preferred()
	available := backend.DetectAvailable()
	backendStatus := "❌ none"
	if preferred != nil && preferred.Name() != "clipboard" {
		backendStatus = fmt.Sprintf("✅ %s", preferred.Name())
	} else if len(available) > 0 {
		backendStatus = fmt.Sprintf("⚠️  clipboard only (%d backends detected)", len(available))
	}
	fmt.Printf("   Backend:    %s\n", backendStatus)

	// Agents
	dir := findAgentsDirQuiet()
	if dir != "" {
		files, _ := filepath.Glob(filepath.Join(dir, "*.md"))
		fmt.Printf("   Agents:     %d loaded from %s\n", len(files), dir)
	} else {
		fmt.Println("   Agents:     ❌ not found")
	}

	// Project context
	cwd, _ := os.Getwd()
	ctx, _ := discovery.Discover(cwd)
	if ctx != nil {
		if ctx.HasSpecs() || ctx.HasSteering() || ctx.Stack.Language != "" {
			fmt.Printf("   Project:    %s\n", ctx.Summary())
		} else {
			fmt.Println("   Project:    (no .kiro/ found — run 'moonbase init')")
		}
	}

	// Kiro agents installed locally?
	localAgents := filepath.Join(cwd, ".kiro", "agents")
	if files, err := filepath.Glob(filepath.Join(localAgents, "*.md")); err == nil && len(files) > 0 {
		fmt.Printf("   Local:      %d agents in .kiro/agents/\n", len(files))
	}

	fmt.Println()
}

// findAgentsDirQuiet finds agents dir without printing errors.
func findAgentsDirQuiet() string {
	if exe, err := os.Executable(); err == nil {
		candidates := []string{
			filepath.Join(filepath.Dir(exe), "..", "agents"),
			filepath.Join(filepath.Dir(exe), "agents"),
		}
		for _, c := range candidates {
			if isAgentsDir(c) {
				abs, _ := filepath.Abs(c)
				return abs
			}
		}
	}
	if cwd, err := os.Getwd(); err == nil {
		if isAgentsDir(filepath.Join(cwd, "agents")) {
			return filepath.Join(cwd, "agents")
		}
	}
	return ""
}

// runLint validates all agent .md files for correctness.
func runLint() {
	fmt.Println("🌙 Moonbase Agent Lint")
	fmt.Println()

	dir := findAgentsDirQuiet()
	if dir == "" {
		fmt.Fprintln(os.Stderr, "❌ Cannot find agents directory")
		os.Exit(1)
	}

	files, _ := filepath.Glob(filepath.Join(dir, "*.md"))
	if len(files) == 0 {
		fmt.Fprintln(os.Stderr, "❌ No agent .md files found")
		os.Exit(1)
	}

	var allAgents []*agents.Agent
	errors := 0

	// Parse all agents
	for _, f := range files {
		agent, err := agents.ParseAgentFile(f)
		if err != nil {
			fmt.Printf("   ❌ %s: parse error: %v\n", filepath.Base(f), err)
			errors++
			continue
		}
		allAgents = append(allAgents, agent)
	}

	// Build name registry for cross-reference validation
	agentNames := make(map[string]bool)
	for _, a := range allAgents {
		agentNames[a.Name] = true
	}

	// Validate each agent
	for _, agent := range allAgents {
		issues := lintAgent(agent, agentNames)
		if len(issues) > 0 {
			fmt.Printf("   ⚠️  %s:\n", agent.Name)
			for _, issue := range issues {
				fmt.Printf("      - %s\n", issue)
			}
			errors += len(issues)
		} else {
			fmt.Printf("   ✅ %s\n", agent.Name)
		}
	}

	fmt.Println()
	if errors > 0 {
		fmt.Printf("   %d issue(s) found.\n", errors)
		os.Exit(1)
	}
	fmt.Printf("   All %d agents valid.\n", len(allAgents))
}

// lintAgent checks a single agent for common issues.
func lintAgent(agent *agents.Agent, knownAgents map[string]bool) []string {
	var issues []string

	// Required fields
	if agent.Name == "" {
		issues = append(issues, "missing 'name' field")
	}
	if agent.Role == "" {
		issues = append(issues, "missing 'role' field")
	}
	if len(agent.Tools) == 0 {
		issues = append(issues, "missing 'tools' field (no tools defined)")
	}
	if agent.Prompt == "" {
		issues = append(issues, "empty prompt body (no markdown content after frontmatter)")
	}

	// Routing cross-references
	if agent.Routing != nil {
		for _, ref := range agent.Routing.Available {
			if !knownAgents[ref] {
				issues = append(issues, fmt.Sprintf("routing references unknown agent: %s", ref))
			}
		}
		for _, ref := range agent.Routing.Trusted {
			if !knownAgents[ref] {
				issues = append(issues, fmt.Sprintf("trusted references unknown agent: %s", ref))
			}
		}
	}

	// Prompt should contain Operating Protocol
	if !containsStr(agent.Prompt, "Operating Protocol") && !containsStr(agent.Prompt, "Evidence Standard") {
		issues = append(issues, "prompt missing Operating Protocol section")
	}

	return issues
}

func containsStr(s, sub string) bool {
	return len(s) > 0 && len(sub) > 0 && findStr(s, sub)
}

func findStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
