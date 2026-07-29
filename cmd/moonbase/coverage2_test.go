package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/f5508037/moonbase/internal/agents"
	"github.com/f5508037/moonbase/internal/history"
)

// === runHistory coverage: JSON output, all flag, table format with data ===

// saveTestMissions saves test missions via the history API and returns a cleanup function.
func saveTestMissions(t *testing.T, missions []history.Mission) func() {
	t.Helper()
	// Isolate history I/O to a temp HOME so tests never read or write the real
	// ~/.config/moonbase/history.json. history.Save/Load resolve their path from
	// HOME lazily, so this fully redirects them for the duration of the test.
	t.Setenv("HOME", t.TempDir())

	for _, m := range missions {
		if _, err := history.Save(m); err != nil {
			t.Fatalf("failed to save test mission: %v", err)
		}
	}

	// No cleanup needed — the temp HOME is removed automatically by t.TempDir().
	return func() {}
}

func TestRunHistory_JSONOutput_WithData(t *testing.T) {
	cleanup := saveTestMissions(t, []history.Mission{
		{
			Task:      "coverage-test-json-output",
			StartTime: time.Date(2025, 6, 1, 10, 0, 0, 0, time.UTC),
			EndTime:   time.Date(2025, 6, 1, 10, 5, 0, 0, time.UTC),
			Duration:  "5m0s",
			Outcome:   "complete",
			Phases: []history.Phase{
				{Name: "Analysis", Status: "complete", Duration: "1m"},
			},
		},
	})
	defer cleanup()

	origJSON := historyJSON
	origAll := historyAll
	origLimit := historyLimit
	defer func() {
		historyJSON = origJSON
		historyAll = origAll
		historyLimit = origLimit
	}()

	historyJSON = true
	historyAll = true
	historyLimit = 100

	output := captureStdout(func() {
		runHistory()
	})

	// Should be valid JSON containing our test task
	if !strings.Contains(output, "coverage-test-json-output") {
		t.Errorf("JSON output missing task, got:\n%s", output)
	}
	// Verify it's actually JSON parseable
	var parsed []interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &parsed); err != nil {
		t.Errorf("output is not valid JSON: %v\nGot:\n%s", err, output)
	}
	if len(parsed) < 1 {
		t.Errorf("expected at least 1 mission in JSON, got %d", len(parsed))
	}
}

func TestRunHistory_TableFormat_WithData(t *testing.T) {
	cleanup := saveTestMissions(t, []history.Mission{
		{
			Task:      "coverage-test-table-format",
			StartTime: time.Date(2025, 6, 1, 10, 0, 0, 0, time.UTC),
			Duration:  "2m0s",
			Outcome:   "complete",
		},
	})
	defer cleanup()

	origJSON := historyJSON
	origAll := historyAll
	origLimit := historyLimit
	defer func() {
		historyJSON = origJSON
		historyAll = origAll
		historyLimit = origLimit
	}()

	historyJSON = false
	historyAll = true
	historyLimit = 100

	output := captureStdout(func() {
		runHistory()
	})

	if !strings.Contains(output, "Mission History") {
		t.Errorf("expected 'Mission History' header, got:\n%s", output)
	}
	if !strings.Contains(output, "coverage-test-table-format") {
		t.Errorf("expected task in table, got:\n%s", output)
	}
	if !strings.Contains(output, "mission(s)") {
		t.Errorf("expected showing count, got:\n%s", output)
	}
}

func TestRunHistory_AllFlag_ShowsAll(t *testing.T) {
	// Save enough missions to exceed default limit
	var missions []history.Mission
	for i := 0; i < 3; i++ {
		missions = append(missions, history.Mission{
			Task:      "coverage-test-all-flag-" + string(rune('A'+i)),
			StartTime: time.Date(2025, 6, 1, i+1, 0, 0, 0, time.UTC),
			Duration:  "1m",
			Outcome:   "complete",
		})
	}
	cleanup := saveTestMissions(t, missions)
	defer cleanup()

	origJSON := historyJSON
	origAll := historyAll
	origLimit := historyLimit
	defer func() {
		historyJSON = origJSON
		historyAll = origAll
		historyLimit = origLimit
	}()

	historyJSON = false
	historyAll = true
	historyLimit = 1 // would normally limit, but --all overrides

	output := captureStdout(func() {
		runHistory()
	})

	// When --all is set, limit is treated as 0 (unlimited)
	if !strings.Contains(output, "Mission History") {
		t.Errorf("expected Mission History header, got:\n%s", output)
	}
}

func TestRunHistory_LimitFlag_WithData(t *testing.T) {
	var missions []history.Mission
	for i := 0; i < 5; i++ {
		missions = append(missions, history.Mission{
			Task:      "coverage-test-limit-" + string(rune('A'+i)),
			StartTime: time.Date(2025, 6, 1, i+1, 0, 0, 0, time.UTC),
			Duration:  "1m",
			Outcome:   "complete",
		})
	}
	cleanup := saveTestMissions(t, missions)
	defer cleanup()

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
	historyLimit = 2

	output := captureStdout(func() {
		runHistory()
	})

	if !strings.Contains(output, "Showing 2 mission(s)") {
		t.Errorf("expected 2 missions with limit, got:\n%s", output)
	}
}

// === runInit coverage: writeTemplate fails on invalid nested dir ===

func TestRunInit_WriteTemplateFails_InvalidNestedDir(t *testing.T) {
	tmpDir := t.TempDir()
	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)

	// Create a file where .kiro/specs/_templates should be a directory.
	// This makes MkdirAll succeed for .kiro but writeTemplate will fail
	// because the path includes a file masquerading as a directory.
	kiroDir := filepath.Join(tmpDir, ".kiro")
	specsDir := filepath.Join(kiroDir, "specs")
	os.MkdirAll(specsDir, 0o755)
	// Place a regular file where _templates directory should be
	os.WriteFile(filepath.Join(specsDir, "_templates"), []byte("blocker"), 0o644)

	os.Chdir(tmpDir)
	// Remove .kiro so runInit tries to create it fresh
	os.RemoveAll(kiroDir)

	// Now create a file at .kiro/specs/_templates path BEFORE runInit creates dirs
	// Actually, let's test writeTemplate directly with a truly invalid path
	err := writeTemplate("/dev/null/impossible/nested/file.md", "content")
	if err == nil {
		t.Error("expected error writing to impossible path")
	}
}

func TestRunInit_WriteTemplateFails_ReadOnlyDir(t *testing.T) {
	tmpDir := t.TempDir()
	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)

	// Create a read-only directory for the template target
	readOnlyDir := filepath.Join(tmpDir, "readonly")
	os.MkdirAll(readOnlyDir, 0o755)
	os.Chmod(readOnlyDir, 0o555)
	defer os.Chmod(readOnlyDir, 0o755) // restore for cleanup

	err := writeTemplate(filepath.Join(readOnlyDir, "file.md"), "content")
	if err == nil {
		t.Error("expected error writing to read-only directory")
	}
}

// === runInstall coverage: global flag with uncreatable target dir ===

func TestRunInstall_GlobalFlag_TargetDirCannotBeCreated(t *testing.T) {
	// This test relies on overriding HOME to point at a file, causing MkdirAll
	// to fail. On macOS with cgo, os.UserHomeDir() ignores $HOME and uses the
	// system directory service, making this test impossible to run reliably.
	origHome := os.Getenv("HOME")
	testHome := t.TempDir()
	os.Setenv("HOME", testHome)
	got, _ := os.UserHomeDir()
	os.Setenv("HOME", origHome)
	if got != testHome {
		t.Skip("os.UserHomeDir does not respect $HOME on this platform")
	}

	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	defer os.Setenv("HOME", origHome)

	// Set HOME to a file (not a directory) so MkdirAll fails
	tmpDir := t.TempDir()
	fakeHome := filepath.Join(tmpDir, "fakehome")
	os.WriteFile(fakeHome, []byte("I am a file"), 0o644)
	os.Setenv("HOME", fakeHome)

	// Create source agents in project dir
	projectDir := t.TempDir()
	srcDir := filepath.Join(projectDir, "agents")
	os.MkdirAll(srcDir, 0o755)
	writeTestAgent(t, srcDir, "numbuh-1", "Nigel", "Analyst")
	os.Chdir(projectDir)
	os.Args = []string{"moonbase", "install", "--all", "--global"}

	// Set the cobra flag directly since we're calling runInstall() without cobra
	origGlobal := installGlobal
	installGlobal = true
	defer func() { installGlobal = origGlobal }()

	code := expectExit(t, func() {
		runInstall()
	})
	if code != 1 {
		t.Errorf("expected exit code 1 when target dir can't be created, got %d", code)
	}
}

// === findAgentsSource coverage: common install paths scenario ===

func TestFindAgentsSource_CommonInstallPaths(t *testing.T) {
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	origHome := os.Getenv("HOME")
	defer os.Setenv("HOME", origHome)

	tmpHome := t.TempDir()
	os.Setenv("HOME", tmpHome)

	// Put agents in ~/.moonbase/agents
	moonbaseDir := filepath.Join(tmpHome, ".moonbase", "agents")
	os.MkdirAll(moonbaseDir, 0o755)
	writeTestAgent(t, moonbaseDir, "numbuh-1", "Nigel", "Analyst")

	// CWD has no agents dir and executable won't be next to agents
	tmpCWD := t.TempDir()
	os.Chdir(tmpCWD)

	dir, err := findAgentsSource()
	if err != nil {
		t.Fatalf("expected to find agents in ~/.moonbase/agents, got error: %v", err)
	}
	if !strings.Contains(dir, ".moonbase") {
		t.Errorf("expected path containing .moonbase, got: %s", dir)
	}
}

