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

// === discoverSkillRegistry tests ===

func TestDiscoverSkillRegistry_FrontmatterToRegistry(t *testing.T) {
	tmpDir := t.TempDir()
	skillsDir := filepath.Join(tmpDir, ".kiro", "skills")
	os.MkdirAll(skillsDir, 0o755)

	// Skill with valid frontmatter
	content := "---\nname: docker-build\ndescription: Docker patterns\n---\n\n# Docker\nContent.\n"
	os.WriteFile(filepath.Join(skillsDir, "docker-build.md"), []byte(content), 0o644)

	registry := discoverSkillRegistry(tmpDir)
	if registry.Len() != 1 {
		t.Fatalf("expected 1 skill in registry, got %d", registry.Len())
	}
	meta := registry.Get("docker-build")
	if meta == nil {
		t.Fatal("expected 'docker-build' in registry")
	}
	if meta.Description != "Docker patterns" {
		t.Errorf("unexpected description: %q", meta.Description)
	}
}

func TestDiscoverSkillRegistry_LegacyNotInRegistry(t *testing.T) {
	tmpDir := t.TempDir()
	skillsDir := filepath.Join(tmpDir, ".kiro", "skills")
	os.MkdirAll(skillsDir, 0o755)

	// Legacy skill (no frontmatter)
	os.WriteFile(filepath.Join(skillsDir, "legacy.md"), []byte("# Legacy\nNo frontmatter.\n"), 0o644)

	registry := discoverSkillRegistry(tmpDir)
	if registry.Len() != 0 {
		t.Fatalf("expected 0 skills in registry (legacy excluded), got %d", registry.Len())
	}
}

func TestDiscoverSkillRegistry_MissingDir(t *testing.T) {
	tmpDir := t.TempDir()
	// No .kiro/skills/ directory

	registry := discoverSkillRegistry(tmpDir)
	if registry == nil {
		t.Fatal("expected non-nil registry")
	}
	if registry.Len() != 0 {
		t.Errorf("expected empty registry, got %d skills", registry.Len())
	}
}

func TestDiscoverSkillRegistry_SubdirectorySkills(t *testing.T) {
	tmpDir := t.TempDir()
	skillsDir := filepath.Join(tmpDir, ".kiro", "skills", "api")
	os.MkdirAll(skillsDir, 0o755)

	content := "---\nname: api-patterns\ndescription: REST patterns\n---\n\n# API\nContent.\n"
	os.WriteFile(filepath.Join(skillsDir, "SKILL.md"), []byte(content), 0o644)

	registry := discoverSkillRegistry(tmpDir)
	if registry.Len() != 1 {
		t.Fatalf("expected 1 skill from subdir, got %d", registry.Len())
	}
}

