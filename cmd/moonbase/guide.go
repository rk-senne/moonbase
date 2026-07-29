package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/rk-senne/moonbase/internal/agents"
	"github.com/rk-senne/moonbase/internal/config"
)

// runGuideOverview shows the general operations manual.
func runGuideOverview() {
	fmt.Print(`🌙 MOONBASE OPERATIONS GUIDE
═══════════════════════════════

QUICK START
  moonbase setup                       Install agents globally (run once)
  moonbase init                        Initialize a project
  moonbase deploy <n> "task"           Deploy a single operative
  moonbase mission "task"              Run the full KND Council pipeline

SINGLE AGENT DEPLOYMENT
  moonbase deploy 1 "break this down"  Analyst — requirements & scope
  moonbase deploy 2 "design approach"  Architect — blueprints & trade-offs
  moonbase deploy 3 "implement this"   Implementer — writes code & tests
  moonbase deploy 4 "verify changes"   QA — verify & risk-gate
  moonbase deploy 5 "review for PR"    Reviewer — final gate & PR prep

SPECIALIST DEPLOYMENT
  moonbase deploy 0 "review arch"      Overseer — architecture review
  moonbase deploy 274 "audit security" Security — red-team & vuln scan
  moonbase deploy 362 "fix CI"         DevOps — infra & deployment
  moonbase deploy 86 "clean dead code" Tech Debt — decommission & cleanup
  moonbase deploy 999 "write docs"     Documentation — ADRs & changelogs
  moonbase deploy 13 "edge cases"      Chaos — weird inputs & edge cases
  moonbase deploy 9 "upgrade lib"      Migration — upgrades & breaking changes
  moonbase deploy z "explain this"     Legacy — archaeology & context recovery

FULL PIPELINE
  moonbase mission "add feature X"
  Pipeline: Analyst → Architect → Implementer → QA → Reviewer
  Risk gate after QA: LOW=proceed, MEDIUM=rework, HIGH=redesign, CRITICAL=stop

PIPE MODE
  echo "fix bug" | moonbase            Pipe to full council
  echo "check" | moonbase deploy 274   Pipe to specific agent
  git diff | moonbase deploy 4         Pipe diff to QA

TIPS
  • Run 'moonbase guide <n>' for detailed guide on any agent
  • Run 'moonbase guide --all' to see all agent guides
  • Spec non-trivial work first (agents work better with specs)
  • Agents auto-discover .kiro/specs/ and .kiro/steering/
  • Run 'moonbase update' to self-update to the latest release

`)
}

// runGuideAgent shows a detailed guide for a specific agent.
func runGuideAgent(id string) {
	// Validate the agent ID
	if !isValidAgentID(id) {
		fmt.Fprintf(os.Stderr, "❌ Invalid agent identifier: %s\n", id)
		fmt.Fprintf(os.Stderr, "   Try: moonbase guide 1, moonbase guide 274, moonbase guide z\n")
		osExit(1)
	}

	// Load agent registry
	cfg := config.Load()
	dir, err := agents.FindAgentsDir(cfg.AgentsDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ %v\n", err)
		osExit(1)
	}

	reg := agents.NewRegistry(dir)
	reg.Reload()

	// Resolve agent name from numbuh
	name := resolveAgentName(id)
	agent := reg.GetByName(name)
	if agent == nil {
		fmt.Fprintf(os.Stderr, "❌ Agent not found: %s\n", id)
		fmt.Fprintf(os.Stderr, "   Run 'moonbase list' to see available operatives.\n")
		osExit(1)
	}

	printAgentGuide(agent)
}

// runGuideAll shows guides for all agents.
func runGuideAll() {
	cfg := config.Load()
	dir, err := agents.FindAgentsDir(cfg.AgentsDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ %v\n", err)
		osExit(1)
	}

	reg := agents.NewRegistry(dir)
	reg.Reload()
	all := reg.All()

	if len(all) == 0 {
		fmt.Fprintln(os.Stderr, "❌ No agents found. Run 'moonbase setup' first.")
		osExit(1)
	}

	fmt.Println("🌙 MOONBASE — FULL AGENT GUIDE")
	fmt.Println("═══════════════════════════════════")
	fmt.Println()

	for i, a := range all {
		printAgentGuide(&a)
		if i < len(all)-1 {
			fmt.Println("───────────────────────────────────────────────────────────────────")
			fmt.Println()
		}
	}
}

