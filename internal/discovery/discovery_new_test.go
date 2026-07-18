package discovery

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// === discoverSkills tests ===

func TestDiscoverSkills_NoSkillsDir(t *testing.T) {
	tmpDir := t.TempDir()
	// No .kiro/skills/ directory

	skills, err := discoverSkills(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if skills != nil {
		t.Errorf("expected nil skills for missing dir, got %v", skills)
	}
}

func TestDiscoverSkills_EmptyDir(t *testing.T) {
	tmpDir := t.TempDir()
	os.MkdirAll(filepath.Join(tmpDir, ".kiro", "skills"), 0o755)

	skills, err := discoverSkills(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(skills) != 0 {
		t.Errorf("expected 0 skills for empty dir, got %d", len(skills))
	}
}

func TestDiscoverSkills_TopLevelMD(t *testing.T) {
	tmpDir := t.TempDir()
	skillsDir := filepath.Join(tmpDir, ".kiro", "skills")
	os.MkdirAll(skillsDir, 0o755)

	// Create a top-level .md file
	os.WriteFile(filepath.Join(skillsDir, "coding-standards.md"), []byte("# Coding Standards\nUse gofmt."), 0o644)

	skills, err := discoverSkills(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(skills) != 1 {
		t.Fatalf("expected 1 skill, got %d", len(skills))
	}
	if skills[0].Name != "coding-standards" {
		t.Errorf("expected name 'coding-standards', got %q", skills[0].Name)
	}
	if !strings.Contains(skills[0].Content, "gofmt") {
		t.Error("expected content to contain 'gofmt'")
	}
}

func TestDiscoverSkills_SKILLmd(t *testing.T) {
	tmpDir := t.TempDir()
	skillsDir := filepath.Join(tmpDir, ".kiro", "skills")
	os.MkdirAll(skillsDir, 0o755)

	os.WriteFile(filepath.Join(skillsDir, "SKILL.md"), []byte("# Main Skill\nImportant knowledge."), 0o644)

	skills, err := discoverSkills(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(skills) != 1 {
		t.Fatalf("expected 1 skill, got %d", len(skills))
	}
	if skills[0].Name != "SKILL" {
		t.Errorf("expected name 'SKILL', got %q", skills[0].Name)
	}
}

func TestDiscoverSkills_SubdirectoryFiles(t *testing.T) {
	tmpDir := t.TempDir()
	skillsDir := filepath.Join(tmpDir, ".kiro", "skills")
	subDir := filepath.Join(skillsDir, "domain")
	os.MkdirAll(subDir, 0o755)

	os.WriteFile(filepath.Join(subDir, "auth.md"), []byte("# Auth\nOAuth2 flow."), 0o644)
	os.WriteFile(filepath.Join(subDir, "payments.md"), []byte("# Payments\nStripe integration."), 0o644)
	// Non-md file should be skipped
	os.WriteFile(filepath.Join(subDir, "notes.txt"), []byte("not a skill"), 0o644)

	skills, err := discoverSkills(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(skills) != 2 {
		t.Fatalf("expected 2 skills from subdirectory, got %d", len(skills))
	}

	// Check naming convention: subdir/filename
	names := make(map[string]bool)
	for _, s := range skills {
		names[s.Name] = true
	}
	if !names["domain/auth"] {
		t.Error("expected skill named 'domain/auth'")
	}
	if !names["domain/payments"] {
		t.Error("expected skill named 'domain/payments'")
	}
}

func TestDiscoverSkills_MixedContent(t *testing.T) {
	tmpDir := t.TempDir()
	skillsDir := filepath.Join(tmpDir, ".kiro", "skills")
	subDir := filepath.Join(skillsDir, "api")
	os.MkdirAll(subDir, 0o755)

	// Top-level files
	os.WriteFile(filepath.Join(skillsDir, "general.md"), []byte("General skill"), 0o644)
	// Subdirectory files
	os.WriteFile(filepath.Join(subDir, "rest.md"), []byte("REST patterns"), 0o644)

	skills, err := discoverSkills(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(skills) != 2 {
		t.Fatalf("expected 2 skills (1 top-level + 1 sub), got %d", len(skills))
	}
}

func TestDiscoverSkills_IgnoresSubdirectoryDirs(t *testing.T) {
	tmpDir := t.TempDir()
	skillsDir := filepath.Join(tmpDir, ".kiro", "skills")
	subDir := filepath.Join(skillsDir, "deep")
	deepDir := filepath.Join(subDir, "deeper")
	os.MkdirAll(deepDir, 0o755)

	// Files in deeper nested dirs should NOT be discovered (only one level)
	os.WriteFile(filepath.Join(deepDir, "hidden.md"), []byte("hidden"), 0o644)
	os.WriteFile(filepath.Join(subDir, "visible.md"), []byte("visible"), 0o644)

	skills, err := discoverSkills(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(skills) != 1 {
		t.Fatalf("expected 1 skill (only one level deep), got %d", len(skills))
	}
	if skills[0].Name != "deep/visible" {
		t.Errorf("expected 'deep/visible', got %q", skills[0].Name)
	}
}

// === discoverPrompts tests ===

func TestDiscoverPrompts_NoPromptsDir(t *testing.T) {
	tmpDir := t.TempDir()

	prompts, err := discoverPrompts(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if prompts != nil {
		t.Errorf("expected nil prompts for missing dir, got %v", prompts)
	}
}

func TestDiscoverPrompts_EmptyDir(t *testing.T) {
	tmpDir := t.TempDir()
	os.MkdirAll(filepath.Join(tmpDir, ".kiro", "prompts"), 0o755)

	prompts, err := discoverPrompts(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(prompts) != 0 {
		t.Errorf("expected 0 prompts, got %d", len(prompts))
	}
}

func TestDiscoverPrompts_LoadsMDFiles(t *testing.T) {
	tmpDir := t.TempDir()
	promptsDir := filepath.Join(tmpDir, ".kiro", "prompts")
	os.MkdirAll(promptsDir, 0o755)

	os.WriteFile(filepath.Join(promptsDir, "review.md"), []byte("Review the code carefully."), 0o644)
	os.WriteFile(filepath.Join(promptsDir, "refactor.md"), []byte("Refactor for clarity."), 0o644)

	prompts, err := discoverPrompts(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(prompts) != 2 {
		t.Fatalf("expected 2 prompts, got %d", len(prompts))
	}

	names := make(map[string]bool)
	for _, p := range prompts {
		names[p.Name] = true
	}
	if !names["review"] {
		t.Error("expected prompt named 'review'")
	}
	if !names["refactor"] {
		t.Error("expected prompt named 'refactor'")
	}
}

func TestDiscoverPrompts_IgnoresNonMD(t *testing.T) {
	tmpDir := t.TempDir()
	promptsDir := filepath.Join(tmpDir, ".kiro", "prompts")
	os.MkdirAll(promptsDir, 0o755)

	os.WriteFile(filepath.Join(promptsDir, "valid.md"), []byte("prompt content"), 0o644)
	os.WriteFile(filepath.Join(promptsDir, "notes.txt"), []byte("not a prompt"), 0o644)
	os.WriteFile(filepath.Join(promptsDir, "data.json"), []byte("{}"), 0o644)

	prompts, err := discoverPrompts(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(prompts) != 1 {
		t.Fatalf("expected 1 prompt (only .md), got %d", len(prompts))
	}
	if prompts[0].Name != "valid" {
		t.Errorf("expected name 'valid', got %q", prompts[0].Name)
	}
}

func TestDiscoverPrompts_IgnoresDirectories(t *testing.T) {
	tmpDir := t.TempDir()
	promptsDir := filepath.Join(tmpDir, ".kiro", "prompts")
	os.MkdirAll(filepath.Join(promptsDir, "subdir"), 0o755)

	os.WriteFile(filepath.Join(promptsDir, "top.md"), []byte("top level prompt"), 0o644)
	// Files in subdirectories should be ignored
	os.WriteFile(filepath.Join(promptsDir, "subdir", "nested.md"), []byte("nested"), 0o644)

	prompts, err := discoverPrompts(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(prompts) != 1 {
		t.Fatalf("expected 1 prompt (ignores subdirs), got %d", len(prompts))
	}
}

// === ComposeCacheablePrefix tests ===

func TestComposeCacheablePrefix_NoContext(t *testing.T) {
	agentPrompt := "# Agent\n\nYou are a helpful agent."

	prefix := ComposeCacheablePrefix(agentPrompt, nil)
	if !strings.Contains(prefix, agentPrompt) {
		t.Error("expected agent prompt in cacheable prefix")
	}
	if strings.Contains(prefix, "--- TASK ---") {
		t.Error("cacheable prefix should NOT contain TASK section")
	}
}

func TestComposeCacheablePrefix_WithContext(t *testing.T) {
	agentPrompt := "# Agent\n\nYou are a code reviewer."
	ctx := &ProjectContext{
		Steering: []SteeringFile{
			{Name: "dev-rules", Content: "Use Go conventions."},
		},
		Stack: StackInfo{
			Language:    "go",
			BuildTool:   "go",
			TestCommand: "go test ./...",
		},
	}

	prefix := ComposeCacheablePrefix(agentPrompt, ctx)

	if !strings.Contains(prefix, "PROJECT RULES") {
		t.Error("expected steering rules in prefix")
	}
	if !strings.Contains(prefix, "code reviewer") {
		t.Error("expected agent prompt in prefix")
	}
	if !strings.Contains(prefix, "PROJECT STACK") {
		t.Error("expected stack info in prefix")
	}
	if strings.Contains(prefix, "--- TASK ---") {
		t.Error("cacheable prefix should NOT contain TASK section")
	}
}

func TestComposeCacheablePrefix_ExcludesTask(t *testing.T) {
	agentPrompt := "# Agent"
	ctx := &ProjectContext{
		Skills: []SkillFile{
			{Name: "auth", Content: "OAuth2 knowledge"},
		},
	}

	prefix := ComposeCacheablePrefix(agentPrompt, ctx)
	full := ComposePrompt(agentPrompt, ctx, "fix the auth bug")

	// The full prompt should contain the task, prefix should not
	if strings.Contains(prefix, "fix the auth bug") {
		t.Error("cacheable prefix should not contain the task")
	}
	if !strings.Contains(full, "fix the auth bug") {
		t.Error("full prompt should contain the task")
	}
	// Prefix should be a prefix of the full prompt (up to the task part)
	if !strings.HasPrefix(full, prefix) {
		t.Error("full prompt should start with the cacheable prefix")
	}
}

func TestComposeCacheablePrefix_WithSpecs(t *testing.T) {
	agentPrompt := "# Agent"
	ctx := &ProjectContext{
		Specs: []SpecFile{
			{Feature: "auth", Type: "requirements", Content: "AC-1: Login works"},
		},
	}

	prefix := ComposeCacheablePrefix(agentPrompt, ctx)
	if !strings.Contains(prefix, "PROJECT SPEC CONTEXT") {
		t.Error("expected spec context in prefix")
	}
	if !strings.Contains(prefix, "AC-1") {
		t.Error("expected AC reference in prefix")
	}
}

func TestComposeCacheablePrefix_WithSkills(t *testing.T) {
	agentPrompt := "# Agent"
	ctx := &ProjectContext{
		Skills: []SkillFile{
			{Name: "testing", Content: "# Testing Guide\nWrite table-driven tests."},
		},
	}

	prefix := ComposeCacheablePrefix(agentPrompt, ctx)
	if !strings.Contains(prefix, "PROJECT SKILLS") {
		t.Error("expected skills section in prefix")
	}
	if !strings.Contains(prefix, "Testing Guide") {
		t.Error("expected skill content in prefix")
	}
}

// === ProjectContext helper method tests ===

func TestProjectContext_HasSkills(t *testing.T) {
	ctx := &ProjectContext{}
	if ctx.HasSkills() {
		t.Error("expected HasSkills=false for empty context")
	}

	ctx.Skills = []SkillFile{{Name: "test"}}
	if !ctx.HasSkills() {
		t.Error("expected HasSkills=true when skills exist")
	}
}

func TestProjectContext_HasPrompts(t *testing.T) {
	ctx := &ProjectContext{}
	if ctx.HasPrompts() {
		t.Error("expected HasPrompts=false for empty context")
	}

	ctx.Prompts = []PromptFile{{Name: "review"}}
	if !ctx.HasPrompts() {
		t.Error("expected HasPrompts=true when prompts exist")
	}
}

func TestProjectContext_Summary_WithSkillsAndPrompts(t *testing.T) {
	ctx := &ProjectContext{
		Skills:  []SkillFile{{Name: "auth"}, {Name: "testing"}},
		Prompts: []PromptFile{{Name: "review"}},
		Stack:   StackInfo{Language: "go", BuildTool: "go"},
	}

	summary := ctx.Summary()
	if !strings.Contains(summary, "Skills: 2") {
		t.Errorf("expected 'Skills: 2' in summary, got: %s", summary)
	}
	if !strings.Contains(summary, "Prompts: 1") {
		t.Errorf("expected 'Prompts: 1' in summary, got: %s", summary)
	}
}
