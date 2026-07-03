package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// captureStdout captures everything written to os.Stdout during fn().
func captureStdout(fn func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	fn()
	w.Close()
	var buf bytes.Buffer
	io.Copy(&buf, r)
	os.Stdout = old
	return buf.String()
}

// === isValidAgentID table-driven tests ===

func TestIsValidAgentID_TableDriven(t *testing.T) {
	tests := []struct {
		id    string
		valid bool
		desc  string
	}{
		// Valid cases
		{"1", true, "single digit"},
		{"274", true, "three digits"},
		{"z", true, "single lowercase letter"},
		{"Z", true, "single uppercase letter"},
		{"council", true, "word"},
		{"numbuh-4", true, "hyphenated name"},
		{"sector-z", true, "hyphenated word"},
		{"knd-council", true, "multi-hyphen"},
		{"a", true, "single char"},
		{"A", true, "single uppercase"},
		{"abc", true, "three lowercase"},
		{"ABC", true, "three uppercase"},
		{"123", true, "three digits"},
		{"a-b-c", true, "multi-hyphen short"},
		{strings.Repeat("a", 20), true, "exactly 20 chars (max)"},

		// Invalid cases
		{"", false, "empty string"},
		{"../etc", false, "path traversal dots"},
		{"../../passwd", false, "double path traversal"},
		{"/etc/passwd", false, "absolute path"},
		{strings.Repeat("a", 21), false, "21 chars (over max)"},
		{strings.Repeat("x", 50), false, "way over max"},
		{"hello world", false, "space"},
		{"hello\tworld", false, "tab"},
		{"hello\nworld", false, "newline"},
		{"hello\x00world", false, "null byte"},
		{"a/b", false, "forward slash"},
		{"a\\b", false, "backslash"},
		{"numbuh_4", false, "underscore"},
		{"numbuh.4", false, "dot"},
		{"$HOME", false, "shell variable"},
		{"`id`", false, "backtick injection"},
		{"$(whoami)", false, "command substitution"},
		{"a;b", false, "semicolon"},
		{"a&b", false, "ampersand"},
		{"a|b", false, "pipe"},
		{"a=b", false, "equals sign"},
		{"a>b", false, "greater than"},
		{"a<b", false, "less than"},
		{"münchen", false, "unicode chars"},
		{"🌙", false, "emoji"},
		{"-", true, "lone hyphen (valid char)"},
		{"--flag", true, "double hyphen start"},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			got := isValidAgentID(tt.id)
			if got != tt.valid {
				t.Errorf("isValidAgentID(%q) = %v, want %v", tt.id, got, tt.valid)
			}
		})
	}
}

// === runMissionDryRun tests ===

func TestRunMissionDryRun_PrintsPhases(t *testing.T) {
	output := captureStdout(func() {
		runMissionDryRun("add rate limiting to the API")
	})

	// Verify key sections are present
	expectedStrings := []string{
		"KND Council — Mission Dry Run",
		"add rate limiting to the API",
		"EXECUTION PLAN",
		"RISK GATE",
		"No backends will be invoked",
	}
	for _, expected := range expectedStrings {
		if !strings.Contains(output, expected) {
			t.Errorf("output missing expected string: %q\nGot:\n%s", expected, output)
		}
	}
}

func TestRunMissionDryRun_ContainsPhaseNumbers(t *testing.T) {
	output := captureStdout(func() {
		runMissionDryRun("test task")
	})

	// Should mention Phase 1 through at least Phase 5 (core pipeline)
	for i := 1; i <= 5; i++ {
		phase := "Phase " + string(rune('0'+i))
		if !strings.Contains(output, phase) {
			t.Errorf("output missing %s\nGot:\n%s", phase, output)
		}
	}
}

func TestRunMissionDryRun_ShowsRiskLevels(t *testing.T) {
	output := captureStdout(func() {
		runMissionDryRun("test task")
	})

	riskLevels := []string{"LOW", "MEDIUM", "HIGH", "CRITICAL"}
	for _, level := range riskLevels {
		if !strings.Contains(output, level) {
			t.Errorf("output missing risk level: %s", level)
		}
	}
}

func TestRunMissionDryRun_ShowsMaxRework(t *testing.T) {
	output := captureStdout(func() {
		runMissionDryRun("test task")
	})

	if !strings.Contains(output, "Max rework loops:") {
		t.Error("output missing max rework loops info")
	}
}

// === runInit tests ===

