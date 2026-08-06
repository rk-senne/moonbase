package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rk-senne/moonbase/internal/agents"
	"github.com/rk-senne/moonbase/internal/backend"
	"github.com/rk-senne/moonbase/internal/compile"
	"github.com/rk-senne/moonbase/internal/config"
	"github.com/rk-senne/moonbase/internal/discovery"
	"github.com/rk-senne/moonbase/internal/mux"
)

// runStatus prints a quick health check of the moonbase environment.
func runStatus() {
	fmt.Println("🌙 Moonbase Status")
	fmt.Println("═══════════════════")
	fmt.Println()

	allGood := true

	// Backend
	preferred := backend.Preferred()
	available := backend.DetectAvailable()
	backendStatus := "❌ none"
	if preferred != nil && preferred.Name() != "clipboard" {
		backendStatus = fmt.Sprintf("✅ %s", preferred.Name())
	} else if len(available) > 0 {
		backendStatus = fmt.Sprintf("⚠️  clipboard only (%d backends detected)", len(available))
		allGood = false
	} else {
		allGood = false
	}
	fmt.Printf("   Backend:    %s\n", backendStatus)

	// Terminal multiplexer (tmux / cmux) — the active integration target.
	if m := mux.Detect(); m.Available() {
		session := ""
		if m.InSession() {
			session = " · in-session"
		}
		fmt.Printf("   Multiplexer:✅ %s%s (notify + split-pane + windows)\n", m.Name(), session)
	} else {
		fmt.Println("   Multiplexer:─ none (install tmux or, on macOS, cmux)")
	}

	// Agents
	dir := findAgentsDirQuiet()
	if dir != "" {
		files, _ := filepath.Glob(filepath.Join(dir, "*.md"))
		fmt.Printf("   Agents:     ✅ %d loaded from %s\n", len(files), dir)
	} else {
		fmt.Println("   Agents:     ❌ not found")
		allGood = false
	}

	// Project context
	cwd := mustGetwd()
	ctx := discovery.Discover(cwd)
	if ctx.HasSpecs() || ctx.HasSteering() || ctx.Stack.Language != "" {
		fmt.Printf("   Project:    ✅ %s\n", ctx.Summary())
	} else {
		fmt.Println("   Project:    ⚠️  no .kiro/ found")
		allGood = false
	}

	// Kiro agents installed locally?
	localAgents := filepath.Join(cwd, ".kiro", "agents")
	if files, err := filepath.Glob(filepath.Join(localAgents, "*.md")); err == nil && len(files) > 0 {
		fmt.Printf("   Local:      ✅ %d agents in .kiro/agents/\n", len(files))
	}

	// Native Interop status
	cfg := config.Load()
	fmt.Println()
	fmt.Println("   NATIVE INTEROP")
	compiledDir := cfg.Compile.OutDir
	if compiledDir == "" {
		compiledDir = ".kiro/agents"
	}
	if !filepath.IsAbs(compiledDir) {
		compiledDir = filepath.Join(cwd, compiledDir)
	}
	compiledFiles, _ := filepath.Glob(filepath.Join(compiledDir, "*.json"))
	totalAgents := 0
	if dir != "" {
		agentFiles, _ := filepath.Glob(filepath.Join(dir, "*.md"))
		totalAgents = len(agentFiles)
	}
	staleCount := 0
	for _, cf := range compiledFiles {
		base := filepath.Base(cf)
		name := strings.TrimSuffix(base, ".json")
		if dir != "" {
			mdPath := filepath.Join(dir, name+".md")
			if stale, sErr := compileIsStale(mdPath, cf); sErr == nil && stale {
				staleCount++
			}
		}
	}
	deployMode := cfg.Deploy.Mode
	if deployMode == "" {
		deployMode = "legacy"
	}
	fmt.Printf("   Compiled:   %d/%d agents", len(compiledFiles), totalAgents)
	if staleCount > 0 {
		fmt.Printf(" (%d stale)", staleCount)
	}
	fmt.Println()
	fmt.Printf("   Deploy:     %s\n", deployMode)
	if cfg.Safety.DelegateToKiro {
		fmt.Println("   Safety:     delegated to Kiro")
	}

	fmt.Println()

	// Actionable next steps when something is missing
	if !allGood {
		fmt.Println("   💡 Next steps:")
		if dir == "" {
			fmt.Println("      • Run 'moonbase install --all' to install agents")
		}
		if preferred == nil || preferred.Name() == "clipboard" {
			fmt.Println("      • Install kiro-cli, codex, or ollama for AI backend")
			fmt.Println("        Or set OPENAI_API_KEY / ANTHROPIC_API_KEY env var")
		}
		if ctx == nil || (!ctx.HasSpecs() && !ctx.HasSteering()) {
			fmt.Println("      • Run 'moonbase init' to make this project agent-ready")
		}
		fmt.Println()
	}
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
		fmt.Fprintln(os.Stderr, "❌ Cannot find agents directory. Run 'moonbase init' or run from project root.")
		osExit(1)
	}

	files, _ := filepath.Glob(filepath.Join(dir, "*.md"))
	if len(files) == 0 {
		fmt.Fprintln(os.Stderr, "❌ No agent .md files found in agents directory.")
		osExit(1)
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
		osExit(1)
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
	if !strings.Contains(agent.Prompt, "Operating Protocol") && !strings.Contains(agent.Prompt, "Evidence Standard") {
		issues = append(issues, "prompt missing Operating Protocol section")
	}

	// MCP server validation
	if len(agent.MCPServers) > 0 {
		seen := make(map[string]bool)
		for _, srv := range agent.MCPServers {
			if srv.Name == "" {
				issues = append(issues, "mcp_server has empty name")
			}
			if srv.Command == "" {
				issues = append(issues, fmt.Sprintf("mcp_server %q has empty command", srv.Name))
			}
			if seen[srv.Name] {
				issues = append(issues, fmt.Sprintf("duplicate mcp_server name %q", srv.Name))
			}
			seen[srv.Name] = true
		}
	}

	return issues
}

// compileIsStale is a thin wrapper around compile.IsStale for the status command.
func compileIsStale(mdPath, jsonPath string) (bool, error) {
	return compile.IsStale(mdPath, jsonPath)
}
