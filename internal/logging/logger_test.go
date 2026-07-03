package logging

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInit_Debug_CreatesLogFile(t *testing.T) {
	tmpDir := t.TempDir()
	// Override LogDir for testing
	oldLogDir := LogDir
	LogDir = tmpDir
	defer func() { LogDir = oldLogDir }()

	Init(true)

	logPath := filepath.Join(tmpDir, "debug.log")
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		t.Fatal("expected debug.log to be created when debug=true")
	}

	// Logger should be non-nil
	if Logger == nil {
		t.Fatal("Logger should not be nil after Init(true)")
	}
}

func TestInit_NoDebug_NoFileCreated(t *testing.T) {
	tmpDir := t.TempDir()
	oldLogDir := LogDir
	LogDir = tmpDir
	defer func() { LogDir = oldLogDir }()

	Init(false)

	logPath := filepath.Join(tmpDir, "debug.log")
	if _, err := os.Stat(logPath); err == nil {
		t.Fatal("expected no debug.log when debug=false")
	}

	// Logger should still be non-nil (discard handler)
	if Logger == nil {
		t.Fatal("Logger should not be nil after Init(false)")
	}
}

func TestInit_Debug_LoggerWritesJSON(t *testing.T) {
	tmpDir := t.TempDir()
	oldLogDir := LogDir
	LogDir = tmpDir
	defer func() { LogDir = oldLogDir }()

	Init(true)

	// Write a log entry
	Logger.Info("test message", "key", "value")

	// Read the log file
	logPath := filepath.Join(tmpDir, "debug.log")
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	content := string(data)
	if content == "" {
		t.Fatal("log file is empty after writing")
	}

	// Parse as JSON to verify format
	var entry map[string]interface{}
	// The file may have trailing newline, split by lines
	lines := strings.Split(strings.TrimSpace(content), "\n")
	if len(lines) == 0 {
		t.Fatal("no log lines found")
	}

	if err := json.Unmarshal([]byte(lines[0]), &entry); err != nil {
		t.Fatalf("log entry is not valid JSON: %v\nContent: %s", err, lines[0])
	}

	// Verify fields
	if msg, ok := entry["msg"]; !ok || msg != "test message" {
		t.Errorf("expected msg='test message', got %v", entry["msg"])
	}
	if val, ok := entry["key"]; !ok || val != "value" {
		t.Errorf("expected key='value', got %v", entry["key"])
	}
}

func TestInit_Debug_LoggerWritesDebugLevel(t *testing.T) {
	tmpDir := t.TempDir()
	oldLogDir := LogDir
	LogDir = tmpDir
	defer func() { LogDir = oldLogDir }()

	Init(true)

	// Write a debug-level entry
	Logger.Debug("debug message", "debug_key", "debug_value")

	logPath := filepath.Join(tmpDir, "debug.log")
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "debug message") {
		t.Error("debug level messages should be written when debug=true")
	}
	if !strings.Contains(content, "debug_key") {
		t.Error("debug key should be present in log output")
	}
}

func TestInit_NoDebug_LoggerDiscards(t *testing.T) {
	tmpDir := t.TempDir()
	oldLogDir := LogDir
	LogDir = tmpDir
	defer func() { LogDir = oldLogDir }()

	Init(false)

	// Write a log entry — should not create any file
	Logger.Info("this should be discarded")

	logPath := filepath.Join(tmpDir, "debug.log")
	if _, err := os.Stat(logPath); err == nil {
		t.Fatal("no log file should be created when debug=false")
	}
}

func TestInit_Debug_AppendsToExistingFile(t *testing.T) {
	tmpDir := t.TempDir()
	oldLogDir := LogDir
	LogDir = tmpDir
	defer func() { LogDir = oldLogDir }()

	// Create the log file first with some content
	logPath := filepath.Join(tmpDir, "debug.log")
	os.WriteFile(logPath, []byte(`{"msg":"existing"}`+"\n"), 0600)

	Init(true)
	Logger.Info("new entry")

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "existing") {
		t.Error("existing content should be preserved (append mode)")
	}
	if !strings.Contains(content, "new entry") {
		t.Error("new entry should be appended")
	}
}

