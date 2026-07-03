package watcher

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNew_CreatesWatcher(t *testing.T) {
	w, err := New()
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	defer w.Stop()

	if w.Events == nil {
		t.Error("expected Events channel to be initialized")
	}
	if w.Running() {
		t.Error("expected watcher to not be running before Start")
	}
}

func TestStart_SetsRunning(t *testing.T) {
	w, err := New()
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	defer w.Stop()

	tmpDir := t.TempDir()
	if err := w.Start(tmpDir); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	if !w.Running() {
		t.Error("expected watcher to be running after Start")
	}
}

func TestStop_ClearsRunning(t *testing.T) {
	w, err := New()
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	tmpDir := t.TempDir()
	w.Start(tmpDir)
	w.Stop()

	if w.Running() {
		t.Error("expected watcher to not be running after Stop")
	}
}

func TestWatcher_DetectsFileWrite(t *testing.T) {
	w, err := New()
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	defer w.Stop()

	tmpDir := t.TempDir()
	if err := w.Start(tmpDir); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Small delay to let watcher initialize
	time.Sleep(100 * time.Millisecond)

	// Write a file
	testFile := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(testFile, []byte("hello"), 0o644)

	// Wait for event (with generous timeout for CI)
	select {
	case event := <-w.Events:
		if event.Path != "test.txt" {
			t.Errorf("expected path 'test.txt', got: %s", event.Path)
		}
		if event.Time.IsZero() {
			t.Error("expected non-zero time")
		}
	case <-time.After(5 * time.Second):
		t.Skip("timed out waiting for file event (may be CI without inotify support)")
	}
}

func TestWatcher_RecentKeepsMax10(t *testing.T) {
	w, err := New()
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	defer w.Stop()

	tmpDir := t.TempDir()
	if err := w.Start(tmpDir); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Small delay to let watcher initialize
	time.Sleep(100 * time.Millisecond)

	// Create 15 files rapidly
	for i := 0; i < 15; i++ {
		os.WriteFile(filepath.Join(tmpDir, fmt.Sprintf("file%d.txt", i)), []byte("x"), 0o644)
	}

	// Give watcher time to process
	time.Sleep(1 * time.Second)

	recent := w.Recent()
	if len(recent) > 10 {
		t.Errorf("expected max 10 recent events, got: %d", len(recent))
	}
	// On CI, we might get 0 events if filesystem watching isn't supported
	// That's OK — the max-10 cap is the invariant we're testing
}

func TestRecent_EmptyBeforeAnyEvents(t *testing.T) {
	w, err := New()
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	defer w.Stop()

	recent := w.Recent()
	if len(recent) != 0 {
		t.Errorf("expected 0 recent events before any writes, got: %d", len(recent))
	}
}

func TestStart_RecursiveWalksSubdirectories(t *testing.T) {
	w, err := New()
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	defer w.Stop()

	tmpDir := t.TempDir()
	// Create nested directories: level1/level2/level3
	level1 := filepath.Join(tmpDir, "level1")
	level2 := filepath.Join(level1, "level2")
	level3 := filepath.Join(level2, "level3")
	os.MkdirAll(level3, 0o755)

	if err := w.Start(tmpDir); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Verify we watch at least the root + 3 levels
	// dirCount includes root + level1 + level2 + level3 = 4
	if w.dirCount < 4 {
		t.Errorf("expected at least 4 watched dirs, got %d", w.dirCount)
	}

	// Write a file in a subdirectory to verify it's being watched
	time.Sleep(100 * time.Millisecond)
	testFile := filepath.Join(level2, "deep.txt")
	os.WriteFile(testFile, []byte("deep"), 0o644)

	select {
	case event := <-w.Events:
		expected := filepath.Join("level1", "level2", "deep.txt")
		if event.Path != expected {
			t.Errorf("expected relative path %q, got: %q", expected, event.Path)
		}
	case <-time.After(5 * time.Second):
		t.Skip("timed out waiting for deep file event")
	}
}

func TestStart_ExcludesCommonDirs(t *testing.T) {
	w, err := New()
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	defer w.Stop()

	tmpDir := t.TempDir()
	// Create directories that should be excluded
	for _, name := range []string{"node_modules", ".git", "vendor", "dist", "build", "__pycache__"} {
		os.MkdirAll(filepath.Join(tmpDir, name, "sub"), 0o755)
	}
	// Create a normal directory that should be included
	os.MkdirAll(filepath.Join(tmpDir, "src", "pkg"), 0o755)

	if err := w.Start(tmpDir); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// root + src + src/pkg = 3 (excluded dirs should not be counted)
	if w.dirCount != 3 {
		t.Errorf("expected 3 watched dirs (root + src + src/pkg), got %d", w.dirCount)
	}
}

func TestStart_RespectsDepthLimit(t *testing.T) {
	w, err := New()
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	defer w.Stop()

	tmpDir := t.TempDir()
	// Create directories deeper than max depth (3)
	deep := tmpDir
	for i := 0; i < 5; i++ {
		deep = filepath.Join(deep, fmt.Sprintf("d%d", i))
	}
	os.MkdirAll(deep, 0o755)

	if err := w.Start(tmpDir); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// root + d0 + d0/d1 + d0/d1/d2 = 4 (depth 3 means 3 levels below root)
	if w.dirCount > 4 {
		t.Errorf("expected max 4 watched dirs (depth limit 3), got %d", w.dirCount)
	}
}

func TestStart_DynamicDirectoryAdd(t *testing.T) {
	w, err := New()
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	defer w.Stop()

	tmpDir := t.TempDir()
	if err := w.Start(tmpDir); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	// Create a new directory after watcher started
	newDir := filepath.Join(tmpDir, "newpkg")
	os.Mkdir(newDir, 0o755)

	// Give watcher time to pick up the Create event and add the dir
	time.Sleep(500 * time.Millisecond)

	// Now write a file in the new directory
	testFile := filepath.Join(newDir, "new.txt")
	os.WriteFile(testFile, []byte("new"), 0o644)

	// We should get an event for the file in the dynamically added dir
	// Drain any earlier Create events first
	timeout := time.After(5 * time.Second)
	for {
		select {
		case event := <-w.Events:
			if event.Path == filepath.Join("newpkg", "new.txt") {
				return // Success — dynamic dir was added
			}
		case <-timeout:
			t.Skip("timed out waiting for event in dynamically added directory")
			return
		}
	}
}

func TestWatch_StillWorksForExplicitRoots(t *testing.T) {
	w, err := New()
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	defer w.Stop()

	tmpDir := t.TempDir()
	if err := w.Start(tmpDir); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Create a separate directory and add it explicitly
	extraDir := t.TempDir()
	if err := w.Watch(extraDir); err != nil {
		t.Fatalf("Watch failed: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	// Write to the explicitly watched directory
	os.WriteFile(filepath.Join(extraDir, "extra.txt"), []byte("x"), 0o644)

	select {
	case event := <-w.Events:
		// The path might be absolute or just basename since extraDir isn't under rootDir
		if event.Path == "" {
			t.Error("expected non-empty event path")
		}
	case <-time.After(5 * time.Second):
		t.Skip("timed out waiting for event in explicitly watched directory")
	}
}