// printAgentGuide prints a formatted usage guide for a single agent.
func printAgentGuide(a *agents.Agent) {
	numbuh := extractNumbuh(a.Name)
	deployCmd := fmt.Sprintf("moonbase deploy %s", numbuh)

	fmt.Printf("🌙 %s — %s\n", strings.ToUpper(a.Name), a.Designation)
	fmt.Printf("   Role: %s\n", a.Role)
	fmt.Println()

	// Description
	if a.Description != "" {
		fmt.Printf("   %s\n", a.Description)
		fmt.Println()
	}

	// When to use
	fmt.Println("   WHEN TO USE")
	printWhenToUse(a)
	fmt.Println()

	// How to deploy
	fmt.Println("   HOW TO USE")
	fmt.Printf("     %s \"your task\"\n", deployCmd)
	fmt.Printf("     echo \"task\" | %s\n", deployCmd)
	fmt.Println()

	// Example tasks
	fmt.Println("   EXAMPLE TASKS")
	printExampleTasks(a)
	fmt.Println()

	// Tools available
	fmt.Println("   TOOLS")
	fmt.Printf("     %s\n", strings.Join(a.Tools, ", "))
	fmt.Println()

	// Shell commands (if any)
	if a.Shell != nil && len(a.Shell.AllowedCommands) > 0 {
		fmt.Println("   SHELL COMMANDS")
		for _, cmd := range a.Shell.AllowedCommands {
			fmt.Printf("     • %s\n", cmd)
		}
		if a.Shell.ReadOnly {
			fmt.Println("     (read-only mode)")
		}
		fmt.Println()
	}

	// Pipeline info
	if a.IsPipeline() {
		fmt.Printf("   PIPELINE POSITION: %d of 5 (core pipeline)\n", *a.PipelinePosition)
		fmt.Println()
	} else if a.IsConditional() {
		fmt.Println("   TYPE: Conditional specialist (auto-triggers in pipeline)")
		fmt.Println()
	}

	// Routing
	if a.Routing != nil && len(a.Routing.Available) > 0 {
		fmt.Println("   HANDS OFF TO")
		fmt.Printf("     %s\n", strings.Join(a.Routing.Available, ", "))
		fmt.Println()
	}
}

// resolveAgentName converts a numbuh/shorthand to the full agent name.
func resolveAgentName(id string) string {
	switch {
	case id == "council" || id == "k":
		return "knd-council"
	case id == "z" || id == "Z":
		return "sector-z"
	default:
		return "numbuh-" + id
	}
}

// printWhenToUse prints context-appropriate "when to use" guidance per agent role.
func printWhenToUse(a *agents.Agent) {
	switch a.Name {
	case "numbuh-1":
		fmt.Println("     • You have a vague idea that needs structure")
		fmt.Println("     • You need acceptance criteria before building")
		fmt.Println("     • You want to scope and risk-assess a feature")
		fmt.Println("     • You need to break an epic into tasks")
	case "numbuh-2":
		fmt.Println("     • You know WHAT to build but not HOW")
		fmt.Println("     • You need to choose between approaches")
		fmt.Println("     • You want to know which files will be affected")
		fmt.Println("     • You need a design doc or ADR")
	case "numbuh-3":
		fmt.Println("     • You have a clear task and want code written")
		fmt.Println("     • You need tests alongside implementation")
		fmt.Println("     • You want to follow an existing design/spec")
		fmt.Println("     • You need a bug fixed with tests to prove it")
	case "numbuh-4":
		fmt.Println("     • You want to verify recent changes work")
		fmt.Println("     • You need someone to run the test suite and interpret results")
		fmt.Println("     • You want to check acceptance criteria are met")
		fmt.Println("     • You need a risk assessment before merging")
	case "numbuh-5":
		fmt.Println("     • Code is ready and you want a review")
		fmt.Println("     • You need a PR description written")
		fmt.Println("     • You want final checks before merging")
		fmt.Println("     • You need someone to assess code quality")
	case "numbuh-0":
		fmt.Println("     • A change touches many files (>5)")
		fmt.Println("     • Core patterns or architecture are being modified")
		fmt.Println("     • You want a high-level sanity check")
		fmt.Println("     • You're introducing a new pattern to the codebase")
	case "numbuh-274":
		fmt.Println("     • Auth, secrets, or permissions code was touched")
		fmt.Println("     • New API endpoints were added")
		fmt.Println("     • You want a security audit or pen-test review")
		fmt.Println("     • Dependencies need CVE checking")
	case "numbuh-362":
		fmt.Println("     • CI/CD pipelines need fixing or updating")
		fmt.Println("     • Docker/container configs changed")
		fmt.Println("     • Environment variables or deploy configs modified")
		fmt.Println("     • You need infrastructure-as-code reviewed")
	case "numbuh-86":
		fmt.Println("     • You suspect dead code or unused dependencies")
		fmt.Println("     • You want to clean up after a feature removal")
		fmt.Println("     • You need to find zombie features")
		fmt.Println("     • Technical debt is piling up")
	case "numbuh-999":
		fmt.Println("     • README needs updating after changes")
		fmt.Println("     • You need an ADR (Architecture Decision Record)")
		fmt.Println("     • CHANGELOG needs a new entry")
		fmt.Println("     • API documentation is out of date")
	case "numbuh-13":
		fmt.Println("     • You want to find edge cases that break things")
		fmt.Println("     • Parsers, validators, or input handling needs stress-testing")
		fmt.Println("     • You want to know what happens with weird/malformed input")
		fmt.Println("     • Fragile flows need chaos testing")
	case "numbuh-9":
		fmt.Println("     • A library/framework needs upgrading")
		fmt.Println("     • Breaking changes from a dependency update")
		fmt.Println("     • Database migrations are needed")
		fmt.Println("     • You're moving between major versions")
	case "sector-z":
		fmt.Println("     • You're touching old, undocumented code")
		fmt.Println("     • You need to understand legacy context")
		fmt.Println("     • Historical decisions need archaeology")
		fmt.Println("     • Mysterious behaviour needs explaining")
	case "knd-council":
		fmt.Println("     • Full feature lifecycle (requirements → code → review)")
		fmt.Println("     • You want the pipeline to handle everything")
		fmt.Println("     • Complex task that needs multiple perspectives")
		fmt.Println("     • Use 'moonbase mission' instead of direct deploy")
	default:
		fmt.Printf("     • Use for: %s\n", a.Role)
	}
}

