package compile

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rk-senne/moonbase/internal/agents"
)

func TestCompile_FullAgent(t *testing.T) {
	// Simulates numbuh-3: full agent with shell, write, hooks, tools, auto_tools
	agent := agents.Agent{
		Name:        "numbuh-3",
		Description: "Writes clean, readable, testable code.",
		Prompt:      "# Numbuh 3\n\nBody content.",
		Tools:       []string{"read", "write", "shell", "grep", "glob", "code", "knowledge"},
		AutoTools:   []string{"read", "write", "grep", "glob", "code"},
		Shell: &agents.ShellConfig{
			AllowedCommands: []string{"go test ./...", "npm test", "make build"},
			ReadOnly:        false,
		},
		Write: &agents.WriteConfig{
			Auto:   []string{"src/**", "internal/**", "docs/**"},
			Denied: []string{".env", "secrets/**"},
		},
		Hooks: &agents.HooksConfig{
			OnActivate: []agents.Hook{
				{Command: "git branch --show-current", TimeoutMs: 5000},
			},
		},
	}

	ka, prompt, err := Compile(agent)
	if err != nil {
		t.Fatalf("Compile failed: %v", err)
	}

	if ka.Name != "numbuh-3" {
		t.Errorf("expected name 'numbuh-3', got %q", ka.Name)
	}
	if ka.Description != "Writes clean, readable, testable code." {
		t.Errorf("unexpected description: %q", ka.Description)
	}
	if ka.Prompt != "file://numbuh-3.prompt.md" {
		t.Errorf("expected prompt ref, got %q", ka.Prompt)
	}
	if prompt != "# Numbuh 3\n\nBody content." {
		t.Errorf("unexpected prompt body: %q", prompt)
	}

	// Tools
	if len(ka.Tools) != 7 {
		t.Errorf("expected 7 tools, got %d", len(ka.Tools))
	}

	// AllowedTools from auto_tools
	if len(ka.AllowedTools) != 5 {
		t.Errorf("expected 5 allowedTools, got %d", len(ka.AllowedTools))
	}

	// ToolsSettings
	if ka.ToolsSettings == nil {
		t.Fatal("expected non-nil toolsSettings")
	}
	if ka.ToolsSettings.Shell == nil {
		t.Fatal("expected non-nil shell settings")
	}
	if len(ka.ToolsSettings.Shell.AllowedCommands) != 3 {
		t.Errorf("expected 3 allowed commands, got %d", len(ka.ToolsSettings.Shell.AllowedCommands))
	}
	if ka.ToolsSettings.Shell.DenyByDefault {
		t.Error("expected denyByDefault false for non-read-only agent")
	}

	// Write settings
	if ka.ToolsSettings.Write == nil {
		t.Fatal("expected non-nil write settings")
	}
	if len(ka.ToolsSettings.Write.AllowedPaths) != 3 {
		t.Errorf("expected 3 allowed paths, got %d", len(ka.ToolsSettings.Write.AllowedPaths))
	}
	if len(ka.ToolsSettings.Write.DeniedPaths) != 2 {
		t.Errorf("expected 2 denied paths, got %d", len(ka.ToolsSettings.Write.DeniedPaths))
	}

	// Hooks
	if ka.Hooks == nil {
		t.Fatal("expected non-nil hooks")
	}
	spawn, ok := ka.Hooks["agentSpawn"]
	if !ok {
		t.Fatal("expected agentSpawn hook")
	}
	if len(spawn) != 1 {
		t.Fatalf("expected 1 agentSpawn hook, got %d", len(spawn))
	}
	if spawn[0].Command != "git branch --show-current" {
		t.Errorf("unexpected hook command: %q", spawn[0].Command)
	}
	if spawn[0].TimeoutMs != 5000 {
		t.Errorf("expected timeout_ms 5000, got %d", spawn[0].TimeoutMs)
	}

	// No MCP servers
	if ka.MCPServers != nil {
		t.Error("expected nil mcpServers for agent without MCP config")
	}
}

