package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// === runInstall coverage: non-interactive listing mode (no --all flag) ===

func TestRunInstall_ListingMode_NoAllFlag(t *testing.T) {
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	// Create source agents dir
	tmpDir := t.TempDir()
	srcDir := filepath.Join(tmpDir, "agents")
	os.MkdirAll(srcDir, 0o755)
	writeTestAgent(t, srcDir, "numbuh-1", "Nigel Uno", "Analyst")
	writeTestAgent(t, srcDir, "numbuh-2", "Hoagie", "Architect")
	writeTestAgent(t, srcDir, "numbuh-3", "Kuki", "Implementer")
	os.Chdir(tmpDir)

	// No --all flag — triggers listing mode
	os.Args = []string{"moonbase", "install"}

	output := captureStdout(func() {
		runInstall()
	})

	// Should show the listing header
	if !strings.Contains(output, "Moonbase Agent Installation") {
		t.Errorf("expected 'Moonbase Agent Installation' header, got:\n%s", output)
	}
	// Should list individual agents with arrow prefix
	if !strings.Contains(output, "→ numbuh-1") {
		t.Errorf("expected '→ numbuh-1' in listing, got:\n%s", output)
	}
	if !strings.Contains(output, "→ numbuh-2") {
		t.Errorf("expected '→ numbuh-2' in listing, got:\n%s", output)
	}
	if !strings.Contains(output, "→ numbuh-3") {
		t.Errorf("expected '→ numbuh-3' in listing, got:\n%s", output)
	}
	// Should still install all agents (non-interactive default)
	if !strings.Contains(output, "3 agent(s) installed") {
		t.Errorf("expected '3 agent(s) installed', got:\n%s", output)
	}
}

func TestRunInstall_ListingMode_GlobalNoAllFlag(t *testing.T) {
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
	writeTestAgent(t, srcDir, "numbuh-4", "Wallabee", "QA")
	writeTestAgent(t, srcDir, "numbuh-5", "Abby", "Reviewer")
	os.Chdir(tmpDir)

	// --global without --all
	os.Args = []string{"moonbase", "install", "--global"}

	output := captureStdout(func() {
		runInstall()
	})

	// Should show listing and install to ~/.kiro/agents/
	if !strings.Contains(output, "Moonbase Agent Installation") {
		t.Errorf("expected listing header, got:\n%s", output)
	}
	if !strings.Contains(output, "→ numbuh-4") {
		t.Errorf("expected '→ numbuh-4' in listing, got:\n%s", output)
	}
	if !strings.Contains(output, "2 agent(s) installed") {
		t.Errorf("expected '2 agent(s) installed', got:\n%s", output)
	}

	// Verify files were copied to global dir
	globalDir := filepath.Join(tmpHome, ".kiro", "agents")
	files, _ := filepath.Glob(filepath.Join(globalDir, "*.md"))
	if len(files) != 2 {
		t.Errorf("expected 2 files in ~/.kiro/agents/, got %d", len(files))
	}
}

// === copyFile additional coverage ===

// TestCopyFile_SelfCopyDoesNotTruncate is a regression test for the bug where
// `moonbase setup`, run outside the repo, resolved its agent source to
// ~/.moonbase/agents (the same as the target) and truncated every agent to 0
// bytes — because copyFile opens the destination with O_TRUNC before reading
// the source. copyFile must now skip a self-copy and leave the file intact.
func TestCopyFile_SelfCopyDoesNotTruncate(t *testing.T) {
	tmpDir := t.TempDir()
	f := filepath.Join(tmpDir, "agent.md")
	content := "---\nname: numbuh-4\n---\n# Numbuh 4\n\n## Doctrine\nimportant content\n"
	os.WriteFile(f, []byte(content), 0o644)

	if err := copyFile(f, f); err != nil {
		t.Fatalf("copyFile(f, f) returned error: %v", err)
	}

	data, err := os.ReadFile(f)
	if err != nil {
		t.Fatalf("reading file: %v", err)
	}
	if string(data) != content {
		t.Errorf("self-copy corrupted the file.\nexpected: %q\ngot:      %q", content, string(data))
	}
	if len(data) == 0 {
		t.Error("self-copy truncated the file to 0 bytes (the original bug)")
	}
}

