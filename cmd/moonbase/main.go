package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/rk-senne/moonbase/internal/agents"
	"github.com/rk-senne/moonbase/internal/backend"
	clip "github.com/rk-senne/moonbase/internal/clipboard"
	"github.com/rk-senne/moonbase/internal/config"
	"github.com/rk-senne/moonbase/internal/discovery"
	"github.com/rk-senne/moonbase/internal/mux"
)

// maxPipeInputSize is the maximum bytes accepted from piped stdin (1MB).
// Prevents OOM if a malicious or accidental large pipe is connected.
const maxPipeInputSize = 1 << 20

func main() {
	if err := rootCmd.Execute(); err != nil {
		osExit(1)
	}
}

// runList displays the full KND operative roster and detected backends.
func runList() {
	fmt.Println("🌙 KND MOONBASE — OPERATIVE ROSTER")
	fmt.Println("═══════════════════════════════════════")
	fmt.Println()

	// Load from all agent directories (built-in + user + project)
	cfg := config.Load()
	builtIn, user, project, _ := agents.FindAllAgentDirs(cfg.AgentsDir)

	reg := agents.NewRegistry(builtIn)
	if builtIn != "" || user != "" || project != "" {
		reg.LoadMultipleDirsSync(builtIn, user, project)
	} else {
		reg.Reload()
	}
	all := reg.All()

	if len(all) > 0 {
		// Group by pipeline position: core (1-5) vs specialists
		fmt.Println("  SECTOR V")
		for _, a := range all {
			if a.PipelinePosition != nil && *a.PipelinePosition >= 1 && *a.PipelinePosition <= 5 {
				mcpTag := mcpCountTag(a)
				fmt.Printf("  [%s] %-18s %-26s %s%s\n", extractNumbuh(a.Name), a.Designation, a.Role, sourceTag(a.Source), mcpTag)
			}
		}
		// Also include numbuh-0 (pipeline position 0 or architect role)
		for _, a := range all {
			if a.PipelinePosition != nil && *a.PipelinePosition == 0 {
				mcpTag := mcpCountTag(a)
				fmt.Printf("  [%s] %-18s %-26s %s%s\n", extractNumbuh(a.Name), a.Designation, a.Role, sourceTag(a.Source), mcpTag)
			}
		}

		fmt.Println()
		fmt.Println("  SPECIALISTS")
		for _, a := range all {
			if a.PipelinePosition == nil || *a.PipelinePosition > 5 {
				num := extractNumbuh(a.Name)
				if num != "" {
					mcpTag := mcpCountTag(a)
					fmt.Printf("  [%s] %-18s %-26s %s%s\n", num, a.Designation, a.Role, sourceTag(a.Source), mcpTag)
				}
			}
		}
	} else {
		// Fallback to hardcoded if registry returns empty
		fmt.Println("  ⚠️  Could not load agents from registry, showing defaults")
		fmt.Println()
		fmt.Println("  SECTOR V")

		hardcodedAgents := []struct {
			num, name, role string
		}{
			{"0", "Monty Uno", "System Architect"},
			{"1", "Nigel Uno", "Analyst"},
			{"2", "Hoagie Gilligan", "Architect"},
			{"3", "Kuki Sanban", "Implementer"},
			{"4", "Wallabee Beatles", "QA"},
			{"5", "Abigail Lincoln", "Reviewer"},
		}

		for _, a := range hardcodedAgents {
			fmt.Printf("  [%s] %-18s %s\n", a.num, a.name, a.role)
		}

		fmt.Println()
		fmt.Println("  SPECIALISTS")

		hardcodedSpecialists := []struct {
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

		for _, a := range hardcodedSpecialists {
			fmt.Printf("  [%s] %-18s %s\n", a.num, a.name, a.role)
		}
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
	if os.Getenv("OPENAI_API_KEY") != "" {
		fmt.Println("  ✓ openai")
	}
	if os.Getenv("ANTHROPIC_API_KEY") != "" {
		fmt.Println("  ✓ anthropic")
	}
	if os.Getenv("MOONSHOT_API_KEY") != "" {
		fmt.Println("  ✓ kimi")
	}
	fmt.Println()
}

// sourceTag returns a display tag for the agent source.
func sourceTag(source string) string {
	switch source {
	case agents.SourceUser:
		return "[user]"
	case agents.SourceProject:
		return "[project]"
	case agents.SourceBuiltIn:
		return "[built-in]"
	default:
		return ""
	}
}

// mcpCountTag returns a display tag showing MCP server count, or empty string if none.
func mcpCountTag(a agents.Agent) string {
	if a.HasMCPServers() {
		return fmt.Sprintf(" [%d MCP]", len(a.MCPServers))
	}
	return ""
}

// extractNumbuh extracts the display number/identifier from an agent name.
// e.g., "numbuh-4" → "4", "sector-z" → "Z", "knd-council" → "K"
func extractNumbuh(name string) string {
	switch {
	case strings.HasPrefix(name, "numbuh-"):
		return strings.TrimPrefix(name, "numbuh-")
	case name == "sector-z":
		return "Z"
	case name == "knd-council":
		return "K"
	default:
		return name
	}
}

func isTerminal() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return true
	}
	return fi.Mode()&os.ModeCharDevice != 0
}



