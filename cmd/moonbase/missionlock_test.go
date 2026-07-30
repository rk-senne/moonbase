package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestAcquireMissionLock_NoExistingLock(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	release, err := acquireMissionLock(false)
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if release == nil {
		t.Fatal("expected non-nil release function")
	}

	// Verify lock file was created
	lockPath := filepath.Join(tmpHome, ".moonbase", "mission.lock")
	data, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("expected lock file at %s: %v", lockPath, err)
	}

	var info missionLockInfo
	if err := json.Unmarshal(data, &info); err != nil {
		t.Fatalf("invalid lock file JSON: %v", err)
	}
	if info.PID != os.Getpid() {
		t.Errorf("expected PID %d, got %d", os.Getpid(), info.PID)
	}
	if info.StartedAt.IsZero() {
		t.Error("expected non-zero StartedAt")
	}

	// Clean up
	release()

	// Verify lock file was removed
	if _, err := os.Stat(lockPath); !os.IsNotExist(err) {
		t.Error("expected lock file to be removed after release")
	}
}

func TestAcquireMissionLock_SecondAcquireWithoutForce_Fails(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	release, err := acquireMissionLock(false)
	if err != nil {
		t.Fatalf("first acquire failed: %v", err)
	}
	defer release()

	// Second acquire without force should fail (same PID is alive)
	_, err = acquireMissionLock(false)
	if err == nil {
		t.Fatal("expected error on second acquire without force")
	}
	if got := err.Error(); got == "" {
		t.Fatal("expected non-empty error message")
	}
	// Verify error message contains expected information
	errMsg := err.Error()
	if !contains(errMsg, "mission already in progress") {
		t.Errorf("error should mention 'mission already in progress', got: %s", errMsg)
	}
	if !contains(errMsg, "--force") {
		t.Errorf("error should mention '--force', got: %s", errMsg)
	}
}

func TestAcquireMissionLock_SecondAcquireWithForce_Succeeds(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	release1, err := acquireMissionLock(false)
	if err != nil {
		t.Fatalf("first acquire failed: %v", err)
	}
	defer release1()

	// Second acquire with force should succeed
	release2, err := acquireMissionLock(true)
	if err != nil {
		t.Fatalf("second acquire with force failed: %v", err)
	}
	defer release2()

	// Verify the lock file has the current PID
	lockPath := filepath.Join(tmpHome, ".moonbase", "mission.lock")
	data, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("failed to read lock file: %v", err)
	}
	var info missionLockInfo
	if err := json.Unmarshal(data, &info); err != nil {
		t.Fatalf("invalid lock file JSON: %v", err)
	}
	if info.PID != os.Getpid() {
		t.Errorf("expected current PID %d, got %d", os.Getpid(), info.PID)
	}
}

func TestAcquireMissionLock_StaleLock_TakenOver(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	// Write a lock file with a very high PID that's almost certainly not alive
	lockDir := filepath.Join(tmpHome, ".moonbase")
	if err := os.MkdirAll(lockDir, 0o700); err != nil {
		t.Fatal(err)
	}

	staleLock := missionLockInfo{
		PID:       999999,
		StartedAt: time.Now().Add(-10 * time.Minute).UTC(),
	}
	data, _ := json.Marshal(staleLock)
	lockPath := filepath.Join(lockDir, "mission.lock")
	if err := os.WriteFile(lockPath, data, 0o600); err != nil {
		t.Fatal(err)
	}

	// Acquire without force — should succeed because PID 999999 is dead
	release, err := acquireMissionLock(false)
	if err != nil {
		t.Fatalf("expected stale lock to be taken over, got error: %v", err)
	}
	defer release()

	// Verify lock now has our PID
	data, err = os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("failed to read lock file: %v", err)
	}
	var info missionLockInfo
	if err := json.Unmarshal(data, &info); err != nil {
		t.Fatalf("invalid lock file JSON: %v", err)
	}
	if info.PID != os.Getpid() {
		t.Errorf("expected current PID %d after taking over stale lock, got %d", os.Getpid(), info.PID)
	}
}

func TestAcquireMissionLock_Release_RemovesFile(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	release, err := acquireMissionLock(false)
	if err != nil {
		t.Fatalf("acquire failed: %v", err)
	}

	lockPath := filepath.Join(tmpHome, ".moonbase", "mission.lock")

	// Verify it exists
	if _, err := os.Stat(lockPath); err != nil {
		t.Fatal("expected lock file to exist before release")
	}

	// Release
	release()

	// Verify removal
	if _, err := os.Stat(lockPath); !os.IsNotExist(err) {
		t.Error("expected lock file to be removed after release()")
	}
}

func TestAcquireMissionLock_CorruptedLockFile_TakenOver(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	// Write a corrupted lock file (invalid JSON)
	lockDir := filepath.Join(tmpHome, ".moonbase")
	if err := os.MkdirAll(lockDir, 0o700); err != nil {
		t.Fatal(err)
	}
	lockPath := filepath.Join(lockDir, "mission.lock")
	if err := os.WriteFile(lockPath, []byte("not valid json{{{"), 0o600); err != nil {
		t.Fatal(err)
	}

	// Should succeed — corrupted lock is treated as stale
	release, err := acquireMissionLock(false)
	if err != nil {
		t.Fatalf("expected corrupted lock to be taken over, got error: %v", err)
	}
	defer release()
}

func TestIsProcessAlive_CurrentPID(t *testing.T) {
	if !isProcessAlive(os.Getpid()) {
		t.Error("expected current process to be alive")
	}
}

func TestIsProcessAlive_DeadPID(t *testing.T) {
	// PID 999999 is almost certainly not running
	if isProcessAlive(999999) {
		t.Skip("PID 999999 unexpectedly alive — skipping test")
	}
}

func TestIsProcessAlive_InvalidPID(t *testing.T) {
	if isProcessAlive(0) {
		t.Error("expected PID 0 to be reported as not alive")
	}
	if isProcessAlive(-1) {
		t.Error("expected PID -1 to be reported as not alive")
	}
}

// contains is a test helper for checking substrings.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstr(s, substr))
}

func containsSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
