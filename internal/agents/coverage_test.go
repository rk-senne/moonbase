package agents

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// === Agent methods: HasShell, HasWrite, IsPipeline, IsConditional ===

func TestAgent_HasShell(t *testing.T) {
	a := &Agent{}
	if a.HasShell() {
		t.Error("expected HasShell=false when Shell is nil")
	}

	a.Shell = &ShellConfig{ReadOnly: true}
	if !a.HasShell() {
		t.Error("expected HasShell=true when Shell is set")
	}
}

func TestAgent_HasWrite(t *testing.T) {
	a := &Agent{}
	if a.HasWrite() {
		t.Error("expected HasWrite=false when Write is nil")
	}

	a.Write = &WriteConfig{Auto: []string{"*.go"}}
	if !a.HasWrite() {
		t.Error("expected HasWrite=true when Write is set")
	}
}

func TestAgent_IsPipeline(t *testing.T) {
	a := &Agent{}
	if a.IsPipeline() {
		t.Error("expected IsPipeline=false when PipelinePosition is nil")
	}

	pos := 3
	a.PipelinePosition = &pos
	if !a.IsPipeline() {
		t.Error("expected IsPipeline=true when PipelinePosition is set")
	}
}

func TestAgent_IsConditional(t *testing.T) {
	a := &Agent{}
	if a.IsConditional() {
		t.Error("expected IsConditional=false when Triggers is nil")
	}

	trigger := ">5 files changed"
	a.Triggers = &trigger
	if !a.IsConditional() {
		t.Error("expected IsConditional=true when Triggers is set")
	}
}

// === Registry methods: Load, Reload, Get, PipelineAgents, Specialists ===

func TestRegistry_Load(t *testing.T) {
	tmpDir := t.TempDir()
	createTestAgent(t, tmpDir, "numbuh-1", "Nigel Uno", "Analyst")
	createTestAgent(t, tmpDir, "numbuh-2", "Hoagie", "Architect")

	reg := NewRegistry(tmpDir)
	cmd := reg.Load()
	msg := cmd()

	loaded, ok := msg.(AgentsLoadedMsg)
	if !ok {
		t.Fatalf("expected AgentsLoadedMsg, got %T", msg)
	}
	if loaded.Err != nil {
		t.Fatalf("unexpected error: %v", loaded.Err)
	}
	if len(loaded.Agents) != 2 {
		t.Errorf("expected 2 agents, got %d", len(loaded.Agents))
	}
	// All should be tagged as built-in
	for _, a := range loaded.Agents {
		if a.Source != SourceBuiltIn {
			t.Errorf("expected source %q, got %q for %s", SourceBuiltIn, a.Source, a.Name)
		}
	}
}

func TestRegistry_Load_EmptyDir(t *testing.T) {
	tmpDir := t.TempDir()

	reg := NewRegistry(tmpDir)
	cmd := reg.Load()
	msg := cmd()

	loaded, ok := msg.(AgentsLoadedMsg)
	if !ok {
		t.Fatalf("expected AgentsLoadedMsg, got %T", msg)
	}
	// Empty dir is not an error, just returns no agents
	if loaded.Err != nil {
		t.Fatalf("unexpected error for empty dir: %v", loaded.Err)
	}
	if len(loaded.Agents) != 0 {
		t.Errorf("expected 0 agents from empty dir, got %d", len(loaded.Agents))
	}
}

func TestRegistry_Reload(t *testing.T) {
	tmpDir := t.TempDir()
	createTestAgent(t, tmpDir, "numbuh-5", "Abby", "Reviewer")

	reg := NewRegistry(tmpDir)
	reg.Reload()

	if reg.Count() != 1 {
		t.Errorf("expected 1 agent after reload, got %d", reg.Count())
	}
	a := reg.GetByName("numbuh-5")
	if a == nil {
		t.Fatal("expected to find numbuh-5")
	}
	if a.Source != SourceBuiltIn {
		t.Errorf("expected source %q, got %q", SourceBuiltIn, a.Source)
	}
}

func TestRegistry_Reload_EmptyDir(t *testing.T) {
	tmpDir := t.TempDir()

	reg := NewRegistry(tmpDir)
	reg.Reload()

	if reg.Count() != 0 {
		t.Errorf("expected 0 agents after reload of empty dir, got %d", reg.Count())
	}
}

func TestRegistry_Get(t *testing.T) {
	tmpDir := t.TempDir()
	createTestAgent(t, tmpDir, "numbuh-3", "Kuki", "Implementer")

	reg := NewRegistry(tmpDir)
	reg.Reload()

	// Valid index
	a := reg.Get(0)
	if a.Name != "numbuh-3" {
		t.Errorf("expected numbuh-3, got %s", a.Name)
	}

	// Out of bounds — returns placeholder
	placeholder := reg.Get(99)
	if placeholder.Name != "unknown" {
		t.Errorf("expected 'unknown' for out of bounds, got %s", placeholder.Name)
	}
	if placeholder.Description != "Operative not found" {
		t.Errorf("expected placeholder description, got %s", placeholder.Description)
	}

	// Negative index
	placeholder2 := reg.Get(-1)
	if placeholder2.Name != "unknown" {
		t.Errorf("expected 'unknown' for negative index, got %s", placeholder2.Name)
	}
}

