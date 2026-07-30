package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/rk-senne/moonbase/internal/agents"
	"github.com/rk-senne/moonbase/internal/compile"
)

// runCompile implements `moonbase compile` — translates .md agents to Kiro-native JSON.
func runCompile() {
	dir := agentsDir()
	reg := agents.NewRegistry(dir)
	if err := reg.LoadSync(); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to load agents: %v\n", err)
		osExit(1)
	}

	// Determine which agents to compile
	var toCompile []agents.Agent
	if compileAgent != "" {
		a := reg.GetByName(compileAgent)
		if a == nil {
			fmt.Fprintf(os.Stderr, "❌ Agent %q not found.\n", compileAgent)
			fmt.Fprintf(os.Stderr, "   Run 'moonbase list' to see available agents.\n")
			osExit(1)
			return // unreachable, but satisfies staticcheck nil analysis
		}
		toCompile = []agents.Agent{*a}
	} else {
		toCompile = reg.All()
	}

	if len(toCompile) == 0 {
		fmt.Fprintln(os.Stderr, "❌ No agents found to compile.")
		osExit(1)
	}

	// Resolve output directory (relative to cwd)
	outDir := compileOut
	if !filepath.IsAbs(outDir) {
		cwd := mustGetwd()
		outDir = filepath.Join(cwd, outDir)
	}

	fmt.Println("🌙 Moonbase Compile → Kiro Native JSON")
	fmt.Printf("   Target: %s\n", outDir)
	fmt.Println()

	// Compile each agent
	errors := 0
	for _, agent := range toCompile {
		ka, promptBody, err := compile.Compile(agent)
		if err != nil {
			fmt.Printf("   ❌ %s: compile error: %v\n", agent.Name, err)
			errors++
			continue
		}

		if err := compile.WriteAgent(ka, promptBody, outDir); err != nil {
			fmt.Printf("   ❌ %s: write error: %v\n", agent.Name, err)
			errors++
			continue
		}

		fmt.Printf("   ✅ %s → %s.json\n", agent.Name, agent.Name)
	}

	fmt.Println()

	// Validate if requested
	if compileValidate {
		runCompileValidation(outDir, toCompile)
	}

	if errors > 0 {
		fmt.Printf("   %d error(s) during compilation.\n", errors)
		osExit(1)
	}

	fmt.Printf("   Compiled %d agent(s).\n", len(toCompile)-errors)
}

// runCompileValidation invokes kiro-cli agent validate on each compiled JSON.
func runCompileValidation(outDir string, compiled []agents.Agent) {
	kiroCLI, err := exec.LookPath("kiro-cli")
	if err != nil {
		fmt.Println("   ⚠️  kiro-cli not found — skipping validation.")
		fmt.Println("      Install kiro-cli to enable schema validation.")
		return
	}

	fmt.Println("   Validating with kiro-cli...")
	fmt.Println()

	validateErrors := 0
	for _, agent := range compiled {
		jsonPath := filepath.Join(outDir, agent.Name+".json")
		if _, err := os.Stat(jsonPath); err != nil {
			continue // Skip if file wasn't written (compile error)
		}

		cmd := exec.Command(kiroCLI, "agent", "validate", "--path", jsonPath)
		output, err := cmd.CombinedOutput()
		if err != nil {
			fmt.Printf("   ❌ %s: validation failed\n", agent.Name)
			if len(output) > 0 {
				fmt.Printf("      %s\n", string(output))
			}
			validateErrors++
		} else {
			if len(output) > 0 {
				// kiro-cli prints nothing on success
				outStr := string(output)
				if outStr != "" {
					fmt.Printf("   ⚠️  %s: %s\n", agent.Name, outStr)
				}
			} else {
				fmt.Printf("   ✓ %s: valid\n", agent.Name)
			}
		}
	}

	fmt.Println()
	if validateErrors > 0 {
		fmt.Printf("   %d agent(s) failed validation.\n", validateErrors)
		osExit(1)
	}
	fmt.Printf("   All %d agent(s) validated successfully.\n", len(compiled))
}
