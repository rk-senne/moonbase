package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	moonbase "github.com/rk-senne/moonbase"
	"github.com/rk-senne/moonbase/internal/discovery"
	"github.com/rk-senne/moonbase/internal/templates"
)

// runInit scaffolds a .kiro/ directory in the current project.
func runInit() {
	cwd := mustGetwd()

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
	ctx := discovery.Discover(cwd)
	stack := "unknown"
	buildTool := ""
	testCmd := ""
	if ctx.Stack.Language != "" {
		stack = ctx.Stack.Language
		buildTool = ctx.Stack.BuildTool
		testCmd = ctx.Stack.TestCommand
	}

	// Create directory structure
	dirs := []string{
		filepath.Join(kiroDir, "specs", "_templates"),
		filepath.Join(kiroDir, "steering"),
		filepath.Join(kiroDir, "agents"),
		filepath.Join(kiroDir, "skills"),
		filepath.Join(kiroDir, "prompts"),
	}
	for _, d := range dirs {
		os.MkdirAll(d, 0o755)
	}

	// Write spec templates
	if err := writeTemplate(filepath.Join(kiroDir, "specs", "_templates", "requirements.md"), requirementsTemplate); err != nil {
		fmt.Printf("   ⚠️  Failed to write requirements.md template: %v\n", err)
	}
	if err := writeTemplate(filepath.Join(kiroDir, "specs", "_templates", "design.md"), designTemplate); err != nil {
		fmt.Printf("   ⚠️  Failed to write design.md template: %v\n", err)
	}
	if err := writeTemplate(filepath.Join(kiroDir, "specs", "_templates", "tasks.md"), tasksTemplate); err != nil {
		fmt.Printf("   ⚠️  Failed to write tasks.md template: %v\n", err)
	}

	// Write steering rules (tailored to detected stack)
	steeringFiles := templates.GenerateSteeringFiles(stack, buildTool, testCmd)
	for filename, content := range steeringFiles {
		if err := writeTemplate(filepath.Join(kiroDir, "steering", filename), content); err != nil {
			fmt.Printf("   ⚠️  Failed to write steering/%s: %v\n", filename, err)
		}
	}

	// Opt-in: data-access & performance standard (only for projects that need it,
	// via --data-access). It is gitignored regardless, so it stays local.
	dataAccessGenerated := false
	if initWithDataAccess {
		content := templates.GenerateDataAccessPerformance(stack)
		if err := writeTemplate(filepath.Join(kiroDir, "steering", "data-access-performance.md"), content); err != nil {
			fmt.Printf("   ⚠️  Failed to write steering/data-access-performance.md: %v\n", err)
		} else {
			dataAccessGenerated = true
		}
	}

	// Write skills README
	if err := writeTemplate(filepath.Join(kiroDir, "skills", "README.md"), skillsReadme); err != nil {
		fmt.Printf("   ⚠️  Failed to write skills/README.md: %v\n", err)
	}

	// Scaffold curated skills library from embedded files (Kiro-native directory style)
	skillsInstalled, skillsErr := writeEmbeddedSkills(filepath.Join(kiroDir, "skills"))
	if skillsErr != nil {
		fmt.Printf("   ⚠️  Failed to scaffold skills: %v\n", skillsErr)
	}

	// Write prompts README
	if err := writeTemplate(filepath.Join(kiroDir, "prompts", "README.md"), promptsReadme); err != nil {
		fmt.Printf("   ⚠️  Failed to write prompts/README.md: %v\n", err)
	}

	// Install agents — prefer an on-disk repo source, fall back to the agents
	// embedded in the binary so `moonbase init` works from any directory.
	agentsSource, err := findAgentsSource()
	if err == nil {
		files, _ := filepath.Glob(filepath.Join(agentsSource, "*.md"))
		for _, f := range files {
			copyFile(f, filepath.Join(kiroDir, "agents", filepath.Base(f)))
		}
		fmt.Printf("   ✅ Installed %d agents to .kiro/agents/\n", len(files))
	} else if n, eErr := writeEmbeddedAgents(filepath.Join(kiroDir, "agents")); eErr == nil && n > 0 {
		fmt.Printf("   ✅ Installed %d agents to .kiro/agents/ (embedded)\n", n)
	} else {
		fmt.Println("   ⚠️  Could not find moonbase agents to install. Run 'moonbase install' later.")
	}

	// Ensure moonbase-managed artifacts are gitignored — installed agents and
	// generated steering baselines are re-creatable and should not be committed
	// into the host project's repository.
	switch status, gErr := ensureMoonbaseGitignored(cwd); {
	case gErr != nil:
		fmt.Printf("   ⚠️  Could not update .gitignore: %v\n", gErr)
	case status == gitignoreAdded:
		fmt.Println("   ✅ Updated .gitignore (moonbase artifacts)")
	case status == gitignoreCreated:
		fmt.Println("   ✅ Created .gitignore (moonbase artifacts)")
	case status == gitignoreAlreadyPresent:
		fmt.Println("   ✅ .gitignore already covers moonbase artifacts")
	}

	fmt.Println("   ✅ Created .kiro/specs/_templates/")
	fmt.Printf("   ✅ Created .kiro/steering/ (6 files, detected: %s)\n", stack)
	if dataAccessGenerated {
		fmt.Println("   ✅ Created .kiro/steering/data-access-performance.md (opt-in)")
	}
	fmt.Printf("   ✅ Created .kiro/skills/ (%d curated skills)\n", skillsInstalled)
	fmt.Println("   ✅ Created .kiro/prompts/ (reusable workflows)")
	fmt.Println()
	fmt.Println("   Next steps:")
	fmt.Println("   1. Review .kiro/steering/ and tailor to your project conventions")
	fmt.Println("   2. Create a spec: cp .kiro/specs/_templates .kiro/specs/my-feature")
	fmt.Println("   3. Deploy an agent: moonbase deploy 1 \"analyze this project\"")
	fmt.Println()
	fmt.Println("   ✨ Project is now agent-ready.")
}