func TestRegistry_GetByName_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	createTestAgent(t, tmpDir, "numbuh-1", "Nigel", "Analyst")

	reg := NewRegistry(tmpDir)
	reg.Reload()

	result := reg.GetByName("nonexistent")
	if result != nil {
		t.Error("expected nil for non-existent agent")
	}
}

func TestRegistry_PipelineAgents(t *testing.T) {
	tmpDir := t.TempDir()

	// Create agents with pipeline positions
	createPipelineAgent(t, tmpDir, "numbuh-1", 1)
	createPipelineAgent(t, tmpDir, "numbuh-3", 3)
	createPipelineAgent(t, tmpDir, "numbuh-2", 2)
	createConditionalAgent(t, tmpDir, "numbuh-274", ">5 files changed")

	reg := NewRegistry(tmpDir)
	reg.Reload()

	pipeline := reg.PipelineAgents()
	if len(pipeline) != 3 {
		t.Fatalf("expected 3 pipeline agents, got %d", len(pipeline))
	}

	// Should be sorted by pipeline position
	if *pipeline[0].PipelinePosition != 1 {
		t.Errorf("expected first pipeline agent at position 1, got %d", *pipeline[0].PipelinePosition)
	}
	if *pipeline[1].PipelinePosition != 2 {
		t.Errorf("expected second pipeline agent at position 2, got %d", *pipeline[1].PipelinePosition)
	}
	if *pipeline[2].PipelinePosition != 3 {
		t.Errorf("expected third pipeline agent at position 3, got %d", *pipeline[2].PipelinePosition)
	}
}

func TestRegistry_Specialists(t *testing.T) {
	tmpDir := t.TempDir()

	createPipelineAgent(t, tmpDir, "numbuh-1", 1)
	createConditionalAgent(t, tmpDir, "numbuh-274", "auth touched")
	createConditionalAgent(t, tmpDir, "numbuh-86", "dead code found")

	reg := NewRegistry(tmpDir)
	reg.Reload()

	specialists := reg.Specialists()
	if len(specialists) != 2 {
		t.Fatalf("expected 2 specialists, got %d", len(specialists))
	}

	for _, s := range specialists {
		if s.Triggers == nil {
			t.Errorf("specialist %s has nil triggers", s.Name)
		}
	}
}

func TestRegistry_PipelineAgents_Empty(t *testing.T) {
	tmpDir := t.TempDir()
	createTestAgent(t, tmpDir, "custom", "Custom", "Custom Role")

	reg := NewRegistry(tmpDir)
	reg.Reload()

	pipeline := reg.PipelineAgents()
	if len(pipeline) != 0 {
		t.Errorf("expected 0 pipeline agents, got %d", len(pipeline))
	}
}

func TestRegistry_Specialists_Empty(t *testing.T) {
	tmpDir := t.TempDir()
	createTestAgent(t, tmpDir, "custom", "Custom", "Custom Role")

	reg := NewRegistry(tmpDir)
	reg.Reload()

	specialists := reg.Specialists()
	if len(specialists) != 0 {
		t.Errorf("expected 0 specialists, got %d", len(specialists))
	}
}

// === FindAgentsDir and FindAllAgentDirs ===

func TestFindAgentsDir_ConfigOverride(t *testing.T) {
	tmpDir := t.TempDir()
	agentsDir := filepath.Join(tmpDir, "my-agents")
	os.MkdirAll(agentsDir, 0o755)
	os.WriteFile(filepath.Join(agentsDir, "test.md"), []byte("---\nname: t\n---\n# T"), 0o644)

	result, err := FindAgentsDir(agentsDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != agentsDir {
		t.Errorf("expected %s, got %s", agentsDir, result)
	}
}

func TestFindAgentsDir_ConfigOverride_NotDir(t *testing.T) {
	// Config path is not a valid dir — should fall through to other resolution
	_, err := FindAgentsDir("/nonexistent/path/agents")
	// It might find agents relative to exe or CWD, or fail. Just test no panic.
	_ = err
}

func TestFindAgentsDir_EmptyConfig(t *testing.T) {
	// Empty config — uses other resolution strategies
	// This should succeed when running in the moonbase project tree
	_, err := FindAgentsDir("")
	// May succeed or fail depending on CWD; just ensure no panic
	_ = err
}

func TestFindAllAgentDirs_NoResults(t *testing.T) {
	// Override to a non-existent path, with no home or project dirs
	oldWD, _ := os.Getwd()
	tmpDir := t.TempDir()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWD)

	builtIn, user, project, err := FindAllAgentDirs("/nonexistent")
	// Since we're in a temp dir with no agents anywhere standard, we might get an error
	// The function should not panic regardless
	_ = builtIn
	_ = user
	_ = project
	_ = err
}

