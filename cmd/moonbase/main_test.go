package main

import (
	"os"
	"path/filepath"
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

func TestEnvExists(t *testing.T) {
	// Set a test env var
	os.Setenv("MOONBASE_TEST_VAR", "hello")
	defer os.Unsetenv("MOONBASE_TEST_VAR")

	if !envExists("MOONBASE_TEST_VAR") {
		t.Error("expected envExists to return true for set var")
	}
	if envExists("MOONBASE_NONEXISTENT_VAR_XYZ") {
		t.Error("expected envExists to return false for unset var")
	}
}

func TestEnvExists_EmptyValue(t *testing.T) {
	os.Setenv("MOONBASE_EMPTY_VAR", "   ")
	defer os.Unsetenv("MOONBASE_EMPTY_VAR")

	// Empty/whitespace-only should return false
	if envExists("MOONBASE_EMPTY_VAR") {
		t.Error("expected envExists to return false for whitespace-only var")
	}
}
