package discovery

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDiscover_RealProject(t *testing.T) {
	// Use moonbase itself as the test project
	projectDir := findProjectRoot(t)
	if projectDir == "" {
		t.Skip("project root not found")
	}

	ctx, err := Discover(projectDir)
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	// Moonbase has .kiro/specs/moonbase-v3/
	if !ctx.HasSpecs() {
		t.Error("expected specs to be found")
	}

	// Check that moonbase-v3 specs are discovered
	foundRequirements := false
	foundDesign := false
	foundTasks := false
	for _, s := range ctx.Specs {
		if s.Feature == "moonbase-v3" {
			switch s.Type {
			case "requirements":
				foundRequirements = true
			case "design":
				foundDesign = true
			case "tasks":
				foundTasks = true
			}
		}
	}
	if !foundRequirements {
		t.Error("expected to find moonbase-v3/requirements.md")
	}
	if !foundDesign {
		t.Error("expected to find moonbase-v3/design.md")
	}
	if !foundTasks {
		t.Error("expected to find moonbase-v3/tasks.md")
	}

	// Moonbase has .kiro/steering/dev-rules.md
	if !ctx.HasSteering() {
		t.Error("expected steering rules to be found")
	}

	foundDevRules := false
	for _, s := range ctx.Steering {
		if s.Name == "dev-rules" {
			foundDevRules = true
		}
	}
	if !foundDevRules {
		t.Error("expected to find dev-rules steering file")
	}

	// Moonbase is a Go project
	if ctx.Stack.Language != "go" {
		t.Errorf("expected stack language 'go', got: %s", ctx.Stack.Language)
	}
	if ctx.Stack.TestCommand != "go test ./..." {
		t.Errorf("expected test command 'go test ./...', got: %s", ctx.Stack.TestCommand)
	}

	// README exists
	if ctx.README == "" {
		t.Error("expected README to be found")
	}
	if !strings.Contains(ctx.README, "Moonbase") {
		t.Error("expected README to contain 'Moonbase'")
	}
}

func TestDiscover_EmptyDir(t *testing.T) {
	tmpDir := t.TempDir()

	ctx, err := Discover(tmpDir)
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	if ctx.HasSpecs() {
		t.Error("expected no specs in empty dir")
	}
	if ctx.HasSteering() {
		t.Error("expected no steering in empty dir")
	}
	if ctx.Stack.Language != "" {
		t.Errorf("expected empty stack, got: %s", ctx.Stack.Language)
	}
	if ctx.README != "" {
		t.Error("expected empty README")
	}
}

func TestShouldExclude_ManualInclusion(t *testing.T) {
	content := []byte("---\ninclusion: manual\n---\n# Some Rule\n\nContent here.\n")
	if !shouldExclude(content) {
		t.Error("expected file with 'inclusion: manual' to be excluded")
	}
}

func TestShouldExclude_AutoInclusion(t *testing.T) {
	content := []byte("---\ninclusion: auto\n---\n# Some Rule\n\nContent here.\n")
	if shouldExclude(content) {
		t.Error("expected file with 'inclusion: auto' to NOT be excluded")
	}
}

func TestShouldExclude_NoFrontmatter(t *testing.T) {
	content := []byte("# Some Rule\n\nJust markdown, no frontmatter.\n")
	if shouldExclude(content) {
		t.Error("expected file without frontmatter to NOT be excluded")
	}
}

func TestShouldExclude_NoInclusionField(t *testing.T) {
	content := []byte("---\ntitle: My Rule\n---\n# Some Rule\n\nHas frontmatter but no inclusion field.\n")
	if shouldExclude(content) {
		t.Error("expected file without inclusion field to NOT be excluded")
	}
}

func TestDiscoverSteering_ManualExclusion(t *testing.T) {
	tmpDir := t.TempDir()
	steeringDir := filepath.Join(tmpDir, ".kiro", "steering")
	os.MkdirAll(steeringDir, 0o755)

	// Write an auto-include file
	autoContent := "# Auto Rule\n\nThis should be included.\n"
	os.WriteFile(filepath.Join(steeringDir, "auto-rule.md"), []byte(autoContent), 0o644)

	// Write a manual-include file
	manualContent := "---\ninclusion: manual\n---\n# Manual Rule\n\nThis should be excluded.\n"
	os.WriteFile(filepath.Join(steeringDir, "manual-rule.md"), []byte(manualContent), 0o644)

	steering, err := discoverSteering(tmpDir)
	if err != nil {
		t.Fatalf("discoverSteering failed: %v", err)
	}

	if len(steering) != 1 {
		t.Fatalf("expected 1 steering file, got %d", len(steering))
	}
	if steering[0].Name != "auto-rule" {
		t.Errorf("expected 'auto-rule', got: %s", steering[0].Name)
	}
}

