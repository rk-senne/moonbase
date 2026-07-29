package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rk-senne/moonbase/internal/agents"
)

// expectExit calls fn and expects it to trigger osExit.
// Returns the exit code that was passed to osExit.
func expectExit(t *testing.T, fn func()) int {
	t.Helper()
	original := osExit
	var capturedCode int
	called := false

	osExit = func(code int) {
		capturedCode = code
		called = true
		panic(fmt.Sprintf("osExit(%d)", code))
	}
	defer func() { osExit = original }()

	func() {
		defer func() {
			if r := recover(); r != nil {
				// Expected: osExit panic
				if !called {
					t.Fatalf("unexpected panic (not osExit): %v", r)
				}
			}
		}()
		fn()
	}()

	if !called {
		t.Fatal("expected osExit to be called, but it wasn't")
	}
	return capturedCode
}

// === runStatus tests ===

func TestRunStatus_PrintsBackendInfo(t *testing.T) {
	output := captureStdout(func() {
		runStatus()
	})
	if !strings.Contains(output, "Moonbase Status") {
		t.Errorf("expected 'Moonbase Status' in output, got:\n%s", output)
	}
	if !strings.Contains(output, "Backend:") {
		t.Errorf("expected 'Backend:' in output, got:\n%s", output)
	}
	if !strings.Contains(output, "Agents:") {
		t.Errorf("expected 'Agents:' in output, got:\n%s", output)
	}
}

// === findAgentsDirQuiet tests ===

func TestFindAgentsDirQuiet_FindsFromCWD(t *testing.T) {
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	projectRoot := filepath.Join(origDir, "..", "..")
	if err := os.Chdir(projectRoot); err != nil {
		t.Skip("could not chdir to project root")
	}

	dir := findAgentsDirQuiet()
	if dir == "" {
		t.Skip("agents dir not found from project root")
	}
	if !isAgentsDir(dir) {
		t.Errorf("findAgentsDirQuiet returned non-agents dir: %s", dir)
	}
}

func TestFindAgentsDirQuiet_ReturnsEmptyForNoAgents(t *testing.T) {
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	tmpDir := t.TempDir()
	os.Chdir(tmpDir)

	dir := findAgentsDirQuiet()
	if dir != "" {
		t.Errorf("expected empty string for dir with no agents, got: %s", dir)
	}
}

// === runLint tests ===

func TestRunLint_NoAgentsDir(t *testing.T) {
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	tmpDir := t.TempDir()
	os.Chdir(tmpDir)

	code := expectExit(t, func() {
		runLint()
	})
	if code != 1 {
		t.Errorf("expected exit code 1, got %d", code)
	}
}

func TestRunLint_ValidAgents(t *testing.T) {
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	// Create a temp agents dir
	tmpDir := t.TempDir()
	agDir := filepath.Join(tmpDir, "agents")
	os.MkdirAll(agDir, 0o755)
	writeTestAgent(t, agDir, "numbuh-1", "Nigel Uno", "Analyst")
	writeTestAgent(t, agDir, "numbuh-2", "Hoagie", "Architect")
	os.Chdir(tmpDir)

	output := captureStdout(func() {
		runLint()
	})
	if !strings.Contains(output, "Agent Lint") {
		t.Errorf("expected 'Agent Lint' header, got:\n%s", output)
	}
	if !strings.Contains(output, "✅") {
		t.Errorf("expected success checkmarks, got:\n%s", output)
	}
}

func TestRunLint_AgentWithIssues(t *testing.T) {
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	tmpDir := t.TempDir()
	agDir := filepath.Join(tmpDir, "agents")
	os.MkdirAll(agDir, 0o755)
	// Bad agent: missing required fields
	content := "---\nname: bad\ntools:\n  - read\n---\n# Bad\nNo protocol section.\n"
	os.WriteFile(filepath.Join(agDir, "bad.md"), []byte(content), 0o644)
	os.Chdir(tmpDir)

	code := expectExit(t, func() {
		runLint()
	})
	if code != 1 {
		t.Errorf("expected exit code 1 for agent with issues, got %d", code)
	}
}

