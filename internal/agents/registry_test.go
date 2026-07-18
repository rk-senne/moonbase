package agents

import (
	"os"
	"path/filepath"
	"testing"
)

func createTestAgent(t *testing.T, dir, name, designation, role string) {
	t.Helper()
	os.MkdirAll(dir, 0o755)
	content := "---\nname: " + name + "\ndesignation: " + designation + "\nrole: " + role + "\n---\n# " + name + "\n\nAgent prompt.\n"
	err := os.WriteFile(filepath.Join(dir, name+".md"), []byte(content), 0o644)
	if err != nil {
		t.Fatalf("failed to create test agent %s: %v", name, err)
	}
}

func TestLoadMultipleDirs_MergesAgents(t *testing.T) {
	tmpDir := t.TempDir()
	builtIn := filepath.Join(tmpDir, "builtin")
	user := filepath.Join(tmpDir, "user")
	project := filepath.Join(tmpDir, "project")

	// Create agents in built-in
	createTestAgent(t, builtIn, "numbuh-1", "Nigel Uno", "Analyst")
	createTestAgent(t, builtIn, "numbuh-2", "Hoagie", "Architect")

	// Create a custom agent in user dir
	createTestAgent(t, user, "custom-agent", "Custom", "Custom Role")

	// Create an override in project dir
	createTestAgent(t, project, "numbuh-1", "Override Nigel", "Custom Analyst")

	reg := NewRegistry(builtIn)
	if err := reg.LoadMultipleDirsSync(builtIn, user, project); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	all := reg.All()
	if len(all) != 3 {
		t.Fatalf("expected 3 agents after merge, got %d", len(all))
	}

	// numbuh-1 should be from project (override)
	n1 := reg.GetByName("numbuh-1")
	if n1 == nil {
		t.Fatal("numbuh-1 not found")
	}
	if n1.Source != SourceProject {
		t.Errorf("expected numbuh-1 source %q, got %q", SourceProject, n1.Source)
	}
	if n1.Designation != "Override Nigel" {
		t.Errorf("expected override designation, got %q", n1.Designation)
	}

	// numbuh-2 should be from built-in
	n2 := reg.GetByName("numbuh-2")
	if n2 == nil {
		t.Fatal("numbuh-2 not found")
	}
	if n2.Source != SourceBuiltIn {
		t.Errorf("expected numbuh-2 source %q, got %q", SourceBuiltIn, n2.Source)
	}

	// custom-agent should be from user
	ca := reg.GetByName("custom-agent")
	if ca == nil {
		t.Fatal("custom-agent not found")
	}
	if ca.Source != SourceUser {
		t.Errorf("expected custom-agent source %q, got %q", SourceUser, ca.Source)
	}
}

func TestLoadMultipleDirs_UserOverridesBuiltIn(t *testing.T) {
	tmpDir := t.TempDir()
	builtIn := filepath.Join(tmpDir, "builtin")
	user := filepath.Join(tmpDir, "user")

	createTestAgent(t, builtIn, "numbuh-4", "Original", "QA")
	createTestAgent(t, user, "numbuh-4", "User Override", "Custom QA")

	reg := NewRegistry(builtIn)
	if err := reg.LoadMultipleDirsSync(builtIn, user, ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	n4 := reg.GetByName("numbuh-4")
	if n4 == nil {
		t.Fatal("numbuh-4 not found")
	}
	if n4.Source != SourceUser {
		t.Errorf("expected source %q, got %q", SourceUser, n4.Source)
	}
	if n4.Designation != "User Override" {
		t.Errorf("expected 'User Override', got %q", n4.Designation)
	}
}

func TestLoadMultipleDirs_HandlesEmptyDirs(t *testing.T) {
	tmpDir := t.TempDir()
	builtIn := filepath.Join(tmpDir, "builtin")

	createTestAgent(t, builtIn, "numbuh-5", "Abby", "Reviewer")

	reg := NewRegistry(builtIn)
	// Empty user and project dirs
	if err := reg.LoadMultipleDirsSync(builtIn, "", ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	all := reg.All()
	if len(all) != 1 {
		t.Fatalf("expected 1 agent, got %d", len(all))
	}
	if all[0].Source != SourceBuiltIn {
		t.Errorf("expected source %q, got %q", SourceBuiltIn, all[0].Source)
	}
}

func TestLoadMultipleDirs_HandlesNonexistentDirs(t *testing.T) {
	tmpDir := t.TempDir()
	builtIn := filepath.Join(tmpDir, "builtin")

	createTestAgent(t, builtIn, "numbuh-3", "Kuki", "Implementer")

	reg := NewRegistry(builtIn)
	// Point to non-existent dirs for user and project
	if err := reg.LoadMultipleDirsSync(builtIn, "/nonexistent/user", "/nonexistent/project"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	all := reg.All()
	if len(all) != 1 {
		t.Fatalf("expected 1 agent, got %d", len(all))
	}
}

func TestReloadMultipleDirs(t *testing.T) {
	tmpDir := t.TempDir()
	builtIn := filepath.Join(tmpDir, "builtin")
	user := filepath.Join(tmpDir, "user")

	createTestAgent(t, builtIn, "numbuh-1", "Nigel", "Analyst")
	createTestAgent(t, user, "extra", "Extra Agent", "Extra")

	reg := NewRegistry(builtIn)
	reg.ReloadMultipleDirs(builtIn, user)

	if reg.Count() != 2 {
		t.Errorf("expected 2 agents, got %d", reg.Count())
	}
}
