package agents

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSplitFrontmatter_Valid(t *testing.T) {
	content := []byte("---\nname: numbuh-1\nrole: Analyst\n---\n# Numbuh 1\n\nBody content here.\n")

	yamlBytes, body, err := SplitFrontmatter(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(string(yamlBytes), "name: numbuh-1") {
		t.Errorf("expected yaml to contain 'name: numbuh-1', got: %s", string(yamlBytes))
	}
	if !strings.Contains(string(yamlBytes), "role: Analyst") {
		t.Errorf("expected yaml to contain 'role: Analyst', got: %s", string(yamlBytes))
	}
	if !strings.Contains(string(body), "# Numbuh 1") {
		t.Errorf("expected body to contain '# Numbuh 1', got: %s", string(body))
	}
	if !strings.Contains(string(body), "Body content here.") {
		t.Errorf("expected body to contain 'Body content here.', got: %s", string(body))
	}
}

func TestSplitFrontmatter_NoFrontmatter(t *testing.T) {
	content := []byte("# Just a markdown file\n\nNo frontmatter here.\n")

	_, _, err := SplitFrontmatter(content)
	if err != ErrNoFrontmatter {
		t.Fatalf("expected ErrNoFrontmatter, got: %v", err)
	}
}

func TestSplitFrontmatter_MalformedNoClosing(t *testing.T) {
	content := []byte("---\nname: broken\nno closing delimiter\n")

	_, _, err := SplitFrontmatter(content)
	if err != ErrMalformedFrontmatter {
		t.Fatalf("expected ErrMalformedFrontmatter, got: %v", err)
	}
}

func TestSplitFrontmatter_EmptyBody(t *testing.T) {
	content := []byte("---\nname: empty\n---\n")

	yamlBytes, body, err := SplitFrontmatter(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(string(yamlBytes), "name: empty") {
		t.Errorf("expected yaml to contain 'name: empty', got: %s", string(yamlBytes))
	}
	if len(strings.TrimSpace(string(body))) != 0 {
		t.Errorf("expected empty body, got: %q", string(body))
	}
}

func TestSplitFrontmatter_LeadingNewlines(t *testing.T) {
	content := []byte("\n\n---\nname: padded\n---\n# Content\n")

	yamlBytes, body, err := SplitFrontmatter(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(string(yamlBytes), "name: padded") {
		t.Errorf("expected yaml to contain 'name: padded', got: %s", string(yamlBytes))
	}
	if !strings.Contains(string(body), "# Content") {
		t.Errorf("expected body to contain '# Content', got: %s", string(body))
	}
}

func TestSplitFrontmatter_MultilineValues(t *testing.T) {
	content := []byte(`---
name: numbuh-4
description: >
  Hits implementation with reality.
  Tests and verifies.
tools:
  - read
  - shell
  - grep
---
# Numbuh 4 — QA

Body here.
`)

	yamlBytes, body, err := SplitFrontmatter(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(string(yamlBytes), "name: numbuh-4") {
		t.Errorf("expected yaml to contain 'name: numbuh-4', got: %s", string(yamlBytes))
	}
	if !strings.Contains(string(yamlBytes), "- read") {
		t.Errorf("expected yaml to contain '- read', got: %s", string(yamlBytes))
	}
	if !strings.Contains(string(body), "# Numbuh 4") {
		t.Errorf("expected body to contain '# Numbuh 4', got: %s", string(body))
	}
}

func TestParseAgentFile_RealAgent(t *testing.T) {
	// Find the agents directory relative to the test
	agentsDir := findAgentsDir(t)
	if agentsDir == "" {
		t.Skip("agents directory not found")
	}

	path := filepath.Join(agentsDir, "numbuh-1.md")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Skipf("agent file not found: %s", path)
	}

	agent, err := ParseAgentFile(path)
	if err != nil {
		t.Fatalf("failed to parse real agent file: %v", err)
	}

	if agent.Name != "numbuh-1" {
		t.Errorf("expected name 'numbuh-1', got: %s", agent.Name)
	}
	if agent.Designation != "Nigel Uno" {
		t.Errorf("expected designation 'Nigel Uno', got: %s", agent.Designation)
	}
	if agent.Role == "" {
		t.Error("expected non-empty role")
	}
	if len(agent.Tools) == 0 {
		t.Error("expected non-empty tools list")
	}
	if agent.PipelinePosition == nil || *agent.PipelinePosition != 1 {
		t.Errorf("expected pipeline_position 1, got: %v", agent.PipelinePosition)
	}
	if agent.Prompt == "" {
		t.Error("expected non-empty prompt (body)")
	}
	if !strings.Contains(agent.Prompt, "# Numbuh 1") {
		t.Error("expected prompt to contain agent header")
	}
	if agent.FilePath != path {
		t.Errorf("expected FilePath %s, got: %s", path, agent.FilePath)
	}
}

func TestParseAgentFile_AllAgents(t *testing.T) {
	agentsDir := findAgentsDir(t)
	if agentsDir == "" {
		t.Skip("agents directory not found")
	}

	files, err := filepath.Glob(filepath.Join(agentsDir, "*.md"))
	if err != nil {
		t.Fatalf("glob failed: %v", err)
	}

	if len(files) != 14 {
		t.Fatalf("expected 14 agent files, found %d", len(files))
	}

	for _, file := range files {
		t.Run(filepath.Base(file), func(t *testing.T) {
			agent, err := ParseAgentFile(file)
			if err != nil {
				t.Fatalf("failed to parse: %v", err)
			}
			if agent.Name == "" {
				t.Error("agent name is empty")
			}
			if agent.Role == "" {
				t.Error("agent role is empty")
			}
			if agent.Prompt == "" {
				t.Error("agent prompt is empty")
			}
			if len(agent.Tools) == 0 {
				t.Error("agent has no tools")
			}
		})
	}
}

// findAgentsDir walks up from the test binary to find the agents/ directory
func findAgentsDir(t *testing.T) string {
	t.Helper()

	// Try relative to working directory
	candidates := []string{
		"../../agents",
		"../../../agents",
		"agents",
	}

	for _, c := range candidates {
		if info, err := os.Stat(c); err == nil && info.IsDir() {
			abs, _ := filepath.Abs(c)
			return abs
		}
	}

	// Try from module root
	wd, _ := os.Getwd()
	for dir := wd; dir != "/"; dir = filepath.Dir(dir) {
		candidate := filepath.Join(dir, "agents")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
	}

	return ""
}

// === Gap Coverage: Malformed YAML fields, empty file, invalid types ===

func TestSplitFrontmatter_EmptyFile(t *testing.T) {
	content := []byte("")

	_, _, err := SplitFrontmatter(content)
	if err != ErrNoFrontmatter {
		t.Fatalf("expected ErrNoFrontmatter for empty file, got: %v", err)
	}
}

func TestSplitFrontmatter_OnlyDelimiters(t *testing.T) {
	// With empty YAML content between delimiters (just a blank line)
	content := []byte("---\n\n---\n")

	yamlBytes, body, err := SplitFrontmatter(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// YAML should be empty/whitespace
	if len(strings.TrimSpace(string(yamlBytes))) != 0 {
		t.Errorf("expected empty yaml, got: %q", string(yamlBytes))
	}
	if len(strings.TrimSpace(string(body))) != 0 {
		t.Errorf("expected empty body, got: %q", string(body))
	}
}

func TestSplitFrontmatter_WhitespaceOnly(t *testing.T) {
	content := []byte("   \n  \n  ")

	_, _, err := SplitFrontmatter(content)
	if err != ErrNoFrontmatter {
		t.Fatalf("expected ErrNoFrontmatter for whitespace-only, got: %v", err)
	}
}

func TestParseAgentFile_MissingNameField(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "no-name.md")

	// Valid YAML frontmatter but missing the 'name' field
	content := "---\nrole: QA\ntools:\n  - read\n  - shell\n---\n# Agent with no name\n\nBody here.\n"
	os.WriteFile(path, []byte(content), 0o644)

	agent, err := ParseAgentFile(path)
	if err != nil {
		t.Fatalf("parse should succeed (YAML is valid), got: %v", err)
	}
	// Name should be empty string — it's valid YAML, just missing the field
	if agent.Name != "" {
		t.Errorf("expected empty name, got: %s", agent.Name)
	}
	if agent.Role != "QA" {
		t.Errorf("expected role 'QA', got: %s", agent.Role)
	}
	if len(agent.Tools) != 2 {
		t.Errorf("expected 2 tools, got: %d", len(agent.Tools))
	}
}

func TestParseAgentFile_InvalidToolsType(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "bad-tools.md")

	// Tools as a string instead of a list — this should cause a YAML unmarshal error
	content := "---\nname: bad-agent\ntools: \"this should be a list\"\n---\n# Bad Agent\n"
	os.WriteFile(path, []byte(content), 0o644)

	_, err := ParseAgentFile(path)
	if err == nil {
		t.Fatal("expected error when tools is a string instead of list")
	}
	if !strings.Contains(err.Error(), "YAML") {
		t.Errorf("expected YAML parse error, got: %v", err)
	}
}

func TestParseAgentFile_InvalidToolsTypeNumber(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "number-tools.md")

	// Tools as a number — YAML unmarshal should fail
	content := "---\nname: bad-agent\ntools: 42\n---\n# Bad Agent\n"
	os.WriteFile(path, []byte(content), 0o644)

	_, err := ParseAgentFile(path)
	if err == nil {
		t.Fatal("expected error when tools is a number")
	}
}

func TestParseAgentFile_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "empty.md")

	os.WriteFile(path, []byte(""), 0o644)

	_, err := ParseAgentFile(path)
	if err == nil {
		t.Fatal("expected error for empty file")
	}
}

