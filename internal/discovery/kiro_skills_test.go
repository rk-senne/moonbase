package discovery

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEmitKiroSkillResources_CreatesDirectories(t *testing.T) {
	srcDir := t.TempDir()
	outDir := filepath.Join(t.TempDir(), "output")

	// Create a skill source file
	path := filepath.Join(srcDir, "api-patterns.md")
	os.WriteFile(path, []byte("---\nname: api-patterns\ndescription: REST API patterns\n---\n\n# API Patterns\n\nContent.\n"), 0o644)

	r := NewSkillRegistry()
	r.Register(SkillMeta{Name: "api-patterns", Description: "REST API patterns", Path: path})

	err := EmitKiroSkillResources(r, outDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check directory was created
	info, err := os.Stat(filepath.Join(outDir, "api-patterns"))
	if err != nil {
		t.Fatalf("expected directory: %v", err)
	}
	if !info.IsDir() {
		t.Error("expected a directory")
	}
	if info.Mode().Perm() != 0o700 {
		t.Errorf("expected 0700 permissions, got %o", info.Mode().Perm())
	}
}

func TestEmitKiroSkillResources_FlatFileConverted(t *testing.T) {
	srcDir := t.TempDir()
	outDir := filepath.Join(t.TempDir(), "output")

	// Flat file with frontmatter
	path := filepath.Join(srcDir, "git-workflow.md")
	os.WriteFile(path, []byte("---\nname: git-workflow\ndescription: Git workflow\n---\n\n# Git\n\nContent.\n"), 0o644)

	r := NewSkillRegistry()
	r.Register(SkillMeta{Name: "git-workflow", Description: "Git workflow", Path: path})

	err := EmitKiroSkillResources(r, outDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check SKILL.md was created
	outPath := filepath.Join(outDir, "git-workflow", "SKILL.md")
	content, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("expected SKILL.md: %v", err)
	}
	if !strings.Contains(string(content), "name: git-workflow") {
		t.Error("expected frontmatter in output")
	}
	if !strings.Contains(string(content), "# Git") {
		t.Error("expected body in output")
	}

	// Check permissions
	info, err := os.Stat(outPath)
	if err != nil {
		t.Fatalf("stat failed: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("expected 0600 file permissions, got %o", info.Mode().Perm())
	}
}

func TestEmitKiroSkillResources_DirectoryStyleCopied(t *testing.T) {
	srcDir := t.TempDir()
	outDir := filepath.Join(t.TempDir(), "output")

	// Directory-style skill (path ends with SKILL.md)
	skillDir := filepath.Join(srcDir, "docker-build")
	os.MkdirAll(skillDir, 0o755)
	path := filepath.Join(skillDir, "SKILL.md")
	original := "---\nname: docker-build\ndescription: Docker patterns\n---\n\n# Docker\n\nPatterns here.\n"
	os.WriteFile(path, []byte(original), 0o644)

	r := NewSkillRegistry()
	r.Register(SkillMeta{Name: "docker-build", Description: "Docker patterns", Path: path})

	err := EmitKiroSkillResources(r, outDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	outPath := filepath.Join(outDir, "docker-build", "SKILL.md")
	content, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("expected SKILL.md: %v", err)
	}
	if string(content) != original {
		t.Errorf("expected exact copy for directory-style skill, got:\n%s", string(content))
	}
}

func TestEmitKiroSkillResources_EmptyRegistry(t *testing.T) {
	outDir := filepath.Join(t.TempDir(), "output")

	r := NewSkillRegistry()
	err := EmitKiroSkillResources(r, outDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Output dir should not exist (nothing emitted)
	if _, err := os.Stat(outDir); err == nil {
		t.Error("expected output dir to not be created for empty registry")
	}
}

func TestEmitKiroSkillResources_OutputDirCreated(t *testing.T) {
	srcDir := t.TempDir()
	outDir := filepath.Join(t.TempDir(), "nested", "output")

	path := filepath.Join(srcDir, "test.md")
	os.WriteFile(path, []byte("---\nname: test\ndescription: Test\n---\n\nBody.\n"), 0o644)

	r := NewSkillRegistry()
	r.Register(SkillMeta{Name: "test", Description: "Test", Path: path})

	err := EmitKiroSkillResources(r, outDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := os.Stat(filepath.Join(outDir, "test", "SKILL.md")); err != nil {
		t.Fatalf("expected output file to exist: %v", err)
	}
}