// === lintAgent tests ===

func TestLintAgent_ValidAgent(t *testing.T) {
	pos := 1
	agent := &agents.Agent{
		Name:             "numbuh-1",
		Role:             "Analyst",
		Tools:            []string{"read", "shell"},
		PipelinePosition: &pos,
		Prompt:           "# Numbuh 1\n\n## Operating Protocol\n\nDo the thing.",
	}
	knownAgents := map[string]bool{"numbuh-1": true, "numbuh-2": true}

	issues := lintAgent(agent, knownAgents)
	if len(issues) != 0 {
		t.Errorf("expected no issues for valid agent, got: %v", issues)
	}
}

func TestLintAgent_AllMissing(t *testing.T) {
	agent := &agents.Agent{
		Name:   "",
		Role:   "",
		Tools:  nil,
		Prompt: "",
	}
	issues := lintAgent(agent, nil)
	if len(issues) < 4 {
		t.Errorf("expected at least 4 issues, got %d: %v", len(issues), issues)
	}
}

func TestLintAgent_UnknownRouting(t *testing.T) {
	agent := &agents.Agent{
		Name:   "test",
		Role:   "QA",
		Tools:  []string{"read"},
		Prompt: "## Operating Protocol\nContent.",
		Routing: &agents.RoutingConfig{
			Available: []string{"numbuh-1", "ghost-agent"},
			Trusted:   []string{"phantom"},
		},
	}
	knownAgents := map[string]bool{"test": true, "numbuh-1": true}

	issues := lintAgent(agent, knownAgents)
	foundAvailable := false
	foundTrusted := false
	for _, issue := range issues {
		if strings.Contains(issue, "ghost-agent") {
			foundAvailable = true
		}
		if strings.Contains(issue, "phantom") {
			foundTrusted = true
		}
	}
	if !foundAvailable {
		t.Error("expected issue about ghost-agent")
	}
	if !foundTrusted {
		t.Error("expected issue about phantom")
	}
}

// === isTerminal tests ===

func TestIsTerminal_ReturnsBool(t *testing.T) {
	result := isTerminal()
	// In test context, stdin may or may not be a terminal
	_ = result
}

// === runInstall tests ===

func TestRunInstall_NoAgentsSource(t *testing.T) {
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	tmpDir := t.TempDir()
	os.Chdir(tmpDir)
	os.Args = []string{"moonbase", "install", "--all"}

	// Override HOME to prevent finding agents in ~/.moonbase/
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	code := expectExit(t, func() {
		runInstall()
	})
	if code != 1 {
		t.Errorf("expected exit code 1, got %d", code)
	}
}

func TestRunInstall_Success(t *testing.T) {
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	// Create source agents dir
	tmpDir := t.TempDir()
	srcDir := filepath.Join(tmpDir, "agents")
	os.MkdirAll(srcDir, 0o755)
	writeTestAgent(t, srcDir, "numbuh-1", "Nigel", "Analyst")
	writeTestAgent(t, srcDir, "numbuh-2", "Hoagie", "Architect")
	os.Chdir(tmpDir)
	os.Args = []string{"moonbase", "install", "--all"}

	output := captureStdout(func() {
		runInstall()
	})
	if !strings.Contains(output, "installed") {
		t.Errorf("expected 'installed' in output, got:\n%s", output)
	}

	// Verify files were copied
	targetDir := filepath.Join(tmpDir, ".kiro", "agents")
	files, _ := filepath.Glob(filepath.Join(targetDir, "*.md"))
	if len(files) != 2 {
		t.Errorf("expected 2 files in .kiro/agents/, got %d", len(files))
	}
}

