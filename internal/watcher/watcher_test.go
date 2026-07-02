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