func TestDiscoverLegacySkills_OnlyLegacy(t *testing.T) {
	tmpDir := t.TempDir()
	skillsDir := filepath.Join(tmpDir, ".kiro", "skills")
	os.MkdirAll(skillsDir, 0o755)

	// Frontmatter skill — should NOT appear in legacy
	os.WriteFile(filepath.Join(skillsDir, "modern.md"),
		[]byte("---\nname: modern\ndescription: Modern\n---\n\nBody.\n"), 0o644)
	// Legacy skill — should appear in legacy
	os.WriteFile(filepath.Join(skillsDir, "legacy.md"),
		[]byte("# Legacy Skill\nNo frontmatter.\n"), 0o644)

	skills, err := discoverLegacySkills(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(skills) != 1 {
		t.Fatalf("expected 1 legacy skill, got %d", len(skills))
	}
	if skills[0].Name != "legacy" {
		t.Errorf("expected name 'legacy', got %q", skills[0].Name)
	}
	if skills[0].Content == "" {
		t.Error("expected legacy skill to have content loaded eagerly")
	}
}

func TestDiscover_SkillRegistry_Populated(t *testing.T) {
	tmpDir := t.TempDir()
	skillsDir := filepath.Join(tmpDir, ".kiro", "skills")
	os.MkdirAll(skillsDir, 0o755)

	os.WriteFile(filepath.Join(skillsDir, "registered.md"),
		[]byte("---\nname: registered\ndescription: In registry\n---\n\nBody.\n"), 0o644)

	ctx := Discover(tmpDir)
	if ctx.SkillRegistry == nil {
		t.Fatal("expected SkillRegistry to be populated")
	}
	if ctx.SkillRegistry.Len() != 1 {
		t.Fatalf("expected 1 in registry, got %d", ctx.SkillRegistry.Len())
	}
}

func TestDiscover_MetadataOnly_NoBodyRead(t *testing.T) {
	tmpDir := t.TempDir()
	skillsDir := filepath.Join(tmpDir, ".kiro", "skills")
	os.MkdirAll(skillsDir, 0o755)

	os.WriteFile(filepath.Join(skillsDir, "meta-only.md"),
		[]byte("---\nname: meta-only\ndescription: Should not read body\n---\n\nExpensive body content.\n"), 0o644)

	ctx := Discover(tmpDir)
	// SkillRegistry should have the skill
	if ctx.SkillRegistry.Get("meta-only") == nil {
		t.Fatal("expected meta-only in registry")
	}
	// But content should not be loaded yet (verify by checking internal state)
	// We can verify by calling LoadContent and checking it works (proves lazy)
	content, err := ctx.SkillRegistry.LoadContent("meta-only")
	if err != nil {
		t.Fatalf("LoadContent failed: %v", err)
	}
	if content != "Expensive body content." {
		t.Errorf("unexpected content: %q", content)
	}
}

// === ComposePrompt skill catalog tests ===

func TestComposePrompt_SkillCatalog_ShowsMetadataOnly(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test-skill.md")
	os.WriteFile(path, []byte("---\nname: test-skill\ndescription: Test desc\n---\n\nSecret body.\n"), 0o644)

	r := NewSkillRegistry()
	r.Register(SkillMeta{Name: "test-skill", Description: "Test desc", Path: path})

	ctx := &ProjectContext{SkillRegistry: r}
	result := ComposePrompt("# Agent", ctx, "Do stuff")

	if !strings.Contains(result, "AVAILABLE SKILLS") {
		t.Error("expected AVAILABLE SKILLS section")
	}
	if !strings.Contains(result, "test-skill") {
		t.Error("expected skill name in catalog")
	}
	if !strings.Contains(result, "Test desc") {
		t.Error("expected description in catalog")
	}
	// Body should NOT be in the prompt (metadata only)
	if strings.Contains(result, "Secret body") {
		t.Error("expected skill body NOT to be included in catalog")
	}
	if !strings.Contains(result, "@skill(name)") {
		t.Error("expected @skill instruction in catalog")
	}
}

func TestComposePrompt_SkillCatalog_FallbackToLegacy(t *testing.T) {
	ctx := &ProjectContext{
		Skills: []SkillFile{
			{Name: "legacy-skill", Content: "Legacy content here"},
		},
	}
	result := ComposePrompt("# Agent", ctx, "Do stuff")

	if !strings.Contains(result, "PROJECT SKILLS") {
		t.Error("expected legacy PROJECT SKILLS section")
	}
	if !strings.Contains(result, "Legacy content here") {
		t.Error("expected legacy skill content")
	}
}

func TestComposePrompt_TaskContainsSkillRef_AutoInjects(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "docker-build.md")
	os.WriteFile(path, []byte("---\nname: docker-build\ndescription: Docker\n---\n\n# Docker Patterns\n\nUse multi-stage.\n"), 0o644)

	r := NewSkillRegistry()
	r.Register(SkillMeta{Name: "docker-build", Description: "Docker", Path: path})

	ctx := &ProjectContext{SkillRegistry: r}
	result := ComposePrompt("# Agent", ctx, "Review using @skill(docker-build)")

	if !strings.Contains(result, "--- SKILL: docker-build ---") {
		t.Error("expected injected skill section")
	}
	if !strings.Contains(result, "Use multi-stage.") {
		t.Error("expected skill body content injected")
	}
	if !strings.Contains(result, "--- END SKILL ---") {
		t.Error("expected skill end delimiter")
	}
}

func TestComposePrompt_SkillNotFound_ShowsNote(t *testing.T) {
	r := NewSkillRegistry()
	r.Register(SkillMeta{Name: "existing", Description: "Exists", Path: "/fake.md"})

	ctx := &ProjectContext{SkillRegistry: r}
	result := ComposePrompt("# Agent", ctx, "Use @skill(nonexistent) please")

	if !strings.Contains(result, "SKILL RESOLUTION") {
		t.Error("expected SKILL RESOLUTION section for not-found")
	}
	if !strings.Contains(result, "nonexistent") {
		t.Error("expected not-found name in resolution note")
	}
	if !strings.Contains(result, "existing") {
		t.Error("expected available skills listed in resolution note")
	}
}

func TestComposePrompt_SkillCatalog_EmptyRegistry(t *testing.T) {
	r := NewSkillRegistry()
	ctx := &ProjectContext{SkillRegistry: r}
	result := ComposePrompt("# Agent", ctx, "Task")

	if strings.Contains(result, "AVAILABLE SKILLS") {
		t.Error("expected no AVAILABLE SKILLS section for empty registry")
	}
}
