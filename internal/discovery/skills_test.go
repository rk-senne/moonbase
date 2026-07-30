package discovery

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
)

// === parseFrontmatterOnly tests ===

func TestParseFrontmatterOnly_Valid(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "docker-build.md")
	content := "---\nname: docker-build\ndescription: Docker multi-stage build patterns.\n---\n\n# Docker Build\n\nLong body content here.\n"
	os.WriteFile(path, []byte(content), 0o644)

	meta, err := parseFrontmatterOnly(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if meta.Name != "docker-build" {
		t.Errorf("expected name 'docker-build', got %q", meta.Name)
	}
	if meta.Description != "Docker multi-stage build patterns." {
		t.Errorf("expected description, got %q", meta.Description)
	}
	if meta.Path != path {
		t.Errorf("expected path %q, got %q", path, meta.Path)
	}
	if meta.Legacy {
		t.Error("expected Legacy=false for valid frontmatter")
	}
}

func TestParseFrontmatterOnly_MissingFrontmatter(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "legacy.md")
	os.WriteFile(path, []byte("# Just Markdown\n\nNo frontmatter here.\n"), 0o644)

	_, err := parseFrontmatterOnly(path)
	if err == nil {
		t.Fatal("expected error for missing frontmatter")
	}
}

func TestParseFrontmatterOnly_EmptyName(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty-name.md")
	content := "---\nname: \"\"\ndescription: Something\n---\n\nBody.\n"
	os.WriteFile(path, []byte(content), 0o644)

	_, err := parseFrontmatterOnly(path)
	if err == nil {
		t.Fatal("expected error for empty name")
	}
}

func TestParseFrontmatterOnly_NameTooLong(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "long-name.md")
	longName := "a"
	for i := 0; i < 65; i++ {
		longName += "a"
	}
	content := "---\nname: " + longName + "\ndescription: Too long\n---\n\nBody.\n"
	os.WriteFile(path, []byte(content), 0o644)

	_, err := parseFrontmatterOnly(path)
	if err == nil {
		t.Fatal("expected error for name exceeding 64 chars")
	}
}

func TestParseFrontmatterOnly_InvalidCharsInName(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "invalid.md")
	content := "---\nname: \"My Skill!\"\ndescription: Invalid chars\n---\n\nBody.\n"
	os.WriteFile(path, []byte(content), 0o644)

	_, err := parseFrontmatterOnly(path)
	if err == nil {
		t.Fatal("expected error for invalid chars in name")
	}
}

func TestParseFrontmatterOnly_NormalizesName(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "normalize.md")
	content := "---\nname: Docker_Build\ndescription: Normalized\n---\n\nBody.\n"
	os.WriteFile(path, []byte(content), 0o644)

	meta, err := parseFrontmatterOnly(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if meta.Name != "docker-build" {
		t.Errorf("expected normalized name 'docker-build', got %q", meta.Name)
	}
}

func TestParseFrontmatterOnly_LargeFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "large.md")
	// Frontmatter is small but body is large (>1KB)
	content := "---\nname: large-skill\ndescription: Large body.\n---\n\n"
	for i := 0; i < 200; i++ {
		content += "This is line number that adds bulk to the file content.\n"
	}
	os.WriteFile(path, []byte(content), 0o644)

	meta, err := parseFrontmatterOnly(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if meta.Name != "large-skill" {
		t.Errorf("expected name 'large-skill', got %q", meta.Name)
	}
}

// === SkillRegistry tests ===

func TestSkillRegistry_Register(t *testing.T) {
	r := NewSkillRegistry()
	r.Register(SkillMeta{Name: "alpha", Description: "First", Path: "/a.md"})
	r.Register(SkillMeta{Name: "beta", Description: "Second", Path: "/b.md"})

	if r.Len() != 2 {
		t.Fatalf("expected 2 skills, got %d", r.Len())
	}
}

func TestSkillRegistry_Register_DuplicateIgnored(t *testing.T) {
	r := NewSkillRegistry()
	r.Register(SkillMeta{Name: "alpha", Description: "First", Path: "/a.md"})
	r.Register(SkillMeta{Name: "alpha", Description: "Duplicate", Path: "/a2.md"})

	if r.Len() != 1 {
		t.Fatalf("expected 1 skill (duplicate ignored), got %d", r.Len())
	}
	meta := r.Get("alpha")
	if meta.Description != "First" {
		t.Errorf("expected first registration to win, got description %q", meta.Description)
	}
}

