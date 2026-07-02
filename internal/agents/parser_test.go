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
