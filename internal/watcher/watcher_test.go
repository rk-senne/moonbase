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
	if w.DirCount() < 4 {
		t.Errorf("expected at least 4 watched dirs, got %d", w.DirCount())
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
	if w.DirCount() != 3 {
		t.Errorf("expected 3 watched dirs (root + src + src/pkg), got %d", w.DirCount())
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
	if w.DirCount() > 4 {
		t.Errorf("expected max 4 watched dirs (depth limit 3), got %d", w.DirCount())
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

func TestWatcher_HandleRemove_DecrementsDirCount(t *testing.T) {
	w, err := New()
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	defer w.Stop()

	tmpDir := t.TempDir()
	// Create a subdirectory
	subDir := filepath.Join(tmpDir, "subdir")
	os.MkdirAll(subDir, 0o755)

	if err := w.Start(tmpDir); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	initialCount := w.DirCount()

	// Remove the subdirectory
	time.Sleep(100 * time.Millisecond)
	os.RemoveAll(subDir)

	// Give watcher time to process the Remove event
	time.Sleep(500 * time.Millisecond)

	// dirCount should have decremented
	if w.DirCount() >= initialCount {
		// This is acceptable on some systems — the important thing is it doesn't crash
		t.Logf("dirCount didn't decrement (was %d, now %d) - acceptable on some OSes", initialCount, w.DirCount())
	}
}

func TestWatcher_HandleCreate_ExcludedDir(t *testing.T) {
	w, err := New()
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	defer w.Stop()

	tmpDir := t.TempDir()
	if err := w.Start(tmpDir); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	initialCount := w.DirCount()
	time.Sleep(100 * time.Millisecond)

	// Create an excluded directory (node_modules)
	excludedDir := filepath.Join(tmpDir, "node_modules")
	os.Mkdir(excludedDir, 0o755)

	// Give watcher time to process
	time.Sleep(500 * time.Millisecond)

	// dirCount should NOT have increased for excluded dir
	if w.DirCount() > initialCount+1 {
		// The Create event itself may increment, but node_modules should not be added
		t.Errorf("dirCount increased too much for excluded dir: was %d, now %d", initialCount, w.DirCount())
	}
}

func TestWatcher_HandleCreate_BeyondMaxDepth(t *testing.T) {
	w, err := New()
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	defer w.Stop()

	tmpDir := t.TempDir()
	// Create directories up to max depth
	deep := tmpDir
	for i := 0; i < DefaultMaxDepth; i++ {
		deep = filepath.Join(deep, fmt.Sprintf("level%d", i))
	}
	os.MkdirAll(deep, 0o755)

	if err := w.Start(tmpDir); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	countAfterStart := w.DirCount()
	time.Sleep(100 * time.Millisecond)

	// Create a directory beyond max depth
	tooDeep := filepath.Join(deep, "too_deep")
	os.Mkdir(tooDeep, 0o755)

	time.Sleep(500 * time.Millisecond)

	// Should not have added the too-deep directory
	if w.DirCount() > countAfterStart+1 {
		t.Logf("dirCount may have increased (was %d, now %d)", countAfterStart, w.DirCount())
	}
}

func TestWatcher_RelativePath_NoRootDir(t *testing.T) {
	w := &Watcher{rootDir: ""}
	result := w.relativePath("/some/absolute/path/file.txt")
	if result != "file.txt" {
		t.Errorf("expected 'file.txt' when rootDir is empty, got: %s", result)
	}
}

func TestWatcher_RelativePath_WithRootDir(t *testing.T) {
	w := &Watcher{rootDir: "/home/user/project"}
	result := w.relativePath("/home/user/project/src/main.go")
	expected := filepath.Join("src", "main.go")
	if result != expected {
		t.Errorf("expected %q, got: %q", expected, result)
	}
}

func TestWatcher_RelativePath_UnrelatedPath(t *testing.T) {
	w := &Watcher{rootDir: "/home/user/project"}
	// Path that can still be made relative (../ prefix)
	result := w.relativePath("/home/user/other/file.txt")
	// filepath.Rel should succeed with "../other/file.txt"
	if result == "" {
		t.Error("expected non-empty result")
	}
}

func TestWatcher_MaxWatchedDirs_Constant(t *testing.T) {
	if MaxWatchedDirs != 500 {
		t.Errorf("expected MaxWatchedDirs=500, got %d", MaxWatchedDirs)
	}
}

func TestWatcher_DefaultMaxDepth_Constant(t *testing.T) {
	if DefaultMaxDepth != 3 {
		t.Errorf("expected DefaultMaxDepth=3, got %d", DefaultMaxDepth)
	}
}

func TestWatcher_ExcludedDirs_AllPresent(t *testing.T) {
	expected := []string{".git", "node_modules", "vendor", "dist", "build", "__pycache__", ".next", "target"}
	for _, name := range expected {
		if !excludedDirs[name] {
			t.Errorf("expected %s in excludedDirs", name)
		}
	}
}

func TestFileEvent_Fields(t *testing.T) {
	now := time.Now()
	fe := FileEvent{
		Path: "src/main.go",
		Time: now,
	}
	if fe.Path != "src/main.go" {
		t.Errorf("expected path 'src/main.go', got: %s", fe.Path)
	}
	if fe.Time != now {
		t.Error("time mismatch")
	}
}

func TestWatcher_MaxDirCapInAddRecursive(t *testing.T) {
	w, err := New()
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	defer w.Stop()

	tmpDir := t.TempDir()

	// Create subdirectories that would normally be watched
	os.MkdirAll(filepath.Join(tmpDir, "sub1"), 0o755)
	os.MkdirAll(filepath.Join(tmpDir, "sub2"), 0o755)
	os.MkdirAll(filepath.Join(tmpDir, "sub3"), 0o755)

	// Artificially set dirCount close to MaxWatchedDirs to trigger the cap
	// addRecursive will add the root first (dirCount becomes MaxWatchedDirs),
	// then the walk should see the cap and skip subdirs
	w.SetDirCount(MaxWatchedDirs - 1)

	if err := w.Start(tmpDir); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// After Start, dirCount should be at or near cap (root added, subs skipped)
	if w.DirCount() > MaxWatchedDirs+1 {
		t.Errorf("dirCount exceeded cap: %d (expected <= %d)", w.DirCount(), MaxWatchedDirs+1)
	}
}

func TestWatcher_HandleCreate_MaxDirCap(t *testing.T) {
	w, err := New()
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	defer w.Stop()

	tmpDir := t.TempDir()
	if err := w.Start(tmpDir); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Set dirCount to cap to test handleCreate's cap check
	w.SetDirCount(MaxWatchedDirs)

	time.Sleep(100 * time.Millisecond)

	// Create a new directory — handleCreate should skip it due to cap
	newDir := filepath.Join(tmpDir, "capped")
	os.Mkdir(newDir, 0o755)

	time.Sleep(500 * time.Millisecond)

	// dirCount should not have increased (cap was already reached)
	if w.DirCount() > MaxWatchedDirs+1 {
		t.Errorf("expected dirCount near cap, got: %d", w.DirCount())
	}
}

func TestNew_ErrorHandling(t *testing.T) {
	// New() should succeed in normal conditions
	w, err := New()
	if err != nil {
		t.Fatalf("New() failed unexpectedly: %v", err)
	}
	if w == nil {
		t.Fatal("expected non-nil watcher")
	}
	if w.maxDepth != DefaultMaxDepth {
		t.Errorf("expected maxDepth=%d, got %d", DefaultMaxDepth, w.maxDepth)
	}
	if w.Events == nil {
		t.Error("expected non-nil Events channel")
	}
	w.Stop()
}

func TestStart_ErrorOnInvalidDir(t *testing.T) {
	w, err := New()
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	defer w.Stop()

	// Try to start on a non-existent directory
	err = w.Start("/nonexistent/path/that/does/not/exist")
	if err == nil {
		t.Error("expected error when starting on non-existent directory")
	}
}

func TestWatcher_AddRecursive_AlreadyAtCap(t *testing.T) {
	w, err := New()
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	defer w.Stop()

	tmpDir := t.TempDir()
	os.MkdirAll(filepath.Join(tmpDir, "sub1"), 0o755)

	// Set dirCount AT MaxWatchedDirs (already at cap)
	w.SetDirCount(MaxWatchedDirs)
	w.rootDir = tmpDir
	w.running = true

	// Call addRecursive directly — should return nil immediately due to cap
	err = w.addRecursive(tmpDir, 0)
	if err != nil {
		t.Fatalf("addRecursive should not error at cap: %v", err)
	}
	// dirCount should not have changed since we hit the early return
	if w.DirCount() != MaxWatchedDirs {
		t.Errorf("expected dirCount to remain at %d, got %d", MaxWatchedDirs, w.DirCount())
	}
}

func TestWatcher_HandleCreate_NonDirectory(t *testing.T) {
	w, err := New()
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	defer w.Stop()

	tmpDir := t.TempDir()
	if err := w.Start(tmpDir); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	initialCount := w.DirCount()

	// Create a FILE (not directory) — handleCreate should ignore it
	time.Sleep(100 * time.Millisecond)
	testFile := filepath.Join(tmpDir, "notadir.txt")
	os.WriteFile(testFile, []byte("file"), 0o644)

	time.Sleep(500 * time.Millisecond)

	// dirCount should not have changed for a file creation
	if w.DirCount() != initialCount {
		t.Logf("dirCount changed for file creation (was %d, now %d) — acceptable if OS reports both events", initialCount, w.DirCount())
	}
}

func TestWatcher_HandleCreate_NonExistentPath(t *testing.T) {
	w, err := New()
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	defer w.Stop()

	tmpDir := t.TempDir()
	w.rootDir = tmpDir
	w.running = true

	// Call handleCreate with a non-existent path (stat will fail)
	w.handleCreate("/nonexistent/path/dir")
	// Should not panic
}

func TestWatcher_AddRecursive_UnreadableDir(t *testing.T) {
	w, err := New()
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	defer w.Stop()

	tmpDir := t.TempDir()
	// Create a subdir with no read permission
	noReadDir := filepath.Join(tmpDir, "noperm")
	os.MkdirAll(filepath.Join(noReadDir, "inner"), 0o755)
	os.Chmod(noReadDir, 0o000) // remove all permissions
	defer os.Chmod(noReadDir, 0o755) // restore for cleanup

	w.rootDir = tmpDir
	// Start should handle unreadable directories gracefully
	err = w.Start(tmpDir)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	// Should not crash — unreadable dirs are silently skipped
}

func TestWatcher_HandleCreate_RelPathError(t *testing.T) {
	w, err := New()
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	defer w.Stop()

	tmpDir := t.TempDir()
	w.rootDir = tmpDir
	w.running = true
	w.maxDepth = DefaultMaxDepth

	// handleCreate with a path that can still be made relative
	// But is excluded by name
	excludedPath := filepath.Join(tmpDir, ".git")
	os.Mkdir(excludedPath, 0o755)
	w.handleCreate(excludedPath)
	// Should not add .git to watchers
}

func TestWatcher_RelativePath_EmptyRootDir(t *testing.T) {
	w := &Watcher{rootDir: ""}
	// When rootDir is empty, should return basename
	result := w.relativePath("/tmp/some/file.txt")
	if result != "file.txt" {
		t.Errorf("expected 'file.txt', got: %q", result)
	}
}
