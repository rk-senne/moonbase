package main

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	moonbase "github.com/rk-senne/moonbase"
	"github.com/rk-senne/moonbase/internal/discovery"
)

func TestWriteEmbeddedSkills_WritesAllToKiroNativeDirs(t *testing.T) {
	targetDir := t.TempDir()

	count, err := writeEmbeddedSkills(targetDir)
	if err != nil {
		t.Fatalf("writeEmbeddedSkills error: %v", err)
	}
	if count < 10 {
		t.Fatalf("expected >=10 skills written, got %d", count)
	}

	// Verify each embedded skill has a corresponding directory with SKILL.md
	sfs, err := moonbase.SkillsFS()
	if err != nil {
		t.Fatalf("SkillsFS() error: %v", err)
	}
	entries, err := fs.Glob(sfs, "*.md")
	if err != nil {
		t.Fatalf("Glob error: %v", err)
	}

	for _, entry := range entries {
		stem := strings.TrimSuffix(entry, ".md")
		skillPath := filepath.Join(targetDir, stem, "SKILL.md")
		info, err := os.Stat(skillPath)
		if err != nil {
			t.Errorf("expected skill file at %s: %v", skillPath, err)
			continue
		}
		if info.Size() == 0 {
			t.Errorf("skill file %s is empty", skillPath)
		}
	}
}

func TestWriteEmbeddedSkills_ContentMatchesSource(t *testing.T) {
	targetDir := t.TempDir()

	_, err := writeEmbeddedSkills(targetDir)
	if err != nil {
		t.Fatalf("writeEmbeddedSkills error: %v", err)
	}

	sfs, err := moonbase.SkillsFS()
	if err != nil {
		t.Fatalf("SkillsFS() error: %v", err)
	}
	entries, err := fs.Glob(sfs, "*.md")
	if err != nil {
		t.Fatalf("Glob error: %v", err)
	}

	for _, entry := range entries {
		stem := strings.TrimSuffix(entry, ".md")
		expected, err := fs.ReadFile(sfs, entry)
		if err != nil {
			t.Fatalf("reading embedded %s: %v", entry, err)
		}
		actual, err := os.ReadFile(filepath.Join(targetDir, stem, "SKILL.md"))
		if err != nil {
			t.Fatalf("reading scaffolded %s/SKILL.md: %v", stem, err)
		}
		if string(actual) != string(expected) {
			t.Errorf("skill %s: scaffolded content does not match embedded source", stem)
		}
	}
}

func TestWriteEmbeddedSkills_DiscoveryRegistryLoadsAll(t *testing.T) {
	targetDir := t.TempDir()

	count, err := writeEmbeddedSkills(targetDir)
	if err != nil {
		t.Fatalf("writeEmbeddedSkills error: %v", err)
	}

	// Use the SkillRegistry to load metadata from scaffolded files
	registry := discovery.NewSkillRegistry()

	// Walk the scaffolded directory structure and register each skill
	entries, err := os.ReadDir(targetDir)
	if err != nil {
		t.Fatalf("reading target dir: %v", err)
	}

	registered := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		skillFile := filepath.Join(targetDir, entry.Name(), "SKILL.md")
		if _, serr := os.Stat(skillFile); serr != nil {
			continue
		}
		meta, perr := discovery.ParseFrontmatterOnlyFromPath(skillFile)
		if perr != nil {
			t.Errorf("ParseFrontmatterOnlyFromPath(%s): %v", skillFile, perr)
			continue
		}
		registry.Register(meta)
		registered++
	}

	if registered != count {
		t.Errorf("registered %d skills, expected %d (count from writeEmbeddedSkills)", registered, count)
	}

	// Verify registry lists all skills (metadata-only, progressive loading)
	listed := registry.List()
	if len(listed) != count {
		t.Errorf("registry.List() returned %d skills, expected %d", len(listed), count)
	}

	// Verify each skill has metadata populated (name and description)
	for _, meta := range listed {
		if meta.Name == "" {
			t.Error("registry entry has empty name")
		}
		if meta.Description == "" {
			t.Errorf("registry entry %q has empty description", meta.Name)
		}
	}
}

func TestRunInit_ScaffoldsAllSkills(t *testing.T) {
	tmpDir := t.TempDir()
	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)
	os.Chdir(tmpDir)

	captureStdout(func() {
		runInit()
	})

	skillsDir := filepath.Join(tmpDir, ".kiro", "skills")

	// README.md must exist
	if _, err := os.Stat(filepath.Join(skillsDir, "README.md")); err != nil {
		t.Error("expected .kiro/skills/README.md to exist")
	}

	// Every embedded skill must be scaffolded as <name>/SKILL.md
	sfs, err := moonbase.SkillsFS()
	if err != nil {
		t.Fatalf("SkillsFS() error: %v", err)
	}
	entries, err := fs.Glob(sfs, "*.md")
	if err != nil {
		t.Fatalf("Glob error: %v", err)
	}

	for _, entry := range entries {
		stem := strings.TrimSuffix(entry, ".md")
		skillPath := filepath.Join(skillsDir, stem, "SKILL.md")
		info, err := os.Stat(skillPath)
		if err != nil {
			t.Errorf("expected skill at %s: %v", skillPath, err)
			continue
		}
		if info.Size() == 0 {
			t.Errorf("skill file %s is empty", skillPath)
		}
	}
}
