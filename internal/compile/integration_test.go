package compile

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/rk-senne/moonbase/internal/agents"
)

// TestCompile_AllAgents is an integration test that compiles all 14 embedded agents
// to a temp directory and verifies each produces valid JSON + prompt files.
func TestCompile_AllAgents(t *testing.T) {
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

	outDir := t.TempDir()

	for _, f := range files {
		agent, err := agents.ParseAgentFile(f)
		if err != nil {
			t.Fatalf("parse %s failed: %v", filepath.Base(f), err)
		}

		ka, promptBody, err := Compile(*agent)
		if err != nil {
			t.Fatalf("compile %s failed: %v", agent.Name, err)
		}

		if err := WriteAgent(ka, promptBody, outDir); err != nil {
			t.Fatalf("write %s failed: %v", agent.Name, err)
		}
	}

	// Verify all 14 .json + 14 .prompt.md files exist
	jsonFiles, _ := filepath.Glob(filepath.Join(outDir, "*.json"))
	promptFiles, _ := filepath.Glob(filepath.Join(outDir, "*.prompt.md"))

	if len(jsonFiles) != 14 {
		t.Errorf("expected 14 .json files, got %d", len(jsonFiles))
	}
	if len(promptFiles) != 14 {
		t.Errorf("expected 14 .prompt.md files, got %d", len(promptFiles))
	}

	// Each JSON should unmarshal without error
	for _, jf := range jsonFiles {
		data, err := os.ReadFile(jf)
		if err != nil {
			t.Errorf("read %s: %v", filepath.Base(jf), err)
			continue
		}

		var ka KiroAgent
		if err := json.Unmarshal(data, &ka); err != nil {
			t.Errorf("unmarshal %s: %v", filepath.Base(jf), err)
			continue
		}

		// Basic sanity checks
		if ka.Name == "" {
			t.Errorf("%s: empty name", filepath.Base(jf))
		}
		if ka.Prompt == "" {
			t.Errorf("%s: empty prompt", filepath.Base(jf))
		}
	}
}

// findAgentsDir walks up from the test binary to find the agents/ directory.
func findAgentsDir(t *testing.T) string {
	t.Helper()

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

	wd, _ := os.Getwd()
	for dir := wd; dir != "/"; dir = filepath.Dir(dir) {
		candidate := filepath.Join(dir, "agents")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
	}

	return ""
}
