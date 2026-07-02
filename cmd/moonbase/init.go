package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/f5508037/moonbase/internal/discovery"
)

// runInit scaffolds a .kiro/ directory in the current project.
func runInit() {
	cwd, _ := os.Getwd()

	fmt.Println("🌙 Moonbase Init — Setting up project for agent-driven development")
	fmt.Println()

	// Check if already initialized
	kiroDir := filepath.Join(cwd, ".kiro")
	if _, err := os.Stat(kiroDir); err == nil {
		fmt.Println("   ⚠️  .kiro/ already exists in this project.")
		fmt.Println("   Use 'moonbase install --all' to update agents.")
		return
	}

	// Detect project stack
	ctx, _ := discovery.Discover(cwd)
	stack := "unknown"
	buildTool := ""
	testCmd := ""
	if ctx != nil && ctx.Stack.Language != "" {
		stack = ctx.Stack.Language
		buildTool = ctx.Stack.BuildTool
		testCmd = ctx.Stack.TestCommand
	}

	// Create directory structure
	dirs := []string{
		filepath.Join(kiroDir, "specs", "_templates"),
		filepath.Join(kiroDir, "steering"),
		filepath.Join(kiroDir, "agents"),
	}
	for _, d := range dirs {
		os.MkdirAll(d, 0o755)
	}

	// Write spec templates
	writeTemplate(filepath.Join(kiroDir, "specs", "_templates", "requirements.md"), requirementsTemplate)
	writeTemplate(filepath.Join(kiroDir, "specs", "_templates", "design.md"), designTemplate)
	writeTemplate(filepath.Join(kiroDir, "specs", "_templates", "tasks.md"), tasksTemplate)

	// Write steering rules (auto-detected from stack)
	steeringContent := generateSteering(stack, buildTool, testCmd)
	writeTemplate(filepath.Join(kiroDir, "steering", "dev-rules.md"), steeringContent)

	// Install agents
	agentsSource, err := findAgentsSource()
	if err == nil {
		files, _ := filepath.Glob(filepath.Join(agentsSource, "*.md"))
		for _, f := range files {
			copyFile(f, filepath.Join(kiroDir, "agents", filepath.Base(f)))
		}
		fmt.Printf("   ✅ Installed %d agents to .kiro/agents/\n", len(files))
	} else {
		fmt.Println("   ⚠️  Could not find moonbase agents to install")
	}

	fmt.Println("   ✅ Created .kiro/specs/_templates/")
	fmt.Printf("   ✅ Created .kiro/steering/dev-rules.md (detected: %s)\n", stack)
	fmt.Println()
	fmt.Println("   Next steps:")
	fmt.Println("   1. Edit .kiro/steering/dev-rules.md with your project conventions")
	fmt.Println("   2. Create a spec: cp .kiro/specs/_templates .kiro/specs/my-feature")
	fmt.Println("   3. Deploy an agent: moonbase deploy 1 \"analyze this project\"")
	fmt.Println()
	fmt.Println("   ✨ Project is now agent-ready.")
}

func writeTemplate(path, content string) {
	os.WriteFile(path, []byte(content), 0o644)
}

func generateSteering(stack, buildTool, testCmd string) string {
	var b strings.Builder
	b.WriteString("# Dev Rules\n\n")
	b.WriteString("## Stack\n")
	if stack != "unknown" {
		b.WriteString(fmt.Sprintf("- Language: %s\n", stack))
		if buildTool != "" {
			b.WriteString(fmt.Sprintf("- Build tool: %s\n", buildTool))
		}
		if testCmd != "" {
			b.WriteString(fmt.Sprintf("- Test command: `%s`\n", testCmd))
		}
	} else {
		b.WriteString("- Language: (detected: unknown — update this)\n")
		b.WriteString("- Build tool: \n")
		b.WriteString("- Test command: \n")
	}
	b.WriteString("\n## Conventions\n")
	b.WriteString("- Follow existing code patterns before introducing new ones\n")
	b.WriteString("- Tests ship with the code, in the same unit of work\n")
	b.WriteString("- Prefer small, focused changes over large rewrites\n")
	b.WriteString("- Every change needs a rollback path\n")
	return b.String()
}

const requirementsTemplate = `# Requirements: {Feature Name}

## User Story
As a {role}, I want to {action}, so that {benefit}.

## Acceptance Criteria

### AC-1.1: {Name}
- **WHEN** {trigger}
- **THEN** {expected behaviour}
- **SHALL** {constraint}

## Scope
### In Scope
### Out of Scope

## Risks
`

const designTemplate = `# Design: {Feature Name}

## Approach
{What and why}

## Files Affected
| File | Change | Purpose |
|------|--------|---------|

## Data Flow
{Step 1 → Step 2 → Step 3}

## Edge Cases
| Scenario | Handling |
|----------|----------|
`

const tasksTemplate = `# Tasks: {Feature Name}

## Task 1: {Description}
- **Requirements:** AC-1.1
- **Files:** ` + "`path/to/file`" + `
- **Action:** {What to do}
- **Test:** {How to verify}
- **Status:** pending

## Checkpoint
- [ ] All tasks complete
- [ ] Tests passing
- [ ] Build succeeds
`