// printExampleTasks prints example tasks appropriate for each agent.
func printExampleTasks(a *agents.Agent) {
	switch a.Name {
	case "numbuh-1":
		fmt.Println("     moonbase deploy 1 \"we need user authentication\"")
		fmt.Println("     moonbase deploy 1 \"spec out the notification system\"")
		fmt.Println("     moonbase deploy 1 \"break down the search feature into tasks\"")
	case "numbuh-2":
		fmt.Println("     moonbase deploy 2 \"design the caching layer\"")
		fmt.Println("     moonbase deploy 2 \"how should we structure the API versioning?\"")
		fmt.Println("     moonbase deploy 2 \"compare REST vs GraphQL for this use case\"")
	case "numbuh-3":
		fmt.Println("     moonbase deploy 3 \"implement the pagination helper\"")
		fmt.Println("     moonbase deploy 3 \"fix the race condition in the queue worker\"")
		fmt.Println("     moonbase deploy 3 \"add unit tests for the auth middleware\"")
	case "numbuh-4":
		fmt.Println("     moonbase deploy 4 \"verify the last 3 commits\"")
		fmt.Println("     moonbase deploy 4 \"run tests and check coverage\"")
		fmt.Println("     moonbase deploy 4 \"check AC-1.1 through AC-1.5 pass\"")
	case "numbuh-5":
		fmt.Println("     moonbase deploy 5 \"review the auth PR for merge\"")
		fmt.Println("     moonbase deploy 5 \"write a PR description for the pagination changes\"")
		fmt.Println("     moonbase deploy 5 \"final check before release\"")
	case "numbuh-0":
		fmt.Println("     moonbase deploy 0 \"review architecture of the new plugin system\"")
		fmt.Println("     moonbase deploy 0 \"is this refactor consistent with our patterns?\"")
		fmt.Println("     moonbase deploy 0 \"audit the module boundaries\"")
	case "numbuh-274":
		fmt.Println("     moonbase deploy 274 \"audit the auth middleware\"")
		fmt.Println("     moonbase deploy 274 \"check for injection vulnerabilities\"")
		fmt.Println("     moonbase deploy 274 \"scan dependencies for CVEs\"")
	case "numbuh-362":
		fmt.Println("     moonbase deploy 362 \"fix the GitHub Actions workflow\"")
		fmt.Println("     moonbase deploy 362 \"add Docker multi-stage build\"")
		fmt.Println("     moonbase deploy 362 \"set up staging environment\"")
	case "numbuh-86":
		fmt.Println("     moonbase deploy 86 \"find and remove dead code\"")
		fmt.Println("     moonbase deploy 86 \"audit unused dependencies\"")
		fmt.Println("     moonbase deploy 86 \"clean up after removing the legacy API\"")
	case "numbuh-999":
		fmt.Println("     moonbase deploy 999 \"update README with new API endpoints\"")
		fmt.Println("     moonbase deploy 999 \"write ADR for the database switch\"")
		fmt.Println("     moonbase deploy 999 \"generate changelog for v2.1\"")
	case "numbuh-13":
		fmt.Println("     moonbase deploy 13 \"fuzz the JSON parser with edge cases\"")
		fmt.Println("     moonbase deploy 13 \"what happens if the DB is down mid-request?\"")
		fmt.Println("     moonbase deploy 13 \"test the form with every weird Unicode input\"")
	case "numbuh-9":
		fmt.Println("     moonbase deploy 9 \"migrate from Express 4 to Express 5\"")
		fmt.Println("     moonbase deploy 9 \"upgrade React from 17 to 18\"")
		fmt.Println("     moonbase deploy 9 \"plan the PostgreSQL 14→16 upgrade\"")
	case "sector-z":
		fmt.Println("     moonbase deploy z \"explain what this legacy billing module does\"")
		fmt.Println("     moonbase deploy z \"why was this workaround added in 2019?\"")
		fmt.Println("     moonbase deploy z \"document the undocumented config flags\"")
	case "knd-council":
		fmt.Println("     moonbase mission \"add pagination to the /users endpoint\"")
		fmt.Println("     moonbase mission \"implement rate limiting\"")
		fmt.Println("     moonbase mission \"add OAuth2 login flow\"")
	default:
		fmt.Printf("     moonbase deploy %s \"your task here\"\n", extractNumbuh(a.Name))
	}
}