func TestCompile_ReadOnlyAgent(t *testing.T) {
	// Simulates numbuh-4: read-only, no write
	agent := agents.Agent{
		Name:        "numbuh-4",
		Description: "QA agent.",
		Prompt:      "# Numbuh 4\n\nQA body.",
		Tools:       []string{"read", "shell", "grep", "glob", "code", "knowledge", "subagent"},
		AutoTools:   []string{"read", "shell", "grep", "glob", "code", "knowledge"},
		Shell: &agents.ShellConfig{
			AllowedCommands: []string{"go test ./...", "npm test", "git diff"},
			ReadOnly:        true,
		},
		Hooks: &agents.HooksConfig{
			OnActivate: []agents.Hook{
				{Command: "git diff --stat", TimeoutMs: 5000},
			},
		},
	}

	ka, _, err := Compile(agent)
	if err != nil {
		t.Fatalf("Compile failed: %v", err)
	}

	// Verify read-only mapping: denyByDefault + autoAllowReadonly
	if ka.ToolsSettings == nil || ka.ToolsSettings.Shell == nil {
		t.Fatal("expected shell settings for read-only agent")
	}
	if !ka.ToolsSettings.Shell.DenyByDefault {
		t.Error("expected denyByDefault=true for read-only agent")
	}
	if !ka.ToolsSettings.Shell.AutoAllowReadonly {
		t.Error("expected autoAllowReadonly=true for read-only agent")
	}

	// Verify NO write settings (read-only agent has no Write config)
	if ka.ToolsSettings.Write != nil {
		t.Error("expected nil write settings for read-only agent")
	}

	// Verify JSON output does NOT contain toolset or readOnly
	data, err := json.Marshal(ka)
	if err != nil {
		t.Fatalf("JSON marshal failed: %v", err)
	}
	jsonStr := string(data)
	if contains(jsonStr, "toolset") {
		t.Error("JSON must NOT contain 'toolset' field")
	}
	if contains(jsonStr, "readOnly") {
		t.Error("JSON must NOT contain 'readOnly' field")
	}
}

func TestCompile_MCPServers(t *testing.T) {
	agent := agents.Agent{
		Name:      "mcp-agent",
		Prompt:    "# MCP Agent\n\nBody.",
		Tools:     []string{"read", "write", "shell"},
		AutoTools: []string{"read"},
		MCPServers: []agents.MCPServerConfig{
			{
				Name:         "github",
				Command:      "npx",
				Args:         []string{"-y", "@modelcontextprotocol/server-github"},
				Env:          map[string]string{"GITHUB_TOKEN": "${GITHUB_TOKEN}"},
				AllowedTools: []string{"create_pull_request", "list_issues"},
			},
			{
				Name:    "postgres",
				Command: "npx",
				Args:    []string{"-y", "@modelcontextprotocol/server-postgres"},
			},
		},
	}

	ka, _, err := Compile(agent)
	if err != nil {
		t.Fatalf("Compile failed: %v", err)
	}

	// Verify mcpServers map
	if len(ka.MCPServers) != 2 {
		t.Fatalf("expected 2 mcpServers, got %d", len(ka.MCPServers))
	}

	gh, ok := ka.MCPServers["github"]
	if !ok {
		t.Fatal("expected 'github' in mcpServers map")
	}
	if gh.Command != "npx" {
		t.Errorf("expected command 'npx', got %q", gh.Command)
	}
	if len(gh.Args) != 2 {
		t.Errorf("expected 2 args, got %d", len(gh.Args))
	}
	if gh.Env["GITHUB_TOKEN"] != "${GITHUB_TOKEN}" {
		t.Errorf("expected GITHUB_TOKEN env, got %v", gh.Env)
	}

	pg, ok := ka.MCPServers["postgres"]
	if !ok {
		t.Fatal("expected 'postgres' in mcpServers map")
	}
	if pg.Command != "npx" {
		t.Errorf("expected command 'npx', got %q", pg.Command)
	}

	// Verify allowedTools includes @name/tool patterns
	expectedAllowed := []string{"read", "@github/create_pull_request", "@github/list_issues"}
	if len(ka.AllowedTools) != len(expectedAllowed) {
		t.Fatalf("expected %d allowedTools, got %d: %v", len(expectedAllowed), len(ka.AllowedTools), ka.AllowedTools)
	}
	for i, exp := range expectedAllowed {
		if ka.AllowedTools[i] != exp {
			t.Errorf("allowedTools[%d]: expected %q, got %q", i, exp, ka.AllowedTools[i])
		}
	}
}

