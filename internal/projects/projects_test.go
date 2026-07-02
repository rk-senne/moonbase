package projects

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscover_DoesNotPanic(t *testing.T) {
	// Should never panic regardless of what directories exist
	projects := Discover()
	_ = projects
}

func TestDiscover_FindsGoProjects(t *testing.T) {
	projects := Discover()

	// If any Go projects exist in standard dev dirs, they should be found
	for _, p := range projects {
		if p.Type == "go" {
			// Verify the path actually has a go.mod
			if _, err := os.Stat(filepath.Join(p.Path, "go.mod")); err != nil {
				t.Errorf("project %s marked as go but no go.mod at %s", p.Name, p.Path)
			}
			return // Found at least one — good
		}
	}
	// Not finding any Go projects is OK (depends on system)
}

func TestDetectType_Go(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module test"), 0o644)

	ptype := detectType(tmpDir)
	if ptype != "go" {
		t.Errorf("expected 'go', got: %s", ptype)
	}
}

func TestDetectType_Node(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte("{}"), 0o644)

	ptype := detectType(tmpDir)
	if ptype != "node" {
		t.Errorf("expected 'node', got: %s", ptype)
	}
}

func TestDetectType_Java(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "pom.xml"), []byte("<project/>"), 0o644)

	ptype := detectType(tmpDir)
	if ptype != "java" {
		t.Errorf("expected 'java', got: %s", ptype)
	}
}

func TestDetectType_Rust(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "Cargo.toml"), []byte("[package]"), 0o644)

	ptype := detectType(tmpDir)
	if ptype != "rust" {
		t.Errorf("expected 'rust', got: %s", ptype)
	}
}

func TestDetectType_Git(t *testing.T) {
	tmpDir := t.TempDir()
	os.MkdirAll(filepath.Join(tmpDir, ".git"), 0o755)

	ptype := detectType(tmpDir)
	if ptype != "git" {
		t.Errorf("expected 'git', got: %s", ptype)
	}
}

func TestDetectType_Empty(t *testing.T) {
	tmpDir := t.TempDir()

	ptype := detectType(tmpDir)
	if ptype != "" {
		t.Errorf("expected empty string for unrecognized dir, got: %s", ptype)
	}
}

func TestDiscover_SortedByName(t *testing.T) {
	projects := Discover()
	for i := 1; i < len(projects); i++ {
		if projects[i].Name < projects[i-1].Name {
			t.Errorf("projects not sorted: %s before %s", projects[i-1].Name, projects[i].Name)
		}
	}
}
