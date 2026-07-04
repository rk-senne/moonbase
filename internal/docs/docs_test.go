package docs

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscover_FindsMarkdownFiles(t *testing.T) {
	// Run from moonbase root — should find docs/*.md
	docs := Discover()

	if len(docs) == 0 {
		t.Skip("no docs found (may not be running from project root)")
	}

	// Should find at least README.md or docs/design.md
	foundMD := false
	for _, d := range docs {
		if filepath.Ext(d.Path) == ".md" {
			foundMD = true
			break
		}
	}
	if !foundMD {
		t.Error("expected at least one .md file in discovered docs")
	}
}

func TestDiscover_SortedByName(t *testing.T) {
	docs := Discover()
	for i := 1; i < len(docs); i++ {
		if docs[i].Name < docs[i-1].Name {
			t.Errorf("docs not sorted: %s before %s", docs[i-1].Name, docs[i].Name)
		}
	}
}

func TestDiscover_NoDuplicates(t *testing.T) {
	docs := Discover()
	seen := make(map[string]bool)
	for _, d := range docs {
		if seen[d.Path] {
			t.Errorf("duplicate doc path: %s", d.Path)
		}
		seen[d.Path] = true
	}
}

func TestRender_ValidMarkdown(t *testing.T) {
	// Create a temp markdown file
	tmpDir := t.TempDir()
	mdFile := filepath.Join(tmpDir, "test.md")
	content := "# Hello\n\nThis is **bold** and *italic*.\n\n- item 1\n- item 2\n"
	os.WriteFile(mdFile, []byte(content), 0o644)

	output, err := Render(mdFile, 80)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}
	if output == "" {
		t.Error("expected non-empty rendered output")
	}
	// Glamour renders to terminal — content should still be present
	if len(output) < len(content)/2 {
		t.Error("rendered output suspiciously short")
	}
}

func TestRender_NonexistentFile(t *testing.T) {
	_, err := Render("/nonexistent/file.md", 80)
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestRender_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	mdFile := filepath.Join(tmpDir, "empty.md")
	os.WriteFile(mdFile, []byte(""), 0o644)

	output, err := Render(mdFile, 80)
	if err != nil {
		t.Fatalf("Render failed on empty file: %v", err)
	}
	_ = output // Empty is valid
}

func TestRender_PlainTextFile(t *testing.T) {
	tmpDir := t.TempDir()
	txtFile := filepath.Join(tmpDir, "plain.txt")
	content := "Just plain text, no markdown formatting at all."
	os.WriteFile(txtFile, []byte(content), 0o644)

	output, err := Render(txtFile, 80)
	if err != nil {
		t.Fatalf("Render failed on plain text: %v", err)
	}
	if output == "" {
		t.Error("expected non-empty output for plain text file")
	}
}

func TestRender_NarrowWidth(t *testing.T) {
	tmpDir := t.TempDir()
	mdFile := filepath.Join(tmpDir, "narrow.md")
	content := "# Title\n\nThis is a very long line that should be word-wrapped when rendered with a narrow width setting for the terminal output.\n"
	os.WriteFile(mdFile, []byte(content), 0o644)

	output, err := Render(mdFile, 20)
	if err != nil {
		t.Fatalf("Render failed with narrow width: %v", err)
	}
	if output == "" {
		t.Error("expected non-empty output")
	}
}

func TestDiscover_FromTempDir_NoResults(t *testing.T) {
	// Change to a temp dir with no docs/, .kiro/, wiki/, spec/ dirs
	tmpDir := t.TempDir()
	orig, _ := os.Getwd()
	defer os.Chdir(orig)
	os.Chdir(tmpDir)

	docs := Discover()
	// Should return empty list (no md/txt files found)
	if len(docs) != 0 {
		t.Errorf("expected 0 docs from empty temp dir, got %d", len(docs))
	}
}

func TestDiscover_FromTempDir_FindsMDAndTXT(t *testing.T) {
	tmpDir := t.TempDir()
	orig, _ := os.Getwd()
	defer os.Chdir(orig)
	os.Chdir(tmpDir)

	// Create docs/ with markdown and txt files
	os.MkdirAll(filepath.Join(tmpDir, "docs"), 0o755)
	os.WriteFile(filepath.Join(tmpDir, "docs", "guide.md"), []byte("# Guide"), 0o644)
	os.WriteFile(filepath.Join(tmpDir, "docs", "notes.txt"), []byte("notes"), 0o644)
	os.WriteFile(filepath.Join(tmpDir, "docs", "image.png"), []byte("png"), 0o644) // should be ignored

	docs := Discover()
	if len(docs) < 2 {
		t.Errorf("expected at least 2 docs (guide.md, notes.txt), got %d", len(docs))
	}

	// Verify .png is NOT included
	for _, d := range docs {
		if filepath.Ext(d.Path) == ".png" {
			t.Error("should not include .png files")
		}
	}

	// Verify we find both .md and .txt
	foundMD := false
	foundTXT := false
	for _, d := range docs {
		if filepath.Ext(d.Path) == ".md" {
			foundMD = true
		}
		if filepath.Ext(d.Path) == ".txt" {
			foundTXT = true
		}
	}
	if !foundMD {
		t.Error("expected to find .md file")
	}
	if !foundTXT {
		t.Error("expected to find .txt file")
	}
}