func TestSkillRegistry_List_PreservesOrder(t *testing.T) {
	r := NewSkillRegistry()
	r.Register(SkillMeta{Name: "charlie", Path: "/c.md"})
	r.Register(SkillMeta{Name: "alpha", Path: "/a.md"})
	r.Register(SkillMeta{Name: "bravo", Path: "/b.md"})

	list := r.List()
	if len(list) != 3 {
		t.Fatalf("expected 3 skills, got %d", len(list))
	}
	expected := []string{"charlie", "alpha", "bravo"}
	for i, meta := range list {
		if meta.Name != expected[i] {
			t.Errorf("position %d: expected %q, got %q", i, expected[i], meta.Name)
		}
	}
}

func TestSkillRegistry_Get_Found(t *testing.T) {
	r := NewSkillRegistry()
	r.Register(SkillMeta{Name: "target", Description: "Found me", Path: "/t.md"})

	meta := r.Get("target")
	if meta == nil {
		t.Fatal("expected to find 'target'")
	}
	if meta.Description != "Found me" {
		t.Errorf("expected description 'Found me', got %q", meta.Description)
	}
}

func TestSkillRegistry_Get_NotFound(t *testing.T) {
	r := NewSkillRegistry()
	r.Register(SkillMeta{Name: "exists", Path: "/e.md"})

	meta := r.Get("nope")
	if meta != nil {
		t.Error("expected nil for non-existent skill")
	}
}

func TestSkillRegistry_Names(t *testing.T) {
	r := NewSkillRegistry()
	r.Register(SkillMeta{Name: "one", Path: "/1.md"})
	r.Register(SkillMeta{Name: "two", Path: "/2.md"})

	names := r.Names()
	if len(names) != 2 {
		t.Fatalf("expected 2 names, got %d", len(names))
	}
	if names[0] != "one" || names[1] != "two" {
		t.Errorf("expected [one, two], got %v", names)
	}
}

func TestSkillRegistry_LoadContent_CachesResult(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cached.md")
	content := "---\nname: cached\ndescription: Test caching\n---\n\n# Cached Content\n\nBody here.\n"
	os.WriteFile(path, []byte(content), 0o644)

	r := NewSkillRegistry()
	r.Register(SkillMeta{Name: "cached", Path: path})

	// First load
	body1, err := r.LoadContent("cached")
	if err != nil {
		t.Fatalf("first load failed: %v", err)
	}
	if body1 == "" {
		t.Fatal("expected non-empty content")
	}
	if body1 != "# Cached Content\n\nBody here." {
		t.Errorf("unexpected body: %q", body1)
	}

	// Delete file — second load should return cached content
	os.Remove(path)

	body2, err := r.LoadContent("cached")
	if err != nil {
		t.Fatalf("cached load failed: %v", err)
	}
	if body2 != body1 {
		t.Error("expected cached result to match first load")
	}
}

func TestSkillRegistry_LoadContent_StripsFrontmatter(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "strip.md")
	content := "---\nname: strip\ndescription: Test strip\n---\n\nBody only.\n"
	os.WriteFile(path, []byte(content), 0o644)

	r := NewSkillRegistry()
	r.Register(SkillMeta{Name: "strip", Path: path})

	body, err := r.LoadContent("strip")
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if body != "Body only." {
		t.Errorf("expected 'Body only.', got %q", body)
	}
}

func TestSkillRegistry_LoadContent_Truncates(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "big.md")
	// Create content larger than maxSkillSize
	body := ""
	for i := 0; i < maxSkillSize+500; i++ {
		body += "x"
	}
	content := "---\nname: big\ndescription: Big\n---\n\n" + body + "\n"
	os.WriteFile(path, []byte(content), 0o644)

	r := NewSkillRegistry()
	r.Register(SkillMeta{Name: "big", Path: path})

	loaded, err := r.LoadContent("big")
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if len(loaded) > maxSkillSize+50 {
		t.Errorf("expected content truncated to ~maxSkillSize, got length %d", len(loaded))
	}
	if loaded[len(loaded)-len("...(truncated)"):] != "...(truncated)" {
		t.Error("expected truncation suffix")
	}
}