func TestRunInit_CreatesKiroStructure(t *testing.T) {
	// Create a temp dir and chdir into it
	tmpDir := t.TempDir()
	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)
	os.Chdir(tmpDir)

	output := captureStdout(func() {
		runInit()
	})

	// Verify .kiro directory structure was created
	kiroDir := filepath.Join(tmpDir, ".kiro")
	if _, err := os.Stat(kiroDir); os.IsNotExist(err) {
		t.Fatal(".kiro/ directory was not created")
	}

	// Verify subdirectories exist
	expectedDirs := []string{
		filepath.Join(kiroDir, "specs", "_templates"),
		filepath.Join(kiroDir, "steering"),
		filepath.Join(kiroDir, "agents"),
	}
	for _, dir := range expectedDirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			t.Errorf("expected directory not created: %s", dir)
		}
	}

	// Verify template files were created
	expectedFiles := []string{
		filepath.Join(kiroDir, "specs", "_templates", "requirements.md"),
		filepath.Join(kiroDir, "specs", "_templates", "design.md"),
		filepath.Join(kiroDir, "specs", "_templates", "tasks.md"),
		filepath.Join(kiroDir, "steering", "dev-rules.md"),
	}
	for _, f := range expectedFiles {
		if _, err := os.Stat(f); os.IsNotExist(err) {
			t.Errorf("expected file not created: %s", f)
		}
	}

	// Output should indicate success
	if !strings.Contains(output, "agent-ready") || !strings.Contains(output, "Moonbase Init") {
		t.Errorf("output missing expected completion messages\nGot:\n%s", output)
	}
}

func TestRunInit_AlreadyInitialized(t *testing.T) {
	tmpDir := t.TempDir()
	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)
	os.Chdir(tmpDir)

	// Pre-create .kiro/
	os.MkdirAll(filepath.Join(tmpDir, ".kiro"), 0o755)

	output := captureStdout(func() {
		runInit()
	})

	// Should warn about existing .kiro
	if !strings.Contains(output, "already exists") {
		t.Errorf("expected 'already exists' warning, got:\n%s", output)
	}
}

func TestRunInit_SteeringContainsStack(t *testing.T) {
	tmpDir := t.TempDir()
	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)
	os.Chdir(tmpDir)

	captureStdout(func() {
		runInit()
	})

	// Read steering file and verify it has stack section
	data, err := os.ReadFile(filepath.Join(tmpDir, ".kiro", "steering", "dev-rules.md"))
	if err != nil {
		t.Fatalf("failed to read steering file: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "## Stack") {
		t.Error("steering file missing ## Stack section")
	}
	if !strings.Contains(content, "## Conventions") {
		t.Error("steering file missing ## Conventions section")
	}
}

// === runList tests ===

func TestRunList_PrintsRoster(t *testing.T) {
	output := captureStdout(func() {
		runList()
	})

	// Should contain the header
	if !strings.Contains(output, "OPERATIVE ROSTER") {
		t.Errorf("output missing roster header\nGot:\n%s", output)
	}

	// Should contain some agent sections
	if !strings.Contains(output, "AI BACKENDS") {
		t.Errorf("output missing AI BACKENDS section\nGot:\n%s", output)
	}
}

func TestRunList_ShowsSectorV(t *testing.T) {
	output := captureStdout(func() {
		runList()
	})

	// Should list either the registry agents or the hardcoded fallback
	if !strings.Contains(output, "SECTOR V") && !strings.Contains(output, "showing defaults") {
		t.Errorf("output missing SECTOR V or fallback notice\nGot:\n%s", output)
	}
}

func TestRunList_ShowsSpecialists(t *testing.T) {
	output := captureStdout(func() {
		runList()
	})

	// Should have a SPECIALISTS section
	if !strings.Contains(output, "SPECIALISTS") {
		t.Errorf("output missing SPECIALISTS section\nGot:\n%s", output)
	}
}

// === runConfig tests ===

func TestRunConfig_PrintsConfiguration(t *testing.T) {
	output := captureStdout(func() {
		runConfig()
	})

	// Should contain the config header
	if !strings.Contains(output, "Moonbase Configuration") {
		t.Errorf("output missing config header\nGot:\n%s", output)
	}

	// Should show the config path
	if !strings.Contains(output, "Path:") {
		t.Errorf("output missing config path\nGot:\n%s", output)
	}
}

// === extractNumbuh tests ===

func TestExtractNumbuh(t *testing.T) {
	tests := []struct {
		name     string
		expected string
	}{
		{"numbuh-1", "1"},
		{"numbuh-4", "4"},
		{"numbuh-13", "13"},
		{"numbuh-274", "274"},
		{"numbuh-362", "362"},
		{"numbuh-999", "999"},
		{"numbuh-0", "0"},
		{"sector-z", "Z"},
		{"knd-council", "K"},
		{"other-name", "other-name"},
		{"", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractNumbuh(tt.name)
			if got != tt.expected {
				t.Errorf("extractNumbuh(%q) = %q, want %q", tt.name, got, tt.expected)
			}
		})
	}
}

// === generateSteering tests ===

func TestGenerateSteering_WithStack(t *testing.T) {
	result := generateSteering("go", "go build", "go test ./...")

	if !strings.Contains(result, "go") {
		t.Error("expected go language in output")
	}
	if !strings.Contains(result, "go build") {
		t.Error("expected build tool in output")
	}
	if !strings.Contains(result, "go test ./...") {
		t.Error("expected test command in output")
	}
}

func TestGenerateSteering_UnknownStack(t *testing.T) {
	result := generateSteering("unknown", "", "")

	if !strings.Contains(result, "detected: unknown") {
		t.Error("expected unknown detected notice")
	}
}

