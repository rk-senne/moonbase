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
