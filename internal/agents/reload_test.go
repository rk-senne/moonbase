package agents

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

const testAgentMD = `---
name: numbuh-test
designation: Test Op
role: Tester
tools: [read]
---

# Test Op

Body.
`

func writeAgent(t *testing.T, dir, name, body string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatalf("writing %s: %v", p, err)
	}
	return p
}

func TestReloadIfChanged(t *testing.T) {
	dir := t.TempDir()
	writeAgent(t, dir, "numbuh-test.md", testAgentMD)

	reg := NewRegistry(dir)

	// First check must load (lastSig starts zero).
	if !reg.ReloadIfChanged() {
		t.Fatal("expected first ReloadIfChanged to load agents")
	}
	if reg.Count() != 1 {
		t.Fatalf("expected 1 agent after first load, got %d", reg.Count())
	}

	// No change → no reload.
	if reg.ReloadIfChanged() {
		t.Error("expected no reload when nothing changed")
	}

	// Add a new agent file; the directory mtime advances → reload.
	time.Sleep(10 * time.Millisecond) // ensure a distinct mtime
	writeAgent(t, dir, "numbuh-two.md", testAgentMD)
	if !reg.ReloadIfChanged() {
		t.Error("expected reload after a new agent file was added")
	}
	if reg.Count() != 2 {
		t.Errorf("expected 2 agents after adding one, got %d", reg.Count())
	}

	// Stable again.
	if reg.ReloadIfChanged() {
		t.Error("expected no reload on a stable directory")
	}
}

func TestReloadIfChanged_MissingDir(t *testing.T) {
	reg := NewRegistry(filepath.Join(t.TempDir(), "does-not-exist"))
	if reg.ReloadIfChanged() {
		t.Error("expected no reload for a missing directory")
	}
}