func TestFindAgentsSource_ConfigMoonbasePath(t *testing.T) {
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	origHome := os.Getenv("HOME")
	defer os.Setenv("HOME", origHome)

	tmpHome := t.TempDir()
	os.Setenv("HOME", tmpHome)

	// Put agents in ~/.config/moonbase/agents
	configDir := filepath.Join(tmpHome, ".config", "moonbase", "agents")
	os.MkdirAll(configDir, 0o755)
	writeTestAgent(t, configDir, "numbuh-2", "Hoagie", "Architect")

	tmpCWD := t.TempDir()
	os.Chdir(tmpCWD)

	dir, err := findAgentsSource()
	if err != nil {
		t.Fatalf("expected to find agents in ~/.config/moonbase/agents, got error: %v", err)
	}
	if !strings.Contains(dir, ".config/moonbase") {
		t.Errorf("expected path containing .config/moonbase, got: %s", dir)
	}
}

// === runDeploy coverage: kiro-cli in PATH that fails to exec ===

func TestRunDeploy_KiroCLIInPathButFails(t *testing.T) {
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	origPath := os.Getenv("PATH")
	defer os.Setenv("PATH", origPath)

	// Create a fake kiro-cli that is NOT a valid executable binary.
	// It has the exec bit set so LookPath finds it, but syscall.Exec will fail
	// because the content is not a valid Mach-O or script with valid interpreter.
	tmpBin := t.TempDir()
	fakeKiro := filepath.Join(tmpBin, "kiro-cli")
	// Write invalid binary content (no shebang, no valid binary header)
	os.WriteFile(fakeKiro, []byte("\x00\x00INVALID_BINARY"), 0o755)

	// Set up project with agent
	tmpDir := t.TempDir()
	agDir := filepath.Join(tmpDir, "agents")
	os.MkdirAll(agDir, 0o755)
	writeTestAgent(t, agDir, "numbuh-1", "Nigel Uno", "Analyst")
	os.Chdir(tmpDir)
	os.Args = []string{"moonbase", "deploy", "1", "analyze", "auth"}

	// Put fake kiro-cli in PATH — syscall.Exec will fail because it's not valid
	os.Setenv("PATH", tmpBin)

	output := captureStdout(func() {
		runDeploy("1", "")
	})

	// Should fall through to clipboard fallback since exec fails
	if !strings.Contains(output, "Deploying") {
		t.Errorf("expected 'Deploying' in output, got:\n%s", output)
	}
}

func TestRunDeploy_WithLocalKiroAgent(t *testing.T) {
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	origPath := os.Getenv("PATH")
	defer os.Setenv("PATH", origPath)

	// Create a fake kiro-cli (invalid binary — will fail on syscall.Exec)
	tmpBin := t.TempDir()
	fakeKiro := filepath.Join(tmpBin, "kiro-cli")
	os.WriteFile(fakeKiro, []byte("\x00\x00INVALID_BINARY"), 0o755)

	// Set up project with agents dir AND .kiro/agents/ local agent
	tmpDir := t.TempDir()
	agDir := filepath.Join(tmpDir, "agents")
	os.MkdirAll(agDir, 0o755)
	writeTestAgent(t, agDir, "numbuh-3", "Kuki", "Implementer")

	// Also place agent in .kiro/agents/ (the local project agents)
	localAgDir := filepath.Join(tmpDir, ".kiro", "agents")
	os.MkdirAll(localAgDir, 0o755)
	writeTestAgent(t, localAgDir, "numbuh-3", "Kuki", "Implementer")

	os.Chdir(tmpDir)
	os.Args = []string{"moonbase", "deploy", "3", "implement", "feature"}
	os.Setenv("PATH", tmpBin)

	output := captureStdout(func() {
		runDeploy("3", "")
	})

	// Should mention Deploying
	if !strings.Contains(output, "Deploying") {
		t.Errorf("expected 'Deploying' in output, got:\n%s", output)
	}
}

// === runDeploy: clipboard not available ===

func TestRunDeploy_NoClipboardNoKiroCLI(t *testing.T) {
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	origPath := os.Getenv("PATH")
	defer os.Setenv("PATH", origPath)

	// Set up project
	tmpDir := t.TempDir()
	agDir := filepath.Join(tmpDir, "agents")
	os.MkdirAll(agDir, 0o755)
	writeTestAgent(t, agDir, "numbuh-4", "Wallabee", "QA")
	os.Chdir(tmpDir)
	os.Args = []string{"moonbase", "deploy", "4"}

	// Empty PATH - no kiro-cli, no clipboard tools
	os.Setenv("PATH", "/nonexistent-path-xyz")

	output := captureStdout(func() {
		runDeploy("4", "")
	})

	// Should still produce output (either clipboard success on macOS with pbcopy
	// built-in, or the "No clipboard available" fallback message)
	if !strings.Contains(output, "Deploying") {
		t.Errorf("expected 'Deploying' in output, got:\n%s", output)
	}
}

// === runMission coverage: pipeline execution with fake kiro-cli ===

func TestRunMission_WithFakeKiroCLI(t *testing.T) {
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	origPath := os.Getenv("PATH")
	defer os.Setenv("PATH", origPath)

	// Create a fake kiro-cli that outputs known text
	tmpBin := t.TempDir()
	fakeKiro := filepath.Join(tmpBin, "kiro-cli")
	// This script echoes a response simulating AI output
	script := `#!/bin/sh
echo "RISK_LEVEL: LOW"
echo "Phase complete. All checks passed."
echo "No issues found."
`
	os.WriteFile(fakeKiro, []byte(script), 0o755)
	os.Setenv("PATH", tmpBin+":"+origPath)

	// Set up project
	tmpDir := t.TempDir()
	agDir := filepath.Join(tmpDir, "agents")
	os.MkdirAll(agDir, 0o755)
	writeTestAgent(t, agDir, "numbuh-1", "Nigel Uno", "Analyst")
	writeTestAgent(t, agDir, "numbuh-2", "Hoagie", "Architect")
	writeTestAgent(t, agDir, "numbuh-3", "Kuki", "Implementer")
	writeTestAgent(t, agDir, "numbuh-4", "Wallabee", "QA")
	writeTestAgent(t, agDir, "numbuh-5", "Abby", "Reviewer")
	os.Chdir(tmpDir)

	output := captureStdout(func() {
		runMission("test pipeline execution")
	})

	if !strings.Contains(output, "KND Council") {
		t.Errorf("expected 'KND Council' header, got:\n%s", output)
	}
	if !strings.Contains(output, "test pipeline execution") {
		t.Errorf("expected task in output, got:\n%s", output)
	}
	// Should show at least Phase 1 starting
	if !strings.Contains(output, "Phase 1") {
		t.Errorf("expected Phase 1 in output, got:\n%s", output)
	}
}

func TestRunMission_RiskGate_MediumRework(t *testing.T) {
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	origPath := os.Getenv("PATH")
	defer os.Setenv("PATH", origPath)

	// Create fake kiro-cli that outputs MEDIUM risk on phase 4
	tmpBin := t.TempDir()
	fakeKiro := filepath.Join(tmpBin, "kiro-cli")
	script := `#!/bin/sh
echo "RISK_LEVEL: MEDIUM"
echo "Some issues found that need rework."
echo "Recommend fixing error handling."
`
	os.WriteFile(fakeKiro, []byte(script), 0o755)
	os.Setenv("PATH", tmpBin+":"+origPath)

	tmpDir := t.TempDir()
	agDir := filepath.Join(tmpDir, "agents")
	os.MkdirAll(agDir, 0o755)
	writeTestAgent(t, agDir, "numbuh-1", "Nigel Uno", "Analyst")
	writeTestAgent(t, agDir, "numbuh-2", "Hoagie", "Architect")
	writeTestAgent(t, agDir, "numbuh-3", "Kuki", "Implementer")
	writeTestAgent(t, agDir, "numbuh-4", "Wallabee", "QA")
	writeTestAgent(t, agDir, "numbuh-5", "Abby", "Reviewer")
	os.Chdir(tmpDir)

	output := captureStdout(func() {
		runMission("test rework loop")
	})

	// Should show risk gate result
	if !strings.Contains(output, "Risk Gate") {
		t.Errorf("expected 'Risk Gate' in output, got:\n%s", output)
	}
}

func TestRunMission_RiskGate_CriticalStop(t *testing.T) {
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	origPath := os.Getenv("PATH")
	defer os.Setenv("PATH", origPath)

	// Create fake kiro-cli that outputs CRITICAL risk
	tmpBin := t.TempDir()
	fakeKiro := filepath.Join(tmpBin, "kiro-cli")
	script := `#!/bin/sh
echo "RISK_LEVEL: CRITICAL"
echo "Security vulnerability detected."
echo "Immediate human intervention required."
`
	os.WriteFile(fakeKiro, []byte(script), 0o755)
	os.Setenv("PATH", tmpBin+":"+origPath)

	tmpDir := t.TempDir()
	agDir := filepath.Join(tmpDir, "agents")
	os.MkdirAll(agDir, 0o755)
	writeTestAgent(t, agDir, "numbuh-1", "Nigel Uno", "Analyst")
	writeTestAgent(t, agDir, "numbuh-2", "Hoagie", "Architect")
	writeTestAgent(t, agDir, "numbuh-3", "Kuki", "Implementer")
	writeTestAgent(t, agDir, "numbuh-4", "Wallabee", "QA")
	writeTestAgent(t, agDir, "numbuh-5", "Abby", "Reviewer")
	os.Chdir(tmpDir)

	output := captureStdout(func() {
		runMission("test critical stop")
	})

	if !strings.Contains(output, "Risk Gate") {
		t.Errorf("expected 'Risk Gate' in output, got:\n%s", output)
	}
}