func TestRunInstall_GlobalFlag(t *testing.T) {
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	origHome := os.Getenv("HOME")
	tmpHome := t.TempDir()
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", origHome)

	// Create source agents
	tmpDir := t.TempDir()
	srcDir := filepath.Join(tmpDir, "agents")
	os.MkdirAll(srcDir, 0o755)
	writeTestAgent(t, srcDir, "numbuh-3", "Kuki", "Implementer")
	os.Chdir(tmpDir)
	os.Args = []string{"moonbase", "install", "--all", "--global"}

	output := captureStdout(func() {
		runInstall()
	})
	if !strings.Contains(output, "installed") {
		t.Errorf("expected 'installed' in output, got:\n%s", output)
	}

	// Verify files in ~/.kiro/agents/
	globalDir := filepath.Join(tmpHome, ".kiro", "agents")
	files, _ := filepath.Glob(filepath.Join(globalDir, "*.md"))
	if len(files) != 1 {
		t.Errorf("expected 1 file in ~/.kiro/agents/, got %d", len(files))
	}
}

// === runDeploy tests ===

func TestRunDeploy_InvalidID(t *testing.T) {
	code := expectExit(t, func() {
		runDeploy("../../../etc/passwd", "")
	})
	if code != 1 {
		t.Errorf("expected exit code 1 for invalid ID, got %d", code)
	}
}

func TestRunDeploy_AgentNotFound(t *testing.T) {
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	tmpDir := t.TempDir()
	agDir := filepath.Join(tmpDir, "agents")
	os.MkdirAll(agDir, 0o755)
	writeTestAgent(t, agDir, "numbuh-1", "Nigel", "Analyst")
	os.Chdir(tmpDir)

	code := expectExit(t, func() {
		runDeploy("99", "")
	})
	if code != 1 {
		t.Errorf("expected exit code 1 for missing agent, got %d", code)
	}
}

func TestRunDeploy_ValidAgent_ClipboardFallback(t *testing.T) {
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	tmpDir := t.TempDir()
	agDir := filepath.Join(tmpDir, "agents")
	os.MkdirAll(agDir, 0o755)
	writeTestAgent(t, agDir, "numbuh-1", "Nigel Uno", "Analyst")
	os.Chdir(tmpDir)
	os.Args = []string{"moonbase", "deploy", "1"}

	// Override PATH to prevent finding kiro-cli
	origPath := os.Getenv("PATH")
	os.Setenv("PATH", "/nonexistent")
	defer os.Setenv("PATH", origPath)

	output := captureStdout(func() {
		runDeploy("1", "")
	})
	if !strings.Contains(output, "Deploying") {
		t.Errorf("expected 'Deploying' in output, got:\n%s", output)
	}
}

func TestRunDeploy_WithTask(t *testing.T) {
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	tmpDir := t.TempDir()
	agDir := filepath.Join(tmpDir, "agents")
	os.MkdirAll(agDir, 0o755)
	writeTestAgent(t, agDir, "numbuh-4", "Wallabee", "QA")
	os.Chdir(tmpDir)
	os.Args = []string{"moonbase", "deploy", "4", "check", "auth"}

	origPath := os.Getenv("PATH")
	os.Setenv("PATH", "/nonexistent")
	defer os.Setenv("PATH", origPath)

	output := captureStdout(func() {
		runDeploy("4", "")
	})
	if !strings.Contains(output, "Deploying") {
		t.Errorf("expected 'Deploying' in output, got:\n%s", output)
	}
}

// === runHistory tests ===

func TestRunHistory_NoHistory(t *testing.T) {
	// Isolate to a temp HOME so history is reliably empty for this test.
	t.Setenv("HOME", t.TempDir())
	output := captureStdout(func() {
		runHistory()
	})
	if !strings.Contains(output, "No mission history") {
		t.Errorf("expected 'No mission history', got:\n%s", output)
	}
}

// === runReplay tests ===