func TestInit_Debug_CreatesDirectoryIfNeeded(t *testing.T) {
	tmpDir := t.TempDir()
	nestedDir := filepath.Join(tmpDir, "nested", "deep", "dir")
	oldLogDir := LogDir
	LogDir = nestedDir
	defer func() { LogDir = oldLogDir }()

	Init(true)

	// Directory should be created
	if _, err := os.Stat(nestedDir); os.IsNotExist(err) {
		t.Fatal("expected nested directory to be created")
	}

	// Log file should exist
	logPath := filepath.Join(nestedDir, "debug.log")
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		t.Fatal("expected debug.log to be created in nested dir")
	}
}

func TestHomeDir_ReturnsNonEmpty(t *testing.T) {
	home := homeDir()
	if home == "" {
		t.Error("homeDir() returned empty string")
	}
}

func TestLogDir_UsesOverride(t *testing.T) {
	oldLogDir := LogDir
	defer func() { LogDir = oldLogDir }()

	LogDir = "/custom/path"
	if got := logDir(); got != "/custom/path" {
		t.Errorf("logDir() = %q, want /custom/path", got)
	}
}

func TestLogDir_UsesDefault(t *testing.T) {
	oldLogDir := LogDir
	defer func() { LogDir = oldLogDir }()

	LogDir = ""
	got := logDir()
	if !strings.Contains(got, ".config/moonbase") {
		t.Errorf("logDir() = %q, expected to contain .config/moonbase", got)
	}
}

func TestInit_MultipleCallsOverwriteLogger(t *testing.T) {
	tmpDir := t.TempDir()
	oldLogDir := LogDir
	LogDir = tmpDir
	defer func() { LogDir = oldLogDir }()

	// Init with debug
	Init(true)
	if Logger == nil {
		t.Fatal("Logger should not be nil")
	}

	// Re-init without debug
	Init(false)
	if Logger == nil {
		t.Fatal("Logger should not be nil after re-init")
	}

	// Writing should not create new content (discard handler now)
	// Note: the file still exists from the first Init
	logPath := filepath.Join(tmpDir, "debug.log")
	infoBefore, _ := os.Stat(logPath)
	sizeBefore := infoBefore.Size()

	Logger.Info("should be discarded")

	infoAfter, _ := os.Stat(logPath)
	sizeAfter := infoAfter.Size()

	if sizeAfter != sizeBefore {
		t.Error("writing after Init(false) should not add to the file")
	}
}

func TestInit_Debug_MkdirAllFails(t *testing.T) {
	// Use a path that can't be created (file as parent)
	tmpDir := t.TempDir()
	// Create a file where the directory would need to be
	blockingFile := filepath.Join(tmpDir, "blocked")
	os.WriteFile(blockingFile, []byte("x"), 0o600)

	oldLogDir := LogDir
	LogDir = filepath.Join(blockingFile, "subdir") // can't mkdir inside a file
	defer func() { LogDir = oldLogDir }()

	Init(true)

	// Logger should still be initialized (discard fallback)
	if Logger == nil {
		t.Fatal("Logger should not be nil even when MkdirAll fails")
	}

	// Verify no log file was created
	logPath := filepath.Join(blockingFile, "subdir", "debug.log")
	if _, err := os.Stat(logPath); err == nil {
		t.Error("log file should not exist when mkdir fails")
	}
}

func TestInit_Debug_OpenFileFails(t *testing.T) {
	tmpDir := t.TempDir()
	oldLogDir := LogDir
	LogDir = tmpDir
	defer func() { LogDir = oldLogDir }()

	// Create debug.log as a directory so OpenFile will fail
	logPath := filepath.Join(tmpDir, "debug.log")
	os.MkdirAll(logPath, 0o700) // it's a dir, not a file

	Init(true)

	// Logger should still be initialized (discard fallback)
	if Logger == nil {
		t.Fatal("Logger should not be nil even when OpenFile fails")
	}
}