func TestSkillRegistry_LoadContent_NotFound(t *testing.T) {
	r := NewSkillRegistry()

	_, err := r.LoadContent("nonexistent")
	if err == nil {
		t.Fatal("expected error for non-existent skill")
	}
}

func TestSkillRegistry_LoadContent_FileDeleted(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "gone.md")
	content := "---\nname: gone\ndescription: Will be deleted\n---\n\nContent.\n"
	os.WriteFile(path, []byte(content), 0o644)

	r := NewSkillRegistry()
	r.Register(SkillMeta{Name: "gone", Path: path})

	// Delete before loading
	os.Remove(path)

	_, err := r.LoadContent("gone")
	if err == nil {
		t.Fatal("expected error when file is deleted before first load")
	}
}

func TestSkillRegistry_ConcurrentAccess(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "concurrent.md")
	content := "---\nname: concurrent\ndescription: Concurrency test\n---\n\nSafe content.\n"
	os.WriteFile(path, []byte(content), 0o644)

	r := NewSkillRegistry()
	r.Register(SkillMeta{Name: "concurrent", Path: path})

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r.List()
			r.Get("concurrent")
			r.Names()
			r.LoadContent("concurrent")
		}()
	}
	wg.Wait()
}

// === ExtractSkillRequests tests ===

func TestExtractSkillRequests_SingleMatch(t *testing.T) {
	text := "Please use @skill(docker-build) for this task."
	names := ExtractSkillRequests(text)
	if len(names) != 1 || names[0] != "docker-build" {
		t.Errorf("expected [docker-build], got %v", names)
	}
}

func TestExtractSkillRequests_MultipleMatches(t *testing.T) {
	text := "Use @skill(docker-build) and @skill(git-workflow) for this."
	names := ExtractSkillRequests(text)
	if len(names) != 2 {
		t.Fatalf("expected 2 matches, got %d", len(names))
	}
	if names[0] != "docker-build" || names[1] != "git-workflow" {
		t.Errorf("expected [docker-build, git-workflow], got %v", names)
	}
}

func TestExtractSkillRequests_Deduplicates(t *testing.T) {
	text := "@skill(auth) and again @skill(auth)"
	names := ExtractSkillRequests(text)
	if len(names) != 1 {
		t.Errorf("expected 1 unique name, got %d: %v", len(names), names)
	}
}

func TestExtractSkillRequests_NoMatches(t *testing.T) {
	text := "No skill references here."
	names := ExtractSkillRequests(text)
	if names != nil {
		t.Errorf("expected nil, got %v", names)
	}
}

func TestExtractSkillRequests_InvalidCharsIgnored(t *testing.T) {
	// Uppercase and special chars don't match the pattern
	text := "@skill(Docker_Build) and @skill(valid-one)"
	names := ExtractSkillRequests(text)
	if len(names) != 1 || names[0] != "valid-one" {
		t.Errorf("expected [valid-one], got %v", names)
	}
}

// === ResolveSkills tests ===

func TestResolveSkills_FoundAndNotFound(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "found.md")
	os.WriteFile(path, []byte("---\nname: found\ndescription: Exists\n---\n\nFound body.\n"), 0o644)

	r := NewSkillRegistry()
	r.Register(SkillMeta{Name: "found", Path: path})

	resolved, notFound := ResolveSkills(r, []string{"found", "missing"})
	if len(resolved) != 1 {
		t.Fatalf("expected 1 resolved, got %d", len(resolved))
	}
	if resolved[0].Name != "found" {
		t.Errorf("expected resolved name 'found', got %q", resolved[0].Name)
	}
	if len(notFound) != 1 || notFound[0] != "missing" {
		t.Errorf("expected notFound=[missing], got %v", notFound)
	}
}

func TestResolveSkills_NilRegistry(t *testing.T) {
	resolved, notFound := ResolveSkills(nil, []string{"any"})
	if resolved != nil {
		t.Error("expected nil resolved for nil registry")
	}
	if len(notFound) != 1 {
		t.Errorf("expected 1 not-found, got %d", len(notFound))
	}
}