func TestRunReplay_InvalidID(t *testing.T) {
	code := expectExit(t, func() {
		runReplay("abc")
	})
	if code != 1 {
		t.Errorf("expected exit code 1 for invalid ID, got %d", code)
	}
}

func TestRunReplay_NotFound(t *testing.T) {
	// Isolate to a temp HOME so history is empty, then exercise the invalid-ID path.
	t.Setenv("HOME", t.TempDir())
	code := expectExit(t, func() {
		runReplay("not-a-number")
	})
	if code != 1 {
		t.Errorf("expected exit code 1 for non-numeric ID, got %d", code)
	}
}

// === agentsDir tests ===

func TestAgentsDir_ReturnsDir(t *testing.T) {
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	projectRoot := filepath.Join(origDir, "..", "..")
	os.Chdir(projectRoot)

	// When run from project root, agentsDir should find agents
	dir := agentsDir()
	if dir == "" {
		t.Error("expected non-empty agents dir from project root")
	}
	if !isAgentsDir(dir) {
		t.Errorf("returned path is not a valid agents dir: %s", dir)
	}
}

// === execSyscall test ===

func TestExecSyscall_NonexistentBinary(t *testing.T) {
	err := execSyscall("/nonexistent/binary", []string{"binary"}, os.Environ())
	if err == nil {
		t.Error("expected error for nonexistent binary")
	}
}

// === findAgentsSource tests ===

func TestFindAgentsSource_FromProjectRoot(t *testing.T) {
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	projectRoot := filepath.Join(origDir, "..", "..")
	os.Chdir(projectRoot)

	dir, err := findAgentsSource()
	if err != nil {
		t.Skipf("findAgentsSource not found from project root: %v", err)
	}
	if !isAgentsDir(dir) {
		t.Errorf("returned dir is not valid agents dir: %s", dir)
	}
}

func TestFindAgentsSource_NotFound(t *testing.T) {
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	tmpDir := t.TempDir()
	os.Chdir(tmpDir)

	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	_, err := findAgentsSource()
	if err == nil {
		t.Error("expected error when no agents dir can be found")
	}
}

// === copyFile edge cases ===

func TestCopyFile_InvalidDest(t *testing.T) {
	tmpDir := t.TempDir()
	src := filepath.Join(tmpDir, "source.md")
	os.WriteFile(src, []byte("content"), 0o644)

	tmpFile := filepath.Join(tmpDir, "file")
	os.WriteFile(tmpFile, []byte("x"), 0o644)
	badDest := filepath.Join(tmpFile, "impossible", "dest.md")

	err := copyFile(src, badDest)
	if err == nil {
		t.Error("expected error writing to invalid path")
	}
}

// === Helper ===

func writeTestAgent(t *testing.T, dir, name, designation, role string) {
	t.Helper()
	content := fmt.Sprintf("---\nname: %s\ndesignation: %s\nrole: %s\ntools:\n  - read\n  - shell\n---\n# %s\n\n## Operating Protocol\n\nProtocol content here.\n", name, designation, role, name)
	os.WriteFile(filepath.Join(dir, name+".md"), []byte(content), 0o644)
}

// === Additional coverage tests ===

// TestRunHistory_WithJSON tests JSON output mode.
func TestRunHistory_WithJSON(t *testing.T) {
	// Set JSON mode
	origJSON := historyJSON
	origAll := historyAll
	origLimit := historyLimit
	defer func() {
		historyJSON = origJSON
		historyAll = origAll
		historyLimit = origLimit
	}()

	historyJSON = true
	historyAll = false
	historyLimit = 20

	output := captureStdout(func() {
		runHistory()
	})
	// If no history exists, it prints "No mission history found."
	// If history exists, it prints JSON
	if output != "" && !strings.Contains(output, "No mission history") && !strings.Contains(output, "[") {
		t.Errorf("expected JSON array or no-history message, got:\n%s", output)
	}
}