func TestGenerateSteering_HasConventions(t *testing.T) {
	result := generateSteering("python", "pip", "pytest")

	if !strings.Contains(result, "## Conventions") {
		t.Error("expected Conventions section")
	}
	if !strings.Contains(result, "## Stack") {
		t.Error("expected Stack section")
	}
}

// === agentsDir resolution test ===

func TestAgentsDir_FindsAgentsFromProjectRoot(t *testing.T) {
	// This test verifies agent resolution from the project root.
	// agentsDir() calls os.Exit(1) on failure, so we test using
	// the underlying agents.FindAgentsDir logic with known paths.
	cwd, _ := os.Getwd()

	// Test running from project root (../../ from cmd/moonbase)
	projectRoot := filepath.Join(cwd, "..", "..")
	agentsDirPath := filepath.Join(projectRoot, "agents")

	if !isAgentsDir(agentsDirPath) {
		t.Skip("not running from moonbase project tree")
	}

	// Verify the agents directory has .md files
	entries, err := os.ReadDir(agentsDirPath)
	if err != nil {
		t.Fatalf("failed to read agents dir: %v", err)
	}

	mdCount := 0
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".md" {
			mdCount++
		}
	}

	if mdCount == 0 {
		t.Error("agents dir has no .md files")
	}
	if mdCount < 10 {
		t.Errorf("expected at least 10 agent files, got %d", mdCount)
	}
}

func TestAgentsDir_IsAgentsDirWithValidFiles(t *testing.T) {
	// Create a temp dir with .md files — should be valid
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "agent-1.md"), []byte("test"), 0o644)

	if !isAgentsDir(tmpDir) {
		t.Error("expected dir with .md files to be valid agents dir")
	}
}

func TestAgentsDir_IsAgentsDirEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	if isAgentsDir(tmpDir) {
		t.Error("expected empty dir to NOT be valid agents dir")
	}
}

func TestAgentsDir_IsAgentsDirWithNonMdFiles(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "readme.txt"), []byte("text"), 0o644)
	os.WriteFile(filepath.Join(tmpDir, "config.yaml"), []byte("yaml"), 0o644)

	if isAgentsDir(tmpDir) {
		t.Error("expected dir with only non-.md files to NOT be valid agents dir")
	}
}

// === snippet name validation tests ===
// The snippet name validation is inline in the command handler,
// so we test the validation logic directly here.

func TestSnippetNameValidation(t *testing.T) {
	tests := []struct {
		name  string
		valid bool
		desc  string
	}{
		{"my-prompt", true, "simple hyphenated name"},
		{"test", true, "simple name"},
		{"my prompt", true, "name with space (allowed)"},
		{"a", true, "single char"},
		{strings.Repeat("x", 100), true, "exactly 100 chars (max)"},
		{strings.Repeat("x", 101), false, "101 chars (over max)"},
		{"has/slash", false, "forward slash"},
		{"has\\backslash", false, "backslash"},
		{"has\x00null", false, "control char (null)"},
		{"has\x01soh", false, "control char (SOH)"},
		{"has\x1fus", false, "control char (US)"},
		{"valid-name_with.dots", true, "dots and underscores are fine"},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			valid := isValidSnippetName(tt.name)
			if valid != tt.valid {
				t.Errorf("isValidSnippetName(%q) = %v, want %v", tt.name, valid, tt.valid)
			}
		})
	}
}

// isValidSnippetName replicates the inline validation from snippet_cmd.go.
// This is extracted here for testability — mirrors the exact logic.
func isValidSnippetName(name string) bool {
	if len(name) > 100 {
		return false
	}
	if strings.ContainsAny(name, "/\\") {
		return false
	}
	for _, r := range name {
		if r < 32 {
			return false
		}
	}
	return true
}

// === sourceTag tests ===

func TestSourceTag(t *testing.T) {
	tests := []struct {
		source   string
		expected string
	}{
		{"user", "[user]"},
		{"project", "[project]"},
		{"built-in", "[built-in]"},
		{"", ""},
		{"unknown", ""},
	}
	for _, tt := range tests {
		t.Run(tt.source, func(t *testing.T) {
			got := sourceTag(tt.source)
			if got != tt.expected {
				t.Errorf("sourceTag(%q) = %q, want %q", tt.source, got, tt.expected)
			}
		})
	}
}

// === writeTemplate tests ===

func TestWriteTemplate(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.md")
	content := "# Test\n\nHello world"

	err := writeTemplate(path, content)
	if err != nil {
		t.Fatalf("writeTemplate failed: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading template: %v", err)
	}
	if string(data) != content {
		t.Errorf("content mismatch:\nexpected: %s\ngot: %s", content, string(data))
	}
}

func TestWriteTemplate_CreatesFile(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "subdir", "test.md")

	// This should fail because parent dir doesn't exist
	err := writeTemplate(path, "content")
	if err == nil {
		t.Error("expected error when parent dir doesn't exist")
	}
}