func TestCompile_MinimalAgent(t *testing.T) {
	agent := agents.Agent{
		Name:   "minimal",
		Prompt: "# Minimal\n",
	}

	ka, prompt, err := Compile(agent)
	if err != nil {
		t.Fatalf("Compile failed: %v", err)
	}

	if ka.Name != "minimal" {
		t.Errorf("expected name 'minimal', got %q", ka.Name)
	}
	if ka.Prompt != "file://minimal.prompt.md" {
		t.Errorf("unexpected prompt: %q", ka.Prompt)
	}
	if prompt != "# Minimal\n" {
		t.Errorf("unexpected prompt body: %q", prompt)
	}

	// All optional fields should be nil/empty
	if ka.AllowedTools != nil {
		t.Error("expected nil allowedTools for minimal agent")
	}
	if ka.ToolsSettings != nil {
		t.Error("expected nil toolsSettings for minimal agent")
	}
	if ka.Hooks != nil {
		t.Error("expected nil hooks for minimal agent")
	}
	if ka.MCPServers != nil {
		t.Error("expected nil mcpServers for minimal agent")
	}
}

func TestCompile_EmptyName(t *testing.T) {
	agent := agents.Agent{
		Prompt: "# No Name\n",
	}

	_, _, err := Compile(agent)
	if err == nil {
		t.Fatal("expected error for empty name")
	}
}

func TestCompile_AllHookTypes(t *testing.T) {
	agent := agents.Agent{
		Name:   "hook-test",
		Prompt: "# Hook Test\n",
		Hooks: &agents.HooksConfig{
			OnActivate:  []agents.Hook{{Command: "echo spawn", TimeoutMs: 1000}},
			PreToolUse:  []agents.Hook{{Command: "echo pre", TimeoutMs: 2000}},
			PostToolUse: []agents.Hook{{Command: "echo post", TimeoutMs: 3000}},
			OnComplete:  []agents.Hook{{Command: "echo stop", TimeoutMs: 4000}},
		},
	}

	ka, _, err := Compile(agent)
	if err != nil {
		t.Fatalf("Compile failed: %v", err)
	}

	if len(ka.Hooks) != 4 {
		t.Fatalf("expected 4 hook types, got %d", len(ka.Hooks))
	}

	tests := []struct {
		key     string
		cmd     string
		timeout int
	}{
		{"agentSpawn", "echo spawn", 1000},
		{"preToolUse", "echo pre", 2000},
		{"postToolUse", "echo post", 3000},
		{"stop", "echo stop", 4000},
	}

	for _, tt := range tests {
		hooks, ok := ka.Hooks[tt.key]
		if !ok {
			t.Errorf("missing hook key %q", tt.key)
			continue
		}
		if len(hooks) != 1 {
			t.Errorf("expected 1 hook for %q, got %d", tt.key, len(hooks))
			continue
		}
		if hooks[0].Command != tt.cmd {
			t.Errorf("hook %q: expected command %q, got %q", tt.key, tt.cmd, hooks[0].Command)
		}
		if hooks[0].TimeoutMs != tt.timeout {
			t.Errorf("hook %q: expected timeout %d, got %d", tt.key, tt.timeout, hooks[0].TimeoutMs)
		}
	}
}