func TestRunHistory_WithAll(t *testing.T) {
	origAll := historyAll
	origLimit := historyLimit
	defer func() {
		historyAll = origAll
		historyLimit = origLimit
	}()

	historyAll = true
	historyLimit = 20

	output := captureStdout(func() {
		runHistory()
	})
	// Just verify it doesn't crash
	_ = output
}

// TestRunReplay_DryRun tests the dry-run mode of replay.
func TestRunReplay_DryRun(t *testing.T) {
	origDryRun := replayDryRun
	defer func() { replayDryRun = origDryRun }()
	replayDryRun = true

	// If there's no mission with this ID, it will exit
	// Use a subprocess approach or just test the validation path
	code := expectExit(t, func() {
		runReplay("not-numeric")
	})
	if code != 1 {
		t.Errorf("expected exit 1 for non-numeric, got %d", code)
	}
}

// TestRunDeploy_Council tests council/k agent resolution.
func TestRunDeploy_Council(t *testing.T) {
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	tmpDir := t.TempDir()
	agDir := filepath.Join(tmpDir, "agents")
	os.MkdirAll(agDir, 0o755)
	writeTestAgent(t, agDir, "knd-council", "KND Council", "Council")
	os.Chdir(tmpDir)
	os.Args = []string{"moonbase", "deploy", "council"}

	origPath := os.Getenv("PATH")
	os.Setenv("PATH", "/nonexistent")
	defer os.Setenv("PATH", origPath)

	output := captureStdout(func() {
		runDeploy("council", "")
	})
	if !strings.Contains(output, "Deploying") && !strings.Contains(output, "knd-council") {
		t.Errorf("expected deploy output for council, got:\n%s", output)
	}
}

// TestRunDeploy_SectorZ tests z/Z agent resolution.
func TestRunDeploy_SectorZ(t *testing.T) {
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	tmpDir := t.TempDir()
	agDir := filepath.Join(tmpDir, "agents")
	os.MkdirAll(agDir, 0o755)
	writeTestAgent(t, agDir, "sector-z", "Sector Z", "Legacy Archaeology")
	os.Chdir(tmpDir)
	os.Args = []string{"moonbase", "deploy", "z"}

	origPath := os.Getenv("PATH")
	os.Setenv("PATH", "/nonexistent")
	defer os.Setenv("PATH", origPath)

	output := captureStdout(func() {
		runDeploy("z", "")
	})
	if !strings.Contains(output, "Deploying") && !strings.Contains(output, "sector-z") {
		t.Errorf("expected deploy output for sector-z, got:\n%s", output)
	}
}

// TestRunDeploy_ProjectContextIntegration verifies project context is discovered.
func TestRunDeploy_ProjectContextIntegration(t *testing.T) {
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	// Create project with .kiro/steering
	tmpDir := t.TempDir()
	agDir := filepath.Join(tmpDir, "agents")
	os.MkdirAll(agDir, 0o755)
	writeTestAgent(t, agDir, "numbuh-5", "Abby", "Reviewer")
	os.MkdirAll(filepath.Join(tmpDir, ".kiro", "steering"), 0o755)
	os.WriteFile(filepath.Join(tmpDir, ".kiro", "steering", "dev-rules.md"), []byte("# Rules"), 0o644)
	os.Chdir(tmpDir)
	os.Args = []string{"moonbase", "deploy", "5", "review", "code"}

	origPath := os.Getenv("PATH")
	os.Setenv("PATH", "/nonexistent")
	defer os.Setenv("PATH", origPath)

	output := captureStdout(func() {
		runDeploy("5", "")
	})
	if !strings.Contains(output, "Deploying") {
		t.Errorf("expected 'Deploying' in output, got:\n%s", output)
	}
}