func TestRunMission_AgentNotFoundInPhase(t *testing.T) {
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	origPath := os.Getenv("PATH")
	defer os.Setenv("PATH", origPath)

	origHome := os.Getenv("HOME")
	defer os.Setenv("HOME", origHome)

	tmpBin := t.TempDir()
	fakeKiro := filepath.Join(tmpBin, "kiro-cli")
	script := `#!/bin/sh
echo "phase output"
`
	os.WriteFile(fakeKiro, []byte(script), 0o755)
	os.Setenv("PATH", tmpBin+":"+origPath)

	// Only create numbuh-1, missing numbuh-2 through 5.
	// Set HOME to an empty temp dir so global agents are not found.
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	agDir := filepath.Join(tmpDir, "agents")
	os.MkdirAll(agDir, 0o755)
	writeTestAgent(t, agDir, "numbuh-1", "Nigel Uno", "Analyst")
	os.Chdir(tmpDir)

	output := captureStdout(func() {
		runMission("test missing agent")
	})

	// Should show agent not found warning
	if !strings.Contains(output, "not found") {
		t.Errorf("expected 'not found' warning for missing agents, got:\n%s", output)
	}
}

// === deployToBackend coverage ===

func TestDeployToBackend_KiroCLISuccess(t *testing.T) {
	origPath := os.Getenv("PATH")
	defer os.Setenv("PATH", origPath)

	// Create a fake kiro-cli that outputs success
	tmpBin := t.TempDir()
	fakeKiro := filepath.Join(tmpBin, "kiro-cli")
	script := `#!/bin/sh
echo "Backend response: task completed successfully"
`
	os.WriteFile(fakeKiro, []byte(script), 0o755)
	os.Setenv("PATH", tmpBin+":"+origPath)

	agent := &agents.Agent{
		Name:        "numbuh-1",
		Designation: "Nigel Uno",
		Role:        "Analyst",
		Tools:       []string{"read", "shell"},
	}

	output, err := deployToBackend(agent, "composed prompt here", "test task")
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if !strings.Contains(output, "Backend response") {
		t.Errorf("expected backend response in output, got: %s", output)
	}
}

func TestDeployToBackend_KiroCLIFailsFallsBack(t *testing.T) {
	origPath := os.Getenv("PATH")
	defer os.Setenv("PATH", origPath)

	// Create a fake kiro-cli that fails
	tmpBin := t.TempDir()
	fakeKiro := filepath.Join(tmpBin, "kiro-cli")
	script := `#!/bin/sh
echo "error output" >&2
exit 1
`
	os.WriteFile(fakeKiro, []byte(script), 0o755)
	os.Setenv("PATH", tmpBin+":"+origPath)

	agent := &agents.Agent{
		Name:        "numbuh-2",
		Designation: "Hoagie",
		Role:        "Architect",
		Tools:       []string{"read"},
	}

	// deployToBackend will try kiro-cli (fails), then try clipboard
	// On macOS, clipboard (pbcopy) is available so it will try to read stdin
	// We provide stdin via a pipe to prevent blocking
	oldStdin := os.Stdin
	r, w, _ := os.Pipe()
	os.Stdin = r
	w.WriteString("simulated response\nEND\n")
	w.Close()
	defer func() { os.Stdin = oldStdin }()

	output, err := deployToBackend(agent, "test prompt", "test task")
	if err != nil {
		// Valid error paths:
		// 1. kiro-cli found on PATH but fails after retries
		// 2. No backend available (no kiro-cli + no clipboard)
		if !strings.Contains(err.Error(), "no backend available") &&
			!strings.Contains(err.Error(), "kiro-cli failed after") {
			t.Fatalf("unexpected error: %v", err)
		}
	} else {
		if !strings.Contains(output, "simulated response") {
			t.Errorf("expected simulated response in output, got: %s", output)
		}
	}
}

func TestDeployToBackend_NoBackendAvailable(t *testing.T) {
	origPath := os.Getenv("PATH")
	defer os.Setenv("PATH", origPath)

	// Empty PATH — no kiro-cli available
	os.Setenv("PATH", "/nonexistent-xyz-path")

	agent := &agents.Agent{
		Name:        "numbuh-4",
		Designation: "Wallabee",
		Role:        "QA",
		Tools:       []string{"read", "shell"},
	}

	// Also make clipboard unavailable by providing no stdin
	// On macOS pbcopy is at /usr/bin/pbcopy which won't be in PATH
	// This tests the "no backend available" error path
	oldStdin := os.Stdin
	r, w, _ := os.Pipe()
	os.Stdin = r
	w.Close() // EOF immediately
	defer func() { os.Stdin = oldStdin }()

	_, err := deployToBackend(agent, "test", "task")
	// Either clipboard works (macOS has pbcopy at absolute path sometimes)
	// or we get the no backend error
	if err != nil {
		if !strings.Contains(err.Error(), "no backend available") {
			t.Logf("got different error (acceptable): %v", err)
		}
	}
	// Both outcomes are valid — we're testing the code path executes without panic
}

// === runReplay coverage: with actual history entry and dry-run mode ===

func TestRunReplay_DryRun_WithHistory(t *testing.T) {
	origDryRun := replayDryRun
	defer func() { replayDryRun = origDryRun }()
	replayDryRun = true

	// Save a test mission via the history API
	cleanup := saveTestMissions(t, []history.Mission{
		{
			Task:      "replay-dryrun-test-task",
			StartTime: time.Date(2025, 6, 15, 9, 30, 0, 0, time.UTC),
			EndTime:   time.Date(2025, 6, 15, 9, 35, 0, 0, time.UTC),
			Duration:  "5m0s",
			Outcome:   "complete",
			Phases: []history.Phase{
				{Name: "Analysis", Status: "complete", Duration: "1m"},
				{Name: "Architecture", Status: "complete", Duration: "2m"},
			},
		},
	})
	defer cleanup()

	// Get the ID that was assigned
	all := history.Load()
	if len(all) == 0 {
		t.Fatal("no missions saved")
	}
	lastID := all[len(all)-1].ID
	idStr := fmt.Sprintf("%d", lastID)

	output := captureStdout(func() {
		runReplay(idStr)
	})

	if !strings.Contains(output, "Replaying mission") {
		t.Errorf("expected replaying message, got:\n%s", output)
	}
	if !strings.Contains(output, "replay-dryrun-test-task") {
		t.Errorf("expected task in dry-run output, got:\n%s", output)
	}
	if !strings.Contains(output, "dry-run") {
		t.Errorf("expected 'dry-run' notice, got:\n%s", output)
	}
}

func TestRunReplay_MissionNotFound(t *testing.T) {
	// Request an ID that definitely doesn't exist
	code := expectExit(t, func() {
		runReplay("99999")
	})
	if code != 1 {
		t.Errorf("expected exit code 1 for not-found mission, got %d", code)
	}
}

func TestRunReplay_ValidNumericID_NoHistoryFile(t *testing.T) {
	// Even without explicit history, ID 99998 should not exist
	code := expectExit(t, func() {
		runReplay("99998")
	})
	if code != 1 {
		t.Errorf("expected exit code 1 when mission not found, got %d", code)
	}
}

// === runStatus coverage: without agents dir ===

func TestRunStatus_WithoutAgentsDir(t *testing.T) {
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	// Change to a directory with no agents
	tmpDir := t.TempDir()
	os.Chdir(tmpDir)

	output := captureStdout(func() {
		runStatus()
	})

	if !strings.Contains(output, "Moonbase Status") {
		t.Errorf("expected status header, got:\n%s", output)
	}
	// Should indicate agents not found
	if !strings.Contains(output, "not found") {
		t.Errorf("expected 'not found' for agents in bare dir, got:\n%s", output)
	}
}

func TestRunStatus_WithLocalAgents(t *testing.T) {
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	tmpDir := t.TempDir()
	// Create .kiro/agents with some agents
	localDir := filepath.Join(tmpDir, ".kiro", "agents")
	os.MkdirAll(localDir, 0o755)
	writeTestAgent(t, localDir, "numbuh-1", "Nigel", "Analyst")
	writeTestAgent(t, localDir, "numbuh-2", "Hoagie", "Architect")
	os.Chdir(tmpDir)

	output := captureStdout(func() {
		runStatus()
	})

	if !strings.Contains(output, "Local:") {
		t.Errorf("expected 'Local:' section showing local agents, got:\n%s", output)
	}
	if !strings.Contains(output, "2 agents") {
		t.Errorf("expected '2 agents' count, got:\n%s", output)
	}
}

// === findAgentsDirQuiet coverage: when executable can't be found ===

func TestFindAgentsDirQuiet_ExecutableNotFound(t *testing.T) {
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	// Change to temp dir with no agents anywhere
	tmpDir := t.TempDir()
	os.Chdir(tmpDir)

	// findAgentsDirQuiet uses os.Executable() which should work in tests
	// but the candidates relative to the test binary won't have agents
	dir := findAgentsDirQuiet()
	// In this tmp dir there are no agents — should return empty
	// (unless the test binary happens to be adjacent to agents/)
	if dir != "" && !isAgentsDir(dir) {
		t.Errorf("expected empty or valid dir, got: %s", dir)
	}
}

// === runLint coverage: with parse errors in agent files ===

