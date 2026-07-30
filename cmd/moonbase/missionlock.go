package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

// missionLockInfo holds the metadata stored in the lock file.
type missionLockInfo struct {
	PID       int       `json:"pid"`
	StartedAt time.Time `json:"started_at"`
}

// missionLockPath returns the path to the mission WIP lock file.
func missionLockPath() string {
	home := mustUserHomeDir()
	return filepath.Join(home, ".moonbase", "mission.lock")
}

// acquireMissionLock attempts to take the mission WIP lock.
// If force is false and a live lock exists (PID alive), it returns an error.
// If the lock is stale (PID dead) or force is true, it takes over.
// Returns a release function that removes the lock file.
func acquireMissionLock(force bool) (release func(), err error) {
	lockPath := missionLockPath()

	// Ensure directory exists
	dir := filepath.Dir(lockPath)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("creating lock directory %s: %w", dir, err)
	}

	// Check existing lock
	data, readErr := os.ReadFile(lockPath)
	if readErr == nil && len(data) > 0 {
		var info missionLockInfo
		if jsonErr := json.Unmarshal(data, &info); jsonErr == nil {
			if isProcessAlive(info.PID) && !force {
				elapsed := time.Since(info.StartedAt).Truncate(time.Second)
				return nil, fmt.Errorf(
					"Mission already in progress (PID %d, started %s ago). Use --force to override.",
					info.PID, elapsed,
				)
			}
			// PID is dead (stale lock) or force — fall through to overwrite
		}
	}

	// Write new lock
	info := missionLockInfo{
		PID:       os.Getpid(),
		StartedAt: time.Now().UTC(),
	}
	lockData, err := json.Marshal(info)
	if err != nil {
		return nil, fmt.Errorf("marshaling lock info: %w", err)
	}
	if err := os.WriteFile(lockPath, lockData, 0o600); err != nil {
		return nil, fmt.Errorf("writing lock file %s: %w", lockPath, err)
	}

	release = func() {
		os.Remove(lockPath)
	}
	return release, nil
}

// isProcessAlive checks whether a process with the given PID is alive.
// Uses the Unix signal-0 technique: kill(pid, 0) succeeds if the process exists.
func isProcessAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	err := syscall.Kill(pid, 0)
	return err == nil
}