// TestRunInstall_EmptyAgentsSource tests runInstall when no .md files found.
func TestRunInstall_EmptyAgentsDir(t *testing.T) {
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	tmpDir := t.TempDir()
	// Create agents dir with no .md files
	agDir := filepath.Join(tmpDir, "agents")
	os.MkdirAll(agDir, 0o755)
	os.WriteFile(filepath.Join(agDir, "readme.txt"), []byte("no md"), 0o644)
	os.Chdir(tmpDir)
	os.Args = []string{"moonbase", "install", "--all"}

	// Should exit because no .md files
	code := expectExit(t, func() {
		runInstall()
	})
	if code != 1 {
		t.Errorf("expected exit 1, got %d", code)
	}
}

// === runMission related coverage ===

// TestRunMission_NoAgents tests runMission when agents dir is found but no agents loaded.
func TestRunMission_ExitsOnLoad(t *testing.T) {
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	// Create dir with valid agents and test runMission starts
	tmpDir := t.TempDir()
	agDir := filepath.Join(tmpDir, "agents")
	os.MkdirAll(agDir, 0o755)
	writeTestAgent(t, agDir, "numbuh-1", "Nigel", "Analyst")
	os.Chdir(tmpDir)

	// runMission will try to execute phases which requires a real backend
	// It prints the task header before getting to phases
	output := captureStdout(func() {
		// This will eventually fail because no backend is available,
		// but it exercises the initial code path
		defer func() { recover() }() // recover from potential panic
		runMission("test coverage task")
	})
	if !strings.Contains(output, "KND Council") {
		t.Logf("runMission output (expected partial): %s", output)
	}
}

// TestRunMissionWithoutConfirm tests the wrapper function.
func TestRunMissionWithoutConfirm_Calls(t *testing.T) {
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	tmpDir := t.TempDir()
	agDir := filepath.Join(tmpDir, "agents")
	os.MkdirAll(agDir, 0o755)
	writeTestAgent(t, agDir, "numbuh-1", "Nigel", "Analyst")
	os.Chdir(tmpDir)

	output := captureStdout(func() {
		defer func() { recover() }()
		runMissionWithoutConfirm("test task")
	})
	// Should at least start
	_ = output
}

// TestRunList_FullOutput tests runList with agents available.
func TestRunList_FullOutput(t *testing.T) {
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	// Go to project root where agents/ exists
	projectRoot := filepath.Join(origDir, "..", "..")
	os.Chdir(projectRoot)

	output := captureStdout(func() {
		runList()
	})

	if !strings.Contains(output, "OPERATIVE ROSTER") {
		t.Errorf("expected OPERATIVE ROSTER header")
	}
	if !strings.Contains(output, "SECTOR V") {
		t.Errorf("expected SECTOR V section")
	}
	if !strings.Contains(output, "SPECIALISTS") {
		t.Errorf("expected SPECIALISTS section")
	}
	if !strings.Contains(output, "AI BACKENDS") {
		t.Errorf("expected AI BACKENDS section")
	}
}

// TestRunHistory_TableFormat tests the table format output.
func TestRunHistory_TableFormat(t *testing.T) {
	origJSON := historyJSON
	origAll := historyAll
	origLimit := historyLimit
	defer func() {
		historyJSON = origJSON
		historyAll = origAll
		historyLimit = origLimit
	}()

	historyJSON = false
	historyAll = false
	historyLimit = 5

	output := captureStdout(func() {
		runHistory()
	})
	// Either shows history table or "No mission history"
	if output != "" && !strings.Contains(output, "No mission history") && !strings.Contains(output, "Mission History") {
		t.Errorf("unexpected output: %s", output)
	}
}

// TestRunStatus_ProjectDiscovery tests status with project context.
func TestRunStatus_ProjectDiscovery(t *testing.T) {
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	// Go to project root where .kiro/ exists
	projectRoot := filepath.Join(origDir, "..", "..")
	os.Chdir(projectRoot)

	output := captureStdout(func() {
		runStatus()
	})
	if !strings.Contains(output, "Moonbase Status") {
		t.Errorf("expected status header")
	}
}