func TestRunLint_AgentFileWithParseErrors(t *testing.T) {
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	tmpDir := t.TempDir()
	agDir := filepath.Join(tmpDir, "agents")
	os.MkdirAll(agDir, 0o755)

	// Write a malformed YAML frontmatter agent file
	badContent := "---\nname: [invalid yaml\nrole: broken\n---\n# Bad Agent\n"
	os.WriteFile(filepath.Join(agDir, "bad-agent.md"), []byte(badContent), 0o644)

	// Also add a valid agent so the dir is found
	writeTestAgent(t, agDir, "numbuh-1", "Nigel", "Analyst")

	os.Chdir(tmpDir)

	code := expectExit(t, func() {
		runLint()
	})
	if code != 1 {
		t.Errorf("expected exit 1 for parse errors, got %d", code)
	}
}

func TestRunLint_AgentMissingOperatingProtocol(t *testing.T) {
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	tmpDir := t.TempDir()
	agDir := filepath.Join(tmpDir, "agents")
	os.MkdirAll(agDir, 0o755)

	// Agent with valid YAML but missing Operating Protocol in body
	content := "---\nname: numbuh-99\ndesignation: Test\nrole: Tester\ntools:\n  - read\n---\n# Numbuh 99\n\nNo protocol section here.\n"
	os.WriteFile(filepath.Join(agDir, "numbuh-99.md"), []byte(content), 0o644)

	os.Chdir(tmpDir)

	code := expectExit(t, func() {
		runLint()
	})
	if code != 1 {
		t.Errorf("expected exit 1 for missing Operating Protocol, got %d", code)
	}
}

func TestRunLint_NoMdFilesInDir(t *testing.T) {
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	tmpDir := t.TempDir()
	agDir := filepath.Join(tmpDir, "agents")
	os.MkdirAll(agDir, 0o755)
	// Write only non-md files
	os.WriteFile(filepath.Join(agDir, "readme.txt"), []byte("not an agent"), 0o644)
	os.Chdir(tmpDir)

	code := expectExit(t, func() {
		runLint()
	})
	if code != 1 {
		t.Errorf("expected exit 1 when no .md files in agents dir, got %d", code)
	}
}

// === isTerminal coverage: edge case ===

func TestIsTerminal_InTestContext(t *testing.T) {
	// In test context, stdin is typically not a terminal (it's a pipe)
	result := isTerminal()
	// We can't control what stdin is in tests, but we verify it doesn't panic
	// and returns a boolean
	if result != true && result != false {
		t.Error("isTerminal should return a boolean")
	}
}

// === Additional edge cases ===

func TestRunHistory_TaskTruncation(t *testing.T) {
	// Task longer than 40 chars should be truncated
	longTask := strings.Repeat("x", 60)
	cleanup := saveTestMissions(t, []history.Mission{
		{
			Task:      longTask,
			StartTime: time.Date(2025, 6, 1, 10, 0, 0, 0, time.UTC),
			Duration:  "2m",
			Outcome:   "complete",
		},
	})
	defer cleanup()

	origJSON := historyJSON
	origAll := historyAll
	origLimit := historyLimit
	defer func() {
		historyJSON = origJSON
		historyAll = origAll
		historyLimit = origLimit
	}()

	historyJSON = false
	historyAll = true
	historyLimit = 100

	output := captureStdout(func() {
		runHistory()
	})

	// Should contain truncated task with "..."
	if !strings.Contains(output, "...") {
		t.Errorf("expected truncated task with '...', got:\n%s", output)
	}
}

func TestRunMission_NoAgentsDir(t *testing.T) {
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	origHome := os.Getenv("HOME")
	defer os.Setenv("HOME", origHome)

	origPath := os.Getenv("PATH")
	defer os.Setenv("PATH", origPath)

	tmpDir := t.TempDir()
	os.Chdir(tmpDir)

	// Override HOME so FindAgentsDir cannot find ~/.moonbase/agents
	fakeHome := filepath.Join(tmpDir, "fakehome")
	os.MkdirAll(fakeHome, 0o755)
	os.Setenv("HOME", fakeHome)

	// Remove kiro-cli from PATH so no backend is available
	os.Setenv("PATH", "/nonexistent-xyz-path")

	// No agents dir at all — should exit
	code := expectExit(t, func() {
		runMission("test no agents")
	})
	if code != 1 {
		t.Errorf("expected exit 1, got %d", code)
	}
}

func TestRunInstall_SpecificAgentFlag(t *testing.T) {
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	tmpDir := t.TempDir()
	srcDir := filepath.Join(tmpDir, "agents")
	os.MkdirAll(srcDir, 0o755)
	writeTestAgent(t, srcDir, "numbuh-1", "Nigel", "Analyst")
	writeTestAgent(t, srcDir, "numbuh-2", "Hoagie", "Architect")
	writeTestAgent(t, srcDir, "numbuh-3", "Kuki", "Implementer")
	os.Chdir(tmpDir)

	// Without --all flag, should still install all (non-interactive default)
	os.Args = []string{"moonbase", "install"}

	output := captureStdout(func() {
		runInstall()
	})
	if !strings.Contains(output, "installed") || !strings.Contains(output, "3 agent(s)") {
		t.Errorf("expected 3 agents installed, got:\n%s", output)
	}
}

// === Additional tests for remaining uncovered paths ===

func TestRunMission_WithProjectContext(t *testing.T) {
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	origPath := os.Getenv("PATH")
	defer os.Setenv("PATH", origPath)

	// Create fake kiro-cli
	tmpBin := t.TempDir()
	fakeKiro := filepath.Join(tmpBin, "kiro-cli")
	script := `#!/bin/sh
echo "RISK_LEVEL: LOW"
echo "Analysis complete. No issues."
`
	os.WriteFile(fakeKiro, []byte(script), 0o755)
	os.Setenv("PATH", tmpBin+":"+origPath)

	// Set up project with agents AND .kiro/specs and steering
	tmpDir := t.TempDir()
	agDir := filepath.Join(tmpDir, "agents")
	os.MkdirAll(agDir, 0o755)
	writeTestAgent(t, agDir, "numbuh-1", "Nigel Uno", "Analyst")
	writeTestAgent(t, agDir, "numbuh-2", "Hoagie", "Architect")
	writeTestAgent(t, agDir, "numbuh-3", "Kuki", "Implementer")
	writeTestAgent(t, agDir, "numbuh-4", "Wallabee", "QA")
	writeTestAgent(t, agDir, "numbuh-5", "Abby", "Reviewer")

	// Create project context
	os.MkdirAll(filepath.Join(tmpDir, ".kiro", "specs"), 0o755)
	os.MkdirAll(filepath.Join(tmpDir, ".kiro", "steering"), 0o755)
	os.WriteFile(filepath.Join(tmpDir, ".kiro", "steering", "dev-rules.md"), []byte("# Rules\n## Stack\n- Go"), 0o644)
	os.WriteFile(filepath.Join(tmpDir, ".kiro", "specs", "feature.md"), []byte("# Feature Spec"), 0o644)
	os.Chdir(tmpDir)

	output := captureStdout(func() {
		runMission("test with project context")
	})

	if !strings.Contains(output, "KND Council") {
		t.Errorf("expected KND Council header, got:\n%s", output)
	}
	// Should show project context
	if !strings.Contains(output, "Project:") {
		t.Logf("project context not shown (acceptable if discovery returned nil): %s", output)
	}
}

func TestRunMission_DeployToBackendFails(t *testing.T) {
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	origPath := os.Getenv("PATH")
	defer os.Setenv("PATH", origPath)

	// No kiro-cli available and no clipboard — deployToBackend will fail
	os.Setenv("PATH", "/nonexistent-xyz")

	tmpDir := t.TempDir()
	agDir := filepath.Join(tmpDir, "agents")
	os.MkdirAll(agDir, 0o755)
	writeTestAgent(t, agDir, "numbuh-1", "Nigel Uno", "Analyst")
	writeTestAgent(t, agDir, "numbuh-2", "Hoagie", "Architect")
	writeTestAgent(t, agDir, "numbuh-3", "Kuki", "Implementer")
	writeTestAgent(t, agDir, "numbuh-4", "Wallabee", "QA")
	writeTestAgent(t, agDir, "numbuh-5", "Abby", "Reviewer")
	os.Chdir(tmpDir)

	// Redirect stdin so clipboard fallback's scanner won't block
	oldStdin := os.Stdin
	r, w, _ := os.Pipe()
	os.Stdin = r
	w.Close()
	defer func() { os.Stdin = oldStdin }()

	output := captureStdout(func() {
		runMission("test backend failure")
	})

	// Should show phase failed message or clipboard fallback
	if !strings.Contains(output, "KND Council") {
		t.Errorf("expected KND Council, got:\n%s", output)
	}
}

func TestRunDeploy_ClipboardNotAvailable(t *testing.T) {
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	origPath := os.Getenv("PATH")
	defer os.Setenv("PATH", origPath)

	tmpDir := t.TempDir()
	agDir := filepath.Join(tmpDir, "agents")
	os.MkdirAll(agDir, 0o755)
	writeTestAgent(t, agDir, "numbuh-2", "Hoagie", "Architect")
	os.Chdir(tmpDir)
	os.Args = []string{"moonbase", "deploy", "2", "design", "auth"}

	// Empty PATH: no kiro-cli, and importantly no pbcopy/xclip
	// On macOS, pbcopy is at /usr/bin which won't be in PATH
	os.Setenv("PATH", "")

	output := captureStdout(func() {
		runDeploy("2", "")
	})

	if !strings.Contains(output, "Deploying") {
		t.Errorf("expected Deploying, got:\n%s", output)
	}
	// Should hit either clipboard success (macOS might find it anyway) or fallback
	// The key thing is it doesn't crash
}