func TestComposePrompt_Full(t *testing.T) {
	agentPrompt := "# Numbuh 1\n\nI am the analyst."
	context := &ProjectContext{
		Steering: []SteeringFile{
			{Name: "dev-rules", Content: "# Dev Rules\n\nUse Go.\n"},
		},
		Specs: []SpecFile{
			{Feature: "feature-x", Type: "requirements", Content: "# Requirements\n\nAC-1.1: Do the thing.\n"},
		},
		Stack: StackInfo{Language: "go", BuildTool: "go", TestCommand: "go test ./..."},
	}

	result := ComposePrompt(agentPrompt, context, "Fix the bug in parser.go")

	// Check ordering: steering before agent prompt
	steeringIdx := strings.Index(result, "PROJECT RULES")
	agentIdx := strings.Index(result, "# Numbuh 1")
	specIdx := strings.Index(result, "PROJECT SPEC CONTEXT")
	taskIdx := strings.Index(result, "TASK")

	if steeringIdx == -1 {
		t.Error("expected steering section in composed prompt")
	}
	if agentIdx == -1 {
		t.Error("expected agent prompt in composed prompt")
	}
	if specIdx == -1 {
		t.Error("expected spec context in composed prompt")
	}
	if taskIdx == -1 {
		t.Error("expected task section in composed prompt")
	}

	if steeringIdx > agentIdx {
		t.Error("expected steering BEFORE agent prompt")
	}
	if agentIdx > specIdx {
		t.Error("expected agent prompt BEFORE spec context")
	}
	if specIdx > taskIdx {
		t.Error("expected spec context BEFORE task")
	}

	// Check content is present
	if !strings.Contains(result, "Use Go.") {
		t.Error("expected steering content in result")
	}
	if !strings.Contains(result, "I am the analyst") {
		t.Error("expected agent prompt in result")
	}
	if !strings.Contains(result, "AC-1.1") {
		t.Error("expected spec content in result")
	}
	if !strings.Contains(result, "Fix the bug") {
		t.Error("expected task in result")
	}
}

func TestComposePrompt_NoContext(t *testing.T) {
	agentPrompt := "# Numbuh 3\n\nI write code."

	result := ComposePrompt(agentPrompt, nil, "Build the widget")

	if !strings.Contains(result, "# Numbuh 3") {
		t.Error("expected agent prompt in result")
	}
	if !strings.Contains(result, "Build the widget") {
		t.Error("expected task in result")
	}
	if strings.Contains(result, "PROJECT RULES") {
		t.Error("expected no steering section when context is nil")
	}
}

func TestComposePrompt_EmptyContext(t *testing.T) {
	agentPrompt := "# Numbuh 4\n\nI test things."
	context := &ProjectContext{}

	result := ComposePrompt(agentPrompt, context, "Test the parser")

	if !strings.Contains(result, "# Numbuh 4") {
		t.Error("expected agent prompt")
	}
	if strings.Contains(result, "PROJECT RULES") {
		t.Error("expected no steering when empty")
	}
	if strings.Contains(result, "PROJECT SPEC") {
		t.Error("expected no specs when empty")
	}
}

func TestStripFrontmatter(t *testing.T) {
	input := "---\ninclusion: auto\ntitle: Test\n---\n# Content\n\nBody here.\n"
	result := stripFrontmatter(input)

	if strings.Contains(result, "inclusion") {
		t.Error("expected frontmatter to be stripped")
	}
	if !strings.Contains(result, "# Content") {
		t.Error("expected body to be preserved")
	}
	if !strings.Contains(result, "Body here.") {
		t.Error("expected body content to be preserved")
	}
}

func TestStripFrontmatter_NoFrontmatter(t *testing.T) {
	input := "# Just Content\n\nNo frontmatter here.\n"
	result := stripFrontmatter(input)

	if result != input {
		t.Errorf("expected unchanged content, got: %s", result)
	}
}

func TestSummary(t *testing.T) {
	ctx := &ProjectContext{
		Specs: []SpecFile{
			{Feature: "feature-a", Type: "requirements"},
			{Feature: "feature-a", Type: "design"},
			{Feature: "feature-b", Type: "tasks"},
		},
		Steering: []SteeringFile{
			{Name: "dev-rules"},
			{Name: "product"},
		},
		Stack: StackInfo{Language: "go", BuildTool: "go"},
	}

	summary := ctx.Summary()
	if !strings.Contains(summary, "Specs:") {
		t.Error("expected Specs in summary")
	}
	if !strings.Contains(summary, "Steering:") {
		t.Error("expected Steering in summary")
	}
	if !strings.Contains(summary, "Stack: go/go") {
		t.Error("expected Stack in summary")
	}
}

// findProjectRoot walks up to find the moonbase project root
func findProjectRoot(t *testing.T) string {
	t.Helper()

	wd, _ := os.Getwd()
	for dir := wd; dir != "/"; dir = filepath.Dir(dir) {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			if _, err := os.Stat(filepath.Join(dir, "agents")); err == nil {
				return dir
			}
		}
	}
	return ""
}