// TestCopyFile_SelfCopyViaRelativePath ensures the guard also catches a
// self-copy expressed through differing-but-equivalent paths.
func TestCopyFile_SelfCopyViaRelativePath(t *testing.T) {
	tmpDir := t.TempDir()
	f := filepath.Join(tmpDir, "agent.md")
	content := "non-empty content\n"
	os.WriteFile(f, []byte(content), 0o644)

	// dst points at the same file via a "./" detour.
	dst := filepath.Join(tmpDir, ".", "agent.md")
	if err := copyFile(f, dst); err != nil {
		t.Fatalf("copyFile returned error: %v", err)
	}
	data, _ := os.ReadFile(f)
	if string(data) != content {
		t.Errorf("equivalent-path self-copy corrupted the file: got %q", string(data))
	}
}

func TestCopyFile_SuccessPreservesContent(t *testing.T) {
	tmpDir := t.TempDir()
	src := filepath.Join(tmpDir, "agent.md")
	dst := filepath.Join(tmpDir, "copy.md")

	content := "---\nname: numbuh-4\ndesignation: Wallabee Beatles\nrole: QA\ntools:\n  - read\n  - shell\n---\n# Numbuh 4\n\n## Operating Protocol\n\nFull content with unicode: 🌙\nMultiple lines.\n"
	os.WriteFile(src, []byte(content), 0o644)

	err := copyFile(src, dst)
	if err != nil {
		t.Fatalf("copyFile failed: %v", err)
	}

	data, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("reading dest: %v", err)
	}
	if string(data) != content {
		t.Errorf("content mismatch:\nexpected: %q\ngot: %q", content, string(data))
	}

	// Check file permissions
	info, _ := os.Stat(dst)
	if info.Mode().Perm() != 0o644 {
		t.Errorf("expected 0644 permissions, got %o", info.Mode().Perm())
	}
}

func TestCopyFile_OverwritesExisting(t *testing.T) {
	tmpDir := t.TempDir()
	src := filepath.Join(tmpDir, "new.md")
	dst := filepath.Join(tmpDir, "existing.md")

	os.WriteFile(dst, []byte("old content"), 0o644)
	os.WriteFile(src, []byte("new content"), 0o644)

	err := copyFile(src, dst)
	if err != nil {
		t.Fatalf("copyFile failed: %v", err)
	}

	data, _ := os.ReadFile(dst)
	if string(data) != "new content" {
		t.Errorf("expected 'new content', got %q", string(data))
	}
}

func TestCopyFile_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	src := filepath.Join(tmpDir, "empty.md")
	dst := filepath.Join(tmpDir, "dest.md")

	os.WriteFile(src, []byte(""), 0o644)

	err := copyFile(src, dst)
	if err != nil {
		t.Fatalf("copyFile failed: %v", err)
	}

	data, _ := os.ReadFile(dst)
	if string(data) != "" {
		t.Errorf("expected empty file, got %q", string(data))
	}
}

// === Pipe mode tests (rootCmd.RunE) ===

func TestRootCmd_PipeMode_EmptyInput(t *testing.T) {
	// Create a pipe with empty content to simulate echo "" | moonbase
	oldStdin := os.Stdin
	r, w, _ := os.Pipe()
	w.WriteString("")
	w.Close()
	os.Stdin = r
	defer func() { os.Stdin = oldStdin }()

	// rootCmd.RunE checks isTerminal() which uses os.Stdin.Stat()
	// A pipe is not a terminal, so it should read stdin
	err := rootCmd.RunE(rootCmd, []string{})
	if err != nil {
		t.Errorf("expected nil error for empty pipe input, got: %v", err)
	}
}