func TestDiscover_SkipsHiddenDirsExceptKiro(t *testing.T) {
	tmpDir := t.TempDir()
	orig, _ := os.Getwd()
	defer os.Chdir(orig)
	os.Chdir(tmpDir)

	// Create .hidden/ dir with a markdown file (should be skipped)
	os.MkdirAll(filepath.Join(tmpDir, ".hidden"), 0o755)
	os.WriteFile(filepath.Join(tmpDir, ".hidden", "secret.md"), []byte("# Secret"), 0o644)

	// Create .kiro/steering/ with a markdown file (should be included)
	os.MkdirAll(filepath.Join(tmpDir, ".kiro", "steering"), 0o755)
	os.WriteFile(filepath.Join(tmpDir, ".kiro", "steering", "rules.md"), []byte("# Rules"), 0o644)

	docs := Discover()

	foundKiro := false
	foundHidden := false
	for _, d := range docs {
		if filepath.Base(d.Path) == "rules.md" {
			foundKiro = true
		}
		if filepath.Base(d.Path) == "secret.md" {
			foundHidden = true
		}
	}

	if !foundKiro {
		t.Error("expected to find .kiro/steering/rules.md")
	}
	if foundHidden {
		t.Error("should not find files in .hidden/ directory")
	}
}

func TestDiscover_SkipsDeepNesting(t *testing.T) {
	tmpDir := t.TempDir()
	orig, _ := os.Getwd()
	defer os.Chdir(orig)
	os.Chdir(tmpDir)

	// Create deeply nested structure: docs/a/b/c/d/deep.md (depth > 3)
	deepPath := filepath.Join(tmpDir, "docs", "a", "b", "c", "d")
	os.MkdirAll(deepPath, 0o755)
	os.WriteFile(filepath.Join(deepPath, "deep.md"), []byte("# Deep"), 0o644)

	// Create shallow file: docs/shallow.md
	os.WriteFile(filepath.Join(tmpDir, "docs", "shallow.md"), []byte("# Shallow"), 0o644)

	docs := Discover()

	foundShallow := false
	foundDeep := false
	for _, d := range docs {
		if filepath.Base(d.Path) == "shallow.md" {
			foundShallow = true
		}
		if filepath.Base(d.Path) == "deep.md" {
			foundDeep = true
		}
	}

	if !foundShallow {
		t.Error("expected to find shallow.md")
	}
	if foundDeep {
		t.Error("should not find files nested more than 3 levels deep")
	}
}

func TestDiscover_DeduplicatesOverlappingSearchDirs(t *testing.T) {
	tmpDir := t.TempDir()
	orig, _ := os.Getwd()
	defer os.Chdir(orig)
	os.Chdir(tmpDir)

	// "." search will find root-level README.md
	// If docs/ also exists, ensure no duplicates from overlapping walks
	os.WriteFile(filepath.Join(tmpDir, "README.md"), []byte("# README"), 0o644)

	docs := Discover()

	seen := make(map[string]int)
	for _, d := range docs {
		seen[d.Path]++
	}
	for path, count := range seen {
		if count > 1 {
			t.Errorf("duplicate entry for %s (count: %d)", path, count)
		}
	}
}

func TestDoc_FieldsPopulated(t *testing.T) {
	tmpDir := t.TempDir()
	orig, _ := os.Getwd()
	defer os.Chdir(orig)
	os.Chdir(tmpDir)

	// Create a docs/ directory with a markdown file (Discover searches "docs" dir)
	os.MkdirAll(filepath.Join(tmpDir, "docs"), 0o755)
	os.WriteFile(filepath.Join(tmpDir, "docs", "test.md"), []byte("# Test"), 0o644)

	docs := Discover()
	if len(docs) == 0 {
		t.Fatal("expected at least 1 doc")
	}

	for _, d := range docs {
		if d.Name == "" {
			t.Error("Doc.Name should not be empty")
		}
		if d.Path == "" {
			t.Error("Doc.Path should not be empty")
		}
	}
}
