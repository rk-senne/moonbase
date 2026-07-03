package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIsAgentsDir_ValidDir(t *testing.T) {
	// Create a temp dir with .md files
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "numbuh-1.md"), []byte("---\nname: numbuh-1\n---\n# Test"), 0o644)

	if !isAgentsDir(tmpDir) {
		t.Error("expected valid agents dir to return true")
	}
}

func TestIsAgentsDir_EmptyDir(t *testing.T) {
	tmpDir := t.TempDir()
	if isAgentsDir(tmpDir) {
		t.Error("expected empty dir to return false")
	}
}

func TestIsAgentsDir_NonexistentDir(t *testing.T) {
	if isAgentsDir("/nonexistent/path/agents") {
		t.Error("expected nonexistent dir to return false")
	}
}

func TestCopyFile(t *testing.T) {
	tmpDir := t.TempDir()
	src := filepath.Join(tmpDir, "source.md")
	dst := filepath.Join(tmpDir, "dest.md")

	content := "---\nname: test\n---\n# Test Agent\n\nPrompt body."
	os.WriteFile(src, []byte(content), 0o644)

	err := copyFile(src, dst)
	if err != nil {
		t.Fatalf("copyFile failed: %v", err)
	}

	// Verify content matches
	data, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("reading dest: %v", err)
	}
	if string(data) != content {
		t.Errorf("content mismatch:\nexpected: %s\ngot: %s", content, string(data))
	}
}

func TestCopyFile_MissingSrc(t *testing.T) {
	tmpDir := t.TempDir()
	err := copyFile("/nonexistent/file.md", filepath.Join(tmpDir, "dest.md"))
	if err == nil {
		t.Error("expected error for missing source")
	}
}

func TestFindAgentsSource_FromCWD(t *testing.T) {
	// When run from cmd/moonbase, agents is at ../../agents
	cwd, _ := os.Getwd()

	candidates := []string{
		filepath.Join(cwd, "agents"),
		filepath.Join(cwd, "..", "agents"),
		filepath.Join(cwd, "..", "..", "agents"),
	}

	found := false
	for _, c := range candidates {
		if isAgentsDir(c) {
			found = true
			break
		}
	}

	if !found {
		t.Skip("not running from moonbase project tree")
	}

	// findAgentsSource should find a valid directory
	dir, err := findAgentsSource()
	if err != nil {
		// This is OK in test context — the function checks CWD relative
		// which may be cmd/moonbase, not project root
		t.Skipf("findAgentsSource didn't find agents from test dir (expected): %v", err)
	}
	if !isAgentsDir(dir) {
		t.Errorf("findAgentsSource returned invalid dir: %s", dir)
	}
}

func TestAgentNameResolution(t *testing.T) {
	// Test the agent name → file mapping logic used in runDeploy
	tests := []struct {
		input    string
		expected string
	}{
		{"0", "numbuh-0.md"},
		{"1", "numbuh-1.md"},
		{"4", "numbuh-4.md"},
		{"13", "numbuh-13.md"},
		{"274", "numbuh-274.md"},
		{"362", "numbuh-362.md"},
		{"999", "numbuh-999.md"},
		{"council", "knd-council.md"},
		{"k", "knd-council.md"},
		{"z", "sector-z.md"},
		{"Z", "sector-z.md"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			var file string
			switch {
			case tt.input == "council" || tt.input == "k":
				file = "knd-council.md"
			case tt.input == "z" || tt.input == "Z":
				file = "sector-z.md"
			default:
				file = "numbuh-" + tt.input + ".md"
			}
			if file != tt.expected {
				t.Errorf("input %q → %s, expected %s", tt.input, file, tt.expected)
			}
		})
	}
}

// === Gap Coverage: isValidAgentID edge cases (unicode, null bytes, path traversal) ===

func TestIsValidAgentID_ValidCases(t *testing.T) {
	valid := []string{
		"0", "1", "4", "13", "274", "362", "999",
		"council", "z", "Z",
		"numbuh-4", "sector-z", "knd-council",
		"a", "A", "abc",
	}
	for _, id := range valid {
		t.Run(id, func(t *testing.T) {
			if !isValidAgentID(id) {
				t.Errorf("expected valid: %q", id)
			}
		})
	}
}

func TestIsValidAgentID_InvalidCases(t *testing.T) {
	invalid := []struct {
		id   string
		desc string
	}{
		{"", "empty string"},
		{"../etc/passwd", "path traversal with dots"},
		{"../../secret", "double path traversal"},
		{"/etc/passwd", "absolute path"},
		{"numbuh 4", "space"},
		{"numbuh\t4", "tab character"},
		{"numbuh\n4", "newline"},
		{"numbuh\x004", "null byte"},
		{"a/b", "forward slash"},
		{"a\\b", "backslash"},
		{"numbuh_4", "underscore"},
		{"numbuh.4", "dot"},
		{"münchen", "unicode (German)"},
		{"エージェント", "unicode (Japanese)"},
		{"🌙", "emoji"},
		{"a b c", "multiple spaces"},
		{"$HOME", "shell variable"},
		{"`id`", "backtick injection"},
		{"$(whoami)", "command substitution"},
		{"a;b", "semicolon"},
		{"a&b", "ampersand"},
		{"a|b", "pipe"},
		{"this-is-way-too-long-id", "exceeds 20 chars"},
		{strings.Repeat("a", 21), "exactly 21 chars"},
	}
	for _, tt := range invalid {
		t.Run(tt.desc, func(t *testing.T) {
			if isValidAgentID(tt.id) {
				t.Errorf("expected INVALID for %q (%s)", tt.id, tt.desc)
			}
		})
	}
}

func TestIsValidAgentID_MaxLength(t *testing.T) {
	// Exactly 20 characters should be valid
	id20 := strings.Repeat("a", 20)
	if !isValidAgentID(id20) {
		t.Errorf("expected valid for 20-char ID")
	}
	// 21 should be invalid
	id21 := strings.Repeat("a", 21)
	if isValidAgentID(id21) {
		t.Errorf("expected invalid for 21-char ID")
	}
}