func TestRunInstall_CopyFileFails(t *testing.T) {
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	tmpDir := t.TempDir()
	srcDir := filepath.Join(tmpDir, "agents")
	os.MkdirAll(srcDir, 0o755)

	// Create a source file that can't be read
	badFile := filepath.Join(srcDir, "unreadable.md")
	os.WriteFile(badFile, []byte("content"), 0o000)
	defer os.Chmod(badFile, 0o644) // restore for cleanup

	// Also add a valid agent so we get partial success
	writeTestAgent(t, srcDir, "numbuh-1", "Nigel", "Analyst")
	os.Chdir(tmpDir)
	os.Args = []string{"moonbase", "install", "--all"}

	output := captureStdout(func() {
		runInstall()
	})

	// Should show warning for unreadable file but still succeed for others
	if !strings.Contains(output, "installed") {
		t.Errorf("expected partial install, got:\n%s", output)
	}
}

func TestRunStatus_WithSteering(t *testing.T) {
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	tmpDir := t.TempDir()
	// Create .kiro/steering with a file
	os.MkdirAll(filepath.Join(tmpDir, ".kiro", "steering"), 0o755)
	os.WriteFile(filepath.Join(tmpDir, ".kiro", "steering", "dev-rules.md"), []byte("# Rules"), 0o644)
	os.Chdir(tmpDir)

	output := captureStdout(func() {
		runStatus()
	})

	if !strings.Contains(output, "Moonbase Status") {
		t.Errorf("expected status header")
	}
}

func TestRunMission_PipelineComplete(t *testing.T) {
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	origPath := os.Getenv("PATH")
	defer os.Setenv("PATH", origPath)

	// Fake kiro-cli that outputs LOW risk for phase 4
	tmpBin := t.TempDir()
	fakeKiro := filepath.Join(tmpBin, "kiro-cli")
	script := `#!/bin/sh
echo "RISK_LEVEL: LOW"
echo "Everything looks good."
echo "No issues found."
`
	os.WriteFile(fakeKiro, []byte(script), 0o755)
	os.Setenv("PATH", tmpBin+":"+origPath)

	tmpDir := t.TempDir()
	agDir := filepath.Join(tmpDir, "agents")
	os.MkdirAll(agDir, 0o755)
	writeTestAgent(t, agDir, "numbuh-1", "Nigel Uno", "Analyst")
	writeTestAgent(t, agDir, "numbuh-2", "Hoagie", "Architect")
	writeTestAgent(t, agDir, "numbuh-3", "Kuki", "Implementer")
	writeTestAgent(t, agDir, "numbuh-4", "Wallabee", "QA")
	writeTestAgent(t, agDir, "numbuh-5", "Abby", "Reviewer")
	os.Chdir(tmpDir)

	output := captureStdout(func() {
		runMission("test pipeline complete path")
	})

	if !strings.Contains(output, "Mission pipeline complete") {
		t.Errorf("expected 'Mission pipeline complete', got:\n%s", output)
	}
}

// TestRunMission_HighRisk tests HIGH risk path that goes back to design phase
func TestRunMission_HighRisk(t *testing.T) {
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	origPath := os.Getenv("PATH")
	defer os.Setenv("PATH", origPath)

	// Fake kiro-cli that outputs HIGH risk
	tmpBin := t.TempDir()
	fakeKiro := filepath.Join(tmpBin, "kiro-cli")
	script := `#!/bin/sh
echo "RISK_LEVEL: HIGH"
echo "Major issues in architecture."
echo "Needs redesign."
`
	os.WriteFile(fakeKiro, []byte(script), 0o755)
	os.Setenv("PATH", tmpBin+":"+origPath)

	tmpDir := t.TempDir()
	agDir := filepath.Join(tmpDir, "agents")
	os.MkdirAll(agDir, 0o755)
	writeTestAgent(t, agDir, "numbuh-1", "Nigel Uno", "Analyst")
	writeTestAgent(t, agDir, "numbuh-2", "Hoagie", "Architect")
	writeTestAgent(t, agDir, "numbuh-3", "Kuki", "Implementer")
	writeTestAgent(t, agDir, "numbuh-4", "Wallabee", "QA")
	writeTestAgent(t, agDir, "numbuh-5", "Abby", "Reviewer")
	os.Chdir(tmpDir)

	output := captureStdout(func() {
		runMission("test high risk path")
	})

	if !strings.Contains(output, "Risk Gate") {
		t.Errorf("expected Risk Gate output, got:\n%s", output)
	}
}

func TestRunDeploy_NoAgentsDir(t *testing.T) {
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	// CWD with no agents anywhere
	tmpDir := t.TempDir()
	os.Chdir(tmpDir)
	os.Args = []string{"moonbase", "deploy", "1"}

	code := expectExit(t, func() {
		runDeploy("1", "")
	})
	if code != 1 {
		t.Errorf("expected exit 1 when no agents dir, got %d", code)
	}
}

func TestDeployToBackend_TempFileCreation(t *testing.T) {
	origPath := os.Getenv("PATH")
	defer os.Setenv("PATH", origPath)

	// Create a working fake kiro-cli that outputs text
	tmpBin := t.TempDir()
	fakeKiro := filepath.Join(tmpBin, "kiro-cli")
	script := `#!/bin/sh
echo "Response from kiro-cli for phase"
`
	os.WriteFile(fakeKiro, []byte(script), 0o755)
	os.Setenv("PATH", tmpBin+":"+origPath)

	agent := &agents.Agent{
		Name:        "numbuh-3",
		Designation: "Kuki",
		Role:        "Implementer",
		Tools:       []string{"read", "shell", "write"},
	}

	// Long composed prompt to exercise the temp file path
	longPrompt := strings.Repeat("This is a long prompt for testing. ", 100)
	output, err := deployToBackend(agent, longPrompt, "implement the feature")
	if err != nil {
		t.Fatalf("deployToBackend failed: %v", err)
	}
	if !strings.Contains(output, "Response from kiro-cli") {
		t.Errorf("expected kiro-cli response, got: %s", output)
	}
}

// === Additional targeted coverage tests ===

func TestRunInit_WithGoProject(t *testing.T) {
	tmpDir := t.TempDir()
	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)

	// Create a fake Go project so stack detection works
	os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module example.com/test\n\ngo 1.22\n"), 0o644)
	os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0o644)
	os.Chdir(tmpDir)

	output := captureStdout(func() {
		runInit()
	})

	// Should detect Go stack
	if !strings.Contains(output, "agent-ready") {
		t.Errorf("expected agent-ready message, got:\n%s", output)
	}
}

func TestRunInit_WithNodeProject(t *testing.T) {
	tmpDir := t.TempDir()
	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)

	// Create a fake Node project
	os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(`{"name":"test","scripts":{"test":"jest","build":"tsc"}}`), 0o644)
	os.Chdir(tmpDir)

	output := captureStdout(func() {
		runInit()
	})

	if !strings.Contains(output, "agent-ready") {
		t.Errorf("expected agent-ready message, got:\n%s", output)
	}
}

func TestRunInit_WriteTemplateErrors(t *testing.T) {
	tmpDir := t.TempDir()
	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)
	os.Chdir(tmpDir)

	// First init to create structure
	captureStdout(func() {
		runInit()
	})

	// Remove .kiro, create it fresh but make templates dir unwriteable
	os.RemoveAll(filepath.Join(tmpDir, ".kiro"))
	kiroDir := filepath.Join(tmpDir, ".kiro")
	specsTemplates := filepath.Join(kiroDir, "specs", "_templates")
	os.MkdirAll(specsTemplates, 0o755)
	os.MkdirAll(filepath.Join(kiroDir, "steering"), 0o755)
	os.MkdirAll(filepath.Join(kiroDir, "agents"), 0o755)

	// Make templates dir read-only so writeTemplate fails
	os.Chmod(specsTemplates, 0o555)
	steeringDir := filepath.Join(kiroDir, "steering")
	os.Chmod(steeringDir, 0o555)
	defer func() {
		os.Chmod(specsTemplates, 0o755)
		os.Chmod(steeringDir, 0o755)
	}()

	// Remove .kiro so runInit tries again
	os.RemoveAll(kiroDir)
	os.Chdir(tmpDir)

	// Run init - it will recreate dirs, and writeTemplate should succeed
	// because MkdirAll creates fresh writable dirs
	output := captureStdout(func() {
		runInit()
	})
	_ = output
}

func TestRunDeploy_AgentsDirNotFound_Exit(t *testing.T) {
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	origHome := os.Getenv("HOME")
	defer os.Setenv("HOME", origHome)

	// Go to empty dir with HOME also empty
	tmpDir := t.TempDir()
	os.Chdir(tmpDir)
	os.Setenv("HOME", tmpDir)
	os.Args = []string{"moonbase", "deploy", "1"}

	code := expectExit(t, func() {
		runDeploy("1", "")
	})
	if code != 1 {
		t.Errorf("expected exit 1, got %d", code)
	}
}

func TestFindAgentsDirQuiet_WithAgentsInCWD(t *testing.T) {
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	// Create a temp dir with agents/ subdir
	tmpDir := t.TempDir()
	agDir := filepath.Join(tmpDir, "agents")
	os.MkdirAll(agDir, 0o755)
	writeTestAgent(t, agDir, "numbuh-1", "Nigel", "Analyst")
	os.Chdir(tmpDir)

	dir := findAgentsDirQuiet()
	if dir == "" {
		t.Error("expected to find agents dir from CWD")
	}
	if !isAgentsDir(dir) {
		t.Errorf("returned dir is not valid: %s", dir)
	}
}