// agentsDir locates the agents directory using the shared resolver.
// Exits with an error if no agents directory can be found.
func agentsDir() string {
	cfg := config.Load()
	dir, err := agents.FindAgentsDir(cfg.AgentsDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "❌ Cannot find agents directory.")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "   To fix, try one of:")
		fmt.Fprintln(os.Stderr, "     • Run 'moonbase setup' to install agents globally (~/.moonbase/agents/)")
		fmt.Fprintln(os.Stderr, "     • Run 'make install' from the moonbase source directory")
		fmt.Fprintln(os.Stderr, "     • Run 'moonbase install --all' to install agents into this project")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "   Run 'moonbase status' for diagnostics")
		osExit(1)
	}
	return dir
}

// loadAgentRegistry creates and loads an agent registry from the default agents directory.
// This is the shared helper for all commands that need the agent registry.
func loadAgentRegistry() *agents.Registry {
	dir := agentsDir()
	reg := agents.NewRegistry(dir)
	if err := reg.LoadSync(); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to load agents: %v\n", err)
		osExit(1)
	}
	return reg
}

// runDeploy deploys a single operative by numbuh to an interactive AI session.
func runDeploy(numbuh string, taskArg string) {
	dir := agentsDir()

	// SECURITY: Validate agent identifier — prevents path traversal via ../
	if !isValidAgentID(numbuh) {
		fmt.Fprintf(os.Stderr, "❌ Invalid agent identifier: %s\n", numbuh)
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "   Valid identifiers:\n")
		fmt.Fprintf(os.Stderr, "     Core:        0, 1, 2, 3, 4, 5\n")
		fmt.Fprintf(os.Stderr, "     Specialists: 9, 13, 86, 274, 362, 999\n")
		fmt.Fprintf(os.Stderr, "     Special:     z (Sector Z), council (KND Council)\n")
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "   Try: moonbase deploy 1 \"your task here\"\n")
		fmt.Fprintf(os.Stderr, "   Or:  moonbase deploy   (interactive picker)\n")
		osExit(1)
	}

	// Check if there's a task argument after the numbuh
	var task string
	if deployTask != "" {
		// --task/-t flag takes priority
		task = deployTask
	} else if taskArg != "" {
		// Task passed from cobra args
		task = taskArg
	}

	// Support pipe mode: read stdin as task if not a terminal and no task yet
	if task == "" && !isTerminal() {
		limited := io.LimitReader(os.Stdin, maxPipeInputSize)
		input, _ := io.ReadAll(limited)
		pipeTask := strings.TrimSpace(string(input))
		if pipeTask != "" {
			task = pipeTask
		}
	}

	// Resolve agent file name from input
	// SECURITY: Only filepath.Base-safe names are constructed here because
	// isValidAgentID restricts to [a-zA-Z0-9-], preventing directory traversal.
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
		fmt.Fprintf(os.Stderr, "❌ Agent not found: %s\n", numbuh)
		fmt.Fprintf(os.Stderr, "   Looked for: %s\n\n", agentFile)
		fmt.Fprintf(os.Stderr, "   To fix:\n")
		fmt.Fprintf(os.Stderr, "     • Run 'moonbase install --all' to install agents\n")
		fmt.Fprintf(os.Stderr, "     • Or run 'moonbase list' to see available operatives\n")
		fmt.Fprintf(os.Stderr, "     • Or run 'moonbase deploy' for interactive picker\n")
		osExit(1)
	}

	// Discover project context
	cwd := mustGetwd()
	ctx := discovery.Discover(cwd)

	fmt.Printf("🌙 Deploying %s — %s (%s)\n", agent.Name, agent.Designation, agent.Role)
	if ctx.HasSpecs() || ctx.HasSteering() {
		fmt.Printf("   Context: %s\n", ctx.Summary())
	}
	if task != "" {
		fmt.Printf("   Task: %s\n", task)
	}
	fmt.Println()

	// Compose prompt with project context and task
	composed := discovery.ComposePrompt(agent.Prompt, ctx, task)

	// If --pane/--cmux set, deploy the agent in a split pane of the active
	// terminal multiplexer (tmux or cmux).
	if deployPane || deployCmux {
		m := mux.Detect()
		if !m.Available() {
			fmt.Fprintln(os.Stderr, "❌ No terminal multiplexer available.")
			fmt.Fprintln(os.Stderr, "   Install tmux (Linux) or cmux (macOS: https://github.com/manaflow-ai/cmux).")
			osExit(1)
		}
		if m.Kind == mux.Tmux && !m.InSession() {
			fmt.Fprintln(os.Stderr, "❌ Not inside a tmux session — cannot split.")
			fmt.Fprintln(os.Stderr, "   Start/attach one first:  tmux new -s moonbase")
			osExit(1)
		}
		// Build the command to run in the pane.
		kiroArgs := []string{"kiro-cli", "chat"}
		localAgent := filepath.Join(cwd, ".kiro", "agents", agent.Name+".md")
		if _, statErr := os.Stat(localAgent); statErr == nil {
			kiroArgs = append(kiroArgs, "--agent", agent.Name)
		}
		if task != "" {
			kiroArgs = append(kiroArgs, task)
		}
		shellCmd := strings.Join(kiroArgs, " ")
		if err := m.SplitRun(mux.Right, shellCmd); err != nil {
			fmt.Fprintf(os.Stderr, "❌ %s split failed: %v\n", m.Name(), err)
			osExit(1)
		}
		fmt.Printf("✅ Deployed %s in a %s split pane\n", agent.Name, m.Name())
		return
	}

	// Native deploy: use Kiro's compiled JSON agent (--native flag or config deploy.mode=native)
	cfg := config.Load()
	useNative := deployNative || cfg.Deploy.Mode == "native"
	if useNative {
		if kiro, kErr := exec.LookPath("kiro-cli"); kErr == nil {
			// Staleness check: warn if compiled JSON is older than source .md
			compiledDir := cfg.Compile.OutDir
			if compiledDir == "" {
				compiledDir = ".kiro/agents"
			}
			if !filepath.IsAbs(compiledDir) {
				compiledDir = filepath.Join(cwd, compiledDir)
			}
			compiledJSON := filepath.Join(compiledDir, agent.Name+".json")

			if _, statErr := os.Stat(compiledJSON); os.IsNotExist(statErr) {
				fmt.Fprintf(os.Stderr, "   ⚠️  Compiled JSON not found: %s\n", compiledJSON)
				fmt.Fprintf(os.Stderr, "      Run 'moonbase compile' first.\n")
				osExit(1)
			}

			// Build native deploy args
			// SECURITY: When safety.delegate_to_kiro is true and native mode is active,
			// SafeEnv is intentionally NOT applied. Kiro's engine handles env isolation,
			// shell permissions, and write path enforcement via the compiled toolsSettings.
			kiroBackend := &backend.Kiro{TrustTools: cfg.TrustTools}
			nativeArgs, nErr := kiroBackend.DeployNative(agent.Name)
			if nErr != nil {
				fmt.Fprintf(os.Stderr, "❌ Native deploy failed: %v\n", nErr)
				osExit(1)
			}

			// Add task as positional input if provided
			if task != "" {
				nativeArgs = append(nativeArgs, task)
			}

			fmt.Printf("   Mode: native (kiro-cli chat --agent %s)\n\n", agent.Name)

			if cfg.Safety.DelegateToKiro {
				// SAFETY DELEGATION: Kiro's engine owns shell/write/hook/env enforcement.
				// moonbase only retains pipeline orchestration, routing, and guardrails.
				execErr := execSyscall(kiro, nativeArgs, nil)
				if execErr != nil {
					fmt.Fprintf(os.Stderr, "   ⚠️  kiro-cli native exec failed: %v\n", execErr)
				}
			} else {
				execErr := execSyscall(kiro, nativeArgs, backend.SafeEnv())
				if execErr != nil {
					fmt.Fprintf(os.Stderr, "   ⚠️  kiro-cli native exec failed: %v\n", execErr)
				}
			}
			return
		}
		fmt.Fprintln(os.Stderr, "   ⚠️  --native requires kiro-cli. Falling back to legacy deploy.")
	}

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

		// SECURITY: syscall.Exec uses SafeEnv — only allowlisted env vars are passed.
		// This replaces this process — full TTY to kiro-cli.
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

// runConfig displays the current moonbase configuration.
func runConfig() {
	cfg := config.Load()
	fmt.Println("🌙 Moonbase Configuration")
	fmt.Printf("   Path: %s\n\n", config.Path())
	fmt.Println(config.Show(cfg))
}

// execSyscall replaces the current process with the given command.
// This gives the target program full terminal control (TTY, colours, readline).
// SECURITY: Always called with SafeEnv() — never passes full os.Environ().
func execSyscall(binary string, args []string, env []string) error {
	return syscall.Exec(binary, args, env)
}

// isValidAgentID checks that an agent identifier contains only safe characters.
// Prevents path traversal attacks via deploy command (CWE-22).
// Only allows: [a-zA-Z0-9-], max 20 chars.
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
