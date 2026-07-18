package pipeline

import (
	"testing"
)

func TestParseMeta_Valid(t *testing.T) {
	output := `Here's what I did:
- Fixed the auth bug
- Updated tests

{"__moonbase_meta": {"files_changed": ["src/auth.ts", "src/auth.test.ts"], "risk": "LOW", "decisions": ["Used existing bcrypt pattern"]}}`

	meta := ParseMeta(output)
	if meta == nil {
		t.Fatal("expected meta to be parsed, got nil")
	}
	if len(meta.FilesChanged) != 2 {
		t.Errorf("expected 2 files, got %d", len(meta.FilesChanged))
	}
	if meta.FilesChanged[0] != "src/auth.ts" {
		t.Errorf("expected src/auth.ts, got %s", meta.FilesChanged[0])
	}
	if meta.Risk != "LOW" {
		t.Errorf("expected LOW, got %s", meta.Risk)
	}
	if len(meta.Decisions) != 1 {
		t.Errorf("expected 1 decision, got %d", len(meta.Decisions))
	}
}

func TestParseMeta_WithACResults(t *testing.T) {
	output := `All acceptance criteria pass.

{"__moonbase_meta": {"risk": "LOW", "ac_results": {"AC-1": "PASS", "AC-2": "PASS", "AC-3": "FAIL"}}}`

	meta := ParseMeta(output)
	if meta == nil {
		t.Fatal("expected meta to be parsed, got nil")
	}
	if len(meta.ACResults) != 3 {
		t.Errorf("expected 3 AC results, got %d", len(meta.ACResults))
	}
	if meta.ACResults["AC-3"] != "FAIL" {
		t.Errorf("expected AC-3=FAIL, got %s", meta.ACResults["AC-3"])
	}
}

func TestParseMeta_NoMeta(t *testing.T) {
	output := "Just a normal response with no structured data."
	meta := ParseMeta(output)
	if meta != nil {
		t.Error("expected nil for output without meta block")
	}
}

func TestParseMeta_MalformedJSON(t *testing.T) {
	output := `{"__moonbase_meta": {"files_changed": ["broken`
	meta := ParseMeta(output)
	if meta != nil {
		t.Error("expected nil for malformed JSON")
	}
}

func TestParseMeta_NestedJSON(t *testing.T) {
	// Agent output might contain other JSON before the meta block
	output := `Here's the config:
{"port": 3001, "host": "localhost"}

And here's the meta:
{"__moonbase_meta": {"files_changed": ["config.ts"], "risk": "MEDIUM"}}`

	meta := ParseMeta(output)
	if meta == nil {
		t.Fatal("expected meta to be parsed, got nil")
	}
	if meta.Risk != "MEDIUM" {
		t.Errorf("expected MEDIUM, got %s", meta.Risk)
	}
	if len(meta.FilesChanged) != 1 || meta.FilesChanged[0] != "config.ts" {
		t.Errorf("unexpected files: %v", meta.FilesChanged)
	}
}

func TestParseMeta_EmptyMeta(t *testing.T) {
	output := `{"__moonbase_meta": {}}`
	meta := ParseMeta(output)
	if meta == nil {
		t.Fatal("expected meta to be parsed (even if empty), got nil")
	}
	if len(meta.FilesChanged) != 0 {
		t.Errorf("expected 0 files, got %d", len(meta.FilesChanged))
	}
}