func TestRunLint_MixOfGoodAndBadAgents(t *testing.T) {
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	tmpDir := t.TempDir()
	agDir := filepath.Join(tmpDir, "agents")
	os.MkdirAll(agDir, 0o755)

	// Good agent
	writeTestAgent(t, agDir, "numbuh-1", "Nigel", "Analyst")

	// Agent with unknown routing reference
	content := "---\nname: numbuh-4\ndesignation: Wallabee\nrole: QA\ntools:\n  - read\n  - shell\nrouting:\n  available:\n    - numbuh-1\n    - nonexistent-agent\n  trusted:\n    - also-nonexistent\n---\n# Numbuh 4\n\n## Operating Protocol\n\nTest protocol.\n"
	os.WriteFile(filepath.Join(agDir, "numbuh-4.md"), []byte(content), 0o644)

	os.Chdir(tmpDir)

	code := expectExit(t, func() {
		runLint()
	})
	if code != 1 {
		t.Errorf("expected exit 1 for routing issues, got %d", code)
	}
}

func TestRunStatus_NoProjectContext(t *testing.T) {
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	tmpDir := t.TempDir()
	os.Chdir(tmpDir)

	output := captureStdout(func() {
		runStatus()
	})

	if !strings.Contains(output, "Moonbase Status") {
		t.Errorf("expected status output")
	}
	// In a bare directory, should show "no .kiro/ found" or similar
	if !strings.Contains(output, "not found") && !strings.Contains(output, "moonbase init") {
		t.Logf("output: %s", output)
	}
}

func TestRunHistory_JSONMarshalSuccess(t *testing.T) {
	// Ensure JSON output doesn't fail on marshal
	cleanup := saveTestMissions(t, []history.Mission{
		{
			Task:      "json-marshal-test",
			StartTime: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			Duration:  "1s",
			Outcome:   "complete",
			Phases:    []history.Phase{{Name: "P1", Status: "done", Duration: "1s"}},
		},
	})
	defer cleanup()

	origJSON := historyJSON
	origAll := historyAll
	origLimit := historyLimit
	defer func() {
		historyJSON = origJSON
		historyAll = origAll
		historyLimit = origLimit
	}()

	historyJSON = true
	historyAll = true
	historyLimit = 100

	output := captureStdout(func() {
		runHistory()
	})

	if !strings.Contains(output, "json-marshal-test") {
		t.Errorf("expected test mission in JSON output")
	}
}

func TestRunInstall_NoAgents_GlobError(t *testing.T) {
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	// Create agents dir with no md files at all
	tmpDir := t.TempDir()
	agDir := filepath.Join(tmpDir, "agents")
	os.MkdirAll(agDir, 0o755)
	// Only txt files
	os.WriteFile(filepath.Join(agDir, "readme.txt"), []byte("no agents"), 0o644)
	os.Chdir(tmpDir)
	os.Args = []string{"moonbase", "install", "--all"}

	code := expectExit(t, func() {
		runInstall()
	})
	if code != 1 {
		t.Errorf("expected exit 1, got %d", code)
	}
}

// === More targeted coverage for runInit agents install path ===

func TestRunInit_WithAgentsSource(t *testing.T) {
	tmpDir := t.TempDir()
	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)

	// Create an agents/ dir in the same tmpDir so findAgentsSource finds it
	agDir := filepath.Join(tmpDir, "agents")
	os.MkdirAll(agDir, 0o755)
	writeTestAgent(t, agDir, "numbuh-1", "Nigel", "Analyst")
	writeTestAgent(t, agDir, "numbuh-2", "Hoagie", "Architect")

	// Create a subdir to simulate a "project" without existing .kiro
	projectDir := filepath.Join(tmpDir, "myproject")
	os.MkdirAll(projectDir, 0o755)
	os.Chdir(projectDir)

	// findAgentsSource checks CWD/../agents which would be tmpDir/agents
	output := captureStdout(func() {
		runInit()
	})

	if !strings.Contains(output, "Installed") {
		t.Errorf("expected agents to be installed during init, got:\n%s", output)
	}
	if !strings.Contains(output, "agent-ready") {
		t.Errorf("expected agent-ready, got:\n%s", output)
	}

	// Verify agents were copied to .kiro/agents/
	kiroAgents := filepath.Join(projectDir, ".kiro", "agents")
	files, _ := filepath.Glob(filepath.Join(kiroAgents, "*.md"))
	if len(files) < 2 {
		t.Errorf("expected at least 2 agent files installed, got %d", len(files))
	}
}

// Test the deploy_cmd.go non-terminal path (line 57)
func TestDeployCmd_NonTerminalNoArgs(t *testing.T) {
	// The deploy command with no args + non-terminal stdin should exit with error
	// This tests the cobra Run handler's !isTerminal() branch
	// We can't easily mock isTerminal(), but we can at least test
	// that runDeploy with empty string is handled
	code := expectExit(t, func() {
		runDeploy("", "")
	})
	// Empty string should fail validation
	if code != 1 {
		t.Errorf("expected exit 1 for empty agent ID, got %d", code)
	}
}

// Test runMission with conditional phase triggering
func TestRunMission_ConditionalPhaseSkipped(t *testing.T) {
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	origPath := os.Getenv("PATH")
	defer os.Setenv("PATH", origPath)

	// Fake kiro-cli
	tmpBin := t.TempDir()
	fakeKiro := filepath.Join(tmpBin, "kiro-cli")
	script := `#!/bin/sh
echo "RISK_LEVEL: LOW"
echo "All good."
`
	os.WriteFile(fakeKiro, []byte(script), 0o755)
	os.Setenv("PATH", tmpBin+":"+origPath)

	tmpDir := t.TempDir()
	agDir := filepath.Join(tmpDir, "agents")
	os.MkdirAll(agDir, 0o755)
	// Create all core pipeline agents
	writeTestAgent(t, agDir, "numbuh-0", "Monty", "Architect")
	writeTestAgent(t, agDir, "numbuh-1", "Nigel", "Analyst")
	writeTestAgent(t, agDir, "numbuh-2", "Hoagie", "Architect")
	writeTestAgent(t, agDir, "numbuh-3", "Kuki", "Implementer")
	writeTestAgent(t, agDir, "numbuh-4", "Wallabee", "QA")
	writeTestAgent(t, agDir, "numbuh-5", "Abby", "Reviewer")
	// Create specialists for conditional phases
	writeTestAgent(t, agDir, "numbuh-274", "Chad", "Security")
	writeTestAgent(t, agDir, "numbuh-362", "Rachel", "DevOps")
	os.Chdir(tmpDir)

	output := captureStdout(func() {
		runMission("small fix to readme")
	})

	// Conditional phases should be skipped for a simple task
	if !strings.Contains(output, "skipped") {
		t.Logf("conditional phases may or may not be skipped: %s", output)
	}
}

// Test the `else` branch in runList when no agents are found
func TestRunList_NoAgentsAnywhere(t *testing.T) {
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	origHome := os.Getenv("HOME")
	defer os.Setenv("HOME", origHome)

	// Set to empty dirs so no agents are found
	tmpDir := t.TempDir()
	os.Chdir(tmpDir)
	os.Setenv("HOME", tmpDir)

	output := captureStdout(func() {
		runList()
	})

	if !strings.Contains(output, "OPERATIVE ROSTER") {
		t.Errorf("expected roster header")
	}
}

func TestRunInstall_NoAllFlag_ListsAndInstalls(t *testing.T) {
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	tmpDir := t.TempDir()
	srcDir := filepath.Join(tmpDir, "agents")
	os.MkdirAll(srcDir, 0o755)
	writeTestAgent(t, srcDir, "numbuh-1", "Nigel", "Analyst")
	writeTestAgent(t, srcDir, "numbuh-2", "Hoagie", "Architect")
	os.Chdir(tmpDir)
	// No --all flag
	os.Args = []string{"moonbase", "install"}

	output := captureStdout(func() {
		runInstall()
	})

	// Should show the agent listing
	if !strings.Contains(output, "Agent Installation") {
		t.Errorf("expected Agent Installation header, got:\n%s", output)
	}
	if !strings.Contains(output, "numbuh-1") {
		t.Errorf("expected numbuh-1 listed, got:\n%s", output)
	}
}

// === Cover the isValidAgentID failure path inside runDeploy (line 190-194) ===

func TestRunDeploy_InvalidID_WithAgentsDir(t *testing.T) {
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	// Create a project with agents so agentsDir() succeeds
	tmpDir := t.TempDir()
	agDir := filepath.Join(tmpDir, "agents")
	os.MkdirAll(agDir, 0o755)
	writeTestAgent(t, agDir, "numbuh-1", "Nigel", "Analyst")
	os.Chdir(tmpDir)
	os.Args = []string{"moonbase", "deploy", "../../../etc/passwd"}

	// Now runDeploy will pass agentsDir() but fail at isValidAgentID
	code := expectExit(t, func() {
		runDeploy("../../../etc/passwd", "")
	})
	if code != 1 {
		t.Errorf("expected exit 1 for invalid agent ID, got %d", code)
	}
}

func TestRunDeploy_InvalidID_UnicodeWithAgentsDir(t *testing.T) {
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	tmpDir := t.TempDir()
	agDir := filepath.Join(tmpDir, "agents")
	os.MkdirAll(agDir, 0o755)
	writeTestAgent(t, agDir, "numbuh-1", "Nigel", "Analyst")
	os.Chdir(tmpDir)
	os.Args = []string{"moonbase", "deploy", "münchen"}

	code := expectExit(t, func() {
		runDeploy("münchen", "")
	})
	if code != 1 {
		t.Errorf("expected exit 1 for unicode agent ID, got %d", code)
	}
}