func writeTemplate(path, content string) error {
	return os.WriteFile(path, []byte(content), 0o644)
}

// gitignoreStatus reports the outcome of ensureMoonbaseGitignored.
type gitignoreStatus int

const (
	gitignoreAlreadyPresent gitignoreStatus = iota // all patterns already ignored, no change made
	gitignoreAdded                                 // one or more patterns appended to an existing .gitignore
	gitignoreCreated                               // a new .gitignore was created with the patterns
)

// moonbaseIgnorePatterns are the .gitignore entries moonbase manages in host
// projects. These artifacts are generated/installed by moonbase and are
// re-creatable (agents via 'moonbase install', steering baselines via
// 'moonbase init'), so they should not be committed into the host repository.
var moonbaseIgnorePatterns = []string{
	".kiro/agents/",
	".kiro/steering/data-access-performance.md",
}

// ensureMoonbaseGitignored makes sure the host project's .gitignore excludes
// every pattern in moonbaseIgnorePatterns. It is idempotent: patterns already
// present (in any of their common forms) are skipped. If .gitignore does not
// exist it is created; otherwise only the missing patterns are appended under a
// labelled section.
func ensureMoonbaseGitignored(projectDir string) (gitignoreStatus, error) {
	gitignorePath := filepath.Join(projectDir, ".gitignore")

	data, err := os.ReadFile(gitignorePath)
	if err != nil && !os.IsNotExist(err) {
		return gitignoreAlreadyPresent, fmt.Errorf("reading .gitignore: %w", err)
	}
	created := os.IsNotExist(err)

	existing := string(data)
	var missing []string
	for _, pattern := range moonbaseIgnorePatterns {
		if !patternAlreadyIgnored(existing, pattern) {
			missing = append(missing, pattern)
		}
	}

	// Nothing to do — every pattern is already ignored.
	if len(missing) == 0 && !created {
		return gitignoreAlreadyPresent, nil
	}

	var b strings.Builder
	b.WriteString(existing)
	if len(existing) > 0 {
		// Ensure a clean newline boundary, then a blank line before our section.
		if !strings.HasSuffix(existing, "\n") {
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}
	b.WriteString("# Moonbase — generated/installed artifacts (re-creatable via moonbase)\n")
	for _, pattern := range missing {
		b.WriteString(pattern + "\n")
	}

	if wErr := os.WriteFile(gitignorePath, []byte(b.String()), 0o644); wErr != nil {
		verb := "appending to"
		if created {
			verb = "creating"
		}
		return gitignoreAlreadyPresent, fmt.Errorf("%s .gitignore: %w", verb, wErr)
	}

	if created {
		return gitignoreCreated, nil
	}
	return gitignoreAdded, nil
}

// patternAlreadyIgnored reports whether the .gitignore content already excludes
// the given pattern. It tolerates common equivalent spellings (with/without
// leading slash or trailing slash, or a trailing wildcard) and ignores comments.
func patternAlreadyIgnored(content, pattern string) bool {
	base := strings.TrimSuffix(strings.TrimPrefix(pattern, "/"), "/")
	equivalents := map[string]bool{
		base:              true,
		base + "/":        true,
		"/" + base:        true,
		"/" + base + "/":  true,
		base + "/*":       true,
		"/" + base + "/*": true,
	}
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if equivalents[trimmed] {
			return true
		}
	}
	return false
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

const skillsReadme = `# Skills

Skills are domain-specific knowledge files that agents reference progressively.
Each skill has YAML frontmatter (name + description) so agents see a lightweight
catalog and load full content on demand with @skill(name).

## Format

` + "```" + `markdown
---
name: my-skill
description: Short description of what this skill provides and when to use it.
---

# Skill Title

Full content here — patterns, examples, rules, etc.
` + "```" + `

## Structure

  skills/
    my-skill/
      SKILL.md          ← directory-style (Kiro-native compatible)
    another-skill.md    ← flat-file style (also supported)

## How Agents Use Skills

1. Agents see the catalog (name + description) in every prompt
2. Agents request specific skills with @skill(name) when needed
3. Full content is injected on demand — saving context window tokens

## Naming Rules

- Lowercase alphanumeric and hyphens only: [a-z0-9-]
- Maximum 64 characters
- Must be unique across all skills
`

const promptsReadme = `# Prompts

Stored prompts are reusable workflows you can invoke by name.
Create .md files here for common tasks.

Example:
  prompts/
    implement.md    # Resume implementation from spec
    review.md       # Run a code review
    diagnose.md     # Debug from an issue report
`

// writeEmbeddedSkills writes each embedded skill .md file into targetDir using
// Kiro-native directory format: <name>/SKILL.md. Returns the count of skills
// written. Each skill is read from the embedded filesystem and written into a
// subdirectory named after the filename stem.
func writeEmbeddedSkills(targetDir string) (int, error) {
	sfs, err := moonbase.SkillsFS()
	if err != nil {
		return 0, fmt.Errorf("loading embedded skills: %w", err)
	}
	entries, err := fs.Glob(sfs, "*.md")
	if err != nil {
		return 0, fmt.Errorf("listing embedded skills: %w", err)
	}

	installed := 0
	for _, name := range entries {
		base := filepath.Base(name)
		// SECURITY: reject any unexpected path components.
		if strings.Contains(base, "..") || strings.ContainsAny(base, `/\`) {
			continue
		}
		stem := strings.TrimSuffix(base, ".md")
		skillDir := filepath.Join(targetDir, stem)
		if err := os.MkdirAll(skillDir, 0o755); err != nil {
			return installed, fmt.Errorf("creating skill dir %s: %w", stem, err)
		}
		data, rerr := fs.ReadFile(sfs, name)
		if rerr != nil {
			return installed, fmt.Errorf("reading embedded skill %s: %w", base, rerr)
		}
		if werr := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), data, 0o644); werr != nil {
			return installed, fmt.Errorf("writing skill %s/SKILL.md: %w", stem, werr)
		}
		installed++
	}
	return installed, nil
}