func TestParseAgentFile_NonexistentFile(t *testing.T) {
	_, err := ParseAgentFile("/nonexistent/agent.md")
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestParseAgentFile_MissingDesignation(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "no-designation.md")

	content := "---\nname: numbuh-test\nrole: Tester\ntools:\n  - read\n---\n# Test Agent\n\nPrompt.\n"
	os.WriteFile(path, []byte(content), 0o644)

	agent, err := ParseAgentFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if agent.Designation != "" {
		t.Errorf("expected empty designation, got: %s", agent.Designation)
	}
}

func TestParseAgentFile_AllOptionalFieldsMissing(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "minimal.md")

	// Absolute minimum valid agent
	content := "---\nname: minimal\n---\n# Minimal\n"
	os.WriteFile(path, []byte(content), 0o644)

	agent, err := ParseAgentFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if agent.Name != "minimal" {
		t.Errorf("expected name 'minimal', got: %s", agent.Name)
	}
	if agent.Shell != nil {
		t.Error("expected nil shell config")
	}
	if agent.PipelinePosition != nil {
		t.Error("expected nil pipeline_position")
	}
	if agent.Routing != nil {
		t.Error("expected nil routing")
	}
	if agent.Triggers != nil {
		t.Error("expected nil triggers")
	}
}

func TestParseAgentFile_ExtraUnknownFields(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "extra-fields.md")

	// Unknown YAML fields should not cause errors (YAML ignores them)
	content := "---\nname: extra\nrole: Test\nfoo: bar\nunknown_field: 123\ntools:\n  - read\n---\n# Extra\n"
	os.WriteFile(path, []byte(content), 0o644)

	agent, err := ParseAgentFile(path)
	if err != nil {
		t.Fatalf("unexpected error for unknown fields: %v", err)
	}
	if agent.Name != "extra" {
		t.Errorf("expected name 'extra', got: %s", agent.Name)
	}
}