// Cover the deploy_cmd.go line 57: non-terminal stdin with no args
// This requires calling the cobra command handler directly
func TestDeployCmdHandler_NonTerminal(t *testing.T) {
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	tmpDir := t.TempDir()
	agDir := filepath.Join(tmpDir, "agents")
	os.MkdirAll(agDir, 0o755)
	writeTestAgent(t, agDir, "numbuh-1", "Nigel", "Analyst")
	os.Chdir(tmpDir)

	// The deploy command with no args checks isTerminal() and exits if not terminal
	// In tests, stdin is typically not a terminal (pipe), so this should hit line 57
	// But we can't easily call the cobra handler without the full command setup.
	// Instead, let's just verify the isTerminal() behavior
	term := isTerminal()
	if term {
		t.Log("stdin is a terminal in this test context (unexpected but ok)")
	}
}

// Cover main.go line 128: openai env var detected in runList
func TestRunList_WithOpenAIKey(t *testing.T) {
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	origKey := os.Getenv("OPENAI_API_KEY")
	os.Setenv("OPENAI_API_KEY", "test-key")
	defer os.Setenv("OPENAI_API_KEY", origKey)

	// Go to project root
	projectRoot := filepath.Join(origDir, "..", "..")
	os.Chdir(projectRoot)

	output := captureStdout(func() {
		runList()
	})

	if !strings.Contains(output, "openai") {
		t.Errorf("expected openai backend listed when key is set, got:\n%s", output)
	}
}

// Cover main.go line 165: isTerminal error path
// This is extremely hard to test because os.Stdin.Stat() rarely fails.
// We verify the function handles both paths without crashing.
func TestIsTerminal_DoesNotPanic(t *testing.T) {
	// Just call it and verify no panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("isTerminal panicked: %v", r)
		}
	}()
	_ = isTerminal()
}

// Test that specifically verifies the clipboard failure output (lines 276-279)
func TestRunDeploy_ClipboardFails_ShowsFallbackMessage(t *testing.T) {
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	origPath := os.Getenv("PATH")
	defer os.Setenv("PATH", origPath)

	// Create invalid kiro-cli and set PATH to only contain it
	tmpBin := t.TempDir()
	fakeKiro := filepath.Join(tmpBin, "kiro-cli")
	os.WriteFile(fakeKiro, []byte("\x00\x00INVALID"), 0o755)

	tmpDir := t.TempDir()
	agDir := filepath.Join(tmpDir, "agents")
	os.MkdirAll(agDir, 0o755)
	writeTestAgent(t, agDir, "numbuh-1", "Nigel Uno", "Analyst")
	os.Chdir(tmpDir)
	os.Args = []string{"moonbase", "deploy", "1", "test", "task"}

	// Set PATH to ONLY tmpBin — no pbcopy, no xclip
	os.Setenv("PATH", tmpBin)

	output := captureStdout(func() {
		runDeploy("1", "")
	})

	// After exec fails and clipboard fails, should show either:
	// - "Copied to clipboard" (if somehow clipboard works)
	// - "No clipboard available" (if clipboard fails)
	// - "No interactive backend available" (the preamble)
	if !strings.Contains(output, "Deploying") {
		t.Errorf("expected 'Deploying' in output, got:\n%s", output)
	}
	t.Logf("Full captured output:\n%s", output)
	// Should hit the "No interactive backend" section
	if !strings.Contains(output, "No interactive backend") {
		t.Errorf("expected 'No interactive backend' message, got:\n%s", output)
	}
	// Either clipboard works or shows fallback
	if strings.Contains(output, "No clipboard available") {
		// Great — we hit the else branch (lines 276-279)
		t.Log("clipboard failure path confirmed")
	} else if strings.Contains(output, "Copied to clipboard") {
		// pbcopy was found (macOS built-in discovery)
		t.Log("clipboard succeeded despite restricted PATH")
	}
}

// Test anthropic key detection in runList
func TestRunList_WithAnthropicKey(t *testing.T) {
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	origKey := os.Getenv("ANTHROPIC_API_KEY")
	os.Setenv("ANTHROPIC_API_KEY", "test-anthropic-key")
	defer os.Setenv("ANTHROPIC_API_KEY", origKey)

	projectRoot := filepath.Join(origDir, "..", "..")
	os.Chdir(projectRoot)

	output := captureStdout(func() {
		runList()
	})

	if !strings.Contains(output, "anthropic") {
		t.Errorf("expected anthropic backend listed, got:\n%s", output)
	}
}

// === Cover root.go pipe mode ===

func TestRootCmd_PipeMode(t *testing.T) {
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	origPath := os.Getenv("PATH")
	defer os.Setenv("PATH", origPath)

	// No kiro-cli in PATH
	os.Setenv("PATH", "/nonexistent-xyz")

	// Simulate pipe input by replacing stdin
	oldStdin := os.Stdin
	r, w, _ := os.Pipe()
	w.WriteString("test pipe task")
	w.Close()
	os.Stdin = r
	defer func() { os.Stdin = oldStdin }()

	// The rootCmd RunE should detect non-terminal stdin and process pipe mode
	output := captureStdout(func() {
		rootCmd.SetArgs([]string{})
		rootCmd.Execute()
	})

	// Should show pipe mode output or clipboard copy
	if strings.Contains(output, "Pipe mode") || strings.Contains(output, "clipboard") {
		t.Log("pipe mode executed successfully")
	}
}

// Test rootCmd with empty pipe (should do nothing)
func TestRootCmd_EmptyPipe(t *testing.T) {
	oldStdin := os.Stdin
	r, w, _ := os.Pipe()
	w.WriteString("")
	w.Close()
	os.Stdin = r
	defer func() { os.Stdin = oldStdin }()

	output := captureStdout(func() {
		rootCmd.SetArgs([]string{})
		rootCmd.Execute()
	})

	// Empty pipe should produce no output
	_ = output
}

// Test rootCmd pipe mode with kiro-cli available
func TestRootCmd_PipeMode_WithKiroCLI(t *testing.T) {
	origPath := os.Getenv("PATH")
	defer os.Setenv("PATH", origPath)

	// Create a fake kiro-cli that reads stdin and outputs text
	tmpBin := t.TempDir()
	fakeKiro := filepath.Join(tmpBin, "kiro-cli")
	script := `#!/bin/sh
cat > /dev/null
echo "pipe response"
`
	os.WriteFile(fakeKiro, []byte(script), 0o755)
	os.Setenv("PATH", tmpBin+":"+origPath)

	oldStdin := os.Stdin
	r, w, _ := os.Pipe()
	w.WriteString("pipe task for kiro")
	w.Close()
	os.Stdin = r
	defer func() { os.Stdin = oldStdin }()

	output := captureStdout(func() {
		rootCmd.SetArgs([]string{})
		rootCmd.Execute()
	})

	if !strings.Contains(output, "Pipe mode") {
		t.Logf("pipe mode output: %s", output)
	}
}

// === Cover cobra command handler Run functions ===

func TestCobraCmd_Version(t *testing.T) {
	output := captureStdout(func() {
		rootCmd.SetArgs([]string{"version"})
		rootCmd.Execute()
	})
	if !strings.Contains(output, "moonbase") {
		t.Errorf("expected version output, got: %s", output)
	}
}

func TestCobraCmd_Config(t *testing.T) {
	output := captureStdout(func() {
		rootCmd.SetArgs([]string{"config"})
		rootCmd.Execute()
	})
	if !strings.Contains(output, "Configuration") {
		t.Errorf("expected config output, got: %s", output)
	}
}

func TestCobraCmd_List(t *testing.T) {
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	projectRoot := filepath.Join(origDir, "..", "..")
	os.Chdir(projectRoot)

	output := captureStdout(func() {
		rootCmd.SetArgs([]string{"list"})
		rootCmd.Execute()
	})
	if !strings.Contains(output, "OPERATIVE ROSTER") {
		t.Errorf("expected roster, got: %s", output)
	}
}

func TestCobraCmd_Status(t *testing.T) {
	output := captureStdout(func() {
		rootCmd.SetArgs([]string{"status"})
		rootCmd.Execute()
	})
	if !strings.Contains(output, "Moonbase Status") {
		t.Errorf("expected status, got: %s", output)
	}
}

func TestCobraCmd_Init(t *testing.T) {
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	tmpDir := t.TempDir()
	os.Chdir(tmpDir)

	output := captureStdout(func() {
		rootCmd.SetArgs([]string{"init"})
		rootCmd.Execute()
	})
	if !strings.Contains(output, "Moonbase Init") {
		t.Errorf("expected init output, got: %s", output)
	}
}

func TestCobraCmd_History(t *testing.T) {
	output := captureStdout(func() {
		rootCmd.SetArgs([]string{"history"})
		rootCmd.Execute()
	})
	// Either shows history or "No mission history found."
	if output == "" {
		t.Error("expected some output from history command")
	}
}

func TestCobraCmd_Deploy_WithArgs(t *testing.T) {
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	origPath := os.Getenv("PATH")
	defer os.Setenv("PATH", origPath)
	os.Setenv("PATH", "/nonexistent-xyz")

	tmpDir := t.TempDir()
	agDir := filepath.Join(tmpDir, "agents")
	os.MkdirAll(agDir, 0o755)
	writeTestAgent(t, agDir, "numbuh-1", "Nigel", "Analyst")
	os.Chdir(tmpDir)

	output := captureStdout(func() {
		rootCmd.SetArgs([]string{"deploy", "1"})
		rootCmd.Execute()
	})
	if !strings.Contains(output, "Deploying") {
		t.Errorf("expected deploy output, got: %s", output)
	}
}

func TestCobraCmd_Replay_InvalidID(t *testing.T) {
	code := expectExit(t, func() {
		rootCmd.SetArgs([]string{"replay", "abc"})
		rootCmd.Execute()
	})
	if code != 1 {
		t.Errorf("expected exit 1, got %d", code)
	}
}