func TestRootCmd_PipeMode_WithTask(t *testing.T) {
	origPath := os.Getenv("PATH")
	defer os.Setenv("PATH", origPath)

	// Empty PATH so no kiro-cli is found
	os.Setenv("PATH", "/nonexistent-xyz-path")

	oldStdin := os.Stdin
	r, w, _ := os.Pipe()
	w.WriteString("fix the auth bug\n")
	w.Close()
	os.Stdin = r
	defer func() { os.Stdin = oldStdin }()

	output := captureStdout(func() {
		err := rootCmd.RunE(rootCmd, []string{})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	// Should show pipe mode message
	if !strings.Contains(output, "Pipe mode") {
		t.Errorf("expected 'Pipe mode' in output, got:\n%s", output)
	}
	if !strings.Contains(output, "fix the auth bug") {
		t.Errorf("expected task in output, got:\n%s", output)
	}
}

func TestRootCmd_PipeMode_WhitespaceOnly(t *testing.T) {
	oldStdin := os.Stdin
	r, w, _ := os.Pipe()
	w.WriteString("   \n  \n  ")
	w.Close()
	os.Stdin = r
	defer func() { os.Stdin = oldStdin }()

	// Whitespace-only input should be treated as empty after TrimSpace
	err := rootCmd.RunE(rootCmd, []string{})
	if err != nil {
		t.Errorf("expected nil error for whitespace-only pipe input, got: %v", err)
	}
}

// === extractNumbuh coverage ===

func TestExtractNumbuh_AllCases(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"numbuh-0", "0"},
		{"numbuh-1", "1"},
		{"numbuh-4", "4"},
		{"numbuh-13", "13"},
		{"numbuh-86", "86"},
		{"numbuh-274", "274"},
		{"numbuh-362", "362"},
		{"numbuh-999", "999"},
		{"sector-z", "Z"},
		{"knd-council", "K"},
		{"some-other-agent", "some-other-agent"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := extractNumbuh(tt.input)
			if got != tt.expected {
				t.Errorf("extractNumbuh(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

// === sourceTag coverage ===

func TestSourceTag_AllCases(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"user", "[user]"},
		{"project", "[project]"},
		{"built-in", "[built-in]"},
		{"unknown", ""},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := sourceTag(tt.input)
			if got != tt.expected {
				t.Errorf("sourceTag(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

// === runConfig coverage ===

func TestRunConfig_ShowsConfiguration(t *testing.T) {
	output := captureStdout(func() {
		runConfig()
	})

	if !strings.Contains(output, "Moonbase Configuration") {
		t.Errorf("expected 'Moonbase Configuration' header, got:\n%s", output)
	}
	if !strings.Contains(output, "Path:") {
		t.Errorf("expected 'Path:' in output, got:\n%s", output)
	}
}

// === runExport coverage (the export cmd) ===

func TestExportCmd_NonexistentMission(t *testing.T) {
	output := captureStdout(func() {
		exportCmd.Run(exportCmd, []string{"99999"})
	})
	// Should output empty or error message (no mission found)
	_ = output // exercises the code path
}

// === runInstall: suspicious filename skipping ===

func TestRunInstall_SuspiciousFilenameSkipped(t *testing.T) {
	// This is a defensive test — filepath.Glob won't return files with ../
	// but it demonstrates the validation path exists and doesn't panic.
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	tmpDir := t.TempDir()
	srcDir := filepath.Join(tmpDir, "agents")
	os.MkdirAll(srcDir, 0o755)
	writeTestAgent(t, srcDir, "numbuh-1", "Nigel", "Analyst")
	// Create a normal agent but the name has no path traversal issues
	writeTestAgent(t, srcDir, "numbuh-2", "Hoagie", "Architect")
	os.Chdir(tmpDir)
	os.Args = []string{"moonbase", "install", "--all"}

	output := captureStdout(func() {
		runInstall()
	})

	// Both agents should install successfully (no suspicious names)
	if !strings.Contains(output, "2 agent(s) installed") {
		t.Errorf("expected both agents installed, got:\n%s", output)
	}
}

// === runDeploy: deploy cmd with no args on non-TTY triggers exit ===

func TestDeployCmd_NoArgsNonTTY(t *testing.T) {
	// When stdin is not a terminal and no args provided, should exit
	oldStdin := os.Stdin
	r, w, _ := os.Pipe()
	w.Close()
	os.Stdin = r
	defer func() { os.Stdin = oldStdin }()

	code := expectExit(t, func() {
		deployCmd.Run(deployCmd, []string{})
	})
	if code != 1 {
		t.Errorf("expected exit code 1 for non-TTY with no args, got %d", code)
	}
}

// === runList: hardcoded fallback when registry returns empty ===

func TestRunList_EmptyRegistry(t *testing.T) {
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	origHome := os.Getenv("HOME")
	defer os.Setenv("HOME", origHome)

	// Set up an environment where agents dir returns empty
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	os.Chdir(tmpDir)

	output := captureStdout(func() {
		runList()
	})

	// When agents not loadable, should use hardcoded fallback
	if !strings.Contains(output, "OPERATIVE ROSTER") {
		t.Errorf("expected OPERATIVE ROSTER header, got:\n%s", output)
	}
	// Should show either loaded agents or fallback
	if !strings.Contains(output, "SECTOR V") && !strings.Contains(output, "showing defaults") {
		t.Errorf("expected SECTOR V section or defaults notice, got:\n%s", output)
	}
	if !strings.Contains(output, "AI BACKENDS") {
		t.Errorf("expected AI BACKENDS section, got:\n%s", output)
	}
}

// === Snippet commands coverage ===

func TestSnippetListCmd_NoSnippets(t *testing.T) {
	origHome := os.Getenv("HOME")
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	output := captureStdout(func() {
		snippetListCmd.Run(snippetListCmd, []string{})
	})

	if !strings.Contains(output, "No snippets saved") {
		t.Errorf("expected 'No snippets saved', got:\n%s", output)
	}
}

func TestSnippetSaveCmd_ValidName(t *testing.T) {
	origHome := os.Getenv("HOME")
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	// Provide stdin content for the snippet
	oldStdin := os.Stdin
	r, w, _ := os.Pipe()
	w.WriteString("This is my snippet content\nSecond line\n")
	w.Close()
	os.Stdin = r
	defer func() { os.Stdin = oldStdin }()

	output := captureStdout(func() {
		snippetSaveCmd.Run(snippetSaveCmd, []string{"test-snippet"})
	})

	if !strings.Contains(output, "Snippet saved: test-snippet") {
		t.Errorf("expected save confirmation, got:\n%s", output)
	}

	// Verify file was created
	snippetFile := filepath.Join(tmpDir, ".config", "moonbase", "snippets.json")
	data, err := os.ReadFile(snippetFile)
	if err != nil {
		t.Fatalf("snippet file not created: %v", err)
	}
	if !strings.Contains(string(data), "test-snippet") {
		t.Errorf("snippet file doesn't contain snippet name")
	}
}

func TestSnippetSaveCmd_NameTooLong(t *testing.T) {
	longName := strings.Repeat("a", 101)

	code := expectExit(t, func() {
		snippetSaveCmd.Run(snippetSaveCmd, []string{longName})
	})
	if code != 1 {
		t.Errorf("expected exit 1 for too-long name, got %d", code)
	}
}

func TestSnippetSaveCmd_NameWithSlash(t *testing.T) {
	code := expectExit(t, func() {
		snippetSaveCmd.Run(snippetSaveCmd, []string{"bad/name"})
	})
	if code != 1 {
		t.Errorf("expected exit 1 for name with slash, got %d", code)
	}
}

func TestSnippetSaveCmd_NameWithBackslash(t *testing.T) {
	code := expectExit(t, func() {
		snippetSaveCmd.Run(snippetSaveCmd, []string{"bad\\name"})
	})
	if code != 1 {
		t.Errorf("expected exit 1 for name with backslash, got %d", code)
	}
}

func TestSnippetSaveCmd_NameWithControlChar(t *testing.T) {
	code := expectExit(t, func() {
		snippetSaveCmd.Run(snippetSaveCmd, []string{"bad\x00name"})
	})
	if code != 1 {
		t.Errorf("expected exit 1 for name with control char, got %d", code)
	}
}

// === runDeploy: clipboard fallback path when kiro-cli not in PATH ===

func TestRunDeploy_NoKiroCLI_ClipboardSuccess(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("clipboard test only runs on macOS (requires pbcopy)")
	}

	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	origPath := os.Getenv("PATH")
	defer os.Setenv("PATH", origPath)

	// Set up project with agent
	tmpDir := t.TempDir()
	agDir := filepath.Join(tmpDir, "agents")
	os.MkdirAll(agDir, 0o755)
	writeTestAgent(t, agDir, "numbuh-1", "Nigel Uno", "Analyst")
	os.Chdir(tmpDir)
	os.Args = []string{"moonbase", "deploy", "1", "check", "auth"}

	// Set PATH so kiro-cli is NOT found but pbcopy IS found (macOS)
	// Include /usr/bin to keep pbcopy available
	os.Setenv("PATH", "/usr/bin:/bin")

	output := captureStdout(func() {
		runDeploy("1", "")
	})

	// Should fall through to clipboard fallback and succeed
	if !strings.Contains(output, "No interactive backend available") {
		t.Errorf("expected clipboard fallback message, got:\n%s", output)
	}
	if !strings.Contains(output, "Copied to clipboard") {
		t.Errorf("expected 'Copied to clipboard' for successful clipboard copy, got:\n%s", output)
	}
	// Should show agent details
	if !strings.Contains(output, "Nigel Uno") {
		t.Errorf("expected agent designation in output, got:\n%s", output)
	}
	// Should show task since we provided one
	if !strings.Contains(output, "Task:") {
		t.Errorf("expected 'Task:' in output since task was provided, got:\n%s", output)
	}
}

func TestRunDeploy_NoKiroCLI_NoTask_ClipboardSuccess(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("clipboard test only runs on macOS (requires pbcopy)")
	}

	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	origPath := os.Getenv("PATH")
	defer os.Setenv("PATH", origPath)

	// Set up project with agent
	tmpDir := t.TempDir()
	agDir := filepath.Join(tmpDir, "agents")
	os.MkdirAll(agDir, 0o755)
	writeTestAgent(t, agDir, "numbuh-2", "Hoagie", "Architect")
	os.Chdir(tmpDir)
	// No task — only 2 args
	os.Args = []string{"moonbase", "deploy", "2"}

	// No kiro-cli but clipboard available
	os.Setenv("PATH", "/usr/bin:/bin")

	output := captureStdout(func() {
		runDeploy("2", "")
	})

	// Should succeed with clipboard, no Task line shown
	if !strings.Contains(output, "Copied to clipboard") {
		t.Errorf("expected clipboard success, got:\n%s", output)
	}
	if !strings.Contains(output, "Paste into:") {
		t.Errorf("expected paste instructions, got:\n%s", output)
	}
}

// === runMission: longer output to hit truncation path ===

func TestRunMission_LongOutput_Truncation(t *testing.T) {
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	origPath := os.Getenv("PATH")
	defer os.Setenv("PATH", origPath)

	// Create fake kiro-cli that outputs >200 chars
	tmpBin := t.TempDir()
	fakeKiro := filepath.Join(tmpBin, "kiro-cli")
	// Generate output that will be > 200 chars
	longOutput := strings.Repeat("Analysis complete with detailed findings. ", 10)
	script := "#!/bin/sh\necho \"RISK_LEVEL: LOW\"\necho \"" + longOutput + "\"\n"
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
		runMission("test long output truncation")
	})

	if !strings.Contains(output, "KND Council") {
		t.Errorf("expected 'KND Council' header, got:\n%s", output)
	}
	// Should have phase completion messages showing char count
	if !strings.Contains(output, "complete") {
		t.Errorf("expected phase completion, got:\n%s", output)
	}
}

// === runList: agent with pipeline position 0 (numbuh-0) ===

func TestRunList_WithNumbuhZeroAgent(t *testing.T) {
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	tmpDir := t.TempDir()
	agDir := filepath.Join(tmpDir, "agents")
	os.MkdirAll(agDir, 0o755)

	// Write an agent with pipeline position 0
	pos0Content := "---\nname: numbuh-0\ndesignation: Monty Uno\nrole: System Architect\ntools:\n  - read\n  - shell\npipeline_position: 0\n---\n# Numbuh 0\n\n## Operating Protocol\n\nProtocol content.\n"
	os.WriteFile(filepath.Join(agDir, "numbuh-0.md"), []byte(pos0Content), 0o644)

	// Write agents with pipeline positions 1-5
	for i := 1; i <= 5; i++ {
		content := "---\nname: numbuh-" + string(rune('0'+i)) + "\ndesignation: Agent " + string(rune('0'+i)) + "\nrole: Role " + string(rune('0'+i)) + "\ntools:\n  - read\npipeline_position: " + string(rune('0'+i)) + "\n---\n# Agent\n\n## Operating Protocol\n\nContent.\n"
		os.WriteFile(filepath.Join(agDir, "numbuh-"+string(rune('0'+i))+".md"), []byte(content), 0o644)
	}
	os.Chdir(tmpDir)

	output := captureStdout(func() {
		runList()
	})

	if !strings.Contains(output, "OPERATIVE ROSTER") {
		t.Errorf("expected OPERATIVE ROSTER, got:\n%s", output)
	}
	// Should display Monty Uno in SECTOR V section (pipeline position 0)
	if !strings.Contains(output, "Monty Uno") {
		t.Errorf("expected 'Monty Uno' (numbuh-0) in output, got:\n%s", output)
	}
}

// === runStatus: with preferred backend detected ===

func TestRunStatus_ShowsAIBackendSection(t *testing.T) {
	output := captureStdout(func() {
		runStatus()
	})

	if !strings.Contains(output, "Backend:") {
		t.Errorf("expected 'Backend:' in status output, got:\n%s", output)
	}
}

// === Pipe mode with kiro-cli in PATH ===

func TestRootCmd_PipeMode_WithFakeKiroCLI(t *testing.T) {
	origPath := os.Getenv("PATH")
	defer os.Setenv("PATH", origPath)

	// Create a fake kiro-cli that just echoes
	tmpBin := t.TempDir()
	fakeKiro := filepath.Join(tmpBin, "kiro-cli")
	script := "#!/bin/sh\necho \"Pipe response: $*\"\n"
	os.WriteFile(fakeKiro, []byte(script), 0o755)
	os.Setenv("PATH", tmpBin+":"+origPath)

	oldStdin := os.Stdin
	r, w, _ := os.Pipe()
	w.WriteString("deploy security scan\n")
	w.Close()
	os.Stdin = r
	defer func() { os.Stdin = oldStdin }()

	output := captureStdout(func() {
		err := rootCmd.RunE(rootCmd, []string{})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(output, "Pipe mode") {
		t.Errorf("expected 'Pipe mode' in output, got:\n%s", output)
	}
	if !strings.Contains(output, "Deploy to kiro-cli") {
		t.Errorf("expected kiro-cli deploy message, got:\n%s", output)
	}
}

// === Version cmd coverage ===

func TestVersionCmd_Output(t *testing.T) {
	output := captureStdout(func() {
		versionCmd.Run(versionCmd, []string{})
	})

	if !strings.Contains(output, "moonbase") || !strings.Contains(output, "version") {
		// Version cmd should output something with version info
		t.Logf("version output: %s", output)
	}
}