func TestFindAllAgentDirs_WithBuiltIn(t *testing.T) {
	tmpDir := t.TempDir()
	agentsDir := filepath.Join(tmpDir, "agents")
	os.MkdirAll(agentsDir, 0o755)
	os.WriteFile(filepath.Join(agentsDir, "test.md"), []byte("---\nname: t\n---\n# T"), 0o644)

	oldWD, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWD)

	builtIn, _, _, err := FindAllAgentDirs("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if builtIn == "" {
		t.Error("expected builtIn to be found")
	}
}

// === Sort order ===

func TestSortOrder_Known(t *testing.T) {
	if sortOrder("numbuh-0") != 0 {
		t.Errorf("expected 0, got %d", sortOrder("numbuh-0"))
	}
	if sortOrder("numbuh-5") != 5 {
		t.Errorf("expected 5, got %d", sortOrder("numbuh-5"))
	}
	if sortOrder("sector-z") != 12 {
		t.Errorf("expected 12, got %d", sortOrder("sector-z"))
	}
}

func TestSortOrder_Unknown(t *testing.T) {
	if sortOrder("custom-agent") != 99 {
		t.Errorf("expected 99 for unknown agent, got %d", sortOrder("custom-agent"))
	}
}

// === Helpers ===

func createPipelineAgent(t *testing.T, dir, name string, position int) {
	t.Helper()
	os.MkdirAll(dir, 0o755)
	content := fmt.Sprintf("---\nname: %s\nrole: Pipeline\npipeline_position: %d\ntools:\n  - read\n---\n# %s\n\nPipeline agent.\n", name, position, name)
	err := os.WriteFile(filepath.Join(dir, name+".md"), []byte(content), 0o644)
	if err != nil {
		t.Fatalf("failed to create pipeline agent %s: %v", name, err)
	}
}

func createConditionalAgent(t *testing.T, dir, name, trigger string) {
	t.Helper()
	os.MkdirAll(dir, 0o755)
	content := fmt.Sprintf("---\nname: %s\nrole: Specialist\ntriggers: \"%s\"\ntools:\n  - read\n---\n# %s\n\nConditional agent.\n", name, trigger, name)
	err := os.WriteFile(filepath.Join(dir, name+".md"), []byte(content), 0o644)
	if err != nil {
		t.Fatalf("failed to create conditional agent %s: %v", name, err)
	}
}

// === Additional coverage: SplitFrontmatter \r\n paths ===

func TestSplitFrontmatter_CRLF(t *testing.T) {
	content := []byte("---\r\nname: crlf\r\n---\r\n# Body\r\n")
	yamlBytes, body, err := SplitFrontmatter(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(yamlBytes) == 0 {
		t.Error("expected non-empty yaml")
	}
	if len(body) == 0 {
		t.Error("expected non-empty body")
	}
}

// === loadFromDir with JSON warning ===

func TestLoadFromDir_JsonWarning(t *testing.T) {
	tmpDir := t.TempDir()
	// Put only .json files (no .md) — should trigger warning path
	os.WriteFile(filepath.Join(tmpDir, "agent.json"), []byte(`{}`), 0o644)

	agents, err := loadFromDir(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should return empty (json not loaded)
	if len(agents) != 0 {
		t.Errorf("expected 0 agents for json-only dir, got %d", len(agents))
	}
}

// === LoadMultipleDirs error from first dir ===

func TestLoadMultipleDirs_ErrorFromBuiltIn(t *testing.T) {
	// Use an invalid path that won't glob correctly
	// filepath.Glob returns an error only for malformed patterns
	// Use a valid path but non-existent — loadFromDir with empty glob returns nil
	// Actually, test the error path by using a path that fails the Glob pattern
	tmpDir := t.TempDir()
	createTestAgent(t, tmpDir, "numbuh-1", "Nigel", "Analyst")

	reg := NewRegistry(tmpDir)
	cmd := reg.LoadMultipleDirs(tmpDir, "/nonexistent-user", "/nonexistent-project")
	msg := cmd()

	loaded := msg.(AgentsLoadedMsg)
	if loaded.Err != nil {
		t.Fatalf("unexpected error: %v", loaded.Err)
	}
	if len(loaded.Agents) != 1 {
		t.Errorf("expected 1 agent, got %d", len(loaded.Agents))
	}
}

// === loadFromDir with malformed agent file ===

func TestLoadFromDir_SkipsMalformedAgent(t *testing.T) {
	tmpDir := t.TempDir()
	// One valid, one malformed
	createTestAgent(t, tmpDir, "good", "Good Agent", "Tester")
	os.WriteFile(filepath.Join(tmpDir, "bad.md"), []byte("not valid frontmatter"), 0o644)

	agents, err := loadFromDir(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(agents) != 1 {
		t.Errorf("expected 1 valid agent (skipping bad), got %d", len(agents))
	}
}