func TestWriteAgent_RoundTrip(t *testing.T) {
	ka := &KiroAgent{
		Name:         "test-agent",
		Description:  "Test description",
		Prompt:       "file://test-agent.prompt.md",
		Tools:        []string{"read", "write"},
		AllowedTools: []string{"read"},
		ToolsSettings: &KiroToolsSettings{
			Shell: &KiroShellSettings{
				AllowedCommands: []string{"go test ./..."},
			},
		},
		Hooks: map[string][]KiroHook{
			"agentSpawn": {{Command: "echo hi", TimeoutMs: 5000}},
		},
	}
	promptBody := "# Test Agent\n\nThis is the prompt body.\n"

	dir := t.TempDir()
	err := WriteAgent(ka, promptBody, dir)
	if err != nil {
		t.Fatalf("WriteAgent failed: %v", err)
	}

	// Verify JSON file exists and unmarshals correctly
	jsonPath := filepath.Join(dir, "test-agent.json")
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatalf("reading JSON file: %v", err)
	}

	var loaded KiroAgent
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("unmarshaling JSON: %v", err)
	}

	if loaded.Name != "test-agent" {
		t.Errorf("expected name 'test-agent', got %q", loaded.Name)
	}
	if loaded.Description != "Test description" {
		t.Errorf("unexpected description: %q", loaded.Description)
	}
	if len(loaded.Tools) != 2 {
		t.Errorf("expected 2 tools, got %d", len(loaded.Tools))
	}
	if len(loaded.AllowedTools) != 1 {
		t.Errorf("expected 1 allowedTool, got %d", len(loaded.AllowedTools))
	}

	// Verify prompt file
	promptPath := filepath.Join(dir, "test-agent.prompt.md")
	promptData, err := os.ReadFile(promptPath)
	if err != nil {
		t.Fatalf("reading prompt file: %v", err)
	}
	if string(promptData) != promptBody {
		t.Errorf("prompt file content mismatch: got %q", string(promptData))
	}
}

func TestWriteAgent_PathTraversal(t *testing.T) {
	ka := &KiroAgent{
		Name:   "../evil",
		Prompt: "file://../evil.prompt.md",
	}

	err := WriteAgent(ka, "evil", t.TempDir())
	if err == nil {
		t.Fatal("expected error for path traversal in name")
	}
}

func TestWriteAgent_EmptyName(t *testing.T) {
	ka := &KiroAgent{
		Prompt: "file://empty.prompt.md",
	}

	err := WriteAgent(ka, "body", t.TempDir())
	if err == nil {
		t.Fatal("expected error for empty name")
	}
}

func TestIsStale_NewerSource(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "agent.md")
	dst := filepath.Join(dir, "agent.json")

	// Create compiled first (older)
	os.WriteFile(dst, []byte("{}"), 0o644)
	time.Sleep(50 * time.Millisecond)
	// Create source after (newer)
	os.WriteFile(src, []byte("---\nname: test\n---\n"), 0o644)

	stale, err := IsStale(src, dst)
	if err != nil {
		t.Fatalf("IsStale error: %v", err)
	}
	if !stale {
		t.Error("expected stale when source is newer")
	}
}

func TestIsStale_OlderSource(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "agent.md")
	dst := filepath.Join(dir, "agent.json")

	// Create source first (older)
	os.WriteFile(src, []byte("---\nname: test\n---\n"), 0o644)
	time.Sleep(50 * time.Millisecond)
	// Create compiled after (newer)
	os.WriteFile(dst, []byte("{}"), 0o644)

	stale, err := IsStale(src, dst)
	if err != nil {
		t.Fatalf("IsStale error: %v", err)
	}
	if stale {
		t.Error("expected not stale when compiled is newer")
	}
}

func TestIsStale_MissingCompiled(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "agent.md")
	os.WriteFile(src, []byte("---\nname: test\n---\n"), 0o644)

	stale, err := IsStale(src, filepath.Join(dir, "nonexistent.json"))
	if err != nil {
		t.Fatalf("IsStale error: %v", err)
	}
	if !stale {
		t.Error("expected stale when compiled file is missing")
	}
}

func TestIsStale_MissingSource(t *testing.T) {
	dir := t.TempDir()
	dst := filepath.Join(dir, "agent.json")
	os.WriteFile(dst, []byte("{}"), 0o644)

	_, err := IsStale(filepath.Join(dir, "nonexistent.md"), dst)
	if err == nil {
		t.Fatal("expected error when source is missing")
	}
}

// contains checks if a string contains a substring.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStr(s, substr))
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