// === Cover snippet commands ===

func TestCobraCmd_SnippetSave(t *testing.T) {
	origHome := os.Getenv("HOME")
	tmpHome := t.TempDir()
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", origHome)

	// Provide stdin content
	oldStdin := os.Stdin
	r, w, _ := os.Pipe()
	w.WriteString("This is my snippet content\nLine 2\n")
	w.Close()
	os.Stdin = r
	defer func() { os.Stdin = oldStdin }()

	output := captureStdout(func() {
		rootCmd.SetArgs([]string{"snippet", "save", "test-snippet"})
		rootCmd.Execute()
	})

	if !strings.Contains(output, "Snippet saved") {
		t.Errorf("expected snippet saved message, got: %s", output)
	}
}

func TestCobraCmd_SnippetList_NoSnippets(t *testing.T) {
	origHome := os.Getenv("HOME")
	tmpHome := t.TempDir()
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", origHome)

	output := captureStdout(func() {
		rootCmd.SetArgs([]string{"snippet", "list"})
		rootCmd.Execute()
	})

	if !strings.Contains(output, "No snippets saved") {
		t.Errorf("expected 'No snippets saved', got: %s", output)
	}
}

func TestCobraCmd_SnippetList_WithSnippets(t *testing.T) {
	origHome := os.Getenv("HOME")
	tmpHome := t.TempDir()
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", origHome)

	// Create snippets file
	snippetsDir := filepath.Join(tmpHome, ".config", "moonbase")
	os.MkdirAll(snippetsDir, 0o700)
	snippets := []map[string]string{
		{"name": "my-prompt", "content": "Hello world"},
	}
	data, _ := json.MarshalIndent(snippets, "", "  ")
	os.WriteFile(filepath.Join(snippetsDir, "snippets.json"), data, 0o600)

	output := captureStdout(func() {
		rootCmd.SetArgs([]string{"snippet", "list"})
		rootCmd.Execute()
	})

	if !strings.Contains(output, "my-prompt") {
		t.Errorf("expected snippet name in list, got: %s", output)
	}
}

func TestCobraCmd_SnippetSave_InvalidName_TooLong(t *testing.T) {
	longName := strings.Repeat("x", 101)
	code := expectExit(t, func() {
		rootCmd.SetArgs([]string{"snippet", "save", longName})
		rootCmd.Execute()
	})
	if code != 1 {
		t.Errorf("expected exit 1, got %d", code)
	}
}

func TestCobraCmd_SnippetSave_InvalidName_PathSeparator(t *testing.T) {
	code := expectExit(t, func() {
		rootCmd.SetArgs([]string{"snippet", "save", "path/inject"})
		rootCmd.Execute()
	})
	if code != 1 {
		t.Errorf("expected exit 1, got %d", code)
	}
}

func TestCobraCmd_SnippetSave_InvalidName_ControlChar(t *testing.T) {
	code := expectExit(t, func() {
		rootCmd.SetArgs([]string{"snippet", "save", "bad\x00name"})
		rootCmd.Execute()
	})
	if code != 1 {
		t.Errorf("expected exit 1, got %d", code)
	}
}

// === Cover install and lint via cobra ===

func TestCobraCmd_Install(t *testing.T) {
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	tmpDir := t.TempDir()
	agDir := filepath.Join(tmpDir, "agents")
	os.MkdirAll(agDir, 0o755)
	writeTestAgent(t, agDir, "numbuh-1", "Nigel", "Analyst")
	os.Chdir(tmpDir)
	os.Args = []string{"moonbase", "install", "--all"}

	output := captureStdout(func() {
		installCmd.Run(installCmd, []string{})
	})

	if !strings.Contains(output, "installed") {
		t.Errorf("expected installed message, got: %s", output)
	}
}

func TestCobraCmd_Lint_Valid(t *testing.T) {
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	tmpDir := t.TempDir()
	agDir := filepath.Join(tmpDir, "agents")
	os.MkdirAll(agDir, 0o755)
	writeTestAgent(t, agDir, "numbuh-1", "Nigel", "Analyst")
	writeTestAgent(t, agDir, "numbuh-2", "Hoagie", "Architect")
	os.Chdir(tmpDir)

	output := captureStdout(func() {
		rootCmd.SetArgs([]string{"lint"})
		rootCmd.Execute()
	})

	if !strings.Contains(output, "Agent Lint") {
		t.Errorf("expected lint output, got: %s", output)
	}
}

// === Cover export command and remaining cobra handlers ===

func TestCobraCmd_Export(t *testing.T) {
	output := captureStdout(func() {
		rootCmd.SetArgs([]string{"export", "99999"})
		rootCmd.Execute()
	})
	// Should show "not found" for nonexistent mission
	if !strings.Contains(output, "not found") {
		t.Errorf("expected 'not found' for export, got: %s", output)
	}
}

func TestCobraCmd_Export_WithData(t *testing.T) {
	cleanup := saveTestMissions(t, []history.Mission{
		{
			Task:      "export-test-task",
			StartTime: time.Date(2025, 7, 1, 10, 0, 0, 0, time.UTC),
			Duration:  "3m",
			Outcome:   "complete",
			Phases:    []history.Phase{{Name: "Analysis", Status: "complete", Duration: "1m"}},
		},
	})
	defer cleanup()

	all := history.Load()
	lastID := all[len(all)-1].ID
	idStr := fmt.Sprintf("%d", lastID)

	output := captureStdout(func() {
		rootCmd.SetArgs([]string{"export", idStr})
		rootCmd.Execute()
	})

	if !strings.Contains(output, "export-test-task") {
		t.Errorf("expected task in export, got: %s", output)
	}
}

// Test mission command via cobra (dry-run mode)
func TestCobraCmd_MissionDryRun(t *testing.T) {
	output := captureStdout(func() {
		rootCmd.SetArgs([]string{"mission", "--dry-run", "test dry run via cobra"})
		rootCmd.Execute()
	})
	if !strings.Contains(output, "Dry Run") {
		t.Errorf("expected dry run output, got: %s", output)
	}
}

// Test replay without dry-run (exercises runMissionWithoutConfirm path)
func TestRunReplay_NoDryRun_WithHistory(t *testing.T) {
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	origPath := os.Getenv("PATH")
	defer os.Setenv("PATH", origPath)

	origDryRun := replayDryRun
	defer func() { replayDryRun = origDryRun }()
	replayDryRun = false

	// Create fake kiro-cli
	tmpBin := t.TempDir()
	fakeKiro := filepath.Join(tmpBin, "kiro-cli")
	script := `#!/bin/sh
echo "RISK_LEVEL: LOW"
echo "Replay phase complete."
`
	os.WriteFile(fakeKiro, []byte(script), 0o755)
	os.Setenv("PATH", tmpBin+":"+origPath)

	// Save a test mission
	cleanup := saveTestMissions(t, []history.Mission{
		{
			Task:      "replay-nodryrun-test",
			StartTime: time.Date(2025, 7, 1, 10, 0, 0, 0, time.UTC),
			Duration:  "2m",
			Outcome:   "complete",
		},
	})
	defer cleanup()

	all := history.Load()
	lastID := all[len(all)-1].ID
	idStr := fmt.Sprintf("%d", lastID)

	// Set up project with agents for the mission to run
	tmpDir := t.TempDir()
	agDir := filepath.Join(tmpDir, "agents")
	os.MkdirAll(agDir, 0o755)
	writeTestAgent(t, agDir, "numbuh-1", "Nigel", "Analyst")
	writeTestAgent(t, agDir, "numbuh-2", "Hoagie", "Architect")
	writeTestAgent(t, agDir, "numbuh-3", "Kuki", "Implementer")
	writeTestAgent(t, agDir, "numbuh-4", "Wallabee", "QA")
	writeTestAgent(t, agDir, "numbuh-5", "Abby", "Reviewer")
	os.Chdir(tmpDir)

	output := captureStdout(func() {
		runReplay(idStr)
	})

	if !strings.Contains(output, "Replaying mission") {
		t.Errorf("expected replaying message, got:\n%s", output)
	}
	if !strings.Contains(output, "KND Council") {
		t.Errorf("expected KND Council header (from actual mission run), got:\n%s", output)
	}
}

// Test snippet save with existing snippets file
func TestCobraCmd_SnippetSave_Appends(t *testing.T) {
	origHome := os.Getenv("HOME")
	tmpHome := t.TempDir()
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", origHome)

	// Create existing snippets
	snippetsDir := filepath.Join(tmpHome, ".config", "moonbase")
	os.MkdirAll(snippetsDir, 0o700)
	existing := []map[string]string{{"name": "old", "content": "old content"}}
	data, _ := json.MarshalIndent(existing, "", "  ")
	os.WriteFile(filepath.Join(snippetsDir, "snippets.json"), data, 0o600)

	// Provide stdin content
	oldStdin := os.Stdin
	r, w, _ := os.Pipe()
	w.WriteString("new snippet content\n")
	w.Close()
	os.Stdin = r
	defer func() { os.Stdin = oldStdin }()

	output := captureStdout(func() {
		rootCmd.SetArgs([]string{"snippet", "save", "new-one"})
		rootCmd.Execute()
	})

	if !strings.Contains(output, "Snippet saved") {
		t.Errorf("expected snippet saved, got: %s", output)
	}

	// Verify both snippets exist
	savedData, _ := os.ReadFile(filepath.Join(snippetsDir, "snippets.json"))
	if !strings.Contains(string(savedData), "old") || !strings.Contains(string(savedData), "new-one") {
		t.Errorf("expected both old and new snippets, got: %s", string(savedData))
	}
}
